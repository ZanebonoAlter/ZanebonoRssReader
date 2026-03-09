package digest

import (
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
	"sort"
	"time"
)

var digestCST = time.FixedZone("CST", 8*3600)

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

func normalizeDigestDate(date time.Time) time.Time {
	return date.In(digestCST)
}

func startOfDigestDay(date time.Time) time.Time {
	current := normalizeDigestDate(date)
	return time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, digestCST)
}

func (g *DigestGenerator) GenerateDailyDigest(date time.Time) ([]CategoryDigest, error) {
	startTime := startOfDigestDay(date)
	endTime := startTime.AddDate(0, 0, 1)

	summaries, err := g.fetchSummariesInRange(startTime, endTime)
	if err != nil {
		return nil, err
	}

	return g.groupByCategory(summaries), nil
}

func (g *DigestGenerator) GenerateWeeklyDigest(date time.Time) ([]CategoryDigest, error) {
	current := normalizeDigestDate(date)
	daysSinceMonday := (int(current.Weekday()) + 6) % 7
	monday := startOfDigestDay(current.AddDate(0, 0, -daysSinceMonday))
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
		Order("created_at DESC").
		Find(&summaries).Error
	return summaries, err
}

func (g *DigestGenerator) groupByCategory(summaries []models.AISummary) []CategoryDigest {
	categoryMap := make(map[uint]*CategoryDigest)
	feedSeenMap := make(map[uint]map[uint]struct{})

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
			feedSeenMap[categoryID] = make(map[uint]struct{})
		}

		categoryMap[categoryID].AISummaries = append(categoryMap[categoryID].AISummaries, summary)

		if summary.FeedID != nil {
			if _, exists := feedSeenMap[categoryID][*summary.FeedID]; !exists {
				feedSeenMap[categoryID][*summary.FeedID] = struct{}{}
				categoryMap[categoryID].FeedCount++
			}
		}
	}

	result := make([]CategoryDigest, 0, len(categoryMap))
	for _, digest := range categoryMap {
		sort.SliceStable(digest.AISummaries, func(i, j int) bool {
			return digest.AISummaries[i].CreatedAt.After(digest.AISummaries[j].CreatedAt)
		})
		result = append(result, *digest)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if len(result[i].AISummaries) == len(result[j].AISummaries) {
			return result[i].CategoryName < result[j].CategoryName
		}
		return len(result[i].AISummaries) > len(result[j].AISummaries)
	})

	return result
}
