package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"my-robot-backend/internal/domain/contentprocessing"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

type ContentCompletionScheduler struct {
	cron              *cron.Cron
	completionService *contentprocessing.ContentCompletionService
	checkInterval     time.Duration
	taskName          string
	isRunning         bool
	isExecuting       bool
	currentArticle    *contentprocessing.ContentCompletionArticleRef
	lastProcessed     *contentprocessing.ContentCompletionArticleRef
	lastError         string
	lastExecutionTime *time.Time
	lastRunSummary    *ContentCompletionRunSummary
	mu                sync.RWMutex
}

type ContentCompletionRunSummary struct {
	StartedAt              string                                         `json:"started_at"`
	FinishedAt             string                                         `json:"finished_at"`
	CompletedCount         int                                            `json:"completed_count"`
	FailedCount            int                                            `json:"failed_count"`
	BlockedCount           int                                            `json:"blocked_count"`
	StaleProcessingCount   int                                            `json:"stale_processing_count"`
	LiveProcessingCount    int                                            `json:"live_processing_count"`
	CurrentArticle         *contentprocessing.ContentCompletionArticleRef `json:"current_article,omitempty"`
	LastProcessed          *contentprocessing.ContentCompletionArticleRef `json:"last_processed,omitempty"`
	StaleProcessingArticle *contentprocessing.ContentCompletionArticleRef `json:"stale_processing_article,omitempty"`
	ErrorSamples           []ContentCompletionErrorSample                 `json:"error_samples,omitempty"`
}

type ContentCompletionErrorSample struct {
	ArticleID uint   `json:"article_id"`
	Message   string `json:"message"`
	Category  string `json:"category"`
}

func NewContentCompletionScheduler(completionService *contentprocessing.ContentCompletionService, checkIntervalMinutes int) *ContentCompletionScheduler {
	taskName := "ai_summary"

	scheduler := &ContentCompletionScheduler{
		cron:              cron.New(),
		completionService: completionService,
		checkInterval:     time.Duration(checkIntervalMinutes) * time.Minute,
		taskName:          taskName,
	}

	interval := fmt.Sprintf("@every %dm", checkIntervalMinutes)
	_, err := scheduler.cron.AddFunc(interval, scheduler.checkAndCompleteArticles)
	if err != nil {
		log.Printf("Failed to schedule AI summary: %v", err)
	}

	return scheduler
}

func (s *ContentCompletionScheduler) Start() error {
	s.cron.Start()
	s.isRunning = true
	log.Printf("AI summary scheduler started (interval: %v)", s.checkInterval)
	s.initSchedulerTask()
	return nil
}

func (s *ContentCompletionScheduler) Stop() {
	s.cron.Stop()
	s.isRunning = false
	log.Println("AI summary scheduler stopped")
}

