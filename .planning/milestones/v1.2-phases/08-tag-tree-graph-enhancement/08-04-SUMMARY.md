---
phase: 08-tag-tree-graph-enhancement
plan: 04
subsystem: ui
tags: [three.js, topic-graph, abstract-tags, glow-effect, sidebar, vue-3, nuxt-4]

# Dependency graph
requires:
  - phase: 08-02
    provides: "Abstract tag descriptions and hierarchy API"
  - phase: 08-01
    provides: "Backend is_abstract field on GraphNode API response"
provides:
  - "Abstract tag node glow effect in 3D topic graph"
  - "Abstract tag detail panel with child tags and filtered article timeline"
  - "isAbstract field propagated through TopicGraphSceneNode pipeline"
affects: [topic-graph, tag-hierarchy, abstract-tags]

# Tech tracking
tech-stack:
  added: []
  patterns: ["Layered glow effect via concentric transparent spheres in Three.js", "Sidebar detail panel driven by abstract tag hierarchy API"]

key-files:
  created: []
  modified:
    - front/app/api/topicGraph.ts
    - front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts
    - front/app/features/topic-graph/components/TopicGraphCanvas.client.vue
    - front/app/features/topic-graph/components/TopicGraphSidebar.vue
    - front/app/features/topic-graph/components/TopicGraphPage.vue
    - front/app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts
    - front/app/features/topic-graph/utils/buildDisplayedTopicGraph.test.ts

key-decisions:
  - "Used concentric transparent spheres for glow effect instead of emissive material (MeshBasicMaterial doesn't support emissive)"
  - "Sidebar fetches full hierarchy via fetchHierarchy() and recursively searches for matching slug"

patterns-established:
  - "Glow layers: 2 semi-transparent spheres at radius *1.6 and *2.1 with lightened color and decreasing opacity"
  - "Abstract detail panel: fetches hierarchy, filters children, shows clickable child tag list + filtered article timeline"

requirements-completed: []

# Metrics
duration: 11min
completed: 2026-04-14
---

# Phase 08 Plan 04: Abstract Tag Glow & Detail Panel Summary

**Abstract tag nodes get concentric sphere glow in 3D graph and a sidebar detail panel showing child tags with filtered article timeline**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-14T05:53:05Z
- **Completed:** 2026-04-14T06:04:29Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Abstract tag nodes visually distinguished with 2-layer concentric glow spheres in 3D topic graph
- `isAbstract` field propagated from API response through `GraphNode` → `TopicGraphSceneNode` → canvas rendering
- Clicking abstract tag nodes opens a detail panel showing child tags (with filter toggle) and a filtered article timeline

## Task Commits

Each task was committed atomically (TDD: test → feature):

1. **Task 1 (RED): isAbstract propagation tests** - `d93f92b` (test)
2. **Task 1 (GREEN): Abstract tag glow effect** - `d4c5062` (feat)
3. **Task 2: Abstract tag detail panel** - `05974d7` (feat)

## Files Created/Modified
- `front/app/api/topicGraph.ts` - Added `is_abstract?: boolean` to `GraphNode` interface
- `front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts` - Added `isAbstract: boolean` to `TopicGraphSceneNode`, propagated from normalized node
- `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue` - Added glow effect (2 concentric transparent spheres) + `lightenColor()` utility for abstract nodes
- `front/app/features/topic-graph/components/TopicGraphSidebar.vue` - Added abstract tag detail panel with child tag list and filtered article timeline
- `front/app/features/topic-graph/components/TopicGraphPage.vue` - Added `abstractNodeSlug` state, wired `handleNodeClick` to pass `isAbstract`
- `front/app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts` - Added tests for `isAbstract` propagation
- `front/app/features/topic-graph/utils/buildDisplayedTopicGraph.test.ts` - Added `isAbstract: false` to test fixtures

## Decisions Made
- **Concentric spheres over emissive material:** `TopicGraphCanvas` uses `MeshBasicMaterial` (not `MeshStandardMaterial`), which lacks `emissive` property. Instead, 2 semi-transparent spheres at 1.6x and 2.1x radius create a soft glow using lightened color and decreasing opacity (0.3 → 0.15).
- **Hierarchy API for sidebar children:** The sidebar uses `useAbstractTagApi().fetchHierarchy()` to get the full nested tree, then recursively searches for the matching slug to extract direct children. This avoids needing a dedicated "children of X" endpoint.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added isAbstract to buildDisplayedTopicGraph test fixtures**
- **Found during:** Task 1 (TDD RED phase)
- **Issue:** Adding required `isAbstract` field to `TopicGraphSceneNode` broke existing `buildDisplayedTopicGraph.test.ts` which constructed nodes without it
- **Fix:** Added `isAbstract: false` to all test fixture objects
- **Files modified:** front/app/features/topic-graph/utils/buildDisplayedTopicGraph.test.ts
- **Verification:** All tests pass (`pnpm test:unit`)
- **Committed in:** d4c5062 (Task 1 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Minimal — test fixture update required by the new required field. No scope creep.

## Issues Encountered
- Windows `date -u` not available — used PowerShell `[DateTime]::UtcNow` workaround for timestamps
- GitNexus has 3 indexed repos; must specify `repo: "my-robot"` for all queries

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Abstract tag visual differentiation (glow) and interaction (detail panel) are complete
- Topic graph 3D visualization now supports both regular and abstract tag nodes with distinct behaviors
- Ready for any future enhancements to abstract tag interaction (e.g., expand/collapse in graph, drag-to-merge)

## Self-Check: PASSED

All 7 key files verified present on disk. All 3 commits (d93f92b, d4c5062, 05974d7) verified in git log.

---
*Phase: 08-tag-tree-graph-enhancement*
*Completed: 2026-04-14*
