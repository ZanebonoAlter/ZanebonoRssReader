package topicgraph

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topicextraction"
	"my-robot-backend/internal/domain/topictypes"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"

	"gorm.io/gorm"
)

func BuildTopicGraph(kind string, anchor time.Time, categoryID, feedID *uint) (*topictypes.TopicGraphResponse, error) {
	windowStart, windowEnd, periodLabel, err := topictypes.ResolveWindow(kind, anchor)
	if err != nil {
		return nil, err
	}

	articleTags, err := fetchArticleTagsData(windowStart, windowEnd, categoryID, feedID)
	if err != nil {
		return nil, err
	}

	nodes, edges, topTopics, articleCount := buildGraphPayloadFromArticles(database.DB, articleTags)
	feedCount := 0
	for _, node := range nodes {
		if node.Kind == "feed" {
			feedCount++
		}
	}

	return &topictypes.TopicGraphResponse{
		Type:         kind,
		AnchorDate:   windowStart.Format("2006-01-02"),
		PeriodLabel:  periodLabel,
		Nodes:        nodes,
		Edges:        edges,
		TopicCount:   len(topTopics),
		ArticleCount: articleCount,
		FeedCount:    feedCount,
		TopTopics:    topTopics,
	}, nil
}

func BuildTopicDetail(kind string, slug string, anchor time.Time, categoryID, feedID *uint) (*topictypes.TopicDetail, error) {
	windowStart, windowEnd, _, err := topictypes.ResolveWindow(kind, anchor)
	if err != nil {
		return nil, err
	}

	// 1. Get topic info
	var topic models.TopicTag
	err = database.DB.Where("slug = ?", slug).First(&topic).Error
	if err != nil {
		// If topic doesn't exist, create a placeholder
		topic = models.TopicTag{
			ID:       0,
			Slug:     slug,
			Label:    strings.ReplaceAll(strings.Title(strings.ReplaceAll(slug, "-", " ")), "Ai", "AI"),
			Category: models.TagCategoryKeyword,
		}
	}

	// 2. Get directly associated articles (new core logic)
	articles, total, err := getTopicArticles(topic.ID, windowStart, windowEnd, 1, 15, categoryID, feedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic articles: %w", err)
	}

	// 3. Get related tags for keyword cloud
	relatedTags, err := getRelatedTags(topic.ID, 20)
	if err != nil {
		// Log warning but don't fail
		logging.Warnf("Warning: failed to get related tags: %v", err)
	}

	// 4. Get AI summaries (optional, kept for backward compatibility)
	summaries, err := fetchSummaries(windowStart, windowEnd, categoryID, feedID)
	if err != nil {
		logging.Warnf("Warning: failed to fetch summaries: %v", err)
	}

	matchedSourceSummaries := make([]models.AISummary, 0)
	matchingSummaries := make([]topictypes.TopicSummaryCard, 0)
	relatedScores := map[string]topictypes.TopicTag{}
	var canonical topictypes.TopicTag

	for _, summary := range summaries {
		topics := summaryTopics(summary)
		if !containsTopic(topics, slug) {
			continue
		}

		canonical = pickTopic(topics, slug)
		matchedSourceSummaries = append(matchedSourceSummaries, summary)

		for _, topic := range topics {
			if topic.Slug == slug {
				continue
			}
			current := relatedScores[topic.Slug]
			topic.Score += current.Score
			relatedScores[topic.Slug] = topic
		}
	}

	if canonical.Slug == "" {
		canonical = topictypes.TopicTag{ID: topic.ID, Label: topic.Label, Slug: topic.Slug, Category: topictypes.NormalizeDisplayCategory(topic.Kind, topic.Category), Icon: topic.Icon, Kind: topictypes.NormalizeTopicKind(topic.Kind, topic.Category), Description: topic.Description, Score: 0}
	}

	articlesBySummary := topictypes.FetchArticlesForSummaries(matchedSourceSummaries)
	for _, summary := range matchedSourceSummaries {
		matchingSummaries = append(matchingSummaries, mapSummaryCard(summary, summaryTopics(summary), articlesBySummary[summary.ID]))
	}

	history, err := buildTopicHistory(kind, slug, anchor, categoryID, feedID)
	if err != nil {
		return nil, err
	}

	sort.SliceStable(matchingSummaries, func(i, j int) bool {
		return matchingSummaries[i].CreatedAt > matchingSummaries[j].CreatedAt
	})

	related := make([]topictypes.TopicTag, 0, len(relatedScores))
	for _, topic := range relatedScores {
		related = append(related, topic)
	}
	sort.SliceStable(related, func(i, j int) bool {
		if related[i].Score == related[j].Score {
			return related[i].Label < related[j].Label
		}
		return related[i].Score > related[j].Score
	})
	if len(related) > 8 {
		related = related[:8]
	}

	return &topictypes.TopicDetail{
		Topic:         canonical,
		Articles:      articles,
		TotalArticles: total,
		RelatedTags:   relatedTags,
		Summaries:     matchingSummaries,
		History:       history,
		RelatedTopics: related,
		SearchLinks: map[string]string{
			"youtube_videos": "https://www.youtube.com/results?search_query=" + url.QueryEscape(canonical.Label),
			"youtube_live":   "https://www.youtube.com/results?search_query=" + url.QueryEscape(canonical.Label+" live"),
		},
		AppLinks: map[string]string{
			"digest_view": "/digest/" + kind,
			"topic_graph": "/topics",
		},
	}, nil
}