func (s *ContentCompletionScheduler) checkAndCompleteArticles() {
	var task models.SchedulerTask
	if err := database.DB.Where("name = ?", s.taskName).First(&task).Error; err != nil {
		log.Printf("Scheduler task not found: %v", err)
		return
	}

	now := time.Now().In(time.FixedZone("CST", 8*3600))
	s.mu.Lock()
	s.isExecuting = true
	s.currentArticle = nil
	s.lastError = ""
	s.lastExecutionTime = &now
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.isExecuting = false
		s.currentArticle = nil
		s.mu.Unlock()
	}()

	task.Status = "running"
	task.LastExecutionTime = &now
	database.DB.Save(&task)

	startTime := time.Now()
	runSummary := &ContentCompletionRunSummary{
		StartedAt: now.Format(time.RFC3339),
	}
	var articles []models.Article
	err := database.DB.
		Joins("JOIN feeds ON feeds.id = articles.feed_id").
		Where("articles.firecrawl_status = ? AND articles.summary_status = ?", "completed", "incomplete").
		Where("feeds.article_summary_enabled = ?", true).
		Preload("Feed").
		Limit(50).
		Find(&articles).Error
	if err != nil {
		task.Status = "error"
		task.LastError = err.Error()
		task.LastErrorTime = &now
		task.FailedExecutions++
		task.ConsecutiveFailures++
		nextRun := now.Add(s.checkInterval)
		task.NextExecutionTime = &nextRun
		database.DB.Save(&task)
		s.mu.Lock()
		s.lastError = err.Error()
		s.mu.Unlock()
		return
	}

	completedIDs := make([]uint, 0, len(articles))
	errors := make([]error, 0)
	for _, article := range articles {
		s.mu.Lock()
		s.currentArticle = contentprocessing.ToArticleRef(article)
		runSummary.CurrentArticle = s.currentArticle
		s.mu.Unlock()

		if article.CompletionAttempts >= article.Feed.MaxCompletionRetries {
			errors = append(errors, fmt.Errorf("article %d: max completion retries exceeded", article.ID))
			runSummary.BlockedCount++
			appendRunError(runSummary, article.ID, "max completion retries exceeded", "retries")
			continue
		}

		if err := s.completionService.CompleteArticle(article.ID); err != nil {
			errors = append(errors, fmt.Errorf("article %d: %w", article.ID, err))
			runSummary.FailedCount++
			appendRunError(runSummary, article.ID, err.Error(), classifyCompletionError(err.Error()))
			s.mu.Lock()
			s.lastError = err.Error()
			s.lastProcessed = contentprocessing.ToArticleRef(article)
			runSummary.LastProcessed = s.lastProcessed
			s.mu.Unlock()
			continue
		}

		completedIDs = append(completedIDs, article.ID)
		runSummary.CompletedCount++
		s.mu.Lock()
		s.lastProcessed = contentprocessing.ToArticleRef(article)
		runSummary.LastProcessed = s.lastProcessed
		s.mu.Unlock()
	}
	duration := time.Since(startTime).Seconds()

	task.LastExecutionDuration = &duration

	if len(errors) > 0 {
		task.Status = "error"
		task.FailedExecutions++
		task.ConsecutiveFailures++
		task.LastError = errors[0].Error()
		task.LastErrorTime = &now
		log.Printf("AI summary completed with errors: %d completed, %d failed", len(completedIDs), len(errors))
	} else {
		task.Status = "idle"
		task.SuccessfulExecutions++
		task.ConsecutiveFailures = 0
		task.LastError = ""
		log.Printf("AI summary completed successfully: %d articles processed", len(completedIDs))
	}

	overview, overviewErr := s.completionService.GetOverview()
	if overviewErr == nil && overview != nil {
		runSummary.BlockedCount = overview.BlockedCount
		runSummary.StaleProcessingCount = overview.StaleProcessingCount
		runSummary.LiveProcessingCount = 0
		runSummary.StaleProcessingArticle = overview.StaleProcessingArticle
	}
	runSummary.FinishedAt = time.Now().In(time.FixedZone("CST", 8*3600)).Format(time.RFC3339)
	if encoded, err := json.Marshal(runSummary); err == nil {
		task.LastExecutionResult = string(encoded)
	}

	task.TotalExecutions++

	nextRun := now.Add(s.checkInterval)
	task.NextExecutionTime = &nextRun
	database.DB.Save(&task)
	s.mu.Lock()
	s.lastRunSummary = runSummary
	s.mu.Unlock()
}

func (s *ContentCompletionScheduler) initSchedulerTask() {
	var task models.SchedulerTask
	err := database.DB.Where("name = ?", s.taskName).First(&task).Error

	if err == nil {
		return
	}

	now := time.Now().In(time.FixedZone("CST", 8*3600))
	nextRun := now.Add(s.checkInterval)

	task = models.SchedulerTask{
		Name:                 s.taskName,
		Description:          "AI summarize article content based on Firecrawl content",
		CheckInterval:        int(s.checkInterval.Seconds()),
		Status:               "idle",
		NextExecutionTime:    &nextRun,
		TotalExecutions:      0,
		SuccessfulExecutions: 0,
		FailedExecutions:     0,
		ConsecutiveFailures:  0,
	}

	database.DB.Create(&task)
	log.Println("AI summary scheduler task initialized")
}

