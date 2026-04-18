package topicanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topictypes"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/jsonutil"
	"my-robot-backend/internal/platform/logging"

	"gorm.io/gorm"
)

const (
	// maxAbstractNameLen limits LLM-returned abstract tag names to prevent abuse
	maxAbstractNameLen = 160

	ActionMerge    = "merge"
	ActionAbstract = "abstract"
	ActionNone     = "none"
)

// TagExtractionResult is the return type for ExtractAbstractTag,
// supporting both "merge" (reuse existing tag) and "abstract" (create abstract parent) actions.
type TagExtractionResult struct {
	Action      string           // "merge" or "abstract"
	MergeTarget *models.TopicTag // for "merge": the existing tag to reuse
	MergeLabel  string           // for "merge": LLM-recommended unified label
	AbstractTag *models.TopicTag // for "abstract": the new abstract parent tag
}

// TagHierarchyNode represents a node in the tag hierarchy tree
type TagHierarchyNode struct {
	ID              uint               `json:"id"`
	Label           string             `json:"label"`
	Slug            string             `json:"slug"`
	Category        string             `json:"category"`
	Icon            string             `json:"icon"`
	FeedCount       int                `json:"feed_count"`
	ArticleCount    int                `json:"article_count"`
	SimilarityScore float64            `json:"similarity_score,omitempty"`
	IsActive        bool               `json:"is_active"`
	QualityScore    float64            `json:"quality_score,omitempty"`
	IsLowQuality    bool               `json:"is_low_quality,omitempty"`
	Children        []TagHierarchyNode `json:"children"`
}

type ExtractAbstractTagOption func(*extractAbstractTagConfig)

type extractAbstractTagConfig struct {
	narrativeContext string
}

func WithNarrativeContext(ctx string) ExtractAbstractTagOption {
	return func(c *extractAbstractTagConfig) {
		c.narrativeContext = ctx
	}
}

func ExtractAbstractTag(ctx context.Context, candidates []TagCandidate, newLabel string, category string, opts ...ExtractAbstractTagOption) (*TagExtractionResult, error) {
	if len(candidates) < 1 {
		return nil, fmt.Errorf("need at least 1 candidate for abstract tag extraction, got %d", len(candidates))
	}

	cfg := &extractAbstractTagConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if category == "" && len(candidates) > 0 && candidates[0].Tag != nil {
		category = candidates[0].Tag.Category
	}
	if category == "" {
		category = "keyword"
	}

	judgments, err := callLLMForTagJudgment(ctx, candidates, newLabel, category, cfg.narrativeContext)
	if err != nil {
		logging.Warnf("Tag judgment LLM call failed: %v", err)
		return nil, err
	}

	return processJudgments(ctx, judgments, candidates, newLabel, category)
}

func processJudgments(ctx context.Context, judgments []tagJudgment, candidates []TagCandidate, newLabel string, category string) (*TagExtractionResult, error) {
	var mergeJudgments, abstractJudgments []tagJudgment
	for _, j := range judgments {
		switch j.Action {
		case ActionMerge:
			mergeJudgments = append(mergeJudgments, j)
		case ActionAbstract:
			abstractJudgments = append(abstractJudgments, j)
		}
	}

	if len(mergeJudgments) > 0 {
		bestMerge := mergeJudgments[0]
		mergeTarget := selectMergeTarget(candidates, bestMerge.MergeTarget, bestMerge.MergeLabel)
		if mergeTarget == nil {
			return nil, fmt.Errorf("no suitable merge target found for label %q (target=%q)", bestMerge.MergeLabel, bestMerge.MergeTarget)
		}
		logging.Infof("Tag judgment: merge into existing tag %q (id=%d), label=%q", mergeTarget.Label, mergeTarget.ID, bestMerge.MergeLabel)
		return &TagExtractionResult{
			Action:      ActionMerge,
			MergeTarget: mergeTarget,
			MergeLabel:  bestMerge.MergeLabel,
		}, nil
	}

	if len(abstractJudgments) > 0 {
		bestAbstract := abstractJudgments[0]
		return processAbstractJudgment(ctx, candidates, bestAbstract, newLabel, category)
	}

	logging.Infof("Tag judgment: all candidates independent for %q", newLabel)
	return &TagExtractionResult{
		Action: ActionNone,
	}, nil
}

