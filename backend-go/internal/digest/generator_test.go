package digest

import (
	"github.com/stretchr/testify/assert"
	"my-robot-backend/internal/models"
	"testing"
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
	assert.Equal(t, "AI技术", result[0].CategoryName)
	assert.Equal(t, 2, result[0].FeedCount)
	assert.Equal(t, "前端开发", result[1].CategoryName)
	assert.Equal(t, 1, result[1].FeedCount)
}
