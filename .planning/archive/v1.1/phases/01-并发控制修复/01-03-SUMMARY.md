---
phase: 01-并发控制修复
plan: 03
subsystem: api
tags: [go, scheduler, websocket, auto-refresh, auto-summary]
requires:
  - phase: 01-并发控制修复
    provides: Firecrawl trigger 接口与回归测试已建立按契约修复的最小变更模式
provides:
  - auto_refresh 在所有 feed 刷新完成后广播 auto_refresh_complete WebSocket 消息
  - auto_refresh_complete 消息包含 triggered_feeds、stale_reset_feeds、duration_seconds、timestamp
  - 新增回归测试锁定完成通知 JSON 契约与广播接线顺序
affects: [scheduler-handler, websocket, frontend-tracking, auto-summary]
tech-stack:
  added: []
  patterns: ["调度器完成通知使用显式 WebSocket payload struct", "在触发下游 auto-summary 前先广播 auto_refresh_complete"]
key-files:
  created: []
  modified:
    - backend-go/internal/platform/ws/hub.go
    - backend-go/internal/jobs/auto_refresh.go
    - backend-go/internal/jobs/auto_refresh_test.go
key-decisions:
  - "复用现有 ws.GetHub().BroadcastRaw 基础设施，而不是新增状态查询 API。"
  - "回归测试拆成 JSON 契约验证 + 源码接线验证，避免为 WebSocket 引入额外测试夹具。"
patterns-established:
  - "Auto-refresh 完成后如果需要让前端感知时机，应先广播完成消息，再触发下游 scheduler。"
  - "面向前端的新 WebSocket 事件应在 ws/hub.go 中声明显式消息结构体，并锁定 JSON 字段名。"
requirements-completed: [CONC-01]
duration: 3 min
completed: 2026-04-11
---

# Phase 01 Plan 03: Auto-refresh 完成通知 Summary

**Auto-refresh 现在会在所有 feed 刷新结束后广播 `auto_refresh_complete` WebSocket 消息，并在通知发出后再触发 auto-summary。**

## Performance

- **Duration:** 3 min
- **Started:** 2026-04-11T12:19:26+08:00
- **Completed:** 2026-04-11T12:22:13+08:00
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- 在 `ws/hub.go` 中新增 `AutoRefreshCompleteMessage`，明确完成通知的 JSON 契约
- `triggerAutoSummaryAfterRefreshes` 现在会计算刷新耗时、广播 `auto_refresh_complete`，然后保持既有 auto-summary 触发行为
- 新增回归测试覆盖消息 JSON 结构和“先广播再触发 auto-summary”的关键接线顺序

## Task Commits

Each task was committed atomically:

1. **Task 1: Define AutoRefreshCompleteMessage type in ws/hub.go** - `11122eb` (feat)
2. **Task 2: Broadcast completion in triggerAutoSummaryAfterRefreshes** - `f8920e7` (feat)

## Files Created/Modified
- `backend-go/internal/platform/ws/hub.go` - 新增 `AutoRefreshCompleteMessage`，为前端提供稳定完成通知 payload
- `backend-go/internal/jobs/auto_refresh.go` - 在 feeds 刷新完成后广播 `auto_refresh_complete`，并把通知放在 auto-summary 触发之前
- `backend-go/internal/jobs/auto_refresh_test.go` - 增加 JSON 契约测试与源码级广播顺序回归测试

## Decisions Made
- 选择复用现有 WebSocket Hub，而不是再加一个状态轮询 API，因为 Phase 目标只是让前端感知“完成时机”
- 广播内容使用独立 struct + JSON tag，而不是直接拼 map，避免前端字段名漂移

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 01 的 3 个计划均已完成，当前并发控制修复阶段具备进入后续 phase 的条件
- 前端若要感知 auto-refresh 完成时机，可直接监听 `auto_refresh_complete` 事件并读取现有字段

## Self-Check: PASSED

- Verified `.planning/phases/01-并发控制修复/01-03-SUMMARY.md` exists on disk
- Verified `wiki/phases/01-03-auto-refresh-completion.md` exists on disk
- Verified commits `11122eb` and `f8920e7` exist in git history

---
*Phase: 01-并发控制修复*
*Completed: 2026-04-11*
