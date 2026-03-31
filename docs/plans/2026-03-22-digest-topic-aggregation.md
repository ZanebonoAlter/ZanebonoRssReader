# Digest Topic Aggregation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make digest tags derive from aggregated article tags instead of standalone digest tags, and expose reusable article tag visualization across article, digest, and topic graph UIs.

**Architecture:** Backend keeps `article_topic_tags` as the only tag source of truth, computes digest-level `aggregated_tags` at runtime, and exposes article tags on the standard article detail API. Frontend adds a reusable article tag component, switches digest/topic graph tag displays to `aggregated_tags`, and highlights the currently selected topic in the topic graph UI.

**Tech Stack:** Go + Gin + GORM + SQLite, Nuxt 4 + Vue 3 + TypeScript

---

### Task 1: Inspect current response shapes and add backend tests first

**Files:**
- Modify: `backend-go/internal/domain/articles/handler_test.go`
- Modify: `backend-go/internal/domain/topicgraph/handler_test.go`
- Modify: `backend-go/internal/domain/digest/` test files that cover preview/detail payloads

**Step 1: Add a failing article detail test**

Assert that `GET /api/articles/:id` returns `tags` populated from `article_topic_tags`.

**Step 2: Add failing topic graph digest tests**

Assert that topic graph digest payloads include `aggregated_tags` and that hotspot digests are no longer empty-tag responses.

**Step 3: Add failing digest response tests**

Assert that digest preview/detail payloads expose digest-level aggregated tags with de-duped entries and `article_count`.

**Step 4: Run targeted backend tests**

Run: `go test ./internal/domain/articles ./internal/domain/topicgraph ./internal/domain/digest -v`

Expected: FAIL on missing `tags` / `aggregated_tags` fields.

### Task 2: Add shared backend digest/article tag aggregation types

**Files:**
- Create: `backend-go/internal/domain/topictypes/digest_tags.go` or equivalent shared type file
- Modify: existing response type files under `backend-go/internal/domain/topicgraph/` and `backend-go/internal/domain/digest/`

**Step 1: Define a shared aggregated tag shape**

Include:

- `label`
- `slug`
- `category`
- `kind` if needed for normalization
- `score` if already available
- `article_count`

**Step 2: Thread the new type through digest and topic graph payload structs**

Add `aggregated_tags` to digest card / summary response types while keeping old `topics` fields only for compatibility where necessary.

**Step 3: Run Go formatting if needed**

Run: `gofmt -w <edited-go-files>`

### Task 3: Implement shared backend aggregation query

**Files:**
- Modify: `backend-go/internal/domain/topicextraction/article_tagger.go`
- Create or modify: helper file under `backend-go/internal/domain/digest/` or shared topic helper package

**Step 1: Reuse article tag lookup for single article**

Expose or reuse a function that returns normalized tags for one article.

**Step 2: Add digest-level aggregation helper**

Input: article IDs

Output: de-duped tags with `article_count`, sorted by count then label.

**Step 3: Make aggregation ignore empty / broken associations safely**

Return empty array instead of failing the whole digest response when a tag row is missing.

**Step 4: Re-run targeted backend tests**

Run: `go test ./internal/domain/topicextraction ./internal/domain/digest -v`

### Task 4: Expose tags on the standard article detail API

**Files:**
- Modify: `backend-go/internal/domain/articles/handler.go`
- Modify: `backend-go/internal/domain/models/article.go` only if response serialization must change there

**Step 1: Extend article detail response**

Fetch article tags during `GetArticle` and include them in the returned payload.

**Step 2: Decide serialization boundary**

Prefer handler-level response extension if `ToDict()` is used by list endpoints and should stay light.

**Step 3: Re-run article tests**

Run: `go test ./internal/domain/articles -v`

### Task 5: Switch digest responses to aggregated tags

**Files:**
- Modify: `backend-go/internal/domain/digest/handler.go`
- Modify: `backend-go/internal/domain/digest/generator.go`
- Modify: any digest presenter/mapper files involved in preview/detail payloads

**Step 1: Attach aggregated tags to each digest summary card**

Use the article IDs already associated with each summary/digest item.

**Step 2: Preserve old fields only if callers still need them**

Do not use summary-owned topics as the displayed digest tag source anymore.

**Step 3: Re-run digest tests**

Run: `go test ./internal/domain/digest -v`

### Task 6: Switch topic graph digest responses to aggregated tags

**Files:**
- Modify: `backend-go/internal/domain/topicgraph/service.go`
- Modify: `backend-go/internal/domain/topicgraph/hotspot_digests.go`
- Modify: `backend-go/internal/domain/topicgraph/handler.go` if response mapping lives there

