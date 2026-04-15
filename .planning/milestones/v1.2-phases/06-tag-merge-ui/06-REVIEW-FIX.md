---
phase: 06-tag-merge-ui
fixed_at: 2026-04-13T12:30:00Z
review_path: .planning/phases/06-tag-merge-ui/06-REVIEW.md
iteration: 1
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 06: Code Review Fix Report

**Fixed at:** 2026-04-13T12:30:00Z
**Source review:** .planning/phases/06-tag-merge-ui/06-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 5
- Fixed: 5
- Skipped: 0

## Fixed Issues

### CR-01: Merge handler has no transaction — race condition risk

**Files modified:** `backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go`
**Commit:** c8d7d19
**Applied fix:** Wrapped the entire check-rename-merge sequence in `database.DB.Transaction` with `FOR UPDATE` row locking on both source and target tags. Added `fmt` and `gorm.io/gorm` imports. The error handling maps transaction errors back to appropriate HTTP status codes (404 for not found, 400 for already merged / conflict, 500 for internal errors).

### WR-01: Silent data loss — ScanSimilarTagPairs skips pairs on DB error

**Files modified:** `backend-go/internal/domain/topicanalysis/tag_merge_preview.go`
**Commit:** 713a20d
**Applied fix:** Added a `skipped` counter that increments when `DB.First` fails for either tag in a pair. After the loop, if `skipped > 0`, logs the count via `log.Printf`. Added `"log"` to imports.

### WR-02: Duplicate watcher fires loadGraph twice per filter change

**Files modified:** `front/app/features/topic-graph/components/TopicGraphPage.vue`
**Commit:** a5565e2
**Applied fix:** Removed the duplicate `watch([selectedFilterCategoryId, selectedFilterFeedId], ...)` block at lines 806-808, keeping only the single watcher at lines 802-804.

### WR-03: No slug collision check when renaming target tag

**Files modified:** `backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go`
**Commit:** c8d7d19 (combined with CR-01)
**Applied fix:** Added a slug uniqueness check inside the transaction before renaming. Queries for active tags with the same slug (excluding the target tag itself). Returns a `CONFLICT:` prefixed error that maps to HTTP 400 with message "a tag with this name already exists".

### WR-04: Rename and merge are not atomic — partial state on failure

**Files modified:** `backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go`
**Commit:** c8d7d19 (solved by CR-01 transaction wrapping)
**Applied fix:** Solved by the CR-01 transaction. Both the rename and merge now run inside a single `database.DB.Transaction`. If `MergeTags` fails after the rename succeeds, the entire transaction rolls back, undoing the rename. `MergeTags` internally opens its own `database.DB.Transaction` which GORM promotes to a savepoint when already inside a transaction.

---

_Fixed: 2026-04-13T12:30:00Z_
_Fixer: the agent (gsd-code-fixer)_
_Iteration: 1_
