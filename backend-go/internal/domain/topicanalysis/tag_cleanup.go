package topicanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/jsonutil"
	"my-robot-backend/internal/platform/logging"

	"gorm.io/gorm"
)

type ZombieTagCriteria struct {
	MinAgeDays int
	Categories []string
}

func CleanupZombieTags(criteria ZombieTagCriteria) (int, error) {
	result := buildZombieTagQuery(database.DB.Model(&models.TopicTag{}), criteria)

	var count int64
	if err := result.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count zombie tags: %w", err)
	}

	if count == 0 {
		return 0, nil
	}

	if err := result.Updates(map[string]interface{}{
		"status": "inactive",
	}).Error; err != nil {
		return 0, fmt.Errorf("deactivate zombie tags: %w", err)
	}

	logging.Infof("CleanupZombieTags: deactivated %d zombie tags", count)
	return int(count), nil
}

func buildZombieTagQuery(db *gorm.DB, criteria ZombieTagCriteria) *gorm.DB {
	query := db.Where("status = ?", "active")
	if len(criteria.Categories) > 0 {
		query = query.Where("category IN ?", criteria.Categories)
	}
	if criteria.MinAgeDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -criteria.MinAgeDays)
		query = query.Where("created_at < ?", cutoff)
	}

	return query.
		Where("NOT EXISTS (SELECT 1 FROM topic_tag_relations r WHERE (r.parent_id = topic_tags.id OR r.child_id = topic_tags.id) AND r.relation_type = ?)", "abstract").
		Where("NOT EXISTS (SELECT 1 FROM article_topic_tags att WHERE att.topic_tag_id = topic_tags.id)")
}

func BuildZombieTagSubQuery(criteria ZombieTagCriteria) string {
	return fmt.Sprintf(`
		SELECT t.id FROM topic_tags t
		WHERE t.status = 'active'
		  AND t.category IN (%s)
		  AND t.created_at < NOW() - INTERVAL '%d days'
		  AND NOT EXISTS (
		    SELECT 1 FROM topic_tag_relations r
		    WHERE (r.parent_id = t.id OR r.child_id = t.id) AND r.relation_type = 'abstract'
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM article_topic_tags att
		    WHERE att.topic_tag_id = t.id
		  )
		`, quoteCategories(criteria.Categories), criteria.MinAgeDays)
}

func quoteCategories(categories []string) string {
	quoted := ""
	for i, c := range categories {
		if i > 0 {
			quoted += ", "
		}
		quoted += fmt.Sprintf("'%s'", c)
	}
	return quoted
}

type FlatTagInfo struct {
	ID           uint               `json:"id"`
	Label        string             `json:"label"`
	Description  string             `json:"description"`
	Source       string             `json:"source"`
	ArticleCount int                `json:"article_count"`
	ChildCount   int                `json:"child_count"`
	Metadata     models.MetadataMap `json:"person_attrs,omitempty"`
}

type flatMergeJudgment struct {
	Merges []flatMergeItem `json:"merges,omitempty"`
	Notes  string          `json:"notes,omitempty"`
}

type flatMergeItem struct {
	SourceID uint   `json:"source_id"`
	TargetID uint   `json:"target_id"`
	Reason   string `json:"reason"`
}

func CollectFlatTagBatch(category string, batchSize int) ([]FlatTagInfo, error) {
	var tags []models.TopicTag
	if err := database.DB.
		Where("category = ? AND status = 'active' AND source = 'abstract'", category).
		Limit(batchSize).
		Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("load abstract tags: %w", err)
	}

	tagIDs := make([]uint, len(tags))
	for i, t := range tags {
		tagIDs[i] = t.ID
	}

	articleCounts := countArticlesByTag(tagIDs, "")

	childCounts := make(map[uint]int)
	var childRows []struct {
		ParentID uint `gorm:"column:parent_id"`
		Cnt      int  `gorm:"column:cnt"`
	}
	database.DB.Model(&models.TopicTagRelation{}).
		Select("parent_id, count(*) as cnt").
		Where("parent_id IN ? AND relation_type = 'abstract'", tagIDs).
		Group("parent_id").
		Scan(&childRows)
	for _, r := range childRows {
		childCounts[r.ParentID] = r.Cnt
	}

	result := make([]FlatTagInfo, len(tags))
	for i, t := range tags {
		result[i] = FlatTagInfo{
			ID:           t.ID,
			Label:        t.Label,
			Description:  truncateStr(t.Description, 200),
			Source:       t.Source,
			ArticleCount: articleCounts[t.ID],
			ChildCount:   childCounts[t.ID],
			Metadata:     t.Metadata,
		}
	}
	return result, nil
}

