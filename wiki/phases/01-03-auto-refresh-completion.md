# 01-03 Auto-refresh 完成通知

## Summary

- `backend-go/internal/platform/ws/hub.go` 新增 `AutoRefreshCompleteMessage`，约定 `auto_refresh_complete` 事件字段
- `backend-go/internal/jobs/auto_refresh.go` 在所有 feed 刷新结束后先广播完成消息，再触发 auto-summary
- `backend-go/internal/jobs/auto_refresh_test.go` 新增 JSON 契约测试与源码接线回归测试，锁定广播字段和顺序

## Why It Matters

- `CONC-01` 要求前端知道 auto-refresh 何时真正完成，而不是只知道 trigger API 已经返回
- 把广播放在 auto-summary 前面，可以让前端明确区分“feeds 刷新完成”和“下游总结开始”两个时刻

## Verification

- `go build ./internal/platform/ws ./internal/jobs`
- `go test ./internal/jobs -run TestAutoRefreshComplete -v`
- `auto_refresh.go` 包含 `auto_refresh_complete`、`ws.GetHub().BroadcastRaw(data)` 与 `triggerAutoSummaryAfterRefreshes(&refreshWG, startTime, summary)`
