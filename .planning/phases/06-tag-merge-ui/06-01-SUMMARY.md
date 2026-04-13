---
phase: 06-tag-merge-ui
plan: 01
subsystem: api
tags: [go, gin, pgvector, embedding, tag-merge]

# Dependency graph
requires:
  - phase: 01-infrastructure-tag-convergence
    provides: MergeTags transaction, DefaultThresholds, embedding infrastructure
provides:
  - GET /api/topic-tags/merge-preview — returns candidate pairs without auto-executing
  - POST /api/topic-tags/merge-with-name — merges tags with custom target name
  - ScanSimilarTagPairs reusable function extracted from auto_tag_merge scheduler
  - GetCandidateArticleTitles helper for preview article lists
affects: [06-02-PLAN, 06-03-PLAN]

# Tech tracking
tech-stack:
  added: []
  patterns: [preview-then-confirm, custom-name-merge]

key-files:
  created:
    - backend-go/internal/domain/topicanalysis/tag_merge_preview.go
    - backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go
  modified:
    - backend-go/internal/app/router.go

key-decisions:
  - "Reused identical pgvector cross-join SQL from auto_tag_merge.go for consistency"
  - "Source/target determination uses same article count heuristic as scheduler"

patterns-established:
  - "Preview API: returns candidates without mutating state, client decides whether to merge"
  - "Custom-name merge: rename target tag before calling existing MergeTags transaction"

requirements-completed: [CONV-02]

# Metrics
duration: 7min
completed: 2026-04-13
---

# Phase 06 Plan 01: Tag Merge Preview & Custom-Name Merge APIs Summary

**Extracted ScanSimilarTagPairs from auto_tag_merge scheduler, added GET /api/topic-tags/merge-preview and POST /api/topic-tags/merge-with-name APIs**

## Performance

- **Duration:** 7 min
- **Started:** 2026-04-13T13:32:09Z
- **Completed:** 2026-04-13T13:39:30Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Extracted ScanSimilarTagPairs as a reusable function decoupled from scheduler auto-merge
- Preview API returns candidate pairs with similarity scores, article counts, and optional article titles
- Custom-name merge API validates inputs, renames target tag via Slugify, then merges transactionally
- Both handlers follow existing tag_management_handler.go patterns (gin.H responses, error handling)

## Task Commits

Each task was committed atomically:

1. **Task 1: Extract ScanSimilarTagPairs + create preview/merge-with-name handlers** - `803399f` (feat)
2. **Task 2: Register routes in router.go** - `e9e01ec` (feat)

## Files Created/Modified
- `backend-go/internal/domain/topicanalysis/tag_merge_preview.go` - ScanSimilarTagPairs and GetCandidateArticleTitles functions
- `backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go` - ScanMergePreviewHandler, MergeTagsWithCustomNameHandler, RegisterTagMergePreviewRoutes
- `backend-go/internal/app/router.go` - Added RegisterTagMergePreviewRoutes call after RegisterTagManagementRoutes

## Decisions Made
- Reused identical pgvector cross-join SQL from auto_tag_merge.go for consistency with scheduler behavior
- Source/target determination uses same article count heuristic (more articles = target, equal = smaller ID = source) as scheduler
- Threading `include_articles` as optional query param (default false) to keep preview fast for large tag sets
- Threat model mitigations applied: new_name validated non-empty, both tags checked for merged status before merge

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Backend APIs ready for frontend consumption in subsequent plans (06-02, 06-03)
- Preview API provides all fields needed for merge candidate cards (D-01, D-05)
- Custom-name merge API supports D-02 custom naming requirement
- No blockers

## Self-Check: PASSED

- FOUND: backend-go/internal/domain/topicanalysis/tag_merge_preview.go
- FOUND: backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go
- FOUND: backend-go/internal/app/router.go
- FOUND: .planning/phases/06-tag-merge-ui/06-01-SUMMARY.md
- FOUND: 803399f (feat(06-01): extract ScanSimilarTagPairs)
- FOUND: e9e01ec (feat(06-01): register merge preview routes)
- go build ./... passed

---
*Phase: 06-tag-merge-ui*
*Completed: 2026-04-13*
