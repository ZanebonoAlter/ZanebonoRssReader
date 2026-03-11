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
	require.NoError(t, db.AutoMigrate(&models.AISettings{}, &models.AISummary{}, &models.TopicTag{}, &models.AISummaryTopic{}))
	database.DB = db
}

func TestTagSummaryPersistsTopics(t *testing.T) {
	setupTopicTaggerTestDB(t)

	originalExtractor := extractTopicsWithAI
	defer func() { extractTopicsWithAI = originalExtractor }()

	extractTopicsWithAI = func(input ExtractionInput) ([]TopicTag, error) {
		return []TopicTag{
			{Label: "AI Agent", Slug: "ai-agent", Kind: "topic", Score: 0.92},
			{Label: "OpenAI", Slug: "openai", Kind: "entity", Score: 0.88},
		}, nil
	}

	summary := models.AISummary{
		Title:     "OpenAI ships agent runtime",
		Summary:   "OpenAI expands AI agent workflows.",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, database.DB.Create(&summary).Error)

	require.NoError(t, TagSummary(&summary))

	var tags []models.TopicTag
	require.NoError(t, database.DB.Order("slug asc").Find(&tags).Error)
	require.Len(t, tags, 2)
	require.Equal(t, "ai-agent", tags[0].Slug)

	var links []models.AISummaryTopic
	require.NoError(t, database.DB.Order("score desc").Find(&links).Error)
	require.Len(t, links, 2)
	require.Equal(t, summary.ID, links[0].SummaryID)
	if links[0].Score < links[1].Score {
		t.Fatalf("expected descending scores, got %v < %v", links[0].Score, links[1].Score)
	}
}

func TestTagSummaryFallsBackWithoutAIConfig(t *testing.T) {
	setupTopicTaggerTestDB(t)

	originalExtractor := extractTopicsWithAI
	defer func() { extractTopicsWithAI = originalExtractor }()

	extractTopicsWithAI = func(input ExtractionInput) ([]TopicTag, error) {
		return nil, errTopicAIUnavailable
	}

	summary := models.AISummary{
		Title:     "NVIDIA infra update",
		Summary:   "Short update",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Feed:      &models.Feed{Title: "NVIDIA Research"},
	}
	require.NoError(t, database.DB.Create(&summary).Error)

	require.NoError(t, TagSummary(&summary))

	var tags []models.TopicTag
	require.NoError(t, database.DB.Find(&tags).Error)
	require.NotEmpty(t, tags)
	require.Contains(t, collectTopicTagSlugs(tags), "nvidia")
}

func collectTopicTagSlugs(tags []models.TopicTag) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		result = append(result, tag.Slug)
	}
	return result
}
