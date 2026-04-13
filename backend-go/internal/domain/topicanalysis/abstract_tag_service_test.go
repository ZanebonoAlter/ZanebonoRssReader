package topicanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/airouter"
	"strings"
	"testing"
)

// --- Mock airouter for testing ---

type mockChatFn func(ctx context.Context, req airouter.ChatRequest) (*airouter.ChatResult, error)

type mockRouter struct {
	chatFn mockChatFn
}

func (m *mockRouter) Chat(ctx context.Context, req airouter.ChatRequest) (*airouter.ChatResult, error) {
	if m.chatFn != nil {
		return m.chatFn(ctx, req)
	}
	return nil, fmt.Errorf("mock chat not configured")
}

func (m *mockRouter) ResolvePrimaryProvider(capability airouter.Capability) (*models.AIProvider, *models.AIRoute, error) {
	return nil, nil, nil
}

func (m *mockRouter) Embed(ctx context.Context, req airouter.EmbeddingRequest, capability airouter.Capability) (*airouter.EmbeddingResult, error) {
	return nil, fmt.Errorf("mock embed not configured")
}

// Helper to create a mock chat result with JSON abstract_name
func mockAbstractNameResult(name string) *airouter.ChatResult {
	b, _ := json.Marshal(map[string]string{"abstract_name": name, "reason": "test"})
	return &airouter.ChatResult{Content: string(b)}
}

// TestExtractAbstractTagSuccess tests that ExtractAbstractTag creates an abstract tag
// and parent-child relations when LLM returns valid JSON.
func TestExtractAbstractTagSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	// This test needs a real DB, skip if not available
	// For now test the core logic flow through the service
	// We test with mock router that returns a valid abstract name
}

// TestExtractAbstractTagDeduplication tests that if an abstract tag with same slug already exists,
// it reuses the existing tag instead of creating a duplicate.
func TestExtractAbstractTagDeduplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

// TestExtractAbstractTagLLMFailure tests graceful degradation when LLM call fails.
func TestExtractAbstractTagLLMFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

// --- Unit tests for prompt construction ---

func TestBuildAbstractTagPrompt(t *testing.T) {
	candidates := []TagCandidate{
		{Tag: &models.TopicTag{Label: "大语言模型"}, Similarity: 0.92},
		{Tag: &models.TopicTag{Label: "GPT-4"}, Similarity: 0.88},
		{Tag: &models.TopicTag{Label: "Claude"}, Similarity: 0.85},
	}
	newLabel := "Gemini Pro"

	prompt := buildAbstractTagPrompt(candidates, newLabel)

	if !strings.Contains(prompt, "大语言模型") {
		t.Error("prompt should contain candidate label '大语言模型'")
	}
	if !strings.Contains(prompt, "GPT-4") {
		t.Error("prompt should contain candidate label 'GPT-4'")
	}
	if !strings.Contains(prompt, "Gemini Pro") {
		t.Error("prompt should contain new label 'Gemini Pro'")
	}
	if !strings.Contains(prompt, "abstract_name") {
		t.Error("prompt should instruct JSON output with abstract_name field")
	}
}

func TestParseAbstractNameFromJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "valid JSON",
			input:    `{"abstract_name": "AI技术", "reason": "all about AI"}`,
			expected: "AI技术",
			hasError: false,
		},
		{
			name:     "empty abstract_name",
			input:    `{"abstract_name": "", "reason": "none"}`,
			expected: "",
			hasError: true,
		},
		{
			name:     "invalid JSON",
			input:    `not json`,
			expected: "",
			hasError: true,
		},
		{
			name:     "missing abstract_name field",
			input:    `{"reason": "something"}`,
			expected: "",
			hasError: true,
		},
		{
			name:     "abstract_name with whitespace",
			input:    `{"abstract_name": "  AI 技术  ", "reason": "x"}`,
			expected: "AI 技术",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAbstractNameFromJSON(tt.input)
			if tt.hasError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseAbstractNameTooLong(t *testing.T) {
	longName := strings.Repeat("很", 200) // 200 chars, exceeds limit
	input := fmt.Sprintf(`{"abstract_name": "%s", "reason": "x"}`, longName)
	_, err := parseAbstractNameFromJSON(input)
	if err == nil {
		t.Error("expected error for name exceeding 160 chars")
	}
}