func processAbstractJudgment(ctx context.Context, candidates []TagCandidate, judgment tagJudgment, newLabel string, category string) (*TagExtractionResult, error) {
	abstractName := judgment.AbstractName
	abstractDesc := judgment.Description

	slug := topictypes.Slugify(abstractName)
	if slug == "" {
		return nil, fmt.Errorf("generated empty slug for abstract name %q", abstractName)
	}

	candidateSlugs := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		if c.Tag != nil {
			candidateSlugs[c.Tag.Slug] = true
		}
	}

	if candidateSlugs[slug] {
		logging.Infof("Abstract name %q (slug=%s) collides with a candidate tag, falling back to merge", abstractName, slug)
		mergeTarget := selectMergeTarget(candidates, abstractName, judgment.MergeLabel)
		if mergeTarget == nil {
			return nil, fmt.Errorf("abstract name %q collides with candidate but no merge target found", abstractName)
		}
		return &TagExtractionResult{
			Action:      ActionMerge,
			MergeTarget: mergeTarget,
			MergeLabel:  abstractName,
		}, nil
	}

	var abstractTag *models.TopicTag
	var addedAnyRelation bool

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Check if an abstract tag with this slug already exists (dedup per D-05)
		var existing models.TopicTag
		if err := tx.Where("slug = ? AND category = ? AND status = ?", slug, category, "active").First(&existing).Error; err == nil {
			abstractTag = &existing
		} else {
			// Create new abstract tag
			abstractTag = &models.TopicTag{
				Slug:        slug,
				Label:       abstractName,
				Category:    category,
				Kind:        category,
				Source:      "abstract",
				Status:      "active",
				Description: abstractDesc,
			}
			if err := tx.Create(abstractTag).Error; err != nil {
				return fmt.Errorf("create abstract tag: %w", err)
			}

			// Generate embedding asynchronously
			go func(tagID uint) {
				es := NewEmbeddingService()
				tag := &models.TopicTag{ID: tagID, Label: abstractName, Category: category}
				emb, genErr := es.GenerateEmbedding(context.Background(), tag)
				if genErr != nil {
					logging.Warnf("Failed to generate embedding for abstract tag %d: %v", tagID, genErr)
					return
				}
				emb.TopicTagID = tagID
				if saveErr := es.SaveEmbedding(emb); saveErr != nil {
					logging.Warnf("Failed to save embedding for abstract tag %d: %v", tagID, saveErr)
					return
				}
				MatchAbstractTagHierarchy(context.Background(), tagID)
			}(abstractTag.ID)
		}

		for _, candidate := range candidates {
			if candidate.Tag == nil {
				continue
			}
			if candidate.Tag.ID == abstractTag.ID {
				continue
			}

			wouldCycle, err := wouldCreateCycle(tx, abstractTag.ID, candidate.Tag.ID)
			if err != nil {
				return fmt.Errorf("check cycle for candidate %d: %w", candidate.Tag.ID, err)
			}
			if wouldCycle {
				logging.Warnf("Skipping cyclic relation: abstract tag %d -> candidate %d", abstractTag.ID, candidate.Tag.ID)
				continue
			}

			var count int64
			tx.Model(&models.TopicTagRelation{}).
				Where("parent_id = ? AND child_id = ? AND relation_type = ?", abstractTag.ID, candidate.Tag.ID, "abstract").
				Count(&count)
			if count > 0 {
				continue
			}

			var existingParentCount int64
			tx.Model(&models.TopicTagRelation{}).
				Where("child_id = ? AND parent_id != ? AND relation_type = ?", candidate.Tag.ID, abstractTag.ID, "abstract").
				Count(&existingParentCount)
			if existingParentCount > 0 {
				logging.Infof("Skipping candidate %d (%s): already has an abstract parent", candidate.Tag.ID, candidate.Tag.Label)
				continue
			}

			relation := models.TopicTagRelation{
				ParentID:        abstractTag.ID,
				ChildID:         candidate.Tag.ID,
				RelationType:    "abstract",
				SimilarityScore: candidate.Similarity,
			}
			if err := tx.Create(&relation).Error; err != nil {
				return fmt.Errorf("create tag relation: %w", err)
			}
			addedAnyRelation = true
		}

		return nil
	})

	if err != nil {
		logging.Warnf("Abstract tag transaction failed: %v", err)
		return nil, err
	}

	logging.Infof("Abstract tag extracted: %s (id=%d) from candidates [%s]",
		abstractTag.Label, abstractTag.ID, candidateLabels(candidates))

	if addedAnyRelation {
		go EnqueueAbstractTagUpdate(abstractTag.ID, "new_child_added")
	}

	return &TagExtractionResult{
		Action:      ActionAbstract,
		AbstractTag: abstractTag,
	}, nil
}

// GetTagHierarchy returns the tag hierarchy tree for a given category filter.
// timeRange filters by article recency: "7d", "30d", or "" for no filter.
func GetTagHierarchy(category string, scopeFeedID uint, scopeCategoryID uint, timeRange string) ([]TagHierarchyNode, error) {
	var scopeTagIDs map[uint]bool
	if scopeFeedID > 0 || scopeCategoryID > 0 {
		var err error
		scopeTagIDs, err = resolveScopeTagIDs(scopeFeedID, scopeCategoryID)
		if err != nil {
			return nil, err
		}
		if len(scopeTagIDs) == 0 {
			return []TagHierarchyNode{}, nil
		}
	}

	query := database.DB.Model(&models.TopicTagRelation{}).
		Where("relation_type = ?", "abstract")

	var relations []models.TopicTagRelation
	if err := query.Find(&relations).Error; err != nil {
		return nil, fmt.Errorf("query tag relations: %w", err)
	}

	if len(relations) == 0 {
		return []TagHierarchyNode{}, nil
	}

	if scopeTagIDs != nil {
		filtered := make([]models.TopicTagRelation, 0, len(relations))
		for _, r := range relations {
			if scopeTagIDs[r.ParentID] || scopeTagIDs[r.ChildID] {
				filtered = append(filtered, r)
			}
		}
		relations = filtered
		if len(relations) == 0 {
			return []TagHierarchyNode{}, nil
		}
	}

	tagIDSet := make(map[uint]bool)
	for _, r := range relations {
		tagIDSet[r.ParentID] = true
		tagIDSet[r.ChildID] = true
	}

	activeTagIDs := resolveActiveTagIDs(timeRange, tagIDSet)

	if timeRange != "" {
		prunedRelations := make([]models.TopicTagRelation, 0, len(relations))
		for _, r := range relations {
			if activeTagIDs[r.ChildID] {
				prunedRelations = append(prunedRelations, r)
			}
		}
		relations = prunedRelations
		if len(relations) == 0 {
			return []TagHierarchyNode{}, nil
		}
		tagIDSet = make(map[uint]bool)
		for _, r := range relations {
			tagIDSet[r.ParentID] = true
			tagIDSet[r.ChildID] = true
		}
	}

	tagIDs := make([]uint, 0, len(tagIDSet))
	for id := range tagIDSet {
		tagIDs = append(tagIDs, id)
	}

	var tags []models.TopicTag
	if err := database.DB.Where("id IN ? AND status = ?", tagIDs, "active").Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("load tags: %w", err)
	}
	tagMap := make(map[uint]*models.TopicTag, len(tags))
	for i := range tags {
		tagMap[tags[i].ID] = &tags[i]
	}

	articleCounts := countArticlesByTag(tagIDs, timeRange)

	if category != "" {
		filteredRelations := make([]models.TopicTagRelation, 0, len(relations))
		for _, r := range relations {
			parent, ok := tagMap[r.ParentID]
			if ok && parent.Category == category {
				filteredRelations = append(filteredRelations, r)
			}
		}
		relations = filteredRelations
	}

	childrenMap := make(map[uint][]TagHierarchyNode)
	parentSet := make(map[uint]bool)
	for _, r := range relations {
		child, ok := tagMap[r.ChildID]
		if !ok {
			continue
		}
		childrenMap[r.ParentID] = append(childrenMap[r.ParentID], TagHierarchyNode{
			ID:              child.ID,
			Label:           child.Label,
			Slug:            child.Slug,
			Category:        child.Category,
			Icon:            child.Icon,
			FeedCount:       child.FeedCount,
			ArticleCount:    articleCounts[child.ID],
			SimilarityScore: r.SimilarityScore,
			IsActive:        timeRange != "" || activeTagIDs[child.ID],
			QualityScore:    child.QualityScore,
			IsLowQuality:    child.Source != "abstract" && child.QualityScore < 0.3,
			Children:        []TagHierarchyNode{},
		})
		parentSet[r.ParentID] = true
	}

	childSet := make(map[uint]bool)
	for _, r := range relations {
		childSet[r.ChildID] = true
	}

	var roots []TagHierarchyNode
	for parentID := range parentSet {
		parent, ok := tagMap[parentID]
		if !ok {
			continue
		}
		if childSet[parentID] && parentSet[parentID] {
			continue
		}
		children := buildHierarchy(childrenMap, parentID)
		roots = append(roots, TagHierarchyNode{
			ID:           parent.ID,
			Label:        parent.Label,
			Slug:         parent.Slug,
			Category:     parent.Category,
			Icon:         parent.Icon,
			FeedCount:    parent.FeedCount,
			ArticleCount: articleCounts[parent.ID],
			IsActive:     timeRange != "" || activeTagIDs[parent.ID],
			QualityScore: parent.QualityScore,
			Children:     children,
		})
	}

	return roots, nil
}

