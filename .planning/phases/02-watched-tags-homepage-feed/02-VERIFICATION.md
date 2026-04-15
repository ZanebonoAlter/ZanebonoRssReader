---
phase: 02-watched-tags-homepage-feed
verified: 2026-04-15T12:30:00Z
status: human_needed
score: 9/9 must-haves verified
overrides_applied: 0
human_verification:
  - test: "在主题图谱页点击标签心形图标，验证关注/取消关注的视觉反馈"
    expected: "已关注显示红色填满心形，未关注显示空心，切换即时无闪烁"
    why_human: "图标颜色和即时反馈速度需要视觉验证"
  - test: "关注标签后进入首页，验证侧边栏'关注标签'分组显示"
    expected: "分组显示'全部关注'入口和每个关注标签，点击'全部关注'显示相关度排序文章"
    why_human: "侧边栏布局和交互细节需要人工确认"
  - test: "取消所有关注标签后，验证侧边栏引导横幅和首页行为"
    expected: "侧边栏显示'关注标签可获取个性化文章推送'引导横幅，首页保持默认时间线"
    why_human: "空状态 UI 和降级行为需要完整流程验证"
  - test: "使用运行中的后端+真实数据，验证关注抽象标签后首页文章筛选"
    expected: "关注抽象标签后，其子标签关联的文章都出现在首页关注推送中"
    why_human: "需要运行中的 PostgreSQL + 真实标签数据才能验证端到端数据流"
---

# Phase 2: 关注标签与首页推送 Verification Report