// getTopicArticles retrieves articles directly associated with a topic tag
func getTopicArticles(topicID uint, startDate, endDate time.Time, page, pageSize int, categoryID, feedID *uint) ([]topictypes.TopicArticleCard, int64, error) {
	var articles []models.Article
	var total int64

	offset := (page - 1) * pageSize

	countQuery := database.DB.Model(&models.Article{}).
		Joins("JOIN article_topic_tags ON articles.id = article_topic_tags.article_id").
		Where("article_topic_tags.topic_tag_id = ?", topicID).
		Where("articles.created_at >= ? AND articles.created_at < ?", startDate, endDate)

	dataQuery := database.DB.Model(&models.Article{}).
		Joins("JOIN article_topic_tags ON articles.id = article_topic_tags.article_id").
		Where("article_topic_tags.topic_tag_id = ?", topicID).
		Where("articles.created_at >= ? AND articles.created_at < ?", startDate, endDate)

	if feedID != nil {
		countQuery = countQuery.Where("articles.feed_id = ?", *feedID)
		dataQuery = dataQuery.Where("articles.feed_id = ?", *feedID)
	} else if categoryID != nil {
		countQuery = countQuery.Joins("JOIN feeds ON feeds.id = articles.feed_id").Where("feeds.category_id = ?", *categoryID)
		dataQuery = dataQuery.Joins("JOIN feeds ON feeds.id = articles.feed_id").Where("feeds.category_id = ?", *categoryID)
	}

	err := countQuery.Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count articles: %w", err)
	}

	err = dataQuery.
		Order("articles.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Omit("tag_count", "relevance_score").
		Find(&articles).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query articles: %w", err)
	}

	// Query articles (tag_count is virtual, computed via subquery in other contexts)
	err = database.DB.Model(&models.Article{}).
		Joins("JOIN article_topic_tags ON articles.id = article_topic_tags.article_id").
		Where("article_topic_tags.topic_tag_id = ?", topicID).
		Where("articles.created_at >= ? AND articles.created_at < ?", startDate, endDate).
		Order("articles.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Omit("tag_count", "relevance_score").
		Find(&articles).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query articles: %w", err)
	}

	// Convert to cards
	cards := make([]topictypes.TopicArticleCard, 0, len(articles))
	for _, article := range articles {
		cards = append(cards, topictypes.TopicArticleCard{
			ID:    article.ID,
			Title: article.Title,
			Link:  article.Link,
		})
	}

	return cards, total, nil
}

// getRelatedTags retrieves tags that co-occur with the given topic
func getRelatedTags(topicID uint, limit int) ([]topictypes.RelatedTag, error) {
	var relatedTags []topictypes.RelatedTag

	// Query tags that co-occur with the current topic in articles
	// Ordered by co-occurrence count
	err := database.DB.Raw(`
		SELECT 
			t.id,
			t.label,
			t.slug,
			t.category,
			t.kind,
			COUNT(*) as cooccurrence
		FROM topic_tags t
		JOIN article_topic_tags at1 ON t.id = at1.topic_tag_id
		JOIN article_topic_tags at2 ON at1.article_id = at2.article_id
		WHERE at2.topic_tag_id = ?
		  AND t.id != ?
		GROUP BY t.id, t.label, t.slug, t.category
		ORDER BY cooccurrence DESC
		LIMIT ?
	`, topicID, topicID, limit).Scan(&relatedTags).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get related tags: %w", err)
	}

	for i := range relatedTags {
		relatedTags[i].Category = topictypes.NormalizeDisplayCategory(relatedTags[i].Kind, relatedTags[i].Category)
		relatedTags[i].Kind = topictypes.NormalizeTopicKind(relatedTags[i].Kind, relatedTags[i].Category)
	}

	return relatedTags, nil
}

