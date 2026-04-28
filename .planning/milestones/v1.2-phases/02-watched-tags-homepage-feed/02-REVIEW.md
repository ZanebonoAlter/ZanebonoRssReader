---
phase: 02-watched-tags-homepage-feed
reviewed: 2026-04-15T20:15:00Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - backend-go/internal/domain/topicanalysis/watched_tags_service.go
  - backend-go/internal/domain/topicanalysis/watched_tags_handler.go
  - backend-go/internal/domain/models/topic_graph.go
  - backend-go/internal/domain/models/article.go
  - backend-go/internal/platform/database/postgres_migrations.go
  - backend-go/internal/domain/articles/handler.go
  - backend-go/internal/app/router.go
  - front/app/api/watchedTags.ts
  - front/app/types/article.ts
  - front/app/features/topic-graph/components/TagHierarchy.vue
  - front/app/features/topic-graph/components/TagHierarchyRow.vue
  - front/app/features/shell/components/AppSidebarView.vue
  - front/app/features/shell/components/FeedLayoutShell.vue
findings:
  critical: 0
  warning: 5
  info: 4
  total: 9
status: issues_found
---

# Phase 2: Code Review Report

**Reviewed:** 2026-04-15T20:15:00Z
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

Reviewed the watched tags feature implementation spanning backend service/handler layer, database migration, article query filtering, frontend API layer, and Vue components. The feature is well-structured overall with proper parameterized queries (no SQL injection risk), correct optimistic UI patterns, and clean migration SQL. Five warnings were found, all related to error handling: handler error codes mask DB failures, and several `.Find()` calls silently ignore errors. Four info items relate to TypeScript `any` usage.

No critical issues or security vulnerabilities found.

## Warnings

### WR-01: Handler returns 404 for all errors including DB save failures

**File:** `backend-go/internal/domain/topicanalysis/watched_tags_handler.go:31-33` and `:55-57`
**Issue:** Both `WatchTagHandler` and `UnwatchTagHandler` treat every error from the service as a 404 response. However, `WatchTag`/`UnwatchTag` can fail at `db.Save()` (line 35/50 in service), which is a database error and should return HTTP 500, not 404. This masks server errors from the user and makes debugging harder.
**Fix:**
```go
// In WatchTagHandler:
tag, err := WatchTag(database.DB, uint(tagID))
if err != nil {
    if err == gorm.ErrRecordNotFound {
        c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "tag not found or is merged"})
    } else {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
    }
    return
}

// Same pattern for UnwatchTagHandler
```
Note: This fix pairs with WR-04 — the service should return a distinguishable error for "merged" vs "not found".

### WR-02: `.Find()` errors silently ignored in `ListWatchedTags`

**File:** `backend-go/internal/domain/topicanalysis/watched_tags_service.go:76` and `:94`
**Issue:** Two database queries in `ListWatchedTags` ignore their errors:
- Line 76: `db.Where("parent_id IN ?", watchedIDs).Find(&relations)` — if this fails, `relations` is empty/nil, and abstract tag metadata silently drops from the response.
- Line 94: `db.Where("id IN ?", childIDs).Select("id, slug").Find(&childTags)` — if this fails, child slug lookups silently fail.

Both should check `.Error` and return early with the error.
**Fix:**
```go
if err := db.Where("parent_id IN ?", watchedIDs).Find(&relations).Error; err != nil {
    return nil, fmt.Errorf("query tag relations: %w", err)
}
// ...
if err := db.Where("id IN ?", childIDs).Select("id, slug").Find(&childTags).Error; err != nil {
    return nil, fmt.Errorf("query child tag slugs: %w", err)
}
```

### WR-03: `.Find()` error silently ignored in `GetWatchedTagIDsExpanded` and `parseAndExpandWatchedTagIDs`

