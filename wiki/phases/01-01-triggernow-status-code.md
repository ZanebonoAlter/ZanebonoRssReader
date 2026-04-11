# 01-01 TriggerNow 状态码常量化

## Summary

- `firecrawl.go`、`content_completion.go`、`preference_update.go` 的 `TriggerNow()` 在锁冲突时统一返回 `http.StatusConflict`
- 保持返回体字段不变：`accepted`、`started`、`reason`、`message`、`status_code`
- 新增 `backend-go/internal/jobs/trigger_now_status_code_test.go`，以源码断言方式防止再次出现硬编码 `409`

## Why It Matters

- `handler.go` 会提取 `result["status_code"]` 作为 HTTP 响应码
- 用 `net/http` 常量比硬编码数字更清晰，也能避免不同 scheduler 风格漂移

## Verification

- `go test ./internal/jobs -run TestTriggerNowStatusCode -v`
- 检查 3 个目标文件均包含 `http.StatusConflict`
- 检查 3 个目标文件不再包含 `status_code": 409`
