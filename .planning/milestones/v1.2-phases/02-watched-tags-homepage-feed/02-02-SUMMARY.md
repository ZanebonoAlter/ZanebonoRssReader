---
phase: 02-watched-tags-homepage-feed
plan: 02
subsystem: frontend
tags: [vue, typescript, sidebar, watched-tags, heart-icon, article-filter]

# Dependency graph
requires:
  - plan: 02-01
    provides: Watched tags CRUD API endpoints + GetArticles watched_tag_ids filtering
provides:
  - useWatchedTagsApi composable (listWatchedTags, watchTag, unwatchTag)
  - Heart icon toggle in TagHierarchy
  - Sidebar watched tags group
  - FeedLayoutShell watched tag article filtering
affects: [homepage-feed, sidebar, tag-hierarchy]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Optimistic UI toggle: immediate local state flip + API sync + rollback on failure"
    - "Watched tag expansion: all watched IDs joined as comma-separated query param"

key-files:
  created:
    - front/app/api/watchedTags.ts
  modified:
    - front/app/types/article.ts
    - front/app/features/topic-graph/components/TagHierarchy.vue
    - front/app/features/topic-graph/components/TagHierarchyRow.vue
    - front/app/features/shell/components/AppSidebarView.vue
    - front/app/features/shell/components/FeedLayoutShell.vue

key-decisions:
  - "Watched tag IDs passed as Set<number> from TagHierarchy to rows for O(1) lookup"
  - "Sidebar watched tags section placed between topic-graph button and categories divider"
  - "Empty watched tags state shows guidance banner with link to /topics"
  - "buildArticleFilters checks watched-tags category before feed/category filters"

patterns-established:
  - "useWatchedTagsApi: snake_case to camelCase mapping in API layer"
  - "WatchedTag type: id, slug, label, category, watchedAt, isAbstract, childSlugs"

requirements-completed: [WATCH-01, FEED-01, FEED-02, FEED-03]

# Metrics
duration: 5min
completed: 2026-04-15
---

# Phase 2 Plan 2: Watched Tags Frontend Summary

**前端关注标签 API 层 + 心形图标 + 侧边栏关注分组 + 文章筛选**

## Performance

- **Duration:** 5 min
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- 创建 useWatchedTagsApi composable，包含 listWatchedTags/watchTag/unwatchTag
- ArticleFilters 扩展 watched_tag_ids + sort_by 字段
- TagHierarchy 标签行添加心形关注图标，即时反馈 + API 失败回滚
- 侧边栏添加"关注标签"分组：全部关注 + 单个标签筛选
- FeedLayoutShell 扩展 buildArticleFilters 支持关注标签筛选
- 无关注标签时显示引导横幅，首页保持默认时间线

## Task Commits

1. **Task 1: API + 心形图标** - `c33532f` (feat)
2. **Task 2: 侧边栏 + 文章筛选** - `b6b05dc` (feat)

## Files Created/Modified
- `front/app/api/watchedTags.ts` — useWatchedTagsApi composable + WatchedTag type
- `front/app/types/article.ts` — ArticleFilters 新增 watched_tag_ids + sort_by
- `front/app/features/topic-graph/components/TagHierarchy.vue` — loadWatchedTags + toggleWatch + watchedTagIds state
- `front/app/features/topic-graph/components/TagHierarchyRow.vue` — watchedTagIds prop + heart icon + toggle-watch emit
- `front/app/features/shell/components/AppSidebarView.vue` — watched tags props/emits + sidebar section
- `front/app/features/shell/components/FeedLayoutShell.vue` — watched tags state + filter logic + handlers

## Decisions Made
- Watched tag IDs 用 Set<number> 传递给子组件，O(1) 查找
- 侧边栏关注标签分组位于主题图谱和分类之间
- 即时反馈：心形图标点击后立即翻转本地状态，API 失败时回滚

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## Next Phase Readiness
- 前端完整可用：关注标签 → 侧边栏显示 → 文章筛选 → 相关度排序
- Phase 3 (日报周报) 可基于 watched tags 数据重构日报生成逻辑

---
*Phase: 02-watched-tags-homepage-feed*
*Completed: 2026-04-15*

## Self-Check: PASSED

- All 6 key files verified FOUND on disk
- Commits c33532f and b6b05dc verified in git log
- `pnpm exec nuxi typecheck` passes cleanly
