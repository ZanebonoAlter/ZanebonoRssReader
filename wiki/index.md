# 本地知识库索引

## Phases
- [01-01 TriggerNow 状态码常量化](phases/01-01-triggernow-status-code.md) - 统一 3 个 scheduler 的锁冲突响应常量，并记录回归测试策略。
- [01-02 Firecrawl batch_id 返回](phases/01-02-firecrawl-batch-id.md) - Firecrawl TriggerNow 成功返回 batch_id，并与 WebSocket 进度广播保持一致。
- [01-03 Auto-refresh 完成通知](phases/01-03-auto-refresh-completion.md) - Auto-refresh 在 feed 刷新完成后广播 `auto_refresh_complete`，前端可在 auto-summary 前收到完成时机。
- [02-01 标签流程统一](phases/02-01-tag-flow-unification.md) - 手动重打标签统一入队 `tag_jobs`，完成后广播 `tag_completed`，且 TagQueue 启动失败改为后台重试。
- [04-01 前端 API 一致性与未读数同步](phases/04-01-frontend-api-consistency.md) - Scheduler trigger 统一走 `apiClient`，单篇/批量已读都会同步分类与未分类 feed 的未读数。
- [04-02 Scheduler status API 统一格式](phases/04-02-scheduler-status-format.md) - 后端 scheduler status 固定为五字段结构，`next_run` 统一为 Unix 时间戳，并把任务级补充信息下沉到专用 helper。
- [260413-p2t 后端队列 Tab 与标签合并重算队列](phases/260413-p2t-backend-queues-and-merge-reembedding.md) - 设置页新增独立后端队列 Tab，并为 MergeTags 增加真实的重算任务队列、worker 与重试接口。
- [260414 Summary 文章级去重标记](phases/260414-summary-article-markers.md) - `auto_summary` 和 `summary_queue` 改为按文章级 summary 标记去重，并追加 Postgres 迁移记录。
