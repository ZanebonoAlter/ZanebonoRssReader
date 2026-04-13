package topicanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topictypes"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"

	"gorm.io/gorm"
)

const (
	// maxAbstractNameLen limits LLM-returned abstract tag names to prevent abuse
	maxAbstractNameLen = 160
)

// TagHierarchyNode represents a node in the tag hierarchy tree
type TagHierarchyNode struct {
	ID              uint               `json:"id"`
	Label           string             `json:"label"`
	Slug            string             `json:"slug"`
	Category        string             `json:"category"`
	Icon            string             `json:"icon"`
	FeedCount       int                `json:"feed_count"`
	SimilarityScore float64            `json:"similarity_score,omitempty"`
	Children        []TagHierarchyNode `json:"children"`
}

// ExtractAbstractTag attempts to extract a common abstract concept from middle-band candidates.
// If LLM succeeds, creates an abstract tag + parent-child relations.
// Returns the abstract tag on success, nil on failure (caller should fall back to creating a normal tag).
func ExtractAbstractTag(ctx context.Context, candidates []TagCandidate, newLabel string, category string) (*models.TopicTag, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates provided for abstract tag extraction")
	}

	// Call LLM to extract abstract name
	abstractName, err := callLLMForAbstractName(ctx, candidates, newLabel)
	if err != nil {
		log.Printf("[WARN] Abstract tag extraction LLM call failed: %v", err)
		return nil, err
	}

	// Generate slug
	slug := topictypes.Slugify(abstractName)
	if slug == "" {
		return nil, fmt.Errorf("generated empty slug for abstract name %q", abstractName)
	}

	// Determine category: inherit from first candidate (per D-05 Claude's Discretion)
	if category == "" && len(candidates) > 0 && candidates[0].Tag != nil {
		category = candidates[0].Tag.Category
	}
	if category == "" {
		category = "keyword"
	}

	var abstractTag *models.TopicTag

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Check if an abstract tag with this slug already exists (dedup per D-05)
		var existing models.TopicTag
		if err := tx.Where("slug = ? AND category = ? AND status = ?", slug, category, "active").First(&existing).Error; err == nil {
			// Reuse existing abstract tag
			abstractTag = &existing
		} else {
			// Create new abstract tag
			abstractTag = &models.TopicTag{
				Slug:     slug,
				Label:    abstractName,
				Category: category,
				Kind:     category,
				Source:   "abstract",
				Status:   "active",
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
					log.Printf("[WARN] Failed to generate embedding for abstract tag %d: %v", tagID, genErr)
					return
				}
				emb.TopicTagID = tagID
				if saveErr := es.SaveEmbedding(emb); saveErr != nil {
					log.Printf("[WARN] Failed to save embedding for abstract tag %d: %v", tagID, saveErr)
				}
			}(abstractTag.ID)
		}

		// Build parent-child relations for all candidates
		for _, candidate := range candidates {
			if candidate.Tag == nil {
				continue
			}
			// Skip self-relation
			if candidate.Tag.ID == abstractTag.ID {
				continue
			}

			// Check if relation already exists
			var count int64
			tx.Model(&models.TopicTagRelation{}).
				Where("parent_id = ? AND child_id = ?", abstractTag.ID, candidate.Tag.ID).
				Count(&count)
			if count > 0 {
				continue // Relation already exists
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
		}

		return nil
	})

	if err != nil {
		log.Printf("[WARN] Abstract tag transaction failed: %v", err)
		return nil, err
	}

	log.Printf("[INFO] Abstract tag extracted: %s (id=%d) from candidates [%s]",
		abstractTag.Label, abstractTag.ID, candidateLabels(candidates))

	return abstractTag, nil
}

