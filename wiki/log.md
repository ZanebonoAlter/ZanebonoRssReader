# 本地知识库日志

## [2026-04-11] ingest | Phase 01 Plan 01 TriggerNow 状态码常量化
- 新增 `phases/01-01-triggernow-status-code.md`
- 记录 CONC-03 的实现结果与测试策略

## [2026-04-11] ingest | Phase 01 Plan 02 Firecrawl batch_id 返回
- 新增 `phases/01-02-firecrawl-batch-id.md`
- 记录 CONC-02 的 batch_id 返回契约与回归测试

## [2026-04-11] ingest | Phase 01 Plan 03 Auto-refresh 完成通知
- 新增 `phases/01-03-auto-refresh-completion.md`
- 记录 `auto_refresh_complete` WebSocket 事件契约与广播顺序

## [2026-04-11] ingest | Phase 02 Plan 01 标签流程统一
- 新增 `phases/02-01-tag-flow-unification.md`
- 记录手动重打标签异步入队、`tag_completed` 事件契约与 TagQueue 后台重试策略

## [2026-04-11] ingest | Phase 04 Plan 01 前端 API 一致性与未读数同步
- 新增 `phases/04-01-frontend-api-consistency.md`
- 记录 scheduler trigger 统一 `apiClient`、已读未读数同步以及未分类 feed 清零策略

## [2026-04-11] ingest | Phase 04 Plan 02 统一后端 scheduler status API 返回格式
- 新增 `phases/04-02-scheduler-status-format.md`
- 记录统一的五字段 status 契约、Unix `next_run` 以及 task details 补充接口

## [2026-04-13] ingest | Quick task 260413-p2t 后端队列 Tab 与标签合并重算队列
- 新增 `phases/260413-p2t-backend-queues-and-merge-reembedding.md`
- 记录 MergeTags 提交后 target-tag 重算入队、独立 merge re-embedding worker/API，以及设置页“后端队列”Tab 的双队列可视化

## [2026-04-14] ingest | Quick task Summary 文章级去重标记
- 新增 `phases/260414-summary-article-markers.md`
- 记录 `articles.feed_summary_id` / `feed_summary_generated_at` 两个字段，以及 summary 命中旧批次时的文章回填逻辑
