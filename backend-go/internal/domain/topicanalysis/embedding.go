package topicanalysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topictypes"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"

	"log"

	"gorm.io/gorm"
)

var (
	ErrNoEmbeddingProvider = errors.New("no embedding provider configured")
	ErrEmbeddingFailed     = errors.New("failed to generate embedding")
)

// EmbeddingMatchThresholds defines similarity thresholds for tag matching
type EmbeddingMatchThresholds struct {
	// High similarity - auto-reuse existing tag
	HighSimilarity float64
	// Low similarity - auto-create new tag
	LowSimilarity float64
	// Middle band - requires AI judgment
	// Tags with similarity between LowSimilarity and HighSimilarity need AI decision
}

// DefaultThresholds provides sensible defaults for matching
var DefaultThresholds = EmbeddingMatchThresholds{
	HighSimilarity: 0.97, // Auto-reuse if similarity >= 0.97
	LowSimilarity:  0.78, // Auto-create if similarity < 0.78
}

// TagMatchResult represents a tag match result
type TagMatchResult struct {
	MatchType    string // "exact", "high_similarity", "low_similarity", "ai_judgment", "no_match"
	ExistingTag  *models.TopicTag
	Similarity   float64
	Candidates   []TagCandidate // For AI judgment mode
	ShouldCreate bool
}

// TagCandidate represents a candidate tag for AI judgment
type TagCandidate struct {
	Tag        *models.TopicTag
	Similarity float64
}

// EmbeddingService handles embedding generation and similarity matching
type EmbeddingService struct {
	router     *airouter.Router
	thresholds EmbeddingMatchThresholds
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService() *EmbeddingService {
	thresholds := DefaultThresholds
	configService := NewEmbeddingConfigService()
	if loaded, err := configService.LoadThresholds(); err == nil {
		thresholds = loaded
	}

	return &EmbeddingService{
		router:     airouter.NewRouter(),
		thresholds: thresholds,
	}
}

// NewEmbeddingServiceWithThresholds creates a service with custom thresholds
func NewEmbeddingServiceWithThresholds(thresholds EmbeddingMatchThresholds) *EmbeddingService {
	return &EmbeddingService{
		router:     airouter.NewRouter(),
		thresholds: thresholds,
	}
}

// GenerateEmbedding generates an embedding for a tag's text representation
func (s *EmbeddingService) GenerateEmbedding(ctx context.Context, tag *models.TopicTag) (*models.TopicTagEmbedding, error) {
	// Build text for embedding: label + aliases + category
	text := buildTagEmbeddingText(tag)
	textHash := hashText(text)

	// Use router with failover to generate embedding
	req := airouter.EmbeddingRequest{
		Input: []string{text},
	}
	result, err := s.router.Embed(ctx, req, airouter.CapabilityEmbedding)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
	}

	if len(result.Embeddings) == 0 || len(result.Embeddings[0]) == 0 {
		return nil, ErrEmbeddingFailed
	}

	// Store embedding as JSON (legacy) and as pgvector format
	vectorJSON, err := json.Marshal(result.Embeddings[0])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding: %w", err)
	}

	// Build pgvector format string: [0.1,0.2,0.3,...]
	pgVecStr := floatsToPgVector(result.Embeddings[0])

	embedding := &models.TopicTagEmbedding{
		TopicTagID:   tag.ID,
		Vector:       string(vectorJSON),
		EmbeddingVec: pgVecStr,
		Dimension:    result.Dimensions,
		Model:        result.Model,
		TextHash:     textHash,
	}

	return embedding, nil
}

