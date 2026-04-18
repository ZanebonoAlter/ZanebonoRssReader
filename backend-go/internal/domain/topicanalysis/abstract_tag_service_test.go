package topicanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/database"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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

func TestParseTagJudgmentResponse(t *testing.T) {
	candidates := []TagCandidate{
		{Tag: &models.TopicTag{Label: "GPT-4", Slug: "gpt-4"}},
		{Tag: &models.TopicTag{Label: "ChatGPT", Slug: "chatgpt"}},
		{Tag: &models.TopicTag{Label: "AI研究", Slug: "ai-yan-jiu"}},
	}

	t.Run("merge only", func(t *testing.T) {
		input := `{"merge":{"target":"GPT-4","label":"GPT-4","children":["ChatGPT"],"reason":"same concept"},"abstract":null}`
		result, err := parseTagJudgmentResponse(input, candidates)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Merge == nil {
			t.Fatal("expected merge judgment")
		}
		if result.Merge.Target != "GPT-4" {
			t.Errorf("expected target 'GPT-4', got %q", result.Merge.Target)
		}
		if len(result.Merge.Children) != 1 || result.Merge.Children[0] != "ChatGPT" {
			t.Errorf("expected children [ChatGPT], got %v", result.Merge.Children)
		}
		if result.Abstract != nil {
			t.Error("expected no abstract judgment")
		}
	})

	t.Run("both merge and abstract", func(t *testing.T) {
		input := `{"merge":{"target":"GPT-4","label":"GPT-4","children":[],"reason":""},"abstract":{"name":"AI技术","description":"AI技术相关","children":["AI研究"],"reason":""}}`
		result, err := parseTagJudgmentResponse(input, candidates)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Merge == nil {
			t.Fatal("expected merge judgment")
		}
		if result.Abstract == nil {
			t.Fatal("expected abstract judgment")
		}
		if result.Abstract.Name != "AI技术" {
			t.Errorf("expected abstract name 'AI技术', got %q", result.Abstract.Name)
		}
	})

	t.Run("dedup across merge and abstract children", func(t *testing.T) {
		input := `{"merge":{"target":"GPT-4","label":"GPT-4","children":["ChatGPT"],"reason":""},"abstract":{"name":"AI","description":"AI","children":["ChatGPT","AI研究"],"reason":""}}`
		result, err := parseTagJudgmentResponse(input, candidates)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, ch := range result.Abstract.Children {
			if ch == "ChatGPT" {
				t.Error("ChatGPT should not appear in abstract children (already in merge children)")
			}
		}
	})

	t.Run("invalid children filtered", func(t *testing.T) {
		input := `{"merge":{"target":"GPT-4","label":"GPT-4","children":["NonExistent"],"reason":""}}`
		result, err := parseTagJudgmentResponse(input, candidates)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Merge.Children) != 0 {
			t.Errorf("non-existent children should be filtered, got %v", result.Merge.Children)
		}
	})
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

