package topicextraction

import (
	"context"
	"encoding/json"
	"fmt"
	"my-robot-backend/internal/domain/topictypes"
	"strings"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

// TagArticle extracts and stores tags for a single article
// This is called during auto_summary to tag individual articles
// Skips if the article already has tags (dedup)
func TagArticle(article *models.Article, feedName, categoryName string) error {
	if article == nil || article.ID == 0 {
		return nil
	}

	// Skip if already tagged
	var existingCount int64
	database.DB.Model(&models.ArticleTopicTag{}).Where("article_id = ?", article.ID).Count(&existingCount)
	if existingCount > 0 {
		return nil
	}

	// Build input for extraction
	input := topictypes.ExtractionInput{
		Title:        article.Title,
		Summary:      buildArticleSummary(*article),
		FeedName:     feedName,
		CategoryName: categoryName,
		ArticleID:    &article.ID,
	}

	// Use the extraction system
	extractor := NewTagExtractor()
	result, err := extractor.ExtractTags(context.Background(), input)

	var tags []topictypes.TopicTag
	var source string

	if err != nil || len(result.Tags) == 0 {
		// Fall back to legacy heuristic extraction
		tags = legacyExtractTopics(input)
		source = "heuristic"
	} else {
		tags = result.Tags
		source = result.Source
	}

	if len(tags) == 0 {
		return nil
	}

	// Process each tag
	for _, tag := range dedupeTagsWithCategory(tags) {
		dbTag, err := findOrCreateTag(tag, source)
		if err != nil {
			continue // Skip on error, don't fail the whole operation
		}

		// Create the association
		link := models.ArticleTopicTag{
			ArticleID:  article.ID,
			TopicTagID: dbTag.ID,
			Score:      tag.Score,
			Source:     source,
		}
		if err := database.DB.Create(&link).Error; err != nil {
			return err
		}
	}

	return nil
}

// buildArticleSummary builds a summary text from article fields
func buildArticleSummary(article models.Article) string {
	summary := article.Description
	if summary == "" {
		summary = article.Content
	}
	if summary == "" && article.FirecrawlContent != "" {
		summary = article.FirecrawlContent
	}
	return summary
}

// TagArticles batch tags multiple articles for a feed
// This is called from auto_summary when processing a feed's articles
func TagArticles(articles []models.Article, feedName, categoryName string) error {
	if len(articles) == 0 {
		return nil
	}

	for i := range articles {
		if err := TagArticle(&articles[i], feedName, categoryName); err != nil {
			// Log error but continue processing other articles
			fmt.Printf("[WARN] Failed to tag article %d: %v\n", articles[i].ID, err)
		}
	}

	return nil
}

// GetArticleTags retrieves all tags for a specific article
func GetArticleTags(articleID uint) ([]topictypes.TopicTag, error) {
	var links []models.ArticleTopicTag
	err := database.DB.Where("article_id = ?", articleID).
		Preload("topictypes.TopicTag").
		Find(&links).Error
	if err != nil {
		return nil, err
	}

	result := make([]topictypes.TopicTag, 0, len(links))
	for _, link := range links {
		if link.TopicTag == nil {
			continue
		}
		result = append(result, topictypes.TopicTag{
			Label:    link.TopicTag.Label,
			Slug:     link.TopicTag.Slug,
			Category: link.TopicTag.Category,
			Icon:     link.TopicTag.Icon,
			Aliases:  parseAliasesFromJSON(link.TopicTag.Aliases),
			Score:    link.Score,
		})
	}

	return result, nil
}

// GetArticlesByTag retrieves articles tagged with a specific tag
func GetArticlesByTag(slug, category string, limit int) ([]models.Article, error) {
	var articles []models.Article

	query := database.DB.
		Joins("JOIN article_topic_tags ON article_topic_tags.article_id = articles.id").
		Joins("JOIN topic_tags ON topic_tags.id = article_topic_tags.topic_tag_id").
		Where("topic_tags.slug = ?", slug)

	if category != "" {
		query = query.Where("topic_tags.category = ?", category)
	}

	err := query.
		Order("articles.pub_date DESC").
		Limit(limit).
		Find(&articles).Error

	return articles, err
}

func parseAliasesFromJSON(aliases string) []string {
	if strings.TrimSpace(aliases) == "" {
		return nil
	}
	var result []string
	if err := json.Unmarshal([]byte(aliases), &result); err != nil {
		return nil
	}
	return result
}
