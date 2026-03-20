package topicanalysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

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
	return &EmbeddingService{
		router:     airouter.NewRouter(),
		client:     airouter.NewEmbeddingClient(),
		thresholds: DefaultThresholds,
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

	// Store embedding as JSON
	vectorJSON, err := json.Marshal(result.Embeddings[0])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding: %w", err)
	}

	embedding := &models.TopicTagEmbedding{
		TopicTagID: tag.ID,
		Vector:     string(vectorJSON),
		Dimension:  result.Dimensions,
		Model:      result.Model,
		TextHash:   textHash,
	}

	return embedding, nil
}

// FindSimilarTags finds existing tags with similar embeddings
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

	var vector []float64
	if err := json.Unmarshal([]byte(embedding.Vector), &vector); err != nil {
		return nil, fmt.Errorf("failed to parse embedding vector: %w", err)
	}

	// Load existing embeddings for the same category
	var existingEmbeddings []models.TopicTagEmbedding
	if err := database.DB.
		Joins("JOIN topic_tags ON topic_tags.id = topic_tag_embeddings.topic_tag_id").
		Where("topic_tags.category = ?", category).
		Preload("TopicTag").
		Find(&existingEmbeddings).Error; err != nil {
		return nil, fmt.Errorf("failed to load existing embeddings: %w", err)
	}

	candidates := make([]TagCandidate, 0, len(existingEmbeddings))
	for _, existing := range existingEmbeddings {
		if existing.TopicTag == nil {
			continue
		}

		var existingVector []float64
		if err := json.Unmarshal([]byte(existing.Vector), &existingVector); err != nil {
			continue
		}

		similarity, err := airouter.CosineSimilarity(vector, existingVector)
		if err != nil {
			continue
		}

		candidates = append(candidates, TagCandidate{
			Tag:        existing.TopicTag,
			Similarity: similarity,
		})
	}

	// Sort by similarity descending
	sortCandidatesBySimilarity(candidates)

	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
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

	// Middle band - AI judgment needed
	return &TagMatchResult{
		MatchType:    "ai_judgment",
		ExistingTag:  best.Tag,
		Similarity:   best.Similarity,
		Candidates:   candidates[:min(3, len(candidates))],
		ShouldCreate: false, // AI will decide
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
	// Check if provider has a specific embedding model configured
	// For now, default to text-embedding-ada-002
	// Provider.Model might be a chat model, so we use the default embedding model
	return "text-embedding-ada-002"
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

func sortCandidatesBySimilarity(candidates []TagCandidate) {
	// Simple bubble sort for small slices
	n := len(candidates)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if candidates[j].Similarity < candidates[j+1].Similarity {
				candidates[j], candidates[j+1] = candidates[j+1], candidates[j]
			}
		}
	}
}
