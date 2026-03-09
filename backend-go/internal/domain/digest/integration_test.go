package digest

import (
	"my-robot-backend/internal/domain/models"
	"testing"
	"time"
)

func TestDigestWorkflow(t *testing.T) {
	config := &DigestConfig{
		DailyEnabled:    true,
		DailyTime:       "09:00",
		FeishuEnabled:   false,
		ObsidianEnabled: false,
	}

	generator := NewDigestGenerator(config)
	digests, err := generator.GenerateDailyDigest(time.Now())
	if err != nil {
		t.Fatalf("Failed to generate daily digest: %v", err)
	}

	if digests == nil {
		t.Fatal("Digests should not be nil")
	}

	for _, digest := range digests {
		if digest.CategoryName == "" {
			t.Error("CategoryName should not be empty")
		}
		if digest.FeedCount < 0 {
			t.Error("FeedCount should be non-negative")
		}
	}
}

func TestDigestIntegrationWorkflow(t *testing.T) {
	config := &DigestConfig{}
	generator := NewDigestGenerator(config)

	category1 := &models.Category{ID: 1, Name: "AI技术"}
	category2 := &models.Category{ID: 2, Name: "前端开发"}
	feed1 := &models.Feed{ID: 1, Title: "Feed 1"}

	summaries := []models.AISummary{
		{
			ID:       1,
			Title:    "Test Summary 1",
			Category: category1,
			Feed:     feed1,
		},
		{
			ID:       2,
			Title:    "Test Summary 2",
			Category: category1,
			Feed:     feed1,
		},
		{
			ID:       3,
			Title:    "Test Summary 3",
			Category: category2,
			Feed:     feed1,
		},
	}

	result := generator.groupByCategory(summaries)

	if len(result) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(result))
	}

	for _, catDigest := range result {
		if catDigest.CategoryName == "AI技术" && catDigest.FeedCount != 2 {
			t.Errorf("AI技术 category should have 2 feeds, got %d", catDigest.FeedCount)
		}
		if catDigest.CategoryName == "前端开发" && catDigest.FeedCount != 1 {
			t.Errorf("前端开发 category should have 1 feed, got %d", catDigest.FeedCount)
		}
	}
}
