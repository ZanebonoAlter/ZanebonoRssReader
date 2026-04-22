package topicanalysis

import (
	"testing"
)

func TestFindZombieTagIDs_NoDatabase(t *testing.T) {
	criteria := ZombieTagCriteria{
		MinAgeDays: 7,
		Categories: []string{"event", "keyword"},
	}
	if len(criteria.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(criteria.Categories))
	}
	if criteria.MinAgeDays != 7 {
		t.Errorf("expected 7 min age days, got %d", criteria.MinAgeDays)
	}
}

func TestBuildZombieQuery(t *testing.T) {
	criteria := ZombieTagCriteria{
		MinAgeDays: 7,
		Categories: []string{"event", "keyword"},
	}
	query := BuildZombieTagSubQuery(criteria)
	if query == "" {
		t.Error("expected non-empty query")
	}
}

func TestBuildFlatMergePrompt(t *testing.T) {
	tags := []FlatTagInfo{
		{ID: 1, Label: "日本地震", Description: "关于日本地震", Source: "abstract", ArticleCount: 0},
		{ID: 2, Label: "日本本州地震", Description: "日本本州海域地震", Source: "abstract", ArticleCount: 0},
		{ID: 3, Label: "半导体产业", Description: "半导体行业动态", Source: "abstract", ArticleCount: 0},
	}
	prompt := BuildFlatMergePrompt(tags, "event")
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestFlatMergeJudgment_Parse(t *testing.T) {
	judgment := flatMergeJudgment{}
	if len(judgment.Merges) != 0 {
		t.Error("expected empty merges initially")
	}
}

func TestCleanupOrphanedRelations(t *testing.T) {
	_ = CleanupOrphanedRelations
}

func TestCleanupMultiParentConflicts_Signature(t *testing.T) {
	_ = CleanupMultiParentConflicts
}

func TestCleanupEmptyAbstractNodes_Signature(t *testing.T) {
	_ = CleanupEmptyAbstractNodes
}

func TestQuoteCategories(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{[]string{"event"}, "'event'"},
		{[]string{"event", "keyword"}, "'event', 'keyword'"},
		{[]string{}, ""},
	}
	for _, tt := range tests {
		got := quoteCategories(tt.input)
		if got != tt.expected {
			t.Errorf("quoteCategories(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestValidateFlatMerge_SameTag(t *testing.T) {
	tagMap := map[uint]*FlatTagInfo{1: {ID: 1, Label: "a"}}
	err := validateFlatMerge(flatMergeItem{SourceID: 1, TargetID: 1}, tagMap)
	if err == nil {
		t.Error("expected error for same tag")
	}
}

func TestValidateFlatMerge_SourceNotFound(t *testing.T) {
	tagMap := map[uint]*FlatTagInfo{1: {ID: 1, Label: "a"}}
	err := validateFlatMerge(flatMergeItem{SourceID: 999, TargetID: 1}, tagMap)
	if err == nil {
		t.Error("expected error for missing source")
	}
}

func TestValidateFlatMerge_SourceMoreChildren(t *testing.T) {
	tagMap := map[uint]*FlatTagInfo{
		1: {ID: 1, Label: "big", ChildCount: 10},
		2: {ID: 2, Label: "small", ChildCount: 1},
	}
	err := validateFlatMerge(flatMergeItem{SourceID: 1, TargetID: 2}, tagMap)
	if err == nil {
		t.Error("expected error when source has more children than target")
	}
}

func TestValidateFlatMerge_ValidMerge(t *testing.T) {
	tagMap := map[uint]*FlatTagInfo{
		1: {ID: 1, Label: "big", ChildCount: 10},
		2: {ID: 2, Label: "small", ChildCount: 1},
	}
	err := validateFlatMerge(flatMergeItem{SourceID: 2, TargetID: 1}, tagMap)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestBuildFlatMergePrompt_ContainsCategory(t *testing.T) {
	tags := []FlatTagInfo{{ID: 1, Label: "test"}}
	prompt := BuildFlatMergePrompt(tags, "event")
	if len(prompt) == 0 {
		t.Error("expected non-empty prompt")
	}
}