// buildHierarchy recursively builds the tree from the childrenMap
func buildHierarchy(childrenMap map[uint][]TagHierarchyNode, parentID uint) []TagHierarchyNode {
	return buildHierarchyWithVisited(childrenMap, parentID, make(map[uint]bool))
}

// buildHierarchyWithVisited recursively builds the tree with visited tracking to prevent cycles
func buildHierarchyWithVisited(childrenMap map[uint][]TagHierarchyNode, parentID uint, visited map[uint]bool) []TagHierarchyNode {
	if visited[parentID] {
		return []TagHierarchyNode{}
	}
	visited[parentID] = true
	defer delete(visited, parentID)

	children, ok := childrenMap[parentID]
	if !ok {
		return []TagHierarchyNode{}
	}
	result := make([]TagHierarchyNode, len(children))
	for i, child := range children {
		result[i] = child
		grandChildren := buildHierarchyWithVisited(childrenMap, child.ID, visited)
		result[i].Children = grandChildren
	}
	return result
}

// UpdateAbstractTagName updates the name and slug of an abstract tag.
func UpdateAbstractTagName(tagID uint, newName string) error {
	if tagID == 0 {
		return fmt.Errorf("tag ID must be > 0")
	}
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("new name must not be empty")
	}
	if len(newName) > maxAbstractNameLen {
		return fmt.Errorf("new name exceeds %d characters", maxAbstractNameLen)
	}

	// Verify this tag has parent relations (is an abstract tag)
	var count int64
	database.DB.Model(&models.TopicTagRelation{}).
		Where("parent_id = ? AND relation_type = ?", tagID, "abstract").
		Count(&count)
	if count == 0 {
		return fmt.Errorf("tag %d is not an abstract tag (no child relations)", tagID)
	}

	newSlug := topictypes.Slugify(newName)
	if newSlug == "" {
		return fmt.Errorf("generated empty slug for name %q", newName)
	}

	// Check slug uniqueness (exclude self)
	var conflictCount int64
	database.DB.Model(&models.TopicTag{}).
		Where("slug = ? AND id != ? AND status = ?", newSlug, tagID, "active").
		Count(&conflictCount)
	if conflictCount > 0 {
		return fmt.Errorf("slug %q already in use by another active tag", newSlug)
	}

	// Update label and slug
	if err := database.DB.Model(&models.TopicTag{}).
		Where("id = ?", tagID).
		Updates(map[string]interface{}{
			"label": newName,
			"slug":  newSlug,
		}).Error; err != nil {
		return fmt.Errorf("update abstract tag name: %w", err)
	}

	// Re-generate embedding asynchronously
	go func(tid uint) {
		es := NewEmbeddingService()
		var tag models.TopicTag
		if err := database.DB.First(&tag, tid).Error; err != nil {
			logging.Warnf("Failed to load tag %d for re-embedding: %v", tid, err)
			return
		}
		emb, err := es.GenerateEmbedding(context.Background(), &tag)
		if err != nil {
			logging.Warnf("Failed to generate embedding for renamed tag %d: %v", tid, err)
			return
		}
		emb.TopicTagID = tid
		if err := es.SaveEmbedding(emb); err != nil {
			logging.Warnf("Failed to save embedding for renamed tag %d: %v", tid, err)
		}
	}(tagID)

	return nil
}

// DetachChildTag removes a child tag from its abstract parent.
// Does not delete the parent even if it has no remaining children (per D-05 history preservation).
func DetachChildTag(parentID, childID uint) error {
	if parentID == 0 || childID == 0 {
		return fmt.Errorf("parent_id and child_id must be > 0")
	}
	if parentID == childID {
		return fmt.Errorf("parent_id and child_id must be different")
	}

	result := database.DB.Where("parent_id = ? AND child_id = ?", parentID, childID).
		Delete(&models.TopicTagRelation{})
	if result.Error != nil {
		return fmt.Errorf("detach child tag: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no relation found between parent %d and child %d", parentID, childID)
	}

	return nil
}

