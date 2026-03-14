package topicgraph

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

var topicGraphCST = time.FixedZone("CST", 8*3600)

func BuildTopicGraph(kind string, anchor time.Time) (*TopicGraphResponse, error) {
	windowStart, windowEnd, periodLabel, err := resolveWindow(kind, anchor)
	if err != nil {
		return nil, err
	}

	summaries, err := fetchSummaries(windowStart, windowEnd)
	if err != nil {
		return nil, err
	}

	nodes, edges, topTopics := buildGraphPayload(summaries)
	feedCount := 0
	for _, node := range nodes {
		if node.Kind == "feed" {
			feedCount++
		}
	}

	return &TopicGraphResponse{
		Type:         kind,
		AnchorDate:   windowStart.Format("2006-01-02"),
		PeriodLabel:  periodLabel,
		Nodes:        nodes,
		Edges:        edges,
		TopicCount:   len(topTopics),
		SummaryCount: len(summaries),
		FeedCount:    feedCount,
		TopTopics:    topTopics,
	}, nil
}

func BuildTopicDetail(kind string, slug string, anchor time.Time) (*TopicDetail, error) {
	windowStart, windowEnd, _, err := resolveWindow(kind, anchor)
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
	articles, total, err := getTopicArticles(topic.ID, windowStart, windowEnd, 1, 15)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic articles: %w", err)
	}

	// 3. Get related tags for keyword cloud
	relatedTags, err := getRelatedTags(topic.ID, 20)
	if err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to get related tags: %v\n", err)
	}

	// 4. Get AI summaries (optional, kept for backward compatibility)
	summaries, err := fetchSummaries(windowStart, windowEnd)
	if err != nil {
		fmt.Printf("Warning: failed to fetch summaries: %v\n", err)
	}

	matchedSourceSummaries := make([]models.AISummary, 0)
	matchingSummaries := make([]TopicSummaryCard, 0)
	relatedScores := map[string]TopicTag{}
	var canonical TopicTag

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
		canonical = TopicTag{ID: topic.ID, Label: topic.Label, Slug: topic.Slug, Category: normalizeDisplayCategory(topic.Kind, topic.Category), Icon: topic.Icon, Kind: normalizeTopicKind(topic.Kind, topic.Category), Score: 0}
	}

	articlesBySummary := fetchArticlesForSummaries(matchedSourceSummaries)
	for _, summary := range matchedSourceSummaries {
		matchingSummaries = append(matchingSummaries, mapSummaryCard(summary, summaryTopics(summary), articlesBySummary[summary.ID]))
	}

	history, err := buildTopicHistory(kind, slug, anchor)
	if err != nil {
		return nil, err
	}

	sort.SliceStable(matchingSummaries, func(i, j int) bool {
		return matchingSummaries[i].CreatedAt > matchingSummaries[j].CreatedAt
	})

	related := make([]TopicTag, 0, len(relatedScores))
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

	return &TopicDetail{
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
func getTopicArticles(topicID uint, startDate, endDate time.Time, page, pageSize int) ([]TopicArticleCard, int64, error) {
	var articles []models.Article
	var total int64

	offset := (page - 1) * pageSize

	// Count total
	err := database.DB.Model(&models.Article{}).
		Joins("JOIN article_topic_tags ON articles.id = article_topic_tags.article_id").
		Where("article_topic_tags.topic_tag_id = ?", topicID).
		Where("articles.created_at >= ? AND articles.created_at < ?", startDate, endDate).
		Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count articles: %w", err)
	}

	// Query articles
	err = database.DB.Model(&models.Article{}).
		Joins("JOIN article_topic_tags ON articles.id = article_topic_tags.article_id").
		Where("article_topic_tags.topic_tag_id = ?", topicID).
		Where("articles.created_at >= ? AND articles.created_at < ?", startDate, endDate).
		Order("articles.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&articles).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query articles: %w", err)
	}

	// Convert to cards
	cards := make([]TopicArticleCard, 0, len(articles))
	for _, article := range articles {
		cards = append(cards, TopicArticleCard{
			ID:    article.ID,
			Title: article.Title,
			Link:  article.Link,
		})
	}

	return cards, total, nil
}

// getRelatedTags retrieves tags that co-occur with the given topic
func getRelatedTags(topicID uint, limit int) ([]RelatedTag, error) {
	var relatedTags []RelatedTag

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
		relatedTags[i].Category = normalizeDisplayCategory(relatedTags[i].Kind, relatedTags[i].Category)
		relatedTags[i].Kind = normalizeTopicKind(relatedTags[i].Kind, relatedTags[i].Category)
	}

	return relatedTags, nil
}

