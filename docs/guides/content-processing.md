# 内容处理链路

## 功能说明

内容处理链路负责把 RSS 原始内容补成更完整的正文，并在需要时继续生成 AI 总结。

## 当前链路

```text
Feed refresh
  -> article created
  -> optional Firecrawl/content completion
  -> article content updated
  -> optional AI summary generation
```

## 关键模块

- Feed 刷新：`backend-go/internal/services/feed_service.go`
- 自动刷新调度器：`backend-go/internal/schedulers/auto_refresh.go`
- 内容补全 handler：`backend-go/internal/handlers/content_completion.go`
- Firecrawl handler：`backend-go/internal/handlers/firecrawl.go`
- 自动总结调度器：`backend-go/internal/schedulers/auto_summary.go`
- 内容补全调度器：`backend-go/internal/schedulers/content_completion.go`
- Firecrawl 调度器：`backend-go/internal/schedulers/firecrawl.go`

## 调度器怎么看

前端现在可以在设置弹窗的 `定时任务` tab 里直接看：

- `auto_refresh` 这轮扫了多少 feed，真正触发了多少刷新
- `auto_summary` 这次手动 trigger 是真开跑了，还是因为配置/状态被拒绝
- `ai_summary` 和 `firecrawl` 的处理队列状态

这块对应接口来自 `GET /api/schedulers/status` 和 `POST /api/schedulers/:name/trigger`。

## 前端相关位置

- 编辑 feed：`front/app/components/dialog/EditFeedDialog.vue`
- 文章正文：`front/app/components/article/ArticleContent.vue`
- 内容补全 composable：`front/app/composables/useContentCompletion.ts`