// FetchTopicArticles is the public API for fetching topic articles with pagination
func FetchTopicArticles(slug string, kind string, anchor time.Time, page, pageSize int) ([]topictypes.TopicArticleCard, int64, error) {
	windowStart, windowEnd, _, err := topictypes.ResolveWindow(kind, anchor)
	if err != nil {
		return nil, 0, err
	}

	// Get topic
	var topic models.TopicTag
	err = database.DB.Where("slug = ?", slug).First(&topic).Error
	if err != nil {
		return nil, 0, fmt.Errorf("topic not found: %w", err)
	}

	return getTopicArticles(topic.ID, windowStart, windowEnd, page, pageSize, nil, nil)
}

func fetchSummaries(start time.Time, end time.Time, categoryID, feedID *uint) ([]models.AISummary, error) {
	var summaries []models.AISummary
	query := database.DB.Where("ai_summaries.created_at >= ? AND ai_summaries.created_at < ?", start, end).
		Preload("Feed").
		Preload("Category").
		Preload("SummaryTopics.TopicTag").
		Order("ai_summaries.created_at DESC")

	if feedID != nil {
		query = query.Where("ai_summaries.feed_id = ?", *feedID)
	} else if categoryID != nil {
		query = query.Joins("JOIN feeds ON feeds.id = ai_summaries.feed_id").
			Where("feeds.category_id = ?", *categoryID)
	}

	err := query.Find(&summaries).Error
	return summaries, err
}

func buildGraphPayload(db *gorm.DB, summaries []models.AISummary) ([]topictypes.GraphNode, []topictypes.GraphEdge, []topictypes.TopicTag) {
	topicNodes := map[string]*topictypes.GraphNode{}
	feedNodes := map[string]*topictypes.GraphNode{}
	edgeMap := map[string]*topictypes.GraphEdge{}
	topicScores := map[string]topictypes.TopicTag{}

	for _, summary := range summaries {
		topics := summaryTopics(summary)
		feedNodeID := feedNodeID(summary)
		if _, exists := feedNodes[feedNodeID]; !exists {
			feedNodes[feedNodeID] = &topictypes.GraphNode{
				ID:           feedNodeID,
				Label:        feedLabel(summary),
				Kind:         "feed",
				Weight:       1,
				Color:        feedColor(summary),
				FeedName:     feedLabel(summary),
				CategoryName: categoryLabel(summary),
			}
		}
		feedNodes[feedNodeID].Weight += 0.35

		for _, topic := range topics {
			if _, exists := topicNodes[topic.Slug]; !exists {
				topicNodes[topic.Slug] = &topictypes.GraphNode{
					ID:           topic.Slug,
					Label:        topic.Label,
					Slug:         topic.Slug,
					Kind:         "topic",
					Category:     topic.Category,
					Icon:         topic.Icon,
					Color:        GetCategoryColor(topic.Category),
					Weight:       topic.Score,
					ArticleCount: 0,
				}
			}
			topicNodes[topic.Slug].Weight += topic.Score
			topicNodes[topic.Slug].ArticleCount++

			merged := topicScores[topic.Slug]
			if merged.Label == "" || merged.Score < topic.Score {
				topicScores[topic.Slug] = topic
			}

			edgeKey := topic.Slug + "::" + feedNodeID
			if _, exists := edgeMap[edgeKey]; !exists {
				edgeMap[edgeKey] = &topictypes.GraphEdge{ID: edgeKey, Source: topic.Slug, Target: feedNodeID, Kind: "topic_feed", Weight: 0}
			}
			edgeMap[edgeKey].Weight += topic.Score
		}

		for i := 0; i < len(topics); i++ {
			for j := i + 1; j < len(topics); j++ {
				a := topics[i]
				b := topics[j]
				if a.Slug == b.Slug {
					continue
				}
				left, right := a.Slug, b.Slug
				if left > right {
					left, right = right, left
				}
				edgeKey := left + "::" + right
				if _, exists := edgeMap[edgeKey]; !exists {
					edgeMap[edgeKey] = &topictypes.GraphEdge{ID: edgeKey, Source: left, Target: right, Kind: "topic_topic", Weight: 0}
				}
				edgeMap[edgeKey].Weight += (a.Score + b.Score) / 2
			}
		}
	}

	nodes := make([]topictypes.GraphNode, 0, len(topicNodes)+len(feedNodes))
	for _, node := range topicNodes {
		nodes = append(nodes, *node)
	}
	for _, node := range feedNodes {
		nodes = append(nodes, *node)
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Weight == nodes[j].Weight {
			return nodes[i].Label < nodes[j].Label
		}
		return nodes[i].Weight > nodes[j].Weight
	})

	edges := make([]topictypes.GraphEdge, 0, len(edgeMap))
	for _, edge := range edgeMap {
		edges = append(edges, *edge)
	}
	sort.SliceStable(edges, func(i, j int) bool { return edges[i].Weight > edges[j].Weight })

	// Identify abstract tags (parent tags in topic_tag_relations)
	findAbstractSlugs(db, topicNodes)

	topTopics := make([]topictypes.TopicTag, 0, len(topicScores))
	for _, topic := range topicScores {
		topTopics = append(topTopics, topic)
	}
	sort.SliceStable(topTopics, func(i, j int) bool {
		if topTopics[i].Score == topTopics[j].Score {
			return topTopics[i].Label < topTopics[j].Label
		}
		return topTopics[i].Score > topTopics[j].Score
	})

	return nodes, edges, topTopics
}