func BuildFlatMergePrompt(tags []FlatTagInfo, category string) string {
	promptData := map[string]interface{}{
		"category": category,
		"total":    len(tags),
		"tags":     tags,
	}

	promptJSON, _ := json.MarshalIndent(promptData, "", "  ")

	return fmt.Sprintf(`你是一位标签分类专家。请分析以下 %s 类别的抽象标签列表，找出语义重复或高度相似的标签对。

标签列表：
%s

请返回以下格式的 JSON：
{
  "merges": [
    {
      "source_id": 123,
      "target_id": 456,
      "reason": "这两个标签描述的是同一个概念，应该合并"
    }
  ],
  "notes": "其他观察（可选）"
}

规则：
1. merges 是可选的，可以为空数组
2. source_id: 被合并的标签（子标签数更少或描述更窄的那个）
3. target_id: 保留的目标标签（子标签数更多或描述更广的那个）
4. 只合并真正描述同一核心概念的标签，不要合并仅有部分重叠的标签
5. 如果没有需要合并的，返回空数组
6. 只返回真正有把握的建议`, category, string(promptJSON))
}

func ExecuteFlatMerge(category string, batchSize int) (int, []string, error) {
	tags, err := CollectFlatTagBatch(category, batchSize)
	if err != nil {
		return 0, nil, fmt.Errorf("collect tags: %w", err)
	}
	if len(tags) == 0 {
		return 0, nil, nil
	}

	prompt := BuildFlatMergePrompt(tags, category)
	judgment, err := callFlatMergeLLM(prompt)
	if err != nil {
		return 0, nil, fmt.Errorf("LLM call: %w", err)
	}

	tagMap := make(map[uint]*FlatTagInfo)
	for i := range tags {
		tagMap[tags[i].ID] = &tags[i]
	}

	var errors []string
	merged := 0
	for _, merge := range judgment.Merges {
		if err := validateFlatMerge(merge, tagMap); err != nil {
			errors = append(errors, fmt.Sprintf("merge %d→%d: %v", merge.SourceID, merge.TargetID, err))
			continue
		}
		if err := MergeTags(merge.SourceID, merge.TargetID); err != nil {
			errors = append(errors, fmt.Sprintf("merge %d→%d: %v", merge.SourceID, merge.TargetID, err))
			continue
		}
		merged++
	}

	logging.Infof("ExecuteFlatMerge(%s): %d tags analyzed, %d merges applied", category, len(tags), merged)
	return merged, errors, nil
}

func validateFlatMerge(merge flatMergeItem, tagMap map[uint]*FlatTagInfo) error {
	source, ok := tagMap[merge.SourceID]
	if !ok {
		return fmt.Errorf("source %d not found", merge.SourceID)
	}
	target, ok := tagMap[merge.TargetID]
	if !ok {
		return fmt.Errorf("target %d not found", merge.TargetID)
	}
	if merge.SourceID == merge.TargetID {
		return fmt.Errorf("same tag")
	}
	if source.ChildCount > 0 && target.ChildCount > 0 {
		if source.ChildCount > target.ChildCount {
			return fmt.Errorf("source has more children (%d) than target (%d), swap recommended", source.ChildCount, target.ChildCount)
		}
	}
	return nil
}