func (s *ContentCompletionScheduler) GetStatus() map[string]interface{} {
	var task models.SchedulerTask
	var taskData map[string]interface{}
	if err := database.DB.Where("name = ?", s.taskName).First(&task).Error; err == nil {
		taskData = task.ToDict()
	}

	overview, err := s.completionService.GetOverview()
	if err != nil {
		log.Printf("failed to load ai summary overview: %v", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"status": func() string {
			if s.isExecuting {
				return "running"
			}
			if s.isRunning {
				return "idle"
			}
			return "stopped"
		}(),
		"check_interval":  int(s.checkInterval.Seconds()),
		"task_name":       s.taskName,
		"is_executing":    s.isExecuting,
		"current_article": s.currentArticle,
		"last_processed":  s.lastProcessed,
		"live_processing_count": func() int {
			if s.isExecuting && s.currentArticle != nil {
				return 1
			}
			return 0
		}(),
		"last_execution_time": s.lastExecutionTime,
		"last_error": func() string {
			if task.LastError != "" {
				return task.LastError
			}
			return s.lastError
		}(),
	}
	if taskData != nil {
		status["database_state"] = taskData
		status["next_run"] = task.NextExecutionTime
		if summary := parseLastRunSummary(task); summary != nil {
			status["last_run_summary"] = summary
		}
	}
	if overview != nil {
		liveProcessingCount := 0
		if s.isExecuting && s.currentArticle != nil {
			liveProcessingCount = 1
		}
		status["overview"] = map[string]interface{}{
			"pending_count":          overview.PendingCount,
			"processing_count":       overview.ProcessingCount,
			"live_processing_count":  liveProcessingCount,
			"stale_processing_count": overview.StaleProcessingCount,
			"completed_count":        overview.CompletedCount,
			"failed_count":           overview.FailedCount,
			"blocked_count":          overview.BlockedCount,
			"total_count":            overview.TotalCount,
			"ai_configured":          overview.AIConfigured,
			"blocked_reasons": map[string]interface{}{
				"waiting_for_firecrawl_count":     overview.BlockedReasons.WaitingForFirecrawlCount,
				"feed_disabled_count":             overview.BlockedReasons.FeedDisabledCount,
				"ai_unconfigured_count":           overview.BlockedReasons.AIUnconfiguredCount,
				"ready_but_missing_content_count": overview.BlockedReasons.ReadyButMissingContentCount,
			},
			"stale_processing_article": overview.StaleProcessingArticle,
		}
		status["stale_processing_count"] = overview.StaleProcessingCount
		status["stale_processing_article"] = overview.StaleProcessingArticle
	}

	return status
}

func parseLastRunSummary(task models.SchedulerTask) *ContentCompletionRunSummary {
	if task.LastExecutionResult == "" {
		return nil
	}

	var summary ContentCompletionRunSummary
	if err := json.Unmarshal([]byte(task.LastExecutionResult), &summary); err != nil {
		return nil
	}

	return &summary
}

func appendRunError(summary *ContentCompletionRunSummary, articleID uint, message, category string) {
	if len(summary.ErrorSamples) >= 5 {
		return
	}
	summary.ErrorSamples = append(summary.ErrorSamples, ContentCompletionErrorSample{
		ArticleID: articleID,
		Message:   message,
		Category:  category,
	})
}

func classifyCompletionError(message string) string {
	message = strings.ToLower(message)
	switch {
	case strings.Contains(message, "unexpected eof"), strings.Contains(message, "timeout"), strings.Contains(message, "connection reset"), strings.Contains(message, "tls"):
		return "network"
	case strings.Contains(message, "not configured"), strings.Contains(message, "api key"), strings.Contains(message, "model"):
		return "config"
	case strings.Contains(message, "no firecrawl content"):
		return "content"
	case strings.Contains(message, "max completion retries"):
		return "retries"
	default:
		return "unknown"
	}
}

func (s *ContentCompletionScheduler) Trigger() {
	go s.checkAndCompleteArticles()
}
