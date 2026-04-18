package narrative

import (
	"fmt"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

type TagInput struct {
	ID           uint   `json:"id"`
	Label        string `json:"label"`
	Category     string `json:"category"`
	Description  string `json:"description"`
	ArticleCount int    `json:"article_count"`
	IsAbstract   bool   `json:"is_abstract"`
	Source       string `json:"source"`
}

type PreviousNarrative struct {
	ID         uint64 `json:"id"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Status     string `json:"status"`
	Generation int    `json:"generation"`
}

func CollectTagInputs(date time.Time) ([]TagInput, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var inputs []TagInput

	rootAbstractTags, err := collectRootAbstractTags(startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("collect root abstract tags: %w", err)
	}
	inputs = append(inputs, rootAbstractTags...)

	unclassifiedTags, err := collectUnclassifiedTags(startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("collect unclassified tags: %w", err)
	}
	inputs = append(inputs, unclassifiedTags...)

	return inputs, nil
}

func collectRootAbstractTags(since, until time.Time) ([]TagInput, error) {
	var parentIDs []uint
	database.DB.Model(&models.TopicTagRelation{}).
		Where("relation_type = ?", "abstract").
		Distinct("parent_id").
		Pluck("parent_id", &parentIDs)

	if len(parentIDs) == 0 {
		return nil, nil
	}

	var childIDs []uint
	database.DB.Model(&models.TopicTagRelation{}).
		Where("relation_type = ? AND parent_id IN ?", "abstract", parentIDs).
		Distinct("child_id").
		Pluck("child_id", &childIDs)

	childSet := make(map[uint]bool, len(childIDs))
	for _, id := range childIDs {
		childSet[id] = true
	}

	var rootIDs []uint
	for _, id := range parentIDs {
		if !childSet[id] {
			rootIDs = append(rootIDs, id)
		}
	}

	if len(rootIDs) == 0 {
		return nil, nil
	}

	var tags []models.TopicTag
	database.DB.Where("id IN ? AND status = ?", rootIDs, "active").Find(&tags)

	type countRow struct {
		TopicTagID uint `json:"topic_tag_id"`
		Cnt        int  `json:"cnt"`
	}
	var counts []countRow
	database.DB.Model(&models.ArticleTopicTag{}).
		Select("article_topic_tags.topic_tag_id, COUNT(DISTINCT article_topic_tags.article_id) as cnt").
		Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
		Where("article_topic_tags.topic_tag_id IN ? AND articles.pub_date >= ? AND articles.pub_date < ?", rootIDs, since, until).
		Group("article_topic_tags.topic_tag_id").
		Scan(&counts)

	countMap := make(map[uint]int, len(counts))
	for _, c := range counts {
		countMap[c.TopicTagID] = c.Cnt
	}

	var inputs []TagInput
	for _, tag := range tags {
		inputs = append(inputs, TagInput{
			ID:           tag.ID,
			Label:        tag.Label,
			Category:     tag.Category,
			Description:  tag.Description,
			ArticleCount: countMap[tag.ID],
			IsAbstract:   true,
			Source:       "abstract",
		})
	}
	return inputs, nil
}

func collectUnclassifiedTags(since, until time.Time) ([]TagInput, error) {
	var allRelated []uint
	database.DB.Model(&models.TopicTagRelation{}).
		Where("relation_type = ?", "abstract").
		Pluck("parent_id", &allRelated)
	var childIDs []uint
	database.DB.Model(&models.TopicTagRelation{}).
		Where("relation_type = ?", "abstract").
		Pluck("child_id", &childIDs)
	allRelated = append(allRelated, childIDs...)
	relatedSet := make(map[uint]bool, len(allRelated))
	for _, id := range allRelated {
		relatedSet[id] = true
	}

	query := database.DB.Model(&models.TopicTag{}).
		Where("status = ? AND source != ?", "active", "abstract")

	if len(relatedSet) > 0 {
		excl := make([]uint, 0, len(relatedSet))
		for id := range relatedSet {
			excl = append(excl, id)
		}
		query = query.Where("id NOT IN ?", excl)
	}

	activeSubquery := database.DB.Model(&models.ArticleTopicTag{}).
		Select("DISTINCT article_topic_tags.topic_tag_id").
		Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
		Where("articles.pub_date >= ? AND articles.pub_date < ?", since, until)
	query = query.Where("id IN (?)", activeSubquery)

	var tags []models.TopicTag
	if err := query.Order("quality_score DESC, feed_count DESC").Limit(100).Find(&tags).Error; err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		return nil, nil
	}

	tagIDs := make([]uint, len(tags))
	for i, t := range tags {
		tagIDs[i] = t.ID
	}

	type countRow struct {
		TopicTagID uint `json:"topic_tag_id"`
		Cnt        int  `json:"cnt"`
	}
	var counts []countRow
	database.DB.Model(&models.ArticleTopicTag{}).
		Select("article_topic_tags.topic_tag_id, COUNT(DISTINCT article_topic_tags.article_id) as cnt").
		Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
		Where("article_topic_tags.topic_tag_id IN ? AND articles.pub_date >= ? AND articles.pub_date < ?", tagIDs, since, until).
		Group("article_topic_tags.topic_tag_id").
		Scan(&counts)

	countMap := make(map[uint]int, len(counts))
	for _, c := range counts {
		countMap[c.TopicTagID] = c.Cnt
	}

	var inputs []TagInput
	for _, tag := range tags {
		inputs = append(inputs, TagInput{
			ID:           tag.ID,
			Label:        tag.Label,
			Category:     tag.Category,
			Description:  tag.Description,
			ArticleCount: countMap[tag.ID],
			IsAbstract:   false,
			Source:       tag.Source,
		})
	}
	return inputs, nil
}

func CollectPreviousNarratives(date time.Time) ([]PreviousNarrative, error) {
	yesterday := date.AddDate(0, 0, -1)
	var narratives []models.NarrativeSummary
	if err := database.DB.
		Where("period = ? AND period_date >= ? AND period_date < ?", "daily", yesterday, date).
		Order("id ASC").
		Find(&narratives).Error; err != nil {
		return nil, err
	}

	var result []PreviousNarrative
	for _, n := range narratives {
		result = append(result, PreviousNarrative{
			ID:         uint64(n.ID),
			Title:      n.Title,
			Summary:    n.Summary,
			Status:     n.Status,
			Generation: n.Generation,
		})
	}
	return result, nil
}
