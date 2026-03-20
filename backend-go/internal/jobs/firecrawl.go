package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"my-robot-backend/internal/domain/contentprocessing"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/ws"
)

type FirecrawlScheduler struct {
	name              string
	checkInterval     int
	stopChan          chan struct{}
	wg                sync.WaitGroup
	executionMutex    sync.Mutex
	status            string
	lastExecutionTime *time.Time
	lastError         string
	concurrency       int
	queueSize         int32
	processingCount   int32
}

func NewFirecrawlScheduler() *FirecrawlScheduler {
	return &FirecrawlScheduler{
		name:          "Firecrawl Content Fetcher",
		checkInterval: 300,
		stopChan:      make(chan struct{}),
		status:        "idle",
		concurrency:   1,
	}
}

func (s *FirecrawlScheduler) Start() error {
	s.stopChan = make(chan struct{})
	s.wg.Add(1)
	go s.run()
	log.Printf("[Firecrawl] Scheduler started")
	return nil
}

func (s *FirecrawlScheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
	log.Printf("[Firecrawl] Scheduler stopped")
}

func (s *FirecrawlScheduler) TriggerNow() map[string]interface{} {
	if !s.executionMutex.TryLock() {
		return map[string]interface{}{
			"accepted":    false,
			"started":     false,
			"reason":      "already_running",
			"message":     "Firecrawl 正在执行中，稍后再试。",
			"status_code": 409,
		}
	}

	go s.runCrawlCycle()
	return map[string]interface{}{
		"accepted": true,
		"started":  true,
		"message":  "Firecrawl scheduler triggered",
	}
}

func (s *FirecrawlScheduler) ResetStats() error {
	s.lastExecutionTime = nil
	s.lastError = ""
	s.status = "idle"
	atomic.StoreInt32(&s.queueSize, 0)
	atomic.StoreInt32(&s.processingCount, 0)
	return nil
}

func (s *FirecrawlScheduler) UpdateInterval(interval int) error {
	if interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}

	s.checkInterval = interval
	s.Stop()
	return s.Start()
}

func (s *FirecrawlScheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Duration(s.checkInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAndCrawl()
		case <-s.stopChan:
			return
		}
	}
}

func (s *FirecrawlScheduler) checkAndCrawl() {
	if !s.executionMutex.TryLock() {
		log.Println("[Firecrawl] Scheduler already running, skipping this cycle")
		return
	}

	s.runCrawlCycle()
}

