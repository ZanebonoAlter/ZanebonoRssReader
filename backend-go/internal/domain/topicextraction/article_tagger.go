package topicextraction

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topictypes"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"
)

func FeedCategoryName(feed models.Feed) string {
	if feed.Category != nil && strings.TrimSpace(feed.Category.Name) != "" {
		return feed.Category.Name
	}
	if feed.CategoryID != nil {
		var cat models.Category
		if err := database.DB.First(&cat, *feed.CategoryID).Error; err == nil && cat.Name != "" {
			return cat.Name
		}
	}
	return ""
}

type tagArticleOptions struct {
	Force bool
}

// TagArticle extracts and stores tags for a single article.
func TagArticle(article *models.Article, feedName, categoryName string) error {
	return tagArticle(article, feedName, categoryName, tagArticleOptions{})
}

// RetagArticle replaces existing tags using the latest article content.
func RetagArticle(article *models.Article, feedName, categoryName string) error {
	return tagArticle(article, feedName, categoryName, tagArticleOptions{Force: true})
}

func tagArticle(article *models.Article, feedName, categoryName string, options tagArticleOptions) error {
	if article == nil || article.ID == 0 {
		return nil
	}

	if options.Force {
		var oldTagIDs []uint
		if err := database.DB.Model(&models.ArticleTopicTag{}).
			Where("article_id = ?", article.ID).
			Pluck("topic_tag_id", &oldTagIDs).Error; err != nil {
			return err
		}

		if err := database.DB.Where("article_id = ?", article.ID).Delete(&models.ArticleTopicTag{}).Error; err != nil {
			return err
		}

		cleanupOrphanedTags(oldTagIDs)
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

	// Build article context for description generation
	articleContext := ""
	if article.Title != "" {
		articleContext = article.Title
	}
	articleSummary := buildArticleSummary(*article)
	if articleSummary != "" {
		if articleContext != "" {
			articleContext += ". "
		}
		runes := []rune(articleSummary)
		if len(runes) > 800 {
			articleSummary = string(runes[:800])
		}
		articleContext += articleSummary
	}

	// Process each tag
	seenTagIDs := make(map[uint]struct{})
	for _, tag := range dedupeTagsWithCategory(tags) {
		dbTag, err := findOrCreateTag(context.Background(), tag, source, articleContext)
		if err != nil {
			logging.Warnf("findOrCreateTag failed for tag %q (category=%s, slug=%s, source=%s, article=%d): %v", tag.Label, tag.Category, topictypes.Slugify(tag.Label), source, article.ID, err)
			continue
		}

		// Skip if we already added this tag ID (prevents duplicate key violations)
		if _, alreadyAdded := seenTagIDs[dbTag.ID]; alreadyAdded {
			continue
		}
		seenTagIDs[dbTag.ID] = struct{}{}

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

const maxSummaryRunesForTagging = 2000

func buildArticleSummary(article models.Article) string {
	var body string
	if s := strings.TrimSpace(article.AIContentSummary); s != "" {
		body = s
	} else if s := strings.TrimSpace(article.FirecrawlContent); s != "" {
		body = s
	} else if s := strings.TrimSpace(article.Content); s != "" {
		body = s
	} else if s := strings.TrimSpace(article.Description); s != "" {
		body = s
	}
	if body == "" {
		return ""
	}
	runes := []rune(body)
	if len(runes) > maxSummaryRunesForTagging {
		body = string(runes[:maxSummaryRunesForTagging])
	}
	return body
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
			logging.Warnf("Failed to tag article %d: %v", articles[i].ID, err)
		}
	}

	return nil
}

// BackfillArticleTags only tags articles that currently have no article tags.
// This is a fallback path for summary-time repair, not the main tagging flow.
func BackfillArticleTags(articles []models.Article, feedName, categoryName string) error {
	if len(articles) == 0 {
		return nil
	}

	for i := range articles {
		var existingCount int64
		if err := database.DB.Model(&models.ArticleTopicTag{}).Where("article_id = ?", articles[i].ID).Count(&existingCount).Error; err != nil {
			logging.Warnf("Failed to inspect article tags for %d: %v", articles[i].ID, err)
			continue
		}
		if existingCount > 0 {
			continue
		}

		if err := TagArticle(&articles[i], feedName, categoryName); err != nil {
			logging.Warnf("Failed to backfill article %d tags: %v", articles[i].ID, err)
		}
	}

	return nil
}

// GetArticleTags retrieves all tags for a specific article
func GetArticleTags(articleID uint) ([]topictypes.TopicTag, error) {
	var links []models.ArticleTopicTag
	err := database.DB.Where("article_id = ?", articleID).
		Preload("TopicTag").
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

func AggregateArticleTags(articleIDs []uint) ([]topictypes.AggregatedTopicTag, error) {
	if len(articleIDs) == 0 {
		return []topictypes.AggregatedTopicTag{}, nil
	}

	uniqueIDs := make([]uint, 0, len(articleIDs))
	seenArticleIDs := make(map[uint]struct{}, len(articleIDs))
	for _, articleID := range articleIDs {
		if articleID == 0 {
			continue
		}
		if _, exists := seenArticleIDs[articleID]; exists {
			continue
		}
		seenArticleIDs[articleID] = struct{}{}
		uniqueIDs = append(uniqueIDs, articleID)
	}

	if len(uniqueIDs) == 0 {
		return []topictypes.AggregatedTopicTag{}, nil
	}

	var links []models.ArticleTopicTag
	err := database.DB.Where("article_id IN ?", uniqueIDs).
		Preload("TopicTag").
		Find(&links).Error
	if err != nil {
		return nil, err
	}

	aggregatedBySlug := make(map[string]*topictypes.AggregatedTopicTag)
	articleSeenBySlug := make(map[string]map[uint]struct{})

	for _, link := range links {
		if link.TopicTag == nil {
			continue
		}

		slug := link.TopicTag.Slug
		if slug == "" {
			continue
		}

		item, exists := aggregatedBySlug[slug]
		if !exists {
			item = &topictypes.AggregatedTopicTag{
				Slug:     slug,
				Label:    link.TopicTag.Label,
				Category: topictypes.NormalizeDisplayCategory(link.TopicTag.Kind, link.TopicTag.Category),
				Kind:     topictypes.NormalizeTopicKind(link.TopicTag.Kind, link.TopicTag.Category),
				Icon:     link.TopicTag.Icon,
				Aliases:  parseAliasesFromJSON(link.TopicTag.Aliases),
				Score:    0,
			}
			aggregatedBySlug[slug] = item
		}

		item.Score += link.Score

		if articleSeenBySlug[slug] == nil {
			articleSeenBySlug[slug] = make(map[uint]struct{})
		}
		if _, exists := articleSeenBySlug[slug][link.ArticleID]; !exists {
			articleSeenBySlug[slug][link.ArticleID] = struct{}{}
			item.ArticleCount++
		}
	}

	result := make([]topictypes.AggregatedTopicTag, 0, len(aggregatedBySlug))
	for _, item := range aggregatedBySlug {
		result = append(result, *item)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ArticleCount == result[j].ArticleCount {
			if result[i].Score == result[j].Score {
				return result[i].Label < result[j].Label
			}
			return result[i].Score > result[j].Score
		}
		return result[i].ArticleCount > result[j].ArticleCount
	})

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
		Omit("tag_count", "relevance_score").
		Order("articles.pub_date DESC").
		Limit(limit).
		Find(&articles).Error

	return articles, err
}

func cleanupOrphanedTags(tagIDs []uint) {
	if len(tagIDs) == 0 {
		return
	}

	var orphanIDs []uint
	database.DB.Model(&models.TopicTag{}).
		Where("id IN ?", tagIDs).
		Where("id NOT IN (SELECT topic_tag_id FROM article_topic_tags)").
		Pluck("id", &orphanIDs)

	if len(orphanIDs) == 0 {
		return
	}

	if err := database.DB.Where("topic_tag_id IN ?", orphanIDs).Delete(&models.TopicTagEmbedding{}).Error; err != nil {
		logging.Warnf("Failed to delete embeddings for orphaned topic tags: %v", err)
	}
	if err := database.DB.Where("id IN ?", orphanIDs).Delete(&models.TopicTag{}).Error; err != nil {
		logging.Warnf("Failed to delete %d orphaned topic tags: %v", len(orphanIDs), err)
	} else {
		logging.Infof("Cleaned up %d orphaned topic tags", len(orphanIDs))
	}
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
