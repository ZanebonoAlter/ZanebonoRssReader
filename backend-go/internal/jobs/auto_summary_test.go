package jobs

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func TestBuildAutoSummaryArticleTextPrefersFirecrawlContent(t *testing.T) {
	article := models.Article{
		Title:            "Firecrawl First",
		Description:      "short desc",
		Content:          "plain rss body",
		FirecrawlContent: "firecrawl markdown body",
		Link:             "https://example.com/a1",
	}

	text := buildAutoSummaryArticleText(article)

	if !containsText(text, "Content: firecrawl markdown body") {
		t.Fatalf("expected firecrawl content, got %q", text)
	}
	if containsText(text, "Content: plain rss body") {
		t.Fatalf("did not expect rss content when firecrawl content exists, got %q", text)
	}
}

func TestBuildAutoSummaryArticleTextFallsBackToContentThenDescription(t *testing.T) {
	article := models.Article{
		Title:       "Fallback",
		Description: "desc body",
		Content:     "plain rss body",
		Link:        "https://example.com/a2",
	}

	text := buildAutoSummaryArticleText(article)
	if !containsText(text, "Content: plain rss body") {
		t.Fatalf("expected rss content fallback, got %q", text)
	}

	article.Content = ""
	text = buildAutoSummaryArticleText(article)
	if !containsText(text, "Description: desc body") {
		t.Fatalf("expected description fallback, got %q", text)
	}
}

func containsText(text string, expected string) bool {
	return strings.Contains(text, expected)
}

func TestAutoSummaryTriggerNowRejectsWhenConfigMissing(t *testing.T) {
	setupSchedulersTestDB(t)

	scheduler := NewAutoSummaryScheduler(60)
	result := scheduler.TriggerNow()

	if result["accepted"] != false {
		t.Fatalf("accepted = %v, want false", result["accepted"])
	}
	if result["reason"] != "ai_config_missing" {
		t.Fatalf("reason = %v, want ai_config_missing", result["reason"])
	}

	var task models.SchedulerTask
	if err := database.DB.Where("name = ?", "auto_summary").First(&task).Error; err != nil {
		t.Fatalf("load scheduler task: %v", err)
	}
	if task.LastError != "AI config not set" {
		t.Fatalf("last error = %q, want AI config not set", task.LastError)
	}
}

func TestAutoSummaryTriggerNowStartsRealRun(t *testing.T) {
	setupSchedulersTestDB(t)

	configJSON, _ := json.Marshal(AIConfig{
		BaseURL:   "https://example.com",
		APIKey:    "token",
		Model:     "test-model",
		TimeRange: 180,
	})
	setting := models.AISettings{
		Key:   "summary_config",
		Value: string(configJSON),
	}
	if err := database.DB.Create(&setting).Error; err != nil {
		t.Fatalf("create setting: %v", err)
	}

	scheduler := NewAutoSummaryScheduler(60)
	result := scheduler.TriggerNow()
	if result["accepted"] != true {
		t.Fatalf("accepted = %v, want true", result["accepted"])
	}
	if result["started"] != true {
		t.Fatalf("started = %v, want true", result["started"])
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var task models.SchedulerTask
		if err := database.DB.Where("name = ?", "auto_summary").First(&task).Error; err == nil && task.TotalExecutions > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	var task models.SchedulerTask
	if err := database.DB.Where("name = ?", "auto_summary").First(&task).Error; err != nil {
		t.Fatalf("load scheduler task: %v", err)
	}
	t.Fatalf("expected auto_summary manual trigger to update task, got total executions %d", task.TotalExecutions)
}
