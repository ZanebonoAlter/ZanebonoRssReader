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
	client     *airouter.EmbeddingClient
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
		client:     airouter.NewEmbeddingClient(),
		thresholds: thresholds,
	}
}

// NewEmbeddingServiceWithThresholds creates a service with custom thresholds
func NewEmbeddingServiceWithThresholds(thresholds EmbeddingMatchThresholds) *EmbeddingService {
	return &EmbeddingService{
		router:     airouter.NewRouter(),
		client:     airouter.NewEmbeddingClient(),
		thresholds: thresholds,
	}
}

// GenerateEmbedding generates an embedding for a tag's text representation
func (s *EmbeddingService) GenerateEmbedding(ctx context.Context, tag *models.TopicTag) (*models.TopicTagEmbedding, error) {
	provider, _, err := s.router.ResolvePrimaryProvider(airouter.CapabilityEmbedding)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoEmbeddingProvider, err)
	}

	// Build text for embedding: label + aliases + category
	text := buildTagEmbeddingText(tag)
	textHash := hashText(text)

	// Use the provider's model or default
	model := getEmbeddingModel(provider)

	result, err := s.client.Embed(ctx, *provider, []string{text}, model)
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
	pgVecStr := floatsToPgVector(nil)
	var vector []float64
	if err := json.Unmarshal([]byte(embedding.Vector), &vector); err != nil {
		return nil, fmt.Errorf("failed to parse embedding vector: %w", err)
	}
	pgVecStr = floatsToPgVector(vector)

	// Use pgvector SQL cosine distance (<=>) for similarity search
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
	// Step 1: Check for exact match by slug in the same category
	slug := topictypes.Slugify(label)
	var existingTag models.TopicTag
	err := database.DB.Where("slug = ? AND category = ?", slug, category).First(&existingTag).Error
	if err == nil {
		// Exact slug match
		return &TagMatchResult{
			MatchType:    "exact",
			ExistingTag:  &existingTag,
			Similarity:   1.0,
			ShouldCreate: false,
		}, nil
	}

	// Step 2: Check for alias match
	if aliases != "" {
		var aliasTags []models.TopicTag
		if err := database.DB.Where("category = ?", category).Find(&aliasTags).Error; err == nil {
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

// SaveEmbedding saves or updates a tag's embedding in the database
func (s *EmbeddingService) SaveEmbedding(embedding *models.TopicTagEmbedding) error {
	// Check if embedding exists for this tag
	var existing models.TopicTagEmbedding
	err := database.DB.Where("topic_tag_id = ?", embedding.TopicTagID).First(&existing).Error

	if err == nil {
		// Update existing
		embedding.ID = existing.ID
		return database.DB.Save(embedding).Error
	}

	// Create new
	return database.DB.Create(embedding).Error
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

func getEmbeddingModel(provider *models.AIProvider) string {
	// Use the model configured in the airouter provider.
	// If the provider has an embedding model set, use it; otherwise fall back.
	if provider.Model != "" {
		return provider.Model
	}
	return "text-embedding-ada-002"
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
	// Simple lowercase
	result := make([]byte, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = byte(r + 32)
		} else {
			result[i] = byte(r)
		}
	}
	return string(result)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
