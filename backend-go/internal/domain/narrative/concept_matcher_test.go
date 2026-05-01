package narrative

import (
	"encoding/json"
	"testing"

	"my-robot-backend/internal/platform/airouter"
)

func TestCosineSimilarity_Identical(t *testing.T) {
	vec := []float64{0.1, 0.2, 0.3}
	sim, err := airouter.CosineSimilarity(vec, vec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sim < 0.999 {
		t.Errorf("expected similarity ~1.0 for identical vectors, got %f", sim)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{0, 1, 0}
	sim, err := airouter.CosineSimilarity(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sim != 0 {
		t.Errorf("expected 0 for orthogonal vectors, got %f", sim)
	}
}

func TestCosineSimilarity_Opposite(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{-1, 0, 0}
	sim, err := airouter.CosineSimilarity(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sim > -0.999 {
		t.Errorf("expected -1 for opposite vectors, got %f", sim)
	}
}

func TestCosineSimilarity_DimensionMismatch(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{1, 2}
	_, err := airouter.CosineSimilarity(a, b)
	if err == nil {
		t.Error("expected error for dimension mismatch")
	}
}

func TestFloatsToPgVectorStr(t *testing.T) {
	v := []float64{0.1, 0.2, 0.3}
	result := floatsToPgVectorStr(v)
	expected := "[0.100000,0.200000,0.300000]"
	if result != expected {
		t.Errorf("floatsToPgVectorStr: expected %q, got %q", expected, result)
	}
}

func TestParseConceptEmbeddingVec_Valid(t *testing.T) {
	vec, _ := json.Marshal([]float64{0.1, 0.2})
	s := string(vec)
	parsed, err := parseConceptEmbeddingVec(&s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed) != 2 || parsed[0] != 0.1 || parsed[1] != 0.2 {
		t.Errorf("unexpected parsed vector: %v", parsed)
	}
}

func TestParseConceptEmbeddingVec_Nil(t *testing.T) {
	_, err := parseConceptEmbeddingVec(nil)
	if err == nil {
		t.Error("expected error for nil")
	}
}

func TestMatchTagToConceptWithVec_NoConcepts(t *testing.T) {
	t.Skip("requires database connection")
}

func TestUnclassifiedBucket(t *testing.T) {
	ClearUnclassifiedBucket()

	tag := TagInput{ID: 1, Label: "test"}
	AddToUnclassifiedBucket(tag)
	AddToUnclassifiedBucket(TagInput{ID: 2, Label: "test2"})

	bucket := GetUnclassifiedBucket()
	if len(bucket) != 2 {
		t.Errorf("expected 2 unclassified tags, got %d", len(bucket))
	}

	ClearUnclassifiedBucket()
	bucket = GetUnclassifiedBucket()
	if len(bucket) != 0 {
		t.Errorf("expected empty bucket after clear, got %d", len(bucket))
	}
}
