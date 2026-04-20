package narrative

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func setupCollectorTestDB(t *testing.T) *gorm.DB {
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
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	database.DB = db
	t.Cleanup(func() { database.DB = nil })
	return db
}

func TestCollectTagInputs_NoActiveTags(t *testing.T) {
	db := setupCollectorTestDB(t)

	_ = db

	date := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	inputs, err := CollectTagInputs(date)
	if err != nil {
		t.Fatalf("CollectTagInputs returned error: %v", err)
	}
	if len(inputs) != 0 {
		t.Fatalf("expected no inputs for empty DB, got %d", len(inputs))
	}
}

func TestCollectTagInputs_WithRootAbstractTag(t *testing.T) {
	db := setupCollectorTestDB(t)

	parentTag := models.TopicTag{Label: "AI", Slug: "ai", Category: "keyword", Status: "active", Source: "abstract"}
	if err := db.Create(&parentTag).Error; err != nil {
		t.Fatalf("create parent tag: %v", err)
	}

	childTag := models.TopicTag{Label: "LLM", Slug: "llm", Category: "keyword", Status: "active", Source: "llm"}
	if err := db.Create(&childTag).Error; err != nil {
		t.Fatalf("create child tag: %v", err)
	}

	relation := models.TopicTagRelation{ParentID: parentTag.ID, ChildID: childTag.ID, RelationType: "abstract"}
	if err := db.Create(&relation).Error; err != nil {
		t.Fatalf("create relation: %v", err)
	}

	feed := models.Feed{Title: "Test Feed", URL: "https://example.com/feed"}
	if err := db.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}

	pubDate := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	article := models.Article{FeedID: feed.ID, Title: "Test Article", Link: "https://example.com/art1", PubDate: &pubDate}
	if err := db.Create(&article).Error; err != nil {
		t.Fatalf("create article: %v", err)
	}

	articleTag := models.ArticleTopicTag{ArticleID: article.ID, TopicTagID: parentTag.ID}
	if err := db.Create(&articleTag).Error; err != nil {
		t.Fatalf("create article-topic tag: %v", err)
	}

	inputs, err := CollectTagInputs(pubDate)
	if err != nil {
		t.Fatalf("CollectTagInputs returned error: %v", err)
	}

	found := false
	for _, inp := range inputs {
		if inp.ID == parentTag.ID && inp.IsAbstract && inp.Source == "abstract" {
			found = true
			if inp.ArticleCount != 1 {
				t.Errorf("expected article_count=1 for root abstract tag, got %d", inp.ArticleCount)
			}
		}
	}
	if !found {
		t.Fatalf("expected root abstract tag in inputs, got %+v", inputs)
	}
}

func TestCollectTagInputs_ExcludesChildOfAbstract(t *testing.T) {
	db := setupCollectorTestDB(t)

	root := models.TopicTag{Label: "Tech", Slug: "tech", Category: "keyword", Status: "active", Source: "abstract"}
	if err := db.Create(&root).Error; err != nil {
		t.Fatalf("create root tag: %v", err)
	}

	child := models.TopicTag{Label: "Cloud", Slug: "cloud", Category: "keyword", Status: "active", Source: "abstract"}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child tag: %v", err)
	}

	grandchild := models.TopicTag{Label: "AWS", Slug: "aws", Category: "keyword", Status: "active", Source: "llm"}
	if err := db.Create(&grandchild).Error; err != nil {
		t.Fatalf("create grandchild tag: %v", err)
	}

	db.Create(&models.TopicTagRelation{ParentID: root.ID, ChildID: child.ID, RelationType: "abstract"})
	db.Create(&models.TopicTagRelation{ParentID: child.ID, ChildID: grandchild.ID, RelationType: "abstract"})

	feed := models.Feed{Title: "Feed", URL: "https://example.com/f2"}
	db.Create(&feed)

	pubDate := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	article := models.Article{FeedID: feed.ID, Title: "Article", Link: "https://example.com/a2", PubDate: &pubDate}
	db.Create(&article)
	db.Create(&models.ArticleTopicTag{ArticleID: article.ID, TopicTagID: root.ID})

	inputs, err := CollectTagInputs(pubDate)
	if err != nil {
		t.Fatalf("CollectTagInputs returned error: %v", err)
	}

	for _, inp := range inputs {
		if inp.ID == child.ID {
			t.Errorf("child-of-abstract tag should not appear as root, got %+v", inp)
		}
	}

	foundRoot := false
	for _, inp := range inputs {
		if inp.ID == root.ID {
			foundRoot = true
		}
	}
	if !foundRoot {
		t.Fatalf("expected root abstract tag (Tech) in inputs, got %+v", inputs)
	}
}

