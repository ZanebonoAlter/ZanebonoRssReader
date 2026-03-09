package jobs

import (
	"encoding/json"
	"testing"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func TestAutoRefreshTriggerNowUpdatesSchedulerTaskAndFeedState(t *testing.T) {
	setupSchedulersTestDB(t)

	feed := models.Feed{
		Title:           "Due feed",
		URL:             "https://example.com/rss",
		RefreshInterval: 15,
	}
	if err := database.DB.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}

	scheduler := &AutoRefreshScheduler{
		checkInterval: time.Minute,
		refreshFeed: func(feedID uint) error {
			return nil
		},
	}

	result := scheduler.TriggerNow()
	if result["accepted"] != true {
		t.Fatalf("accepted = %v, want true", result["accepted"])
	}
	if result["started"] != true {
		t.Fatalf("started = %v, want true", result["started"])
	}

	var task models.SchedulerTask
	if err := database.DB.Where("name = ?", "auto_refresh").First(&task).Error; err != nil {
		t.Fatalf("load scheduler task: %v", err)
	}
	if task.TotalExecutions != 1 {
		t.Fatalf("total executions = %d, want 1", task.TotalExecutions)
	}
	if task.LastExecutionTime == nil {
		t.Fatal("expected last execution time to be set")
	}
	if task.LastExecutionResult == "" {
		t.Fatal("expected last execution result to be stored")
	}

	var summary AutoRefreshRunSummary
	if err := json.Unmarshal([]byte(task.LastExecutionResult), &summary); err != nil {
		t.Fatalf("parse last execution result: %v", err)
	}
	if summary.ScannedFeeds != 1 {
		t.Fatalf("scanned feeds = %d, want 1", summary.ScannedFeeds)
	}
	if summary.TriggeredFeeds != 1 {
		t.Fatalf("triggered feeds = %d, want 1", summary.TriggeredFeeds)
	}

	var refreshedFeed models.Feed
	if err := database.DB.First(&refreshedFeed, feed.ID).Error; err != nil {
		t.Fatalf("reload feed: %v", err)
	}
	if refreshedFeed.RefreshStatus != "refreshing" {
		t.Fatalf("refresh status = %q, want refreshing", refreshedFeed.RefreshStatus)
	}
	if refreshedFeed.LastRefreshAt == nil {
		t.Fatal("expected last refresh timestamp to be set")
	}
}
