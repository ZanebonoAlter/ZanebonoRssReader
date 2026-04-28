---
phase: 02-标签流程统一
reviewed: 2026-04-11T06:36:00Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - backend-go/internal/domain/articles/handler.go
  - backend-go/internal/domain/articles/handler_test.go
  - backend-go/internal/domain/topicextraction/tag_queue.go
  - backend-go/internal/platform/ws/hub.go
  - backend-go/internal/platform/ws/hub_test.go
  - docs/api/articles.md
  - docs/architecture/backend-go.md
  - wiki/phases/02-01-tag-flow-unification.md
  - wiki/index.md
  - wiki/log.md
findings:
  critical: 0
  warning: 2
  info: 0
  total: 2
status: issues_found
---

# Code Review

## Summary

审查了 Phase 02 的 10 个文件，重点覆盖手动重打标签异步入队、`tag_completed` WebSocket 契约、TagQueue 后台重试语义，以及对应文档/知识库同步。

## Findings

### WR-01 `RetagArticleHandler` 对活跃任务的回查条件不正确，可能把成功入队误报成 500

- file: `backend-go/internal/domain/articles/handler.go`
- line: `216-242`
- issue: `TagJobQueue.Enqueue()` 会复用同一篇文章已有的 `pending` 或 `leased` 任务，而 handler 在入队后只按 `status = pending` 回查 job（`Where("article_id = ? AND status = ?", article.ID, string(models.JobStatusPending))`）。当任务已经被 worker 抢占为 `leased`，或者新建任务在回查前被快速 claim，接口会走到 `failed to retrieve job_id` 并返回 500，尽管本次请求实际上已经成功并入同一个任务。
- fix: 让 `Enqueue()` 直接返回最终任务记录或 job ID，避免二次查询；至少要把回查条件扩展到 `pending` + `leased`，并返回真实状态而不是硬编码 `pending`。补一个回归测试覆盖“已有 leased 任务时再次 POST /tags”场景。

### WR-02 异步改造后没有可用的失败态回传路径，客户端可能永远等不到终态

- file: `backend-go/internal/domain/topicextraction/tag_queue.go`
- line: `242-255`
- issue: 标签任务失败时只调用 `MarkFailed()`，没有像成功路径那样广播 WebSocket 事件；而文档在 `docs/api/articles.md:72` 要求前端“监听 `tag_completed` 或轮询 job 状态”，但当前路由层只提供了 summaries 队列的 `queue/jobs/:job_id` 查询，没有对应的 tag job 查询入口。结果是：手动重打标签一旦失败，HTTP 已经返回 success，前端既收不到失败事件，也没有正式查询通道拿到失败原因。
- fix: 二选一即可闭环：1) 增加 `tag_failed`/`tag_finished` WebSocket 事件并包含 `job_id`、`status`、`error`；2) 暴露 tag job 状态查询 API，并把文档改成真实可用的查询路径。无论采用哪种方式，都应补充失败态集成测试。

## Test Notes

- 已运行 `go test ./internal/domain/articles -run "TestGetArticleReturnsArticleTags|TestGetArticlesReturnsTagCount|TestRetagArticleReturnsUpdatedTags" -v`
- 已运行 `go test ./internal/domain/topicextraction -v`
- 已运行 `go test ./internal/platform/ws -v`

上述测试均通过；当前问题属于并发/失败路径覆盖不足，现有 happy-path 测试没有暴露出来。
