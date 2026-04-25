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

func TestCollectTagInputs_IncludesEntireAbstractTree(t *testing.T) {
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

	foundRoot := false
	foundChild := false
	foundGrandchild := false
	for _, inp := range inputs {
		if inp.ID == root.ID {
			foundRoot = true
		}
		if inp.ID == child.ID {
			foundChild = true
			if inp.ParentLabel != "Tech" {
				t.Errorf("expected child ParentLabel=Tech, got %q", inp.ParentLabel)
			}
		}
		if inp.ID == grandchild.ID {
			foundGrandchild = true
			if inp.ParentLabel != "Cloud" {
				t.Errorf("expected grandchild ParentLabel=Cloud, got %q", inp.ParentLabel)
			}
		}
	}
	if !foundRoot {
		t.Errorf("expected root abstract tag (Tech) in inputs")
	}
	if !foundChild {
		t.Errorf("expected child abstract tag (Cloud) in inputs")
	}
	if !foundGrandchild {
		t.Errorf("expected grandchild tag (AWS) in inputs")
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

func TestCollectTagInputs_UnclassifiedOnlyWatchedAndTop10(t *testing.T) {
	db := setupCollectorTestDB(t)

	watched := models.TopicTag{Label: "Watched", Slug: "watched", Category: "keyword", Status: "active", Source: "llm", QualityScore: 0.2, IsWatched: true}
	db.Create(&watched)

	feed := models.Feed{Title: "Feed", URL: "https://example.com/f4"}
	db.Create(&feed)

	pubDate := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	article := models.Article{FeedID: feed.ID, Title: "Article watched", Link: "https://example.com/watched", PubDate: &pubDate}
	db.Create(&article)
	db.Create(&models.ArticleTopicTag{ArticleID: article.ID, TopicTagID: watched.ID})

	for i := 0; i < 11; i++ {
		qs := 0.9 - float64(i)*0.05
		if qs < 0.1 {
			qs = 0.1
		}
		tag := models.TopicTag{
			Label:        fmt.Sprintf("Tag%d", i),
			Slug:         fmt.Sprintf("tag%d", i),
			Category:     "keyword",
			Status:       "active",
			Source:       "llm",
			QualityScore: qs,
		}
		db.Create(&tag)
		a := models.Article{FeedID: feed.ID, Title: "Article " + tag.Label, Link: "https://example.com/" + tag.Slug, PubDate: &pubDate}
		db.Create(&a)
		db.Create(&models.ArticleTopicTag{ArticleID: a.ID, TopicTagID: tag.ID})
	}

	lowQ := models.TopicTag{Label: "LowQ", Slug: "lowq", Category: "keyword", Status: "active", Source: "llm", QualityScore: 0.05}
	db.Create(&lowQ)
	a := models.Article{FeedID: feed.ID, Title: "Article lowq", Link: "https://example.com/lowq", PubDate: &pubDate}
	db.Create(&a)
	db.Create(&models.ArticleTopicTag{ArticleID: a.ID, TopicTagID: lowQ.ID})

	inputs, err := CollectTagInputs(pubDate)
	if err != nil {
		t.Fatalf("CollectTagInputs returned error: %v", err)
	}

	foundWatched := false
	foundLowQ := false
	nonWatchedCount := 0
	for _, inp := range inputs {
		if inp.ID == watched.ID {
			foundWatched = true
			if !inp.IsWatched {
				t.Errorf("expected IsWatched=true for watched tag")
			}
		}
		if inp.ID == lowQ.ID {
			foundLowQ = true
		}
		if inp.Source == "llm" && !inp.IsAbstract && !inp.IsWatched {
			nonWatchedCount++
		}
	}

	if !foundWatched {
		t.Errorf("expected watched tag in inputs even with low quality_score")
	}
	if nonWatchedCount > 10 {
		t.Errorf("expected at most 10 non-watched unclassified tags, got %d", nonWatchedCount)
	}
	if foundLowQ {
		t.Errorf("expected low quality non-watched tag to be excluded (ranked beyond top 10)")
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

func setupCollectorTestDBWithCategories(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupCollectorTestDB(t)

	if err := db.AutoMigrate(&models.Category{}); err != nil {
		t.Fatalf("migrate categories: %v", err)
	}
	return db
}

func TestCollectCategoryNarrativeSummaries_NoData(t *testing.T) {
	setupCollectorTestDBWithCategories(t)

	date := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	result, err := CollectCategoryNarrativeSummaries(date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil for empty DB, got %d items", len(result))
	}
}

func TestCollectCategoryNarrativeSummaries_BasicGrouping(t *testing.T) {
	db := setupCollectorTestDBWithCategories(t)

	cat1 := models.Category{Name: "Tech", Slug: "tech", Icon: "💻"}
	cat2 := models.Category{Name: "Science", Slug: "science", Icon: "🔬"}
	db.Create(&cat1)
	db.Create(&cat2)

	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	cat1ID := cat1.ID
	cat2ID := cat2.ID

	narratives := []models.NarrativeSummary{
		{Title: "T1", Summary: "S1", Status: "emerging", Period: "daily", PeriodDate: date, ScopeType: models.NarrativeScopeTypeFeedCategory, ScopeCategoryID: &cat1ID, Source: "ai", RelatedTagIDs: "[]", RelatedArticleIDs: "[]"},
		{Title: "T2", Summary: "S2", Status: "continuing", Period: "daily", PeriodDate: date, ScopeType: models.NarrativeScopeTypeFeedCategory, ScopeCategoryID: &cat1ID, Source: "ai", RelatedTagIDs: "[]", RelatedArticleIDs: "[]"},
		{Title: "T3", Summary: "S3", Status: "emerging", Period: "daily", PeriodDate: date, ScopeType: models.NarrativeScopeTypeFeedCategory, ScopeCategoryID: &cat2ID, Source: "ai", RelatedTagIDs: "[]", RelatedArticleIDs: "[]"},
		{Title: "T4", Summary: "S4", Status: "emerging", Period: "daily", PeriodDate: date, ScopeType: models.NarrativeScopeTypeFeedCategory, ScopeCategoryID: &cat2ID, Source: "ai", RelatedTagIDs: "[]", RelatedArticleIDs: "[]"},
	}
	for _, n := range narratives {
		if err := db.Create(&n).Error; err != nil {
			t.Fatalf("create narrative: %v", err)
		}
	}

	result, err := CollectCategoryNarrativeSummaries(date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 category groups, got %d", len(result))
	}

	totalNarratives := 0
	for _, ci := range result {
		totalNarratives += len(ci.Narratives)
	}
	if totalNarratives != 4 {
		t.Errorf("expected 4 total narratives, got %d", totalNarratives)
	}
}

func TestCollectCategoryNarrativeSummaries_ExcludesEnding(t *testing.T) {
	db := setupCollectorTestDBWithCategories(t)

	cat := models.Category{Name: "Tech", Slug: "tech", Icon: "💻"}
	db.Create(&cat)

	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	catID := cat.ID

	narratives := []models.NarrativeSummary{
		{Title: "Active", Summary: "S1", Status: "emerging", Period: "daily", PeriodDate: date, ScopeType: models.NarrativeScopeTypeFeedCategory, ScopeCategoryID: &catID, Source: "ai", RelatedTagIDs: "[]", RelatedArticleIDs: "[]"},
		{Title: "Ending", Summary: "S2", Status: models.NarrativeStatusEnding, Period: "daily", PeriodDate: date, ScopeType: models.NarrativeScopeTypeFeedCategory, ScopeCategoryID: &catID, Source: "ai", RelatedTagIDs: "[]", RelatedArticleIDs: "[]"},
	}
	for _, n := range narratives {
		if err := db.Create(&n).Error; err != nil {
			t.Fatalf("create narrative: %v", err)
		}
	}

	result, err := CollectCategoryNarrativeSummaries(date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 category group, got %d", len(result))
	}
	if len(result[0].Narratives) != 1 {
		t.Fatalf("expected 1 narrative (ending excluded), got %d", len(result[0].Narratives))
	}
	if result[0].Narratives[0].Title != "Active" {
		t.Errorf("expected 'Active', got %q", result[0].Narratives[0].Title)
	}
}

func TestCollectCategoryNarrativeSummaries_CapsPerCategory(t *testing.T) {
	db := setupCollectorTestDBWithCategories(t)

	cat := models.Category{Name: "Tech", Slug: "tech", Icon: "💻"}
	db.Create(&cat)

	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	catID := cat.ID

	for i := 0; i < 7; i++ {
		n := models.NarrativeSummary{
			Title:             fmt.Sprintf("Narrative %d", i),
			Summary:           fmt.Sprintf("Summary %d", i),
			Status:            "emerging",
			Period:            "daily",
			PeriodDate:        date,
			ScopeType:         models.NarrativeScopeTypeFeedCategory,
			ScopeCategoryID:   &catID,
			Source:            "ai",
			RelatedTagIDs:     "[]",
			RelatedArticleIDs: "[]",
		}
		if err := db.Create(&n).Error; err != nil {
			t.Fatalf("create narrative %d: %v", i, err)
		}
	}

	result, err := CollectCategoryNarrativeSummaries(date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 category group, got %d", len(result))
	}
	if len(result[0].Narratives) != 5 {
		t.Fatalf("expected 5 narratives (capped), got %d", len(result[0].Narratives))
	}
}

func TestCollectCategoryNarrativeSummaries_CapsTotal(t *testing.T) {
	db := setupCollectorTestDBWithCategories(t)

	var categories []models.Category
	for i := 0; i < 7; i++ {
		cat := models.Category{
			Name: fmt.Sprintf("Cat%d", i),
			Slug: fmt.Sprintf("cat%d", i),
			Icon: "📁",
		}
		db.Create(&cat)
		categories = append(categories, cat)
	}

	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)

	for i, cat := range categories {
		catID := cat.ID
		for j := 0; j < 5; j++ {
			articleIDs := "[1,2]"
			n := models.NarrativeSummary{
				Title:             fmt.Sprintf("N%d-%d", i, j),
				Summary:           fmt.Sprintf("S%d-%d", i, j),
				Status:            "emerging",
				Period:            "daily",
				PeriodDate:        date,
				ScopeType:         models.NarrativeScopeTypeFeedCategory,
				ScopeCategoryID:   &catID,
				Source:            "ai",
				RelatedTagIDs:     "[]",
				RelatedArticleIDs: articleIDs,
			}
			if err := db.Create(&n).Error; err != nil {
				t.Fatalf("create narrative: %v", err)
			}
		}
	}

	result, err := CollectCategoryNarrativeSummaries(date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	total := 0
	for _, ci := range result {
		total += len(ci.Narratives)
	}
	if total > 30 {
		t.Errorf("expected total narratives capped at 30, got %d", total)
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
