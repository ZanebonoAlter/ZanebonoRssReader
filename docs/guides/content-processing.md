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
- 内容补全 handler：`backend-go/internal/handlers/content_completion.go`
- Firecrawl handler：`backend-go/internal/handlers/firecrawl.go`
- 内容补全调度器：`backend-go/internal/schedulers/content_completion.go`
- Firecrawl 调度器：`backend-go/internal/schedulers/firecrawl.go`

## 前端相关位置

- 编辑 feed：`front/app/components/dialog/EditFeedDialog.vue`
- 文章正文：`front/app/components/article/ArticleContent.vue`
- 内容补全 composable：`front/app/composables/useContentCompletion.ts`
