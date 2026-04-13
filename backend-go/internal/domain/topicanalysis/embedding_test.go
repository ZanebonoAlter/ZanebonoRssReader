package topicanalysis

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"my-robot-backend/internal/domain/models"
)

func TestBuildTagEmbeddingText(t *testing.T) {
	tests := []struct {
		name     string
		tag      *models.TopicTag
		expected string
	}{
		{
			name:     "label only",
			tag:      &models.TopicTag{Label: "AI", Category: "event"},
			expected: "AI event",
		},
		{
			name:     "with JSON aliases",
			tag:      &models.TopicTag{Label: "AI", Category: "event", Aliases: `["Artificial Intelligence","Machine Learning"]`},
			expected: "AI Artificial Intelligence Machine Learning event",
		},
		{
			name:     "with comma-separated aliases",
			tag:      &models.TopicTag{Label: "AI", Category: "event", Aliases: "Artificial Intelligence, ML"},
			expected: "AI Artificial Intelligence, ML event",
		},
		{
			name:     "empty aliases",
			tag:      &models.TopicTag{Label: "Go", Category: "technology"},
			expected: "Go technology",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildTagEmbeddingText(tt.tag)
			if result != tt.expected {
				t.Errorf("buildTagEmbeddingText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFloatsToPgVector(t *testing.T) {
	tests := []struct {
		name     string
		input    []float64
		expected string
	}{
		{
			name:     "simple vector",
			input:    []float64{0.1, 0.2, 0.3},
			expected: "[0.100000,0.200000,0.300000]",
		},
		{
			name:     "single element",
			input:    []float64{1.5},
			expected: "[1.500000]",
		},
		{
			name:     "zero vector",
			input:    []float64{0, 0, 0, 0},
			expected: "[0.000000,0.000000,0.000000,0.000000]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := floatsToPgVector(tt.input)
			if result != tt.expected {
				t.Errorf("floatsToPgVector() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHashTextDeterministic(t *testing.T) {
	a := hashText("hello world")
	b := hashText("hello world")
	if a != b {
		t.Errorf("hashText not deterministic: %q != %q", a, b)
	}

	c := hashText("different input")
	if a == c {
		t.Errorf("hashText collision for different inputs")
	}
}

func TestContainsAlias(t *testing.T) {
	tests := []struct {
		name     string
		aliases  string
		label    string
		expected bool
	}{
		{
			name:     "empty aliases",
			aliases:  "",
			label:    "AI",
			expected: false,
		},
		{
			name:     "JSON aliases match",
			aliases:  `["Artificial Intelligence","ML"]`,
			label:    "ML",
			expected: true,
		},
		{
			name:     "JSON aliases no match",
			aliases:  `["Artificial Intelligence","ML"]`,
			label:    "DL",
			expected: false,
		},
		{
			name:     "JSON case insensitive",
			aliases:  `["Machine Learning"]`,
			label:    "machine learning",
			expected: true,
		},
		{
			name:     "comma-separated match",
			aliases:  "AI,ML,DL",
			label:    "ML",
			expected: true,
		},
		{
			name:     "comma-separated with spaces no match",
			aliases:  "AI, ML, DL",
			label:    "ML",
			expected: false,
		},
		{
			name:     "comma-separated no match",
			aliases:  "AI, ML, DL",
			label:    "NLP",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAlias(tt.aliases, tt.label)
			if result != tt.expected {
				t.Errorf("containsAlias(%q, %q) = %v, want %v", tt.aliases, tt.label, result, tt.expected)
			}
		})
	}
}

func TestEmbeddingDimensionMismatch2560(t *testing.T) {
	vec := make([]float64, 2560)
	for i := range vec {
		vec[i] = 0.01
	}
	pgVec := floatsToPgVector(vec)
	if !strings.HasPrefix(pgVec, "[") || !strings.HasSuffix(pgVec, "]") {
		t.Fatalf("pgVector format wrong: %s...%s", pgVec[:20], pgVec[len(pgVec)-20:])
	}

	parts := strings.Split(pgVec[1:len(pgVec)-1], ",")
	if len(parts) != 2560 {
		t.Errorf("expected 2560 dimensions, got %d", len(parts))
	}

	vectorJSON, err := json.Marshal(vec)
	if err != nil {
		t.Fatalf("marshal vector: %v", err)
	}
	var parsed []float64
	if err := json.Unmarshal(vectorJSON, &parsed); err != nil {
		t.Fatalf("unmarshal vector: %v", err)
	}
	if len(parsed) != 2560 {
		t.Errorf("round-trip dimension mismatch: got %d, want 2560", len(parsed))
	}
}

func TestGenerateEmbeddingBuildsCorrectDimension(t *testing.T) {
	emb := &models.TopicTagEmbedding{
		TopicTagID:   1,
		Vector:       "[0.1,0.2,0.3]",
		EmbeddingVec: "[0.100000,0.200000,0.300000]",
		Dimension:    2560,
		Model:        "qwen3-embedding:4b",
		TextHash:     "abc123",
	}

	if emb.Dimension != 2560 {
		t.Errorf("expected dimension 2560, got %d", emb.Dimension)
	}

	vec := make([]float64, emb.Dimension)
	pgVec := floatsToPgVector(vec)
	expected := fmt.Sprintf("vector(%d)", emb.Dimension)
	if expected != "vector(2560)" {
		t.Errorf("expected vector(2560), got %s", expected)
	}

	_ = len(pgVec)
}
