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

	summaries, err := fetchSummaries(windowStart, windowEnd)
	if err != nil {
		return nil, err
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
		canonical = TopicTag{Label: strings.ReplaceAll(strings.Title(strings.ReplaceAll(slug, "-", " ")), "Ai", "AI"), Slug: slug, Kind: "topic", Score: 0}
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
				topicNodes[topic.Slug] = &GraphNode{ID: topic.Slug, Label: topic.Label, Slug: topic.Slug, Kind: "topic", Weight: topic.Score, SummaryCount: 0}
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
				Label: link.TopicTag.Label,
				Slug:  link.TopicTag.Slug,
				Kind:  link.TopicTag.Kind,
				Score: link.Score,
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
