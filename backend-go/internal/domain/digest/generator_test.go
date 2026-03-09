package digest

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"my-robot-backend/internal/domain/models"
)

func TestGroupByCategory(t *testing.T) {
	config := &DigestConfig{}
	generator := NewDigestGenerator(config)

	summaries := []models.AISummary{
		{
			ID:       1,
			Title:    "Test Summary 1",
			Category: &models.Category{ID: 1, Name: "AI技术"},
		},
		{
			ID:       2,
			Title:    "Test Summary 2",
			Category: &models.Category{ID: 1, Name: "AI技术"},
		},
		{
			ID:       3,
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
			Title:    "Categorized Item",
			Category: &models.Category{ID: 1, Name: "AI技术"},
		},
		{
			ID:       2,
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
		{ID: 1, Title: "Item 1", Category: &models.Category{ID: 1, Name: "Tech"}},
		{ID: 2, Title: "Item 2", Category: &models.Category{ID: 1, Name: "Tech"}},
		{ID: 3, Title: "Item 3", Category: &models.Category{ID: 1, Name: "Tech"}},
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
		{ID: 1, Category: &models.Category{ID: 1, Name: "AI技术"}},
		{ID: 2, Category: &models.Category{ID: 1, Name: "AI技术"}},
		{ID: 3, Category: &models.Category{ID: 2, Name: "前端开发"}},
		{ID: 4, Category: &models.Category{ID: 3, Name: "后端开发"}},
		{ID: 5, Category: &models.Category{ID: 1, Name: "AI技术"}},
		{ID: 6, Category: nil},
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
		{ID: 1, Category: &models.Category{ID: 3, Name: "Cat C"}},
		{ID: 2, Category: &models.Category{ID: 1, Name: "Cat A"}},
		{ID: 3, Category: &models.Category{ID: 2, Name: "Cat B"}},
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
