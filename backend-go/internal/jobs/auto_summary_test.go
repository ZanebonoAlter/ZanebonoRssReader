package jobs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/airouter"
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

func TestGenerateSummaryForFeedSkipsExistingBatchAndLogsMetadata(t *testing.T) {
	setupSchedulersTestDB(t)
	if err := database.DB.AutoMigrate(&models.Category{}, &models.UserPreference{}, &models.AISummary{}, &models.TopicTag{}, &models.AISummaryTopic{}, &models.ArticleTopicTag{}, &models.AIProvider{}, &models.AIRoute{}, &models.AIRouteProvider{}, &models.AICallLog{}); err != nil {
		t.Fatalf("extra migrate: %v", err)
	}

	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"summary body"}}]}`))
	}))
	defer aiServer.Close()

	provider := models.AIProvider{Name: "summary-primary", ProviderType: airouter.ProviderTypeOpenAICompatible, BaseURL: aiServer.URL, APIKey: "token", Model: "test-model", Enabled: true}
	if err := database.DB.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	route := models.AIRoute{Name: airouter.DefaultRouteName, Capability: string(airouter.CapabilitySummary), Enabled: true, Strategy: "ordered_failover"}
	if err := database.DB.Create(&route).Error; err != nil {
		t.Fatalf("create route: %v", err)
	}
	if err := database.DB.Create(&models.AIRouteProvider{RouteID: route.ID, ProviderID: provider.ID, Priority: 1, Enabled: true}).Error; err != nil {
		t.Fatalf("create route provider: %v", err)
	}

	feed := models.Feed{Title: "Feed", URL: "https://example.com/feed", AISummaryEnabled: true}
	if err := database.DB.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}

	pub := time.Now()
	articles := make([]models.Article, 0, 21)
	for i := 0; i < 21; i++ {
		articles = append(articles, models.Article{
			FeedID:  feed.ID,
			Title:   fmt.Sprintf("Article %02d", i+1),
			Link:    fmt.Sprintf("https://example.com/%02d", i+1),
			Content: "rss body",
			PubDate: &pub,
		})
	}
	if err := database.DB.Create(&articles).Error; err != nil {
		t.Fatalf("create articles: %v", err)
	}

	firstBatchIDs := make([]uint, 0, 20)
	for i := 0; i < 20; i++ {
		firstBatchIDs = append(firstBatchIDs, articles[i].ID)
	}
	firstBatchJSON, _ := json.Marshal(firstBatchIDs)
	existing := models.AISummary{FeedID: &feed.ID, Title: "Existing batch", Summary: "saved", Articles: string(firstBatchJSON), ArticleCount: len(firstBatchIDs), TimeRange: 180}
	if err := database.DB.Create(&existing).Error; err != nil {
		t.Fatalf("create existing summary: %v", err)
	}

	scheduler := NewAutoSummaryScheduler(60)
	scheduler.aiConfig = &AIConfig{TimeRange: 180}

	generated, err := scheduler.generateSummaryForFeed(&feed)
	if err != nil {
		t.Fatalf("generate summary: %v", err)
	}
	if !generated {
		t.Fatal("expected generateSummaryForFeed to generate missing batch")
	}

	var summaries []models.AISummary
	if err := database.DB.Order("id ASC").Find(&summaries).Error; err != nil {
		t.Fatalf("load summaries: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("summary count = %d, want 2", len(summaries))
	}

	var logs []models.AICallLog
	if err := database.DB.Order("id ASC").Find(&logs).Error; err != nil {
		t.Fatalf("load logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("summary log count = %d, want 1", len(logs))
	}
	if !strings.Contains(logs[0].RequestMeta, fmt.Sprintf(`"feed_id":%d`, feed.ID)) {
		t.Fatalf("request_meta missing feed_id: %s", logs[0].RequestMeta)
	}
	if !strings.Contains(logs[0].RequestMeta, `"batch_num":2`) {
		t.Fatalf("request_meta missing batch_num: %s", logs[0].RequestMeta)
	}
	if !strings.Contains(logs[0].RequestMeta, `"article_ids":[`) {
		t.Fatalf("request_meta missing article_ids: %s", logs[0].RequestMeta)
	}
}
