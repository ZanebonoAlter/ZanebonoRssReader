# Phase 08: 标签树增强与图谱交互优化 — Research

**Status:** RESEARCH COMPLETE
**Date:** 2026-04-14

## Architecture Summary

Phase 8 extends the tag hierarchy and topic graph with description extraction, time filtering, abstract tag visualization, merge preview migration, and node reassignment. All changes are additive — no core tag convergence/merge logic is modified.

## Key Findings

### Backend — TopicTag Model
- `topic_graph.go`: TopicTag struct has no `Description` field yet. Adding it requires a migration to `topic_tags` table.
- Current fields: ID, Slug, Label, Category, Icon, Aliases, IsCanonical, Source, FeedCount, Status, MergedIntoID, CreatedAt, UpdatedAt, Kind.
- GORM model is straightforward — adding `Description string` field with `gorm:"type:text"` is sufficient.

### Backend — Description Generation (D-01)
- `tagger.go:findOrCreateTag()` is the creation entry point. It calls `es.TagMatch()` for three-level matching, then falls through to creation (line 242-262).
- **Integration point:** After `database.DB.Create(&newTag)` succeeds (line 253), add async description generation via LLM.
- The LLM call pattern is already established in `abstract_tag_service.go:callLLMForAbstractName()`:
  - Uses `airouter.NewRouter()` with `CapabilityTopicTagging`
  - `JSONMode: true`, `Temperature: 0.3`
  - Response is parsed via `json.Unmarshal` into a struct.
- Description prompt should include: tag label, slug, category, and associated article titles/summaries.
- **Challenge:** At creation time, the article isn't linked yet. The description should be generated AFTER the `AISummaryTopic` link is created (line 92-100). This means the description generation should happen after the tag is linked to the summary, with access to the summary's title/summary text.
- **Alternative:** Pass the article context to `findOrCreateTag()` and generate description after creation. Or use a deferred queue approach (similar to embedding generation).

### Backend — Abstract Tag Description (D-02)
- `abstract_tag_service.go:callLLMForAbstractName()` currently returns only `abstract_name`.
- Modify `buildAbstractTagPrompt()` to request both `abstract_name` AND `description` in the JSON response.
- Modify `parseAbstractNameFromJSON()` to return a struct with both fields (or create a new parser).
- When creating the abstract tag (line 74-81), set the `Description` field.
- If reusing an existing abstract tag (line 69-71), skip description (already set).

### Backend — Time Filtering (D-03, D-04)
- `abstract_tag_service.go:GetTagHierarchy()` already accepts `scopeFeedID` and `scopeCategoryID` parameters.
- Adding time range requires a JOIN with `article_topic_tags` → `articles` to filter by `published_at`.
- The query at line 160-164 loads `TopicTagRelation` records. A time filter would add a subquery: "only include tags that have articles within the time range."
- Inactive tags (no articles in range) should still appear in the tree but with `is_active: false` flag.
- The `TagHierarchyNode` struct needs an `IsActive` bool field.

### Frontend — TagHierarchy Time Filter (D-03, D-04)
- `TagHierarchy.vue` already has `selectedCategory`, `showUnclassified`, `searchQuery` filters.
- Adding time filter: new ref `timeRange` (e.g., '7d', '30d', 'custom') with date inputs for custom range.
- Pass `timeRange` to API via query params.
- CSS: `opacity: 0.4` for inactive tags (D-04), applied via `:class` binding on `isActive`.

### Frontend — Graph Abstract Tag Visualization (D-05, D-06)
- `buildTopicGraphViewModel.ts`: `resolveNodeAccent()` maps category to color. Abstract tags need a glow/halo effect.
- Current category colors: event=#f59e0b, person=#10b981, keyword=#6366f1 (matches D-05).
- For glow: Add an `isAbstract` flag to the node data, then in `TopicGraphCanvas.client.vue`, apply emissive material to abstract nodes.
- `buildNodeObject()` in the canvas file creates Three.js mesh objects. For abstract nodes, add an emissive color (same as accent but brighter) or an outline pass.
- Detail panel on click (D-06): Reuse existing sidebar pattern. Add a panel that shows child tags + article timeline with sub-tag filtering.

### Frontend — TagMergePreview Migration (D-07, D-08)
- `TagMergePreview.vue` is currently mounted in `TopicGraphPage.vue`.
- `GlobalSettingsDialog.vue` has a tab system (line 37): 'feeds', 'categories', 'general', 'backend-queues', 'preferences', 'firecrawl', 'schedulers'.
- Add a new tab 'tag-merge' or embed it in the existing 'backend-queues' tab (next to `MergeReembeddingQueuePanel`).
- D-08: After merge completes (the `merged` emit from TagMergePreview), show a toast prompting user to trigger abstract layer rebuild.
- Rebuild is a manual trigger — implement as a button in the toast or in the merge panel.

### Frontend — Node Manual Reassignment
- Not explicitly in success criteria, but implied: "标签树节点支持手动调整到其他节点"
- Requires: backend endpoint for reassigning a tag's parent in the hierarchy.
- Frontend: drag action or context menu on TagHierarchyRow → opens a modal with embedding-similar abstract tags for selection.

## Patterns to Reuse

| Pattern | Location | Usage |
|---------|----------|-------|
| JSON mode LLM call | `abstract_tag_service.go:callLLMForAbstractName()` | Extend for description |
| Async embedding generation | `tagger.go:generateAndSaveEmbedding()` | Mirror for async description |
| TagHierarchy filter pattern | `TagHierarchy.vue:selectedCategory` | Add time filter |
| Category color mapping | `buildTopicGraphViewModel.ts:TOPIC_CATEGORY_ACCENTS` | Add abstract glow |
| Tab pattern in settings | `GlobalSettingsDialog.vue:activeTab` | Add tag-merge tab |
| Toast/prompt pattern | Existing toast composables | Merge rebuild prompt |

## Risks

1. **LLM cost**: Description generation adds an LLM call per new tag. Mitigation: async + only for truly new tags (not reused).
2. **Graph performance**: Emissive materials add GPU overhead. Mitigation: only apply to abstract nodes (typically fewer).
3. **Time filter query performance**: JOIN with articles may be slow for large tag sets. Mitigation: indexed `published_at` column, scope to active tags only.

## Validation Architecture

Phase 8 modifies both backend (Go) and frontend (Vue). Verification should include:
- Backend: `go test ./internal/domain/topicanalysis/... -v`
- Frontend: `pnpm exec nuxi typecheck && pnpm test:unit && pnpm build`
- Manual: Visual check of tag hierarchy with time filter, graph glow effects, settings page merge panel.
