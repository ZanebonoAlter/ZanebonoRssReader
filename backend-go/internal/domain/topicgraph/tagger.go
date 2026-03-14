package topicgraph

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

var errTopicAIUnavailable = errors.New("topic AI unavailable")

// TagSummary extracts and stores tags for an AI summary
// This is the main entry point called from the automatic summary scheduler
func TagSummary(summary *models.AISummary) error {
	if summary == nil || summary.ID == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	input := ExtractionInput{
		Title:        summary.Title,
		Summary:      summary.Summary,
		FeedName:     feedLabel(*summary),
		CategoryName: categoryLabel(*summary),
	}

	// Use the new extraction system
	extractor := NewTagExtractor()
	result, err := extractor.ExtractTags(ctx, input)

	var tags []TopicTag
	var source string

	if err != nil || len(result.Tags) == 0 {
		// Fall back to legacy heuristic extraction
		tags = legacyExtractTopics(input)
		source = "heuristic"
	} else {
		tags = result.Tags
		source = result.Source
	}

	if len(tags) == 0 {
		return nil
	}

	// Delete existing tag associations for this summary
	if err := database.DB.Where("summary_id = ?", summary.ID).Delete(&models.AISummaryTopic{}).Error; err != nil {
		return err
	}

	// Process each tag
	for _, tag := range dedupeTagsWithCategory(tags) {
		dbTag, err := findOrCreateTag(tag, source)
		if err != nil {
			continue // Skip on error, don't fail the whole operation
		}

		// Create the association
		link := models.AISummaryTopic{
			SummaryID:  summary.ID,
			TopicTagID: dbTag.ID,
			Score:      tag.Score,
			Source:     source,
		}
		if err := database.DB.Create(&link).Error; err != nil {
			return err
		}
	}

	return nil
}

// legacyExtractTopics is the old heuristic-based extraction (for fallback)
func legacyExtractTopics(input ExtractionInput) []TopicTag {
	// Use the existing extractor.go logic
	rawTags := ExtractTopics(input)
	result := make([]TopicTag, len(rawTags))
	for i, t := range rawTags {
		category := normalizeDisplayCategory(t.Kind, t.Category)
		result[i] = TopicTag{
			Label:    t.Label,
			Slug:     t.Slug,
			Category: category,
			Kind:     t.Kind, // Keep for backward compat
			Score:    t.Score,
		}
	}
	return result
}

// findOrCreateTag finds an existing tag or creates a new one
func findOrCreateTag(tag TopicTag, source string) (*models.TopicTag, error) {
	slug := slugify(tag.Label)
	category := normalizeDisplayCategory(tag.Kind, tag.Category)
	kind := normalizeTopicKind(tag.Kind, category)

	// Try to find existing tag by slug and category
	var dbTag models.TopicTag
	err := database.DB.Where("slug = ? AND category = ?", slug, category).First(&dbTag).Error
	if err == nil {
		// Found existing tag - update label and source if needed
		dbTag.Label = tag.Label
		dbTag.Category = category
		dbTag.Source = source
		if tag.Icon != "" {
			dbTag.Icon = tag.Icon
		}
		if len(tag.Aliases) > 0 {
			aliasesJSON, _ := json.Marshal(tag.Aliases)
			dbTag.Aliases = string(aliasesJSON)
		}
		dbTag.Kind = kind
		if err := database.DB.Save(&dbTag).Error; err != nil {
			return nil, err
		}
		return &dbTag, nil
	}

	// Create new tag
	aliasesJSON, _ := json.Marshal(tag.Aliases)
	newTag := models.TopicTag{
		Slug:        slug,
		Label:       tag.Label,
		Category:    category,
		Kind:        kind,
		Icon:        tag.Icon,
		Aliases:     string(aliasesJSON),
		IsCanonical: true,
		Source:      source,
	}
	if err := database.DB.Create(&newTag).Error; err != nil {
		return nil, err
	}

	return &newTag, nil
}

func normalizeDisplayCategory(kind string, fallback string) string {
	switch kind {
	case "topic":
		return "event"
	case "entity":
		return "person"
	case "keyword":
		return "keyword"
	}

	switch fallback {
	case "topic":
		return "event"
	case "entity":
		return "person"
	case "event", "person", "keyword":
		return fallback
	default:
		return "keyword"
	}
}

func normalizeTopicKind(kind string, category string) string {
	switch kind {
	case "topic", "entity", "keyword":
		return kind
	}

	switch category {
	case "event":
		return "topic"
	case "person":
		return "entity"
	default:
		return "keyword"
	}
}

// dedupeTagsWithCategory removes duplicate tags based on (category, slug)
func dedupeTagsWithCategory(items []TopicTag) []TopicTag {
	best := make(map[string]TopicTag)
	for _, item := range items {
		if item.Slug == "" {
			item.Slug = slugify(item.Label)
		}
		if item.Category == "" {
			item.Category = "keyword"
		}
		key := item.Category + ":" + item.Slug
		current, exists := best[key]
		if !exists || current.Score < item.Score {
			best[key] = item
		}
	}

	result := make([]TopicTag, 0, len(best))
	for _, item := range best {
		result = append(result, item)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].Label < result[j].Label
		}
		return result[i].Score > result[j].Score
	})

	return result
}

// sortTagsByScore sorts tags by score descending
func sortTagsByScore(tags []TopicTag) {
	sort.SliceStable(tags, func(i, j int) bool {
		if tags[i].Score == tags[j].Score {
			return tags[i].Label < tags[j].Label
		}
		return tags[i].Score > tags[j].Score
	})
}

// slugify creates a URL-safe slug from a string (uses extractor.go implementation)

// dedupeTopics is kept for backward compatibility with extractor.go
func dedupeTopics(items []TopicTag) []TopicTag {
	return dedupeTagsWithCategory(items)
}
