package topicanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/jsonutil"
	"my-robot-backend/internal/platform/logging"

	"gorm.io/gorm"
)

const maxHierarchyDepth = 4

func checkDepthLimit(tx *gorm.DB, parentID, childID uint) error {
	childSubtreeDepth := getAbstractSubtreeDepth(tx, childID)
	parentAncestryDepth := getTagDepthFromRootDB(tx, parentID)
	if childSubtreeDepth+parentAncestryDepth+1 > maxHierarchyDepth {
		return fmt.Errorf("depth limit: placing subtree(depth=%d) under parent(ancestry=%d) would exceed max depth %d", childSubtreeDepth, parentAncestryDepth, maxHierarchyDepth)
	}
	return nil
}

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
	thresholds := es.GetThresholds()

	treeDuplicates, err := findCrossLayerDuplicateCandidatesFn(ctx, abstractTagID, abstractTag.Category)
	if err != nil {
		logging.Warnf("MatchAbstractTagHierarchy: cross-layer dedup search failed: %v", err)
	} else {
		for _, dup := range treeDuplicates {
			if dup.Tag == nil {
				continue
			}
			shouldMerge, reason, judgeErr := judgeCrossLayerDuplicateFn(ctx, abstractTagID, dup.Tag.ID)
			if judgeErr != nil {
				logging.Warnf("MatchAbstractTagHierarchy: cross-layer judge failed for %d vs %d: %v", abstractTagID, dup.Tag.ID, judgeErr)
				continue
			}
			if !shouldMerge {
				logging.Infof("MatchAbstractTagHierarchy: candidate %d rejected by cross-layer judge for %d: %s", dup.Tag.ID, abstractTagID, reason)
				continue
			}
			if mergeErr := mergeTagsFn(abstractTagID, dup.Tag.ID); mergeErr != nil {
				logging.Warnf("MatchAbstractTagHierarchy: cross-layer merge failed for %d into %d: %v", abstractTagID, dup.Tag.ID, mergeErr)
				continue
			}
			logging.Infof("MatchAbstractTagHierarchy: merged %d into %d (cross-layer dedup, reason=%s)", abstractTagID, dup.Tag.ID, reason)
			return
		}
	}

	candidates, err := es.FindSimilarAbstractTags(ctx, abstractTagID, abstractTag.Category, 0)
	if err != nil {
		logging.Warnf("MatchAbstractTagHierarchy: failed to find similar abstract tags for %d: %v", abstractTagID, err)
		return
	}
	if len(candidates) == 0 {
		return
	}

	var highSimilars []TagCandidate
	var mediumSimilars []TagCandidate
	for _, candidate := range candidates {
		if candidate.Tag == nil {
			continue
		}
		if candidate.Similarity >= thresholds.HighSimilarity {
			highSimilars = append(highSimilars, candidate)
		} else if candidate.Similarity >= thresholds.LowSimilarity {
			mediumSimilars = append(mediumSimilars, candidate)
		}
	}

	for _, candidate := range highSimilars {
		if err := mergeOrLinkSimilarAbstract(ctx, abstractTagID, candidate.Tag.ID); err != nil {
			logging.Warnf("MatchAbstractTagHierarchy: merge/link failed for %d vs %d: %v", abstractTagID, candidate.Tag.ID, err)
			continue
		}
		return
	}

	if len(mediumSimilars) == 0 {
		return
	}

	if len(mediumSimilars) == 1 {
		candidate := mediumSimilars[0]
		judgment, err := judgeAbstractRelationship(ctx, abstractTagID, candidate.Tag.ID)
		if err != nil {
			logging.Warnf("MatchAbstractTagHierarchy: AI judgment failed for %d vs %d: %v", abstractTagID, candidate.Tag.ID, err)
			return
		}
		processAbstractRelationJudgment(ctx, abstractTagID, candidate.Tag.ID, judgment)
		return
	}

	batchResults, err := batchJudgeAbstractRelationships(ctx, &abstractTag, mediumSimilars)
	if err != nil {
		logging.Warnf("MatchAbstractTagHierarchy: batch AI judgment failed for %d: %v", abstractTagID, err)
		return
	}

	for _, r := range batchResults {
		if r.Action == "merge" {
			sourceID, targetID := abstractTagID, r.TagID
			if strings.EqualFold(r.Target, "A") {
				sourceID, targetID = r.TagID, abstractTagID
			}
			if mergeErr := MergeTags(sourceID, targetID); mergeErr != nil {
				logging.Warnf("MatchAbstractTagHierarchy: merge failed for %d into %d: %v", sourceID, targetID, mergeErr)
				continue
			}
			logging.Infof("MatchAbstractTagHierarchy: merged %d into %d (batch AI judged, reason=%s)", sourceID, targetID, r.Reason)
			return
		}
	}

	for _, r := range batchResults {
		switch r.Action {
		case "parent_A":
			if err := linkAbstractParentChild(r.TagID, abstractTagID); err != nil {
				logging.Warnf("MatchAbstractTagHierarchy: failed to link %d under %d: %v", r.TagID, abstractTagID, err)
				continue
			}
			logging.Infof("MatchAbstractTagHierarchy: %d is child of %d (batch AI judged, reason=%s)", r.TagID, abstractTagID, r.Reason)
		case "parent_B":
			if err := linkAbstractParentChild(abstractTagID, r.TagID); err != nil {
				logging.Warnf("MatchAbstractTagHierarchy: failed to link %d under %d: %v", abstractTagID, r.TagID, err)
				continue
			}
			logging.Infof("MatchAbstractTagHierarchy: %d is child of %d (batch AI judged, reason=%s)", abstractTagID, r.TagID, r.Reason)
		case "skip":
			logging.Infof("MatchAbstractTagHierarchy: skipped %d vs %d (batch AI judged, reason=%s)", abstractTagID, r.TagID, r.Reason)
		}
	}
}