func TestCollectTagInputs_UnclassifiedTags(t *testing.T) {
	db := setupCollectorTestDB(t)

	tag := models.TopicTag{Label: "Go", Slug: "go-lang", Category: "keyword", Status: "active", Source: "llm", QualityScore: 0.9}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}

	feed := models.Feed{Title: "Feed", URL: "https://example.com/f3"}
	db.Create(&feed)

	pubDate := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	article := models.Article{FeedID: feed.ID, Title: "Go Article", Link: "https://example.com/a3", PubDate: &pubDate}
	db.Create(&article)
	db.Create(&models.ArticleTopicTag{ArticleID: article.ID, TopicTagID: tag.ID})

	inputs, err := CollectTagInputs(pubDate)
	if err != nil {
		t.Fatalf("CollectTagInputs returned error: %v", err)
	}

	found := false
	for _, inp := range inputs {
		if inp.ID == tag.ID {
			found = true
			if inp.IsAbstract {
				t.Errorf("expected IsAbstract=false for unclassified tag, got true")
			}
			if inp.Source != "llm" {
				t.Errorf("expected source=llm, got %s", inp.Source)
			}
		}
	}
	if !found {
		t.Fatalf("expected unclassified tag (Go) in inputs, got %+v", inputs)
	}
}

func TestCollectPreviousNarratives_NoData(t *testing.T) {
	setupCollectorTestDB(t)

	date := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	result, err := CollectPreviousNarratives(date, "", nil)
	if err != nil {
		t.Fatalf("CollectPreviousNarratives returned error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d items", len(result))
	}
}

func TestCollectPreviousNarratives_WithData(t *testing.T) {
	db := setupCollectorTestDB(t)

	day1 := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)

	narratives := []models.NarrativeSummary{
		{Title: "Story A", Summary: "Summary A", Status: "emerging", Period: "daily", PeriodDate: day1, Generation: 0, Source: "ai"},
		{Title: "Story B", Summary: "Summary B", Status: "continuing", Period: "daily", PeriodDate: day1, Generation: 1, Source: "ai"},
	}
	for _, n := range narratives {
		if err := db.Create(&n).Error; err != nil {
			t.Fatalf("create narrative: %v", err)
		}
	}

	result, err := CollectPreviousNarratives(day2, "", nil)
	if err != nil {
		t.Fatalf("CollectPreviousNarratives returned error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 narratives, got %d", len(result))
	}

	foundA := false
	foundB := false
	for _, r := range result {
		if r.Title == "Story A" {
			foundA = true
			if r.Status != "emerging" {
				t.Errorf("Story A: expected status emerging, got %s", r.Status)
			}
			if r.Generation != 0 {
				t.Errorf("Story A: expected generation 0, got %d", r.Generation)
			}
		}
		if r.Title == "Story B" {
			foundB = true
			if r.Status != "continuing" {
				t.Errorf("Story B: expected status continuing, got %s", r.Status)
			}
			if r.Generation != 1 {
				t.Errorf("Story B: expected generation 1, got %d", r.Generation)
			}
		}
	}
	if !foundA || !foundB {
		t.Fatalf("missing narratives: foundA=%v foundB=%v in %+v", foundA, foundB, result)
	}
}

func TestCollectPreviousNarratives_OnlyLooksAtYesterday(t *testing.T) {
	db := setupCollectorTestDB(t)

	day := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	db.Create(&models.NarrativeSummary{
		Title: "Old Story", Summary: "Old", Status: "ending", Period: "daily", PeriodDate: day, Generation: 0, Source: "ai",
	})

	queryDate := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	result, err := CollectPreviousNarratives(queryDate, "", nil)
	if err != nil {
		t.Fatalf("CollectPreviousNarratives returned error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty for query date with no yesterday data, got %d", len(result))
	}
}