func buildTopicHistory(kind string, slug string, anchor time.Time, categoryID, feedID *uint) ([]topictypes.TopicHistoryPoint, error) {
	history := make([]topictypes.TopicHistoryPoint, 0, 7)
	for i := 6; i >= 0; i-- {
		var pointAnchor time.Time
		if kind == "weekly" {
			pointAnchor = anchor.AddDate(0, 0, -7*i)
		} else {
			pointAnchor = anchor.AddDate(0, 0, -i)
		}

		start, end, label, err := topictypes.ResolveWindow(kind, pointAnchor)
		if err != nil {
			return nil, err
		}

		articleTags, err := fetchArticleTagsData(start, end, categoryID, feedID)
		if err != nil {
			return nil, err
		}

		count := 0
		articleSet := make(map[uint]bool)
		for _, at := range articleTags {
			if at.TopicTag != nil && at.TopicTag.Slug == slug {
				articleSet[at.ArticleID] = true
			}
		}
		count = len(articleSet)

		history = append(history, topictypes.TopicHistoryPoint{
			AnchorDate: start.Format("2006-01-02"),
			Count:      count,
			Label:      label,
		})
	}

	return history, nil
}

func summaryTopics(summary models.AISummary) []topictypes.TopicTag {
	if len(summary.SummaryTopics) > 0 {
		result := make([]topictypes.TopicTag, 0, len(summary.SummaryTopics))
		for _, link := range summary.SummaryTopics {
			if link.TopicTag == nil {
				continue
			}
			result = append(result, topictypes.TopicTag{
				ID:          link.TopicTag.ID,
				Label:       link.TopicTag.Label,
				Slug:        link.TopicTag.Slug,
				Category:    topictypes.NormalizeDisplayCategory(link.TopicTag.Kind, link.TopicTag.Category),
				Icon:        link.TopicTag.Icon,
				Aliases:     parseAliasesFromJSON(link.TopicTag.Aliases),
				Kind:        topictypes.NormalizeTopicKind(link.TopicTag.Kind, link.TopicTag.Category),
				Description: link.TopicTag.Description,
				Score:       link.Score,
			})
		}
		if len(result) > 0 {
			return topicextraction.DedupeTopics(result)
		}
	}

	return topicextraction.ExtractTopics(topictypes.ExtractionInput{
		Title:        summary.Title,
		Summary:      summary.Summary,
		FeedName:     feedLabel(summary),
		CategoryName: categoryLabel(summary),
	})
}

func mapSummaryCard(summary models.AISummary, topics []topictypes.TopicTag, articles []topictypes.TopicArticleCard) topictypes.TopicSummaryCard {
	aggregatedTags, err := topicextraction.AggregateArticleTags(parseTopicSummaryArticleIDs(summary.Articles))
	if err != nil {
		aggregatedTags = []topictypes.AggregatedTopicTag{}
	}

	return topictypes.TopicSummaryCard{
		ID:             summary.ID,
		Title:          summary.Title,
		Summary:        summary.Summary,
		FeedName:       feedLabel(summary),
		FeedIcon:       feedIcon(summary),
		FeedColor:      feedColor(summary),
		CategoryName:   categoryLabel(summary),
		ArticleCount:   summary.ArticleCount,
		CreatedAt:      summary.CreatedAt.In(topictypes.TopicGraphCST).Format(time.RFC3339),
		Topics:         topics,
		AggregatedTags: aggregatedTags,
		Articles:       articles,
	}
}

func parseTopicSummaryArticleIDs(raw string) []uint {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var articleIDs []uint
	if err := json.Unmarshal([]byte(raw), &articleIDs); err != nil {
		return nil
	}

	return articleIDs
}

func containsTopic(items []topictypes.TopicTag, slug string) bool {
	for _, item := range items {
		if item.Slug == slug {
			return true
		}
	}
	return false
}

func pickTopic(items []topictypes.TopicTag, slug string) topictypes.TopicTag {
	for _, item := range items {
		if item.Slug == slug {
			return item
		}
	}
	return topictypes.TopicTag{}
}