// ReassignTagParent moves a tag to a new abstract parent.
// If the tag currently has a parent, the old relation is removed first.
// If the tag is itself an abstract tag (has children), reassignment is blocked to prevent nesting.
func ReassignTagParent(tagID, newParentID uint) error {
	if tagID == 0 || newParentID == 0 {
		return fmt.Errorf("tag_id and new_parent_id must be > 0")
	}
	if tagID == newParentID {
		return fmt.Errorf("tag_id and new_parent_id must be different")
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Check if new parent exists
		var parent models.TopicTag
		if err := tx.First(&parent, newParentID).Error; err != nil {
			return fmt.Errorf("parent tag not found: %w", err)
		}

		// 2. Check if tag is itself an abstract tag (has children)
		var childCount int64
		tx.Model(&models.TopicTagRelation{}).Where("parent_id = ?", tagID).Count(&childCount)
		if childCount > 0 {
			return fmt.Errorf("cannot reassign an abstract tag that has children")
		}

		// 3. Check if this would create a cycle
		wouldCycle, err := wouldCreateCycle(tx, newParentID, tagID)
		if err != nil {
			return fmt.Errorf("check cycle for reassignment: %w", err)
		}
		if wouldCycle {
			return fmt.Errorf("reassigning tag %d to parent %d would create a cycle", tagID, newParentID)
		}

		// 4. Remove from current parent (if any)
		tx.Where("child_id = ?", tagID).Delete(&models.TopicTagRelation{})

		// 5. Create new relation
		relation := models.TopicTagRelation{
			ParentID:     newParentID,
			ChildID:      tagID,
			RelationType: "abstract",
		}
		if err := tx.Create(&relation).Error; err != nil {
			return fmt.Errorf("create reassignment relation: %w", err)
		}

		return nil
	})
}

// --- Internal helpers ---

// wouldCreateCycle checks if adding a parent-child relation would create a cycle.
// Returns true if adding parentID -> childID would create a cycle.
func wouldCreateCycle(tx *gorm.DB, parentID, childID uint) (bool, error) {
	// Use BFS to check if parentID is reachable from childID (which would create a cycle)
	visited := make(map[uint]bool)
	queue := []uint{childID}
	visited[childID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == parentID {
			return true, nil
		}

		// Find all parents of current node
		var relations []models.TopicTagRelation
		if err := tx.Where("child_id = ? AND relation_type = ?", current, "abstract").Find(&relations).Error; err != nil {
			return false, fmt.Errorf("query relations for cycle check: %w", err)
		}

		for _, r := range relations {
			if !visited[r.ParentID] {
				visited[r.ParentID] = true
				queue = append(queue, r.ParentID)
			}
		}
	}

	return false, nil
}

type tagJudgment struct {
	CandidateLabel string
	Action         string
	MergeTarget    string
	MergeLabel     string
	AbstractName   string
	Description    string
	Reason         string
}

const judgmentBatchSize = 8

type previousRoundResult struct {
	CandidateLabel string
	Action         string
	TargetLabel    string
}

func selectMergeTarget(candidates []TagCandidate, mergeTarget string, mergeLabel string) *models.TopicTag {
	mergeTargetSlug := topictypes.Slugify(mergeTarget)
	for _, c := range candidates {
		if c.Tag != nil && c.Tag.Slug == mergeTargetSlug {
			return c.Tag
		}
	}

	mergeLabelSlug := topictypes.Slugify(mergeLabel)
	for _, c := range candidates {
		if c.Tag != nil && c.Tag.Slug == mergeLabelSlug {
			return c.Tag
		}
	}

	for _, c := range candidates {
		if c.Tag != nil && c.Tag.Label == mergeTarget {
			return c.Tag
		}
	}

	for _, c := range candidates {
		if c.Tag != nil && c.Tag.Source != "abstract" {
			return c.Tag
		}
	}

	for _, c := range candidates {
		if c.Tag != nil {
			return c.Tag
		}
	}
	return nil
}

func callLLMForTagJudgment(ctx context.Context, candidates []TagCandidate, newLabel string, category string, narrativeContext string) ([]tagJudgment, error) {
	var allJudgments []tagJudgment
	var previousResults []previousRoundResult

	for batchStart := 0; batchStart < len(candidates); batchStart += judgmentBatchSize {
		batchEnd := batchStart + judgmentBatchSize
		if batchEnd > len(candidates) {
			batchEnd = len(candidates)
		}
		batch := candidates[batchStart:batchEnd]

		judgments, err := callLLMForTagJudgmentBatch(ctx, batch, newLabel, category, narrativeContext, previousResults)
		if err != nil {
			logging.Warnf("Tag judgment batch %d-%d failed: %v", batchStart, batchEnd, err)
			continue
		}

		for _, j := range judgments {
			allJudgments = append(allJudgments, j)
			previousResults = append(previousResults, previousRoundResult{
				CandidateLabel: j.CandidateLabel,
				Action:         j.Action,
				TargetLabel:    j.MergeTarget,
			})
		}
	}

	if len(allJudgments) == 0 {
		return nil, fmt.Errorf("all judgment batches failed")
	}

	return allJudgments, nil
}

func callLLMForTagJudgmentBatch(ctx context.Context, batch []TagCandidate, newLabel string, category string, narrativeContext string, previousResults []previousRoundResult) ([]tagJudgment, error) {
	router := airouter.NewRouter()
	prompt := buildBatchTagJudgmentPrompt(batch, newLabel, category, previousResults)

	if narrativeContext != "" {
		prompt += fmt.Sprintf("\n\nAdditional context from narrative analysis:\n%s\nUse this context to help determine if these tags belong to the same thematic thread.", narrativeContext)
	}

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy assistant. Respond only with valid JSON array."},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "array",
			Items: &airouter.JSONSchema{
				Type: "object",
				Properties: map[string]airouter.SchemaProperty{
					"candidate_label": {Type: "string", Description: "判断针对的候选标签名称"},
					"action":          {Type: "string", Description: "判断结果：merge 表示新标签与该候选是同一概念应合并，abstract 表示需要创建抽象概括标签，none 表示无关联"},
					"merge_target":    {Type: "string", Description: "action=merge 时必填：指定新标签应合并到哪个候选（填候选标签名称）"},
					"merge_label":     {Type: "string", Description: "action=merge 时必填：合并后的统一名称"},
					"abstract_name":   {Type: "string", Description: "action=abstract 时必填：抽象标签名称（1-160字）"},
					"description":     {Type: "string", Description: "action=abstract 时必填：抽象标签中文客观描述（500字以内）"},
					"reason":          {Type: "string", Description: "判断理由"},
				},
				Required: []string{"candidate_label", "action", "reason"},
			},
		},
		Temperature: func() *float64 { f := 0.3; return &f }(),
		Metadata: map[string]any{
			"operation":       "tag_judgment_batch",
			"candidate_count": len(batch),
			"new_label":       newLabel,
		},
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	return parseBatchTagJudgmentResponse(result.Content)
}

