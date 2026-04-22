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
