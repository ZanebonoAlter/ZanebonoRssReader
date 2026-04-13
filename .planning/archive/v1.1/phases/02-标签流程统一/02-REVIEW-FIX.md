---
phase: 02-标签流程统一
fixed_at: 2026-04-11T14:56:00Z
review_path: .planning/phases/02-标签流程统一/02-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 02: Code Review Fix Report

**Fixed at:** 2026-04-11T14:56:00Z
**Source review:** .planning/phases/02-标签流程统一/02-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 2
- Fixed: 2
- Skipped: 0

## Fixed Issues

### WR-01: `RetagArticleHandler` 对活跃任务的回查条件不正确，可能把成功入队误报成 500

**Files modified:** `backend-go/internal/domain/articles/handler.go`, `backend-go/internal/domain/articles/handler_test.go`
**Commit:** 0ad4c94
**Applied fix:** 将 handler 中的 job 回查条件从 `status = 'pending'` 扩展为 `status IN ('pending', 'leased')`，匹配 `Enqueue()` 的复用逻辑。同时返回 `tagJob.Status`（实际状态）而非硬编码的 `"pending"`。补充了回归测试 `TestRetagArticleWithExistingLeasedJob`，验证当已有 leased 任务时 POST /tags 仍返回 200 并正确返回 leased 状态。

### WR-02: 异步改造后没有可用的失败态回传路径，客户端可能永远等不到终态

**Files modified:** `backend-go/internal/domain/topicextraction/tag_queue.go`, `backend-go/internal/platform/ws/hub.go`
**Commit:** 43a24c2
**Applied fix:** 在 `ws/hub.go` 中新增 `TagFailedMessage` 结构体（type: `tag_failed`，含 `article_id`、`job_id`、`error` 字段）。在 `tag_queue.go` 中新增 `broadcastTagFailed` 方法，并在 `processJob` 的三个失败路径（panic recovery、article fetch 失败、tag 执行失败）中调用，确保前端能通过 WebSocket 收到标签任务失败通知。

---

_Fixed: 2026-04-11T14:56:00Z_
_Fixer: the agent (gsd-code-fixer)_
_Iteration: 1_
