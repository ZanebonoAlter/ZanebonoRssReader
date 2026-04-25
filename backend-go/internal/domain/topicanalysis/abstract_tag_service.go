package topicanalysis

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topictypes"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"

	"gorm.io/gorm"
)

const (
	maxAbstractNameLen = 160
)

var errInsufficientAbstractChildren = errors.New("abstract tag needs enough children")

var (
	findSimilarExistingAbstractFn       = findSimilarExistingAbstract
	aiJudgeNarrowerConceptFn            = aiJudgeNarrowerConcept
	aiJudgeBestParentFn                 = aiJudgeBestParent
	findCrossLayerDuplicateCandidatesFn = findCrossLayerDuplicateCandidates
	judgeCrossLayerDuplicateFn          = judgeCrossLayerDuplicate
	aiJudgeAlternativePlacementFn       = aiJudgeAlternativePlacement
	mergeTagsFn                         = MergeTags
)

type TagExtractionResult struct {
	Merge           *MergeResult       `json:"merge,omitempty"`
	Abstract        *AbstractResult    `json:"abstract,omitempty"`
	MergeChildren   []*models.TopicTag `json:"merge_children,omitempty"`
	LLMExplicitNone bool               `json:"llm_explicit_none,omitempty"`
}

type MergeResult struct {
	Target *models.TopicTag `json:"target"`
	Label  string           `json:"label"`
}

type AbstractResult struct {
	Tag      *models.TopicTag   `json:"tag"`
	Children []*models.TopicTag `json:"children"`
}

func (r *TagExtractionResult) HasMerge() bool    { return r != nil && r.Merge != nil }
func (r *TagExtractionResult) HasAbstract() bool { return r != nil && r.Abstract != nil }
func (r *TagExtractionResult) HasAction() bool   { return r.HasMerge() || r.HasAbstract() }

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

		var topSim float64
		for _, c := range candidates {
			if c.Tag != nil && c.Tag.ID == mergeTarget.ID {
				topSim = c.Similarity
				break
			}
		}
		if topSim > 0 && topSim < mergeMinSimilarity {
			logging.Warnf("Tag judgment: rejecting merge for %q — top candidate %q similarity %.4f < %.2f", newLabel, mergeTarget.Label, topSim, mergeMinSimilarity)
			result.Merge = nil
		} else {
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
		result.LLMExplicitNone = true
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
				if existing.Kind == "abstract" || existing.Source == "abstract" {
					abstractTag = &existing
				} else {
					logging.Infof("processAbstractJudgment: slug match %q found non-abstract tag %d (%q, kind=%s), skipping reuse",
						slug, existing.ID, existing.Label, existing.Kind)
				}
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
		for _, child := range abstractChildren {
			go func(childID uint) {
				_, _ = resolveMultiParentConflict(childID)
			}(child.ID)
		}
	}

	return &AbstractResult{
		Tag:      abstractTag,
		Children: abstractChildren,
	}, nil
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
