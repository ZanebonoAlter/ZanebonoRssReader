---
phase: 06-tag-merge-ui
plan: 03
subsystem: ui
tags: [vue, typescript, tailwind, tag-merge, modal]

# Dependency graph
requires:
  - phase: 06-tag-merge-ui/06-02
    provides: useTagMergePreviewApi composable, tagMerge types
provides:
  - TagMergePreview.vue — full modal UI for scan/preview/merge/summary flow (D-01 through D-04)
  - Entry point button in TopicGraphPage sidebar
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: [teleport-modal-state-machine, card-based-preview-ui, inline-edit-pattern]

key-files:
  created:
    - front/app/features/topic-graph/components/TagMergePreview.vue
  modified:
    - front/app/features/topic-graph/components/TopicGraphPage.vue

key-decisions:
  - "State machine: idle → scanning → preview → summary → close → idle, clean reset on close"
  - "Trigger button placed in sidebar rail after FeedCategoryFilter for proximity to tag controls"
  - "handleMergeComplete refreshes graph data via loadGraph()"

patterns-established:
  - "Teleport modal with multi-state flow: scanning/preview/summary controlled by single state ref"
  - "Per-card loading via mergingIds array for concurrent merge tracking"

requirements-completed: [CONV-02]

# Metrics
duration: 9min
completed: 2026-04-13
---

# Phase 06 Plan 03: Tag Merge Preview UI Summary

**TagMergePreview modal with card-based candidate display, inline name editing, per-card merge/skip, batch merge, and summary view — wired into TopicGraphPage sidebar**

## Performance

- **Duration:** 9 min
- **Started:** 2026-04-13T13:56:42Z
- **Completed:** 2026-04-13T14:05:46Z
- **Tasks:** 3 (2 auto + 1 checkpoint noted)
- **Files modified:** 2

## Accomplishments
- Created complete TagMergePreview.vue modal with 4-state flow (idle/scanning/preview/summary)
- Cards display source→target tags with similarity badge and article counts (D-01)
- Expandable article titles in two-column layout per card (D-01)
- Inline name editing with pencil icon, save/cancel, custom name per candidate (D-02)
- Per-card merge and skip buttons with loading states (D-03)
- Batch merge button for all remaining candidates (D-03)
- Summary view with merged/skipped/failed counts and detail list (D-04)
- Entry point button added to TopicGraphPage sidebar with editorial dark theme styling

## Task Commits

Each task was committed atomically:

1. **Task 1: Create TagMergePreview.vue with full UI** - `aa0dae2` (feat)
2. **Task 2: Verify TagMergePreview UI end-to-end** - checkpoint:human-verify (noted for human verification)
3. **Task 3: Add entry point button to TopicGraphPage** - `63b671f` (feat)

## Files Created/Modified
- `front/app/features/topic-graph/components/TagMergePreview.vue` — Complete modal component (817 lines): scan trigger, card layout, inline edit, merge/skip actions, summary view
- `front/app/features/topic-graph/components/TopicGraphPage.vue` — Added import, ref, trigger button in sidebar, TagMergePreview instance, handleMergeComplete handler

## Decisions Made
- State machine uses single `state` ref (`idle|scanning|preview|summary`) for clear flow control
- Trigger button placed after FeedCategoryFilter in sidebar rail — close to other tag management controls
- handleMergeComplete calls loadGraph() to refresh all topic data after merge operations
- Used `_summary` prefix for unused parameter to satisfy TypeScript lint

## Deviations from Plan

None - plan executed exactly as written.

## Checkpoint: Task 2 (human-verify)

Task 2 is a `checkpoint:human-verify` gate requiring manual UI verification. The following flows need human testing:
1. Run frontend + backend, navigate to topics page
2. Click "标签合并预览" button in sidebar
3. Verify scan runs and candidates appear in cards
4. Test expand articles, inline name edit, merge, skip, batch merge, and summary flows

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Tag merge UI complete with all D-01 through D-04 requirements implemented
- Typecheck and build both pass
- Ready for human verification of UI flows
- No blockers

## Self-Check: PASSED

- FOUND: front/app/features/topic-graph/components/TagMergePreview.vue
- FOUND: front/app/features/topic-graph/components/TopicGraphPage.vue
- FOUND: aa0dae2 (feat(06-03): create TagMergePreview.vue)
- FOUND: 63b671f (feat(06-03): add tag merge preview entry point)
- pnpm exec nuxi typecheck passed
- pnpm build passed

---
*Phase: 06-tag-merge-ui*
*Completed: 2026-04-13*
