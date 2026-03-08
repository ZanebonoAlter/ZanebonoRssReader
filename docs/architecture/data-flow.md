# 数据流

## 应用主链路

```text
RSS Feed
  -> backend-go feed refresh
  -> articles saved to SQLite
  -> optional content processing
  -> optional AI summary / digest generation
  -> frontend fetches and renders
```

## 前端数据流

```text
page/component
  -> app/api/*
  -> backend API
  -> store or feature state
  -> UI render
```

规则：

- 后端 JSON 用 snake_case
- 前端内部状态用 camelCase
- 转换集中在 API 映射层，不要散到组件里

## 后端数据流

```text
router
  -> handler
  -> service/domain logic
  -> database
  -> JSON response
```

## 定时任务链路

- 自动刷新 feed
- Firecrawl/内容补全处理文章正文
- AI 总结聚合文章内容
- Digest 生成日报周报
- 阅读偏好任务聚合行为数据