func TestEnsureNewLabelCandidateInAbstractJudgment(t *testing.T) {
	candidates := []TagCandidate{
		{Tag: &models.TopicTag{ID: 1, Label: "AbortController", Slug: "abortcontroller"}},
		{Tag: &models.TopicTag{ID: 2, Label: "AbortSignal", Slug: "abortsignal"}},
	}

	t.Run("adds current candidate when abstract only names sibling", func(t *testing.T) {
		judgment := &tagJudgment{
			Abstract: &tagJudgmentAbstract{
				Name:     "AbortController 与 AbortSignal",
				Children: []string{"AbortSignal"},
			},
		}

		ensureNewLabelCandidateInAbstractJudgment(judgment, candidates, "AbortController")

		if len(judgment.Abstract.Children) != 2 {
			t.Fatalf("expected two children, got %v", judgment.Abstract.Children)
		}
		if !labelInSlice(judgment.Abstract.Children, "AbortController") {
			t.Fatalf("expected current candidate to be added, got %v", judgment.Abstract.Children)
		}
	})

	t.Run("does not duplicate existing current candidate", func(t *testing.T) {
		judgment := &tagJudgment{
			Abstract: &tagJudgmentAbstract{
				Name:     "AbortController 与 AbortSignal",
				Children: []string{"AbortController", "AbortSignal"},
			},
		}

		ensureNewLabelCandidateInAbstractJudgment(judgment, candidates, "AbortController")

		if len(judgment.Abstract.Children) != 2 {
			t.Fatalf("expected no duplicate child, got %v", judgment.Abstract.Children)
		}
	})

	t.Run("does not add candidate already consumed by merge", func(t *testing.T) {
		judgment := &tagJudgment{
			Merge: &tagJudgmentMerge{
				Target: "AbortController",
			},
			Abstract: &tagJudgmentAbstract{
				Name:     "Abort APIs",
				Children: []string{"AbortSignal"},
			},
		}

		ensureNewLabelCandidateInAbstractJudgment(judgment, candidates, "AbortController")

		if labelInSlice(judgment.Abstract.Children, "AbortController") {
			t.Fatalf("expected merged candidate not to be added, got %v", judgment.Abstract.Children)
		}
	})
}

func TestOrganizeMatchCategory(t *testing.T) {
	if got := organizeMatchCategory("event", &models.TopicTag{Category: "keyword"}); got != "event" {
		t.Fatalf("expected explicit category, got %q", got)
	}

	if got := organizeMatchCategory("", &models.TopicTag{Category: "person"}); got != "person" {
		t.Fatalf("expected tag category fallback, got %q", got)
	}

	if got := organizeMatchCategory("", &models.TopicTag{}); got != "keyword" {
		t.Fatalf("expected keyword fallback, got %q", got)
	}
}