**Step 1: Add aggregated tags to topic detail digest summaries**

Ensure `BuildTopicDetail` returns digest cards with `aggregated_tags`.

**Step 2: Add aggregated tags to hotspot digest responses**

Ensure `/topic-graph/tag/:slug/digests` returns actual digest tags instead of empty placeholders on the frontend.

**Step 3: Re-run topic graph tests**

Run: `go test ./internal/domain/topicgraph -v`

### Task 7: Add frontend types for article tags and aggregated digest tags

**Files:**
- Modify: `front/app/types/article.ts`
- Modify: `front/app/api/topicGraph.ts`
- Modify: any digest API type file under `front/app/api/digest.ts`

**Step 1: Extend `Article` with `tags`**

Add a typed tag array compatible with backend article detail payload.

**Step 2: Extend digest/topic graph card types**

Add `aggregated_tags` to digest-related response types.

**Step 3: Update any local mapper helpers**

Make sure old `topics`-based mapping is no longer the default display path.

**Step 4: Run typecheck**

Run: `pnpm exec nuxi typecheck`

Expected: FAIL until UI updates are complete.

### Task 8: Build reusable article tag display component

**Files:**
- Create: `front/app/features/articles/components/ArticleTagList.vue`
- Optionally create: tag helper file under `front/app/features/articles/utils/`

**Step 1: Implement reusable props**

Support:

- `tags`
- `highlightedSlugs?`
- `compact?`
- `grouped?`
- `maxVisible?`

**Step 2: Keep styling lightweight and reusable**

Use existing editorial palette, avoid introducing a new design language.

**Step 3: Add a focused component test if test setup exists nearby**

Verify rendering, truncation, and highlighted-tag state.

### Task 9: Add tags to `ArticleContentView.vue`

**Files:**
- Modify: `front/app/features/articles/components/ArticleContentView.vue`

**Step 1: Render article tags in the common article detail UI**

Place them near the article title/meta area so all consumers inherit the same display.

**Step 2: Handle empty tags cleanly**

Do not render an empty shell when an article has no tags.

**Step 3: Run targeted frontend tests or typecheck**

Run: `pnpm exec nuxi typecheck`

### Task 10: Switch digest UI to aggregated tags

**Files:**
- Modify: `front/app/features/digest/components/DigestDetail.vue`
- Modify: any digest card/list components that currently display summary topics as digest tags

**Step 1: Replace digest tag source**

Render `aggregated_tags` as the digest-level index display.

**Step 2: Add lightweight index context copy if needed**

Example: `索引标签` / `来自 N 篇文章`.

**Step 3: Verify digest article preview still reuses `ArticleContentView`**

No duplicate per-page tag implementation.

### Task 11: Adjust topic graph UI to show both digest-level and article-level tags

**Files:**
- Modify: `front/app/features/topic-graph/components/TopicGraphPage.vue`
- Modify: `front/app/features/topic-graph/components/TopicTimeline.vue`
- Modify: `front/app/features/topic-graph/components/TimelineItem.vue`
- Modify: `front/app/features/topic-graph/components/TopicGraphSidebar.vue` if digest tag display lives there

**Step 1: Map digest cards from `aggregated_tags`**

Remove placeholder empty-tag behavior for hotspot digests.

**Step 2: Highlight currently selected topic slug**

When the current topic/tag appears in digest aggregated tags or article tags, render it in a highlighted state.

**Step 3: Keep the UI readable**

Default to compact tag rows with truncation/expansion rather than full tag walls.

**Step 4: Re-run frontend checks**

Run: `pnpm exec nuxi typecheck && pnpm test:unit -- app/features/topic-graph/components/TopicTimeline.test.ts`

### Task 12: Update docs and verify end-to-end behavior

**Files:**
- Modify: `docs/guides/topic-graph.md`
- Modify: `docs/architecture/backend-go.md`
- Modify: `docs/architecture/frontend.md`

**Step 1: Document the new digest tag semantics**

Explain that digest tags are article-derived aggregate indexes, not standalone digest tags.

**Step 2: Run final targeted verification**

Backend:

`go test ./internal/domain/articles ./internal/domain/digest ./internal/domain/topicgraph -v`

Frontend:

`pnpm exec nuxi typecheck`

**Step 3: Manual verification checklist**

- Open a tagged article from the main reader and confirm tags display
- Open a digest article modal and confirm the same tag UI appears
- Open `/topics`, inspect hotspot digests and topic-detail digests, and confirm aggregated tags render and highlight correctly
