package topicgraph

import (
	"encoding/json"
	"fmt"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

// HotspotDigestCard represents a digest summary for hotspot display
// Returned when tracing from article tag back to containing digests
type HotspotDigestCard struct {
	ID              uint                `json:"id"`
	Title           string              `json:"title"`
	Summary         string              `json:"summary"`
	FeedName        string              `json:"feed_name"`
	FeedColor       string              `json:"feed_color"`
	CategoryName    string              `json:"category_name"`
	ArticleCount    int                 `json:"article_count"`
	CreatedAt       time.Time           `json:"created_at"`
	MatchedArticles []HotspotArticleRef `json:"matched_articles,omitempty"`
}

// HotspotArticleRef represents a matched article reference
type HotspotArticleRef struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

// GetDigestsByArticleTag retrieves digests that contain articles with the given tag
// This enables the reverse trace: Tag -> Articles -> Digests (containing those articles)
func GetDigestsByArticleTag(tagSlug string, kind string, anchor time.Time, limit int) ([]HotspotDigestCard, error) {
	windowStart, windowEnd, _, err := resolveWindow(kind, anchor)
	if err != nil {
		return nil, err
	}

	// Step 1: Get the topic tag
	var topicTag models.TopicTag
	err = database.DB.Where("slug = ?", tagSlug).First(&topicTag).Error
	if err != nil {
		return nil, fmt.Errorf("topic tag not found: %w", err)
	}

	// Step 2: Get articles with this tag in the time window
	var articleIDs []uint
	err = database.DB.Model(&models.ArticleTopicTag{}).
		Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
		Where("article_topic_tags.topic_tag_id = ?", topicTag.ID).
		Where("articles.created_at >= ? AND articles.created_at < ?", windowStart, windowEnd).
		Pluck("articles.id", &articleIDs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get articles: %w", err)
	}

	if len(articleIDs) == 0 {
		return []HotspotDigestCard{}, nil
	}

	// Step 3: Get summaries that contain any of these articles
	// The articles field in ai_summaries is a JSON array of article IDs
	var summaries []models.AISummary
	err = database.DB.
		Where("created_at >= ? AND created_at < ?", windowStart, windowEnd).
		Preload("Feed").
		Preload("Category").
		Order("created_at DESC").
		Find(&summaries).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get summaries: %w", err)
	}

	// Step 4: Filter summaries that contain our articles and build result
	var result []HotspotDigestCard
	for _, summary := range summaries {
		matchedArticles := getMatchedArticlesFromSummary(summary, articleIDs)
		if len(matchedArticles) == 0 {
			continue
		}

		card := HotspotDigestCard{
			ID:              summary.ID,
			Title:           summary.Title,
			Summary:         summary.Summary,
			ArticleCount:    summary.ArticleCount,
			CreatedAt:       summary.CreatedAt,
			MatchedArticles: matchedArticles,
		}

		if summary.Feed != nil {
			card.FeedName = summary.Feed.Title
			card.FeedColor = summary.Feed.Color
		}

		if summary.Category != nil {
			card.CategoryName = summary.Category.Name
		}

		result = append(result, card)
		if limit > 0 && len(result) >= limit {
			break
		}
	}

	return result, nil
}

// getMatchedArticlesFromSummary extracts matched articles from a summary's articles JSON field
func getMatchedArticlesFromSummary(summary models.AISummary, targetArticleIDs []uint) []HotspotArticleRef {
	if summary.Articles == "" {
		return nil
	}

	var articleIDs []uint
	if err := json.Unmarshal([]byte(summary.Articles), &articleIDs); err != nil {
		return nil
	}

	// Build lookup set for target articles
	targetSet := make(map[uint]bool)
	for _, id := range targetArticleIDs {
		targetSet[id] = true
	}

	// Find matches
	var matched []HotspotArticleRef
	for _, id := range articleIDs {
		if targetSet[id] {
			matched = append(matched, HotspotArticleRef{ID: id})
		}
	}

	return matched
}
