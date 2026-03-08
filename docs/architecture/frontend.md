# 前端架构

## 技术栈

- Nuxt 4
- Vue 3
- TypeScript
- Pinia
- Tailwind CSS v4

## 当前入口

- 应用壳：`front/app/app.vue`
- 首页路由：`front/app/pages/index.vue`
- Digest 路由：`front/app/pages/digest/index.vue`
- 文章主布局：`front/app/components/FeedLayout.vue`

## 当前问题

- `components/` 过大，领域边界不清
- `composables/api/`、`services/`、`stores/api.ts` 有重叠
- `syncToLocalStores()` 让数据流变绕

## 目标结构

```text
front/app/
├── app.vue
├── pages/
├── api/
├── features/
├── stores/
├── shared/
└── plugins/
```

## 目录规则

- `api/` - 唯一 HTTP 边界
- `features/` - 业务域代码
- `shared/` - 跨 feature 复用
- `stores/` - 全局状态或跨页面状态
- `pages/` - 路由入口，只做组装

## 主要业务域

- `shell` - 应用壳、顶部栏、侧边栏、主布局
- `categories` - 分类管理
- `feeds` - 订阅源管理与刷新
- `articles` - 文章列表与正文阅读
- `summaries` - AI 总结
- `digest` - 日报周报
- `preferences` - 阅读偏好

## 迁移原则

- 先搬目录，再修 import
- 先保行为，再收缩 `apiStore`
- snake_case 到 camelCase 的转换只放在 API 边界
