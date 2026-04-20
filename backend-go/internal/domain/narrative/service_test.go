package narrative

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func setupServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(
		&models.NarrativeSummary{},
		&models.TopicTag{},
		&models.TopicTagRelation{},
		&models.ArticleTopicTag{},
		&models.Article{},
		&models.Feed{},
		&models.Category{},
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	database.DB = db
	t.Cleanup(func() { database.DB = nil })
	return db
}

func TestResolveGlobalGeneration_NoPrevious(t *testing.T) {
	setupServiceTestDB(t)

	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	gen := resolveGlobalGeneration(date)
	if gen != 0 {
		t.Errorf("expected generation 0 with no previous, got %d", gen)
	}
}

func TestResolveGlobalGeneration_WithPrevious(t *testing.T) {
	db := setupServiceTestDB(t)

	yesterday := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	db.Create(&models.NarrativeSummary{
		Title:      "Prev Global",
		Summary:    "Yesterday",
		Status:     "continuing",
		Period:     "daily",
		PeriodDate: yesterday,
		Generation: 2,
		Source:     "ai",
		ScopeType:  models.NarrativeScopeTypeGlobal,
	})

	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	gen := resolveGlobalGeneration(date)
	if gen != 3 {
		t.Errorf("expected generation 3 (max_gen 2 + 1), got %d", gen)
	}
}

func TestMarkEndedGlobalNarratives_NoIntersection(t *testing.T) {
	db := setupServiceTestDB(t)

	yesterday := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	prevTagIDs, _ := json.Marshal([]uint{10, 20})
	db.Create(&models.NarrativeSummary{
		Title:          "Prev Global",
		Summary:        "Yesterday",
		Status:         "continuing",
		Period:         "daily",
		PeriodDate:     yesterday,
		Generation:     1,
		Source:         "ai",
		ScopeType:      models.NarrativeScopeTypeGlobal,
		RelatedTagIDs:  string(prevTagIDs),
	})

	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	currentOutputs := []NarrativeOutput{
		{Title: "Today", Summary: "Today", Status: "emerging", RelatedTagIDs: []uint{99}},
	}
	prevGlobal := []PreviousNarrative{
		{ID: 1, Title: "Prev Global", Summary: "Yesterday", Status: "continuing", Generation: 1},
	}

	markEndedGlobalNarratives(date, currentOutputs, prevGlobal)

	var updated models.NarrativeSummary
	db.First(&updated, 1)
	if updated.Status != models.NarrativeStatusEnding {
		t.Errorf("expected status 'ending', got %q", updated.Status)
	}
}

func TestMarkEndedGlobalNarratives_HasIntersection(t *testing.T) {
	db := setupServiceTestDB(t)

	yesterday := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	prevTagIDs, _ := json.Marshal([]uint{10, 20})
	db.Create(&models.NarrativeSummary{
		Title:          "Prev Global",
		Summary:        "Yesterday",
		Status:         "continuing",
		Period:         "daily",
		PeriodDate:     yesterday,
		Generation:     1,
		Source:         "ai",
		ScopeType:      models.NarrativeScopeTypeGlobal,
		RelatedTagIDs:  string(prevTagIDs),
	})

	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	currentOutputs := []NarrativeOutput{
		{Title: "Today", Summary: "Today", Status: "emerging", RelatedTagIDs: []uint{10}},
	}
	prevGlobal := []PreviousNarrative{
		{ID: 1, Title: "Prev Global", Summary: "Yesterday", Status: "continuing", Generation: 1},
	}

	markEndedGlobalNarratives(date, currentOutputs, prevGlobal)

	var updated models.NarrativeSummary
	db.First(&updated, 1)
	if updated.Status == models.NarrativeStatusEnding {
		t.Error("narrative with tag intersection should NOT be marked as ending")
	}
}

func TestMarkEndedGlobalNarratives_NoPrevious(t *testing.T) {
	setupServiceTestDB(t)

	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	currentOutputs := []NarrativeOutput{
		{Title: "Today", Summary: "Today", Status: "emerging", RelatedTagIDs: []uint{1}},
	}

	markEndedGlobalNarratives(date, currentOutputs, nil)
}
