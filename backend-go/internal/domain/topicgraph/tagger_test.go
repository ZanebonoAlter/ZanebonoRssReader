package topicgraph

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func setupTopicTaggerTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:topic_tagger_test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.AISettings{}, &models.AISummary{}, &models.TopicTag{}, &models.TopicTagEmbedding{}, &models.AISummaryTopic{}))
	database.DB = db
}

func TestTagSummaryPersistsTopics(t *testing.T) {
	setupTopicTaggerTestDB(t)

	summary := models.AISummary{
		Title:     "OpenAI ships agent runtime",
		Summary:   "OpenAI expands AI agent workflows.",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, database.DB.Create(&summary).Error)

	// Test using legacy extraction (since we can't mock the TagExtractor easily)
	// This tests the fallback path which uses ExtractTopics from extractor.go
	err := TagSummary(&summary)
	require.NoError(t, err)

	var tags []models.TopicTag
	require.NoError(t, database.DB.Order("slug asc").Find(&tags).Error)
	require.NotEmpty(t, tags, "Should have created tags from summary")

	var links []models.AISummaryTopic
	require.NoError(t, database.DB.Order("score desc").Find(&links).Error)
	require.NotEmpty(t, links, "Should have created tag links")
}

func TestTagSummaryFallsBackWithoutAIConfig(t *testing.T) {
	setupTopicTaggerTestDB(t)

	// Create a summary with NVIDIA-related content
	summary := models.AISummary{
		Title:     "NVIDIA infra update",
		Summary:   "NVIDIA releases new GPU infrastructure updates for AI workloads.",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Feed:      &models.Feed{Title: "NVIDIA Research"},
	}
	require.NoError(t, database.DB.Create(&summary).Error)

	// Run TagSummary - will fall back to heuristic extraction
	err := TagSummary(&summary)
	require.NoError(t, err)

	var tags []models.TopicTag
	require.NoError(t, database.DB.Find(&tags).Error)
	require.NotEmpty(t, tags, "Should have tags from heuristic extraction")
	require.Contains(t, collectTopicTagSlugs(tags), "nvidia")
}

func TestFindOrCreateTag(t *testing.T) {
	setupTopicTaggerTestDB(t)

	// Test creating new tag
	tag := TopicTag{
		Label:    "Test Tag",
		Category: "keyword",
		Slug:     "test-tag",
		Score:    0.8,
	}

	created, err := findOrCreateTag(tag, "llm")
	require.NoError(t, err)
	require.Equal(t, "test-tag", created.Slug)
	require.Equal(t, "keyword", created.Category)

	// Test finding existing tag
	found, err := findOrCreateTag(tag, "llm")
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID, "Should find same tag")
}

func TestDedupeTagsWithCategory(t *testing.T) {
	tags := []TopicTag{
		{Label: "AI", Slug: "ai", Category: "keyword", Score: 0.9},
		{Label: "AI", Slug: "ai", Category: "keyword", Score: 0.8}, // Duplicate
		{Label: "AI", Slug: "ai", Category: "event", Score: 0.7},   // Different category
		{Label: "GPT", Slug: "gpt", Category: "keyword", Score: 0.85},
	}

	result := dedupeTagsWithCategory(tags)
	require.Len(t, result, 3, "Should have 3 unique tags (category+slug)")
}

func TestLegacyExtractTopics(t *testing.T) {
	input := ExtractionInput{
		Title:    "Test Article",
		Summary:  "OpenAI releases GPT-5 with advanced reasoning capabilities.",
		FeedName: "AI News",
	}

	tags := legacyExtractTopics(input)
	require.NotEmpty(t, tags, "Should extract tags from input")

	for _, tag := range tags {
		require.NotEmpty(t, tag.Label)
		require.NotEmpty(t, tag.Slug)
		require.NotEmpty(t, tag.Category)
		require.NotEmpty(t, tag.Kind)
	}
}

func TestCategoryValidation(t *testing.T) {
	// Test category normalization
	require.Equal(t, "event", validateCategory("event"))
	require.Equal(t, "person", validateCategory("person"))
	require.Equal(t, "keyword", validateCategory("keyword"))
	require.Equal(t, "keyword", validateCategory("unknown"))
	require.Equal(t, "keyword", validateCategory(""))
}

func collectTopicTagSlugs(tags []models.TopicTag) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		result = append(result, tag.Slug)
	}
	return result
}
