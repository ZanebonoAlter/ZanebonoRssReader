package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"my-robot-backend/internal/domain/feeds"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

type AutoRefreshScheduler struct {
	cron           *cron.Cron
	checkInterval  time.Duration
	feedService    *feeds.FeedService
	refreshFeed    func(uint) error
	isRunning      bool
	isExecuting    bool
	executionMutex sync.Mutex
}

type AutoRefreshRunSummary struct {
	TriggerSource          string `json:"trigger_source"`
	StartedAt              string `json:"started_at"`
	FinishedAt             string `json:"finished_at"`
	ScannedFeeds           int    `json:"scanned_feeds"`
	DueFeeds               int    `json:"due_feeds"`
	TriggeredFeeds         int    `json:"triggered_feeds"`
	AlreadyRefreshingFeeds int    `json:"already_refreshing_feeds"`
	Reason                 string `json:"reason"`
}

func NewAutoRefreshScheduler(checkInterval int) *AutoRefreshScheduler {
	feedService := feeds.NewFeedService()
	return &AutoRefreshScheduler{
		cron:          cron.New(),
		checkInterval: time.Duration(checkInterval) * time.Second,
		feedService:   feedService,
		refreshFeed:   feedService.RefreshFeed,
		isRunning:     false,
	}
}

func (s *AutoRefreshScheduler) Start() error {
	if s.isRunning {
		return fmt.Errorf("scheduler already running")
	}

	scheduleExpr := fmt.Sprintf("@every %ds", int64(s.checkInterval.Seconds()))
	if _, err := s.cron.AddFunc(scheduleExpr, s.checkAndRefreshFeeds); err != nil {
		return fmt.Errorf("failed to schedule auto-refresh: %w", err)
	}

	s.cron.Start()
	s.isRunning = true
	log.Printf("Auto-refresh scheduler started with interval: %v", s.checkInterval)
	s.initSchedulerTask()

	return nil
}

func (s *AutoRefreshScheduler) Stop() {
	if !s.isRunning {
		return
	}

	s.cron.Stop()
	s.isRunning = false
	log.Println("Auto-refresh scheduler stopped")
}

func (s *AutoRefreshScheduler) checkAndRefreshFeeds() {
	if !s.executionMutex.TryLock() {
		log.Println("Auto-refresh scheduler already running, skipping this cycle")
		return
	}
	s.isExecuting = true
	defer func() {
		s.isExecuting = false
		s.executionMutex.Unlock()
	}()

	_, _ = s.runRefreshCycle("scheduled")
}

func (s *AutoRefreshScheduler) runRefreshCycle(triggerSource string) (*AutoRefreshRunSummary, error) {
	startTime := time.Now()
	s.updateSchedulerStatus("running", "", nil, nil)

	summary := &AutoRefreshRunSummary{
		TriggerSource: triggerSource,
		StartedAt:     startTime.Format(time.RFC3339),
	}

	var feeds []models.Feed
	if err := database.DB.Where("refresh_interval > 0").Find(&feeds).Error; err != nil {
		log.Printf("Error querying feeds: %v", err)
		summary.Reason = "query_failed"
		summary.FinishedAt = time.Now().Format(time.RFC3339)
		s.updateSchedulerStatus("idle", err.Error(), &startTime, summary)
		return summary, err
	}
	summary.ScannedFeeds = len(feeds)

	now := time.Now()
	for _, feed := range feeds {
		if !s.needsRefresh(&feed, now) {
			continue
		}

		summary.DueFeeds++
		if feed.RefreshStatus == "refreshing" {
			summary.AlreadyRefreshingFeeds++
			continue
		}

		s.markFeedRefreshing(feed.ID, now)
		go s.refreshFeedAsync(feed.ID)
		summary.TriggeredFeeds++
	}

	summary.FinishedAt = time.Now().Format(time.RFC3339)
	summary.Reason = autoRefreshReason(summary)
	if summary.TriggeredFeeds > 0 {
		log.Printf("Auto-refresh: triggered %d feed(s)", summary.TriggeredFeeds)
	}

	s.updateSchedulerStatus("idle", "", &startTime, summary)
	return summary, nil
}

func (s *AutoRefreshScheduler) needsRefresh(feed *models.Feed, now time.Time) bool {
	if feed.LastRefreshAt == nil {
		return true
	}

	timeSinceRefresh := now.Sub(*feed.LastRefreshAt)
	interval := time.Duration(feed.RefreshInterval) * time.Minute

	return timeSinceRefresh >= interval
}

func (s *AutoRefreshScheduler) refreshFeedAsync(feedID uint) {
	if err := s.refreshFeed(feedID); err != nil {
		log.Printf("Error refreshing feed %d: %v", feedID, err)
	}
}

func (s *AutoRefreshScheduler) GetStatus() map[string]interface{} {
	entries := s.cron.Entries()

	var nextRun time.Time
	if len(entries) > 0 {
		nextRun = entries[0].Next
	}

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
		"check_interval": int(s.checkInterval.Seconds()),
		"next_run":       nextRun.Format(time.RFC3339),
		"is_executing":   s.isExecuting,
	}

	var task models.SchedulerTask
	if err := database.DB.Where("name = ?", "auto_refresh").First(&task).Error; err == nil {
		status["database_state"] = task.ToDict()
		status["next_run"] = task.NextExecutionTime
		if summary := parseAutoRefreshRunSummary(task); summary != nil {
			status["last_run_summary"] = summary
		}
	}

	return status
}

func (s *AutoRefreshScheduler) Trigger() {
	go s.checkAndRefreshFeeds()
}