// FindSimilarTags finds existing tags with similar embeddings using pgvector SQL
func (s *EmbeddingService) FindSimilarTags(ctx context.Context, tag *models.TopicTag, category string, limit int) ([]TagCandidate, error) {
	// Generate embedding for the new tag
	tagWithText := &models.TopicTag{
		Label:    tag.Label,
		Category: category,
		Aliases:  tag.Aliases,
	}

	embedding, err := s.GenerateEmbedding(ctx, tagWithText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Build pgvector format for the query vector
	var vector []float64
	if err := json.Unmarshal([]byte(embedding.Vector), &vector); err != nil {
		return nil, fmt.Errorf("failed to parse embedding vector: %w", err)
	}
	pgVecStr := floatsToPgVector(vector)

	// Use pgvector SQL cosine distance (<=>) for similarity search
	// Filter out merged tags (only match active tags)
	type simRow struct {
		TagID    uint    `gorm:"column:tag_id"`
		Distance float64 `gorm:"column:distance"`
	}
	var rows []simRow
	query := `
		SELECT t.id AS tag_id, e.embedding <=> ?::vector AS distance
		FROM topic_tag_embeddings e
		JOIN topic_tags t ON t.id = e.topic_tag_id
		WHERE t.category = ?
		  AND (t.status = 'active' OR t.status = '' OR t.status IS NULL)
		  AND e.embedding IS NOT NULL
		ORDER BY e.embedding <=> ?::vector
		LIMIT ?
	`
	if err := database.DB.Raw(query, pgVecStr, category, pgVecStr, limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to query similar tags: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil
	}

	// Load tags and compute similarity (1 - distance)
	tagIDs := make([]uint, len(rows))
	for i, r := range rows {
		tagIDs[i] = r.TagID
	}
	var tags []models.TopicTag
	if err := database.DB.Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("failed to load tags: %w", err)
	}
	tagMap := make(map[uint]*models.TopicTag, len(tags))
	for i := range tags {
		tagMap[tags[i].ID] = &tags[i]
	}

	candidates := make([]TagCandidate, 0, len(rows))
	for _, r := range rows {
		t, ok := tagMap[r.TagID]
		if !ok {
			continue
		}
		similarity := 1.0 - r.Distance
		candidates = append(candidates, TagCandidate{
			Tag:        t,
			Similarity: similarity,
		})
	}

	return candidates, nil
}

// TagMatch decides how to handle a candidate tag
func (s *EmbeddingService) TagMatch(ctx context.Context, label, category string, aliases string) (*TagMatchResult, error) {
	// Step 1: Check for exact match by slug in the same category (active tags only)
	slug := topictypes.Slugify(label)
	var existingTag models.TopicTag
	err := database.DB.Scopes(activeTagFilter).Where("slug = ? AND category = ?", slug, category).First(&existingTag).Error
	if err == nil {
		// Exact slug match
		return &TagMatchResult{
			MatchType:    "exact",
			ExistingTag:  &existingTag,
			Similarity:   1.0,
			ShouldCreate: false,
		}, nil
	}

	// Step 2: Check for alias match (active tags only)
	if aliases != "" {
		var aliasTags []models.TopicTag
		if err := database.DB.Scopes(activeTagFilter).Where("category = ?", category).Find(&aliasTags).Error; err == nil {
			for _, t := range aliasTags {
				if containsAlias(t.Aliases, label) {
					return &TagMatchResult{
						MatchType:    "exact",
						ExistingTag:  &t,
						Similarity:   1.0,
						ShouldCreate: false,
					}, nil
				}
			}
		}
	}

	// Step 3: Vector similarity matching
	candidate := &models.TopicTag{
		Label:    label,
		Category: category,
		Aliases:  aliases,
	}

	candidates, err := s.FindSimilarTags(ctx, candidate, category, 5)
	if err != nil {
		// Fall back to creating new tag on embedding failure
		return &TagMatchResult{
			MatchType:    "no_match",
			ShouldCreate: true,
		}, nil
	}

	if len(candidates) == 0 {
		return &TagMatchResult{
			MatchType:    "no_match",
			ShouldCreate: true,
		}, nil
	}

	best := candidates[0]

	// Step 4: Apply thresholds
	if best.Similarity >= s.thresholds.HighSimilarity {
		// High similarity - auto-reuse
		return &TagMatchResult{
			MatchType:    "high_similarity",
			ExistingTag:  best.Tag,
			Similarity:   best.Similarity,
			ShouldCreate: false,
		}, nil
	}

	if best.Similarity < s.thresholds.LowSimilarity {
		// Low similarity - auto-create
		return &TagMatchResult{
			MatchType:    "low_similarity",
			ExistingTag:  best.Tag,
			Similarity:   best.Similarity,
			ShouldCreate: true,
		}, nil
	}

	// Middle band - per CONV-03, skip AI judgment and create new tag
	// Candidates are still populated for logging/debugging purposes
	return &TagMatchResult{
		MatchType:    "ai_judgment",
		ExistingTag:  best.Tag,
		Similarity:   best.Similarity,
		Candidates:   candidates[:min(3, len(candidates))],
		ShouldCreate: true, // Degrades to creating new tag instead of AI judgment
	}, nil
}

// activeTagFilter returns a GORM scope that filters to only active (non-merged) tags.
// Used by query functions to exclude merged tags from match candidates.
// Includes empty-string status check for rows created before the migration ran.
func activeTagFilter(db *gorm.DB) *gorm.DB {
	return db.Where("status = ? OR status = ?", "active", "")
}

