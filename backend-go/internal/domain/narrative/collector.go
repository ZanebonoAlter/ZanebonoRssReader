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

func CollectTagInputsByCategory(date time.Time, categoryID uint) ([]TagInput, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var feedIDs []uint
	if err := database.DB.Model(&models.Feed{}).
		Where("category_id = ?", categoryID).
		Pluck("id", &feedIDs).Error; err != nil || len(feedIDs) == 0 {
		return nil, nil
	}

	var tagIDs []uint
	database.DB.Model(&models.ArticleTopicTag{}).
		Select("DISTINCT article_topic_tags.topic_tag_id").
		Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
		Where("articles.feed_id IN ? AND articles.pub_date >= ? AND articles.pub_date < ?", feedIDs, startOfDay, endOfDay).
		Pluck("article_topic_tags.topic_tag_id", &tagIDs)

	if len(tagIDs) == 0 {
		return nil, nil
	}

	var tags []models.TopicTag
	database.DB.Where("id IN ? AND status = ?", tagIDs, "active").
		Order("quality_score DESC, feed_count DESC").
		Limit(100).
		Find(&tags)

	if len(tags) == 0 {
		return nil, nil
	}

	type countRow struct {
		TopicTagID uint `json:"topic_tag_id"`
		Cnt        int  `json:"cnt"`
	}
	var counts []countRow
	database.DB.Model(&models.ArticleTopicTag{}).
		Select("article_topic_tags.topic_tag_id, COUNT(DISTINCT article_topic_tags.article_id) as cnt").
		Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
		Where("article_topic_tags.topic_tag_id IN ? AND articles.feed_id IN ? AND articles.pub_date >= ? AND articles.pub_date < ?", tagIDs, feedIDs, startOfDay, endOfDay).
		Group("article_topic_tags.topic_tag_id").
		Scan(&counts)

	countMap := make(map[uint]int, len(counts))
	for _, c := range counts {
		countMap[c.TopicTagID] = c.Cnt
	}

	inputs := make([]TagInput, 0, len(tags))
	for _, tag := range tags {
		inputs = append(inputs, TagInput{
			ID:           tag.ID,
			Label:        tag.Label,
			Category:     tag.Category,
			Description:  tag.Description,
			ArticleCount: countMap[tag.ID],
			Source:       tag.Source,
		})
	}
	return inputs, nil
}

type ActiveCategory struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	Color        string `json:"color"`
	ArticleCount int    `json:"article_count"`
	TagCount     int    `json:"tag_count"`
}

func CollectActiveCategories(date time.Time) ([]ActiveCategory, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var categories []models.Category
	database.DB.Find(&categories)

	var result []ActiveCategory
	for _, cat := range categories {
		var feedIDs []uint
		database.DB.Model(&models.Feed{}).Where("category_id = ?", cat.ID).Pluck("id", &feedIDs)
		if len(feedIDs) == 0 {
			continue
		}

		var articleCount int64
		database.DB.Model(&models.Article{}).
			Where("feed_id IN ? AND pub_date >= ? AND pub_date < ?", feedIDs, startOfDay, endOfDay).
			Count(&articleCount)

		if articleCount == 0 {
			continue
		}

		var tagCount int64
		database.DB.Model(&models.ArticleTopicTag{}).
			Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
			Where("articles.feed_id IN ? AND articles.pub_date >= ? AND articles.pub_date < ?", feedIDs, startOfDay, endOfDay).
			Distinct("article_topic_tags.topic_tag_id").
			Count(&tagCount)

		result = append(result, ActiveCategory{
			ID:           cat.ID,
			Name:         cat.Name,
			Icon:         cat.Icon,
			Color:        cat.Color,
			ArticleCount: int(articleCount),
			TagCount:     int(tagCount),
		})
	}
	return result, nil
}

func CollectPreviousNarratives(date time.Time, scopeType string, categoryID *uint) ([]PreviousNarrative, error) {
	yesterday := date.AddDate(0, 0, -1)
	query := database.DB.
		Where("period = ? AND period_date >= ? AND period_date < ?", "daily", yesterday, date)

	if scopeType != "" {
		query = query.Where("scope_type = ?", scopeType)
		if categoryID != nil {
			query = query.Where("scope_category_id = ?", *categoryID)
		}
	}

	var narratives []models.NarrativeSummary
	if err := query.Order("id ASC").Find(&narratives).Error; err != nil {
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
