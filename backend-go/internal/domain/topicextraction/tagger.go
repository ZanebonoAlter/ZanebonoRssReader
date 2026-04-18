package topicextraction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topicanalysis"
	"my-robot-backend/internal/domain/topictypes"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
	"my-robot-backend/internal/platform/logging"
)

var errTopicAIUnavailable = errors.New("topic AI unavailable")

var (
	embeddingService          *topicanalysis.EmbeddingService
	embeddingServiceOnce      sync.Once
	embeddingQueueService     *topicanalysis.EmbeddingQueueService
	embeddingQueueServiceOnce sync.Once
)

func getEmbeddingService() *topicanalysis.EmbeddingService {
	embeddingServiceOnce.Do(func() {
		embeddingService = topicanalysis.NewEmbeddingService()
	})
	return embeddingService
}

func getEmbeddingQueueService() *topicanalysis.EmbeddingQueueService {
	embeddingQueueServiceOnce.Do(func() {
		embeddingQueueService = topicanalysis.NewEmbeddingQueueService(nil)
	})
	return embeddingQueueService
}

// TagSummary extracts and stores tags for an AI summary
// This is the main entry point called from the automatic summary scheduler
// Skips if the summary already has tags (dedup)
func TagSummary(summary *models.AISummary, feedName, categoryName string) error {
	if summary == nil || summary.ID == 0 {
		return nil
	}

	var existingCount int64
	database.DB.Model(&models.AISummaryTopic{}).Where("summary_id = ?", summary.ID).Count(&existingCount)
	if existingCount > 0 {
		return nil
	}

	input := topictypes.ExtractionInput{
		Title:        summary.Title,
		Summary:      summary.Summary,
		FeedName:     feedName,
		CategoryName: categoryName,
		SummaryID:    &summary.ID,
	}

	// Use the new extraction system
	extractor := NewTagExtractor()
	result, err := extractor.ExtractTags(context.Background(), input)

	var tags []topictypes.TopicTag
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

	// Build article context for description generation
	articleContext := ""
	if summary.Title != "" {
		articleContext = summary.Title
	}
	if summary.Summary != "" {
		if articleContext != "" {
			articleContext += ". "
		}
		runes := []rune(summary.Summary)
		if len(runes) > 2000 {
			articleContext += string(runes[:2000])
		} else {
			articleContext += summary.Summary
		}
	}

	// Process each tag
	for _, tag := range dedupeTagsWithCategory(tags) {
		dbTag, err := findOrCreateTag(context.Background(), tag, source, articleContext)
		if err != nil {
			logging.Warnf("findOrCreateTag failed for tag %q (category=%s, slug=%s, source=%s, summary=%d): %v", tag.Label, tag.Category, topictypes.Slugify(tag.Label), source, summary.ID, err)
			continue
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
func legacyExtractTopics(input topictypes.ExtractionInput) []topictypes.TopicTag {
	// Use the existing extractor.go logic
	rawTags := ExtractTopics(input)
	result := make([]topictypes.TopicTag, len(rawTags))
	for i, t := range rawTags {
		category := NormalizeDisplayCategory(t.Kind, t.Category)
		result[i] = topictypes.TopicTag{
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
// Uses three-level matching: exact/alias → embedding similarity → create new
func findOrCreateTag(ctx context.Context, tag topictypes.TopicTag, source string, articleContext string) (*models.TopicTag, error) {
	slug := topictypes.Slugify(tag.Label)
	category := NormalizeDisplayCategory(tag.Kind, tag.Category)
	kind := NormalizeTopicKind(tag.Kind, category)

	// Build aliases string for TagMatch
	aliases := tag.Aliases
	if len(aliases) == 0 {
		aliases = []string{}
	}
	aliasesJSON, _ := json.Marshal(aliases)

	// Try embedding-based three-level matching
	es := getEmbeddingService()
	if es != nil {
		matchResult, err := es.TagMatch(ctx, tag.Label, category, string(aliasesJSON))
		if err != nil {
			// Embedding unavailable — fall back to exact match
			logging.Warnf("TagMatch failed, falling back to exact match: %v", err)
		} else {
			switch matchResult.MatchType {
			case "exact":
				if matchResult.ExistingTag != nil {
					existing := matchResult.ExistingTag
					existing.Label = tag.Label
					existing.Category = category
					existing.Source = source
					if tag.Icon != "" {
						existing.Icon = tag.Icon
					}
					if len(tag.Aliases) > 0 {
						aJSON, _ := json.Marshal(tag.Aliases)
						existing.Aliases = string(aJSON)
					}
					existing.Kind = kind
					if err := database.DB.Save(existing).Error; err != nil {
						return nil, err
					}
					go ensureTagEmbedding(es, existing.ID)
					go backfillTagDescription(existing.ID, existing.Label, existing.Category, existing.Description, articleContext)
					return existing, nil
				}

			case "candidates":
				candidates := matchResult.Candidates
				logging.Infof("Batch tag judgment for %q: %d candidates (top similarity %.2f)", tag.Label, len(candidates), matchResult.Similarity)
				result, judgmentErr := topicanalysis.ExtractAbstractTag(ctx, candidates, tag.Label, category)
				if judgmentErr != nil || result == nil {
					logging.Warnf("Tag judgment failed for %q, falling back to new tag creation: %v", tag.Label, judgmentErr)
					break
				}

				if result.Action == topicanalysis.ActionMerge {
					existing := result.MergeTarget
					if result.MergeLabel != "" {
						existing.Label = result.MergeLabel
					} else {
						existing.Label = tag.Label
					}
					existing.Category = category
					existing.Source = source
					if len(tag.Aliases) > 0 {
						aJSON, _ := json.Marshal(tag.Aliases)
						existing.Aliases = string(aJSON)
					}
					if tag.Icon != "" {
						existing.Icon = tag.Icon
					}
					existing.Kind = kind
					if err := database.DB.Save(existing).Error; err != nil {
						logging.Warnf("Failed to save merged tag %d: %v", existing.ID, err)
						break
					}
					go ensureTagEmbedding(es, existing.ID)
					go backfillTagDescription(existing.ID, existing.Label, existing.Category, existing.Description, articleContext)
					return existing, nil
				}

				if result.Action == topicanalysis.ActionNone {
					logging.Infof("Tag judgment: none — tag %q is independent from %d candidates, creating new tag", tag.Label, len(candidates))
					break
				}

				for _, c := range candidates {
					if c.Tag != nil {
						if delErr := topicanalysis.DeleteTagEmbedding(c.Tag.ID); delErr != nil {
							logging.Warnf("Failed to delete embedding for child tag %d: %v", c.Tag.ID, delErr)
						}
					}
				}
				newTag, childErr := createChildOfAbstract(ctx, es, tag, category, kind, source, articleContext, string(aliasesJSON), result.AbstractTag)
				if childErr != nil {
					logging.Warnf("Failed to create child of abstract %d: %v", result.AbstractTag.ID, childErr)
					break
				}
				return newTag, nil

			case "no_match":
			}
		}
	}

	// Fallback: exact slug+category match (when embedding unavailable)
	// or creation path for no_match/candidates that fell through
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
			aJSON, _ := json.Marshal(tag.Aliases)
			dbTag.Aliases = string(aJSON)
		}
		dbTag.Kind = kind
		if err := database.DB.Save(&dbTag).Error; err != nil {
			return nil, err
		}
		// Backfill embedding if missing (fallback path)
		if es != nil {
			go ensureTagEmbedding(es, dbTag.ID)
		}
		go backfillTagDescription(dbTag.ID, dbTag.Label, dbTag.Category, dbTag.Description, articleContext)
		return &dbTag, nil
	}

	// Create new tag
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

	if articleContext != "" {
		go generateTagDescription(newTag.ID, tag.Label, category, articleContext)
	} else if es != nil {
		go generateAndSaveEmbedding(es, &newTag)
	}

	return &newTag, nil
}

// createChildOfAbstract creates a new tag as a child of an abstract parent,
// then deletes the new tag's embedding to prevent it from appearing in future matches.
func createChildOfAbstract(ctx context.Context, es *topicanalysis.EmbeddingService, tag topictypes.TopicTag, category, kind, source, articleContext, aliasesJSON string, abstractParent *models.TopicTag) (*models.TopicTag, error) {
	slug := topictypes.Slugify(tag.Label)
	newTag := models.TopicTag{
		Slug:        slug,
		Label:       tag.Label,
		Category:    category,
		Kind:        kind,
		Icon:        tag.Icon,
		Aliases:     aliasesJSON,
		IsCanonical: true,
		Source:      source,
	}
	if err := database.DB.Create(&newTag).Error; err != nil {
		return nil, fmt.Errorf("create child tag of abstract %d: %w", abstractParent.ID, err)
	}

	relation := models.TopicTagRelation{
		ParentID:     abstractParent.ID,
		ChildID:      newTag.ID,
		RelationType: "abstract",
	}
	if err := database.DB.Create(&relation).Error; err != nil {
		logging.Warnf("Failed to create parent-child relation: abstract %d -> child %d: %v", abstractParent.ID, newTag.ID, err)
		if es != nil {
			go generateAndSaveEmbedding(es, &newTag)
		}
	} else {
		logging.Infof("Child tag '%s' (id=%d) linked to abstract '%s' (id=%d)", newTag.Label, newTag.ID, abstractParent.Label, abstractParent.ID)
		var abstractSiblingCount int64
		database.DB.Model(&models.TopicTagRelation{}).
			Joins("JOIN topic_tags ON topic_tags.id = topic_tag_relations.child_id").
			Where("topic_tag_relations.parent_id = ? AND topic_tag_relations.relation_type = ? AND topic_tags.source = ?",
				abstractParent.ID, "abstract", "abstract").
			Count(&abstractSiblingCount)
		if abstractSiblingCount > 0 && es != nil {
			go generateAndSaveEmbedding(es, &newTag)
		}
	}

	if articleContext != "" {
		go generateTagDescription(newTag.ID, tag.Label, category, articleContext)
	} else if es != nil {
		go generateAndSaveEmbedding(es, &newTag)
	}

	return &newTag, nil
}

// backfillTagDescription triggers async description generation only if the tag currently has no description.
// Safe to call from any reuse path — skips silently if description already exists or context is empty.
func backfillTagDescription(tagID uint, label, category, currentDesc, articleContext string) {
	if currentDesc != "" || articleContext == "" {
		return
	}
	go generateTagDescription(tagID, label, category, articleContext)
}

// generateTagDescription generates a description for a tag via LLM and updates the database.
// Runs in a goroutine — never blocks tag creation. Failures are logged and swallowed.
func generateTagDescription(tagID uint, label, category, articleContext string) {
	defer func() {
		if r := recover(); r != nil {
			logging.Warnf("generateTagDescription panic for tag %d: %v", tagID, r)
		}
	}()

	if category == "person" {
		generatePersonTagDescription(tagID, label, articleContext)
		return
	}

	router := airouter.NewRouter()
	prompt := fmt.Sprintf(`Generate a concise description (1-2 sentences) for this tag.
Tag: %q
Category: %s
Context from article: %s

Description requirements:
- Must be in Chinese (中文)
- Objective, factual statement — no subjective opinions or qualifiers
- Must explain what the tag refers to, not just repeat the label
- Keep under 500 characters
- Examples of good descriptions:
  * Tag "ChatGPT" → "OpenAI开发的大型语言模型聊天机器人，基于GPT架构，支持多轮对话和文本生成"
  * Tag "苹果WWDC 2024" → "苹果公司于2024年6月举办的全球开发者大会，发布了Apple Intelligence等多项更新"
  * Tag "Sam Altman" → "OpenAI首席执行官，曾多次参与AI安全与治理相关的公开讨论"

Respond with JSON: {"description": "your answer"}`, label, category, articleContext)

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "你是一个标签分类助手，只输出合法JSON。"},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"description": {Type: "string", Description: "标签的中文客观描述，不超过500字符"},
			},
			Required: []string{"description"},
		},
		Temperature: func() *float64 { f := 0.3; return &f }(),
	}

	result, err := router.Chat(context.Background(), req)
	if err != nil {
		logging.Warnf("Description LLM call failed for tag %d: %v", tagID, err)
		return
	}

	var parsed struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil || parsed.Description == "" {
		logging.Warnf("Failed to parse description for tag %d", tagID)
		return
	}

	// Truncate to 500 chars (threat model T-08-01)
	desc := parsed.Description
	if len([]rune(desc)) > 500 {
		desc = string([]rune(desc)[:500])
	}

	if err := database.DB.Model(&models.TopicTag{}).Where("id = ?", tagID).Update("description", desc).Error; err != nil {
		logging.Warnf("Failed to save description for tag %d: %v", tagID, err)
		return
	}

	qs := getEmbeddingQueueService()
	if err := qs.Enqueue(tagID); err != nil {
		logging.Warnf("Failed to enqueue re-embedding after description update for tag %d: %v", tagID, err)
	}
}

