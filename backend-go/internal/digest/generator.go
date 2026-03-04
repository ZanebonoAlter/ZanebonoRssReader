package digest

import (
	"my-robot-backend/internal/models"
	"my-robot-backend/pkg/database"
	"time"
)

type DigestGenerator struct {
	config *DigestConfig
}

func NewDigestGenerator(config *DigestConfig) *DigestGenerator {
	return &DigestGenerator{config: config}
}

type CategoryDigest struct {
	CategoryName string
	CategoryID   uint
	FeedCount    int
	AISummaries  []models.AISummary
}

func (g *DigestGenerator) GenerateDailyDigest(date time.Time) ([]CategoryDigest, error) {
	startTime := date.Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)

	var summaries []models.AISummary
	err := database.DB.Where("created_at >= ? AND created_at < ?", startTime, endTime).
		Preload("Feed").
		Preload("Category").
		Find(&summaries).Error

	if err != nil {
		return nil, err
	}

	return g.groupByCategory(summaries), nil
}

func (g *DigestGenerator) GenerateWeeklyDigest(date time.Time) ([]CategoryDigest, error) {
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := date.AddDate(0, 0, -weekday+1)
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.Local)
	sunday := monday.AddDate(0, 0, 7)

	var summaries []models.AISummary
	err := database.DB.Where("created_at >= ? AND created_at < ?", monday, sunday).
		Preload("Feed").
		Preload("Category").
		Find(&summaries).Error

	if err != nil {
		return nil, err
	}

	return g.groupByCategory(summaries), nil
}

func (g *DigestGenerator) groupByCategory(summaries []models.AISummary) []CategoryDigest {
	categoryMap := make(map[uint]*CategoryDigest)

	for _, summary := range summaries {
		categoryID := uint(0)
		categoryName := "未分类"

		if summary.Category != nil {
			categoryID = summary.Category.ID
			categoryName = summary.Category.Name
		}

		if _, exists := categoryMap[categoryID]; !exists {
			categoryMap[categoryID] = &CategoryDigest{
				CategoryName: categoryName,
				CategoryID:   categoryID,
				FeedCount:    0,
				AISummaries:  []models.AISummary{},
			}
		}

		categoryMap[categoryID].AISummaries = append(categoryMap[categoryID].AISummaries, summary)
		categoryMap[categoryID].FeedCount++
	}

	result := make([]CategoryDigest, 0, len(categoryMap))
	for _, digest := range categoryMap {
		result = append(result, *digest)
	}

	return result
}
