package topicanalysis

import (
	"context"
	"fmt"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"

	"gorm.io/gorm"
)

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

	for _, candidate := range candidates {
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

		childDepth := getAbstractSubtreeDepth(database.DB, abstractTagID)
		parentDepth := getTagDepthFromRoot(candidate.Tag.ID)
		if childDepth+parentDepth+1 > 4 {
			alternativeID, reason, altErr := aiJudgeAlternativePlacementFn(ctx, abstractTagID, candidate.Tag.ID)
			if altErr != nil {
				logging.Warnf("MatchAbstractTagHierarchy: depth-limit AI judgment failed for %d vs %d: %v", abstractTagID, candidate.Tag.ID, altErr)
				continue
			}
			if alternativeID > 0 {
				if linkErr := linkAbstractParentChild(abstractTagID, alternativeID); linkErr != nil {
					logging.Warnf("MatchAbstractTagHierarchy: alternative placement failed for %d under %d: %v", abstractTagID, alternativeID, linkErr)
				} else {
					logging.Infof("MatchAbstractTagHierarchy: depth limit rerouted %d under %d: %s", abstractTagID, alternativeID, reason)
				}
			}
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
	candidates, err := es.FindSimilarAbstractTags(ctx, abstractTagID, abstractTag.Category, 0)
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

	depthErr := ""
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		wouldCycle, err := wouldCreateCycle(tx, parentID, childID)
		if err != nil {
			return fmt.Errorf("cycle check: %w", err)
		}
		if wouldCycle {
			return fmt.Errorf("would create cycle: parent=%d, child=%d", parentID, childID)
		}

		childSubtreeDepth := getAbstractSubtreeDepth(tx, childID)
		parentAncestryDepth := getTagDepthFromRoot(parentID)
		if childSubtreeDepth+parentAncestryDepth+1 > 4 {
			depthErr = fmt.Sprintf("depth limit: placing subtree(depth=%d) under parent(ancestry=%d) would exceed max depth 4", childSubtreeDepth, parentAncestryDepth)
			return fmt.Errorf("%s", depthErr)
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
		if depthErr != "" {
			if alternativeID, reason, altErr := aiJudgeAlternativePlacementFn(context.Background(), childID, parentID); altErr != nil {
				logging.Warnf("linkAbstractParentChild: alternative placement lookup failed for child=%d parent=%d: %v", childID, parentID, altErr)
			} else if alternativeID > 0 {
				logging.Infof("linkAbstractParentChild: depth limit prevented %d -> %d, suggested alternative parent %d: %s", parentID, childID, alternativeID, reason)
			}
		}
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

func resolveMultiParentConflict(childID uint) (bool, error) {
	if database.DB == nil {
		return false, nil
	}
	var relations []models.TopicTagRelation
	if err := database.DB.
		Where("child_id = ? AND relation_type = ?", childID, "abstract").
		Preload("Parent").
		Find(&relations).Error; err != nil {
		return false, fmt.Errorf("load parents for child %d: %w", childID, err)
	}
	if len(relations) <= 1 {
		return false, nil
	}

	var childTag models.TopicTag
	if err := database.DB.First(&childTag, childID).Error; err != nil {
		return false, fmt.Errorf("load child tag %d: %w", childID, err)
	}

	var parents []parentWithInfo
	for _, r := range relations {
		if r.Parent == nil {
			continue
		}
		parents = append(parents, parentWithInfo{RelationID: r.ID, Parent: r.Parent})
	}
	if len(parents) <= 1 {
		return false, nil
	}
	if resolved, err := removeRedundantAncestorParents(childID, parents); err != nil {
		return false, err
	} else if resolved {
		return true, nil
	}

	bestIdx, err := aiJudgeBestParentFn(context.Background(), &childTag, parents)
	if err != nil {
		return false, fmt.Errorf("judge best parent for child %d: %w", childID, err)
	}

	removed := 0
	for i, p := range parents {
		if i == bestIdx {
			continue
		}
		if delErr := database.DB.Delete(&models.TopicTagRelation{}, p.RelationID).Error; delErr != nil {
			return false, fmt.Errorf("remove relation %d: %w", p.RelationID, delErr)
		} else {
			removed++
			logging.Infof("resolveMultiParentConflict: removed parent %d (%s) from child %d (%s), keeping parent %d (%s)",
				p.Parent.ID, p.Parent.Label, childID, childTag.Label,
				parents[bestIdx].Parent.ID, parents[bestIdx].Parent.Label)
		}
	}

	keptParent := parents[bestIdx].Parent
	go EnqueueAbstractTagUpdate(keptParent.ID, "multi_parent_resolved")

	return removed > 0, nil
}

func removeRedundantAncestorParents(childID uint, parents []parentWithInfo) (bool, error) {
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

			if delErr := database.DB.Delete(&models.TopicTagRelation{}, maybeAncestor.RelationID).Error; delErr != nil {
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
