# Phase 2: 关注标签与首页推送 - Context

**Gathered:** 2026-04-15
**Status:** Ready for planning

<domain>
## Phase Boundary

用户可以关注特定标签（含抽象标签），首页看到关注标签关联的文章推送。包含：关注标签 CRUD API、前端关注交互（主题图谱页）、首页侧边栏关注标签筛选入口、关注标签文章流（全部+单标签）、按匹配标签数的相关度排序（抽象标签权重更高）、无关注标签时的引导降级。

不包含：日报周报重构（Phase 3）、标签推荐（Phase 5）、标签提取/合并逻辑变更。

</domain>

<decisions>
## Implementation Decisions

### 关注交互位置与方式
- **D-01:** 关注/取消关注操作集中在主题图谱页（`topics.vue`），不在首页
- **D-02:** 每个标签卡片上有心形图标，单击切换关注状态（心形填满+红色=已关注，空心=未关注）
- **D-03:** 关注状态切换采用即时反馈策略 — 前端立即更新 UI，同时后台同步 API
- **D-04:** 关注标签无数量上限，用户自由决定关注多少标签

### 首页推送展示
- **D-05:** 首页侧边栏新增"关注标签"分组（与分类/Feed 并列），作为新的筛选入口
- **D-06:** 侧边栏有"全部关注"入口 + 每个关注标签单独的筛选器
- **D-07:** 点击"全部关注"显示所有关注标签关联文章的混合流，点击某标签只看该标签文章
- **D-08:** 关注标签文章流显示全部关联了关注标签的文章，不区分 watched_at 前后
- **D-09:** 侧边栏关注标签列表每次进入时从后端加载，确保数据一致性

### 相关度排序
- **D-10:** 相关度排序仅按匹配的关注标签数量排序，不涉及 embedding 距离计算
- **D-11:** "全部关注"混合流默认按相关度排序（匹配标签数优先），用户可切换为时间倒序
- **D-12:** 单个标签筛选时排序按时间倒序（同标签下相关度排序无意义）

### 抽象标签支持
- **D-13:** 抽象标签（Phase 7 引入）可以直接被关注，与具体标签一视同仁
- **D-14:** 关注抽象标签后，其所有子标签关联的文章都出现在文章流中
- **D-15:** 相关度排序时，匹配抽象标签的权重高于匹配具体标签（抽象标签代表更广泛主题）

### 无关注标签时的过渡
- **D-16:** 未关注任何标签时，侧边栏"关注标签"分组显示引导横幅，引导用户前往主题图谱页关注标签
- **D-17:** 引导横幅每次用户点击侧边栏"关注标签"入口且无关注标签时都显示
- **D-18:** 无关注标签时首页保持默认时间线不变，不显示空白

### the agent's Discretion
- 侧边栏关注标签分组的具体视觉设计（折叠/展开、图标选择）
- 主题图谱页标签卡片上心形图标的具体位置和大小
- 后端 API 的请求/响应结构细节
- 分页加载策略（复用现有 useArticlePagination 模式）
- 抽象标签相关度的具体权重倍数

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 后端模型与标签系统
- `backend-go/internal/domain/models/topic_graph.go` — TopicTag、TopicTagEmbedding、ArticleTopicTag 模型定义（需新增 is_watched/watched_at 字段）
- `backend-go/internal/domain/models/topic_tag_relation.go` — TopicTagRelation 抽象标签层级关系（关注抽象标签时需查询子标签）
- `backend-go/internal/domain/topicanalysis/tag_management_handler.go` — 现有标签管理 handler（关注 API 可参考此处模式）
- `backend-go/internal/app/router.go` — HTTP 路由定义（需新增关注标签路由组）

### 前端核心架构
- `front/app/pages/index.vue` — 首页入口，渲染 FeedLayoutShell
- `front/app/features/shell/components/FeedLayoutShell.vue` — 首页三栏布局主组件（需修改侧边栏和文章列表）
- `front/app/features/shell/components/AppSidebarView.vue` — 侧边栏组件（需新增关注标签分组）
- `front/app/api/topicGraph.ts` — 标签相关前端 API（需扩展关注相关接口）

### 主题图谱页
- `front/app/pages/topics.vue` — 主题图谱页（关注操作入口）
- `front/app/features/topic-graph/components/TopicGraphPage.vue` — 图谱页主组件
- `front/app/features/topic-graph/components/TagHierarchy.vue` — 标签树组件（关注图标可在此添加）

### 先前阶段决策
- `.planning/phases/01-infrastructure-tag-convergence/01-CONTEXT.md` — Phase 1 决策，特别是 pgvector 基础设施
- `.planning/phases/07-middle-band-abstract-tags/07-CONTEXT.md` — Phase 7 抽象标签决策

No external specs — requirements fully captured in REQUIREMENTS.md (WATCH-01~03, FEED-01~03).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `ArticleTopicTag` 模型 + `article_topic_tags` 表：文章-标签关联表，用于查询关注标签的文章
- `TopicTag` 模型：已有 status/category/icon/aliases/description 字段，可直接扩展 is_watched/watched_at
- `TopicTagRelation` 模型：抽象标签层级关系，关注抽象标签时需 JOIN 查询子标签文章
- `useArticlePagination` composable：分页加载模式可复用
- `FeedLayoutShell.vue` 三栏架构：侧边栏分组筛选 → 文章列表 → 内容面板，关注标签可作为新的侧边栏分组
- `topicGraph.ts` API：已有标签搜索/合并接口，可扩展关注相关接口
- `tag_management_handler.go`：已有 SearchTags/MergeTags handler，关注 API 可参考路由注册模式

### Established Patterns
- **侧边栏分组模式**: 分类/Feed/收藏 作为侧边栏分组，点击切换文章列表筛选
- **关注 API 模式**: WATCH-02 要求 CRUD API，可参考现有 handler 模式（gin.H 响应）
- **即时 UI 反馈**: 前端先更新状态，API 失败时回滚
- **ID 转换**: 后端 uint → 前端 string，snake_case → camelCase 在 API/store 边界
- **GORM 模型扩展**: 新增字段 + AutoMigrate

### Integration Points
- `router.go` — 新增 `/api/topic-tags/watch` 路由组（关注/取消关注/列表）
- `AppSidebarView.vue` — 新增关注标签分组入口（在"主题图谱"按钮下方、分类列表上方）
- `FeedLayoutShell.vue` — 新增 selectedCategory='watched-tags' + selectedWatchedTag 筛选逻辑
- `ArticleListPanelShell.vue` — 文章列表需支持关注标签筛选参数
- `topicGraph.ts` — 新增关注/取消关注/列表 API 调用
- `TopicGraphPage.vue` / `TagHierarchy.vue` — 标签卡片上添加心形关注图标

</code_context>

<specifics>
## Specific Ideas

- 心形图标作为关注标识（填满+红色=已关注，空心=未关注）
- 抽象标签可以直接关注，关注后其子标签关联的文章全部出现在文章流
- 相关度排序时抽象标签权重更高（因为代表更广泛的主题）
- 引导横幅在每次无关注标签时都显示，不做"不再提示"的关闭
- 侧边栏关注标签列表每次进入时从后端加载，不做 store 缓存

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 02-watched-tags-homepage-feed*
*Context gathered: 2026-04-15*