// FetchTopicArticles is the public API for fetching topic articles with pagination
func FetchTopicArticles(slug string, kind string, anchor time.Time, page, pageSize int) ([]TopicArticleCard, int64, error) {
	windowStart, windowEnd, _, err := resolveWindow(kind, anchor)
	if err != nil {
		return nil, 0, err
	}

	// Get topic
	var topic models.TopicTag
	err = database.DB.Where("slug = ?", slug).First(&topic).Error
	if err != nil {
		return nil, 0, fmt.Errorf("topic not found: %w", err)
	}

	return getTopicArticles(topic.ID, windowStart, windowEnd, page, pageSize)
}

func resolveWindow(kind string, anchor time.Time) (time.Time, time.Time, string, error) {
	current := anchor.In(topicGraphCST)
	dayStart := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, topicGraphCST)

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

func fetchSummaries(start time.Time, end time.Time) ([]models.AISummary, error) {
	var summaries []models.AISummary
	err := database.DB.Where("created_at >= ? AND created_at < ?", start, end).
		Preload("Feed").
		Preload("Category").
		Preload("SummaryTopics.TopicTag").
		Order("created_at DESC").
		Find(&summaries).Error
	return summaries, err
}

func buildGraphPayload(summaries []models.AISummary) ([]GraphNode, []GraphEdge, []TopicTag) {
	topicNodes := map[string]*GraphNode{}
	feedNodes := map[string]*GraphNode{}
	edgeMap := map[string]*GraphEdge{}
	topicScores := map[string]TopicTag{}

	for _, summary := range summaries {
		topics := summaryTopics(summary)
		feedNodeID := feedNodeID(summary)
		if _, exists := feedNodes[feedNodeID]; !exists {
			feedNodes[feedNodeID] = &GraphNode{
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
				topicNodes[topic.Slug] = &GraphNode{
					ID:           topic.Slug,
					Label:        topic.Label,
					Slug:         topic.Slug,
					Kind:         "topic",
					Category:     topic.Category,
					Icon:         topic.Icon,
					Color:        GetCategoryColor(topic.Category),
					Weight:       topic.Score,
					SummaryCount: 0,
				}
			}
			topicNodes[topic.Slug].Weight += topic.Score
			topicNodes[topic.Slug].SummaryCount++

			merged := topicScores[topic.Slug]
			if merged.Label == "" || merged.Score < topic.Score {
				topicScores[topic.Slug] = topic
			}

			edgeKey := topic.Slug + "::" + feedNodeID
			if _, exists := edgeMap[edgeKey]; !exists {
				edgeMap[edgeKey] = &GraphEdge{ID: edgeKey, Source: topic.Slug, Target: feedNodeID, Kind: "topic_feed", Weight: 0}
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
					edgeMap[edgeKey] = &GraphEdge{ID: edgeKey, Source: left, Target: right, Kind: "topic_topic", Weight: 0}
				}
				edgeMap[edgeKey].Weight += (a.Score + b.Score) / 2
			}
		}
	}

	nodes := make([]GraphNode, 0, len(topicNodes)+len(feedNodes))
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

	edges := make([]GraphEdge, 0, len(edgeMap))
	for _, edge := range edgeMap {
		edges = append(edges, *edge)
	}
	sort.SliceStable(edges, func(i, j int) bool { return edges[i].Weight > edges[j].Weight })

	topTopics := make([]TopicTag, 0, len(topicScores))
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

func buildTopicHistory(kind string, slug string, anchor time.Time) ([]TopicHistoryPoint, error) {
	history := make([]TopicHistoryPoint, 0, 7)
	for i := 6; i >= 0; i-- {
		var pointAnchor time.Time
		if kind == "weekly" {
			pointAnchor = anchor.AddDate(0, 0, -7*i)
		} else {
			pointAnchor = anchor.AddDate(0, 0, -i)
		}

		start, end, label, err := resolveWindow(kind, pointAnchor)
		if err != nil {
			return nil, err
		}
		summaries, err := fetchSummaries(start, end)
		if err != nil {
			return nil, err
		}

		count := 0
		for _, summary := range summaries {
			if containsTopic(summaryTopics(summary), slug) {
				count++
			}
		}

		history = append(history, TopicHistoryPoint{
			AnchorDate: start.Format("2006-01-02"),
			Count:      count,
			Label:      label,
		})
	}

	return history, nil
}

func summaryTopics(summary models.AISummary) []TopicTag {
	if len(summary.SummaryTopics) > 0 {
		result := make([]TopicTag, 0, len(summary.SummaryTopics))
		for _, link := range summary.SummaryTopics {
			if link.TopicTag == nil {
				continue
			}
			result = append(result, TopicTag{
				ID:       link.TopicTag.ID,
				Label:    link.TopicTag.Label,
				Slug:     link.TopicTag.Slug,
				Category: normalizeDisplayCategory(link.TopicTag.Kind, link.TopicTag.Category),
				Icon:     link.TopicTag.Icon,
				Aliases:  parseAliasesFromJSON(link.TopicTag.Aliases),
				Kind:     normalizeTopicKind(link.TopicTag.Kind, link.TopicTag.Category),
				Score:    link.Score,
			})
		}
		if len(result) > 0 {
			return dedupeTopics(result)
		}
	}

	return ExtractTopics(ExtractionInput{
		Title:        summary.Title,
		Summary:      summary.Summary,
		FeedName:     feedLabel(summary),
		CategoryName: categoryLabel(summary),
	})
}

