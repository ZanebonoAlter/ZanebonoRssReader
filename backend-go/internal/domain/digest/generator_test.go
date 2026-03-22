package digest

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func setupDigestGeneratorTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:digest_generator_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&DigestConfig{}, &models.Category{}, &models.Feed{}, &models.Article{}, &models.AISummary{}, &models.TopicTag{}, &models.AISummaryTopic{}, &models.ArticleTopicTag{}))
	database.DB = db
}

func TestGroupByCategory(t *testing.T) {
	config := &DigestConfig{}
	generator := NewDigestGenerator(config)

	summaries := []models.AISummary{
		{
			ID:       1,
			FeedID:   uintPtr(1),
			Title:    "Test Summary 1",
			Category: &models.Category{ID: 1, Name: "AI技术"},
		},
		{
			ID:       2,
			FeedID:   uintPtr(2),
			Title:    "Test Summary 2",
			Category: &models.Category{ID: 1, Name: "AI技术"},
		},
		{
			ID:       3,
			FeedID:   uintPtr(3),
			Title:    "Test Summary 3",
			Category: &models.Category{ID: 2, Name: "前端开发"},
		},
	}

	result := generator.groupByCategory(summaries)

	assert.Equal(t, 2, len(result))

	categoryMap := make(map[string]CategoryDigest)
	for _, cat := range result {
		categoryMap[cat.CategoryName] = cat
	}

	aiTech, exists := categoryMap["AI技术"]
	assert.True(t, exists)
	assert.Equal(t, 2, aiTech.FeedCount)
	assert.Equal(t, 2, len(aiTech.AISummaries))

	frontend, exists := categoryMap["前端开发"]
	assert.True(t, exists)
	assert.Equal(t, 1, frontend.FeedCount)
	assert.Equal(t, 1, len(frontend.AISummaries))
}

func TestGroupByCategory_WithNilCategory(t *testing.T) {
	config := &DigestConfig{}
	generator := NewDigestGenerator(config)

	summaries := []models.AISummary{
		{
			ID:       1,
			FeedID:   uintPtr(1),
			Title:    "Categorized Item",
			Category: &models.Category{ID: 1, Name: "AI技术"},
		},
		{
			ID:       2,
			FeedID:   uintPtr(2),
			Title:    "Uncategorized Item",
			Category: nil,
		},
	}

	result := generator.groupByCategory(summaries)

	assert.Equal(t, 2, len(result))

	categoryMap := make(map[string]CategoryDigest)
	for _, cat := range result {
		categoryMap[cat.CategoryName] = cat
	}

	uncategorized, exists := categoryMap["未分类"]
	assert.True(t, exists)
	assert.Equal(t, uint(0), uncategorized.CategoryID)
	assert.Equal(t, 1, uncategorized.FeedCount)
}

func TestGroupByCategory_EmptySummaries(t *testing.T) {
	config := &DigestConfig{}
	generator := NewDigestGenerator(config)

	result := generator.groupByCategory([]models.AISummary{})

	assert.Equal(t, 0, len(result))
}

func TestGroupByCategory_SingleCategory(t *testing.T) {
	config := &DigestConfig{}
	generator := NewDigestGenerator(config)

	summaries := []models.AISummary{
		{ID: 1, FeedID: uintPtr(1), Title: "Item 1", Category: &models.Category{ID: 1, Name: "Tech"}},
		{ID: 2, FeedID: uintPtr(2), Title: "Item 2", Category: &models.Category{ID: 1, Name: "Tech"}},
		{ID: 3, FeedID: uintPtr(3), Title: "Item 3", Category: &models.Category{ID: 1, Name: "Tech"}},
	}

	result := generator.groupByCategory(summaries)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, "Tech", result[0].CategoryName)
	assert.Equal(t, 3, result[0].FeedCount)
}

func TestCategoryDigest_MultipleItems(t *testing.T) {
	config := &DigestConfig{}
	generator := NewDigestGenerator(config)

	summaries := []models.AISummary{
		{ID: 1, FeedID: uintPtr(1), Category: &models.Category{ID: 1, Name: "AI技术"}},
		{ID: 2, FeedID: uintPtr(2), Category: &models.Category{ID: 1, Name: "AI技术"}},
		{ID: 3, FeedID: uintPtr(3), Category: &models.Category{ID: 2, Name: "前端开发"}},
		{ID: 4, FeedID: uintPtr(4), Category: &models.Category{ID: 3, Name: "后端开发"}},
		{ID: 5, FeedID: uintPtr(5), Category: &models.Category{ID: 1, Name: "AI技术"}},
		{ID: 6, FeedID: uintPtr(6), Category: nil},
	}

	result := generator.groupByCategory(summaries)

	assert.Equal(t, 4, len(result))

	categoryMap := make(map[string]CategoryDigest)
	for _, cat := range result {
		categoryMap[cat.CategoryName] = cat
	}

	assert.Equal(t, 3, categoryMap["AI技术"].FeedCount)
	assert.Equal(t, 1, categoryMap["前端开发"].FeedCount)
	assert.Equal(t, 1, categoryMap["后端开发"].FeedCount)
	assert.Equal(t, 1, categoryMap["未分类"].FeedCount)
}

