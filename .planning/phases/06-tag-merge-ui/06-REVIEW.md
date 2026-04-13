---
phase: 06-tag-merge-ui
reviewed: 2026-04-13T12:00:00Z
depth: standard
files_reviewed: 7
files_reviewed_list:
  - backend-go/internal/app/router.go
  - backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go
  - backend-go/internal/domain/topicanalysis/tag_merge_preview.go
  - front/app/api/tagMergePreview.ts
  - front/app/features/topic-graph/components/TagMergePreview.vue
  - front/app/features/topic-graph/components/TopicGraphPage.vue
  - front/app/types/tagMerge.ts
findings:
  critical: 1
  warning: 4
  info: 3
  total: 8
status: issues_found
---

# Phase 06: Code Review Report

**Reviewed:** 2026-04-13T12:00:00Z
**Depth:** standard
**Files Reviewed:** 7
**Status:** issues_found

## Summary

Reviewed the tag merge preview feature spanning backend (Go/Gin) and frontend (Vue 3/TypeScript). The feature adds two API endpoints for scanning similar tag pairs and merging with custom names, plus a modal UI component.

One critical issue: the merge handler lacks transaction safety, creating a race condition that can corrupt data under concurrent requests. Four warnings cover silent data loss from error swallowing, duplicate watchers causing double API calls, missing slug collision handling, and non-atomic rename-then-merge logic. Three info items note type mapping gaps, dead code paths, and `any` usage.

## Critical Issues

### CR-01: Merge handler has no transaction — race condition risk

**File:** `backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go:95-131`
**Issue:** `MergeTagsWithCustomNameHandler` reads both tags, checks their status, renames the target, and calls `MergeTags` — all as separate DB operations without a transaction. Two concurrent requests involving the same tag pair can both pass the `"already merged"` check (lines 105-112), then both proceed to rename and merge, causing double-merge or data corruption.
**Fix:**
```go
// Wrap the entire check-rename-merge sequence in a transaction
err := database.DB.Transaction(func(tx *gorm.DB) error {
    var source, target models.TopicTag
    if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&source, body.SourceTagID).Error; err != nil {
        return fmt.Errorf("source tag not found: %w", err)
    }
    if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&target, body.TargetTagID).Error; err != nil {
        return fmt.Errorf("target tag not found: %w", err)
    }
    if source.Status == "merged" || target.Status == "merged" {
        return fmt.Errorf("tag already merged")
    }
    // rename + merge using tx instead of database.DB
    // ...
    return nil
})
```

## Warnings

### WR-01: Silent data loss — ScanSimilarTagPairs skips pairs on DB error

**File:** `backend-go/internal/domain/topicanalysis/tag_merge_preview.go:78-83`
**Issue:** When `DB.First(&tag1, ...)` or `DB.First(&tag2, ...)` fails, the pair is silently skipped with `continue`. The caller has no way to know that results are incomplete. A transient DB error during one lookup would return a partial candidate list with no indication of data loss.
**Fix:** Accumulate errors and return them, or at least log them:
```go
var skipped int
for _, pair := range pairs {
    var tag1, tag2 models.TopicTag
    if err := database.DB.First(&tag1, pair.SourceID).Error; err != nil {
        skipped++
        continue
    }
    // ...
}
if skipped > 0 {
    log.Printf("ScanSimilarTagPairs: skipped %d pairs due to DB lookup errors", skipped)
}
```

### WR-02: Duplicate watcher fires loadGraph twice per filter change

**File:** `front/app/features/topic-graph/components/TopicGraphPage.vue:802-808`
**Issue:** Two identical watchers are registered on `[selectedFilterCategoryId, selectedFilterFeedId]` (lines 802-804 and 806-808). Every time either filter changes, `loadGraph()` runs twice, sending duplicate API requests.
**Fix:** Remove the duplicate watcher at lines 806-808:
```typescript
// Keep only one:
watch([selectedFilterCategoryId, selectedFilterFeedId], () => {
  void loadGraph()
})
// Remove the second identical watcher (lines 806-808)
```

### WR-03: No slug collision check when renaming target tag

**File:** `backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go:115-126`
**Issue:** When `newName` differs from `target.Label`, the handler generates a new slug via `Slugify(newName)` and updates the target. If another active tag already has that slug, this creates a unique constraint violation (if one exists) or silent slug collision. Either way, downstream lookups by slug become ambiguous.
**Fix:** Check for slug uniqueness before renaming:
```go
newSlug := topictypes.Slugify(newName)
var conflictCount int64
database.DB.Model(&models.TopicTag{}).
    Where("slug = ? AND id != ? AND (status = 'active' OR status = '' OR status IS NULL)", newSlug, target.ID).
    Count(&conflictCount)
if conflictCount > 0 {
    c.JSON(http.StatusConflict, gin.H{"success": false, "error": "a tag with this name already exists"})
    return
}
```

### WR-04: Rename and merge are not atomic — partial state on failure

**File:** `backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go:115-131`
**Issue:** The handler first renames the target tag (lines 117-126), then calls `MergeTags` (line 128). If `MergeTags` fails after the rename succeeds, the target tag is left with a new name/label but the merge didn't happen. The user sees "failed" but the tag was already renamed — inconsistent state.
**Fix:** This is naturally solved by wrapping both operations in a transaction (as recommended in CR-01). The transaction rollback would undo the rename if the merge fails.

## Info

### IN-01: Backend JSON field names don't match frontend type property names

**File:** `backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go:18-19` and `front/app/types/tagMerge.ts:18-19`
**Issue:** The backend `mergePreviewCandidate` struct uses `json:"source_article_list"` and `json:"target_article_list"`, but the frontend `TagMergeCandidate` type defines `sourceArticleTitles` and `targetArticleTitles`. The API client in `tagMergePreview.ts` does not perform snake_case → camelCase mapping for these fields. Unless `apiClient.get` auto-transforms keys, these optional arrays will always be `undefined`.
**Fix:** Either align the backend JSON tags to `source_article_titles`/`target_article_titles` and add API-layer mapping, or ensure the API client handles the key transformation. Verify the actual runtime behavior — if articles appear in the UI, the mapping works elsewhere and this is just a naming inconsistency.

### IN-02: `any` type in TopicGraphPage filterTopics function

**File:** `front/app/features/topic-graph/components/TopicGraphPage.vue:194`
**Issue:** `filterTopics(topics: any[], query: string)` uses `any` for the topics parameter, bypassing TypeScript type checking. The project convention is to avoid `any`.
**Fix:**
```typescript
function filterTopics(topics: { label: string; slug: string }[], query: string) {
  if (!query.trim()) return topics
  const lowerQuery = query.toLowerCase()
  return topics.filter(topic =>
    topic.label.toLowerCase().includes(lowerQuery) ||
    topic.slug.toLowerCase().includes(lowerQuery)
  )
}
```

### IN-03: `includeArticles` always true — false path never exercised

**File:** `front/app/features/topic-graph/components/TagMergePreview.vue:46`
**Issue:** The scan call always passes `{ limit: 50, includeArticles: true }`, making the `include_articles` parameter's `false` default path unreachable from this UI. The API supports it, but it's dead code in the current frontend flow.
**Fix:** No action required. This is informational — the parameter exists for potential future use (e.g., faster scan without article lists). Just noting the unused path.

---

_Reviewed: 2026-04-13T12:00:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
