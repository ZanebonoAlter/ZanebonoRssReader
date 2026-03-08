# 阅读偏好指南

## 功能说明

系统会记录阅读行为，用来生成偏好分数，并为 AI 总结和排序提供参考。

## 记录内容

- 文章打开和关闭
- 滚动深度
- 阅读时长
- 收藏行为

## 数据链路

```text
ArticleContent
  -> reading behavior API
  -> reading_behaviors table
  -> preference aggregation
  -> user_preferences table
  -> settings panel display
```

## 前端位置

- 阅读追踪：`front/app/composables/useReadingTracker.ts`
- 偏好状态：`front/app/stores/preferences.ts`

## 后端位置

- 行为接口：`backend-go/internal/handlers/reading_behavior.go`
- 偏好服务：`backend-go/internal/services/preference_service.go`

## 相关接口

- `POST /api/reading-behavior/track`
- `POST /api/reading-behavior/track-batch`
- `GET /api/reading-behavior/stats`
- `GET /api/user-preferences`
- `POST /api/user-preferences/update`
