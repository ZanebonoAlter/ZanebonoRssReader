---
phase: 04-api
plan: 02
subsystem: api
tags: [go, gin, scheduler, status-api, jobs]
requires:
  - phase: 04-api
    provides: 前端已统一 scheduler trigger 调用与本地状态同步，现继续统一后端 scheduler status 契约
provides:
  - 所有 scheduler status API 统一返回 `name/status/check_interval/next_run/is_executing`
  - 五个 jobs scheduler 的 `GetStatus()` 统一返回 `SchedulerStatusResponse`
  - handler 回归测试覆盖单个 scheduler 与 alias 路由的统一响应结构
affects: [scheduler-ui, handler-tests, runtime-status]
tech-stack:
  added: []
  patterns: [shared scheduler status struct, unix timestamp next_run, handler-side legacy status normalization]
key-files:
  created:
    - backend-go/internal/jobs/scheduler_status_response_test.go
    - wiki/phases/04-02-scheduler-status-format.md
  modified:
    - backend-go/internal/jobs/handler.go
    - backend-go/internal/jobs/auto_refresh.go
    - backend-go/internal/jobs/firecrawl.go
    - backend-go/internal/jobs/content_completion.go
    - backend-go/internal/jobs/auto_summary.go
    - backend-go/internal/jobs/preference_update.go
    - backend-go/internal/jobs/content_completion_test.go
    - backend-go/internal/jobs/handler_test.go
    - wiki/index.md
    - wiki/log.md
key-decisions:
  - "统一 status API 以 `SchedulerStatusResponse` 为唯一必填契约，`next_run` 统一为 Unix 时间戳。"
  - "保留 `GetTaskStatusDetails()` 作为队列概览补充接口，避免 `GetTasksStatus` 丢失 Firecrawl / Content Completion 的任务统计。"
  - "handler 对仍返回 map 的 scheduler 做兼容归一化，避免本次计划外扩大到 digest 实现。"
patterns-established:
  - "Scheduler status 接口必须至少返回 `name/status/check_interval/next_run/is_executing` 五个字段。"
  - "需要额外任务统计时，走专门的 task-details helper，而不是污染统一 status 契约。"
requirements-completed: [API-04]
duration: 10 min
completed: 2026-04-11
---

# Phase 04 Plan 02: 统一后端 scheduler status API 返回格式 Summary

**后端 scheduler status API 现统一返回稳定的五字段结构，前端可用同一解析逻辑消费 Auto Refresh、Auto Summary、Content Completion、Preference Update 与 Firecrawl 状态。**

## Performance

- **Duration:** 10 min
- **Started:** 2026-04-11T17:14:27+08:00
- **Completed:** 2026-04-11T17:24:26+08:00
- **Tasks:** 3
- **Files modified:** 10

## Accomplishments
- 在 `backend-go/internal/jobs/handler.go` 中定义 `SchedulerStatusResponse`，固定 `name/status/check_interval/next_run/is_executing` 五个字段与 snake_case JSON tag。
- 五个 scheduler 的 `GetStatus()` 统一改为返回 `SchedulerStatusResponse`，并把 `next_run` 统一规范为 Unix 时间戳。
- `handler_test.go` 与新增的 `scheduler_status_response_test.go` 覆盖 struct 契约、scheduler 返回值以及单个/alias status endpoint 的统一响应结构。

## Task Commits

Each task was committed atomically:

1. **Task 1: Define unified SchedulerStatusResponse struct (RED)** - `c0c884f` (test)
2. **Task 1: Define unified SchedulerStatusResponse struct (GREEN)** - `057c747` (feat)
3. **Task 2: Update all scheduler GetStatus() methods to return unified struct (RED)** - `3a3fcff` (test)
4. **Task 2: Update all scheduler GetStatus() methods to return unified struct (GREEN)** - `875b090` (feat)
5. **Task 3: Update API handlers to return unified response** - `216981d` (test)

## Files Created/Modified
- `backend-go/internal/jobs/handler.go` - 定义统一 status struct，并在 handler 中兼容归一化 legacy map 响应。
- `backend-go/internal/jobs/auto_refresh.go` - `GetStatus()` 改为返回 `SchedulerStatusResponse`，从 scheduler task 读取 Unix `next_run`。
- `backend-go/internal/jobs/auto_summary.go` - 统一 auto-summary status 返回结构与时间戳类型。
- `backend-go/internal/jobs/content_completion.go` - `GetStatus()` 只返回统一字段，额外队列统计下沉到 `GetTaskStatusDetails()`。
- `backend-go/internal/jobs/firecrawl.go` - 新增 `nextRun` 跟踪并统一 Firecrawl status 返回结构。
- `backend-go/internal/jobs/preference_update.go` - 统一 Preference Update status 返回结构。
- `backend-go/internal/jobs/scheduler_status_response_test.go` - 覆盖 struct 契约与五个 scheduler 的统一 status 格式。
- `backend-go/internal/jobs/content_completion_test.go` - 调整为验证统一 status + task details 分层。
- `backend-go/internal/jobs/handler_test.go` - 验证 status endpoint 与 alias 路由统一响应结构。
- `wiki/phases/04-02-scheduler-status-format.md` / `wiki/index.md` / `wiki/log.md` - 更新本地知识库记录本次 API 契约收敛。

## Decisions Made
- `next_run` 统一使用 Unix 时间戳，避免不同 scheduler 混用 `time.Time` / RFC3339 string。
- `name` 使用面向前端展示的调度器名（如 `Auto Refresh`、`Firecrawl Crawler`），不再暴露 handler 内部 slug。
- `GetTasksStatus` 继续保留任务级补充信息，但不再依赖统一 status 契约承载额外字段。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] handler 需要先适配新的 GetStatus 返回类型**
- **Found during:** Task 2 (Update all scheduler GetStatus() methods to return unified struct)
- **Issue:** `GetStatus()` 从 `map[string]interface{}` 改为 `SchedulerStatusResponse` 后，`handler.go` 与 `GetTasksStatus` 会立刻编译失败。
- **Fix:** 在 `handler.go` 中加入统一 status 归一化逻辑，并为仍需要额外统计的 scheduler 增加 `GetTaskStatusDetails()` 辅助接口。
- **Files modified:** `backend-go/internal/jobs/handler.go`, `backend-go/internal/jobs/content_completion.go`, `backend-go/internal/jobs/firecrawl.go`
- **Verification:** `go test ./internal/jobs -v -run TestSchedulerStatusFormat`; `go test ./internal/jobs -v`
- **Committed in:** `875b090`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** 这是统一 status 契约时必须同步修复的编译阻塞项；修复后仍保持最小改动范围在 `internal/jobs` 与相关测试内。

## Issues Encountered
- Task 3 的 handler 行为已在 Task 2 的阻塞修复中完成，因此本任务的新增 endpoint 回归测试直接通过，没有再引入额外 handler 代码改动。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 04 的 scheduler status 契约已固定，前端后续可直接依赖统一五字段解析逻辑。
- 本计划未触碰 `STATE.md` / `ROADMAP.md` / `REQUIREMENTS.md`，可由编排器后续统一推进收尾。

## Self-Check: PASSED

- Verified files exist: `04-02-SUMMARY.md`, `wiki/phases/04-02-scheduler-status-format.md`, `wiki/index.md`, `wiki/log.md`
- Verified commits exist: `c0c884f`, `057c747`, `3a3fcff`, `875b090`, `216981d`

---
*Phase: 04-api*
*Completed: 2026-04-11*
