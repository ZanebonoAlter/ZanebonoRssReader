package jobs

import (
	"encoding/json"
	"log"
	"sync"
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
	status            string
	lastExecutionTime *time.Time
	lastError         string
	concurrency       int
}

func NewFirecrawlScheduler() *FirecrawlScheduler {
	return &FirecrawlScheduler{
		name:          "Firecrawl Content Fetcher",
		checkInterval: 300,
		stopChan:      make(chan struct{}),
		status:        "idle",
		concurrency:   3,
	}
}

func (s *FirecrawlScheduler) Start() error {
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
	startTime := time.Now()
	s.status = "running"
	s.lastExecutionTime = &startTime

	defer func() {
		s.status = "idle"
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

	type crawlResult struct {
		articleID uint
		success   bool
		errMsg    string
		title     string
	}

	results := make(chan crawlResult, len(articles))
	semaphore := make(chan struct{}, s.concurrency)

	var wg sync.WaitGroup

	for i := range articles {
		article := articles[i]
		wg.Add(1)
		go func(art models.Article) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var feed models.Feed
			if err := database.DB.First(&feed, art.FeedID).Error; err != nil {
				results <- crawlResult{
					articleID: art.ID,
					success:   false,
					errMsg:    err.Error(),
					title:     art.Title,
				}
				return
			}

			database.DB.Model(&art).Update("firecrawl_status", "processing")

			s.broadcastProgress(batchID, "processing", len(articles), 0, 0, &ws.FirecrawlArticleProgress{
				ID:     art.ID,
				Title:  art.Title,
				Status: "processing",
			})

			result, crawlErr := firecrawlService.ScrapePage(art.Link)
			if crawlErr != nil {
				results <- crawlResult{
					articleID: art.ID,
					success:   false,
					errMsg:    crawlErr.Error(),
					title:     art.Title,
				}
				log.Printf("[Firecrawl] Failed to crawl %s: %v", art.Link, crawlErr)
				return
			}

			now := time.Now()
			updates := map[string]interface{}{
				"firecrawl_status":     "completed",
				"firecrawl_content":    result.Data.Markdown,
				"firecrawl_crawled_at": now,
			}
			if feed.ContentCompletionEnabled {
				updates["content_status"] = "incomplete"
			}
			database.DB.Model(&art).Updates(updates)

			results <- crawlResult{
				articleID: art.ID,
				success:   true,
				title:     art.Title,
			}
		}(article)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	completed := 0
	failed := 0
	for res := range results {
		if res.success {
			completed++
		} else {
			failed++
			database.DB.Model(&models.Article{}).
				Where("id = ?", res.articleID).
				Updates(map[string]interface{}{
					"firecrawl_status": "failed",
					"firecrawl_error":  res.errMsg,
				})
		}

		s.broadcastProgress(batchID, "processing", len(articles), completed, failed, &ws.FirecrawlArticleProgress{
			ID:    res.articleID,
			Title: res.title,
			Status: func() string {
				if res.success {
					return "completed"
				}
				return "failed"
			}(),
			Error: res.errMsg,
		})
	}

	duration := time.Since(startTime).Seconds()
	log.Printf("[Firecrawl] Crawled %d/%d articles in %.2fs", completed, len(articles), duration)

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
	}
}

func (s *FirecrawlScheduler) Trigger() {
	log.Println("[Firecrawl] Manual trigger received")
	go s.checkAndCrawl()
}
