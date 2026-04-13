---
phase: 260413-p2t
plan: 01
subsystem: api
tags: [merge-reembedding, embedding-queue, gin, postgres, nuxt, vue]
requires:
  - phase: 01-infrastructure-tag-convergence
    provides: tag merge semantics, async embedding queue groundwork, settings dialog entrypoint
provides:
  - merge-triggered re-embedding queue storage, worker, retry API, and tests
  - post-merge enqueue hook for surviving target tags only
  - dedicated backend-queues settings tab showing both queue panels
affects: [tag convergence, operator visibility, embedding maintenance, settings UI]
tech-stack:
  added: []
  patterns: [isolated queue service per backlog, transaction-then-enqueue merge hook, API-panel parity for operator tooling]
key-files:
  created:
    - backend-go/internal/domain/models/merge_reembedding_queue.go
    - backend-go/internal/domain/topicanalysis/merge_reembedding_queue.go
    - backend-go/internal/domain/topicanalysis/merge_reembedding_queue_handler.go
    - backend-go/internal/domain/topicanalysis/merge_reembedding_queue_test.go
    - backend-go/internal/domain/topicanalysis/merge_tags_reembedding_test.go
    - front/app/api/mergeReembeddingQueue.ts
    - front/app/features/ai/components/MergeReembeddingQueuePanel.vue
  modified:
    - backend-go/internal/domain/topicanalysis/embedding.go
    - backend-go/internal/platform/database/postgres_migrations.go
    - backend-go/internal/app/router.go
    - backend-go/internal/app/runtime.go
    - front/app/api/index.ts
    - front/app/components/dialog/GlobalSettingsDialog.vue
key-decisions:
  - "Keep merge re-embedding queue independent from the existing embedding queue so operators can inspect each backlog separately."
  - "Enqueue target-tag re-embedding only after MergeTags commits, so the worker never sees half-migrated state."
  - "Move queue visibility into a dedicated settings tab instead of burying it under 通用设置."
patterns-established:
  - "Queue handlers expose status/tasks/retry with the same success/data/error envelope used elsewhere."
  - "Queue UIs poll live status and pair summary cards with paginated task tables."
requirements-completed: [CONV-02, INFRA-03]
duration: 30m
completed: 2026-04-13
---

# Phase 260413-p2t Plan 01: Backend Queues Tab Summary

**独立后端队列设置页同时展示 embedding 队列与标签合并后的重算队列，并在 MergeTags 成功后真实落库异步重算目标标签 embedding。**

## Performance

- **Duration:** 30 min
- **Started:** 2026-04-13T18:05:00+08:00
- **Completed:** 2026-04-13T18:34:56+08:00
- **Tasks:** 3
- **Files modified:** 15

## Accomplishments
- 新增 `merge_reembedding_queues` 持久化模型、worker、status/tasks/retry API，以及针对状态统计、去重、失败重试的后端测试。
- `MergeTags` 在事务成功提交后为 surviving target tag 入队重算任务，且入队失败会显式返回错误。
- 设置弹窗新增“后端队列”Tab，并把原 embedding 队列与新的标签合并重算队列一起暴露给运营查看与重试。

## Task Commits

Each task was committed atomically:

1. **Task 1: Add real merge re-embedding queue backend with worker and API** - `84d0f64` (feat)
2. **Task 2: Enqueue re-embedding from MergeTags after successful merge** - `077ecdb` (feat)
3. **Task 3: Move queue UI into a dedicated backend-queues tab and add merge queue panel** - `67997d8` (feat)

_Note: Quick task docs were intentionally left uncommitted for the orchestrator._

