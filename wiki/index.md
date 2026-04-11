# 本地知识库索引

## Phases
- [01-01 TriggerNow 状态码常量化](phases/01-01-triggernow-status-code.md) - 统一 3 个 scheduler 的锁冲突响应常量，并记录回归测试策略。
- [01-02 Firecrawl batch_id 返回](phases/01-02-firecrawl-batch-id.md) - Firecrawl TriggerNow 成功返回 batch_id，并与 WebSocket 进度广播保持一致。
- [01-03 Auto-refresh 完成通知](phases/01-03-auto-refresh-completion.md) - Auto-refresh 在 feed 刷新完成后广播 `auto_refresh_complete`，前端可在 auto-summary 前收到完成时机。
- [02-01 标签流程统一](phases/02-01-tag-flow-unification.md) - 手动重打标签统一入队 `tag_jobs`，完成后广播 `tag_completed`，且 TagQueue 启动失败改为后台重试。
