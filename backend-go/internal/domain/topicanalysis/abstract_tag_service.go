package topicanalysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topictypes"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/jsonutil"
	"my-robot-backend/internal/platform/logging"
	"my-robot-backend/internal/platform/ws"

	"gorm.io/gorm"
)

const (
	maxAbstractNameLen = 160
)

var errInsufficientAbstractChildren = errors.New("abstract tag needs enough children")

var (
	findSimilarExistingAbstractFn = findSimilarExistingAbstract
	aiJudgeNarrowerConceptFn      = aiJudgeNarrowerConcept
)

// TagExtractionResult is the return type for ExtractAbstractTag.
// Merge and Abstract can both be set simultaneously.
type TagExtractionResult struct {
	Merge         *MergeResult       // if set: the new tag should merge into this target
	Abstract      *AbstractResult    // if set: create/return abstract parent with children
	MergeChildren []*models.TopicTag // additional candidates that should also merge into Merge.Target
}

type MergeResult struct {
	Target *models.TopicTag
	Label  string
}

type AbstractResult struct {
	Tag      *models.TopicTag
	Children []*models.TopicTag // candidates linked as children of this abstract tag
}

func (r *TagExtractionResult) HasMerge() bool    { return r != nil && r.Merge != nil }
func (r *TagExtractionResult) HasAbstract() bool { return r != nil && r.Abstract != nil }
func (r *TagExtractionResult) HasAction() bool   { return r.HasMerge() || r.HasAbstract() }

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
	caller           string
}

func WithNarrativeContext(ctx string) ExtractAbstractTagOption {
	return func(c *extractAbstractTagConfig) {
		c.narrativeContext = ctx
	}
}

