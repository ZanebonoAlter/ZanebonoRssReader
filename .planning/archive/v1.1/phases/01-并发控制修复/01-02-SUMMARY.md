---
phase: 01-并发控制修复
plan: 02
subsystem: api
tags: [go, scheduler, firecrawl, websocket, batch-id]
requires:
  - phase: 01-并发控制修复
    provides: TriggerNow 锁冲突响应已统一为标准返回格式
provides:
  - Firecrawl TriggerNow 成功响应立即返回 batch_id
  - TriggerNow 返回的 batch_id 与 runCrawlCycle WebSocket 进度广播保持一致
  - 新增源码级回归测试覆盖 TriggerNow 与 runCrawlCycle 的 batch_id 约束
affects: [scheduler-handler, firecrawl-progress, frontend-tracking]
tech-stack:
  added: []
  patterns: ["TriggerNow 在异步任务启动前生成可追踪 batch_id", "用源码级回归测试锁定批次号透传与广播一致性"]
key-files:
  created: []
  modified:
    - backend-go/internal/jobs/firecrawl.go
    - backend-go/internal/jobs/firecrawl_test.go
key-decisions:
  - "batch_id 在 TriggerNow 内生成，并同时提供给 HTTP 响应与 runCrawlCycle，避免前端拿到的批次号与 WebSocket 漂移。"
  - "沿用轻量源码断言测试，而不是构造完整 Firecrawl 运行时依赖，保证并发修复回归测试稳定。"
patterns-established:
  - "需要前端跟踪异步 scheduler 执行时，应在 TriggerNow 返回可关联的 batch_id，并复用到后续进度广播。"
  - "对调度器契约类修复，优先增加聚焦源码约束的回归测试。"
requirements-completed: [CONC-02]
duration: 4 min
completed: 2026-04-11
---

# Phase 01 Plan 02: Firecrawl batch_id 返回 Summary

**Firecrawl TriggerNow 现在会立即返回可追踪的 batch_id，并保证该 batch_id 与 WebSocket `firecrawl_progress` 广播完全一致。**

## Performance

- **Duration:** 4 min
- **Started:** 2026-04-11T04:10:55Z
- **Completed:** 2026-04-11T04:14:13Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `FirecrawlScheduler.TriggerNow()` 在成功触发时生成 `batch_id` 并返回给前端
- `runCrawlCycle(batchID string)` 复用同一个批次号做 processing/completed 广播
- 新增并拆分回归测试，分别保护 TriggerNow 响应契约与 runCrawlCycle 的 batch_id 透传约束

## Task Commits

Each task was committed atomically:

1. **Task 1: Generate batch_id in TriggerNow and pass to runCrawlCycle**
   - `26f2fb1` (test)
   - `ed2d49b` (feat)
2. **Task 2: Update runCrawlCycle to accept batch_id parameter**
   - `8a3a311` (test)

## Files Created/Modified
- `backend-go/internal/jobs/firecrawl.go` - 在 TriggerNow 中生成并返回 `batch_id`，并将 `runCrawlCycle` 改为显式接收 `batchID string`
- `backend-go/internal/jobs/firecrawl_test.go` - 为 TriggerNow 响应和 runCrawlCycle 批次号透传添加源码级回归测试

## Decisions Made
- 选择在 `TriggerNow()` 里提前生成 `batch_id`，让前端拿到的值与后台广播使用同一个源头
- 保持 `checkAndCrawl()` 独立生成批次号，确保 cron 触发路径也能沿用同一广播格式而不引入共享状态字段

## Deviations from Plan

- Task 1 的实现已经同时完成了 Task 2 所需的生产代码调整，因此 Task 2 的原子提交聚焦为额外回归测试拆分，避免为了“补一个提交”引入冗余代码改动。

---

**Total deviations:** 0 auto-fixed
**Impact on plan:** 生产代码仍按计划完成；额外提交只用于把重叠任务的回归覆盖拆开，未扩大范围。

## Issues Encountered
- 初版回归测试错误假设 `firecrawl.go` 里只能出现一次时间戳格式生成；实际 `checkAndCrawl()` 也需要生成 cron 批次号。已将断言收敛为“`runCrawlCycle` 不再自行生成 batch_id”，验证通过。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Firecrawl trigger API 已能向前端暴露稳定的 `batch_id`，后续前端或状态查询能力可直接复用该契约
- 当前修改仅触达 `firecrawl.go` 与对应测试，适合继续串行执行本 phase 后续 plan

## Self-Check: PASSED

- Verified `.planning/phases/01-并发控制修复/01-02-SUMMARY.md` exists on disk
- Verified commits `26f2fb1`, `ed2d49b`, and `8a3a311` exist in git history

---
*Phase: 01-并发控制修复*
*Completed: 2026-04-11*