func parseBatchTagJudgmentResponse(content string) ([]tagJudgment, error) {
	content = jsonutil.SanitizeLLMJSON(content)

	var parsed []struct {
		CandidateLabel string `json:"candidate_label"`
		Action         string `json:"action"`
		MergeTarget    string `json:"merge_target"`
		MergeLabel     string `json:"merge_label"`
		AbstractName   string `json:"abstract_name"`
		Description    string `json:"description"`
		Reason         string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return parseSingleJudgmentFallback(content)
	}

	var judgments []tagJudgment
	for _, p := range parsed {
		action := strings.ToLower(strings.TrimSpace(p.Action))
		if action != ActionMerge && action != ActionAbstract && action != ActionNone {
			continue
		}

		j := tagJudgment{
			CandidateLabel: strings.TrimSpace(p.CandidateLabel),
			Action:         action,
			Reason:         p.Reason,
		}

		switch action {
		case ActionMerge:
			j.MergeTarget = strings.TrimSpace(p.MergeTarget)
			j.MergeLabel = strings.TrimSpace(p.MergeLabel)
			if j.MergeLabel == "" {
				j.MergeLabel = j.CandidateLabel
			}
		case ActionAbstract:
			j.AbstractName = strings.TrimSpace(p.AbstractName)
			if j.AbstractName == "" {
				continue
			}
			if len(j.AbstractName) > maxAbstractNameLen {
				j.AbstractName = j.AbstractName[:maxAbstractNameLen]
			}
			j.Description = strings.TrimSpace(p.Description)
			if len(j.Description) > 500 {
				j.Description = j.Description[:500]
			}
		}

		judgments = append(judgments, j)
	}

	if len(judgments) == 0 {
		return nil, fmt.Errorf("no valid judgments parsed from LLM response")
	}

	return judgments, nil
}

func parseSingleJudgmentFallback(content string) ([]tagJudgment, error) {
	var parsed struct {
		Action       string `json:"action"`
		MergeLabel   string `json:"merge_label"`
		AbstractName string `json:"abstract_name"`
		Description  string `json:"description"`
		Reason       string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as array or object: %w", err)
	}

	action := strings.ToLower(strings.TrimSpace(parsed.Action))
	if action != ActionMerge && action != ActionAbstract && action != ActionNone {
		return nil, fmt.Errorf("invalid action %q", parsed.Action)
	}

	return []tagJudgment{{
		Action:       action,
		MergeLabel:   strings.TrimSpace(parsed.MergeLabel),
		AbstractName: strings.TrimSpace(parsed.AbstractName),
		Description:  strings.TrimSpace(parsed.Description),
		Reason:       parsed.Reason,
	}}, nil
}

func buildCandidateList(candidates []TagCandidate, newLabel string) string {
	var parts []string
	for _, c := range candidates {
		if c.Tag != nil {
			tagType := "normal"
			if c.Tag.Source == "abstract" {
				tagType = "abstract"
			}
			desc := ""
			if c.Tag.Description != "" {
				runes := []rune(c.Tag.Description)
				if len(runes) > 200 {
					desc = fmt.Sprintf(" (description: %s...)", string(runes[:200]))
				} else {
					desc = fmt.Sprintf(" (description: %s)", c.Tag.Description)
				}
			}
			parts = append(parts, fmt.Sprintf("- %q (similarity: %.2f, type: %s)%s", c.Tag.Label, c.Similarity, tagType, desc))
		}
	}
	parts = append(parts, fmt.Sprintf("- %q (new tag)", newLabel))
	return strings.Join(parts, "\n")
}

func buildPreviousResultsSummary(results []previousRoundResult) string {
	if len(results) == 0 {
		return ""
	}
	var parts []string
	for _, r := range results {
		switch r.Action {
		case ActionMerge:
			parts = append(parts, fmt.Sprintf("- %q → merge (into %q)", r.CandidateLabel, r.TargetLabel))
		case ActionAbstract:
			parts = append(parts, fmt.Sprintf("- %q → abstract (%q)", r.CandidateLabel, r.TargetLabel))
		case ActionNone:
			parts = append(parts, fmt.Sprintf("- %q → none (independent)", r.CandidateLabel))
		}
	}
	return fmt.Sprintf("Previous round decisions:\n%s\n", strings.Join(parts, "\n"))
}