func processAbstractRelationJudgment(ctx context.Context, abstractTagID, candidateTagID uint, judgment *abstractRelationJudgment) {
	switch judgment.Action {
	case "merge":
		sourceID, targetID := abstractTagID, candidateTagID
		if strings.EqualFold(judgment.Target, "A") {
			sourceID, targetID = candidateTagID, abstractTagID
		}
		if mergeErr := MergeTags(sourceID, targetID); mergeErr != nil {
			logging.Warnf("MatchAbstractTagHierarchy: merge failed for %d into %d: %v", sourceID, targetID, mergeErr)
			return
		}
		logging.Infof("MatchAbstractTagHierarchy: merged %d into %d (AI judged, reason=%s)", sourceID, targetID, judgment.Reason)
	case "parent_A":
		if err := linkAbstractParentChild(candidateTagID, abstractTagID); err != nil {
			logging.Warnf("MatchAbstractTagHierarchy: failed to link %d under %d: %v", candidateTagID, abstractTagID, err)
			return
		}
		logging.Infof("MatchAbstractTagHierarchy: %d is child of %d (AI judged, reason=%s)", candidateTagID, abstractTagID, judgment.Reason)
	case "parent_B":
		if err := linkAbstractParentChild(abstractTagID, candidateTagID); err != nil {
			logging.Warnf("MatchAbstractTagHierarchy: failed to link %d under %d: %v", abstractTagID, candidateTagID, err)
			return
		}
		logging.Infof("MatchAbstractTagHierarchy: %d is child of %d (AI judged, reason=%s)", abstractTagID, candidateTagID, judgment.Reason)
	case "skip":
		logging.Infof("MatchAbstractTagHierarchy: skipped %d vs %d (AI judged, reason=%s)", abstractTagID, candidateTagID, judgment.Reason)
	}
}