func WithCaller(caller string) ExtractAbstractTagOption {
	return func(c *extractAbstractTagConfig) {
		c.caller = caller
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

	judgment, err := callLLMForTagJudgment(ctx, candidates, newLabel, category, cfg.narrativeContext, cfg.caller)
	if err != nil {
		logging.Warnf("Tag judgment LLM call failed: %v", err)
		return nil, err
	}

	return processJudgment(ctx, judgment, candidates, newLabel, category)
}

func processJudgment(ctx context.Context, judgment *tagJudgment, candidates []TagCandidate, newLabel string, category string) (*TagExtractionResult, error) {
	result := &TagExtractionResult{}

	if judgment.Merge != nil {
		mergeTarget := selectMergeTarget(candidates, judgment.Merge.Target, judgment.Merge.Label)
		if mergeTarget == nil {
			return nil, fmt.Errorf("no suitable merge target found for label %q (target=%q)", judgment.Merge.Label, judgment.Merge.Target)
		}
		logging.Infof("Tag judgment: merge into existing tag %q (id=%d), label=%q", mergeTarget.Label, mergeTarget.ID, judgment.Merge.Label)

		result.Merge = &MergeResult{
			Target: mergeTarget,
			Label:  judgment.Merge.Label,
		}

		for _, childLabel := range judgment.Merge.Children {
			for _, c := range candidates {
				if c.Tag != nil && c.Tag.Label == childLabel && c.Tag.ID != mergeTarget.ID {
					result.MergeChildren = append(result.MergeChildren, c.Tag)
				}
			}
		}
	}

	if judgment.Abstract != nil {
		ensureNewLabelCandidateInAbstractJudgment(judgment, candidates, newLabel)
		abstractResult, err := processAbstractJudgment(ctx, candidates, judgment.Abstract, newLabel, category)
		if err != nil {
			if result.HasMerge() {
				logging.Warnf("Abstract judgment failed but merge succeeded, returning merge only: %v", err)
				return result, nil
			}
			return nil, err
		}
		if abstractResult != nil {
			result.Abstract = abstractResult
		}
	}

	if !result.HasAction() {
		logging.Infof("Tag judgment: all candidates independent for %q", newLabel)
	}

	return result, nil
}

func processAbstractJudgment(ctx context.Context, candidates []TagCandidate, judgment *tagJudgmentAbstract, newLabel string, category string) (*AbstractResult, error) {
	abstractName := judgment.Name
	abstractDesc := judgment.Description
	newLabelIsCandidate := candidateLabelForNewLabel(candidates, newLabel) != ""

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
		logging.Infof("Abstract name %q (slug=%s) collides with a candidate tag, skipping abstract creation", abstractName, slug)
		return nil, nil
	}

	var abstractTag *models.TopicTag
	if existingAbstract := findSimilarExistingAbstractFn(ctx, abstractName, abstractDesc, category, candidates); existingAbstract != nil {
		logging.Infof("processAbstractJudgment: reusing existing abstract tag %d (%q) instead of creating new %q",
			existingAbstract.ID, existingAbstract.Label, abstractName)
		abstractTag = existingAbstract
	}

	abstractChildSet := make(map[string]bool, len(judgment.Children))
	for _, ch := range judgment.Children {
		abstractChildSet[ch] = true
	}

	var createdNewAbstract bool
	var abstractChildren []*models.TopicTag

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if abstractTag == nil {
			var existing models.TopicTag
			if err := tx.Where("slug = ? AND category = ? AND status = ?", slug, category, "active").First(&existing).Error; err == nil {
				abstractTag = &existing
			}
		}

		if abstractTag == nil {
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
			createdNewAbstract = true

			go func(tagID uint, name, cat string) {
				es := NewEmbeddingService()
				tag := &models.TopicTag{ID: tagID, Label: name, Category: cat}
				for _, embType := range []string{EmbeddingTypeIdentity, EmbeddingTypeSemantic} {
					emb, genErr := es.GenerateEmbedding(context.Background(), tag, embType)
					if genErr != nil {
						logging.Warnf("Failed to generate %s embedding for abstract tag %d: %v", embType, tagID, genErr)
						continue
					}
					emb.TopicTagID = tagID
					if saveErr := es.SaveEmbedding(emb); saveErr != nil {
						logging.Warnf("Failed to save %s embedding for abstract tag %d: %v", embType, tagID, saveErr)
					}
				}
				MatchAbstractTagHierarchy(context.Background(), tagID)
				adoptNarrowerAbstractChildren(context.Background(), tagID)
			}(abstractTag.ID, abstractName, category)
		}

		for _, candidate := range candidates {
			if candidate.Tag == nil {
				continue
			}
			if candidate.Tag.ID == abstractTag.ID {
				continue
			}
			if !abstractChildSet[candidate.Tag.Label] {
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
				abstractChildren = append(abstractChildren, candidate.Tag)
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
			abstractChildren = append(abstractChildren, candidate.Tag)
		}

		minChildren := 1
		if newLabelIsCandidate {
			minChildren = 2
		}
		if len(abstractChildren) < minChildren {
			return errInsufficientAbstractChildren
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, errInsufficientAbstractChildren) {
			logging.Infof("Skipping abstract tag %q: only %d child relation(s) could be linked", abstractName, len(abstractChildren))
			return nil, nil
		}
		logging.Warnf("Abstract tag transaction failed: %v", err)
		return nil, err
	}

	logging.Infof("Abstract tag extracted: %s (id=%d) with children [%s]",
		abstractTag.Label, abstractTag.ID, strings.Join(judgment.Children, ", "))

	if len(abstractChildren) > 0 {
		if !createdNewAbstract && abstractTag.Source == "abstract" {
			go adoptNarrowerAbstractChildren(context.Background(), abstractTag.ID)
		}
		go EnqueueAbstractTagUpdate(abstractTag.ID, "new_child_added")
	}

	return &AbstractResult{
		Tag:      abstractTag,
		Children: abstractChildren,
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
		propagateActiveAncestors(activeTagIDs, relations)
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
			IsActive:        activeTagIDs[child.ID],
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
			IsActive:     activeTagIDs[parent.ID],
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
		for _, embType := range []string{EmbeddingTypeIdentity, EmbeddingTypeSemantic} {
			emb, err := es.GenerateEmbedding(context.Background(), &tag, embType)
			if err != nil {
				logging.Warnf("Failed to generate %s embedding for renamed tag %d: %v", embType, tid, err)
				continue
			}
			emb.TopicTagID = tid
			if err := es.SaveEmbedding(emb); err != nil {
				logging.Warnf("Failed to save %s embedding for renamed tag %d: %v", embType, tid, err)
			}
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

func loadAbstractChildLabels(tagID uint, limit int) []string {
	var labels []string
	database.DB.Model(&models.TopicTag{}).
		Joins("JOIN topic_tag_relations ON topic_tag_relations.child_id = topic_tags.id").
		Where("topic_tag_relations.parent_id = ? AND topic_tag_relations.relation_type = ?", tagID, "abstract").
		Order("topic_tag_relations.similarity_score DESC").
		Limit(limit).
		Pluck("topic_tags.label", &labels)
	if labels == nil {
		labels = []string{}
	}
	return labels
}

func formatChildLabels(labels []string) string {
	if len(labels) == 0 {
		return "(无子标签)"
	}
	return strings.Join(labels, ", ")
}

func findSimilarExistingAbstract(ctx context.Context, name, desc, category string, candidates []TagCandidate) *models.TopicTag {
	es := NewEmbeddingService()
	thresholds := es.GetThresholds()
	probe := &models.TopicTag{
		Label:       name,
		Description: desc,
		Category:    category,
		Source:      "abstract",
	}
	similar, err := es.FindSimilarTags(ctx, probe, category, 8, EmbeddingTypeSemantic)
	if err != nil {
		logging.Warnf("findSimilarExistingAbstract: embedding search failed: %v", err)
		return nil
	}

	existingAbstracts := make([]models.TopicTag, 0, len(similar))
	for _, candidate := range similar {
		if candidate.Tag == nil || candidate.Tag.Source != "abstract" {
			continue
		}
		if candidate.Similarity < thresholds.LowSimilarity {
			continue
		}
		existingAbstracts = append(existingAbstracts, *candidate.Tag)
		if len(existingAbstracts) == 5 {
			break
		}
	}
	if len(existingAbstracts) == 0 {
		return nil
	}

	candidateLabels := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Tag != nil {
			candidateLabels = append(candidateLabels, candidate.Tag.Label)
		}
	}

	router := airouter.NewRouter()
	abstractInfo := make([]string, 0, len(existingAbstracts))
	for _, existing := range existingAbstracts {
		children := loadAbstractChildLabels(existing.ID, 5)
		abstractInfo = append(abstractInfo,
			fmt.Sprintf("- ID %d: %q (描述: %s, 子标签: %s)", existing.ID, existing.Label, truncateStr(existing.Description, 100), formatChildLabels(children)))
	}

	prompt := fmt.Sprintf(`一个新的抽象标签即将被创建，请检查以下已有抽象标签中是否有描述同一概念的。

即将创建的抽象标签: %q (描述: %s)
其候选子标签: %s

已有同 category 抽象标签:
%s

规则:
- 只有当已有标签描述的核心概念与新标签完全相同时才返回
- 不要把宽泛程度不同的标签当作同一概念

返回 JSON: {"reuse_id": 0 或 已有标签的ID, "reason": "简要说明"}`,
		name, truncateStr(desc, 200), formatChildLabels(candidateLabels), strings.Join(abstractInfo, "\n"))

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
				"reuse_id": {Type: "integer", Description: "复用的已有标签ID，0表示不匹配"},
				"reason":   {Type: "string", Description: "判断理由"},
			},
			Required: []string{"reuse_id", "reason"},
		},
		Temperature: func() *float64 { f := 0.2; return &f }(),
		Metadata: map[string]any{
			"operation":    "find_similar_existing_abstract",
			"new_name":     name,
			"category":     category,
			"candidates_n": len(candidates),
			"shortlist_n":  len(existingAbstracts),
		},
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		logging.Warnf("findSimilarExistingAbstract: LLM call failed: %v", err)
		return nil
	}

	var parsed struct {
		ReuseID uint   `json:"reuse_id"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
		logging.Warnf("findSimilarExistingAbstract: parse failed: %v", err)
		return nil
	}
	if parsed.ReuseID == 0 {
		return nil
	}

	for i := range existingAbstracts {
		if existingAbstracts[i].ID == parsed.ReuseID {
			logging.Infof("findSimilarExistingAbstract: found match %d (%q) for new %q: %s",
				existingAbstracts[i].ID, existingAbstracts[i].Label, name, parsed.Reason)
			return &existingAbstracts[i]
		}
	}

	logging.Warnf("findSimilarExistingAbstract: reuse_id %d not found in shortlist", parsed.ReuseID)
	return nil
}

func aiJudgeNarrowerConcept(ctx context.Context, parentTag *models.TopicTag, candidateTag *models.TopicTag) (bool, error) {
	parentChildren := loadAbstractChildLabels(parentTag.ID, 5)
	candidateChildren := loadAbstractChildLabels(candidateTag.ID, 5)

	router := airouter.NewRouter()
	prompt := fmt.Sprintf(`判断候选标签是否是目标标签的更窄（更具体）概念，应该作为其子标签。

目标标签（潜在的父标签）: %q (描述: %s)
目标标签的子标签: %s

候选标签（潜在的子标签）: %q (描述: %s)
候选标签的子标签: %s

规则:
- 如果候选标签描述的是目标标签范围内的一个具体方面、子集或特定场景，则它是更窄概念
- 如果两者是同一层级或候选更宽泛，返回 false
- 如果候选的子标签与目标的子标签高度重叠，说明是同一概念，返回 false

返回 JSON: {"narrower": true/false, "reason": "简要说明"}`,
		parentTag.Label, truncateStr(parentTag.Description, 200), formatChildLabels(parentChildren),
		candidateTag.Label, truncateStr(candidateTag.Description, 200), formatChildLabels(candidateChildren))

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
				"narrower": {Type: "boolean", Description: "候选标签是否是目标标签的更窄概念"},
				"reason":   {Type: "string", Description: "判断理由"},
			},
			Required: []string{"narrower", "reason"},
		},
		Temperature: func() *float64 { f := 0.2; return &f }(),
		Metadata: map[string]any{
			"operation":     "adopt_narrower_abstract",
			"parent_tag":    parentTag.ID,
			"candidate_tag": candidateTag.ID,
		},
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return false, fmt.Errorf("LLM call failed: %w", err)
	}

	var parsed struct {
		Narrower bool   `json:"narrower"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
		return false, fmt.Errorf("parse LLM response: %w", err)
	}

	return parsed.Narrower, nil
}