func buildBatchTagJudgmentPrompt(candidates []TagCandidate, newLabel string, category string, previousResults []previousRoundResult) string {
	tagList := buildCandidateList(candidates, newLabel)
	prevSummary := buildPreviousResultsSummary(previousResults)

	perCandidateRule := `
For each candidate tag, return an independent judgment:
- merge: the new tag and this candidate are the SAME concept — they should be unified
  - merge_target: fill with the candidate label that the new tag should merge into
  - merge_label: the unified name after merge
- abstract: the new tag and this candidate are DISTINCT but RELATED concepts — they need an abstract parent tag
  - abstract_name: name for the abstract parent tag (1-160 chars)
  - description: objective Chinese description (≤500 chars)
- none: the new tag has no meaningful relationship with this candidate

Important:
- If similarity >= 0.97 with a normal candidate, usually merge is correct
- If a candidate is an abstract tag, merging into one of its concrete children is often better than creating another child
- You can return different actions for different candidates
- merge_target should reference one of the candidate labels from the list above

Return a JSON array:
[
  {"candidate_label": "候选标签名", "action": "merge/abstract/none", "merge_target": "目标候选", "merge_label": "统一名称", "abstract_name": "抽象名", "description": "描述", "reason": "理由"},
  ...
]`

	switch category {
	case "person":
		return fmt.Sprintf(`以下是语义相似的人物标签:
%s
%s
请为每个候选标签独立判断与新标签 %q 的关系:
%s

判断标准:
- 只有同一人物的不同叫法才用 merge
- 不同人物只有在紧密的组织/角色关系（同团队、同家族、师徒关系）时才用 abstract
- 仅仅"同领域"、"同事件参与者"不构成 abstract 的充分理由，用 none
- 绝不要创建形如"人物A与人物B"的抽象标签——如果 abstract_name 只是列举人名，用 none`, tagList, prevSummary, newLabel, perCandidateRule)

	case "event":
		return fmt.Sprintf(`以下是语义相似的事件标签:
%s
%s
请为每个候选标签独立判断与新标签 %q 的关系:
%s

判断标准:
- 只有明确是同一事件的不同表述才用 merge
- 相似但独立的事件（如同一系列的不同事件）用 abstract
- 没有实质关联（只是语义相似度碰巧高）时用 none`, tagList, prevSummary, newLabel, perCandidateRule)

	default:
		return fmt.Sprintf(`Given these semantically similar tags:
%s
%s
Judge the relationship between the new tag %q and each candidate independently:
%s

Criteria:
- merge: same concept with different names/spellings
- abstract: distinct but related concepts sharing a common theme
- none: no meaningful relationship beyond semantic similarity
- abstract_name/merge_label should be in the original language of the tags
- description must be objective, factual — no subjective opinions`, tagList, prevSummary, newLabel, perCandidateRule)
	}
}

// resolveActiveTagIDs returns a set of tag IDs that have articles published within the given time range.
// If timeRange is empty or invalid, all tags in the candidate set are considered active.
func resolveActiveTagIDs(timeRange string, candidateIDs map[uint]bool) map[uint]bool {
	result := make(map[uint]bool, len(candidateIDs))

	if timeRange == "" {
		// No filter — all tags are active
		for id := range candidateIDs {
			result[id] = true
		}
		return result
	}

	var activeIDs []uint
	switch {
	case timeRange == "1d":
		since := time.Now().AddDate(0, 0, -1)
		database.DB.Model(&models.ArticleTopicTag{}).
			Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
			Where("articles.pub_date >= ?", since).
			Where("article_topic_tags.topic_tag_id IN ?", candidateIDSetToSlice(candidateIDs)).
			Pluck("DISTINCT article_topic_tags.topic_tag_id", &activeIDs)
	case timeRange == "7d":
		since := time.Now().AddDate(0, 0, -7)
		database.DB.Model(&models.ArticleTopicTag{}).
			Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
			Where("articles.pub_date >= ?", since).
			Where("article_topic_tags.topic_tag_id IN ?", candidateIDSetToSlice(candidateIDs)).
			Pluck("DISTINCT article_topic_tags.topic_tag_id", &activeIDs)
	case timeRange == "30d":
		since := time.Now().AddDate(0, 0, -30)
		database.DB.Model(&models.ArticleTopicTag{}).
			Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
			Where("articles.pub_date >= ?", since).
			Where("article_topic_tags.topic_tag_id IN ?", candidateIDSetToSlice(candidateIDs)).
			Pluck("DISTINCT article_topic_tags.topic_tag_id", &activeIDs)
	case strings.HasPrefix(timeRange, "custom:"):
		parts := strings.SplitN(timeRange, ":", 3)
		if len(parts) != 3 {
			for id := range candidateIDs {
				result[id] = true
			}
			return result
		}
		startDate := parts[1]
		endDate := parts[2]
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
			for id := range candidateIDs {
				result[id] = true
			}
			return result
		}
		if _, err := time.Parse("2006-01-02", endDate); err != nil {
			for id := range candidateIDs {
				result[id] = true
			}
			return result
		}
		database.DB.Model(&models.ArticleTopicTag{}).
			Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
			Where("articles.pub_date >= ? AND articles.pub_date <= ?", startDate+" 00:00:00", endDate+" 23:59:59").
			Where("article_topic_tags.topic_tag_id IN ?", candidateIDSetToSlice(candidateIDs)).
			Pluck("DISTINCT article_topic_tags.topic_tag_id", &activeIDs)
	default:
		// Invalid value — treat as no filter
		for id := range candidateIDs {
			result[id] = true
		}
		return result
	}

	for _, id := range activeIDs {
		result[id] = true
	}
	return result
}

// candidateIDSetToSlice converts a map[uint]bool to []uint
func candidateIDSetToSlice(m map[uint]bool) []uint {
	result := make([]uint, 0, len(m))
	for id := range m {
		result = append(result, id)
	}
	return result
}

// countArticlesByTag returns article counts per tag ID, optionally filtered by time range.
// Uses a single GROUP BY query instead of N+1.
func countArticlesByTag(tagIDs []uint, timeRange string) map[uint]int {
	result := make(map[uint]int)
	if len(tagIDs) == 0 {
		return result
	}

	query := database.DB.Model(&models.ArticleTopicTag{}).
		Select("topic_tag_id, count(*) as cnt").
		Where("topic_tag_id IN ?", tagIDs)

	switch {
	case timeRange == "1d":
		since := time.Now().AddDate(0, 0, -1)
		query = query.Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
			Where("articles.pub_date >= ?", since)
	case timeRange == "7d":
		since := time.Now().AddDate(0, 0, -7)
		query = query.Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
			Where("articles.pub_date >= ?", since)
	case timeRange == "30d":
		since := time.Now().AddDate(0, 0, -30)
		query = query.Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
			Where("articles.pub_date >= ?", since)
	case strings.HasPrefix(timeRange, "custom:"):
		parts := strings.SplitN(timeRange, ":", 3)
		if len(parts) == 3 {
			if _, err1 := time.Parse("2006-01-02", parts[1]); err1 == nil {
				if _, err2 := time.Parse("2006-01-02", parts[2]); err2 == nil {
					query = query.Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
						Where("articles.pub_date >= ? AND articles.pub_date <= ?", parts[1]+" 00:00:00", parts[2]+" 23:59:59")
				}
			}
		}
	}

	var rows []struct {
		TopicTagID uint `json:"topic_tag_id"`
		Cnt        int  `json:"cnt"`
	}
	query.Group("topic_tag_id").Scan(&rows)

	for _, row := range rows {
		result[row.TopicTagID] = row.Cnt
	}
	return result
}