func callFlatMergeLLM(prompt string) (*flatMergeJudgment, error) {
	router := airouter.NewRouter()
	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy cleanup assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"merges": {
					Type: "array",
					Items: &airouter.SchemaProperty{
						Type: "object",
						Properties: map[string]airouter.SchemaProperty{
							"source_id": {Type: "integer"},
							"target_id": {Type: "integer"},
							"reason":    {Type: "string"},
						},
						Required: []string{"source_id", "target_id", "reason"},
					},
				},
				"notes": {Type: "string"},
			},
		},
		Temperature: func() *float64 { f := 0.2; return &f }(),
		Metadata:    map[string]any{"operation": "tag_flat_merge"},
	}

	result, err := router.Chat(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	content := jsonutil.SanitizeLLMJSON(result.Content)
	var judgment flatMergeJudgment
	if err := json.Unmarshal([]byte(content), &judgment); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}
	return &judgment, nil
}

func CleanupOrphanedRelations() (int, error) {
	result := database.DB.Where(
		"relation_type = 'abstract' AND (parent_id IN (SELECT id FROM topic_tags WHERE status != 'active') OR child_id IN (SELECT id FROM topic_tags WHERE status != 'active'))",
	).Delete(&models.TopicTagRelation{})

	if result.Error != nil {
		return 0, fmt.Errorf("cleanup orphaned relations: %w", result.Error)
	}

	deleted := int(result.RowsAffected)
	if deleted > 0 {
		logging.Infof("CleanupOrphanedRelations: removed %d orphaned relations", deleted)
	}
	return deleted, nil
}

func CleanupMultiParentConflicts() (int, []string, error) {
	var conflicts []struct {
		ChildID uint `gorm:"column:child_id"`
		Cnt     int  `gorm:"column:cnt"`
	}
	database.DB.Model(&models.TopicTagRelation{}).
		Select("child_id, count(*) as cnt").
		Where("relation_type = 'abstract'").
		Group("child_id").
		Having("count(*) > 1").
		Scan(&conflicts)

	if len(conflicts) == 0 {
		return 0, nil, nil
	}

	// 收集所有冲突详情
	var multiConflicts []multiParentConflict
	for _, c := range conflicts {
		var relations []models.TopicTagRelation
		if err := database.DB.Where("child_id = ? AND relation_type = ?", c.ChildID, "abstract").
			Preload("Parent").Find(&relations).Error; err != nil {
			continue
		}
		var parents []parentWithInfo
		var childTag models.TopicTag
		for _, r := range relations {
			if r.Parent != nil {
				parents = append(parents, parentWithInfo{RelationID: r.ID, Parent: r.Parent, SimilarityScore: r.SimilarityScore})
			}
		}
		if len(parents) <= 1 {
			continue
		}
		if err := database.DB.First(&childTag, c.ChildID).Error; err != nil {
			continue
		}
		multiConflicts = append(multiConflicts, multiParentConflict{
			ChildID: c.ChildID,
			Parents: parents,
			Child:   &childTag,
		})
	}

	// 批量解决（每批最多 10 个冲突）
	batchSize := 10
	totalResolved := 0
	var allErrors []string
	for i := 0; i < len(multiConflicts); i += batchSize {
		end := i + batchSize
		if end > len(multiConflicts) {
			end = len(multiConflicts)
		}
		batch := multiConflicts[i:end]
		resolved, errors := batchResolveMultiParentConflicts(batch)
		totalResolved += resolved
		allErrors = append(allErrors, errors...)
	}

	logging.Infof("CleanupMultiParentConflicts: resolved %d conflicts", totalResolved)
	return totalResolved, allErrors, nil
}

func deactivateTagsWithCleanup(tagIDs []uint) error {
	if len(tagIDs) == 0 {
		return nil
	}
	database.DB.Where("topic_tag_id IN ?", tagIDs).Delete(&models.TopicTagEmbedding{})
	database.DB.Where("parent_id IN ? OR child_id IN ?", tagIDs, tagIDs).
		Where("relation_type = ?", "abstract").Delete(&models.TopicTagRelation{})
	return database.DB.Model(&models.TopicTag{}).Where("id IN ?", tagIDs).
		Updates(map[string]interface{}{"status": "inactive"}).Error
}