func (s *AutoRefreshScheduler) TriggerNow() map[string]interface{} {
	if !s.executionMutex.TryLock() {
		return map[string]interface{}{
			"accepted":    false,
			"started":     false,
			"reason":      "already_running",
			"message":     "后台刷新正在执行中，先别重复点。",
			"status_code": http.StatusConflict,
		}
	}
	s.isExecuting = true
	defer func() {
		s.isExecuting = false
		s.executionMutex.Unlock()
	}()

	summary, err := s.runRefreshCycle("manual")
	if err != nil {
		return map[string]interface{}{
			"accepted":    false,
			"started":     false,
			"reason":      "refresh_cycle_failed",
			"message":     err.Error(),
			"summary":     summary,
			"status_code": http.StatusInternalServerError,
		}
	}

	message := "手动扫描完成，没有 feed 到点。"
	if summary.TriggeredFeeds > 0 {
		message = fmt.Sprintf("手动扫描完成，已触发 %d 个 feed 刷新。", summary.TriggeredFeeds)
	} else if summary.AlreadyRefreshingFeeds > 0 {
		message = "手动扫描完成，但到点的 feed 已在刷新中。"
	} else if summary.ScannedFeeds == 0 {
		message = "当前没有开启自动刷新的 feed。"
	}

	return map[string]interface{}{
		"accepted":  true,
		"started":   summary.TriggeredFeeds > 0,
		"effectful": summary.TriggeredFeeds > 0,
		"reason":    summary.Reason,
		"message":   message,
		"summary":   summary,
	}
}

func (s *AutoRefreshScheduler) initSchedulerTask() {
	var task models.SchedulerTask
	now := time.Now()
	nextRun := now.Add(s.checkInterval)

	if err := database.DB.Where("name = ?", "auto_refresh").First(&task).Error; err == nil {
		updates := map[string]interface{}{
			"description":         "Auto-refresh RSS feeds",
			"check_interval":      int(s.checkInterval.Seconds()),
			"next_execution_time": &nextRun,
		}

		if task.Status == "" || task.Status == "success" || task.Status == "failed" {
			updates["status"] = "idle"
			updates["last_error"] = ""
		}

		database.DB.Model(&task).Updates(updates)
		return
	}

	task = models.SchedulerTask{
		Name:              "auto_refresh",
		Description:       "Auto-refresh RSS feeds",
		CheckInterval:     int(s.checkInterval.Seconds()),
		Status:            "idle",
		NextExecutionTime: &nextRun,
	}
	database.DB.Create(&task)
}

func (s *AutoRefreshScheduler) markFeedRefreshing(feedID uint, now time.Time) {
	database.DB.Model(&models.Feed{}).Where("id = ?", feedID).Updates(map[string]interface{}{
		"refresh_status":  "refreshing",
		"last_refresh_at": &now,
		"refresh_error":   "",
	})
}

func autoRefreshReason(summary *AutoRefreshRunSummary) string {
	switch {
	case summary.ScannedFeeds == 0:
		return "no_feeds_enabled"
	case summary.TriggeredFeeds > 0:
		return "feeds_triggered"
	case summary.DueFeeds == 0:
		return "no_feeds_due"
	case summary.AlreadyRefreshingFeeds > 0:
		return "all_due_feeds_already_refreshing"
	default:
		return "scan_complete"
	}
}

func (s *AutoRefreshScheduler) updateSchedulerStatus(status, lastError string, startTime *time.Time, summary *AutoRefreshRunSummary) {
	var task models.SchedulerTask
	err := database.DB.Where("name = ?", "auto_refresh").First(&task).Error
	now := time.Now()

	resultJSON := ""
	if summary != nil {
		if data, marshalErr := json.Marshal(summary); marshalErr == nil {
			resultJSON = string(data)
		}
	}

	if err == nil {
		task.Status = status
		task.LastError = lastError
		if lastError != "" {
			errTime := now
			task.LastErrorTime = &errTime
		}
		if resultJSON != "" {
			task.LastExecutionResult = resultJSON
		}

		if startTime != nil {
			duration := time.Since(*startTime)
			durationFloat := float64(duration.Seconds())
			task.LastExecutionDuration = &durationFloat
			task.LastExecutionTime = &now
			task.TotalExecutions++
			if lastError == "" {
				task.SuccessfulExecutions++
				task.ConsecutiveFailures = 0
			} else {
				task.FailedExecutions++
				task.ConsecutiveFailures++
			}
			nextExecution := now.Add(s.checkInterval)
			task.NextExecutionTime = &nextExecution
		}

		database.DB.Save(&task)
		return
	}

	nextExecution := now.Add(s.checkInterval)
	var durationFloat *float64
	if startTime != nil {
		d := time.Since(*startTime)
		df := float64(d.Seconds())
		durationFloat = &df
	}

	task = models.SchedulerTask{
		Name:                  "auto_refresh",
		Description:           "Auto-refresh RSS feeds",
		CheckInterval:         int(s.checkInterval.Seconds()),
		Status:                status,
		LastError:             lastError,
		NextExecutionTime:     &nextExecution,
		LastExecutionDuration: durationFloat,
		LastExecutionResult:   resultJSON,
	}
	if startTime != nil {
		task.LastExecutionTime = &now
		task.TotalExecutions = 1
		if lastError == "" {
			task.SuccessfulExecutions = 1
		} else {
			task.FailedExecutions = 1
			task.ConsecutiveFailures = 1
		}
	}
	database.DB.Create(&task)
}

func parseAutoRefreshRunSummary(task models.SchedulerTask) *AutoRefreshRunSummary {
	if task.LastExecutionResult == "" {
		return nil
	}

	var summary AutoRefreshRunSummary
	if err := json.Unmarshal([]byte(task.LastExecutionResult), &summary); err != nil {
		return nil
	}

	return &summary
}