// candidateLabels returns a comma-separated list of candidate labels for logging.
func candidateLabels(candidates []TagCandidate) string {
	labels := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if c.Tag != nil {
			labels = append(labels, c.Tag.Label)
		}
	}
	return strings.Join(labels, ", ")
}

// resolveScopeTagIDs returns the set of topic tag IDs that are associated with articles
// matching the given feed_id or category_id scope.
func resolveScopeTagIDs(feedID uint, categoryID uint) (map[uint]bool, error) {
	if feedID == 0 && categoryID == 0 {
		return nil, nil
	}

	query := database.DB.Model(&models.ArticleTopicTag{}).
		Select("DISTINCT topic_tag_id")

	if feedID > 0 {
		query = query.Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
			Where("articles.feed_id = ?", feedID)
	} else if categoryID > 0 {
		query = query.Joins("JOIN articles ON articles.id = article_topic_tags.article_id").
			Joins("JOIN feeds ON feeds.id = articles.feed_id").
			Where("feeds.category_id = ?", categoryID)
	}

	var tagIDs []uint
	if err := query.Pluck("topic_tag_id", &tagIDs).Error; err != nil {
		return nil, fmt.Errorf("resolve scope tag IDs: %w", err)
	}

	result := make(map[uint]bool, len(tagIDs))
	for _, id := range tagIDs {
		result[id] = true
	}
	return result, nil
}

// GetUnclassifiedTags returns tags that are NOT part of any abstract hierarchy.
// These are active tags that do not appear as parent or child in any abstract relation.
func GetUnclassifiedTags(category string, scopeFeedID uint, scopeCategoryID uint, timeRange string) ([]TagHierarchyNode, error) {
	// Collect all tag IDs involved in abstract relations
	var relatedTagIDs []uint
	database.DB.Model(&models.TopicTagRelation{}).
		Where("relation_type = ?", "abstract").
		Pluck("parent_id", &relatedTagIDs)
	var childIDs []uint
	database.DB.Model(&models.TopicTagRelation{}).
		Where("relation_type = ?", "abstract").
		Pluck("child_id", &childIDs)
	relatedTagIDs = append(relatedTagIDs, childIDs...)
	relatedSet := make(map[uint]bool, len(relatedTagIDs))
	for _, id := range relatedTagIDs {
		relatedSet[id] = true
	}

	query := database.DB.Model(&models.TopicTag{}).
		Where("status = ? AND source != ?", "active", "abstract").
		Where("id IN (SELECT DISTINCT topic_tag_id FROM article_topic_tags)")

	if len(relatedSet) > 0 {
		allRelated := make([]uint, 0, len(relatedSet))
		for id := range relatedSet {
			allRelated = append(allRelated, id)
		}
		query = query.Where("id NOT IN ?", allRelated)
	}

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if scopeFeedID > 0 || scopeCategoryID > 0 {
		scopeTagIDs, err := resolveScopeTagIDs(scopeFeedID, scopeCategoryID)
		if err != nil {
			return nil, err
		}
		if len(scopeTagIDs) == 0 {
			return []TagHierarchyNode{}, nil
		}
		scopeSlice := make([]uint, 0, len(scopeTagIDs))
		for id := range scopeTagIDs {
			scopeSlice = append(scopeSlice, id)
		}
		query = query.Where("id IN ?", scopeSlice)
	}

	var tags []models.TopicTag
	if err := query.Order("quality_score DESC, feed_count DESC, label ASC").Limit(200).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("query unclassified tags: %w", err)
	}

	// Resolve active status based on time range
	tagIDSet := make(map[uint]bool, len(tags))
	for _, tag := range tags {
		tagIDSet[tag.ID] = true
	}
	activeTagIDs := resolveActiveTagIDs(timeRange, tagIDSet)

	if timeRange != "" {
		filteredTags := make([]models.TopicTag, 0, len(tags))
		for _, tag := range tags {
			if activeTagIDs[tag.ID] {
				filteredTags = append(filteredTags, tag)
			}
		}
		tags = filteredTags
	}

	nodes := make([]TagHierarchyNode, 0, len(tags))
	for _, tag := range tags {
		nodes = append(nodes, TagHierarchyNode{
			ID:           tag.ID,
			Label:        tag.Label,
			Slug:         tag.Slug,
			Category:     tag.Category,
			Icon:         tag.Icon,
			FeedCount:    tag.FeedCount,
			IsActive:     true,
			QualityScore: tag.QualityScore,
			IsLowQuality: tag.Source != "abstract" && tag.QualityScore < 0.3,
			Children:     []TagHierarchyNode{},
		})
	}
	return nodes, nil
}

