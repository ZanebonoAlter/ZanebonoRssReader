---
phase: 08-tag-tree-graph-enhancement
plan: 03
subsystem: backend+frontend
tags: [gorm, postgres, time-filter, is-active, opacity, tag-hierarchy]

# Dependency graph
requires:
  - phase: 08-01
    provides: TopicTag model with Description field
  - phase: 08-02
    provides: Abstract tag description generation
provides:
  - GetTagHierarchy timeRange parameter and isActive flag
  - resolveActiveTagIDs helper for time-based tag activity
  - Frontend time filter UI (全部/7天/30天)
  - Inactive tag dimming (opacity-40)
affects: [08-04, 08-05, 08-06, frontend-topic-graph]

# Tech tracking
tech-stack:
  added: []
  patterns: [time-range filtering via ArticleTopicTag JOIN articles.pub_date, is_active flag propagation through hierarchy]

key-files:
  created: []
  modified:
    - backend-go/internal/domain/topicanalysis/abstract_tag_service.go
    - backend-go/internal/domain/topicanalysis/abstract_tag_handler.go
    - backend-go/internal/domain/topicanalysis/abstract_tag_service_test.go
    - front/app/types/topicTag.ts
    - front/app/api/abstractTags.ts
    - front/app/features/topic-graph/components/TagHierarchy.vue
    - front/app/features/topic-graph/components/TagHierarchyRow.vue

key-decisions:
  - "Used ArticleTopicTag JOIN articles (not AISummaryTopic) for time filtering — direct article-to-tag link"
  - "Used articles.pub_date column (not published_at) — matches actual Article model"
  - "GetUnclassifiedTags also gained timeRange parameter for consistency"
  - "Invalid time_range values silently treated as no filter per T-08-04"

patterns-established:
  - "resolveActiveTagIDs: shared helper for time-based tag activity across GetTagHierarchy and GetUnclassifiedTags"
  - "Opacity-based inactive dimming: opacity-40 on wrapper div preserves layout"

requirements-completed: []

# Metrics
duration: 9min
completed: 2026-04-14
---

# Phase 08 Plan 03: 后端时间筛选 API + 前端时间筛选 UI Summary

**Tag hierarchy gains time-range filtering (7d/30d) via ArticleTopicTag JOIN articles.pub_date, with is_active flag and frontend opacity-40 dimming for inactive tags**

## Performance

- **Duration:** 9 min
- **Started:** 2026-04-14T05:40:53Z
- **Completed:** 2026-04-14T05:49:53Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Backend GetTagHierarchy supports timeRange parameter (7d/30d/empty) filtering by article pub_date
- TagHierarchyNode includes is_active boolean flag for frontend consumption
- Frontend TagHierarchy displays time filter buttons (全部/7天/30天) that trigger API reload
- Inactive tags rendered with opacity-40 dimming while preserving hierarchy structure
- T-08-04 mitigated: invalid time_range values silently ignored (no filter applied)

## Task Commits

Each task was committed atomically:

1. **Task 1: 后端 GetTagHierarchy 时间筛选 + isActive 标记** - `9cc9f34` (feat)
2. **Task 2: 前端 TagHierarchy 时间筛选 UI + TagHierarchyRow 置灰** - `80c33dd` (feat)

## Files Created/Modified
- `backend-go/internal/domain/topicanalysis/abstract_tag_service.go` - Added IsActive field, timeRange param, resolveActiveTagIDs helper
- `backend-go/internal/domain/topicanalysis/abstract_tag_handler.go` - Handler reads time_range query param
- `backend-go/internal/domain/topicanalysis/abstract_tag_service_test.go` - Tests for resolveActiveTagIDs and candidateIDSetToSlice
- `front/app/types/topicTag.ts` - Added isActive: boolean to TagHierarchyNode
- `front/app/api/abstractTags.ts` - Added is_active mapping, timeRange param to fetchHierarchy
- `front/app/features/topic-graph/components/TagHierarchy.vue` - Time filter UI (全部/7天/30天)
- `front/app/features/topic-graph/components/TagHierarchyRow.vue` - Opacity-40 wrapper for inactive tags