type mergeReembeddingEnqueuer interface {
	Enqueue(sourceTagID, targetTagID uint) error
}

var defaultMergeReembeddingQueueFactory = func() mergeReembeddingEnqueuer {
	return NewMergeReembeddingQueueService(nil)
}

var mergeReembeddingQueueFactory = defaultMergeReembeddingQueueFactory

// MergeTags merges sourceTag into targetTag within a database transaction.
// 1. Update all article_topic_tags from source to target (dedup conflicts)
// 2. Update all ai_summary_topics from source to target (dedup conflicts)
// 3. Set source tag status='merged', merged_into_id=target.ID
// 4. Delete source tag's embedding (stale after merge)
// 5. Recalculate target tag's feed_count
func MergeTags(sourceTagID, targetTagID uint) error {
	if sourceTagID == targetTagID {
		return fmt.Errorf("cannot merge tag into itself (id=%d)", sourceTagID)
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Step 1: Migrate article_topic_tags references
		// Find all source links first
		var sourceLinks []models.ArticleTopicTag
		if err := tx.Where("topic_tag_id = ?", sourceTagID).Find(&sourceLinks).Error; err != nil {
			return fmt.Errorf("find source article_topic_tags: %w", err)
		}

		for _, link := range sourceLinks {
			// Check if target already has a link for this article
			var existingCount int64
			if err := tx.Model(&models.ArticleTopicTag{}).
				Where("article_id = ? AND topic_tag_id = ?", link.ArticleID, targetTagID).
				Count(&existingCount).Error; err != nil {
				return fmt.Errorf("check existing article_topic_tag for article %d: %w", link.ArticleID, err)
			}

			if existingCount > 0 {
				// Target already covers this article — delete the source link
				if err := tx.Delete(&link).Error; err != nil {
					return fmt.Errorf("delete duplicate article_topic_tag %d: %w", link.ID, err)
				}
			} else {
				// No conflict — update the source link to point to target
				if err := tx.Model(&link).Update("topic_tag_id", targetTagID).Error; err != nil {
					return fmt.Errorf("update article_topic_tag %d to target: %w", link.ID, err)
				}
			}
		}

		// Step 2: Migrate ai_summary_topics references (same dedup logic)
		var sourceSummaryLinks []models.AISummaryTopic
		if err := tx.Where("topic_tag_id = ?", sourceTagID).Find(&sourceSummaryLinks).Error; err != nil {
			return fmt.Errorf("find source ai_summary_topics: %w", err)
		}

		for _, link := range sourceSummaryLinks {
			var existingCount int64
			if err := tx.Model(&models.AISummaryTopic{}).
				Where("summary_id = ? AND topic_tag_id = ?", link.SummaryID, targetTagID).
				Count(&existingCount).Error; err != nil {
				return fmt.Errorf("check existing ai_summary_topic for summary %d: %w", link.SummaryID, err)
			}

			if existingCount > 0 {
				if err := tx.Delete(&link).Error; err != nil {
					return fmt.Errorf("delete duplicate ai_summary_topic %d: %w", link.ID, err)
				}
			} else {
				if err := tx.Model(&link).Update("topic_tag_id", targetTagID).Error; err != nil {
					return fmt.Errorf("update ai_summary_topic %d to target: %w", link.ID, err)
				}
			}
		}

		// Step 3: Mark source tag as merged
		if err := tx.Model(&models.TopicTag{}).
			Where("id = ?", sourceTagID).
			Updates(map[string]interface{}{
				"status":         "merged",
				"merged_into_id": targetTagID,
			}).Error; err != nil {
			return fmt.Errorf("mark source tag as merged: %w", err)
		}

		// Step 4: Delete source tag's embedding (stale after merge)
		if err := tx.Where("topic_tag_id = ?", sourceTagID).Delete(&models.TopicTagEmbedding{}).Error; err != nil {
			return fmt.Errorf("delete source tag embedding: %w", err)
		}

		// Step 5: Recalculate target tag's feed_count
		if err := tx.Model(&models.TopicTag{}).
			Where("id = ?", targetTagID).
			Update("feed_count", tx.Model(&models.ArticleTopicTag{}).
				Select("COUNT(DISTINCT af.feed_id)").
				Joins("JOIN articles a ON a.id = article_topic_tags.article_id").
				Joins("JOIN article_feeds af ON af.article_id = a.id").
				Where("article_topic_tags.topic_tag_id = ?", targetTagID),
			).Error; err != nil {
			return fmt.Errorf("recalculate target feed_count: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if err := mergeReembeddingQueueFactory().Enqueue(sourceTagID, targetTagID); err != nil {
		return fmt.Errorf("enqueue merge re-embedding task: %w", err)
	}

	return nil
}

// SaveEmbedding saves or updates a tag's embedding in the database.
// If the actual vector dimension differs from the column definition, it alters the column type.
func (s *EmbeddingService) SaveEmbedding(embedding *models.TopicTagEmbedding) error {
	if embedding.Dimension > 0 && embedding.EmbeddingVec != "" {
		if err := ensureVectorDimension(embedding.Dimension); err != nil {
			return fmt.Errorf("ensure vector dimension %d: %w", embedding.Dimension, err)
		}
	}

	var existing models.TopicTagEmbedding
	err := database.DB.Where("topic_tag_id = ?", embedding.TopicTagID).First(&existing).Error

	if err == nil {
		embedding.ID = existing.ID
		return database.DB.Save(embedding).Error
	}

	return database.DB.Create(embedding).Error
}

// ensureVectorDimension checks if the embedding column matches the required dimension
// and alters it (plus the index) if not. Drops index before ALTER, recreates after.
// For dimensions > 2000, uses IVFFlat instead of HNSW (HNSW limit is 2000).
func ensureVectorDimension(dim int) error {
	var typeStr string
	if err := database.DB.Raw(`
		SELECT format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		WHERE c.relname = 'topic_tag_embeddings' AND a.attname = 'embedding'
	`).Row().Scan(&typeStr); err != nil {
		return nil
	}

	expected := fmt.Sprintf("vector(%d)", dim)
	if typeStr == expected {
		return nil
	}

	log.Printf("[INFO] Altering embedding column from %s to %s", typeStr, expected)

	// Drop index first — it depends on the column type
	_ = database.DB.Exec("DROP INDEX IF EXISTS idx_topic_tag_embeddings_embedding").Error

	if err := database.DB.Exec(fmt.Sprintf(
		"ALTER TABLE topic_tag_embeddings ALTER COLUMN embedding TYPE %s", expected,
	)).Error; err != nil {
		return fmt.Errorf("alter embedding column to %s: %w", expected, err)
	}

	// Recreate index — HNSW supports max 2000 dimensions
	if dim <= 2000 {
		if err := database.DB.Exec(
			"CREATE INDEX idx_topic_tag_embeddings_embedding ON topic_tag_embeddings USING hnsw (embedding vector_cosine_ops)",
		).Error; err != nil {
			log.Printf("[WARN] Failed to recreate HNSW index: %v", err)
		}
	} else {
		log.Printf("[INFO] Dimension %d exceeds HNSW limit (2000), skipping vector index", dim)
	}

	return nil
}

// GetEmbedding retrieves the embedding for a tag
func (s *EmbeddingService) GetEmbedding(tagID uint) (*models.TopicTagEmbedding, error) {
	var embedding models.TopicTagEmbedding
	err := database.DB.Where("topic_tag_id = ?", tagID).First(&embedding).Error
	if err != nil {
		return nil, err
	}
	return &embedding, nil
}

// BuildTextForEmbedding creates the text representation for embedding
func buildTagEmbeddingText(tag *models.TopicTag) string {
	text := tag.Label

	if tag.Aliases != "" {
		var aliases []string
		if err := json.Unmarshal([]byte(tag.Aliases), &aliases); err == nil {
			for _, alias := range aliases {
				text += " " + alias
			}
		} else {
			// Legacy: comma-separated
			text += " " + tag.Aliases
		}
	}

	text += " " + tag.Category

	return text
}

func hashText(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])
}

// floatsToPgVector converts a float64 slice to pgvector string format: [0.1,0.2,0.3]
func floatsToPgVector(v []float64) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%f", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func containsAlias(aliasesJSON, label string) bool {
	if aliasesJSON == "" {
		return false
	}

	var aliases []string
	if err := json.Unmarshal([]byte(aliasesJSON), &aliases); err != nil {
		// Try comma-separated
		aliases = splitByComma(aliasesJSON)
	}

	labelLower := lower(label)
	for _, alias := range aliases {
		if lower(alias) == labelLower {
			return true
		}
	}
	return false
}

func splitByComma(s string) []string {
	var result []string
	current := ""
	for _, r := range s {
		if r == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func lower(s string) string {
	return strings.ToLower(s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