**Phase Goal:** 用户可以关注特定标签，首页看到关注标签关联的文章推送
**Verified:** 2026-04-15T12:30:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | 用户可以关注/取消关注标签，关注状态持久化并记录 watched_at | ✓ VERIFIED | TopicTag 模型有 `IsWatched bool` + `WatchedAt *time.Time` 字段 (topic_graph.go:55-56); `WatchTag()` 设置 `is_watched=true, watched_at=now()` 并 `db.Save()` 持久化 (watched_tags_service.go:23-38) |
| 2 | 关注标签 CRUD API 可用（列表、设置关注、取消关注） | ✓ VERIFIED | 3 个 handler + `RegisterWatchedTagsRoutes` 注册路由 (watched_tags_handler.go:12-77); router.go:168 注册到 `/api/topic-tags` group |
| 3 | 关注抽象标签时，其子标签关联的文章都包含在查询结果中 | ✓ VERIFIED | `parseAndExpandWatchedTagIDs()` 查询 `topic_tag_relations WHERE parent_id IN ids` 获取子标签 ID 并合并到筛选列表 (handler.go:227-257); `GetWatchedTagIDsExpanded()` 同样展开 (watched_tags_service.go:134-163) |
| 4 | 首页展示关注标签关联的文章流，按时间倒序排列 | ✓ VERIFIED | GetArticles 支持 `watched_tag_ids` 参数，JOIN `article_topic_tags` 筛选，默认 `ORDER BY articles.pub_date DESC` (handler.go:70,102,156) |
| 5 | 用户可按单个关注标签筛选文章 | ✓ VERIFIED | FeedLayoutShell `handleWatchedTagClick()` 设置 `selectedWatchedTagId`，`buildArticleFilters()` 传入单个 tag ID (FeedLayoutShell.vue:99,299-309) |
| 6 | 文章列表支持按相关度排序（匹配标签数，抽象标签权重更高） | ✓ VERIFIED | `sort_by=relevance` 时使用 `SUM(CASE WHEN ... THEN 2.0 ELSE 1.0 END)` 计算权重，抽象标签子标签权重 2x (handler.go:106-110); Article 模型有 `RelevanceScore` 字段 (article.go:34) |
| 7 | 无关注标签时首页回退到完整时间线，不显示空白 | ✓ VERIFIED | `buildArticleFilters()` 中 `selectedCategory === 'watched-tags'` 且 `watchedTags` 为空时不添加 filter (FeedLayoutShell.vue:97-104); `watched_tag_ids` 为空时后端查询行为完全不变 (handler.go:82-93) |
| 8 | 首页侧边栏显示'关注标签'分组，含'全部关注'入口和每个关注标签筛选器 | ✓ VERIFIED | AppSidebarView 有 `watched-tags-section`，包含 `watchedTagsClick` 全部关注入口和 `watchedTagClick` 单标签按钮 (AppSidebarView.vue:205-230); FeedLayoutShell 传递 `watchedTags` 和 `selectedWatchedTagId` props (FeedLayoutShell.vue:500-501) |
| 9 | 用户可在主题图谱页通过心形图标关注/取消关注标签 | ✓ VERIFIED | TagHierarchy 加载 watched tags 状态 (TagHierarchy.vue:131-140), 传递 `watchedTagIds` Set 给 TagHierarchyRow (TagHierarchy.vue:449); TagHierarchyRow 有心形按钮 `mdi:heart`/`mdi:heart-outline` (TagHierarchyRow.vue:106-114), 点击触发 `toggleWatch` (TagHierarchy.vue:142-161) |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `backend-go/internal/domain/models/topic_graph.go` | TopicTag IsWatched + WatchedAt 字段 | ✓ VERIFIED | Lines 55-56, gorm 标签正确 |
| `backend-go/internal/domain/topicanalysis/watched_tags_handler.go` | 关注标签 HTTP handlers | ✓ VERIFIED | 3 handlers + RegisterWatchedTagsRoutes, 77 行实质性代码 |
| `backend-go/internal/domain/topicanalysis/watched_tags_service.go` | 关注标签业务逻辑 | ✓ VERIFIED | WatchTag/UnwatchTag/ListWatchedTags/GetWatchedTagIDsExpanded, 163 行 |
| `backend-go/internal/domain/articles/handler.go` | GetArticles watched_tag_ids 扩展 | ✓ VERIFIED | Lines 70-93 参数解析, 101-117 JOIN + relevance 排序, 227-257 展开函数 |
| `backend-go/internal/app/router.go` | 路由注册 RegisterWatchedTagsRoutes | ✓ VERIFIED | Line 168 |
| `backend-go/internal/domain/models/article.go` | RelevanceScore 字段 | ✓ VERIFIED | `gorm:"->;column:relevance_score"` read-only computed column |
| `backend-go/internal/platform/database/postgres_migrations.go` | 迁移添加 is_watched/watched_at | ✓ VERIFIED | Migration 20260415_0001 |
| `front/app/api/watchedTags.ts` | useWatchedTagsApi composable | ✓ VERIFIED | 36 行, listWatchedTags/watchTag/unwatchTag + snake→camelCase 映射 |
| `front/app/types/article.ts` | ArticleFilters 扩展 | ✓ VERIFIED | `watched_tag_ids?: string` + `sort_by?: 'relevance' | 'date'` |
| `front/app/features/topic-graph/components/TagHierarchy.vue` | loadWatchedTags + toggleWatch | ✓ VERIFIED | 697 行, Set<number> 状态, 乐观更新 + API 失败回滚 |
| `front/app/features/topic-graph/components/TagHierarchyRow.vue` | 心形关注图标 | ✓ VERIFIED | mdi:heart/heart-outline, watchedTagIds prop, toggle-watch emit |
| `front/app/features/shell/components/AppSidebarView.vue` | 侧边栏关注标签分组 | ✓ VERIFIED | watchedTags/selectedWatchedTagId props, watchedTagsClick/watchedTagClick emits, 空状态引导横幅 |
| `front/app/features/shell/components/FeedLayoutShell.vue` | 关注标签筛选逻辑 | ✓ VERIFIED | watchedTags state, loadWatchedTags(), buildArticleFilters() 分支, 两个 handler 函数 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| watched_tags_handler.go | watched_tags_service.go | WatchTag/UnwatchTag/ListWatchedTags 函数调用 | ✓ WIRED | 直接调用带 database.DB 参数 |
| router.go | watched_tags_handler.go | RegisterWatchedTagsRoutes | ✓ WIRED | Line 168: `topicanalysisdomain.RegisterWatchedTagsRoutes(api)` |
| articles/handler.go | article_topic_tags table | JOIN query for watched tag filtering | ✓ WIRED | `JOIN article_topic_tags att ON ... AND att.topic_tag_id IN ?` |
| FeedLayoutShell.vue | /api/articles?watched_tag_ids | useArticlePagination + buildArticleFilters | ✓ WIRED | watched_tag_ids 和 sort_by 参数在 buildArticleFilters 中构建 |
| AppSidebarView.vue | watchedTags.ts | loadWatchedTags on mount (via FeedLayoutShell) | ✓ WIRED | FeedLayoutShell.onMounted 调用 loadWatchedTags() |
| TagHierarchy.vue | /api/topic-tags/:id/watch | useWatchedTagsApi().watchTag/unwatchTag | ✓ WIRED | toggleWatch() 调用 API，乐观更新 + 失败回滚 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| FeedLayoutShell.vue | `watchedTags` | `loadWatchedTags()` → `useWatchedTagsApi().listWatchedTags()` → GET /api/topic-tags/watched | DB: `WHERE is_watched = true AND status = 'active'` | ✓ FLOWING |
| FeedLayoutShell.vue | `articles` (via pagination) | `fetchFirstPage(buildArticleFilters())` → GET /api/articles?watched_tag_ids=... | DB: JOIN article_topic_tags, 真实文章数据 | ✓ FLOWING |
| TagHierarchy.vue | `watchedTagIds` | `loadWatchedTags()` → API → Set<number> | DB: watched tags 查询 | ✓ FLOWING |
| AppSidebarView.vue | `watchedTags` | props from FeedLayoutShell | FeedLayoutShell 的 API 加载结果 | ✓ FLOWING |
| GetArticles handler | `expandedTagIDs` | `parseAndExpandWatchedTagIDs()` → DB query topic_tag_relations | 真实标签关系数据 | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go build | `cd backend-go && go build ./...` | 成功无错误 | ✓ PASS |
| Frontend typecheck | `cd front && pnpm exec nuxi typecheck` | 成功无类型错误 | ✓ PASS |
| TopicTag 模型 IsWatched 字段 | grep IsWatched backend-go/internal/domain/models/topic_graph.go | Line 55: `IsWatched bool` | ✓ PASS |
| RegisterWatchedTagsRoutes 导出 | grep RegisterWatchedTagsRoutes backend-go/internal/domain/topicanalysis/watched_tags_handler.go | Line 70: func 定义 | ✓ PASS |
| useWatchedTagsApi 导出 | grep "export function" front/app/api/watchedTags.ts | Line 13: export function useWatchedTagsApi | ✓ PASS |
| ArticleFilters watched_tag_ids | grep watched_tag_ids front/app/types/article.ts | Line 52: 字段存在 | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-----------|-------------|--------|----------|
| WATCH-01 | 02-01, 02-02 | 用户可在标签列表页面勾选/取消关注标签 | ✓ SATISFIED | TagHierarchy 心形图标 + WatchTag/UnwatchTag API |
| WATCH-02 | 02-01 | 后端提供关注标签 CRUD API | ✓ SATISFIED | 3 个 handler + RegisterWatchedTagsRoutes 注册 3 条路由 |
| WATCH-03 | 02-01 | 关注标签变更时记录 watched_at | ✓ SATISFIED | WatchTag() 设置 `watched_at = time.Now()` 并持久化 |
| FEED-01 | 02-01, 02-02 | 首页展示关注标签关联的文章流，按时间倒序排列 | ✓ SATISFIED | GetArticles watched_tag_ids 筛选 + 默认 pub_date DESC |
| FEED-02 | 02-01, 02-02 | 支持按单个关注标签筛选文章 | ✓ SATISFIED | FeedLayoutShell handleWatchedTagClick + buildArticleFilters 分支 |
| FEED-03 | 02-01, 02-02 | 文章列表支持按相关度排序 | ✓ SATISFIED | sort_by=relevance + SUM(CASE WHEN) 权重计算，抽象标签 2x |