func feedNodeID(summary models.AISummary) string {
	if summary.FeedID != nil {
		return fmt.Sprintf("feed-%d", *summary.FeedID)
	}
	return "feed-unknown"
}

func feedLabel(summary models.AISummary) string {
	if summary.Feed != nil && strings.TrimSpace(summary.Feed.Title) != "" {
		return summary.Feed.Title
	}
	return "未知订阅源"
}

func feedColor(summary models.AISummary) string {
	if summary.Feed != nil && strings.TrimSpace(summary.Feed.Color) != "" {
		return summary.Feed.Color
	}
	return "#3b6b87"
}

func feedIcon(summary models.AISummary) string {
	if summary.Feed != nil && strings.TrimSpace(summary.Feed.Icon) != "" {
		return summary.Feed.Icon
	}
	return "mdi:rss"
}

func categoryLabel(summary models.AISummary) string {
	if summary.Category != nil && strings.TrimSpace(summary.Category.Name) != "" {
		return summary.Category.Name
	}
	return "未分类"
}

// GetCategoryColor returns the color for a given category
func GetCategoryColor(category string) string {
	switch category {
	case "event":
		return "#f59e0b" // amber
	case "person":
		return "#10b981" // emerald
	case "keyword":
		return "#6366f1" // indigo
	default:
		return "#6366f1" // default to indigo for unknown categories
	}
}

// parseAliasesFromJSON parses the Aliases JSON string into a string slice
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

// BuildTopicsByCategory builds topic lists grouped by category from article tags
// Only includes tags extracted by LLM (not heuristic feed/category names)
func BuildTopicsByCategory(kind string, anchor time.Time, categoryID, feedID *uint) (*topictypes.TopicsByCategoryResult, error) {
	windowStart, windowEnd, _, err := topictypes.ResolveWindow(kind, anchor)
	if err != nil {
		return nil, err
	}

	// Get articles from the time window with their LLM-extracted tags
	// Filter by source='llm' to exclude heuristic tags (feed names, category names)
	var articleTags []models.ArticleTopicTag
	query := database.DB.
		Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
		Joins("JOIN topic_tags ON topic_tags.id = article_topic_tags.topic_tag_id").
		Where("articles.created_at >= ? AND articles.created_at < ?", windowStart, windowEnd).
		Where("article_topic_tags.source = ?", "llm")

	if feedID != nil {
		query = query.Where("articles.feed_id = ?", *feedID)
	} else if categoryID != nil {
		query = query.Joins("JOIN feeds ON feeds.id = articles.feed_id").
			Where("feeds.category_id = ?", *categoryID)
	}

	err = query.Preload("TopicTag").
		Find(&articleTags).Error
	if err != nil {
		return nil, err
	}

	// Group tags by category and aggregate scores
	eventScores := make(map[string]*topictypes.TopicTag)
	personScores := make(map[string]*topictypes.TopicTag)
	keywordScores := make(map[string]*topictypes.TopicTag)

	for _, at := range articleTags {
		if at.TopicTag == nil {
			continue
		}

		tag := topictypes.TopicTag{
			ID:           at.TopicTag.ID,
			Label:        at.TopicTag.Label,
			Slug:         at.TopicTag.Slug,
			Category:     topictypes.NormalizeDisplayCategory(at.TopicTag.Kind, at.TopicTag.Category),
			Icon:         at.TopicTag.Icon,
			Kind:         topictypes.NormalizeTopicKind(at.TopicTag.Kind, at.TopicTag.Category),
			Description:  at.TopicTag.Description,
			Score:        at.Score,
			QualityScore: at.TopicTag.QualityScore,
			IsLowQuality: at.TopicTag.Source != "abstract" && at.TopicTag.QualityScore < 0.3,
		}

		switch tag.Category {
		case models.TagCategoryEvent:
			if existing, ok := eventScores[tag.Slug]; ok {
				existing.Score += tag.Score
			} else {
				eventScores[tag.Slug] = &tag
			}
		case models.TagCategoryPerson:
			if existing, ok := personScores[tag.Slug]; ok {
				existing.Score += tag.Score
			} else {
				personScores[tag.Slug] = &tag
			}
		default: // keyword
			if existing, ok := keywordScores[tag.Slug]; ok {
				existing.Score += tag.Score
			} else {
				keywordScores[tag.Slug] = &tag
			}
		}
	}

	enrichAbstractTags(database.DB, eventScores, personScores, keywordScores)
	finalizeTopicTagQuality(eventScores, personScores, keywordScores)

	result := &topictypes.TopicsByCategoryResult{
		Events:   sortTagsByScoreMap(eventScores),
		People:   sortTagsByScoreMap(personScores),
		Keywords: sortTagsByScoreMap(keywordScores),
	}

	return result, nil
}