**File:** `backend-go/internal/domain/topicanalysis/watched_tags_service.go:155` and `backend-go/internal/domain/articles/handler.go:249`
**Issue:** Same pattern as WR-02. In `GetWatchedTagIDsExpanded` (line 155), if the relations query fails, `childTagIDs` will be empty, causing articles from child tags to be silently excluded from watched-tag feeds. In `parseAndExpandWatchedTagIDs` (handler.go:249), the same issue — failed relation lookups silently narrow the article result set.
**Fix:**
```go
// In GetWatchedTagIDsExpanded:
if err := db.Where("parent_id IN ?", watchedIDs).Find(&relations).Error; err != nil {
    return nil, nil, fmt.Errorf("query tag relations for expansion: %w", err)
}

// In parseAndExpandWatchedTagIDs:
if err := database.DB.Where("parent_id IN ?", ids).Find(&relations).Error; err != nil {
    return nil, nil, fmt.Errorf("query tag relations for expansion: %w", err)
}
```

### WR-04: `WatchTag` returns `gorm.ErrRecordNotFound` for merged tags

**File:** `backend-go/internal/domain/topicanalysis/watched_tags_service.go:28-29`
**Issue:** When a tag exists but has `status == "merged"`, the function returns `gorm.ErrRecordNotFound`. This conflates two different conditions: "tag doesn't exist" vs "tag exists but is merged". Callers cannot distinguish between them, leading to generic error messages in the handler. This also makes WR-01 harder to fix correctly since the same error type serves two meanings.
**Fix:**
```go
if tag.Status == "merged" {
    return nil, fmt.Errorf("tag %d is merged and cannot be watched", tagID)
}
```

### WR-05: `selectedArticle` uses `any` type instead of `Article`

**File:** `front/app/features/shell/components/FeedLayoutShell.vue:41`
**Issue:** `const selectedArticle = ref<any>(null)` uses `any` type. Per project conventions (`front/AGENTS.md`: "DON'T use `any` types"), this should be typed as `Article | null`. The same file also uses `any` at line 96 for `filters` (should be `ArticleFilters`) and line 43 for `selectedSummary`. Using `any` disables TypeScript checking on these critical data flows.
**Fix:**
```typescript
import type { Article } from '~/types/article'

const selectedArticle = ref<Article | null>(null)
```

## Info

### IN-01: API layer uses `any` types instead of `WatchedTag` interface

**File:** `front/app/api/watchedTags.ts:16` and `:27`
**Issue:** `apiClient.get<any>` and `return { ...res, data } as any` bypass TypeScript checking. The `WatchedTag` interface is defined in the same file but not used as the API return type. The snake_case→camelCase mapping on lines 19-26 already converts the shape correctly.
**Fix:** Type the API return properly, e.g. `apiClient.get<WatchedTag[]>('/topic-tags/watched')` and remove `as any` casts.

### IN-02: `watch` import placed mid-code in `TagHierarchyRow.vue`

**File:** `front/app/features/topic-graph/components/TagHierarchyRow.vue:75`
**Issue:** `import { watch } from 'vue'` appears at line 75, after function definitions and reactive declarations. While `<script setup>` hoists imports, this placement is confusing and breaks the project's import order convention (Vue/Nuxt imports first). The `ref` import at line 3 already imports from `vue` — both should be in the same import statement.
**Fix:**
```typescript
// Line 3 — merge into single import:
import { ref, watch } from 'vue'
// Remove line 75
```

### IN-03: `buildArticleFilters` uses `any` instead of `ArticleFilters`

**File:** `front/app/features/shell/components/FeedLayoutShell.vue:96`
**Issue:** `const filters: any = {}` should use the `ArticleFilters` type defined in `front/app/types/article.ts`. This would provide compile-time validation that filter keys match what the API expects.
**Fix:**
```typescript
import type { ArticleFilters } from '~/types/article'

const filters: ArticleFilters = {}
```

### IN-04: Non-standard router access in `AppSidebarView.vue`

**File:** `front/app/features/shell/components/AppSidebarView.vue:129`
**Issue:** `const navigateTo = useNuxtApp().$router ? (path) => useNuxtApp().$router.push(path) : () => {}` is a convoluted pattern. Nuxt provides `navigateTo()` (auto-imported) or `useRouter()` composables that handle SSR and edge cases properly. This is pre-existing code, not introduced in this phase.
**Fix:** Replace with `const router = useRouter()` and use `router.push(path)`, or use Nuxt's `navigateTo(path)` directly.

---

_Reviewed: 2026-04-15T20:15:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
