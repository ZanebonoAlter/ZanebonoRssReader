package summaries

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func setupSummaryQueueTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:summary_queue_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	database.DB = db
	if err := database.DB.AutoMigrate(&models.Category{}, &models.Feed{}); err != nil {
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
