---
phase: 06-tag-merge-ui
plan: 02
subsystem: api
tags: [typescript, frontend, tag-merge, api-client]

# Dependency graph
requires:
  - phase: 06-tag-merge-ui/06-01
    provides: GET /api/topic-tags/merge-preview and POST /api/topic-tags/merge-with-name backend endpoints
provides:
  - front/app/types/tagMerge.ts — typed interfaces for merge preview candidates and merge results
  - front/app/api/tagMergePreview.ts — useTagMergePreviewApi composable for preview scan and custom-name merge
affects: [06-03-PLAN]

# Tech tracking
tech-stack:
  added: []
  patterns: [useXxxApi-composable, camelCase-type-mapping]

key-files:
  created:
    - front/app/types/tagMerge.ts
    - front/app/api/tagMergePreview.ts
  modified: []

key-decisions:
  - "Article titles optional via include_articles query param, matching backend behavior"
  - "POST body uses snake_case (backend convention), types use camelCase (frontend convention)"

patterns-established:
  - "useTagMergePreviewApi follows existing useTopicGraphApi pattern with buildQueryParams for GET"

requirements-completed: [CONV-02]

# Metrics
duration: 4min
completed: 2026-04-13
---

# Phase 06 Plan 02: Tag Merge Types & API Layer Summary

**Frontend TypeScript types and API composable for merge preview scan and custom-name merge, bridging backend snake_case to frontend camelCase**

## Performance

- **Duration:** 4 min
- **Started:** 2026-04-13T13:44:31Z
- **Completed:** 2026-04-13T13:48:52Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Defined 6 TypeScript interfaces matching backend response shape with camelCase mapping
- Created useTagMergePreviewApi composable following existing useTopicGraphApi pattern
- API functions handle snake_case→camelCase boundary: POST body uses snake_case, types use camelCase

## Task Commits

Each task was committed atomically:

1. **Task 1: Define TypeScript types for merge preview** - `e802f83` (feat)
2. **Task 2: Create API functions for preview and custom merge** - `1cb7f2a` (feat)

## Files Created/Modified
- `front/app/types/tagMerge.ts` — TagMergeCandidate, MergePreviewResponse, MergeWithCustomNameRequest, MergeWithCustomNameResult, MergeSummary, ArticleTitlePreview interfaces
- `front/app/api/tagMergePreview.ts` — useTagMergePreviewApi with scanMergePreview (GET) and mergeTagsWithCustomName (POST)

## Decisions Made
- Article titles arrays (sourceArticleTitles, targetArticleTitles) are optional — only populated when include_articles=true query param is passed
- MergeSummary type included for Plan 03 batch result display, even though not used in this plan's API calls
- Used apiClient.buildQueryParams for GET query string construction, matching topicGraph.ts pattern

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- API layer ready for UI component consumption in Plan 03
- Types provide all fields needed for merge candidate cards (D-01, D-05)
- Custom-name merge API supports D-02 custom naming requirement
- No blockers

## Self-Check: PASSED

- FOUND: front/app/types/tagMerge.ts
- FOUND: front/app/api/tagMergePreview.ts
- FOUND: e802f83 (feat(06-02): define TypeScript types)
- FOUND: 1cb7f2a (feat(06-02): create API functions)
- pnpm exec nuxi typecheck passed

---
*Phase: 06-tag-merge-ui*
*Completed: 2026-04-13*