## Files Created/Modified
- `backend-go/internal/domain/models/merge_reembedding_queue.go` - 定义合并后 embedding 重算任务模型与 source/target tag 关系。
- `backend-go/internal/domain/topicanalysis/merge_reembedding_queue.go` - 实现独立队列服务、去重、worker、失败重试与状态聚合。
- `backend-go/internal/domain/topicanalysis/merge_reembedding_queue_handler.go` - 暴露 `/api/embedding/merge-reembedding/*` 查询与重试接口。
- `backend-go/internal/domain/topicanalysis/merge_reembedding_queue_test.go` - 覆盖队列 status、dedupe、retry 行为。
- `backend-go/internal/domain/topicanalysis/merge_tags_reembedding_test.go` - 覆盖 MergeTags 提交后入队与入队失败错误回传。
- `backend-go/internal/domain/topicanalysis/embedding.go` - 在 MergeTags 事务提交后触发 target tag 队列入队。
- `backend-go/internal/platform/database/postgres_migrations.go` - 增加 merge re-embedding 队列表迁移。
- `backend-go/internal/app/router.go` - 注册 merge re-embedding 队列路由。
- `backend-go/internal/app/runtime.go` - 启停 merge re-embedding worker。
- `front/app/api/embeddingQueue.ts` - 前端旧 embedding 队列 API 封装。
- `front/app/api/mergeReembeddingQueue.ts` - 前端 merge re-embedding 队列 API 封装。
- `front/app/features/ai/components/EmbeddingQueuePanel.vue` - 旧 embedding 队列面板。
- `front/app/features/ai/components/MergeReembeddingQueuePanel.vue` - 新的标签合并重算队列面板。
- `front/app/components/dialog/GlobalSettingsDialog.vue` - 新增“后端队列”Tab，并移走 general 页内的队列内容。

## Decisions Made
- 复用现有 embedding queue 的接口形状，但不抽象成通用框架，避免为单次需求引入额外复杂度。
- merge queue 的去重维度按 `target_tag_id + pending/processing` 计算，防止重复合并把同一 surviving tag 推入并行重算。
- 前端保留两套队列面板，而不是把两个 backlog 混成一张表，方便运营区分“初始 embedding 缺口”和“合并后的重算积压”。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 补齐已被路由与设置页引用但未入库的旧队列文件**
- **Found during:** Task 1 / Task 3
- **Issue:** 当前工作树里 `router.go`、`runtime.go`、`GlobalSettingsDialog.vue` 已引用旧 embedding queue handler/panel/API，但这些文件仍是未跟踪状态；如果不一并纳入，本次提交后的仓库无法构建。
- **Fix:** 把旧 embedding queue handler、前端 API 与面板和本次 quick task 一起纳入对应任务提交，确保每次提交后仓库可编译。
- **Files modified:** `backend-go/internal/domain/topicanalysis/embedding_queue_handler.go`, `backend-go/internal/domain/topicanalysis/tag_management_handler.go`, `front/app/api/embeddingQueue.ts`, `front/app/features/ai/components/EmbeddingQueuePanel.vue`
- **Verification:** `go build ./...`, `pnpm exec nuxi typecheck`, `pnpm build`
- **Committed in:** `84d0f64`, `67997d8`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** 仅补齐本 quick task 直接依赖的缺失文件，没有额外扩展范围。

## Issues Encountered
- `gitnexus_detect_changes(scope="all")` 因仓库存在大量无关脏改动而给出 critical 级别总览；后续改用 staged scope 逐个任务核对，避免被无关变更污染判断。

## User Setup Required
None - no external service configuration required.

## Known Stubs
None.

## Next Phase Readiness
- 标签合并后的 embedding 重算已具备可观察、可重试的后台链路。
- 设置页的后端队列入口已就位，后续若增加更多后台 backlog，可继续复用同一 Tab 扩展。

## Self-Check: PASSED

- FOUND: `.planning/quick/260413-p2t-tab-embedding-embedding/260413-p2t-SUMMARY.md`
- FOUND: `wiki/phases/260413-p2t-backend-queues-and-merge-reembedding.md`
- FOUND commit: `84d0f64`
- FOUND commit: `077ecdb`
- FOUND commit: `67997d8`