func mapSummaryCard(summary models.AISummary, topics []TopicTag, articles []TopicArticleCard) TopicSummaryCard {
	return TopicSummaryCard{
		ID:           summary.ID,
		Title:        summary.Title,
		Summary:      summary.Summary,
		FeedName:     feedLabel(summary),
		FeedColor:    feedColor(summary),
		CategoryName: categoryLabel(summary),
		ArticleCount: summary.ArticleCount,
		CreatedAt:    summary.CreatedAt.In(topicGraphCST).Format(time.RFC3339),
		Topics:       topics,
		Articles:     articles,
	}
}

func fetchArticlesForSummaries(summaries []models.AISummary) map[uint][]TopicArticleCard {
	result := make(map[uint][]TopicArticleCard, len(summaries))
	articleIDs := make([]uint, 0)
	summaryArticleIDs := make(map[uint][]uint, len(summaries))

	for _, summary := range summaries {
		if strings.TrimSpace(summary.Articles) == "" {
			continue
		}

		var ids []uint
		if err := json.Unmarshal([]byte(summary.Articles), &ids); err != nil || len(ids) == 0 {
			continue
		}

		summaryArticleIDs[summary.ID] = ids
		articleIDs = append(articleIDs, ids...)
	}

	if len(articleIDs) == 0 {
		return result
	}

	var articles []models.Article
	if err := database.DB.Where("id IN ?", articleIDs).Find(&articles).Error; err != nil {
		return result
	}

	articleMap := make(map[uint]models.Article, len(articles))
	for _, article := range articles {
		articleMap[article.ID] = article
	}

	for summaryID, ids := range summaryArticleIDs {
		cards := make([]TopicArticleCard, 0, len(ids))
		for _, articleID := range ids {
			article, ok := articleMap[articleID]
			if !ok {
				continue
			}
			cards = append(cards, TopicArticleCard{ID: article.ID, Title: article.Title, Link: article.Link})
		}
		result[summaryID] = cards
	}

	return result
}

func containsTopic(items []TopicTag, slug string) bool {
	for _, item := range items {
		if item.Slug == slug {
			return true
		}
	}
	return false
}

func pickTopic(items []TopicTag, slug string) TopicTag {
	for _, item := range items {
		if item.Slug == slug {
			return item
		}
	}
	return TopicTag{}
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

// TopicsByCategoryResult holds tags grouped by category
type TopicsByCategoryResult struct {
	Events   []TopicTag `json:"events"`
	People   []TopicTag `json:"people"`
	Keywords []TopicTag `json:"keywords"`
}

// BuildTopicsByCategory builds topic lists grouped by category from article tags
func BuildTopicsByCategory(kind string, anchor time.Time) (*TopicsByCategoryResult, error) {
	windowStart, windowEnd, _, err := resolveWindow(kind, anchor)
	if err != nil {
		return nil, err
	}

	// Get articles from the time window with their tags
	var articleTags []models.ArticleTopicTag
	err = database.DB.
		Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
		Joins("JOIN topic_tags ON topic_tags.id = article_topic_tags.topic_tag_id").
		Where("articles.created_at >= ? AND articles.created_at < ?", windowStart, windowEnd).
		Preload("TopicTag").
		Find(&articleTags).Error
	if err != nil {
		return nil, err
	}

	// Group tags by category and aggregate scores
	eventScores := make(map[string]*TopicTag)
	personScores := make(map[string]*TopicTag)
	keywordScores := make(map[string]*TopicTag)

	for _, at := range articleTags {
		if at.TopicTag == nil {
			continue
		}

		tag := TopicTag{
			ID:       at.TopicTag.ID,
			Label:    at.TopicTag.Label,
			Slug:     at.TopicTag.Slug,
			Category: normalizeDisplayCategory(at.TopicTag.Kind, at.TopicTag.Category),
			Icon:     at.TopicTag.Icon,
			Kind:     normalizeTopicKind(at.TopicTag.Kind, at.TopicTag.Category),
			Score:    at.Score,
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

	// Convert maps to slices and sort by score
	result := &TopicsByCategoryResult{
		Events:   sortTagsByScoreMap(eventScores),
		People:   sortTagsByScoreMap(personScores),
		Keywords: sortTagsByScoreMap(keywordScores),
	}

	return result, nil
}

// sortTagsByScoreMap converts a map of tags to a sorted slice
func sortTagsByScoreMap(tagMap map[string]*TopicTag) []TopicTag {
	result := make([]TopicTag, 0, len(tagMap))
	for _, tag := range tagMap {
		result = append(result, *tag)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].Label < result[j].Label
		}
		return result[i].Score > result[j].Score
	})

	return result
}
