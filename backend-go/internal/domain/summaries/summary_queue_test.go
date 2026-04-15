package summaries

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
)

func setupSummaryQueueTestDB(t *testing.T) {
	t.Helper()

	dbName := url.QueryEscape(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	database.DB = db
	if err := database.DB.AutoMigrate(&models.Category{}, &models.Feed{}, &models.UserPreference{}, &models.Article{}, &models.AISummary{}, &models.TopicTag{}, &models.AISummaryTopic{}, &models.ArticleTopicTag{}, &models.AIProvider{}, &models.AIRoute{}, &models.AIRouteProvider{}, &models.AICallLog{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
}

func TestBuildQueueArticleTextPrefersFirecrawlContent(t *testing.T) {
	article := models.Article{
		Title:            "Firecrawl First",
		Description:      "desc body",
		Content:          "rss body",
		FirecrawlContent: "firecrawl body",
		Link:             "https://example.com/a1",
	}

	text := buildQueueArticleText(article)

	if !strings.Contains(text, "内容: firecrawl body") {
		t.Fatalf("expected firecrawl content, got %q", text)
	}
	if strings.Contains(text, "内容: rss body") {
		t.Fatalf("did not expect rss content when firecrawl content exists, got %q", text)
	}
}

func TestBuildQueueArticleTextFallsBackToContentThenDescription(t *testing.T) {
	article := models.Article{
		Title:       "Fallback",
		Description: "desc body",
		Content:     "rss body",
		Link:        "https://example.com/a2",
	}

	text := buildQueueArticleText(article)
	if !strings.Contains(text, "内容: rss body") {
		t.Fatalf("expected rss content fallback, got %q", text)
	}

	article.Content = ""
	text = buildQueueArticleText(article)
	if !strings.Contains(text, "描述: desc body") {
		t.Fatalf("expected description fallback, got %q", text)
	}
	if strings.Contains(text, "内容:") {
		t.Fatalf("did not expect empty content section, got %q", text)
	}
}

func TestSubmitBatchUsesFeedIDsWhenProvided(t *testing.T) {
	setupSummaryQueueTestDB(t)

	category := models.Category{Name: "Tech"}
	if err := database.DB.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	feedA := models.Feed{Title: "Feed A", URL: "https://example.com/a", CategoryID: &category.ID, AISummaryEnabled: true}
	feedB := models.Feed{Title: "Feed B", URL: "https://example.com/b", CategoryID: &category.ID, AISummaryEnabled: true}
	if err := database.DB.Create(&feedA).Error; err != nil {
		t.Fatalf("create feedA: %v", err)
	}
	if err := database.DB.Create(&feedB).Error; err != nil {
		t.Fatalf("create feedB: %v", err)
	}

	queue := GetSummaryQueue()
	batch := queue.SubmitBatch(nil, []uint{feedB.ID}, AIConfig{APIKey: "token"})

	if batch.TotalJobs != 1 {
		t.Fatalf("total jobs = %d, want 1", batch.TotalJobs)
	}
	if len(batch.Jobs) != 1 || batch.Jobs[0].FeedID == nil || *batch.Jobs[0].FeedID != feedB.ID {
		t.Fatalf("jobs = %#v, want only feedB", batch.Jobs)
	}
}

func TestGenerateSummaryForFeedSkipsExistingBatchAndLogsMetadata(t *testing.T) {
	setupSummaryQueueTestDB(t)

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

	articles := make([]models.Article, 0, 11)
	pub := time.Now()
	for i := 0; i < 11; i++ {
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

	firstBatchIDs := make([]uint, 0, 10)
	for i := 0; i < 10; i++ {
		firstBatchIDs = append(firstBatchIDs, articles[i].ID)
	}
	firstBatchJSON, _ := json.Marshal(firstBatchIDs)
	existing := models.AISummary{FeedID: &feed.ID, Title: "Existing batch", Summary: "saved", Articles: string(firstBatchJSON), ArticleCount: len(firstBatchIDs), TimeRange: 180}
	if err := database.DB.Create(&existing).Error; err != nil {
		t.Fatalf("create existing summary: %v", err)
	}

	queue := &SummaryQueue{}
	result, err := queue.generateSummaryForFeed(&feed.ID, nil, feed.Title, "", AIConfig{TimeRange: 180})
	if err != nil {
		t.Fatalf("generate summary: %v", err)
	}
	if result == nil {
		t.Fatal("expected a generated summary for missing batch")
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

	var refreshedArticles []models.Article
	if err := database.DB.Order("id ASC").Find(&refreshedArticles).Error; err != nil {
		t.Fatalf("load articles: %v", err)
	}
	for i := 0; i < 10; i++ {
		if refreshedArticles[i].FeedSummaryID == nil || *refreshedArticles[i].FeedSummaryID != existing.ID {
			t.Fatalf("article %d feed_summary_id = %v, want %d", refreshedArticles[i].ID, refreshedArticles[i].FeedSummaryID, existing.ID)
		}
		if refreshedArticles[i].FeedSummaryGeneratedAt == nil {
			t.Fatalf("article %d feed_summary_generated_at is nil", refreshedArticles[i].ID)
		}
	}
	if refreshedArticles[10].FeedSummaryID == nil || *refreshedArticles[10].FeedSummaryID != summaries[1].ID {
		t.Fatalf("new article feed_summary_id = %v, want %d", refreshedArticles[10].FeedSummaryID, summaries[1].ID)
	}
	if refreshedArticles[10].FeedSummaryGeneratedAt == nil {
		t.Fatal("new article feed_summary_generated_at is nil")
	}
}

func TestGenerateSummaryForFeedOnlyUsesUnmarkedArticles(t *testing.T) {
	setupSummaryQueueTestDB(t)

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

	now := time.Now()
	articles := []models.Article{
		{FeedID: feed.ID, Title: "Old", Link: "https://example.com/old", Content: "rss body", PubDate: &now},
		{FeedID: feed.ID, Title: "New", Link: "https://example.com/new", Content: "rss body", PubDate: &now},
	}
	if err := database.DB.Create(&articles).Error; err != nil {
		t.Fatalf("create articles: %v", err)
	}

	existingIDsJSON, _ := json.Marshal([]uint{articles[0].ID})
	existing := models.AISummary{FeedID: &feed.ID, Title: "Existing batch", Summary: "saved", Articles: string(existingIDsJSON), ArticleCount: 1, TimeRange: 180}
	if err := database.DB.Create(&existing).Error; err != nil {
		t.Fatalf("create existing summary: %v", err)
	}
	if err := database.DB.Model(&models.Article{}).Where("id = ?", articles[0].ID).Updates(map[string]any{
		"feed_summary_id":           existing.ID,
		"feed_summary_generated_at": now,
	}).Error; err != nil {
		t.Fatalf("mark existing article: %v", err)
	}

	queue := &SummaryQueue{}
	result, err := queue.generateSummaryForFeed(&feed.ID, nil, feed.Title, "", AIConfig{TimeRange: 180})
	if err != nil {
		t.Fatalf("generate summary: %v", err)
	}
	if result == nil {
		t.Fatal("expected a generated summary for new article")
	}

	var summaries []models.AISummary
	if err := database.DB.Order("id ASC").Find(&summaries).Error; err != nil {
		t.Fatalf("load summaries: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("summary count = %d, want 2", len(summaries))
	}
	if summaries[1].Articles != fmt.Sprintf("[%d]", articles[1].ID) {
		t.Fatalf("new summary articles = %s, want [%d]", summaries[1].Articles, articles[1].ID)
	}
}