func adoptNarrowerAbstractChildren(ctx context.Context, abstractTagID uint) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			logging.Warnf("adoptNarrowerAbstractChildren panic for tag %d: %v", abstractTagID, r)
			retErr = fmt.Errorf("panic: %v", r)
		}
	}()

	var abstractTag models.TopicTag
	if err := database.DB.First(&abstractTag, abstractTagID).Error; err != nil {
		return fmt.Errorf("tag %d not found: %w", abstractTagID, err)
	}

	es := NewEmbeddingService()
	candidates, err := es.FindSimilarAbstractTags(ctx, abstractTagID, abstractTag.Category, 0)
	if err != nil {
		return fmt.Errorf("similarity search failed for %d: %w", abstractTagID, err)
	}
	if len(candidates) == 0 {
		return nil
	}

	thresholds := es.GetThresholds()
	var eligible []TagCandidate
	for _, c := range candidates {
		if c.Tag != nil && c.Similarity >= thresholds.LowSimilarity {
			eligible = append(eligible, c)
		}
	}
	if len(eligible) == 0 {
		return nil
	}

	narrowerIDs, err := batchJudgeNarrowerConceptsFn(ctx, &abstractTag, eligible)
	if err != nil {
		return fmt.Errorf("batch judge narrower failed for %d: %w", abstractTagID, err)
	}

	adopted := 0
	for _, cid := range narrowerIDs {
		if err := reparentOrLinkAbstractChild(ctx, cid, abstractTagID); err != nil {
			logging.Warnf("adoptNarrowerAbstractChildren: failed to link %d under %d: %v", cid, abstractTagID, err)
			continue
		}
		adopted++
	}

	if adopted > 0 {
		logging.Infof("adoptNarrowerAbstractChildren: abstract tag %d (%s) adopted %d narrower abstract tags", abstractTagID, abstractTag.Label, adopted)
		EnqueueAbstractTagUpdate(abstractTagID, "adopted_narrower_children")
	}
	return nil
}