## Decisions Made
- Used ArticleTopicTag JOIN articles (not AISummaryTopic) for time filtering — direct article-to-tag relationship
- Used articles.pub_date column matching actual Article model (plan referenced non-existent published_at)
- GetUnclassifiedTags also updated with timeRange for UI consistency
- Invalid time_range values silently treated as no filter per threat model T-08-04

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected column name from published_at to pub_date**
- **Found during:** Task 1 (GetTagHierarchy implementation)
- **Issue:** Plan referenced `articles.published_at` but actual Article model uses `PubDate` mapped to `pub_date` column
- **Fix:** Used `articles.pub_date >= ?` in the GORM query
- **Files modified:** abstract_tag_service.go
- **Verification:** `go build ./...` passes
- **Committed in:** 9cc9f34 (Task 1 commit)

**2. [Rule 1 - Bug] Used ArticleTopicTag instead of AISummaryTopic for article-tag join**
- **Found during:** Task 1 (time filtering query design)
- **Issue:** Plan referenced AISummaryTopic JOIN but ArticleTopicTag is the direct article-to-tag link
- **Fix:** Query uses `article_topic_tags JOIN articles ON articles.id = article_topic_tags.article_id`
- **Files modified:** abstract_tag_service.go
- **Verification:** `go build ./...` passes
- **Committed in:** 9cc9f34 (Task 1 commit)

**3. [Rule 2 - Missing Critical] Added timeRange to GetUnclassifiedTags**
- **Found during:** Task 1 (GetUnclassifiedTags also returns TagHierarchyNode)
- **Issue:** GetUnclassifiedTags returns TagHierarchyNode but plan only modified GetTagHierarchy
- **Fix:** Added timeRange parameter and resolveActiveTagIDs call to GetUnclassifiedTags
- **Files modified:** abstract_tag_service.go
- **Verification:** All tests pass, build succeeds
- **Committed in:** 9cc9f34 (Task 1 commit)

**4. [Rule 2 - Missing Critical] Added time_range input validation per T-08-04**
- **Found during:** Task 1 (resolveActiveTagIDs implementation)
- **Issue:** Threat model T-08-04 requires validating time_range to only accept predefined values
- **Fix:** resolveActiveTagIDs treats invalid values as no filter (whitelist approach with "7d"/"30d")
- **Files modified:** abstract_tag_service.go
- **Verification:** Unit test TestResolveActiveTagIDsInvalidValue passes
- **Committed in:** 9cc9f34 (Task 1 commit)

**5. [Rule 3 - Blocking] Corrected handler file path**
- **Found during:** Task 1 (reading source files)
- **Issue:** Plan referenced `handler.go` but actual file is `abstract_tag_handler.go`
- **Fix:** Read and modified the correct file
- **Files modified:** abstract_tag_handler.go
- **Committed in:** 9cc9f34 (Task 1 commit)

---

**Total deviations:** 5 auto-fixed (2 bug, 2 missing critical, 1 blocking)
**Impact on plan:** All corrections necessary for correctness, security, and consistency. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Time-range filtering fully functional end-to-end (backend API + frontend UI)
- Ready for 08-04-PLAN.md (abstract tag graph visualization or next plan)

## Self-Check: PASSED

- FOUND: backend-go/internal/domain/topicanalysis/abstract_tag_service.go
- FOUND: backend-go/internal/domain/topicanalysis/abstract_tag_handler.go
- FOUND: backend-go/internal/domain/topicanalysis/abstract_tag_service_test.go
- FOUND: front/app/types/topicTag.ts
- FOUND: front/app/api/abstractTags.ts
- FOUND: front/app/features/topic-graph/components/TagHierarchy.vue
- FOUND: front/app/features/topic-graph/components/TagHierarchyRow.vue
- FOUND: 9cc9f34 (feat 08-03)
- FOUND: 80c33dd (feat 08-03)

---
*Phase: 08-tag-tree-graph-enhancement*
*Completed: 2026-04-14*