func generatePersonTagDescription(tagID uint, label, articleContext string) {
	defer func() {
		if r := recover(); r != nil {
			logging.Warnf("generatePersonTagDescription panic for tag %d: %v", tagID, r)
		}
	}()

	router := airouter.NewRouter()

	prompt := fmt.Sprintf(`Given this person tag and article context, generate a description and extract structured attributes.

Tag: %q
Category: person
Context from article: %s

Description requirements:
- Must be in Chinese (中文)
- Objective, factual statement about WHO this person IS, not what they said or did in this specific article
- Keep under 500 characters
- Focus on: identity, position, affiliation

Structured attributes to extract:
- country: nationality or primary country of activity (中文, e.g. "美国", "中国")
- organization: primary organization or institution (中文)
- role: primary position or title (中文, e.g. "财政部长", "CEO")
- domains: areas of expertise or influence, as array of strings (中文, e.g. ["经济政策", "金融监管"])

Respond with JSON: {"description": "your answer", "person_attrs": {"country": "...", "organization": "...", "role": "...", "domains": [...]}}`, label, articleContext)

	req := airouter.ChatRequest{
		Capability: airouter.CapabilityTopicTagging,
		Messages: []airouter.Message{
			{Role: "system", Content: "你是一个标签分类助手，只输出合法JSON。"},
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
		JSONSchema: &airouter.JSONSchema{
			Type: "object",
			Properties: map[string]airouter.SchemaProperty{
				"description": {Type: "string", Description: "人物标签的中文客观描述"},
				"person_attrs": {
					Type: "object",
					Properties: map[string]airouter.SchemaProperty{
						"country":      {Type: "string", Description: "国籍或主要活动国家"},
						"organization": {Type: "string", Description: "主要组织或机构"},
						"role":         {Type: "string", Description: "主要职务或头衔"},
						"domains":      {Type: "array", Items: &airouter.SchemaProperty{Type: "string"}, Description: "专业领域"},
					},
				},
			},
			Required: []string{"description", "person_attrs"},
		},
		Temperature: func() *float64 { f := 0.3; return &f }(),
	}

	result, err := router.Chat(context.Background(), req)
	if err != nil {
		logging.Warnf("Person description LLM call failed for tag %d: %v", tagID, err)
		return
	}

	var parsed struct {
		Description string `json:"description"`
		PersonAttrs struct {
			Country      string   `json:"country"`
			Organization string   `json:"organization"`
			Role         string   `json:"role"`
			Domains      []string `json:"domains"`
		} `json:"person_attrs"`
	}
	if err := json.Unmarshal([]byte(result.Content), &parsed); err != nil || parsed.Description == "" {
		logging.Warnf("Failed to parse person description for tag %d", tagID)
		return
	}

	desc := parsed.Description
	if len([]rune(desc)) > 500 {
		desc = string([]rune(desc)[:500])
	}

	metadataMap := map[string]any{
		"country":      parsed.PersonAttrs.Country,
		"organization": parsed.PersonAttrs.Organization,
		"role":         parsed.PersonAttrs.Role,
		"domains":      parsed.PersonAttrs.Domains,
	}

	if err := database.DB.Model(&models.TopicTag{}).Where("id = ?", tagID).Updates(map[string]any{
		"description": desc,
		"metadata":    models.MetadataMap(metadataMap),
	}).Error; err != nil {
		logging.Warnf("Failed to save description+metadata for person tag %d: %v", tagID, err)
		return
	}

	qs := getEmbeddingQueueService()
	if err := qs.Enqueue(tagID); err != nil {
		logging.Warnf("Failed to enqueue re-embedding after person description update for tag %d: %v", tagID, err)
	}
}

// generateAndSaveEmbedding generates and stores an embedding for a tag in a goroutine.
// Uses recover to ensure embedding generation failure never blocks tag creation.
func generateAndSaveEmbedding(es *topicanalysis.EmbeddingService, tag *models.TopicTag) {
	qs := getEmbeddingQueueService()
	if err := qs.Enqueue(tag.ID); err != nil {
		logging.Warnf("Failed to enqueue embedding for tag %d: %v", tag.ID, err)
	}
}

// ensureTagEmbedding checks if a tag already has an embedding and generates one if missing.
// Used to backfill embeddings for tags created before the pgvector migration.
func ensureTagEmbedding(es *topicanalysis.EmbeddingService, tagID uint) {
	qs := getEmbeddingQueueService()
	if err := qs.Enqueue(tagID); err != nil {
		logging.Warnf("Failed to enqueue embedding for tag %d: %v", tagID, err)
	}
}

func NormalizeDisplayCategory(kind string, fallback string) string {
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

func NormalizeTopicKind(kind string, category string) string {
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
func dedupeTagsWithCategory(items []topictypes.TopicTag) []topictypes.TopicTag {
	best := make(map[string]topictypes.TopicTag)
	for _, item := range items {
		if item.Slug == "" {
			item.Slug = topictypes.Slugify(item.Label)
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

	result := make([]topictypes.TopicTag, 0, len(best))
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
func sortTagsByScore(tags []topictypes.TopicTag) {
	sort.SliceStable(tags, func(i, j int) bool {
		if tags[i].Score == tags[j].Score {
			return tags[i].Label < tags[j].Label
		}
		return tags[i].Score > tags[j].Score
	})
}

// topictypes.Slugify creates a URL-safe slug from a string (uses extractor.go implementation)

// dedupeTopics is kept for backward compatibility with extractor.go
func DedupeTopics(items []topictypes.TopicTag) []topictypes.TopicTag {
	return dedupeTagsWithCategory(items)
}
