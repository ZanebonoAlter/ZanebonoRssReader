---
phase: 08-tag-tree-graph-enhancement
plan: 06
subsystem: backend+frontend
tags: [reassign, tag-hierarchy, abstract-tags, gin, vue-3, modal]

# Dependency graph
requires:
  - phase: 08-02
    provides: Abstract tag hierarchy API and service layer
  - phase: 08-03
    provides: TagHierarchyNode with is_active, time filtering
provides:
  - ReassignTagParent service function (transaction-based tag reassignment)
  - POST /topic-tags/:id/reassign endpoint
  - Frontend reassign modal with abstract tag candidate list
  - Reassign operation button in TagHierarchyRow
affects: [tag-hierarchy, abstract-tags, topic-graph]

# Tech tracking
tech-stack:
  added: []
  patterns: [transaction-based tag reassignment, abstract tag candidate collection from tree, modal with scrollable candidate list]

key-files:
  created: []
  modified:
    - backend-go/internal/domain/topicanalysis/abstract_tag_service.go
    - backend-go/internal/domain/topicanalysis/abstract_tag_handler.go
    - backend-go/internal/domain/topicanalysis/abstract_tag_service_test.go
    - front/app/api/abstractTags.ts
    - front/app/features/topic-graph/components/TagHierarchyRow.vue
    - front/app/features/topic-graph/components/TagHierarchy.vue

key-decisions:
  - "ReassignTagParent blocks abstract tags with children to prevent nesting confusion (threat model T-08-08)"
  - "Frontend collects abstract tag candidates from the full tree (excluding self) rather than calling a separate API"
  - "Reassign button uses arrow-right icon (mdi:arrow-right-bold) to distinguish from detach (link-off)"

patterns-established:
  - "Transaction-based tag reassignment: delete old relation + create new relation in single transaction"
  - "Modal candidate list: collect from existing tree data, no extra API call needed"

requirements-completed: []

# Metrics
duration: 8min
completed: 2026-04-14
---

# Phase 08 Plan 06: 标签树节点手动归类 Summary

**Tag tree nodes can be manually reassigned to different abstract parents via a modal with abstract tag candidates, backed by a transaction-based API**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-14T14:00:00Z
- **Completed:** 2026-04-14T14:08:00Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Backend `ReassignTagParent` function with transaction-based tag reassignment (delete old relation, create new one)
- Backend `POST /topic-tags/:id/reassign` endpoint with validation
- Blocks reassignment of abstract tags that have children (threat model T-08-08)
- Frontend `reassignTag` API function in abstractTags.ts
- Reassign button (arrow icon) in TagHierarchyRow for child nodes
- Reassign modal in TagHierarchy showing all abstract tags as potential parents
- After reassignment, hierarchy tree refreshes automatically

## Task Commits

Each task was committed atomically (TDD pattern):

1. **Task 1 (RED): Failing tests for ReassignTagParent validation** - `4e0bcf7` (test)
2. **Task 1 (GREEN): Implement ReassignTagParent + handler + route** - `d01142c` (feat)
3. **Task 2: Frontend reassign modal and operation menu** - `e918fbf` (feat)

## Files Created/Modified

- `backend-go/internal/domain/topicanalysis/abstract_tag_service.go` - Added ReassignTagParent function (transaction-based)
- `backend-go/internal/domain/topicanalysis/abstract_tag_handler.go` - Added ReassignTagHandler, registered route
- `backend-go/internal/domain/topicanalysis/abstract_tag_service_test.go` - Added validation tests for ReassignTagParent
- `front/app/api/abstractTags.ts` - Added reassignTag API function
- `front/app/features/topic-graph/components/TagHierarchyRow.vue` - Added reassign button with arrow icon, emits 'reassign' event
- `front/app/features/topic-graph/components/TagHierarchy.vue` - Added reassign modal state, candidate collection, confirm/cancel handlers, modal UI

## Decisions Made

- ReassignTagParent blocks abstract tags with children to prevent nesting confusion (per threat model T-08-08)
- Frontend collects abstract tag candidates from the existing tree data rather than calling a separate API endpoint — simpler and avoids extra network roundtrip
- Reassign button uses arrow-right icon (mdi:arrow-right-bold) to visually distinguish from detach (mdi:link-off)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Tag tree nodes can now be manually reassigned to different abstract parents
- All 6 plans of Phase 08 are complete
- Phase 08 complete, ready for next phase or milestone wrap-up

## Self-Check: PASSED

- FOUND: backend-go/internal/domain/topicanalysis/abstract_tag_service.go
- FOUND: backend-go/internal/domain/topicanalysis/abstract_tag_handler.go
- FOUND: backend-go/internal/domain/topicanalysis/abstract_tag_service_test.go
- FOUND: front/app/api/abstractTags.ts
- FOUND: front/app/features/topic-graph/components/TagHierarchyRow.vue
- FOUND: front/app/features/topic-graph/components/TagHierarchy.vue
- FOUND: 4e0bcf7 (test 08-06)
- FOUND: d01142c (feat 08-06)
- FOUND: e918fbf (feat 08-06)
- go test: PASSED
- go build: PASSED
- pnpm exec nuxi typecheck: PASSED
- pnpm build: PASSED

---

*Phase: 08-tag-tree-graph-enhancement*
*Completed: 2026-04-14*
