# 01-02 Firecrawl batch_id 返回

## Summary

- `backend-go/internal/jobs/firecrawl.go` 的 `TriggerNow()` 成功返回值新增 `batch_id`
- `runCrawlCycle(batchID string)` 复用同一个批次号做 `firecrawl_progress` 广播，前端可直接关联 trigger 响应与 WebSocket 推送
- `backend-go/internal/jobs/firecrawl_test.go` 新增源码级回归测试，分别锁定 TriggerNow 响应契约与 runCrawlCycle 的批次号透传

## Why It Matters

- `CONC-02` 要求 Firecrawl scheduler 的 `TriggerNow()` 让前端感知实际执行结果
- 前端只要拿到 `batch_id`，就能监听同 ID 的 WebSocket 进度消息，而不用猜测当前是哪一轮抓取

## Verification

- `go test ./internal/jobs -run TestFirecrawlTriggerNowBatchID -v`
- `go test ./internal/jobs -run TestFirecrawlRunCrawlCycleUsesInjectedBatchID -v`
- `firecrawl.go` 包含 `"batch_id": batchID` 与 `runCrawlCycle(batchID string)`