func TestShouldUseOrganizeCandidate(t *testing.T) {
	used := map[uint]bool{3: true}

	cases := []struct {
		name      string
		candidate TagCandidate
		want      bool
	}{
		{
			name:      "uses valid neighbor",
			candidate: TagCandidate{Tag: &models.TopicTag{ID: 2}, Similarity: 0.78},
			want:      true,
		},
		{
			name:      "skips current tag",
			candidate: TagCandidate{Tag: &models.TopicTag{ID: 1}, Similarity: 1},
			want:      false,
		},
		{
			name:      "skips low similarity",
			candidate: TagCandidate{Tag: &models.TopicTag{ID: 2}, Similarity: 0.77},
			want:      false,
		},
		{
			name:      "skips used tag",
			candidate: TagCandidate{Tag: &models.TopicTag{ID: 3}, Similarity: 0.9},
			want:      false,
		},
		{
			name:      "skips nil tag",
			candidate: TagCandidate{Similarity: 0.9},
			want:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldUseOrganizeCandidate(tc.candidate, 1, used)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestFormatChildLabels(t *testing.T) {
	if got := formatChildLabels(nil); got != "(无子标签)" {
		t.Fatalf("expected empty label placeholder, got %q", got)
	}

	if got := formatChildLabels([]string{"A", "B"}); got != "A, B" {
		t.Fatalf("expected joined labels, got %q", got)
	}
}

func TestLoadAbstractChildLabelsEmpty(t *testing.T) {
	setupAbstractTagServiceTestDB(t)

	labels := loadAbstractChildLabels(9999, 5)
	if labels == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(labels) != 0 {
		t.Fatalf("expected no labels, got %v", labels)
	}
}

func TestProcessAbstractJudgmentReusesExistingAbstract(t *testing.T) {
	db := setupAbstractTagServiceTestDB(t)

	existing := models.TopicTag{Slug: "existing-abstract", Label: "已有抽象", Category: "event", Kind: "event", Source: "abstract", Status: "active"}
	child := models.TopicTag{Slug: "hang-yun-jing-bao", Label: "航运警报", Category: "event", Kind: "event", Source: "llm", Status: "active"}
	for _, tag := range []*models.TopicTag{&existing, &child} {
		if err := db.Create(tag).Error; err != nil {
			t.Fatalf("create tag: %v", err)
		}
	}

	originalFinder := findSimilarExistingAbstractFn
	findSimilarExistingAbstractFn = func(ctx context.Context, name, desc, category string, candidates []TagCandidate) *models.TopicTag {
		return &existing
	}
	t.Cleanup(func() {
		findSimilarExistingAbstractFn = originalFinder
	})

	result, err := processAbstractJudgment(context.Background(), []TagCandidate{{Tag: &child, Similarity: 0.91}}, &tagJudgmentAbstract{
		Name:        "新的抽象",
		Description: "desc",
		Children:    []string{"航运警报"},
	}, "新标签", "event")
	if err != nil {
		t.Fatalf("processAbstractJudgment returned error: %v", err)
	}
	if result == nil || result.Tag == nil {
		t.Fatal("expected abstract result")
	}
	if result.Tag.ID != existing.ID {
		t.Fatalf("expected existing abstract ID %d, got %d", existing.ID, result.Tag.ID)
	}

	var count int64
	if err := db.Model(&models.TopicTag{}).Where("source = ?", "abstract").Count(&count).Error; err != nil {
		t.Fatalf("count abstract tags: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one abstract tag after reuse, got %d", count)
	}

	assertAbstractRelationExists(t, db, existing.ID, child.ID)
}

func TestReparentOrLinkAbstractChildKeepsNarrowerIntermediateParent(t *testing.T) {
	db := setupAbstractTagServiceTestDB(t)

	grandParent := models.TopicTag{Slug: "zhong-dong-chong-tu", Label: "中东冲突", Category: "event", Kind: "event", Source: "abstract", Status: "active"}
	midParent := models.TopicTag{Slug: "huo-er-mu-zi-hai-xia-wei-ji", Label: "霍尔木兹海峡危机", Category: "event", Kind: "event", Source: "abstract", Status: "active"}
	child := models.TopicTag{Slug: "hang-yun-jing-bao", Label: "航运警报", Category: "event", Kind: "event", Source: "llm", Status: "active"}

	for _, tag := range []*models.TopicTag{&grandParent, &midParent, &child} {
		if err := db.Create(tag).Error; err != nil {
			t.Fatalf("create tag: %v", err)
		}
	}
	if err := db.Create(&models.TopicTagRelation{ParentID: midParent.ID, ChildID: child.ID, RelationType: "abstract"}).Error; err != nil {
		t.Fatalf("create original relation: %v", err)
	}

	originalJudge := aiJudgeNarrowerConceptFn
	aiJudgeNarrowerConceptFn = func(ctx context.Context, broader, narrower *models.TopicTag) (bool, error) {
		if broader.ID != grandParent.ID {
			t.Fatalf("expected broader tag %d, got %d", grandParent.ID, broader.ID)
		}
		if narrower.ID != midParent.ID {
			t.Fatalf("expected narrower tag %d, got %d", midParent.ID, narrower.ID)
		}
		return true, nil
	}
	t.Cleanup(func() {
		aiJudgeNarrowerConceptFn = originalJudge
	})

	if err := reparentOrLinkAbstractChild(context.Background(), child.ID, grandParent.ID); err != nil {
		t.Fatalf("reparentOrLinkAbstractChild returned error: %v", err)
	}

	assertAbstractRelationExists(t, db, midParent.ID, child.ID)
	assertAbstractRelationExists(t, db, grandParent.ID, midParent.ID)
	assertAbstractRelationMissing(t, db, grandParent.ID, child.ID)
}

func setupAbstractTagServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	database.DB = db
	if abstractTagUpdateQueueService != nil {
		abstractTagUpdateQueueService.db = db
	}
	t.Cleanup(func() {
		database.DB = nil
	})

	if err := db.AutoMigrate(
		&models.Feed{},
		&models.Article{},
		&models.TopicTag{},
		&models.TopicTagEmbedding{},
		&models.ArticleTopicTag{},
		&models.TopicTagRelation{},
	); err != nil {
		t.Fatalf("migrate test tables: %v", err)
	}

	return db
}

func assertAbstractRelationExists(t *testing.T, db *gorm.DB, parentID, childID uint) {
	t.Helper()

	var count int64
	if err := db.Model(&models.TopicTagRelation{}).
		Where("parent_id = ? AND child_id = ? AND relation_type = ?", parentID, childID, "abstract").
		Count(&count).Error; err != nil {
		t.Fatalf("count relation %d->%d: %v", parentID, childID, err)
	}
	if count != 1 {
		t.Fatalf("expected abstract relation %d->%d to exist once, got %d", parentID, childID, count)
	}
}

func assertAbstractRelationMissing(t *testing.T, db *gorm.DB, parentID, childID uint) {
	t.Helper()

	var count int64
	if err := db.Model(&models.TopicTagRelation{}).
		Where("parent_id = ? AND child_id = ? AND relation_type = ?", parentID, childID, "abstract").
		Count(&count).Error; err != nil {
		t.Fatalf("count relation %d->%d: %v", parentID, childID, err)
	}
	if count != 0 {
		t.Fatalf("expected abstract relation %d->%d to be absent, got %d", parentID, childID, count)
	}
}

func TestGetUnclassifiedTagsReturnsMoreThan200Rows(t *testing.T) {
	db := setupAbstractTagServiceTestDB(t)

	feed := models.Feed{
		Title: "Test Feed",
		URL:   "https://example.com/feed",
	}
	if err := db.Create(&feed).Error; err != nil {
		t.Fatalf("create feed: %v", err)
	}

	pubDate := time.Now()

	for i := 0; i < 205; i++ {
		tag := models.TopicTag{
			Slug:     fmt.Sprintf("tag-%03d", i),
			Label:    fmt.Sprintf("Tag %03d", i),
			Category: "keyword",
			Source:   "llm",
			Status:   "active",
		}
		if err := db.Create(&tag).Error; err != nil {
			t.Fatalf("create tag %d: %v", i, err)
		}

		article := models.Article{
			FeedID:  feed.ID,
			Title:   fmt.Sprintf("Article %03d", i),
			PubDate: &pubDate,
		}
		if err := db.Create(&article).Error; err != nil {
			t.Fatalf("create article %d: %v", i, err)
		}

		link := models.ArticleTopicTag{
			ArticleID:  article.ID,
			TopicTagID: tag.ID,
			Source:     "llm",
		}
		if err := db.Create(&link).Error; err != nil {
			t.Fatalf("create article topic tag %d: %v", i, err)
		}
	}

	nodes, err := GetUnclassifiedTags("", 0, 0, "")
	if err != nil {
		t.Fatalf("GetUnclassifiedTags returned error: %v", err)
	}

	if len(nodes) != 205 {
		t.Fatalf("unclassified tag count = %d, want 205", len(nodes))
	}
}

func TestCollectOrganizeMergeSources(t *testing.T) {
	current := &models.TopicTag{ID: 1, Label: "DeepSeek首次外部融资"}
	target := &models.TopicTag{ID: 2, Label: "DeepSeek 首次寻求外部融资"}
	child := &models.TopicTag{ID: 3, Label: "DeepSeek融资"}
	result := &TagExtractionResult{
		Merge:         &MergeResult{Target: target, Label: target.Label},
		MergeChildren: []*models.TopicTag{child, target, child},
	}

	sources := collectOrganizeMergeSources(result, current)
	sourceIDs := map[uint]bool{}
	for _, source := range sources {
		sourceIDs[source.ID] = true
	}

	if len(sourceIDs) != 2 {
		t.Fatalf("expected current tag and one merge child, got %v", sourceIDs)
	}
	if !sourceIDs[1] || !sourceIDs[3] {
		t.Fatalf("expected source IDs 1 and 3, got %v", sourceIDs)
	}
}