func linkAbstractParentChild(childID, parentID uint) error {
	var parent, child models.TopicTag
	if err := database.DB.First(&parent, parentID).Error; err != nil {
		return fmt.Errorf("load parent tag %d: %w", parentID, err)
	}
	if err := database.DB.First(&child, childID).Error; err != nil {
		return fmt.Errorf("load child tag %d: %w", childID, err)
	}
	if parent.Kind != "abstract" && parent.Source != "abstract" {
		return fmt.Errorf("linkAbstractParentChild: parent %d (%q) is not abstract (kind=%s source=%s)", parentID, parent.Label, parent.Kind, parent.Source)
	}
	if child.Kind != "abstract" && child.Source != "abstract" {
		return fmt.Errorf("linkAbstractParentChild: child %d (%q) is not abstract (kind=%s source=%s)", childID, child.Label, child.Kind, child.Source)
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		wouldCycle, err := wouldCreateCycle(tx, parentID, childID)
		if err != nil {
			return fmt.Errorf("cycle check: %w", err)
		}
		if wouldCycle {
			return fmt.Errorf("would create cycle: parent=%d, child=%d", parentID, childID)
		}

		childSubtreeDepth := getAbstractSubtreeDepth(tx, childID)
		parentAncestryDepth := getTagDepthFromRootDB(tx, parentID)
		if childSubtreeDepth+parentAncestryDepth+1 > maxHierarchyDepth {
			return fmt.Errorf("depth limit: placing subtree(depth=%d) under parent(ancestry=%d) would exceed max depth %d", childSubtreeDepth, parentAncestryDepth, maxHierarchyDepth)
		}

		var count int64
		tx.Model(&models.TopicTagRelation{}).
			Where("parent_id = ? AND child_id = ?", parentID, childID).
			Count(&count)
		if count > 0 {
			return nil
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

	go func(id uint) {
		_, _ = resolveMultiParentConflict(id)
	}(childID)
	go enqueueEmbeddingsForNormalChildren(parentID)
	go EnqueueAbstractTagUpdate(parentID, "hierarchy_linked")

	return nil
}

type parentWithInfo struct {
	RelationID uint
	Parent     *models.TopicTag
}

type multiParentConflict struct {
	ChildID uint
	Parents []parentWithInfo
	Child   *models.TopicTag
}

type batchParentJudgment struct {
	Decisions []parentDecision `json:"decisions"`
}

type parentDecision struct {
	ChildID   uint `json:"child_id"`
	BestIndex int  `json:"best_index"` // 0-based index in parents list
}

// batchResolveMultiParentConflicts resolves multiple multi-parent conflicts in a single LLM call.
// First attempts to remove redundant ancestor parents without LLM, then batches remaining conflicts.
func batchResolveMultiParentConflicts(conflicts []multiParentConflict) (int, []string) {
	if len(conflicts) == 0 {
		return 0, nil
	}

	// Phase 1: Remove redundant ancestor parents (no LLM needed)
	resolved := 0
	var errors []string
	var remaining []multiParentConflict

	for _, c := range conflicts {
		txResolved := false
		if err := database.DB.Transaction(func(tx *gorm.DB) error {
			ok, err := removeRedundantAncestorParentsTx(tx, c.ChildID, c.Parents)
			if err != nil {
				return err
			}
			txResolved = ok
			return nil
		}); err != nil {
			errors = append(errors, fmt.Sprintf("child %d: ancestor check: %v", c.ChildID, err))
			continue
		}
		if txResolved {
			resolved++
			continue
		}
		remaining = append(remaining, c)
	}

	if len(remaining) == 0 {
		return resolved, errors
	}

	// Phase 2: Batch LLM judgment for remaining conflicts
	type conflictEntry struct {
		ChildID uint     `json:"child_id"`
		Child   string   `json:"child_label"`
		Parents []string `json:"parent_labels"`
	}
	var entries []conflictEntry
	for _, c := range remaining {
		var parentLabels []string
		for _, p := range c.Parents {
			parentLabels = append(parentLabels, fmt.Sprintf("%d:%s", p.Parent.ID, p.Parent.Label))
		}
		entries = append(entries, conflictEntry{
			ChildID: c.ChildID,
			Child:   c.Child.Label,
			Parents: parentLabels,
		})
	}

	entriesJSON, _ := json.MarshalIndent(entries, "", "  ")
	prompt := fmt.Sprintf(`以下标签有多个父标签（多父冲突），请为每个子标签选择最合适的父标签。

冲突列表：
%s

规则：
- 选择最具体、最相关的父标签
- 如果子标签与某个父标签有直接从属关系，选该父标签
- 如果子标签是某父标签领域的具体实例，选该父标签

返回 JSON: {"decisions": [{"child_id": ID, "best_index": 父标签在列表中的序号(从0开始)}, ...]}`,
		string(entriesJSON))

	router := airouter.NewRouter()
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
				"decisions": {
					Type: "array",
					Items: &airouter.SchemaProperty{
						Type: "object",
						Properties: map[string]airouter.SchemaProperty{
							"child_id":   {Type: "integer"},
							"best_index": {Type: "integer"},
						},
						Required: []string{"child_id", "best_index"},
					},
				},
			},
			Required: []string{"decisions"},
		},
		Temperature: func() *float64 { f := 0.2; return &f }(),
		Metadata: map[string]any{
			"operation":      "batch_resolve_multi_parent",
			"conflict_count": len(remaining),
		},
	}

	result, err := router.Chat(context.Background(), req)
	if err != nil {
		logging.Warnf("batchResolveMultiParentConflicts: LLM call failed: %v", err)
		return resolved, errors
	}

	content := jsonutil.SanitizeLLMJSON(result.Content)
	var judgment batchParentJudgment
	if err := json.Unmarshal([]byte(content), &judgment); err != nil {
		logging.Warnf("batchResolveMultiParentConflicts: parse failed: %v", err)
		return resolved, errors
	}

	conflictMap := make(map[uint]*multiParentConflict)
	for i := range remaining {
		conflictMap[remaining[i].ChildID] = &remaining[i]
	}

	for _, decision := range judgment.Decisions {
		conflict, ok := conflictMap[decision.ChildID]
		if !ok {
			continue
		}
		if decision.BestIndex < 0 || decision.BestIndex >= len(conflict.Parents) {
			errors = append(errors, fmt.Sprintf("child %d: invalid best_index %d", decision.ChildID, decision.BestIndex))
			continue
		}

		// Wrap in transaction for atomicity
		childID := decision.ChildID
		bestIdx := decision.BestIndex
		parents := conflict.Parents
		if err := database.DB.Transaction(func(tx *gorm.DB) error {
			for i, p := range parents {
				if i == bestIdx {
					continue
				}
				if err := tx.Delete(&models.TopicTagRelation{}, p.RelationID).Error; err != nil {
					return fmt.Errorf("remove relation %d from child %d: %w", p.RelationID, childID, err)
				}
			}
			return nil
		}); err != nil {
			errors = append(errors, err.Error())
			continue
		}

		logging.Infof("batchResolveMultiParentConflicts: resolved child %d, kept parent %d",
			decision.ChildID, conflict.Parents[decision.BestIndex].Parent.ID)
		resolved++
	}

	return resolved, errors
}

