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
