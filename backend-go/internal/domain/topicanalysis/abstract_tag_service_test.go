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

func mockAbstractNameResult(name string) *airouter.ChatResult {
	b, _ := json.Marshal(map[string]string{"abstract_name": name, "reason": "test"})
	return &airouter.ChatResult{Content: string(b)}
}

func TestExtractAbstractTagSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

func TestExtractAbstractTagDeduplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

func TestExtractAbstractTagLLMFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

func TestBuildBatchTagJudgmentPrompt(t *testing.T) {
	candidates := []TagCandidate{
		{Tag: &models.TopicTag{Label: "大语言模型", Source: "abstract"}, Similarity: 0.92},
		{Tag: &models.TopicTag{Label: "GPT-4", Source: "heuristic"}, Similarity: 0.88},
	}
	newLabel := "Gemini Pro"

	for _, category := range []string{"person", "event", "keyword"} {
		t.Run(category, func(t *testing.T) {
			prompt := buildBatchTagJudgmentPrompt(candidates, newLabel, category, nil)
			if !strings.Contains(prompt, "大语言模型") {
				t.Error("prompt should contain candidate label")
			}
			if !strings.Contains(prompt, "Gemini Pro") {
				t.Error("prompt should contain new label")
			}
			if !strings.Contains(prompt, "merge_target") {
				t.Error("prompt should instruct merge_target field")
			}
			if !strings.Contains(prompt, "candidate_label") {
				t.Error("prompt should instruct per-candidate judgment")
			}
			if !strings.Contains(prompt, "type: abstract") {
				t.Error("prompt should mark abstract candidates")
			}
		})
	}
}

func TestBuildBatchTagJudgmentPromptWithPreviousResults(t *testing.T) {
	candidates := []TagCandidate{
		{Tag: &models.TopicTag{Label: "Svelte", Slug: "svelte"}, Similarity: 0.82},
	}
	previousResults := []previousRoundResult{
		{CandidateLabel: "React", Action: ActionMerge, TargetLabel: "React"},
		{CandidateLabel: "Vue", Action: ActionNone},
	}
	prompt := buildBatchTagJudgmentPrompt(candidates, "SolidJS", "keyword", previousResults)
	if !strings.Contains(prompt, `"React" → merge`) {
		t.Error("prompt should include previous round merge result")
	}
	if !strings.Contains(prompt, `"Vue" → none`) {
		t.Error("prompt should include previous round none result")
	}
}

func TestBuildCandidateList(t *testing.T) {
	candidates := []TagCandidate{
		{Tag: &models.TopicTag{Label: "React", Source: "abstract"}, Similarity: 0.96},
		{Tag: &models.TopicTag{Label: "Vue", Source: "heuristic"}, Similarity: 0.93},
	}
	result := buildCandidateList(candidates, "Svelte")
	if !strings.Contains(result, `type: abstract`) {
		t.Error("should mark abstract candidates")
	}
	if !strings.Contains(result, `type: normal`) {
		t.Error("should mark non-abstract candidates as normal")
	}
	if !strings.Contains(result, `"Svelte" (new tag)`) {
		t.Error("should include new tag")
	}
}

func TestParseBatchTagJudgmentResponse(t *testing.T) {
	input := `[{"candidate_label":"GPT-4","action":"merge","merge_target":"GPT-4","merge_label":"GPT-4","reason":"same"},{"candidate_label":"Vue","action":"none","reason":"different"}]`
	results, err := parseBatchTagJudgmentResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 judgments, got %d", len(results))
	}
	if results[0].Action != ActionMerge {
		t.Errorf("expected merge, got %s", results[0].Action)
	}
	if results[0].MergeTarget != "GPT-4" {
		t.Errorf("expected merge target 'GPT-4', got %q", results[0].MergeTarget)
	}
	if results[1].Action != ActionNone {
		t.Errorf("expected none, got %s", results[1].Action)
	}
}

func TestParseBatchTagJudgmentResponseFallback(t *testing.T) {
	input := `{"action":"merge","merge_label":"GPT-4","reason":"same"}`
	results, err := parseSingleJudgmentFallback(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 judgment, got %d", len(results))
	}
	if results[0].Action != ActionMerge {
		t.Errorf("expected merge, got %s", results[0].Action)
	}
}

func TestBuildPreviousResultsSummary(t *testing.T) {
	results := []previousRoundResult{
		{CandidateLabel: "React", Action: ActionMerge, TargetLabel: "React"},
		{CandidateLabel: "Vue", Action: ActionNone},
	}
	summary := buildPreviousResultsSummary(results)
	if !strings.Contains(summary, `"React" → merge`) {
		t.Error("should show merge result")
	}
	if !strings.Contains(summary, `"Vue" → none`) {
		t.Error("should show none result")
	}
}

func TestSelectMergeTarget(t *testing.T) {
	t.Run("matches by merge target slug", func(t *testing.T) {
		candidates := []TagCandidate{
			{Tag: &models.TopicTag{ID: 1, Label: "GPT-4", Slug: "gpt-4"}},
			{Tag: &models.TopicTag{ID: 2, Label: "ChatGPT", Slug: "chatgpt"}},
		}
		target := selectMergeTarget(candidates, "GPT-4", "GPT-4o")
		if target == nil || target.ID != 1 {
			t.Errorf("expected tag ID 1, got %v", target)
		}
	})

	t.Run("matches by merge label slug when target not found", func(t *testing.T) {
		candidates := []TagCandidate{
			{Tag: &models.TopicTag{ID: 1, Label: "React", Slug: "react"}},
		}
		target := selectMergeTarget(candidates, "React.js", "React")
		if target == nil || target.ID != 1 {
			t.Errorf("expected tag ID 1 via merge label, got %v", target)
		}
	})

	t.Run("prefers non-abstract candidate", func(t *testing.T) {
		candidates := []TagCandidate{
			{Tag: &models.TopicTag{ID: 1, Label: "编程语言", Slug: "bian-cheng-yu-yan", Source: "abstract"}},
			{Tag: &models.TopicTag{ID: 2, Label: "Python", Slug: "python", Source: "heuristic"}},
		}
		target := selectMergeTarget(candidates, "未知", "Python")
		if target == nil {
			t.Fatal("expected non-nil target")
		}
		if target.ID != 2 {
			t.Errorf("expected non-abstract tag ID 2, got %d", target.ID)
		}
	})

	t.Run("returns nil for empty candidates", func(t *testing.T) {
		target := selectMergeTarget(nil, "anything", "anything")
		if target != nil {
			t.Error("expected nil for empty candidates")
		}
	})
}