func reparentOrLinkAbstractChild(ctx context.Context, childID, newParentID uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		wouldCycle, err := wouldCreateCycle(tx, newParentID, childID)
		if err != nil {
			return fmt.Errorf("cycle check: %w", err)
		}
		if wouldCycle {
			return fmt.Errorf("would create cycle: parent=%d, child=%d", newParentID, childID)
		}

		var count int64
		tx.Model(&models.TopicTagRelation{}).
			Where("parent_id = ? AND child_id = ? AND relation_type = ?", newParentID, childID, "abstract").
			Count(&count)
		if count > 0 {
			return nil
		}

		var oldParentRelation models.TopicTagRelation
		if err := tx.Where("child_id = ? AND relation_type = ?", childID, "abstract").First(&oldParentRelation).Error; err == nil {
			var oldParentTag models.TopicTag
			if err := tx.First(&oldParentTag, oldParentRelation.ParentID).Error; err != nil {
				return fmt.Errorf("load old parent tag: %w", err)
			}

			var newParentTag models.TopicTag
			if err := tx.First(&newParentTag, newParentID).Error; err != nil {
				return fmt.Errorf("load new parent tag: %w", err)
			}

			narrower, err := aiJudgeNarrowerConceptFn(ctx, &newParentTag, &oldParentTag)
			if err != nil {
				return fmt.Errorf("judge old parent narrower: %w", err)
			}
			if !narrower {
				return fmt.Errorf("child %d already has parent %d which is not narrower than %d", childID, oldParentRelation.ParentID, newParentID)
			}

			oldParentCycle, err := wouldCreateCycle(tx, newParentID, oldParentRelation.ParentID)
			if err != nil {
				return fmt.Errorf("cycle check for old parent: %w", err)
			}
			if oldParentCycle {
				return fmt.Errorf("would create cycle via old parent: parent=%d, old_parent=%d", newParentID, oldParentRelation.ParentID)
			}

			relation := models.TopicTagRelation{
				ParentID:     newParentID,
				ChildID:      oldParentRelation.ParentID,
				RelationType: "abstract",
			}
			return tx.Where("parent_id = ? AND child_id = ? AND relation_type = ?", newParentID, oldParentRelation.ParentID, "abstract").FirstOrCreate(&relation).Error
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("find old parent: %w", err)
		}

		relation := models.TopicTagRelation{
			ParentID:     newParentID,
			ChildID:      childID,
			RelationType: "abstract",
		}
		return tx.Create(&relation).Error
	})
}

