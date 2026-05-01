# 前端架构

## 技术栈

- Nuxt 4.2.2
- Vue 3.5
- TypeScript
- Pinia
- Tailwind CSS v4
- Iconify
- Day.js
- marked
- motion-v
- Vitest

## 入口与路由

- 应用壳入口：`front/app/app.vue`
- 主阅读页：`front/app/pages/index.vue`
- Digest 总览：`front/app/pages/digest/index.vue`
- Digest 单视图：`front/app/pages/digest/[id].vue`
- Topic Graph：`front/app/pages/topics.vue`

`app.vue` 只做一件事：启动时调用 `apiStore.initialize()`，先拉分类、订阅源和文章，再渲染页面。

## 当前目录分层

```text
front/
├─ app/
│  ├─ api/                 # 唯一 HTTP 边界
│  ├─ assets/css/          # 全局主题与基础样式
│  ├─ components/          # 通用组件与对话框
│  ├─ composables/         # 少量跨 feature 的通用能力
│  ├─ features/            # 业务实现主体
│  ├─ pages/               # Nuxt 路由入口
│  ├─ plugins/             # Nuxt 插件
│  ├─ stores/              # Pinia store
│  ├─ types/               # 领域类型
│  ├─ utils/               # 常量和纯工具函数
│  └─ app.vue
├─ nuxt.config.ts
└─ package.json
```

## feature 划分

- `features/shell`：主壳、顶部栏、侧栏、文章列表栏
- `features/articles`：正文阅读、内容补全状态、Firecrawl 全文、AI 整理稿
- `features/summaries`：AI 总结列表、队列进度、WebSocket 实时更新
- `features/digest`：日报、周报、详情页、配置抽屉
- `features/feeds`：自动刷新和刷新轮询
- `features/preferences`：阅读行为埋点与偏好相关逻辑
- `features/topic-graph`：主题图谱、热点标签、话题详情、analysis、timeline、标签层级（TagHierarchy）、标签合并预览、叙事面板

这套结构已经替代旧的“业务都堆在 `components/`”的方式。新功能优先进入 `features/*`。

## 数据层约定

### API 层

`front/app/api/*` 是唯一 HTTP 边界。

- `client.ts` 统一封装 `fetch`
- 各领域模块只关心自己的接口
- query 参数统一走 `buildQueryParams()`
- 文件上传和下载也在这里收口

当前已落地的模块包括：

- `categories.ts`
- `feeds.ts`
- `articles.ts`
- `summaries.ts`
- `digest.ts`
- `opml.ts`
- `reading_behavior.ts`
- `firecrawl.ts`
- `scheduler.ts`
- `aiAdmin.ts`
- `topicGraph.ts`
- `abstractTags.ts`
- `embeddingConfig.ts`
- `embeddingQueue.ts`
- `mergeReembeddingQueue.ts`
- `tagMergePreview.ts`
- `watchedTags.ts`

### Store 层

`useApiStore()` 是前端数据主源。

- 持有 `categories`、`feeds`、`allFeeds`、`articles`
- 负责后端返回值到前端字段的映射，例如 `article_summary_enabled -> articleSummaryEnabled`、`summary_status -> summaryStatus`
- 负责增删改后重新拉取必要数据
- 负责文章已读、收藏、批量已读等更新

`useFeedsStore()` 和 `useArticlesStore()` 现在是派生视图层。

- `feedsStore` 基于 `apiStore` 暴露分类分组、未读数等计算结果
- `articlesStore` 基于 `apiStore` 暴露筛选、排序、当前文章等视图状态
- 不再维护本地副本
- 不再依赖手动 `syncToLocalStores()`

`usePreferencesStore()` 独立负责阅读偏好和统计。

## 数据映射规则

- 后端字段以 `snake_case` 为主
- 前端 store 和组件内部统一用 `camelCase`
- ID 在前端统一存成 `string`
- 数字 ID 与字符串 ID 的转换只应发生在 API 边界或 store 映射层
- 文章处理相关字段统一使用 `articleSummaryEnabled`、`summaryStatus`、`summaryGeneratedAt`

