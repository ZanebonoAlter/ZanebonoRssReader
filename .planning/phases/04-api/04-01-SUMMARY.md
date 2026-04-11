---
phase: 04-api
plan: 01
subsystem: api
tags: [nuxt, pinia, vitest, api-client, unread-count]
requires: []
provides:
  - Scheduler trigger requests now use the shared apiClient response path
  - Article read updates keep sidebar unread counts in sync across cached feed collections
  - Mark-all-as-read clears categorized and uncategorized unread counts locally
affects: [04-02, sidebar-state, scheduler-ui]
tech-stack:
  added: []
  patterns: [shared apiClient usage, local unread-count synchronization helpers, vitest store mocking]
key-files:
  created:
    - front/app/api/scheduler.test.ts
    - front/app/stores/api.test.ts
  modified:
    - front/app/api/scheduler.ts
    - front/app/stores/api.ts
    - front/vitest.config.ts
key-decisions:
  - "Scheduler trigger requests now delegate directly to apiClient.post to keep response parsing and errors consistent."
  - "Unread counts are synchronized locally across feeds and allFeeds instead of forcing a full refetch after every read action."
  - "Mark-all actions clear unread counts through a shared feed matcher so uncategorized feeds are covered by the same path."
patterns-established:
  - "Frontend API helpers should reuse apiClient instead of raw fetch for JSON endpoints."
  - "Pinia store tests can stub Nuxt auto-import globals and mock composable APIs directly in Vitest."
requirements-completed: [API-01, API-02, API-03]
duration: 5 min
completed: 2026-04-11
---

# Phase 04 Plan 01: 前端 API 调用统一 + 状态同步修复 Summary

**Scheduler trigger 统一走 apiClient，文章已读/全部已读会同步刷新分类与未分类 feed 的未读数。**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-11T17:03:01+08:00
- **Completed:** 2026-04-11T17:07:43+08:00
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments
- `front/app/api/scheduler.ts` 移除了 raw fetch，scheduler trigger 与其他前端 API 调用保持同一套错误处理与返回格式。
- `front/app/stores/api.ts` 在文章 read 状态切换后同步更新 `feeds` / `allFeeds` 两套集合，避免 sidebar 未读数漂移。
- `markAllAsRead` 现在会覆盖全部 feed，包括未分类 feed，并补充了对应的 Vitest 回归测试。

## Task Commits

Each task was committed atomically:

1. **Task 1: Replace raw fetch with apiClient in scheduler API (RED)** - `c704174` (test)
2. **Task 1: Replace raw fetch with apiClient in scheduler API (GREEN)** - `2c46d39` (fix)
3. **Task 2: Refresh unreadCount after article update (RED)** - `9ceafa8` (test)
4. **Task 2: Refresh unreadCount after article update (GREEN)** - `203ceec` (fix)
5. **Task 3: Fix markAllAsRead to cover all feeds including uncategorized (RED)** - `27d1cfc` (test)
6. **Task 3: Fix markAllAsRead to cover all feeds including uncategorized (GREEN)** - `5604ae4` (fix)

## Files Created/Modified
- `front/app/api/scheduler.ts` - scheduler trigger 改为直接调用 `apiClient.post`
- `front/app/stores/api.ts` - 新增未读数同步 helper，并修复 mark-all 的本地计数清零逻辑
- `front/app/api/scheduler.test.ts` - 覆盖 scheduler trigger 必须走 `apiClient.post`
- `front/app/stores/api.test.ts` - 覆盖 updateArticle / markAllAsRead 的未读数同步回归
- `front/vitest.config.ts` - 为 Vitest 增加 `~` alias，支持前端模块测试加载

## Decisions Made
- 直接复用 `apiClient.post`，避免 `scheduler.ts` 维护独立的 fetch + JSON 解析分支。
- 未读数同步选择本地精准更新，而不是每次重新拉全量 feed 列表，保持最小改动与即时 UI 反馈。
- `feeds` 与 `allFeeds` 可能指向不同对象集合，因此同步逻辑需要同时覆盖两套列表。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 为 Vitest 补齐 `~` alias 解析**
- **Found during:** Task 1 (RED)
- **Issue:** `scheduler.test.ts` 无法解析 `~/utils/api`，红测被测试环境阻塞。
- **Fix:** 在 `front/vitest.config.ts` 增加 `~ -> ./app` alias。
- **Files modified:** `front/vitest.config.ts`
- **Verification:** `pnpm test:unit -- app/api/scheduler.test.ts`
- **Committed in:** `c704174`

**2. [Rule 3 - Blocking] 为 Pinia store 测试补齐 Nuxt 自动导入全局**
- **Found during:** Task 2 / Task 3 tests
- **Issue:** `defineStore` / `ref` 在 Vitest 中不是 Nuxt 自动注入，`api.ts` 无法在测试环境加载。
- **Fix:** 在 `front/app/stores/api.test.ts` 中显式注入测试用全局并改为动态导入 store。
- **Files modified:** `front/app/stores/api.test.ts`
- **Verification:** `pnpm test:unit -- app/stores/api.test.ts`
- **Committed in:** `5604ae4`

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** 都是测试基础设施阻塞项；修复后按计划完成 API-01/02/03，无额外产品范围扩张。

## Issues Encountered
- `pnpm test:unit` 的全量运行暴露了与本计划无关的既有失败：`app/features/topic-graph/components/TopicTimeline.test.ts > emits filter-change from header`。已记录到 `deferred-items.md`，未在本计划内处理。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 04-01 的前端 API 一致性与未读数同步修复已完成，可继续执行 `04-02-PLAN.md`。
- 若要恢复前端全量单测全绿，需要单独处理 `TopicTimeline.test.ts` 的既有失败。

## Self-Check: PASSED

- Verified files exist: `04-01-SUMMARY.md`, `deferred-items.md`, `wiki/phases/04-01-frontend-api-consistency.md`
- Verified commits exist: `c704174`, `2c46d39`, `9ceafa8`, `203ceec`, `27d1cfc`, `5604ae4`

---
*Phase: 04-api*
*Completed: 2026-04-11*
