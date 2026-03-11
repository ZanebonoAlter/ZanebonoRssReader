package summaries

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/preferences"
)

func setupAISummaryPromptBuilderTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&models.Category{}, &models.Feed{}, &models.UserPreference{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	return db
}

func TestBuildPersonalizedPromptIncludesReadingHabitsForFractionalAverages(t *testing.T) {
	db := setupAISummaryPromptBuilderTestDB(t)
	prefService := preferences.NewPreferenceService(db)
	builder := NewAISummaryPromptBuilder(prefService, db)

	feed := models.Feed{Title: "Feed A", URL: "https://example.com/feed-a"}
	if err := db.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}

	prefs := []models.UserPreference{
		{FeedID: &feed.ID, PreferenceScore: 0.9, AvgReadingTime: 7, ScrollDepthAvg: 72},
		{FeedID: &feed.ID, PreferenceScore: 0.8, AvgReadingTime: 8, ScrollDepthAvg: 88},
	}
	if err := db.Create(&prefs).Error; err != nil {
		t.Fatalf("create preferences: %v", err)
	}

	prompt, context, err := builder.BuildPersonalizedPrompt(feed.Title, "", "Article body", 1, "en")
	if err != nil {
		t.Fatalf("BuildPersonalizedPrompt returned error: %v", err)
	}

	if !context.Personalized {
		t.Fatalf("expected personalized context, got %#v", context)
	}

	if !strings.Contains(prompt, "### Reading Habits") {
		t.Fatalf("expected reading habits section in prompt, got %q", prompt)
	}

	if !strings.Contains(prompt, "Average reading time: 8 seconds") {
		t.Fatalf("expected rounded average reading time in prompt, got %q", prompt)
	}
}

func TestBuildPersonalizedPromptDeduplicatesPreferredFeedsAndCategories(t *testing.T) {
	db := setupAISummaryPromptBuilderTestDB(t)
	prefService := preferences.NewPreferenceService(db)
	builder := NewAISummaryPromptBuilder(prefService, db)

	category := models.Category{Name: "Tech"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	feed := models.Feed{Title: "Feed A", URL: "https://example.com/feed-a", CategoryID: &category.ID}
	if err := db.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}

	prefs := []models.UserPreference{
		{FeedID: &feed.ID, PreferenceScore: 0.9, AvgReadingTime: 10, ScrollDepthAvg: 60},
		{FeedID: &feed.ID, PreferenceScore: 0.8, AvgReadingTime: 12, ScrollDepthAvg: 65},
		{CategoryID: &category.ID, PreferenceScore: 0.7, AvgReadingTime: 10, ScrollDepthAvg: 50},
		{CategoryID: &category.ID, PreferenceScore: 0.6, AvgReadingTime: 14, ScrollDepthAvg: 55},
	}
	if err := db.Create(&prefs).Error; err != nil {
		t.Fatalf("create preferences: %v", err)
	}

	prompt, _, err := builder.BuildPersonalizedPrompt(feed.Title, category.Name, "Article body", 1, "en")
	if err != nil {
		t.Fatalf("BuildPersonalizedPrompt returned error: %v", err)
	}

	if count := strings.Count(prompt, "- Feed A\n"); count != 1 {
		t.Fatalf("expected preferred feed once, got %d in %q", count, prompt)
	}

	if count := strings.Count(prompt, "- Tech\n"); count != 1 {
		t.Fatalf("expected preferred category once, got %d in %q", count, prompt)
	}
}