func CleanupZeroArticleTags(categories []string) (int, error) {
	query := database.DB.Model(&models.TopicTag{}).
		Where("status = ? AND kind != ? AND source != ?", "active", "abstract", "abstract").
		Where("category IN ?", categories).
		Where("NOT EXISTS (SELECT 1 FROM article_topic_tags att WHERE att.topic_tag_id = topic_tags.id)")

	var ids []uint
	if err := query.Pluck("topic_tags.id", &ids).Error; err != nil {
		return 0, fmt.Errorf("pluck zero-article tag ids: %w", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	if err := deactivateTagsWithCleanup(ids); err != nil {
		return 0, fmt.Errorf("cleanup zero-article tags: %w", err)
	}

	logging.Infof("CleanupZeroArticleTags: deactivated %d zero-article tags in categories %v", len(ids), categories)
	return len(ids), nil
}

func CleanupLowQualitySingleArticleTags(category string, maxScore float64) (int, error) {
	query := database.DB.Model(&models.TopicTag{}).
		Where("status = ? AND kind != ? AND source != ?", "active", "abstract", "abstract").
		Where("category = ?", category).
		Where("quality_score < ?", maxScore).
		Where("(SELECT COUNT(*) FROM article_topic_tags att WHERE att.topic_tag_id = topic_tags.id) = 1")

	var ids []uint
	if err := query.Pluck("topic_tags.id", &ids).Error; err != nil {
		return 0, fmt.Errorf("pluck low-quality single-article tag ids: %w", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	if err := deactivateTagsWithCleanup(ids); err != nil {
		return 0, fmt.Errorf("cleanup low-quality single-article tags: %w", err)
	}

	logging.Infof("CleanupLowQualitySingleArticleTags: deactivated %d low-quality single-article tags in category %s (maxScore=%.2f)", len(ids), category, maxScore)
	return len(ids), nil
}

func CleanupStaleZeroScoreTags(ageDays int) (int, error) {
	cutoff := time.Now().AddDate(0, 0, -ageDays)
	query := database.DB.Model(&models.TopicTag{}).
		Where("status = ? AND kind != ? AND source != ?", "active", "abstract", "abstract").
		Where("quality_score < ?", 0.05).
		Where("created_at < ?", cutoff)

	var ids []uint
	if err := query.Pluck("topic_tags.id", &ids).Error; err != nil {
		return 0, fmt.Errorf("pluck stale zero-score tag ids: %w", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	if err := deactivateTagsWithCleanup(ids); err != nil {
		return 0, fmt.Errorf("cleanup stale zero-score tags: %w", err)
	}

	logging.Infof("CleanupStaleZeroScoreTags: deactivated %d stale zero-score tags (age > %d days)", len(ids), ageDays)
	return len(ids), nil
}

func CleanupEmptyAbstractNodes() (int, error) {
	query := database.DB.Model(&models.TopicTag{}).
		Where("source = ? AND status = ?", "abstract", "active").
		Where("NOT EXISTS (SELECT 1 FROM topic_tag_relations r WHERE r.parent_id = topic_tags.id AND r.relation_type = ?)", "abstract")

	var ids []uint
	if err := query.Pluck("topic_tags.id", &ids).Error; err != nil {
		return 0, fmt.Errorf("load empty abstract ids: %w", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	if err := database.DB.Model(&models.TopicTag{}).Where("id IN ?", ids).Updates(map[string]interface{}{
		"status": "inactive",
	}).Error; err != nil {
		return 0, fmt.Errorf("cleanup empty abstracts: %w", err)
	}

	logging.Infof("CleanupEmptyAbstractNodes: deactivated %d empty abstract tags", len(ids))
	return len(ids), nil
}
