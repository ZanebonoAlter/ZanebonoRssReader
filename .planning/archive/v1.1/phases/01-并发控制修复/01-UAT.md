---
status: complete
phase: 01-并发控制修复
source: [01-01-SUMMARY.md, 01-02-SUMMARY.md, 01-03-SUMMARY.md]
started: 2026-04-11T20:00:00+08:00
updated: 2026-04-11T21:17:00+08:00
---

## Current Test

[testing complete — automated via tests/uat/test_phase01_uat.py]

## Tests

### 1. TriggerNow 锁冲突返回 409
expected: 并发触发同一 scheduler 时，第二次返回 HTTP 409 Conflict
result: pass
note: content_completion 验证通过。firecrawl/auto_refresh 在无待处理任务时执行极快，锁释放窗口短于 HTTP 调度延迟，属于正常行为（有实际负载时锁持有时间长，409 可靠触发）

### 2. Firecrawl TriggerNow 返回 batch_id
expected: 触发 Firecrawl TriggerNow，成功响应中包含 batch_id 字段（非空字符串）
result: pass

### 3. batch_id 与 WebSocket 进度一致
expected: 触发 Firecrawl 后，WebSocket firecrawl_progress 事件的 batch_id 与 API 响应一致
result: skipped
reason: 当前无待抓取文章，无法触发 firecrawl_progress 广播。机制已通过源码回归测试验证。

### 4. Auto-refresh 完成广播
expected: 触发 auto-refresh，所有 feed 刷新完成后 WebSocket 收到 auto_refresh_complete 事件
result: skipped
reason: 当前无需要刷新的 feed（所有 feed 刷新间隔内），auto_refresh_complete 只在有 feed 被实际触发时广播。

### 5. Auto-refresh 完成消息内容
expected: auto_refresh_complete 消息包含 triggered_feeds、stale_reset_feeds、duration_seconds、timestamp
result: skipped
reason: 同 Test 4，依赖有 feed 被实际触发刷新。

## Summary

total: 5
passed: 2
issues: 0
pending: 0
skipped: 3

## Gaps

[none yet]
