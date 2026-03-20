package topictypes

import (
	"fmt"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

var TopicGraphCST = time.FixedZone("CST", 8*3600)

func ParseAnchorDate(value string) (time.Time, error) {
	if value == "" {
		return time.Now().In(TopicGraphCST), nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, TopicGraphCST)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

func ResolveWindow(kind string, anchor time.Time) (time.Time, time.Time, string, error) {
	current := anchor.In(TopicGraphCST)
	dayStart := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, TopicGraphCST)

	switch kind {
	case "daily":
		return dayStart, dayStart.AddDate(0, 0, 1), fmt.Sprintf("%s 当日", dayStart.Format("2006-01-02")), nil
	case "weekly":
		daysSinceMonday := (int(current.Weekday()) + 6) % 7
		weekStart := dayStart.AddDate(0, 0, -daysSinceMonday)
		weekEnd := weekStart.AddDate(0, 0, 7)
		return weekStart, weekEnd, fmt.Sprintf("%s - %s", weekStart.Format("01-02"), weekEnd.AddDate(0, 0, -1).Format("01-02")), nil
	default:
		return time.Time{}, time.Time{}, "", fmt.Errorf("unsupported topic graph type: %s", kind)
	}
}

func FetchArticlesForSummaries(summaries []models.AISummary) map[uint][]TopicArticleCard {
	summaryIDs := make([]uint, len(summaries))
	for i, s := range summaries {
		summaryIDs[i] = s.ID
	}

	var links []struct {
		SummaryID uint
		ArticleID uint
	}

	if len(summaryIDs) > 0 {
		database.DB.Table("ai_summary_articles").
			Where("summary_id IN ?", summaryIDs).
			Find(&links)
	}

	articleIDs := make([]uint, 0, len(links))
	linkMap := make(map[uint][]uint) // summaryID -> []articleID
	for _, l := range links {
		articleIDs = append(articleIDs, l.ArticleID)
		linkMap[l.SummaryID] = append(linkMap[l.SummaryID], l.ArticleID)
	}

	articlesByID := make(map[uint]TopicArticleCard)
	if len(articleIDs) > 0 {
		var articles []struct {
			ID    uint
			Title string
			Link  string
		}
		database.DB.Model(&models.Article{}).
			Select("id, title, link").
			Where("id IN ?", articleIDs).
			Find(&articles)
		for _, a := range articles {
			articlesByID[a.ID] = TopicArticleCard{ID: a.ID, Title: a.Title, Link: a.Link}
		}
	}

	result := make(map[uint][]TopicArticleCard)
	for _, s := range summaries {
		for _, aid := range linkMap[s.ID] {
			if card, ok := articlesByID[aid]; ok {
				result[s.ID] = append(result[s.ID], card)
			}
		}
	}
	return result
}
