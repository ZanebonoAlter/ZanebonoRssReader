# 04-02 Scheduler status API 统一格式

## Summary

- `backend-go/internal/jobs/handler.go` 新增 `SchedulerStatusResponse`，固定 `name/status/check_interval/next_run/is_executing` 五字段。
- `auto_refresh`、`auto_summary`、`content_completion`、`preference_update`、`firecrawl` 的 `GetStatus()` 都改为直接返回统一 struct。
- `GetTasksStatus` 额外引入 task details helper，保留 Firecrawl / Content Completion 的队列统计，而不再把这些字段混入统一 status API。

## Why It Matters

- `API-04` 要求所有 scheduler status endpoint 返回一致结构，前端不再需要为不同 scheduler 写分支解析逻辑。
- 统一 `next_run` 为 Unix 时间戳后，避免不同 scheduler 混用 `time.Time`、RFC3339 字符串和零值空串导致状态漂移。

## Verification

- `go build ./internal/jobs`
- `go test ./internal/jobs -v -run TestSchedulerStatusFormat`
- `go test ./internal/jobs -v -run "TestGetSchedulerStatus(ReturnsUnifiedResponseShape|AliasUsesSameUnifiedShape)"`
- `go test ./internal/jobs -v`