## 页面骨架

主阅读页由 `FeedLayoutShell.vue` 组织为三栏：

- 左栏：分类、订阅源、快捷入口
- 中栏：文章列表或 AI 总结列表
- 右栏：文章正文或 AI 总结详情

Digest 页面走独立路由和独立视觉壳，不复用主阅读页的三栏壳。

### Topics页面架构

#### 组件结构

- `TopicGraphPage`: 主页面容器
  - `TopicGraphHeader`: 头部控制区（返回首页、刷新图谱）
  - `TopicGraphCanvas`: 3D 拓扑图
  - `TopicAnalysisTabs`: 分析分类 Tabs
  - `TopicAnalysisPanel`: 分析内容展示
  - `TopicGraphSidebar`: 右侧详情栏
  - `ArticleTagList`: digest/article 标签的通用可视化组件

#### 状态管理

- 选中状态统一在 `TopicGraphPage` 管理
- `selectedCategory`: 当前选中的分类（event/person/keyword）
- `selectedTagInCategory`: 当前选中的标签 slug
- `highlightedNodeIds`: 需要高亮的节点 ID 列表

#### 数据流

1. 用户点击分类入口（热点分类按钮）后更新 `selectedCategory`
2. 页面基于 `selectedCategory` 计算 `highlightedNodeIds`
3. `highlightedNodeIds` 传给 `TopicGraphCanvas` 并驱动图谱高亮
4. 用户切换分析 Tabs 或选中标签后，底部分析面板按类型加载对应分析数据

#### 标签展示约定

- `ArticleContentView` 现在会直接展示标准 article tags
- digest 列表、digest 详情、topic graph timeline 使用 digest 的 `aggregated_tags`
- `/topics` 中当前选中的 topic slug 会在 digest/article 标签上做高亮

## 设计系统

当前前端已经切到 editorial / magazine 风格，而不是常见 SaaS 蓝紫模板。

- 主色：Ink Blue
- 强调色：Print Red
- 背景：Paper Warmth
- 阴影：偏纸张和印刷感
- 全局主题变量定义在 `front/app/assets/css/main.css`
- 分类和订阅源可选色定义在 `front/app/utils/constants.ts`

明确约束：

- 不用紫色 / 靛蓝色方案
- 不用纯平背景
- 不用默认 shadcn / Material 风格
- 不做对称、平均分栏的模板布局

## 运行与环境

- 前端开发端口：`http://localhost:3000`
- 后端 API：`http://localhost:5000/api`
- AI 总结 WebSocket：基于运行时 `apiBase` 推导，默认连 `ws://localhost:5000/ws`

## 相关文档

- `docs/developer/frontend-architecture.md`
- `docs/operations/architecture/data-flow.md`
- `docs/developer/frontend-features.md`
- `docs/user-guide/topic-graph/guide.md`
- `docs/developer/operations.md`
---
# 前端组件分工

## 路由层

- `front/app/pages/index.vue` 只挂载 `FeedLayoutShell`
- `front/app/pages/digest/index.vue` 只挂载 `DigestListView`
- `front/app/pages/digest/[id].vue` 只负责把 `daily` / `weekly` 路由参数锁定给 digest 视图

路由层不承载业务状态。

## Shell 层

### `front/app/features/shell/components/FeedLayoutShell.vue`

主阅读页编排层。

- 串起顶部栏、侧栏、文章列表栏、正文区
- 管理当前选中的分类、feed、文章、AI 总结
- 管理添加 / 编辑 / 导入 / 设置类弹窗
- 负责手动刷新、全部已读、OPML 导出
- 在 feed 刷新后轮询订阅源状态并回填文章列表
- 负责从主壳跳转到 `/digest`

### `front/app/features/shell/components/AppHeaderShell.vue`

- 顶部操作按钮
- 刷新消息提示
- “全部已读”、新增 feed、分类、导入、设置入口

### `front/app/features/shell/components/AppSidebarShell.vue`

- 分类树
- feed 列表
- 全部文章、收藏、AI 总结、Digest 快捷入口
- 分类和 feed 的编辑入口
- 侧栏折叠 / 展开

