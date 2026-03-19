package jobs

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/contentprocessing"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func setupSchedulersTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	database.DB = db
	if err := database.DB.AutoMigrate(&models.Feed{}, &models.Article{}, &models.SchedulerTask{}, &models.AISettings{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
}

func TestContentCompletionSchedulerGetStatusIncludesOverviewAndCurrentArticle(t *testing.T) {
	setupSchedulersTestDB(t)

	feed := models.Feed{
		Title:                 "Feed",
		URL:                   "https://feed.example/rss",
		ArticleSummaryEnabled: true,
		FirecrawlEnabled:      true,
		MaxCompletionRetries:  3,
	}
	if err := database.DB.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}

	article := models.Article{
		FeedID:           feed.ID,
		Title:            "Queue me",
		Link:             "https://feed.example/a1",
		FirecrawlStatus:  "completed",
		SummaryStatus:    "incomplete",
		FirecrawlContent: "ready",
	}
	if err := database.DB.Create(&article).Error; err != nil {
		t.Fatalf("create article: %v", err)
	}

	nextRun := time.Now().Add(time.Hour)
	task := models.SchedulerTask{
		Name:              "ai_summary",
		Description:       "AI summarize Firecrawl content",
		CheckInterval:     3600,
		Status:            "running",
		NextExecutionTime: &nextRun,
	}
	if err := database.DB.Create(&task).Error; err != nil {
		t.Fatalf("create scheduler task: %v", err)
	}

	scheduler := &ContentCompletionScheduler{
		completionService: contentprocessing.NewContentCompletionService("http://localhost:11235"),
		checkInterval:     time.Hour,
		taskName:          "ai_summary",
		isExecuting:       true,
		currentArticle: &contentprocessing.ContentCompletionArticleRef{
			ID:     article.ID,
			FeedID: article.FeedID,
			Title:  article.Title,
		},
	}

	status := scheduler.GetStatus()
	if status["is_executing"] != true {
		t.Fatalf("is_executing = %v, want true", status["is_executing"])
	}

	overview, ok := status["overview"].(map[string]interface{})
	if !ok {
		t.Fatalf("overview missing or invalid: %#v", status["overview"])
	}
	if overview["pending_count"] != 1 {
		t.Fatalf("pending_count = %v, want 1", overview["pending_count"])
	}

	current, ok := status["current_article"].(*contentprocessing.ContentCompletionArticleRef)
	if !ok {
		t.Fatalf("current article missing or invalid: %#v", status["current_article"])
	}
	if current.Title != "Queue me" {
		t.Fatalf("current article title = %q, want Queue me", current.Title)
	}

	if status["live_processing_count"] != 1 {
		t.Fatalf("live_processing_count = %v, want 1", status["live_processing_count"])
	}
	if status["stale_processing_count"] != 0 {
		t.Fatalf("stale_processing_count = %v, want 0", status["stale_processing_count"])
	}
}

func TestParseLastRunSummaryFromSchedulerTask(t *testing.T) {
	nextRun := time.Now().Add(time.Hour)
	task := models.SchedulerTask{
		Name:                "ai_summary",
		NextExecutionTime:   &nextRun,
		LastExecutionResult: `{"started_at":"2026-03-08T10:00:00+08:00","finished_at":"2026-03-08T10:01:00+08:00","completed_count":3,"failed_count":1,"blocked_count":2,"stale_processing_count":1,"error_samples":[{"article_id":18118,"message":"unexpected EOF","category":"network"}]}`,
	}

	summary := parseLastRunSummary(task)
	if summary == nil {
		t.Fatal("expected parsed last run summary")
	}
	if summary.CompletedCount != 3 {
		t.Fatalf("completed_count = %d, want 3", summary.CompletedCount)
	}
	if len(summary.ErrorSamples) != 1 || summary.ErrorSamples[0].Category != "network" {
		t.Fatalf("error samples = %#v, want network category", summary.ErrorSamples)
	}
}