// GetTagHierarchy returns the tag hierarchy tree for a given category filter.
func GetTagHierarchy(category string) ([]TagHierarchyNode, error) {
	// Query all abstract relations
	query := database.DB.Model(&models.TopicTagRelation{}).
		Where("relation_type = ?", "abstract")

	var relations []models.TopicTagRelation
	if err := query.Find(&relations).Error; err != nil {
		return nil, fmt.Errorf("query tag relations: %w", err)
	}

	if len(relations) == 0 {
		return []TagHierarchyNode{}, nil
	}

	// Collect all unique tag IDs
	tagIDSet := make(map[uint]bool)
	for _, r := range relations {
		tagIDSet[r.ParentID] = true
		tagIDSet[r.ChildID] = true
	}
	tagIDs := make([]uint, 0, len(tagIDSet))
	for id := range tagIDSet {
		tagIDs = append(tagIDs, id)
	}

	// Load all referenced tags
	var tags []models.TopicTag
	if err := database.DB.Where("id IN ? AND status = ?", tagIDs, "active").Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("load tags: %w", err)
	}
	tagMap := make(map[uint]*models.TopicTag, len(tags))
	for i := range tags {
		tagMap[tags[i].ID] = &tags[i]
	}

	// Filter by category if specified
	if category != "" {
		// Only include trees where the root parent matches the category
		filteredRelations := make([]models.TopicTagRelation, 0, len(relations))
		parentHasCategory := make(map[uint]bool)
		for _, r := range relations {
			parent, ok := tagMap[r.ParentID]
			if ok && parent.Category == category {
				filteredRelations = append(filteredRelations, r)
				parentHasCategory[r.ParentID] = true
			}
			// Also keep relations for children that are themselves parents in this category
		}
		relations = filteredRelations
	}

	// Build parent → children map
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
			SimilarityScore: r.SimilarityScore,
			Children:        []TagHierarchyNode{},
		})
		parentSet[r.ParentID] = true
	}

	// Find root parents (tags that are parents but not children in any relation in this set)
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
		// A root is a parent that is not also a child (unless it's itself)
		if childSet[parentID] && parentSet[parentID] {
			// This is a nested parent — it's both a parent and a child
			continue
		}
		children := buildHierarchy(childrenMap, parentID)
		roots = append(roots, TagHierarchyNode{
			ID:        parent.ID,
			Label:     parent.Label,
			Slug:      parent.Slug,
			Category:  parent.Category,
			Icon:      parent.Icon,
			FeedCount: parent.FeedCount,
			Children:  children,
		})
	}

	return roots, nil
}

// buildHierarchy recursively builds the tree from the childrenMap
func buildHierarchy(childrenMap map[uint][]TagHierarchyNode, parentID uint) []TagHierarchyNode {
	children, ok := childrenMap[parentID]
	if !ok {
		return []TagHierarchyNode{}
	}
	for i, child := range children {
		grandChildren := buildHierarchy(childrenMap, child.ID)
		children[i].Children = grandChildren
	}
	return children
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
			log.Printf("[WARN] Failed to load tag %d for re-embedding: %v", tid, err)
			return
		}
		emb, err := es.GenerateEmbedding(context.Background(), &tag)
		if err != nil {
			log.Printf("[WARN] Failed to generate embedding for renamed tag %d: %v", tid, err)
			return
		}
		emb.TopicTagID = tid
		if err := es.SaveEmbedding(emb); err != nil {
			log.Printf("[WARN] Failed to save embedding for renamed tag %d: %v", tid, err)
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

// --- Internal helpers ---

// callLLMForAbstractName calls the LLM to extract a common abstract concept from candidates.
func callLLMForAbstractName(ctx context.Context, candidates []TagCandidate, newLabel string) (string, error) {
	router := airouter.NewRouter()
	prompt := buildAbstractTagPrompt(candidates, newLabel)

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "You are a tag taxonomy assistant. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		JSONMode:    true,
		Temperature: func() *float64 { f := 0.3; return &f }(),
	}

	result, err := router.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	return parseAbstractNameFromJSON(result.Content)
}

// buildAbstractTagPrompt constructs the prompt for abstract tag extraction.
func buildAbstractTagPrompt(candidates []TagCandidate, newLabel string) string {
	var parts []string
	for _, c := range candidates {
		if c.Tag != nil {
			parts = append(parts, fmt.Sprintf("- %q (similarity: %.2f)", c.Tag.Label, c.Similarity))
		}
	}
	parts = append(parts, fmt.Sprintf("- %q (new tag)", newLabel))

	return fmt.Sprintf(`Given these semantically similar tags:
%s

Extract a common abstract concept that encompasses ALL of them.
The abstract name should be broader and more general.

Respond with JSON:
{"abstract_name": "your answer", "reason": "brief explanation"}

Rules:
- abstract_name must be 1-160 characters
- abstract_name should be in the original language of the tags
- Prefer concise, meaningful names over vague descriptions`,
		strings.Join(parts, "\n"))
}

// parseAbstractNameFromJSON extracts the abstract_name from LLM JSON response.
func parseAbstractNameFromJSON(content string) (string, error) {
	content = strings.TrimSpace(content)

	var result struct {
		AbstractName string `json:"abstract_name"`
		Reason       string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return "", fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	name := strings.TrimSpace(result.AbstractName)
	if name == "" {
		return "", fmt.Errorf("LLM returned empty abstract_name")
	}
	if len(name) > maxAbstractNameLen {
		return "", fmt.Errorf("LLM returned abstract_name exceeding %d characters", maxAbstractNameLen)
	}

	return name, nil
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