### `front/app/features/shell/components/ArticleListPanelShell.vue`

- 中栏列表壳
- 文章点击、收藏、筛选等交互桥接到 view 组件

## Articles

### `front/app/features/articles/components/ArticleCardView.vue`

- 单篇文章卡片
- 已读、收藏、摘要信息和基础元数据展示

### `front/app/features/articles/components/ArticleContentView.vue`

文章阅读主组件，也是当前最复杂的前端组件之一。

- 展示正文、元信息、封面图
- 切换预览模式与 iframe 模式
- 支持上下篇切换
- 支持全屏阅读
- 支持收藏、打开原文
- 集成阅读行为埋点
- 展示 Firecrawl 抓取状态与 AI 整理状态
- 支持手动抓取全文、手动生成整理稿
- 在“原始内容”和“Firecrawl 全文”之间切换
- 如果已有 `aiContentSummary`，优先展示整理稿

### `front/app/features/articles/components/ContentCompletionView.vue`

- 更偏状态展示
- 用于内容补全过程反馈

### `front/app/features/articles/composables/useContentCompletion.ts`

- 查询单篇文章内容补全状态
- 触发单篇文章补全
- 触发整条 feed 的批量补全

### `front/app/features/articles/composables/useArticleProcessingStatus.ts`

- 将 Firecrawl / AI 状态转成适合 UI 展示的标签、图标和语气色

## Summaries

### `front/app/features/summaries/components/AISummariesListView.vue`

- AI 总结列表页
- 支持按分类、feed、日期过滤
- 支持分页和每页数量切换
- 支持删除总结
- 支持发起新的总结任务
- 通过 WebSocket 监听总结队列进度

### `front/app/features/summaries/components/AISummaryDetailView.vue`

- 展示单条 AI 总结详情
- 配合主壳右栏显示

### `front/app/features/summaries/composables/useSummaryWebSocket.ts`

- 连接 `/ws`
- 接收队列进度
- 把 WS 消息转成前端批次对象
- 处理断线重连

## Digest

### `front/app/features/digest/components/DigestListView.vue`

- Digest 总览壳
- 在日报和周报之间切换
- 通过日期选择器切换锚点日
- 拉取预览、执行状态和当前版面
- 支持立即执行
- 支持打开设置抽屉
- 左栏展示分类和状态，中栏展示 feed 级 AI 总结，右栏挂 `DigestDetail`

### `front/app/features/digest/components/DigestDetail.vue`

- 展示单条 digest summary 的正文
- 渲染 markdown
- 拉取关联文章
- 支持在弹窗里直接阅读关联文章
- 能把总结里的链接映射回已知文章

### `front/app/features/digest/components/DigestSettings.vue`

- 配置日报 / 周报
- 配置飞书推送
- 配置 Obsidian 导出
- 测试飞书和 Obsidian 连接

## Feeds 与 Preferences

### `front/app/features/feeds/composables/useAutoRefresh.ts`

- 启动全局自动刷新逻辑

### `front/app/features/feeds/composables/useRefreshPolling.ts`

- 提供刷新轮询能力
- 配合主壳刷新 feed 状态

### `front/app/features/preferences/composables/useReadingTracker.ts`

- 跟踪文章打开、关闭、滚动、收藏、取消收藏
- 30 秒批量上报一次
- 达到事件数量阈值也会主动上报

## 通用组件层

`front/app/components/*` 现在主要保留可复用组件，而不是业务入口。

主要包括：

- `components/dialog/*`：新增 / 编辑 / 导入 / 全局设置弹窗
- `components/feed/*`：feed 图标和刷新状态
- `components/category/*`：分类卡片
- `components/common/*`：通用 UI 小组件
- `components/layout/*`：布局相关组件
- `components/ai/AISummary.vue`：临时 AI 分析卡片
- `components/article/*`：文章相关通用组件

## 约定

- 业务壳在 `features/*/components/*`
- 路由只做挂载
- API 不写进组件
- 跨层数据流优先走 store 和 composable
- 旧式 `components/*` 业务壳不再继续扩展