// sortTagsByScoreMap converts a map of tags to a sorted slice
func sortTagsByScoreMap(tagMap map[string]*topictypes.TopicTag) []topictypes.TopicTag {
	result := make([]topictypes.TopicTag, 0, len(tagMap))
	for _, tag := range tagMap {
		result = append(result, *tag)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].QualityScore == result[j].QualityScore {
			if result[i].Score == result[j].Score {
				return result[i].Label < result[j].Label
			}
			return result[i].Score > result[j].Score
		}
		return result[i].QualityScore > result[j].QualityScore
	})

	return result
}

func finalizeTopicTagQuality(tagMaps ...map[string]*topictypes.TopicTag) {
	for _, tagMap := range tagMaps {
		for _, tag := range tagMap {
			tag.IsLowQuality = !tag.IsAbstract && tag.QualityScore < 0.3
		}
	}
}

// ArticleTagData represents aggregated data from article_topic_tags for graph building
type ArticleTagData struct {
	ArticleID uint
	FeedID    uint
	FeedTitle string
	FeedColor string
	TopicTag  *models.TopicTag
	Score     float64
}

// fetchArticleTagsData retrieves article-topic associations with feed info for graph building
func fetchArticleTagsData(start, end time.Time, categoryID, feedID *uint) ([]ArticleTagData, error) {
	var articleTags []models.ArticleTopicTag
	query := database.DB.
		Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
		Joins("JOIN topic_tags ON topic_tags.id = article_topic_tags.topic_tag_id").
		Where("articles.created_at >= ? AND articles.created_at < ?", start, end).
		Where("article_topic_tags.source = ?", "llm")

	if feedID != nil {
		query = query.Where("articles.feed_id = ?", *feedID)
	} else if categoryID != nil {
		query = query.Joins("JOIN feeds ON feeds.id = articles.feed_id").
			Where("feeds.category_id = ?", *categoryID)
	}

	err := query.
		Preload("TopicTag").
		Preload("Article.Feed").
		Find(&articleTags).Error
	if err != nil {
		return nil, err
	}

	data := make([]ArticleTagData, 0, len(articleTags))
	for _, at := range articleTags {
		if at.TopicTag == nil || at.Article == nil {
			continue
		}
		feedTitle := "未知订阅源"
		feedColor := "#3b6b87"
		if at.Article.Feed.ID != 0 {
			if strings.TrimSpace(at.Article.Feed.Title) != "" {
				feedTitle = at.Article.Feed.Title
			}
			if strings.TrimSpace(at.Article.Feed.Color) != "" {
				feedColor = at.Article.Feed.Color
			}
		}
		data = append(data, ArticleTagData{
			ArticleID: at.ArticleID,
			FeedID:    at.Article.FeedID,
			FeedTitle: feedTitle,
			FeedColor: feedColor,
			TopicTag:  at.TopicTag,
			Score:     at.Score,
		})
	}

	return data, nil
}