func (s *FirecrawlScheduler) runCrawlCycle() {
	defer s.executionMutex.Unlock()

	startTime := time.Now()
	s.status = "running"
	s.lastExecutionTime = &startTime
	atomic.StoreInt32(&s.processingCount, 0)

	defer func() {
		s.status = "idle"
		atomic.StoreInt32(&s.queueSize, 0)
		atomic.StoreInt32(&s.processingCount, 0)
	}()

	config, err := contentprocessing.GetFirecrawlConfig()
	if err != nil {
		s.lastError = err.Error()
		log.Printf("[Firecrawl] Config error: %v", err)
		return
	}

	if !config.Enabled {
		return
	}

	firecrawlService := contentprocessing.NewFirecrawlService(config)

	var articles []models.Article
	database.DB.
		Joins("JOIN feeds ON feeds.id = articles.feed_id").
		Where("feeds.firecrawl_enabled = ? AND articles.firecrawl_status = ?", true, "pending").
		Limit(50).
		Find(&articles)

	if len(articles) == 0 {
		return
	}

	batchID := time.Now().Format("20060102150405")
	s.broadcastProgress(batchID, "processing", len(articles), 0, 0, nil)

	atomic.StoreInt32(&s.queueSize, int32(len(articles)))
	atomic.StoreInt32(&s.processingCount, 0)
	log.Printf("[Firecrawl] Starting sequential processing of %d articles (concurrency=1)", len(articles))

	completed := 0
	failed := 0

	// 单线程串行处理，一个一个来
	for i := range articles {
		art := articles[i]

		var feed models.Feed
		if err := database.DB.First(&feed, art.FeedID).Error; err != nil {
			failed++
			database.DB.Model(&art).Updates(map[string]interface{}{
				"firecrawl_status": "failed",
				"firecrawl_error":  err.Error(),
			})
			s.broadcastProgress(batchID, "processing", len(articles), completed, failed, &ws.FirecrawlArticleProgress{
				ID:     art.ID,
				Title:  art.Title,
				Status: "failed",
				Error:  err.Error(),
			})
			continue
		}

		database.DB.Model(&art).Update("firecrawl_status", "processing")

		s.broadcastProgress(batchID, "processing", len(articles), completed, failed, &ws.FirecrawlArticleProgress{
			ID:     art.ID,
			Title:  art.Title,
			Status: "processing",
		})

		result, crawlErr := firecrawlService.ScrapePage(art.Link)
		if crawlErr != nil {
			failed++
			database.DB.Model(&art).Updates(map[string]interface{}{
				"firecrawl_status": "failed",
				"firecrawl_error":  crawlErr.Error(),
			})
			s.broadcastProgress(batchID, "processing", len(articles), completed, failed, &ws.FirecrawlArticleProgress{
				ID:     art.ID,
				Title:  art.Title,
				Status: "failed",
				Error:  crawlErr.Error(),
			})
			log.Printf("[Firecrawl] Failed to crawl %s: %v", art.Link, crawlErr)
			continue
		}

		now := time.Now()
		updates := map[string]interface{}{
			"firecrawl_status":     "completed",
			"firecrawl_content":    result.Data.Markdown,
			"firecrawl_crawled_at": now,
		}
		if feed.ArticleSummaryEnabled {
			updates["summary_status"] = "incomplete"
		}
		database.DB.Model(&art).Updates(updates)

		completed++
		s.broadcastProgress(batchID, "processing", len(articles), completed, failed, &ws.FirecrawlArticleProgress{
			ID:     art.ID,
			Title:  art.Title,
			Status: "completed",
		})

		// 更新队列状态
		atomic.StoreInt32(&s.queueSize, int32(len(articles)-completed-failed))
		atomic.StoreInt32(&s.processingCount, 0)

		// 每次处理完一个后稍微停顿一下，避免对目标站点造成压力
		time.Sleep(500 * time.Millisecond)
	}

	atomic.StoreInt32(&s.queueSize, 0)
	atomic.StoreInt32(&s.processingCount, 0)

	duration := time.Since(startTime).Seconds()
	log.Printf("[Firecrawl] Sequential crawl completed: %d completed, %d failed out of %d articles in %.2fs", completed, failed, len(articles), duration)

	s.broadcastProgress(batchID, "completed", len(articles), completed, failed, nil)
	s.lastError = ""
}

func (s *FirecrawlScheduler) broadcastProgress(batchID, status string, total, completed, failed int, current *ws.FirecrawlArticleProgress) {
	hub := ws.GetHub()
	msg := ws.FirecrawlProgressMessage{
		Type:      "firecrawl_progress",
		BatchID:   batchID,
		Status:    status,
		Total:     total,
		Completed: completed,
		Failed:    failed,
		Current:   current,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[Firecrawl] Failed to marshal progress: %v", err)
		return
	}
	hub.BroadcastRaw(data)
}

func (s *FirecrawlScheduler) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"name":                s.name,
		"status":              s.status,
		"check_interval":      s.checkInterval,
		"last_execution_time": s.lastExecutionTime,
		"last_error":          s.lastError,
		"concurrency":         s.concurrency,
		"queue_size":          atomic.LoadInt32(&s.queueSize),
		"processing":          atomic.LoadInt32(&s.processingCount),
	}
}

func (s *FirecrawlScheduler) Trigger() {
	log.Println("[Firecrawl] Manual trigger received")
	go s.checkAndCrawl()
}
