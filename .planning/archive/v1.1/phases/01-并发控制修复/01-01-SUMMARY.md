---
phase: 01-并发控制修复
plan: 01
subsystem: api
tags: [go, scheduler, http, triggernow]
requires: []
provides:
  - TriggerNow 锁定失败响应统一使用 http.StatusConflict 常量
  - 新增源码级测试防止 scheduler 再次回归到硬编码 409
affects: [scheduler-handler, jobs]
tech-stack:
  added: []
  patterns: ["TriggerNow 锁定失败响应统一使用 net/http 常量", "用源码断言测试保护 Go 常量约束"]
key-files:
  created:
    - backend-go/internal/jobs/trigger_now_status_code_test.go
  modified:
    - backend-go/internal/jobs/firecrawl.go
    - backend-go/internal/jobs/content_completion.go
    - backend-go/internal/jobs/preference_update.go
key-decisions:
  - "用单个源码断言测试覆盖 3 个 scheduler，直接验证 http.StatusConflict 常量约束。"
  - "保持 TriggerNow 返回字段不变，只替换 status_code 的实现方式。"
patterns-established:
  - "Scheduler TriggerNow 的锁冲突响应应返回 reason/message/status_code，并使用 net/http 常量。"
  - "这类语义约束优先用轻量源码测试防回归。"
requirements-completed: [CONC-03]
duration: 4 min
completed: 2026-04-11
---

# Phase 01 Plan 01: TriggerNow 状态码常量化 Summary

**3 个 scheduler 的锁冲突响应统一改为 http.StatusConflict，并补上源码回归测试防止再次硬编码 409。**

## Performance

- **Duration:** 4 min
- **Started:** 2026-04-11T04:02:14Z
- **Completed:** 2026-04-11T04:06:11Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- firecrawl/content_completion/preference_update 的 TriggerNow 锁冲突响应全部改为 `http.StatusConflict`
- 新增 `TestTriggerNowStatusCode`，覆盖 3 个 scheduler 的常量约束与基础返回形状
- 保持 `accepted=false`、`reason=already_running` 等既有响应契约不变，兼容 handler 的 `status_code` 提取逻辑

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix status_code in firecrawl.go**
   - `4d1189d` (test)
   - `037b4ac` (fix)
2. **Task 2: Fix status_code in content_completion.go**
   - `c4ff340` (test)
   - `e8eda13` (fix)
3. **Task 3: Fix status_code in preference_update.go**
   - `edb4bec` (test)
   - `3763529` (fix)

## Files Created/Modified
- `backend-go/internal/jobs/trigger_now_status_code_test.go` - 用源码断言验证 3 个 scheduler 使用 `http.StatusConflict` 且未回退到硬编码 409
- `backend-go/internal/jobs/firecrawl.go` - Firecrawl TriggerNow 锁冲突响应改用 `http.StatusConflict`
- `backend-go/internal/jobs/content_completion.go` - 内容补全 TriggerNow 锁冲突响应改用 `http.StatusConflict`
- `backend-go/internal/jobs/preference_update.go` - 偏好更新 TriggerNow 锁冲突响应改用 `http.StatusConflict`

## Decisions Made
- 使用一个聚合测试而不是 3 个分散测试，减少重复并让计划验证命令稳定指向 `TestTriggerNowStatusCode`
- 不改返回字段结构，只做最小正确修改，确保 `respondTriggerResult` 继续按既有方式提取 `status_code`

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- CONC-03 已满足，后续计划可继续在统一 TriggerNow 返回格式基础上扩展并发控制修复
- 当前改动仅触达目标 scheduler 文件和回归测试，适合继续串行执行本 phase 后续 plan

## Self-Check: PASSED

- Verified summary and wiki files exist on disk
- Verified all 6 task commits are present in git history

---
*Phase: 01-并发控制修复*
*Completed: 2026-04-11*