func resolveMultiParentConflict(childID uint) (bool, error) {
	if database.DB == nil {
		return false, nil
	}
	var result bool
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var relations []models.TopicTagRelation
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("child_id = ? AND relation_type = ?", childID, "abstract").
			Preload("Parent").
			Find(&relations).Error; err != nil {
			return err
		}
		if len(relations) <= 1 {
			return nil
		}

		var childTag models.TopicTag
		if err := tx.First(&childTag, childID).Error; err != nil {
			return fmt.Errorf("load child tag %d: %w", childID, err)
		}

		var parents []parentWithInfo
		for _, r := range relations {
			if r.Parent == nil {
				continue
			}
			parents = append(parents, parentWithInfo{RelationID: r.ID, Parent: r.Parent})
		}
		if len(parents) <= 1 {
			return nil
		}
		if resolved, err := removeRedundantAncestorParentsTx(tx, childID, parents); err != nil {
			return err
		} else if resolved {
			result = true
			return nil
		}

		bestIdx, err := aiJudgeBestParentFn(context.Background(), &childTag, parents)
		if err != nil {
			return fmt.Errorf("judge best parent for child %d: %w", childID, err)
		}

		removed := 0
		for i, p := range parents {
			if i == bestIdx {
				continue
			}
			if delErr := tx.Delete(&models.TopicTagRelation{}, p.RelationID).Error; delErr != nil {
				return fmt.Errorf("remove relation %d: %w", p.RelationID, delErr)
			} else {
				removed++
				logging.Infof("resolveMultiParentConflict: removed parent %d (%s) from child %d (%s), keeping parent %d (%s)",
					p.Parent.ID, p.Parent.Label, childID, childTag.Label,
					parents[bestIdx].Parent.ID, parents[bestIdx].Parent.Label)
			}
		}

		keptParent := parents[bestIdx].Parent
		go EnqueueAbstractTagUpdate(keptParent.ID, "multi_parent_resolved")

		result = removed > 0
		return nil
	})
	return result, err
}

func removeRedundantAncestorParentsTx(tx *gorm.DB, childID uint, parents []parentWithInfo) (bool, error) {
	removed := 0
	for _, maybeAncestor := range parents {
		for _, maybeDescendant := range parents {
			if maybeAncestor.Parent.ID == maybeDescendant.Parent.ID {
				continue
			}
			ancestor, err := isAbstractAncestorOf(maybeAncestor.Parent.ID, maybeDescendant.Parent.ID)
			if err != nil {
				return false, err
			}
			if !ancestor {
				continue
			}

			if delErr := tx.Delete(&models.TopicTagRelation{}, maybeAncestor.RelationID).Error; delErr != nil {
				return false, fmt.Errorf("remove redundant ancestor relation %d: %w", maybeAncestor.RelationID, delErr)
			}
			removed++
			logging.Infof("resolveMultiParentConflict: removed redundant ancestor parent %d (%s) from child %d, keeping narrower parent %d (%s)",
				maybeAncestor.Parent.ID, maybeAncestor.Parent.Label, childID,
				maybeDescendant.Parent.ID, maybeDescendant.Parent.Label)
			break
		}
	}

	return removed > 0, nil
}

func isAbstractAncestorOf(ancestorID, descendantID uint) (bool, error) {
	if ancestorID == 0 || descendantID == 0 || ancestorID == descendantID {
		return false, nil
	}

	visited := map[uint]bool{descendantID: true}
	queue := []uint{descendantID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		var relations []models.TopicTagRelation
		if err := database.DB.Where("child_id = ? AND relation_type = ?", current, "abstract").Find(&relations).Error; err != nil {
			return false, fmt.Errorf("load parents for ancestor check %d: %w", current, err)
		}
		for _, relation := range relations {
			if relation.ParentID == ancestorID {
				return true, nil
			}
			if !visited[relation.ParentID] {
				visited[relation.ParentID] = true
				queue = append(queue, relation.ParentID)
			}
		}
	}

	return false, nil
}