func callLLMForTagJudgment(ctx context.Context, candidates []TagCandidate, newLabel string, category string, narrativeContext string, caller string) (*tagJudgment, error) {
	router := airouter.NewRouter()
	prompt := buildTagJudgmentPrompt(candidates, newLabel, category, narrativeContext)

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
				"merge": {Type: "object", Description: "合并判断：新标签与哪些候选是同一概念。omit if not applicable", Properties: map[string]airouter.SchemaProperty{
					"target":   {Type: "string", Description: "合并目标候选标签名称"},
					"label":    {Type: "string", Description: "合并后的统一名称"},
					"children": {Type: "array", Description: "与 target 同概念的其他候选标签名称列表", Items: &airouter.SchemaProperty{Type: "string"}},
					"reason":   {Type: "string", Description: "判断理由"},
				}, Required: []string{"target", "label", "children"}},
				"abstract": {Type: "object", Description: "抽象判断：需要为哪些候选创建抽象父标签。omit if not applicable", Properties: map[string]airouter.SchemaProperty{
					"name":        {Type: "string", Description: "抽象标签名称（1-160字）"},
					"description": {Type: "string", Description: "抽象标签中文客观描述（500字以内）"},
					"children":    {Type: "array", Description: "应作为该抽象标签子标签的候选名称列表", Items: &airouter.SchemaProperty{Type: "string"}},
					"reason":      {Type: "string", Description: "判断理由"},
				}, Required: []string{"name", "description", "children"}},
			},
		},
		Temperature: func() *float64 { f := 0.3; return &f }(),
		Metadata: map[string]any{
			"operation":       "tag_judgment",
			"caller":          caller,
			"candidate_count": len(candidates),
			"new_label":       newLabel,
			"category":        category,
			"candidates":      buildCandidateSummary(candidates),
		},
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	logging.Infof("Tag judgment LLM response for %q: %s", newLabel, result.Content)

	return parseTagJudgmentResponse(result.Content, candidates)
}

type tagJudgment struct {
	Merge    *tagJudgmentMerge    `json:"merge,omitempty"`
	Abstract *tagJudgmentAbstract `json:"abstract,omitempty"`
}

type tagJudgmentMerge struct {
	Target   string   `json:"target"`
	Label    string   `json:"label"`
	Children []string `json:"children"`
	Reason   string   `json:"reason"`
}

type tagJudgmentAbstract struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Children    []string `json:"children"`
	Reason      string   `json:"reason"`
}

func ensureNewLabelCandidateInAbstractJudgment(judgment *tagJudgment, candidates []TagCandidate, newLabel string) {
	if judgment == nil || judgment.Abstract == nil || len(judgment.Abstract.Children) == 0 {
		return
	}

	label := candidateLabelForNewLabel(candidates, newLabel)
	if label == "" || labelInSlice(judgment.Abstract.Children, label) {
		return
	}

	if judgment.Merge != nil {
		if judgment.Merge.Target == label || labelInSlice(judgment.Merge.Children, label) {
			return
		}
	}

	judgment.Abstract.Children = append(judgment.Abstract.Children, label)
}

func candidateLabelForNewLabel(candidates []TagCandidate, newLabel string) string {
	newSlug := topictypes.Slugify(newLabel)
	if newSlug == "" {
		return ""
	}
	for _, c := range candidates {
		if c.Tag == nil {
			continue
		}
		if c.Tag.Slug == newSlug || topictypes.Slugify(c.Tag.Label) == newSlug {
			return c.Tag.Label
		}
	}
	return ""
}

