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
- 主布局实现：`front/app/features/shell/components/FeedLayoutShell.vue`

## 当前结构

- `api/` 已是前端唯一 HTTP 边界
- `features/` 已承接主要业务实现
- `pages/` 只做路由挂载和入口分发
- `components/` 只保留通用组件，不再承载主业务壳层
- `stores/api.ts` 负责后端数据拉取与边界转换
- `stores/feeds.ts`、`stores/articles.ts` 现在直接消费 `apiStore`，不再手工同步副本

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
- `features/` - 业务域代码与主要实现
- `shared/` - 跨 feature 复用
- `stores/` - 全局状态与衍生视图状态
- `pages/` - 路由入口，只做组装

## 当前落地结果

```text
front/app/
├── api/                     # 已承接真实 API 实现
├── features/
│   ├── articles/            # ArticleContent / ArticleCard / ContentCompletion
│   ├── digest/              # Digest 页面实现与设置抽屉
│   ├── feeds/               # auto refresh / refresh polling
│   ├── preferences/         # reading tracker
│   ├── shell/               # AppHeader / AppSidebar / ArticleListPanel / FeedLayout
│   └── summaries/           # AI summary list/detail + websocket
├── stores/
│   ├── api.ts               # 数据源与 API 边界映射
│   ├── feeds.ts             # 从 apiStore 暴露 feed/category 视图
│   └── articles.ts          # 从 apiStore 暴露 article 视图
├── components/              # 通用组件：dialog/common/feed/category
├── composables/             # 仅保留通用 composable
└── pages/
```

## 主要业务域

- `shell` - 应用壳、顶部栏、侧边栏、主布局
- `categories` - 分类管理
- `feeds` - 订阅源管理与刷新
- `articles` - 文章列表与正文阅读
- `summaries` - AI 总结
- `digest` - 日报周报
- `preferences` - 阅读偏好

## 状态流

- `apiStore` 是后端数据的单一来源
- `feedsStore` 和 `articlesStore` 不再复制数组，只暴露基于 `apiStore` 的视图
- `syncToLocalStores()` 已移除
- snake_case 到 camelCase 的转换只放在 API 边界

## 迁移原则

- 新代码优先写进 `features/*`、`api/*`、`shared/*`
- 业务实现统一放在 `features/*`
- 旧兼容壳已删除，不再从 `components/*`、`composables/*` 走旧入口
- 新功能不要再接回 `services/` 或手工同步链

## 配套文档

- 功能说明：`docs/guides/frontend-features.md`
- 组件分工：`docs/architecture/frontend-components.md`