// buildGraphPayloadFromArticles builds graph nodes and edges from article tag data
func buildGraphPayloadFromArticles(db *gorm.DB, data []ArticleTagData) ([]topictypes.GraphNode, []topictypes.GraphEdge, []topictypes.TopicTag, int) {
	topicNodes := map[string]*topictypes.GraphNode{}
	feedNodes := map[string]*topictypes.GraphNode{}
	edgeMap := map[string]*topictypes.GraphEdge{}
	topicScores := map[string]topictypes.TopicTag{}
	articleSet := make(map[uint]bool)

	for _, item := range data {
		articleSet[item.ArticleID] = true

		feedNodeID := fmt.Sprintf("feed-%d", item.FeedID)
		if _, exists := feedNodes[feedNodeID]; !exists {
			feedNodes[feedNodeID] = &topictypes.GraphNode{
				ID:       feedNodeID,
				Label:    item.FeedTitle,
				Kind:     "feed",
				Weight:   1,
				Color:    item.FeedColor,
				FeedName: item.FeedTitle,
			}
		}
		feedNodes[feedNodeID].Weight += 0.35

		topicSlug := item.TopicTag.Slug
		topicLabel := item.TopicTag.Label
		topicCategory := topictypes.NormalizeDisplayCategory(item.TopicTag.Kind, item.TopicTag.Category)

		if _, exists := topicNodes[topicSlug]; !exists {
			topicNodes[topicSlug] = &topictypes.GraphNode{
				ID:           topicSlug,
				Label:        topicLabel,
				Slug:         topicSlug,
				Kind:         "topic",
				Category:     topicCategory,
				Icon:         item.TopicTag.Icon,
				Color:        GetCategoryColor(topicCategory),
				Weight:       0,
				ArticleCount: 0,
			}
		}
		topicNodes[topicSlug].Weight += item.Score
		topicNodes[topicSlug].ArticleCount++

		merged := topicScores[topicSlug]
		if merged.Label == "" || merged.Score < item.Score {
			topicScores[topicSlug] = topictypes.TopicTag{
				ID:           item.TopicTag.ID,
				Label:        topicLabel,
				Slug:         topicSlug,
				Category:     topicCategory,
				Icon:         item.TopicTag.Icon,
				Kind:         topictypes.NormalizeTopicKind(item.TopicTag.Kind, item.TopicTag.Category),
				Score:        item.Score,
				QualityScore: item.TopicTag.QualityScore,
				IsLowQuality: item.TopicTag.Source != "abstract" && item.TopicTag.QualityScore < 0.3,
			}
		}

		edgeKey := topicSlug + "::" + feedNodeID
		if _, exists := edgeMap[edgeKey]; !exists {
			edgeMap[edgeKey] = &topictypes.GraphEdge{ID: edgeKey, Source: topicSlug, Target: feedNodeID, Kind: "topic_feed", Weight: 0}
		}
		edgeMap[edgeKey].Weight += item.Score
	}

	// Build topic-topic edges from co-occurrence in same article
	articleTopics := make(map[uint][]string)
	for _, item := range data {
		articleTopics[item.ArticleID] = append(articleTopics[item.ArticleID], item.TopicTag.Slug)
	}
	for _, slugs := range articleTopics {
		for i := 0; i < len(slugs); i++ {
			for j := i + 1; j < len(slugs); j++ {
				if slugs[i] == slugs[j] {
					continue
				}
				left, right := slugs[i], slugs[j]
				if left > right {
					left, right = right, left
				}
				edgeKey := left + "::" + right
				if _, exists := edgeMap[edgeKey]; !exists {
					edgeMap[edgeKey] = &topictypes.GraphEdge{ID: edgeKey, Source: left, Target: right, Kind: "topic_topic", Weight: 0}
				}
				edgeMap[edgeKey].Weight += 0.5
			}
		}
	}

	// Identify abstract tags (parent tags in topic_tag_relations)
	findAbstractSlugs(db, topicNodes)

	nodes := make([]topictypes.GraphNode, 0, len(topicNodes)+len(feedNodes))
	for _, node := range topicNodes {
		nodes = append(nodes, *node)
	}
	for _, node := range feedNodes {
		nodes = append(nodes, *node)
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Weight == nodes[j].Weight {
			return nodes[i].Label < nodes[j].Label
		}
		return nodes[i].Weight > nodes[j].Weight
	})

	edges := make([]topictypes.GraphEdge, 0, len(edgeMap))
	for _, edge := range edgeMap {
		edges = append(edges, *edge)
	}
	sort.SliceStable(edges, func(i, j int) bool { return edges[i].Weight > edges[j].Weight })

	topTopics := make([]topictypes.TopicTag, 0, len(topicScores))
	for _, topic := range topicScores {
		topic.Score = topicNodes[topic.Slug].Weight
		topTopics = append(topTopics, topic)
	}
	markTopicTagsQuality(topTopics)
	sort.SliceStable(topTopics, func(i, j int) bool {
		if topTopics[i].QualityScore == topTopics[j].QualityScore {
			if topTopics[i].Score == topTopics[j].Score {
				return topTopics[i].Label < topTopics[j].Label
			}
			return topTopics[i].Score > topTopics[j].Score
		}
		return topTopics[i].QualityScore > topTopics[j].QualityScore
	})

	return nodes, edges, topTopics, len(articleSet)
}

func markTopicTagsQuality(tags []topictypes.TopicTag) {
	for i := range tags {
		tags[i].IsLowQuality = !tags[i].IsAbstract && tags[i].QualityScore < 0.3
	}
}

