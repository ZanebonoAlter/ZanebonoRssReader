---
phase: 02-标签流程统一
plan: 01
subsystem: api
tags: [go, gin, websocket, tag-queue, sqlite]
requires: []
provides:
  - 手动重打标签 API 改为异步入队并返回 job_id
  - tag_completed WebSocket 契约与队列完成广播
  - TagQueue 启动失败后台重试且不阻塞应用启动
affects: [articles-api, topicextraction, websocket, runtime]
tech-stack:
  added: []
  patterns: [异步任务入队响应, WebSocket 完成广播, 后台非阻塞启动重试]
key-files:
  created:
    - backend-go/internal/platform/ws/hub_test.go
  modified:
    - backend-go/internal/platform/ws/hub.go
    - backend-go/internal/domain/articles/handler.go
    - backend-go/internal/domain/articles/handler_test.go
    - backend-go/internal/domain/topicextraction/tag_queue.go
    - docs/api/articles.md
    - docs/architecture/backend-go.md
    - wiki/phases/02-01-tag-flow-unification.md
    - wiki/index.md
    - wiki/log.md
key-decisions:
  - 手动 `/api/articles/:article_id/tags` 只负责 enqueue 并返回 pending job 元数据，不再同步返回 tags
  - TagQueue 在 MarkCompleted 之后读取 article tags 并广播 `tag_completed`，让前端以队列完成事件为准刷新
  - TagQueue.Start 首次失败后立即返回 nil，后台每 30 秒重试一次，最多 10 次
patterns-established:
  - WebSocket 完成类消息统一采用 `{type, article_id, job_id, tags}` JSON 结构
  - 手动补救型 API 与后台 worker 解耦，使用 job queue 保持主路径一致
requirements-completed: [TAG-03, TAG-04]
duration: 7 min
completed: 2026-04-11
---

# Phase 02 Plan 01: 标签流程统一 Summary

**手动重打标签 API 现在异步写入 `tag_jobs`，并通过 `tag_completed` WebSocket 事件把队列完成结果推送给前端。**

## Performance

- **Duration:** 7 min
- **Started:** 2026-04-11T06:20:00Z
- **Completed:** 2026-04-11T06:27:00Z
- **Tasks:** 4
- **Files modified:** 10

## Accomplishments
- 新增 `TagCompletedMessage` / `TagCompletedItem` 负载，并用 TDD 锁定 JSON 契约
- 手动重打标签接口统一改为 `TagJobQueue.Enqueue`，返回 `job_id/article_id/status`
- TagQueue 完成任务后广播 `tag_completed`，且启动失败时改为后台重试，不再阻塞应用启动
- 同步更新 API / 架构文档与本地 wiki，记录新的异步契约

## Task Commits

Each task was committed atomically:

1. **Task 1: 新增 TagCompletedMessage WebSocket 消息类型（RED）** - `7404c23` (test)
2. **Task 1: 新增 TagCompletedMessage WebSocket 消息类型（GREEN）** - `3a82e36` (feat)
3. **Task 2: 改造 RetagArticleHandler 为异步 enqueue（RED）** - `c4762cb` (test)
4. **Task 2: 改造 RetagArticleHandler 为异步 enqueue（GREEN）** - `372170b` (feat)
5. **Task 3: TagQueue processJob 添加 WebSocket 通知** - `efb70e4` (feat)
6. **Task 4: TagQueue.Start 添加非阻塞启动重试机制** - `d4d7369` (fix)

## Files Created/Modified
- `backend-go/internal/platform/ws/hub.go` - 定义 `tag_completed` WebSocket 消息结构
- `backend-go/internal/platform/ws/hub_test.go` - 校验 `tag_completed` JSON 负载字段
- `backend-go/internal/domain/articles/handler.go` - 手动重打标签接口改为异步入队并返回 job 元数据
- `backend-go/internal/domain/articles/handler_test.go` - 校验接口返回 job_id 且写入 `tag_jobs`
- `backend-go/internal/domain/topicextraction/tag_queue.go` - 广播 `tag_completed`，并实现后台重试启动
- `docs/api/articles.md` - 更新手动打标签 API 与 WebSocket 契约
- `docs/architecture/backend-go.md` - 更新标签主流程与 TagQueue 重试说明
- `wiki/phases/02-01-tag-flow-unification.md` - 沉淀本 phase 的知识库条目
- `wiki/index.md` - 登记新的 phase 页面
- `wiki/log.md` - 追加本次知识库维护日志

## Decisions Made
- 手动补救入口不再绕过主队列，避免与 Firecrawl / ContentCompletion 形成两套标签路径。
- 前端完成态以 `tag_completed` 事件为准，而不是同步 HTTP 响应里的 tags 快照。
- 启动失败改为后台重试；这样 runtime 可以先起来，再等待数据库或表可用。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] 给后台重试补上停止信号与空数据库保护**
- **Found during:** Task 4 (TagQueue.Start 添加非阻塞启动重试机制)
- **Issue:** 如果首次启动失败后立即关闭服务，后台重试 goroutine 可能继续运行；同时 `database.DB` 为空时会在 `DB()` 调用处触发 nil dereference。
- **Fix:** 让 `backgroundRetry` 监听 `stopChan`，并在 `tryStart()` 开头显式校验 `database.DB` 是否已初始化。
- **Files modified:** `backend-go/internal/domain/topicextraction/tag_queue.go`
- **Verification:** `go test ./internal/domain/topicextraction -v`、`go build ./...`
- **Committed in:** `d4d7369`

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** 补丁只加强关闭与初始化安全性，没有改变计划要求的 API / WebSocket / 重试契约。

## Issues Encountered
- `gitnexus_detect_changes(scope: all)` 因仓库已有大量无关脏变更而返回 `critical`；本次通过逐文件 stage/commit，将提交范围限制在计划相关文件。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- TAG-03 / TAG-04 已落地，后续前端只需消费新的异步响应与 `tag_completed` 事件。
- 若需要更完整的手动验证，可在本地运行后调用 `/api/articles/:id/tags` 并观察 `tag_jobs` 与 WebSocket 消息。

## Self-Check: PASSED

- 已确认以下文件存在：`02-01-SUMMARY.md`、`wiki/phases/02-01-tag-flow-unification.md`、`docs/api/articles.md`、`backend-go/internal/platform/ws/hub_test.go`
- 已确认以下提交可在 `git log --oneline --all` 中找到：`7404c23`、`3a82e36`、`c4762cb`、`372170b`、`efb70e4`、`d4d7369`、`ade8972`

---
*Phase: 02-标签流程统一*
*Completed: 2026-04-11*
