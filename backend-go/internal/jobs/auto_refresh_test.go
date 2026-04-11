package jobs

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"my-robot-backend/internal/app/runtimeinfo"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/ws"
)

type stubAutoSummaryTrigger struct {
	triggered chan struct{}
}

func (s stubAutoSummaryTrigger) TriggerNow() map[string]interface{} {
	select {
	case s.triggered <- struct{}{}:
	default:
	}
	return map[string]interface{}{"accepted": true, "started": true}
}

func TestAutoRefreshCompleteMessageJSON(t *testing.T) {
	msg := ws.AutoRefreshCompleteMessage{
		Type:            "auto_refresh_complete",
		TriggeredFeeds:  3,
		StaleResetFeeds: 1,
		DurationSeconds: 2.5,
		Timestamp:       "2026-04-11T04:20:00Z",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal auto refresh complete message: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal auto refresh complete message: %v", err)
	}

	if payload["type"] != "auto_refresh_complete" {
		t.Fatalf("type = %v, want auto_refresh_complete", payload["type"])
	}
	if payload["triggered_feeds"] != float64(3) {
		t.Fatalf("triggered_feeds = %v, want 3", payload["triggered_feeds"])
	}
	if payload["stale_reset_feeds"] != float64(1) {
		t.Fatalf("stale_reset_feeds = %v, want 1", payload["stale_reset_feeds"])
	}
	if payload["duration_seconds"] != 2.5 {
		t.Fatalf("duration_seconds = %v, want 2.5", payload["duration_seconds"])
	}
	if payload["timestamp"] != "2026-04-11T04:20:00Z" {
		t.Fatalf("timestamp = %v, want 2026-04-11T04:20:00Z", payload["timestamp"])
	}
}

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
		refreshFeed: func(ctx context.Context, feedID uint) error {
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
}

func TestAutoRefreshTriggerNowRunsAutoSummaryAfterTriggeredRefreshesFinish(t *testing.T) {
	setupSchedulersTestDB(t)

	triggered := make(chan struct{}, 1)
	runtimeinfo.AutoSummarySchedulerInterface = stubAutoSummaryTrigger{triggered: triggered}
	defer func() {
		runtimeinfo.AutoSummarySchedulerInterface = nil
	}()

	feed := models.Feed{
		Title:           "Due feed",
		URL:             "https://example.com/rss",
		RefreshInterval: 15,
	}
	if err := database.DB.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}

	refreshDone := make(chan struct{})
	scheduler := &AutoRefreshScheduler{
		checkInterval: time.Minute,
		refreshFeed: func(ctx context.Context, feedID uint) error {
			defer close(refreshDone)
			return nil
		},
	}

	result := scheduler.TriggerNow()
	if result["accepted"] != true {
		t.Fatalf("accepted = %v, want true", result["accepted"])
	}

	select {
	case <-refreshDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for refresh to finish")
	}

	select {
	case <-triggered:
	case <-time.After(time.Second):
		t.Fatal("expected auto summary trigger after refresh completion")
	}
}