**Orphaned requirements:** None — all 6 Phase 2 requirements covered by plans.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | - | - | - | No TODO/FIXME/placeholder/empty implementations found in phase files |

### Human Verification Required

### 1. 心形图标视觉反馈

**Test:** 在主题图谱页 (/topics) 点击标签行上的心形图标
**Expected:** 已关注状态显示红色填满心形 (mdi:heart, text-red-500)，未关注显示空心 (mdi:heart-outline)；切换即时响应，无页面刷新或闪烁
**Why human:** 图标颜色渲染和交互流畅度需要视觉确认

### 2. 侧边栏关注标签分组布局

**Test:** 关注 2-3 个标签后回到首页，查看侧边栏
**Expected:** 在"主题图谱"按钮下方显示"关注标签"分组标题，"全部关注"入口（心形图标），以及每个关注标签的名称和图标；抽象标签用不同图标 (mdi:tag-multiple) 区分
**Why human:** 布局间距、图标对齐、选中高亮状态需要人工确认

### 3. 空状态引导横幅

**Test:** 取消关注所有标签后回到首页，查看侧边栏
**Expected:** "关注标签"分组内显示引导文字"关注标签可获取个性化文章推送"和"前往关注"按钮，点击按钮导航到 /topics；首页文章列表保持默认时间线不变
**Why human:** 空状态 UI 和降级行为需要完整流程验证

### 4. 端到端数据流（需运行服务）

**Test:** 启动后端 (go run cmd/server/main.go) + 前端 (pnpm dev)，关注一个抽象标签和其子标签
**Expected:** 首页点击"全部关注"后显示关联文章（含子标签文章），按相关度排序；点击单个标签只显示该标签文章，按时间排序；抽象标签关注后其子标签关联文章也出现在结果中
**Why human:** 需要 PostgreSQL + 真实标签数据才能验证完整的数据流

### Gaps Summary

**自动化验证全部通过。** 9 个可观测真相全部在代码层面得到验证：
- 后端：TopicTag 模型扩展、3 个 CRUD API 端点、GetArticles 关注标签筛选 + 相关度排序（抽象标签 2x 权重）、数据库迁移
- 前端：useWatchedTagsApi composable、TagHierarchy 心形图标（乐观更新 + 失败回滚）、侧边栏关注标签分组、FeedLayoutShell 筛选逻辑、空状态引导横幅
- 构建：`go build ./...` 和 `pnpm exec nuxi typecheck` 均通过
- 需求：6 个 requirement (WATCH-01~03, FEED-01~03) 全部覆盖

4 项人工验证待确认，主要是 UI 视觉和端到端数据流测试。

---

_Verified: 2026-04-15T12:30:00Z_
_Verifier: the agent (gsd-verifier)_