func TestCategoryDigest_SortedOutput(t *testing.T) {
	config := &DigestConfig{}
	generator := NewDigestGenerator(config)

	summaries := []models.AISummary{
		{ID: 1, FeedID: uintPtr(1), Category: &models.Category{ID: 3, Name: "Cat C"}},
		{ID: 2, FeedID: uintPtr(2), Category: &models.Category{ID: 1, Name: "Cat A"}},
		{ID: 3, FeedID: uintPtr(3), Category: &models.Category{ID: 2, Name: "Cat B"}},
	}

	result := generator.groupByCategory(summaries)

	sortedResult := make([]CategoryDigest, len(result))
	copy(sortedResult, result)
	sort.Slice(sortedResult, func(i, j int) bool {
		return sortedResult[i].CategoryName < sortedResult[j].CategoryName
	})

	assert.Equal(t, 3, len(sortedResult))
	assert.Equal(t, "Cat A", sortedResult[0].CategoryName)
	assert.Equal(t, "Cat B", sortedResult[1].CategoryName)
	assert.Equal(t, "Cat C", sortedResult[2].CategoryName)
}

func uintPtr(value uint) *uint {
	return &value
}

func TestBuildPreviewAggregatesArticleTagsForDigestSummaries(t *testing.T) {
	setupDigestGeneratorTestDB(t)

	config := defaultDigestConfig()
	require.NoError(t, database.DB.Create(&config).Error)

	category := models.Category{Name: "AI", Slug: "ai", Color: "#3b6b87", Icon: "mdi:brain"}
	require.NoError(t, database.DB.Create(&category).Error)

	feed := models.Feed{Title: "OpenAI Blog", URL: "https://example.com/openai", CategoryID: &category.ID, Color: "#3b6b87", Icon: "mdi:rss"}
	require.NoError(t, database.DB.Create(&feed).Error)

	createdAt := time.Date(2026, 3, 22, 8, 0, 0, 0, time.FixedZone("CST", 8*3600))
	articles := []models.Article{
		{FeedID: feed.ID, Title: "Runtime launch", Link: "https://example.com/runtime", CreatedAt: createdAt},
		{FeedID: feed.ID, Title: "OpenAI memo", Link: "https://example.com/memo", CreatedAt: createdAt.Add(30 * time.Minute)},
	}
	for i := range articles {
		require.NoError(t, database.DB.Create(&articles[i]).Error)
	}

	topicTags := []models.TopicTag{
		{Label: "AI Agent", Slug: "ai-agent", Category: models.TagCategoryKeyword, Kind: "keyword"},
		{Label: "OpenAI", Slug: "openai", Category: models.TagCategoryKeyword, Kind: "keyword"},
	}
	for i := range topicTags {
		require.NoError(t, database.DB.Create(&topicTags[i]).Error)
	}

	require.NoError(t, database.DB.Create(&models.ArticleTopicTag{ArticleID: articles[0].ID, TopicTagID: topicTags[0].ID, Score: 1.0, Source: "llm"}).Error)
	require.NoError(t, database.DB.Create(&models.ArticleTopicTag{ArticleID: articles[1].ID, TopicTagID: topicTags[0].ID, Score: 0.8, Source: "llm"}).Error)
	require.NoError(t, database.DB.Create(&models.ArticleTopicTag{ArticleID: articles[1].ID, TopicTagID: topicTags[1].ID, Score: 0.7, Source: "llm"}).Error)

	summary := models.AISummary{
		FeedID:       &feed.ID,
		CategoryID:   &category.ID,
		Title:        "AI Agent 日报",
		Summary:      "整理当天话题",
		Articles:     fmt.Sprintf("[%d,%d]", articles[0].ID, articles[1].ID),
		ArticleCount: 2,
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}
	require.NoError(t, database.DB.Create(&summary).Error)

	preview, _, _, err := buildPreview("daily", createdAt)
	require.NoError(t, err)
	require.Len(t, preview.Categories, 1)
	require.Len(t, preview.Categories[0].Summaries, 1)

	aggregatedTags := preview.Categories[0].Summaries[0].AggregatedTags
	require.Len(t, aggregatedTags, 2)
	require.Equal(t, "ai-agent", aggregatedTags[0].Slug)
	require.Equal(t, 2, aggregatedTags[0].ArticleCount)
	require.Equal(t, "openai", aggregatedTags[1].Slug)
	require.Equal(t, 1, aggregatedTags[1].ArticleCount)
}
