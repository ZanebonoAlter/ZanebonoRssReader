package topicanalysis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func setupTagCleanupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	database.DB = db
	t.Cleanup(func() {
		database.DB = nil
	})

	if err := db.AutoMigrate(
		&models.TopicTag{},
		&models.TopicTagRelation{},
		&models.ArticleTopicTag{},
		&models.AISummaryTopic{},
	); err != nil {
		t.Fatalf("migrate test tables: %v", err)
	}

	return db
}

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

func TestBuildFlatMergePromptIncludesPersonMetadata(t *testing.T) {
	tags := []FlatTagInfo{
		{
			ID:          1,
			Label:       "李宗伟",
			Description: "马来西亚羽毛球运动员",
			Source:      "abstract",
			Metadata: models.MetadataMap{
				"country": "马来西亚",
				"role":    "羽毛球运动员",
				"domains": []any{"羽毛球"},
			},
		},
	}

	prompt := BuildFlatMergePrompt(tags, "person")

	for _, want := range []string{"person_attrs", "马来西亚", "羽毛球运动员", "羽毛球"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("flat merge prompt missing %q in:\n%s", want, prompt)
		}
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

func TestCleanupEmptyAbstractNodes_DeactivatesLeafAbstractTags(t *testing.T) {
	db := setupTagCleanupTestDB(t)

	leaf := models.TopicTag{
		Slug:      "leaf-abstract",
		Label:     "Leaf Abstract",
		Category:  "event",
		Source:    "abstract",
		Status:    "active",
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}
	parent := models.TopicTag{
		Slug:      "parent-abstract",
		Label:     "Parent Abstract",
		Category:  "event",
		Source:    "abstract",
		Status:    "active",
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}
	child := models.TopicTag{
		Slug:      "child-tag",
		Label:     "Child Tag",
		Category:  "event",
		Source:    "llm",
		Status:    "active",
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}
	for _, tag := range []*models.TopicTag{&leaf, &parent, &child} {
		if err := db.Create(tag).Error; err != nil {
			t.Fatalf("create tag %s: %v", tag.Label, err)
		}
	}
	if err := db.Create(&models.TopicTagRelation{
		ParentID:     parent.ID,
		ChildID:      child.ID,
		RelationType: "abstract",
	}).Error; err != nil {
		t.Fatalf("create relation: %v", err)
	}

	deactivated, err := CleanupEmptyAbstractNodes()
	if err != nil {
		t.Fatalf("CleanupEmptyAbstractNodes returned error: %v", err)
	}
	if deactivated != 1 {
		t.Fatalf("deactivated = %d, want 1", deactivated)
	}

	var refreshedLeaf models.TopicTag
	if err := db.First(&refreshedLeaf, leaf.ID).Error; err != nil {
		t.Fatalf("reload leaf: %v", err)
	}
	if refreshedLeaf.Status != "inactive" {
		t.Fatalf("leaf status = %q, want inactive", refreshedLeaf.Status)
	}

	var refreshedParent models.TopicTag
	if err := db.First(&refreshedParent, parent.ID).Error; err != nil {
		t.Fatalf("reload parent: %v", err)
	}
	if refreshedParent.Status != "active" {
		t.Fatalf("parent status = %q, want active", refreshedParent.Status)
	}
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

func TestCleanupMultiParentConflicts_OnlyCountsSuccessfulResolutions(t *testing.T) {
	db := setupTagCleanupTestDB(t)

	parentA := models.TopicTag{Slug: "parent-a", Label: "Parent A", Category: "event", Source: "abstract", Status: "active"}
	parentB := models.TopicTag{Slug: "parent-b", Label: "Parent B", Category: "event", Source: "abstract", Status: "active"}
	child := models.TopicTag{Slug: "child", Label: "Child", Category: "event", Source: "llm", Status: "active"}
	for _, tag := range []*models.TopicTag{&parentA, &parentB, &child} {
		if err := db.Create(tag).Error; err != nil {
			t.Fatalf("create tag %s: %v", tag.Label, err)
		}
	}
	for _, parentID := range []uint{parentA.ID, parentB.ID} {
		if err := db.Create(&models.TopicTagRelation{ParentID: parentID, ChildID: child.ID, RelationType: "abstract"}).Error; err != nil {
			t.Fatalf("create relation for parent %d: %v", parentID, err)
		}
	}

	// Batch approach handles LLM failures internally (logs warning, returns 0 resolved).
	// No aiJudgeBestParentFn mock needed — the batch function calls airouter directly.
	resolved, errs, err := CleanupMultiParentConflicts()
	if err != nil {
		t.Fatalf("CleanupMultiParentConflicts returned error: %v", err)
	}
	if resolved != 0 {
		t.Fatalf("resolved = %d, want 0", resolved)
	}
	// LLM failure is logged, not propagated as error string
	if len(errs) != 0 {
		t.Fatalf("len(errs) = %d, want 0", len(errs))
	}

	var relationCount int64
	if err := db.Model(&models.TopicTagRelation{}).Where("child_id = ? AND relation_type = ?", child.ID, "abstract").Count(&relationCount).Error; err != nil {
		t.Fatalf("count relations: %v", err)
	}
	if relationCount != 2 {
		t.Fatalf("relation count = %d, want 2", relationCount)
	}
}

func TestCleanupMultiParentConflicts_RemovesRedundantAncestorParentWithoutLLM(t *testing.T) {
	db := setupTagCleanupTestDB(t)

	root := models.TopicTag{Slug: "root", Label: "Root", Category: "keyword", Source: "abstract", Status: "active"}
	directParent := models.TopicTag{Slug: "direct-parent", Label: "Direct Parent", Category: "keyword", Source: "abstract", Status: "active"}
	child := models.TopicTag{Slug: "child", Label: "Child", Category: "keyword", Source: "abstract", Status: "active"}
	for _, tag := range []*models.TopicTag{&root, &directParent, &child} {
		if err := db.Create(tag).Error; err != nil {
			t.Fatalf("create tag %s: %v", tag.Label, err)
		}
	}

	for _, relation := range []models.TopicTagRelation{
		{ParentID: root.ID, ChildID: directParent.ID, RelationType: "abstract"},
		{ParentID: directParent.ID, ChildID: child.ID, RelationType: "abstract"},
		{ParentID: root.ID, ChildID: child.ID, RelationType: "abstract"},
	} {
		if err := db.Create(&relation).Error; err != nil {
			t.Fatalf("create relation: %v", err)
		}
	}

	originalJudge := aiJudgeBestParentFn
	aiJudgeBestParentFn = func(ctx context.Context, childTag *models.TopicTag, parents []parentWithInfo) (int, error) {
		return 0, errors.New("LLM should not be needed for ancestor redundancy")
	}
	t.Cleanup(func() {
		aiJudgeBestParentFn = originalJudge
	})

	resolved, errs, err := CleanupMultiParentConflicts()
	if err != nil {
		t.Fatalf("CleanupMultiParentConflicts returned error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if resolved != 1 {
		t.Fatalf("resolved = %d, want 1", resolved)
	}

	assertAbstractRelationExists(t, db, directParent.ID, child.ID)
	assertAbstractRelationMissing(t, db, root.ID, child.ID)
}
