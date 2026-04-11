# 02-01 标签流程统一

## Summary

- `backend-go/internal/domain/articles/handler.go` 的手动重打标签接口改为异步 enqueue `tag_jobs`，立即返回 `job_id/article_id/status`
- `backend-go/internal/platform/ws/hub.go` 新增 `TagCompletedMessage` / `TagCompletedItem`，约定 `tag_completed` WebSocket 负载
- `backend-go/internal/domain/topicextraction/tag_queue.go` 在任务完成后广播 `tag_completed`，并把 `Start()` 改为首次失败立即返回、后台定时重试

## Why It Matters

- `TAG-03` 要求所有文章标签入口统一走 `TagJobQueue`，避免手动 API 直接调用 `RetagArticle` 造成旁路
- `TAG-04` 要求 TagQueue 启动失败不能拖垮应用启动，同时前端需要在异步队列完成后得到明确完成信号

## Verification

- `go test ./internal/domain/articles -run TestRetagArticle -v`
- `go test ./internal/domain/topicextraction -v`
- `go build ./...`
- `tag_queue.go` 包含 `go q.backgroundRetry()`、`TagQueue retry attempt`、`ws.GetHub().BroadcastRaw(data)`
