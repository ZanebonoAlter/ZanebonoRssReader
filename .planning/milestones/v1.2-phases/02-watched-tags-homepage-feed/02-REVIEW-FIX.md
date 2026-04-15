---
phase: 02-watched-tags-homepage-feed
fixed_at: 2026-04-15T20:30:00Z
review_path: .planning/phases/02-watched-tags-homepage-feed/02-REVIEW.md
iteration: 1
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 2: Code Review Fix Report

**Fixed at:** 2026-04-15T20:30:00Z
**Source review:** .planning/phases/02-watched-tags-homepage-feed/02-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 5
- Fixed: 5
- Skipped: 0

## Fixed Issues

### WR-01: Handler returns 404 for all errors including DB save failures

**Files modified:** `backend-go/internal/domain/topicanalysis/watched_tags_handler.go`
**Commit:** 1ed3700
**Applied fix:** Both `WatchTagHandler` and `UnwatchTagHandler` now check if the error is `gorm.ErrRecordNotFound` to return 404, and return HTTP 500 with the error message for all other errors (DB save failures, etc.). Added `gorm.io/gorm` import for the error constant.

### WR-02: .Find() errors silently ignored in ListWatchedTags

**Files modified:** `backend-go/internal/domain/topicanalysis/watched_tags_service.go`
**Commit:** da67fe6
**Applied fix:** Both `.Find()` calls in `ListWatchedTags` now check `.Error` and return wrapped errors: `"query tag relations: %w"` for the parent_id relation query and `"query child tag slugs: %w"` for the child tag slug lookup.

### WR-03: .Find() error silently ignored in GetWatchedTagIDsExpanded and parseAndExpandWatchedTagIDs

**Files modified:** `backend-go/internal/domain/topicanalysis/watched_tags_service.go`, `backend-go/internal/domain/articles/handler.go`
**Commit:** 5435b9e
**Applied fix:** Added error checks after `.Find()` in `GetWatchedTagIDsExpanded` (watched_tags_service.go:160) and `parseAndExpandWatchedTagIDs` (handler.go:249). Both return wrapped errors with context. Added `fmt` import to articles/handler.go.

### WR-04: WatchTag returns gorm.ErrRecordNotFound for merged tags

**Files modified:** `backend-go/internal/domain/topicanalysis/watched_tags_service.go`
**Commit:** 1f900b6
**Applied fix:** Changed merged tag error from `gorm.ErrRecordNotFound` to `fmt.Errorf("tag %d is merged and cannot be watched", tagID)`. This allows the handler (WR-01) to properly distinguish "tag not found" from "tag is merged". Added `fmt` import to the service file.

### WR-05: selectedArticle uses `any` type instead of `Article`

**Files modified:** `front/app/features/shell/components/FeedLayoutShell.vue`
**Commit:** d9f38d3
**Applied fix:** Added proper type imports (`Article`, `ArticleFilters` from `~/types/article`, `AISummary` from `~/types/ai`). Replaced `ref<any>(null)` with `ref<Article | null>(null)` for `selectedArticle`, `ref<AISummary | null>(null)` for `selectedSummary`, and `any` with `ArticleFilters` for `filters` in `buildArticleFilters`. Verified with `pnpm exec nuxi typecheck`.

---

_Fixed: 2026-04-15T20:30:00Z_
_Fixer: the agent (gsd-code-fixer)_
_Iteration: 1_
