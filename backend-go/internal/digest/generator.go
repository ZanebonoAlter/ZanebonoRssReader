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

	summaries, err := g.fetchSummariesInRange(startTime, endTime)
	if err != nil {
		return nil, err
	}

	return g.groupByCategory(summaries), nil
}

func (g *DigestGenerator) GenerateWeeklyDigest(date time.Time) ([]CategoryDigest, error) {
	daysSinceMonday := (int(date.Weekday()) + 6) % 7
	monday := date.AddDate(0, 0, -daysSinceMonday)
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, date.Location())
	sunday := monday.AddDate(0, 0, 7)

	summaries, err := g.fetchSummariesInRange(monday, sunday)
	if err != nil {
		return nil, err
	}

	return g.groupByCategory(summaries), nil
}

func (g *DigestGenerator) fetchSummariesInRange(start, end time.Time) ([]models.AISummary, error) {
	var summaries []models.AISummary
	err := database.DB.Where("created_at >= ? AND created_at < ?", start, end).
		Preload("Feed").
		Preload("Category").
		Find(&summaries).Error
	return summaries, err
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