// MatchAbstractTagHierarchy searches for similar abstract tags after a new abstract tag is created,
// and establishes parent-child relationships for multi-level hierarchy.
func MatchAbstractTagHierarchy(ctx context.Context, abstractTagID uint) {
	defer func() {
		if r := recover(); r != nil {
			logging.Warnf("MatchAbstractTagHierarchy panic for tag %d: %v", abstractTagID, r)
		}
	}()

	var abstractTag models.TopicTag
	if err := database.DB.First(&abstractTag, abstractTagID).Error; err != nil {
		logging.Warnf("MatchAbstractTagHierarchy: tag %d not found: %v", abstractTagID, err)
		return
	}

	es := NewEmbeddingService()
	candidates, err := es.FindSimilarAbstractTags(ctx, abstractTagID, abstractTag.Category, 5)
	if err != nil {
		logging.Warnf("MatchAbstractTagHierarchy: failed to find similar abstract tags for %d: %v", abstractTagID, err)
		return
	}
	if len(candidates) == 0 {
		return
	}

	best := candidates[0]
	thresholds := es.GetThresholds()

	if best.Similarity >= thresholds.HighSimilarity {
		if err := linkAbstractParentChild(abstractTagID, best.Tag.ID); err != nil {
			logging.Warnf("MatchAbstractTagHierarchy: failed to link %d under %d: %v", abstractTagID, best.Tag.ID, err)
			return
		}
		logging.Infof("Abstract tag %d linked under existing abstract %d (similarity=%.4f)", abstractTagID, best.Tag.ID, best.Similarity)
		return
	}

	if best.Similarity >= thresholds.LowSimilarity {
		parentID, childID, err := aiJudgeAbstractHierarchy(ctx, abstractTagID, best.Tag.ID)
		if err != nil {
			logging.Warnf("MatchAbstractTagHierarchy: AI judgment failed for %d vs %d: %v", abstractTagID, best.Tag.ID, err)
			return
		}
		if err := linkAbstractParentChild(childID, parentID); err != nil {
			logging.Warnf("MatchAbstractTagHierarchy: failed to link %d under %d: %v", childID, parentID, err)
			return
		}
		logging.Infof("Abstract hierarchy: %d is child of %d (AI judged, similarity=%.4f)", childID, parentID, best.Similarity)
	}
}

// linkAbstractParentChild creates a parent-child relation between two abstract tags.
// Prevents a child from being adopted by multiple parents (one-parent rule).
func linkAbstractParentChild(childID, parentID uint) error {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		wouldCycle, err := wouldCreateCycle(tx, parentID, childID)
		if err != nil {
			return fmt.Errorf("cycle check: %w", err)
		}
		if wouldCycle {
			return fmt.Errorf("would create cycle: parent=%d, child=%d", parentID, childID)
		}

		var count int64
		tx.Model(&models.TopicTagRelation{}).
			Where("parent_id = ? AND child_id = ?", parentID, childID).
			Count(&count)
		if count > 0 {
			return nil
		}

		var existingParentCount int64
		tx.Model(&models.TopicTagRelation{}).
			Where("child_id = ? AND parent_id != ?", childID, parentID).
			Count(&existingParentCount)
		if existingParentCount > 0 {
			return fmt.Errorf("child %d already has an abstract parent, skipping", childID)
		}

		relation := models.TopicTagRelation{
			ParentID:     parentID,
			ChildID:      childID,
			RelationType: "abstract",
		}
		return tx.Create(&relation).Error
	})
	if err != nil {
		return err
	}

	go enqueueEmbeddingsForNormalChildren(parentID)
	go EnqueueAbstractTagUpdate(parentID, "hierarchy_linked")

	return nil
}

// aiJudgeAbstractHierarchy uses LLM to determine which abstract tag is broader (parent) vs more specific (child).
func aiJudgeAbstractHierarchy(ctx context.Context, tag1ID, tag2ID uint) (parentID, childID uint, err error) {
	var tag1, tag2 models.TopicTag
	if err := database.DB.First(&tag1, tag1ID).Error; err != nil {
		return 0, 0, fmt.Errorf("load tag %d: %w", tag1ID, err)
	}
	if err := database.DB.First(&tag2, tag2ID).Error; err != nil {
		return 0, 0, fmt.Errorf("load tag %d: %w", tag2ID, err)
	}

	router := airouter.NewRouter()
	prompt := fmt.Sprintf(`Given two abstract topic tags, determine which concept is broader (more general) and which is more specific.

Tag A: %q (description: %s)
Tag B: %q (description: %s)

Respond with JSON:
{"parent": "A" or "B", "reason": "brief explanation"}

Rules:
- The parent should be the more general/broader concept
- If they are equally broad, choose the one with a shorter/more concise label as parent
- If unclear, default to "A" as parent`, tag1.Label, truncateStr(tag1.Description, 200), tag2.Label, truncateStr(tag2.Description, 200))

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"parent": {Type: "string", Description: "更宽泛的标签标识，A 或 B"},
				"reason": {Type: "string", Description: "判断理由"},
			},
			Required: []string{"parent", "reason"},
		},
		Temperature: func() *float64 { f := 0.3; return &f }(),
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return 0, 0, fmt.Errorf("LLM call failed: %w", err)
	}

	var parsed struct {
		Parent string `json:"parent"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
		return 0, 0, fmt.Errorf("parse LLM response: %w", err)
	}

	if strings.ToUpper(parsed.Parent) == "B" {
		return tag2ID, tag1ID, nil
	}
	return tag1ID, tag2ID, nil
}

// truncateStr truncates a string to maxLen characters.
func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

func enqueueEmbeddingsForNormalChildren(parentID uint) {
	defer func() {
		if r := recover(); r != nil {
			logging.Warnf("enqueueEmbeddingsForNormalChildren panic: %v", r)
		}
	}()

	var normalChildIDs []uint
	database.DB.Model(&models.TopicTag{}).
		Joins("JOIN topic_tag_relations ON topic_tag_relations.child_id = topic_tags.id").
		Where("topic_tag_relations.parent_id = ? AND topic_tag_relations.relation_type = ? AND topic_tags.source != ?",
			parentID, "abstract", "abstract").
		Pluck("topic_tags.id", &normalChildIDs)

	if len(normalChildIDs) == 0 {
		return
	}

	var existingEmbTagIDs []uint
	database.DB.Model(&models.TopicTagEmbedding{}).
		Where("topic_tag_id IN ?", normalChildIDs).
		Pluck("topic_tag_id", &existingEmbTagIDs)
	existingSet := make(map[uint]bool, len(existingEmbTagIDs))
	for _, id := range existingEmbTagIDs {
		existingSet[id] = true
	}

	qs := NewEmbeddingQueueService(nil)
	for _, id := range normalChildIDs {
		if !existingSet[id] {
			if err := qs.Enqueue(id); err != nil {
				logging.Warnf("Failed to enqueue embedding for normal child %d under abstract %d: %v", id, parentID, err)
			}
		}
	}
}