func labelInSlice(labels []string, target string) bool {
	for _, label := range labels {
		if label == target {
			return true
		}
	}
	return false
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

func parseTagJudgmentResponse(content string, candidates []TagCandidate) (*tagJudgment, error) {
	content = jsonutil.SanitizeLLMJSON(content)

	var raw struct {
		Merge *struct {
			Target   string   `json:"target"`
			Label    string   `json:"label"`
			Children []string `json:"children"`
			Reason   string   `json:"reason"`
		} `json:"merge"`
		Abstract *struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Children    []string `json:"children"`
			Reason      string   `json:"reason"`
		} `json:"abstract"`
	}

	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse tag judgment response: %w", err)
	}

	candidateLabels := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		if c.Tag != nil {
			candidateLabels[c.Tag.Label] = true
		}
	}

	filterChildren := func(children []string) []string {
		var valid []string
		for _, ch := range children {
			ch = strings.TrimSpace(ch)
			if candidateLabels[ch] {
				valid = append(valid, ch)
			}
		}
		return valid
	}

	judgment := &tagJudgment{}
	usedLabels := make(map[string]bool)

	if raw.Merge != nil && raw.Merge.Target != "" {
		judgment.Merge = &tagJudgmentMerge{
			Target:   strings.TrimSpace(raw.Merge.Target),
			Label:    strings.TrimSpace(raw.Merge.Label),
			Children: filterChildren(raw.Merge.Children),
			Reason:   raw.Merge.Reason,
		}
		if judgment.Merge.Label == "" {
			judgment.Merge.Label = judgment.Merge.Target
		}
		usedLabels[judgment.Merge.Target] = true
		for _, ch := range judgment.Merge.Children {
			usedLabels[ch] = true
		}
	}

	if raw.Abstract != nil && raw.Abstract.Name != "" {
		abstractName := strings.TrimSpace(raw.Abstract.Name)
		if len(abstractName) > maxAbstractNameLen {
			abstractName = abstractName[:maxAbstractNameLen]
		}
		desc := strings.TrimSpace(raw.Abstract.Description)
		if len(desc) > 500 {
			desc = desc[:500]
		}
		var dedupedChildren []string
		for _, ch := range filterChildren(raw.Abstract.Children) {
			if !usedLabels[ch] {
				dedupedChildren = append(dedupedChildren, ch)
				usedLabels[ch] = true
			}
		}
		judgment.Abstract = &tagJudgmentAbstract{
			Name:        abstractName,
			Description: desc,
			Children:    dedupedChildren,
			Reason:      raw.Abstract.Reason,
		}
	}

	return judgment, nil
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

func buildTagJudgmentPrompt(candidates []TagCandidate, newLabel string, category string, narrativeContext string) string {
	tagList := buildCandidateList(candidates, newLabel)

	rules := `
Analyze ALL candidates together and return ONE JSON object with at most two sections:

{
  "merge": {
    "target": "the candidate label the new tag should merge into",
    "label": "unified name after merge",
    "children": ["other candidate labels that are also the SAME concept as the target"],
    "reason": "..."
  },
  "abstract": {
    "name": "abstract parent tag name (1-160 chars)",
    "description": "objective Chinese description (≤500 chars)",
    "children": ["candidate labels that are RELATED but DISTINCT — should be grouped under this abstract tag"],
    "reason": "..."
  }
}

Rules:
- "merge" and "abstract" are both optional — omit a section entirely if not applicable
- merge: the new tag and listed children are ALL the SAME concept as the target — they should be unified
  - "target" must be one of the candidate labels
  - "children" lists other candidates that are also the same concept (NOT including the target itself)
  - If similarity >= 0.97 with a normal candidate, merge is usually correct
  - If a candidate is an abstract tag, merging into one of its concrete children is often better
- abstract: PREFER creating abstract tags when 2+ candidates share a domain, theme, or category
  - Be GENEROUS: candidates that share any thematic connection should be grouped under an abstract tag
  - Good abstracts: "AI Models" for GPT-4+Claude+Gemini, "Space Industry" for SpaceX+Starlink, "Central Banks" for Fed+ECB+BOJ, "US Tech CEOs" for Musk+Altman+Nadella
  - Even moderate connections (same industry, same region, same event type) warrant abstract tags
  - abstract_name should be a concise, specific category name — NOT just listing the children's names
  - Do NOT include any candidate already listed in merge.children here
- A candidate must appear in at most ONE section (merge.children or abstract.children), never both
- If truly no relationship exists for any candidate, return {"merge": null, "abstract": null}
- When in doubt, prefer creating an abstract tag over leaving candidates ungrouped`

	categoryRules := ""
	switch category {
	case "person":
		categoryRules = `
Person-specific rules:
- merge: ONLY when different names for the SAME person (e.g. "Tim Cook" and "蒂姆·库克")
- abstract: CREATE abstract tags for people who share affiliations, roles, domains, or national relevance
  - Examples: "中国科技领袖" for 雷军+余承东, "AI公司CEO" for Sam Altman+Satya Nadella, "美国政治人物" for Trump+Biden
  - People in the same industry, organization, or country's public sphere are related enough for abstract
  - Do NOT create overly vague tags like "人物" — add thematic specificity (e.g. "中国科技领袖" not "人物")`
	case "event":
		categoryRules = `
Event-specific rules:
- merge: ONLY when clearly the SAME event with different descriptions
- abstract: CREATE abstract tags for related events
  - Examples: "2024年大选" for multiple election events, "科技发布会" for WWDC+Google I/O, "中东冲突" for related military events
  - Events sharing a theme, industry, region, or time period should be grouped under an abstract tag
  - Same event type across different instances (e.g. quarterly earnings, annual conferences) should be grouped`
	default:
		categoryRules = `
- abstract_name/merge_label should be in the original language of the tags
- description must be objective, factual — no subjective opinions
- When in doubt, prefer creating an abstract tag over leaving candidates ungrouped`
	}

	prompt := fmt.Sprintf(`Candidates:
%s

New tag: %q (category: %s)
%s
%s`, tagList, newLabel, category, rules, categoryRules)

	if narrativeContext != "" {
		prompt += fmt.Sprintf("\n\nAdditional context from narrative analysis:\n%s\nUse this context to help determine relationships.", narrativeContext)
	}

	return prompt
}

