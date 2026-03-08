package schedulers

import (
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"my-robot-backend/internal/models"
	"my-robot-backend/internal/services"
	"my-robot-backend/pkg/database"
)

type AutoRefreshScheduler struct {
	cron          *cron.Cron
	checkInterval time.Duration
	feedService   *services.FeedService
	isRunning     bool
}

func NewAutoRefreshScheduler(checkInterval int) *AutoRefreshScheduler {
	return &AutoRefreshScheduler{
		cron:          cron.New(),
		checkInterval: time.Duration(checkInterval) * time.Second,
		feedService:   services.NewFeedService(),
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
	var feeds []models.Feed
	if err := database.DB.Where("refresh_interval > 0").Find(&feeds).Error; err != nil {
		log.Printf("Error querying feeds: %v", err)
		return
	}

	now := time.Now()
	refreshCount := 0

	for _, feed := range feeds {
		if s.needsRefresh(&feed, now) {
			if feed.RefreshStatus != "refreshing" {
				go s.refreshFeedAsync(feed.ID)
				refreshCount++
			}
		}
	}

	if refreshCount > 0 {
		log.Printf("Auto-refresh: triggered %d feed(s)", refreshCount)
	}
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
	if err := s.feedService.RefreshFeed(feedID); err != nil {
		log.Printf("Error refreshing feed %d: %v", feedID, err)
	}
}

func (s *AutoRefreshScheduler) GetStatus() map[string]interface{} {
	entries := s.cron.Entries()

	var nextRun time.Time
	if len(entries) > 0 {
		nextRun = entries[0].Next
	}

	return map[string]interface{}{
		"status": func() string {
			if s.isRunning {
				return "idle"
			}
			return "stopped"
		}(),
		"check_interval": int(s.checkInterval.Seconds()),
		"next_run":       nextRun.Format(time.RFC3339),
	}
}

func (s *AutoRefreshScheduler) Trigger() {
	go s.checkAndRefreshFeeds()
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