// GetPendingArticlesByTag retrieves articles that have the given tag but are not in any digest
func GetPendingArticlesByTag(tagSlug string, kind string, anchor time.Time) (*topictypes.PendingArticlesResponse, error) {
	windowStart, windowEnd, _, err := topictypes.ResolveWindow(kind, anchor)
	if err != nil {
		return nil, err
	}

	// Step 1: Get the topic tag
	var topicTag models.TopicTag
	err = database.DB.Where("slug = ?", tagSlug).First(&topicTag).Error
	if err != nil {
		return nil, fmt.Errorf("topic tag not found: %w", err)
	}

	// Step 2: Get articles with this tag (and all child tags if abstract) in the time window
	tagIDSet := collectAllChildTagIDs(topicTag.ID)
	tagIDs := make([]uint, 0, len(tagIDSet))
	for id := range tagIDSet {
		tagIDs = append(tagIDs, id)
	}

	var taggedArticles []models.Article
	err = database.DB.
		Joins("JOIN article_topic_tags ON articles.id = article_topic_tags.article_id").
		Where("article_topic_tags.topic_tag_id IN ?", tagIDs).
		Where("articles.created_at >= ? AND articles.created_at < ?", windowStart, windowEnd).
		Preload("Feed").
		Omit("tag_count", "relevance_score").
		Distinct().
		Find(&taggedArticles).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get tagged articles: %w", err)
	}

	if len(taggedArticles) == 0 {
		return &topictypes.PendingArticlesResponse{Articles: []topictypes.PendingArticle{}, Total: 0}, nil
	}

	// Step 3: Get all article IDs that are already in digests
	var summaries []models.AISummary
	err = database.DB.
		Where("created_at >= ? AND created_at < ?", windowStart, windowEnd).
		Where("articles IS NOT NULL AND articles != ''").
		Find(&summaries).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get summaries: %w", err)
	}

	// Build set of article IDs that are in any digest
	digestArticleIDs := make(map[uint]bool)
	for _, summary := range summaries {
		ids := parseTopicSummaryArticleIDs(summary.Articles)
		for _, id := range ids {
			digestArticleIDs[id] = true
		}
	}

	// Step 4: Filter articles that are not in any digest
	var pendingArticles []topictypes.PendingArticle
	for _, article := range taggedArticles {
		if digestArticleIDs[article.ID] {
			continue
		}

		pa := topictypes.PendingArticle{
			ID:    article.ID,
			Title: article.Title,
			Link:  article.Link,
		}

		if article.PubDate != nil {
			pa.PubDate = article.PubDate.In(topictypes.TopicGraphCST).Format(time.RFC3339)
		}

		if article.Feed.ID != 0 {
			pa.FeedName = article.Feed.Title
			pa.FeedIcon = article.Feed.Icon
			pa.FeedColor = article.Feed.Color
		} else {
			pa.FeedName = "未知订阅源"
		}

		pendingArticles = append(pendingArticles, pa)
	}

	return &topictypes.PendingArticlesResponse{
		Articles: pendingArticles,
		Total:    len(pendingArticles),
	}, nil
}

// findAbstractSlugs queries topic_tag_relations to identify which tag slugs are abstract parents.
// It annotates matching nodes in the topicNodes map with IsAbstract=true.
func findAbstractSlugs(db *gorm.DB, topicNodes map[string]*topictypes.GraphNode) {
	var abstractParentIDs []uint
	db.Model(&models.TopicTagRelation{}).
		Select("DISTINCT parent_id").
		Pluck("parent_id", &abstractParentIDs)

	if len(abstractParentIDs) == 0 {
		return
	}

	var parentTags []models.TopicTag
	db.Where("id IN ?", abstractParentIDs).Find(&parentTags)

	abstractSlugs := make(map[string]bool, len(parentTags))
	for _, t := range parentTags {
		abstractSlugs[t.Slug] = true
	}

	for slug, node := range topicNodes {
		if abstractSlugs[slug] {
			node.IsAbstract = true
		}
	}
}

// enrichAbstractTags queries topic_tag_relations and enriches tags in the category maps
// with IsAbstract flag and ChildSlugs for parent tags.
func enrichAbstractTags(db *gorm.DB, tagMaps ...map[string]*topictypes.TopicTag) {
	var relations []models.TopicTagRelation
	db.Preload("Parent").Preload("Child").Find(&relations)
	if len(relations) == 0 {
		return
	}

	parentToChildren := make(map[uint][]string)
	parentByID := make(map[uint]*models.TopicTag)
	for _, rel := range relations {
		if rel.Parent != nil && rel.Child != nil {
			parentToChildren[rel.ParentID] = append(parentToChildren[rel.ParentID], rel.Child.Slug)
			parentByID[rel.ParentID] = rel.Parent
		}
	}

	childToParents := make(map[uint]uint)
	for _, rel := range relations {
		childToParents[rel.ChildID] = rel.ParentID
	}

	abstractSlugs := make(map[string]bool, len(parentByID))
	for _, parent := range parentByID {
		abstractSlugs[parent.Slug] = true
	}

	for _, m := range tagMaps {
		for slug, tag := range m {
			if abstractSlugs[slug] {
				tag.IsAbstract = true
			}
		}
	}

	allSlugs := make(map[string]*topictypes.TopicTag)
	for _, m := range tagMaps {
		for slug, tag := range m {
			allSlugs[slug] = tag
		}
	}

	for parentID, childSlugs := range parentToChildren {
		parent, ok := parentByID[parentID]
		if !ok {
			continue
		}
		tag, exists := allSlugs[parent.Slug]
		if !exists {
			continue
		}
		tag.IsAbstract = true
		tag.ChildSlugs = childSlugs
	}
}