// propagateActiveAncestors walks up the relation chain and marks all ancestors
// of directly-active tags as active. Without this, intermediate abstract tags
// that have no direct article links get pruned during time-range filtering,
// causing top-level abstract tags to disappear.
func propagateActiveAncestors(activeTagIDs map[uint]bool, relations []models.TopicTagRelation) {
	childToParents := make(map[uint][]uint)
	for _, r := range relations {
		childToParents[r.ChildID] = append(childToParents[r.ChildID], r.ParentID)
	}

	queue := make([]uint, 0, len(activeTagIDs))
	for id := range activeTagIDs {
		queue = append(queue, id)
	}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, parentID := range childToParents[id] {
			if !activeTagIDs[parentID] {
				activeTagIDs[parentID] = true
				queue = append(queue, parentID)
			}
		}
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
	if err := query.Order("quality_score DESC, feed_count DESC, label ASC").Find(&tags).Error; err != nil {
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

	thresholds := es.GetThresholds()
	maxCheck := 3
	if len(candidates) < maxCheck {
		maxCheck = len(candidates)
	}

	for i := 0; i < maxCheck; i++ {
		candidate := candidates[i]
		if candidate.Tag == nil {
			continue
		}

		if candidate.Similarity >= thresholds.HighSimilarity {
			if err := mergeOrLinkSimilarAbstract(ctx, abstractTagID, candidate.Tag.ID); err != nil {
				logging.Warnf("MatchAbstractTagHierarchy: merge/link failed for %d vs %d: %v", abstractTagID, candidate.Tag.ID, err)
			}
			continue
		}

		if candidate.Similarity < thresholds.LowSimilarity {
			continue
		}

		parentID, childID, err := aiJudgeAbstractHierarchy(ctx, abstractTagID, candidate.Tag.ID)
		if err != nil {
			logging.Warnf("MatchAbstractTagHierarchy: AI judgment failed for %d vs %d: %v", abstractTagID, candidate.Tag.ID, err)
			continue
		}
		if err := linkAbstractParentChild(childID, parentID); err != nil {
			logging.Warnf("MatchAbstractTagHierarchy: failed to link %d under %d: %v", childID, parentID, err)
			continue
		}
		logging.Infof("Abstract hierarchy: %d is child of %d (AI judged, similarity=%.4f)", childID, parentID, candidate.Similarity)
	}
}

func adoptNarrowerAbstractChildren(ctx context.Context, abstractTagID uint) {
	defer func() {
		if r := recover(); r != nil {
			logging.Warnf("adoptNarrowerAbstractChildren panic for tag %d: %v", abstractTagID, r)
		}
	}()

	var abstractTag models.TopicTag
	if err := database.DB.First(&abstractTag, abstractTagID).Error; err != nil {
		logging.Warnf("adoptNarrowerAbstractChildren: tag %d not found: %v", abstractTagID, err)
		return
	}

	es := NewEmbeddingService()
	candidates, err := es.FindSimilarAbstractTags(ctx, abstractTagID, abstractTag.Category, 5)
	if err != nil || len(candidates) == 0 {
		if err != nil {
			logging.Warnf("adoptNarrowerAbstractChildren: similarity search failed for %d: %v", abstractTagID, err)
		}
		return
	}

	thresholds := es.GetThresholds()
	adopted := 0
	for _, candidate := range candidates {
		if candidate.Tag == nil || candidate.Similarity < thresholds.LowSimilarity {
			continue
		}

		isNarrower, err := aiJudgeNarrowerConceptFn(ctx, &abstractTag, candidate.Tag)
		if err != nil {
			logging.Warnf("adoptNarrowerAbstractChildren: AI judgment failed for %d vs %d: %v", abstractTagID, candidate.Tag.ID, err)
			continue
		}
		if !isNarrower {
			continue
		}

		if err := reparentOrLinkAbstractChild(ctx, candidate.Tag.ID, abstractTagID); err != nil {
			logging.Warnf("adoptNarrowerAbstractChildren: failed to link %d under %d: %v", candidate.Tag.ID, abstractTagID, err)
			continue
		}
		adopted++
	}

	if adopted > 0 {
		logging.Infof("adoptNarrowerAbstractChildren: abstract tag %d (%s) adopted %d narrower abstract tags", abstractTagID, abstractTag.Label, adopted)
		go EnqueueAbstractTagUpdate(abstractTagID, "adopted_narrower_children")
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

func mergeOrLinkSimilarAbstract(ctx context.Context, tag1ID, tag2ID uint) error {
	var tag1, tag2 models.TopicTag
	if err := database.DB.First(&tag1, tag1ID).Error; err != nil {
		return fmt.Errorf("load tag %d: %w", tag1ID, err)
	}
	if err := database.DB.First(&tag2, tag2ID).Error; err != nil {
		return fmt.Errorf("load tag %d: %w", tag2ID, err)
	}

	children1 := loadAbstractChildLabels(tag1ID, 5)
	children2 := loadAbstractChildLabels(tag2ID, 5)

	router := airouter.NewRouter()
	prompt := fmt.Sprintf(`两个抽象标签非常相似，请判断它们的关系。

标签 A: %q (描述: %s)
A 的子标签: %s

标签 B: %q (描述: %s)
B 的子标签: %s

判断:
- 如果它们描述的是完全相同的概念（只是表述不同），返回 "merge"
- 如果 A 是 B 的上位概念（更宽泛），返回 "parent_A"
- 如果 B 是 A 的上位概念（更宽泛），返回 "parent_B"

返回 JSON: {"action": "merge"|"parent_A"|"parent_B", "target": "A"|"B", "reason": "简要说明"}`,
		tag1.Label, truncateStr(tag1.Description, 200), formatChildLabels(children1),
		tag2.Label, truncateStr(tag2.Description, 200), formatChildLabels(children2))

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
				"action": {Type: "string", Description: "merge, parent_A, 或 parent_B"},
				"target": {Type: "string", Description: "A 或 B，merge 时保留的标签"},
				"reason": {Type: "string", Description: "判断理由"},
			},
			Required: []string{"action", "target", "reason"},
		},
		Temperature: func() *float64 { f := 0.2; return &f }(),
		Metadata: map[string]any{
			"operation": "merge_or_link_similar_abstract",
			"tag_a":     tag1ID,
			"tag_b":     tag2ID,
		},
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	var parsed struct {
		Action string `json:"action"`
		Target string `json:"target"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil {
		return fmt.Errorf("parse LLM response: %w", err)
	}

	switch parsed.Action {
	case "merge":
		sourceID, targetID := tag1ID, tag2ID
		sourceLabel, targetLabel := tag1.Label, tag2.Label
		if strings.EqualFold(parsed.Target, "A") {
			sourceID, targetID = tag2ID, tag1ID
			sourceLabel, targetLabel = tag2.Label, tag1.Label
		}
		logging.Infof("mergeOrLinkSimilarAbstract: merging %d (%s) into %d (%s), reason: %s",
			sourceID, sourceLabel, targetID, targetLabel, parsed.Reason)
		return MergeTags(sourceID, targetID)
	case "parent_A":
		return linkAbstractParentChild(tag2ID, tag1ID)
	case "parent_B":
		return linkAbstractParentChild(tag1ID, tag2ID)
	default:
		return fmt.Errorf("unknown action %q from LLM", parsed.Action)
	}
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

func buildCandidateSummary(candidates []TagCandidate) []string {
	summaries := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if c.Tag != nil {
			summaries = append(summaries, fmt.Sprintf("%s(id=%d,sim=%.2f,src=%s)", c.Tag.Label, c.Tag.ID, c.Similarity, c.Tag.Source))
		}
	}
	return summaries
}

func organizeMatchCategory(requestCategory string, tag *models.TopicTag) string {
	if strings.TrimSpace(requestCategory) != "" {
		return strings.TrimSpace(requestCategory)
	}
	if tag != nil && strings.TrimSpace(tag.Category) != "" {
		return strings.TrimSpace(tag.Category)
	}
	return "keyword"
}

func shouldUseOrganizeCandidate(candidate TagCandidate, currentTagID uint, used map[uint]bool) bool {
	if candidate.Tag == nil {
		return false
	}
	if candidate.Tag.ID == currentTagID {
		return false
	}
	if candidate.Similarity < DefaultThresholds.LowSimilarity {
		return false
	}
	return !used[candidate.Tag.ID]
}

func collectOrganizeMergeSources(result *TagExtractionResult, currentTag *models.TopicTag) []*models.TopicTag {
	if result == nil || result.Merge == nil || result.Merge.Target == nil {
		return nil
	}

	sourceByID := make(map[uint]*models.TopicTag)
	if currentTag != nil && currentTag.ID != 0 && currentTag.ID != result.Merge.Target.ID {
		sourceByID[currentTag.ID] = currentTag
	}
	for _, child := range result.MergeChildren {
		if child == nil || child.ID == 0 || child.ID == result.Merge.Target.ID {
			continue
		}
		sourceByID[child.ID] = child
	}

	sources := make([]*models.TopicTag, 0, len(sourceByID))
	for _, source := range sourceByID {
		sources = append(sources, source)
	}
	return sources
}

func applyOrganizeMerge(result *TagExtractionResult, currentTag *models.TopicTag) []*models.TopicTag {
	sources := collectOrganizeMergeSources(result, currentTag)
	if len(sources) == 0 {
		return nil
	}

	merged := make([]*models.TopicTag, 0, len(sources))
	for _, source := range sources {
		if err := MergeTags(source.ID, result.Merge.Target.ID); err != nil {
			logging.Warnf("OrganizeUnclassifiedTags: merge %d (%s) into %d (%s) failed: %v",
				source.ID, source.Label, result.Merge.Target.ID, result.Merge.Target.Label, err)
			continue
		}
		merged = append(merged, source)
	}
	return merged
}

// OrganizeUnclassifiedTags finds unclassified tags, groups similar ones,
// and runs ExtractAbstractTag on each group to build hierarchy.
func OrganizeUnclassifiedTags(ctx context.Context, category string, _ int) (*OrganizeResult, error) {
	tags, err := GetUnclassifiedTags(category, 0, 0, "")
	if err != nil {
		return nil, fmt.Errorf("query unclassified tags: %w", err)
	}
	if len(tags) == 0 {
		return &OrganizeResult{TotalUnclassified: 0, Processed: 0}, nil
	}

	totalUnclassified := len(tags)

	es := NewEmbeddingService()

	tagModels := loadTagModels(tags)

	processed := 0
	var groups []OrganizeGroup
	used := make(map[uint]bool)

	broadcastOrganizeProgress("processing", totalUnclassified, processed, nil, 0, category)

	for i, tag := range tags {
		if used[tag.ID] {
			continue
		}

		if tagModels[tag.ID] == nil {
			continue
		}

		currentTag := tagModels[tag.ID]
		matchCategory := organizeMatchCategory(category, currentTag)

		similarCandidates, err := es.FindSimilarTags(ctx, currentTag, matchCategory, 5, EmbeddingTypeSemantic)
		if err != nil {
			logging.Warnf("OrganizeUnclassifiedTags: FindSimilarTags failed for tag %d: %v", tag.ID, err)
			continue
		}

		var groupCandidates []TagCandidate
		groupCandidates = append(groupCandidates, TagCandidate{
			Tag:        currentTag,
			Similarity: 1.0,
		})

		for _, sc := range similarCandidates {
			if !shouldUseOrganizeCandidate(sc, tag.ID, used) {
				continue
			}
			groupCandidates = append(groupCandidates, sc)
		}

		if len(groupCandidates) < 2 {
			continue
		}

		logging.Infof("OrganizeUnclassifiedTags: processing group for %q with %d candidates", tag.Label, len(groupCandidates))

		result, err := ExtractAbstractTag(ctx, groupCandidates, tag.Label, matchCategory, WithCaller("OrganizeUnclassifiedTags"))
		if err != nil {
			logging.Warnf("OrganizeUnclassifiedTags: ExtractAbstractTag failed for %q: %v", tag.Label, err)
			continue
		}

		action := "none"
		mergedSources := applyOrganizeMerge(result, currentTag)
		hasMergeAction := len(mergedSources) > 0
		hasAbstractAction := result != nil && result.HasAbstract()
		if hasMergeAction || hasAbstractAction {
			processed++
			if hasMergeAction && hasAbstractAction {
				action = "merge+abstract"
			} else if hasMergeAction {
				action = "merge"
			} else if hasAbstractAction {
				action = "abstract"
			}

			if hasMergeAction {
				used[result.Merge.Target.ID] = true
				for _, c := range mergedSources {
					used[c.ID] = true
				}
			}
			if hasAbstractAction {
				used[result.Abstract.Tag.ID] = true
				for _, c := range result.Abstract.Children {
					used[c.ID] = true
				}
			}
		} else if result != nil && result.HasMerge() {
			logging.Infof("OrganizeUnclassifiedTags: merge judgment for %q had no mergeable sources", tag.Label)
		}

		g := OrganizeGroup{
			NewLabel:       tag.Label,
			CandidateCount: len(groupCandidates),
			Action:         action,
		}
		groups = append(groups, g)

		broadcastOrganizeProgress("processing", totalUnclassified, processed, &g, i+1, category)

		_ = i
	}

	broadcastOrganizeProgress("completed", totalUnclassified, processed, nil, 0, category)

	return &OrganizeResult{
		TotalUnclassified: totalUnclassified,
		Processed:         processed,
		Groups:            groups,
	}, nil
}

func broadcastOrganizeProgress(status string, total, processed int, currentGroup *OrganizeGroup, currentIndex int, category string) {
	hub := ws.GetHub()
	msg := ws.OrganizeProgressMessage{
		Type:              "organize_progress",
		Status:            status,
		TotalUnclassified: total,
		Processed:         processed,
		Category:          category,
	}
	if currentGroup != nil {
		msg.CurrentGroup = &ws.OrganizeGroupInfo{
			NewLabel:       currentGroup.NewLabel,
			CandidateCount: currentGroup.CandidateCount,
			Action:         currentGroup.Action,
		}
	}
	data, err := json.Marshal(msg)
	if err != nil {
		logging.Warnf("broadcastOrganizeProgress: marshal failed: %v", err)
		return
	}
	hub.BroadcastRaw(data)
}

func loadTagModels(nodes []TagHierarchyNode) map[uint]*models.TopicTag {
	result := make(map[uint]*models.TopicTag, len(nodes))
	for _, n := range nodes {
		var tag models.TopicTag
		if err := database.DB.First(&tag, n.ID).Error; err == nil {
			result[n.ID] = &tag
		}
	}
	return result
}

type OrganizeResult struct {
	TotalUnclassified int             `json:"total_unclassified"`
	Processed         int             `json:"processed"`
	Groups            []OrganizeGroup `json:"groups,omitempty"`
}

type OrganizeGroup struct {
	NewLabel       string `json:"new_label"`
	CandidateCount int    `json:"candidate_count"`
	Action         string `json:"action"`
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
