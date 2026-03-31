package feeds

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func setupFeedsTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	database.DB = db
	if err := database.DB.AutoMigrate(&models.Feed{}, &models.Article{}, &models.TopicTag{}, &models.ArticleTopicTag{}, &models.FirecrawlJob{}, &models.TagJob{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
}

func TestBuildArticleFromEntryTracksOnlyRunnableStates(t *testing.T) {
	service := NewFeedService()
	entry := ParsedEntry{
		Title:       "Fresh News",
		Description: "desc",
		Content:     "content",
		Link:        "https://example.com/article",
		Author:      "bot",
	}

	fullPipelineFeed := models.Feed{ID: 1, FirecrawlEnabled: true, ArticleSummaryEnabled: true}
	fullPipelineArticle := service.buildArticleFromEntry(fullPipelineFeed, entry)
	if fullPipelineArticle.FirecrawlStatus != "pending" {
		t.Fatalf("firecrawl status = %q, want pending", fullPipelineArticle.FirecrawlStatus)
	}
	if fullPipelineArticle.SummaryStatus != "incomplete" {
		t.Fatalf("summary status = %q, want incomplete", fullPipelineArticle.SummaryStatus)
	}

	manualOnlyFeed := models.Feed{ID: 2, FirecrawlEnabled: false, ArticleSummaryEnabled: true}
	manualOnlyArticle := service.buildArticleFromEntry(manualOnlyFeed, entry)
	if manualOnlyArticle.FirecrawlStatus != "" {
		t.Fatalf("firecrawl status = %q, want empty", manualOnlyArticle.FirecrawlStatus)
	}
	if manualOnlyArticle.SummaryStatus != "complete" {
		t.Fatalf("summary status = %q, want complete", manualOnlyArticle.SummaryStatus)
	}

	disabledFeed := models.Feed{ID: 3, FirecrawlEnabled: false, ArticleSummaryEnabled: false}
	disabledArticle := service.buildArticleFromEntry(disabledFeed, entry)
	if disabledArticle.FirecrawlStatus != "" {
		t.Fatalf("disabled firecrawl status = %q, want empty", disabledArticle.FirecrawlStatus)
	}
	if disabledArticle.SummaryStatus != "complete" {
		t.Fatalf("disabled summary status = %q, want complete", disabledArticle.SummaryStatus)
	}
}

func TestCleanupOldArticlesKeepsActiveCompletionArticles(t *testing.T) {
	setupFeedsTestDB(t)

	service := NewFeedService()
	feed := models.Feed{
		Title:                 "Feed",
		URL:                   fmt.Sprintf("https://example.com/%s", t.Name()),
		MaxArticles:           2,
		FirecrawlEnabled:      true,
		ArticleSummaryEnabled: true,
	}
	if err := database.DB.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}

	now := time.Now()
	articles := []models.Article{
		{FeedID: feed.ID, Title: "new complete", Link: "https://example.com/new", PubDate: ptrTime(now.Add(-1 * time.Hour)), SummaryStatus: "complete", FirecrawlStatus: "completed"},
		{FeedID: feed.ID, Title: "middle complete", Link: "https://example.com/middle", PubDate: ptrTime(now.Add(-2 * time.Hour)), SummaryStatus: "complete", FirecrawlStatus: "completed"},
		{FeedID: feed.ID, Title: "old incomplete", Link: "https://example.com/old", PubDate: ptrTime(now.Add(-3 * time.Hour)), SummaryStatus: "incomplete", FirecrawlStatus: "completed", FirecrawlContent: "ready"},
	}
	if err := database.DB.Create(&articles).Error; err != nil {
		t.Fatalf("create articles: %v", err)
	}

	service.cleanupOldArticles(&feed)

	var remaining []models.Article
	if err := database.DB.Where("feed_id = ?", feed.ID).Order("pub_date DESC").Find(&remaining).Error; err != nil {
		t.Fatalf("load remaining articles: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("remaining articles = %d, want 2", len(remaining))
	}

	titles := map[string]bool{}
	for _, article := range remaining {
		titles[article.Title] = true
	}

	if !titles["old incomplete"] {
		t.Fatalf("expected incomplete article to be preserved, remaining = %#v", titles)
	}
	if titles["middle complete"] {
		t.Fatalf("expected oldest removable complete article to be deleted, remaining = %#v", titles)
	}
}

func TestRefreshFeedEnqueuesTagJobWhenCompletionDisabled(t *testing.T) {
	setupFeedsTestDB(t)

	rssServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>OpenAI Feed</title>
    <description>Feed for tests</description>
    <link>https://example.com</link>
    <item>
      <title>OpenAI launches new AI agent runtime</title>
      <link>https://example.com/openai-agent</link>
      <description>OpenAI agentic workflow update</description>
      <pubDate>Sun, 22 Mar 2026 09:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`))
	}))
	defer rssServer.Close()

	feed := models.Feed{
		Title:                 "Seed Feed",
		URL:                   rssServer.URL,
		MaxArticles:           10,
		FirecrawlEnabled:      false,
		ArticleSummaryEnabled: false,
	}
	if err := database.DB.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}

	service := NewFeedService()
	if err := service.RefreshFeed(context.Background(), feed.ID); err != nil {
		t.Fatalf("refresh feed: %v", err)
	}

	var article models.Article
	if err := database.DB.First(&article).Error; err != nil {
		t.Fatalf("load article: %v", err)
	}

	var jobCount int64
	if err := database.DB.Model(&models.TagJob{}).Where("article_id = ?", article.ID).Count(&jobCount).Error; err != nil {
		t.Fatalf("count tag jobs: %v", err)
	}
	if jobCount != 1 {
		t.Fatalf("tag job count = %d, want 1", jobCount)
	}
}

func TestRefreshFeedEnqueuesFirecrawlJobWhenEnabled(t *testing.T) {
	setupFeedsTestDB(t)

	rssServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Queued Feed</title>
    <description>Feed for tests</description>
    <link>https://example.com</link>
    <item>
      <title>Queued article</title>
      <link>https://example.com/queued</link>
      <description>queued desc</description>
      <pubDate>Sun, 22 Mar 2026 09:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`))
	}))
	defer rssServer.Close()

	feed := models.Feed{
		Title:            "Queued Feed",
		URL:              rssServer.URL,
		MaxArticles:      10,
		FirecrawlEnabled: true,
	}
	if err := database.DB.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}

	service := NewFeedService()
	if err := service.RefreshFeed(context.Background(), feed.ID); err != nil {
		t.Fatalf("refresh feed: %v", err)
	}

	var article models.Article
	if err := database.DB.First(&article).Error; err != nil {
		t.Fatalf("load article: %v", err)
	}

	var count int64
	if err := database.DB.Model(&models.FirecrawlJob{}).Where("article_id = ?", article.ID).Count(&count).Error; err != nil {
		t.Fatalf("count firecrawl jobs: %v", err)
	}
	if count != 1 {
		t.Fatalf("firecrawl job count = %d, want 1", count)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
