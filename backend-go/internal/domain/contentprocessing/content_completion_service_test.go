package contentprocessing

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
)

func setupServicesTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	database.DB = db
	if err := database.DB.AutoMigrate(&models.Feed{}, &models.Article{}, &models.SchedulerTask{}, &models.AIProvider{}, &models.AIRoute{}, &models.AIRouteProvider{}, &models.AICallLog{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
}

func TestCompleteArticleWithForceUsesAIRouterRoute(t *testing.T) {
	setupServicesTestDB(t)

	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"# Test\n\n## 导读\n- routed"}}]}`))
	}))
	defer aiServer.Close()

	provider := models.AIProvider{Name: "completion-primary", ProviderType: airouter.ProviderTypeOpenAICompatible, BaseURL: aiServer.URL, APIKey: "token", Model: "test-model", Enabled: true}
	if err := database.DB.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	route := models.AIRoute{Name: airouter.DefaultRouteName, Capability: string(airouter.CapabilityArticleCompletion), Enabled: true, Strategy: "ordered_failover"}
	if err := database.DB.Create(&route).Error; err != nil {
		t.Fatalf("create route: %v", err)
	}
	if err := database.DB.Create(&models.AIRouteProvider{RouteID: route.ID, ProviderID: provider.ID, Priority: 1, Enabled: true}).Error; err != nil {
		t.Fatalf("create route provider: %v", err)
	}

	feed := models.Feed{Title: "Feed", URL: fmt.Sprintf("https://example.com/%s", t.Name()), ContentCompletionEnabled: true, FirecrawlEnabled: true, MaxCompletionRetries: 2}
	if err := database.DB.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}

	article := models.Article{FeedID: feed.ID, Title: "Need routing", Link: "https://example.com/a1", FirecrawlStatus: "completed", FirecrawlContent: "body", ContentStatus: "incomplete"}
	if err := database.DB.Create(&article).Error; err != nil {
		t.Fatalf("create article: %v", err)
	}

	service := NewContentCompletionService("http://localhost:11235")
	if err := service.CompleteArticleWithForce(article.ID, false); err != nil {
		t.Fatalf("complete article: %v", err)
	}

	var refreshed models.Article
	if err := database.DB.First(&refreshed, article.ID).Error; err != nil {
		t.Fatalf("reload article: %v", err)
	}
	if refreshed.ContentStatus != "complete" {
		t.Fatalf("content status = %q, want complete", refreshed.ContentStatus)
	}
	if refreshed.AIContentSummary == "" {
		t.Fatal("expected AI content summary to be populated from router")
	}
}

func TestGetOverviewCountsQueueState(t *testing.T) {
	setupServicesTestDB(t)

	feedEnabled := models.Feed{
		Title:                    "Enabled Feed",
		URL:                      "https://enabled.example/rss",
		ContentCompletionEnabled: true,
		FirecrawlEnabled:         true,
		MaxCompletionRetries:     3,
	}
	feedDisabled := models.Feed{
		Title:                    "Disabled Feed",
		URL:                      "https://disabled.example/rss",
		ContentCompletionEnabled: false,
		FirecrawlEnabled:         true,
		MaxCompletionRetries:     3,
	}

	if err := database.DB.Create(&feedEnabled).Error; err != nil {
		t.Fatalf("create enabled feed: %v", err)
	}
	if err := database.DB.Create(&feedDisabled).Error; err != nil {
		t.Fatalf("create disabled feed: %v", err)
	}

	now := time.Now()
	articles := []models.Article{
		{FeedID: feedEnabled.ID, Title: "eligible-1", Link: "https://a/1", FirecrawlStatus: "completed", ContentStatus: "incomplete", FirecrawlContent: "ready"},
		{FeedID: feedEnabled.ID, Title: "eligible-2", Link: "https://a/2", FirecrawlStatus: "completed", ContentStatus: "incomplete", FirecrawlContent: "ready"},
		{FeedID: feedEnabled.ID, Title: "processing", Link: "https://a/3", FirecrawlStatus: "completed", ContentStatus: "pending", FirecrawlContent: "ready"},
		{FeedID: feedEnabled.ID, Title: "done", Link: "https://a/4", FirecrawlStatus: "completed", ContentStatus: "complete", ContentFetchedAt: &now},
		{FeedID: feedEnabled.ID, Title: "failed", Link: "https://a/5", FirecrawlStatus: "completed", ContentStatus: "failed", CompletionError: "boom"},
		{FeedID: feedEnabled.ID, Title: "waiting crawl", Link: "https://a/6", FirecrawlStatus: "pending", ContentStatus: "incomplete"},
		{FeedID: feedDisabled.ID, Title: "feed disabled", Link: "https://a/7", FirecrawlStatus: "completed", ContentStatus: "incomplete", FirecrawlContent: "ready"},
	}

	if err := database.DB.Create(&articles).Error; err != nil {
		t.Fatalf("create articles: %v", err)
	}

	service := NewContentCompletionService("http://localhost:11235")
	overview, err := service.GetOverview()
	if err != nil {
		t.Fatalf("get overview: %v", err)
	}

	if overview.PendingCount != 2 {
		t.Fatalf("pending count = %d, want 2", overview.PendingCount)
	}
	if overview.ProcessingCount != 1 {
		t.Fatalf("processing count = %d, want 1", overview.ProcessingCount)
	}
	if overview.CompletedCount != 1 {
		t.Fatalf("completed count = %d, want 1", overview.CompletedCount)
	}
	if overview.FailedCount != 1 {
		t.Fatalf("failed count = %d, want 1", overview.FailedCount)
	}
	if overview.BlockedCount != 2 {
		t.Fatalf("blocked count = %d, want 2", overview.BlockedCount)
	}
	if overview.TotalCount != 7 {
		t.Fatalf("total count = %d, want 7", overview.TotalCount)
	}
	if overview.BlockedReasons.WaitingForFirecrawlCount != 1 {
		t.Fatalf("waiting for firecrawl = %d, want 1", overview.BlockedReasons.WaitingForFirecrawlCount)
	}
	if overview.BlockedReasons.FeedDisabledCount != 1 {
		t.Fatalf("feed disabled = %d, want 1", overview.BlockedReasons.FeedDisabledCount)
	}
	if overview.BlockedReasons.AIUnconfiguredCount != 2 {
		t.Fatalf("ai unconfigured = %d, want 2", overview.BlockedReasons.AIUnconfiguredCount)
	}
	if overview.BlockedReasons.ReadyButMissingContentCount != 0 {
		t.Fatalf("ready but missing content = %d, want 0", overview.BlockedReasons.ReadyButMissingContentCount)
	}
	if overview.LiveProcessingCount != 0 {
		t.Fatalf("live processing count = %d, want 0", overview.LiveProcessingCount)
	}
	if overview.StaleProcessingCount != 1 {
		t.Fatalf("stale processing count = %d, want 1", overview.StaleProcessingCount)
	}
	if overview.StaleProcessingArticle == nil || overview.StaleProcessingArticle.Title != "processing" {
		t.Fatalf("stale processing article = %#v, want processing", overview.StaleProcessingArticle)
	}
}
