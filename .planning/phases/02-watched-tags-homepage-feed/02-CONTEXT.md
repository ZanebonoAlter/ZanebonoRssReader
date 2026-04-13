# Phase 2: 关注标签与首页推送 - Context

**Gathered:** 2026-04-13
**Status:** Ready for planning

<domain>
## Phase Boundary

用户可以关注特定标签，首页看到关注标签关联的文章推送。包含：关注标签 CRUD API、前端关注交互、首页侧边栏关注标签筛选入口、关注标签文章流（全部+单标签）、按匹配标签数的相关度排序、无关注标签时的引导降级。

不包含：日报周报重构（Phase 3）、标签推荐（Phase 5）、标签提取/合并逻辑变更。

</domain>

<decisions>
## Implementation Decisions

### 关注交互位置与方式
- **D-01:** 关注/取消关注操作集中在主题图谱页（`topics.vue`），不在首页
- **D-02:** 每个标签卡片上有切换图标（心形或眼睛），单击切换关注状态
- **D-03:** 主题图谱页有独立面板展示当前已关注标签，按分类分组（事件、人物、关键词）
- **D-04:** 关注状态切换采用即时反馈策略 — 前端立即更新 UI，同时后台同步 API

### 首页推送展示
- **D-05:** 首页侧边栏新增"关注标签"分组，作为新的筛选入口（与分类/Feed 并列）
- **D-06:** 侧边栏有"全部关注"入口 + 每个关注标签单独的筛选器
- **D-07:** 点击"全部关注"显示所有关注标签关联文章的混合流，点击某标签只看该标签文章
- **D-08:** 侧边栏关注标签采用扁平列表 + 分类颜色点 + 标签名 + 文章数量
- **D-09:** 侧边栏支持按分类（事件/人物/关键词）筛选关注标签

### 相关度排序
- **D-10:** 相关度排序仅按匹配的关注标签数量排序，不涉及 embedding 距离计算
- **D-11:** "全部关注"混合流默认按相关度排序（匹配标签数优先），用户可切换为时间倒序
- **D-12:** 单个标签筛选时排序按时间倒序（同标签下相关度排序无意义）

### 无关注标签时的过渡
- **D-13:** 未关注任何标签时，侧边栏"关注标签"分组显示引导横幅，引导用户前往主题图谱页关注标签
- **D-14:** 引导横幅每次用户点击侧边栏"关注标签"入口且无关注标签时都显示
- **D-15:** 无关注标签时首页保持默认时间线不变，不显示空白

### 关注标签策略
- **D-16:** 关注标签无数量上限，用户自由决定关注多少标签
- **D-17:** 文章流显示全部关联了关注标签的文章，不区分 watched_at 前后

### the agent's Discretion
- 侧边栏关注标签分组的具体视觉设计（折叠/展开、图标选择）
- 切换关注状态的图标样式（心形/眼睛/书签等）
- 关注面板在主题图谱页的具体位置和布局
- 后端 API 的请求/响应结构细节
- 分页加载策略（复用现有 useArticlePagination 模式）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 后端模型与标签系统
- `backend-go/internal/domain/models/topic_graph.go` — TopicTag、TopicTagEmbedding、ArticleTopicTag 模型定义（需新增 is_watched/watched_at 字段）
- `backend-go/internal/domain/topicextraction/handler.go` — 现有标签相关 handler（关注 API 可参考此处模式）
- `backend-go/internal/app/router.go` — HTTP 路由定义（需新增关注标签路由组）

### 前端核心架构
- `front/app/pages/index.vue` — 首页入口，渲染 FeedLayoutShell
- `front/app/features/shell/components/FeedLayoutShell.vue` — 首页三栏布局主组件（需修改侧边栏和文章列表）
- `front/app/features/shell/components/AppSidebarShell.vue` — 侧边栏组件（需新增关注标签分组）
- `front/app/api/topicGraph.ts` — 标签相关前端 API（需扩展关注相关接口）
- `front/app/stores/api.ts` — 主数据 store

### 主题图谱页
- `front/app/pages/topics.vue` — 主题图谱页（关注操作入口）
- `front/app/features/topic-graph/components/TopicGraphPage.vue` — 图谱页主组件
- `front/app/features/topic-graph/components/TopicGraphSidebar.vue` — 图谱页侧边栏（关注面板可在此或新增）

### Phase 1 决策
- `.planning/phases/01-infrastructure-tag-convergence/01-CONTEXT.md` — Phase 1 决策，特别是 status/merged 过滤、pgvector 基础设施

No external specs — requirements fully captured in REQUIREMENTS.md (WATCH-01~03, FEED-01~03).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `ArticleTopicTag` 模型 + `article_topic_tags` 表：文章-标签关联表，用于查询关注标签的文章
- `TopicTag` 模型：已有 status/category/icon/aliases 字段，可直接扩展 is_watched/watched_at
- `useArticlePagination` composable：分页加载模式可复用
- `FeedLayoutShell.vue` 三栏架构：侧边栏分组筛选 → 文章列表 → 内容面板，关注标签可作为新的侧边栏分组
- `topicGraph.ts` API：已有标签相关接口模式，可扩展
- `TopicGraphSidebar.vue`：已有标签分类筛选组件模式

### Established Patterns
- **侧边栏分组模式**: 分类/Feed/收藏 作为侧边栏分组，点击切换文章列表筛选
- **关注 API 模式**: WATCH-02 要求 CRUD API，可参考现有 handler 模式（gin.H 响应）
- **即时 UI 反馈**: 前端先更新状态，API 失败时回滚
- **ID 转换**: 后端 uint → 前端 string，snake_case → camelCase 在 API/store 边界
- **GORM 模型扩展**: 新增字段 + AutoMigrate

### Integration Points
- `router.go` — 新增 `/api/tags/watch` 或 `/api/topic-tags/watch` 路由组
- `AppSidebarShell.vue` — 新增关注标签分组入口
- `ArticleListPanelShell.vue` — 文章列表需支持关注标签筛选参数
- `FeedLayoutShell.vue` — 整合关注标签筛选逻辑到现有的 selectedCategory/selectedFeed 模式
- `topicGraph.ts` — 新增关注/取消关注/列表 API 调用

</code_context>

<specifics>
## Specific Ideas

- 侧边栏关注标签要同时支持扁平列表和按分类筛选两种视图
- "全部关注"混合流默认按匹配标签数排序（非时间倒序），与单标签筛选的默认时间排序不同
- 引导横幅在每次无关注标签时都显示，不做"不再提示"的关闭

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 02-watched-tags-homepage-feed*
*Context gathered: 2026-04-13*
