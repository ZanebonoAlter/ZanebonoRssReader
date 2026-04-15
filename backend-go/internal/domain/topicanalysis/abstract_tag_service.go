package topicanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

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
	QualityScore    float64            `json:"quality_score,omitempty"`
	IsLowQuality    bool               `json:"is_low_quality,omitempty"`
	Icon            string             `json:"icon"`
	FeedCount       int                `json:"feed_count"`
	SimilarityScore float64            `json:"similarity_score,omitempty"`
	IsActive        bool               `json:"is_active"`
	Children        []TagHierarchyNode `json:"children"`
}

// ExtractAbstractTag attempts to extract a common abstract concept from middle-band candidates.
// If LLM succeeds, creates an abstract tag + parent-child relations.
// Returns the abstract tag on success, nil on failure (caller should fall back to creating a normal tag).
func ExtractAbstractTag(ctx context.Context, candidates []TagCandidate, newLabel string, category string) (*models.TopicTag, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates provided for abstract tag extraction")
	}

	// Call LLM to extract abstract name and description
	abstractName, abstractDesc, err := callLLMForAbstractName(ctx, candidates, newLabel)
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
			QualityScore:    child.QualityScore,
			IsLowQuality:    child.Source != "abstract" && child.QualityScore < 0.3,
	}

	// Resolve active tag IDs based on time range
	activeTagIDs := resolveActiveTagIDs(timeRange, tagIDSet)

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
			IsActive:        activeTagIDs[child.ID],
			Children:        []TagHierarchyNode{},
		})
		parentSet[r.ParentID] = true
	}

	// Find root parents (tags that are parents but not children in any relation in this set)
			QualityScore: parent.QualityScore,
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
			IsActive:  activeTagIDs[parent.ID],
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

		// 3. Remove from current parent (if any)
		tx.Where("child_id = ?", tagID).Delete(&models.TopicTagRelation{})

		// 4. Create new relation
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

// callLLMForAbstractName calls the LLM to extract a common abstract concept and description from candidates.
func callLLMForAbstractName(ctx context.Context, candidates []TagCandidate, newLabel string) (string, string, error) {
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
		return "", "", fmt.Errorf("LLM call failed: %w", err)
	}

	return parseAbstractTagResponse(result.Content)
}

// buildAbstractTagPrompt constructs the prompt for abstract tag extraction.
func buildAbstractTagPrompt(candidates []TagCandidate, newLabel string) string {
	var parts []string
	for _, c := range candidates {
		if c.Tag != nil {
			desc := ""
			if c.Tag.Description != "" {
				desc = fmt.Sprintf(" (description: %s)", c.Tag.Description)
			}
			parts = append(parts, fmt.Sprintf("- %q (similarity: %.2f)%s", c.Tag.Label, c.Similarity, desc))
		}
	}
	parts = append(parts, fmt.Sprintf("- %q (new tag)", newLabel))

	return fmt.Sprintf(`Given these semantically similar tags:
%s

Extract a common abstract concept that encompasses ALL of them.
The abstract name should be broader and more general.
Also generate a brief description (1-2 sentences) summarizing the common theme based on the child tag descriptions.

Respond with JSON:
{"abstract_name": "your answer", "description": "brief description", "reason": "explanation"}

Rules:
- abstract_name must be 1-160 characters
- description must be 1-500 characters
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

// parseAbstractTagResponse extracts abstract_name and description from LLM JSON response.
func parseAbstractTagResponse(content string) (string, string, error) {
	content = strings.TrimSpace(content)

	var result struct {
		AbstractName string `json:"abstract_name"`
		Description  string `json:"description"`
		Reason       string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return "", "", fmt.Errorf("failed to parse LLM response: %w", err)
	}

	name := strings.TrimSpace(result.AbstractName)
	if name == "" {
		return "", "", fmt.Errorf("LLM returned empty abstract_name")
	}
	if len(name) > maxAbstractNameLen {
		return "", "", fmt.Errorf("abstract_name exceeds %d characters", maxAbstractNameLen)
	}

	desc := strings.TrimSpace(result.Description)
	if len(desc) > 500 {
		desc = desc[:500]
	}

	return name, desc, nil
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

	nodes := make([]TagHierarchyNode, 0, len(tags))
	for _, tag := range tags {
		nodes = append(nodes, TagHierarchyNode{
			ID:        tag.ID,
			Label:     tag.Label,
			QualityScore: tag.QualityScore,
			IsLowQuality: tag.Source != "abstract" && tag.QualityScore < 0.3,
			Slug:      tag.Slug,
			Category:  tag.Category,
			Icon:      tag.Icon,
			FeedCount: tag.FeedCount,
			IsActive:  activeTagIDs[tag.ID],
			Children:  []TagHierarchyNode{},
		})
	}
	return nodes, nil
}
