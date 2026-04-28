---
phase: 02-watched-tags-homepage-feed
plan: 01
subsystem: api
tags: [go, gin, gorm, postgresql, watched-tags, relevance-sort, abstract-tags]

# Dependency graph
requires:
  - phase: 07-middle-band-abstract-tags
    provides: TopicTagRelation model and abstract tag hierarchy
  - phase: 01-infrastructure-tag-convergence
    provides: TopicTag model, ArticleTopicTag model, topic_tag_relations table
provides:
  - Watched tag CRUD API (watch/unwatch/list)
  - GetArticles watched_tag_ids filtering with abstract tag expansion
  - Relevance-based article sorting (abstract tags weighted 2x)
  - Database migration for is_watched/watched_at columns
affects: [02-02-PLAN, front-api, homepage-feed]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Watched tag expansion: abstract tags auto-include child tag articles"
    - "Relevance scoring: SUM(CASE WHEN) subquery for weighted tag matching"

key-files:
  created:
    - backend-go/internal/domain/topicanalysis/watched_tags_service.go
    - backend-go/internal/domain/topicanalysis/watched_tags_handler.go
  modified:
    - backend-go/internal/domain/models/topic_graph.go
    - backend-go/internal/domain/models/article.go
    - backend-go/internal/platform/database/postgres_migrations.go
    - backend-go/internal/domain/articles/handler.go
    - backend-go/internal/app/router.go

key-decisions:
  - "Relevance score uses subquery instead of GROUP BY to avoid GORM scan issues"
  - "Watched tag expansion queries topic_tag_relations at request time for fresh data"
  - "Separate count query for watched tags to avoid JOIN count inflation"

patterns-established:
  - "parseAndExpandWatchedTagIDs: comma-separated tag IDs + abstract tag child expansion"
  - "WatchedTagInfo struct: extends TopicTag with is_abstract/child_slugs for API response"

requirements-completed: [WATCH-01, WATCH-02, WATCH-03, FEED-01, FEED-02, FEED-03]

# Metrics
duration: 6min
completed: 2026-04-15
---

# Phase 2 Plan 1: Watched Tags Backend Summary

**关注标签 CRUD API（watch/unwatch/list）+ 首页文章查询支持 watched_tag_ids 筛选和相关度排序（抽象标签权重 2x）**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-14T23:45:06Z
- **Completed:** 2026-04-14T23:51:09Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- TopicTag 模型扩展 is_watched/watched_at 字段，PostgreSQL 迁移自动添加列
- 3 个关注标签 API 端点：GET /api/topic-tags/watched、POST /:tag_id/watch、POST /:tag_id/unwatch
- GetArticles 扩展 watched_tag_ids 参数筛选 + sort_by=relevance 按匹配标签数排序
- 抽象标签关注时自动展开子标签文章，相关度权重 2x

## Task Commits

Each task was committed atomically:

1. **Task 1: TopicTag model + watched tags CRUD** - `e583a1b` (feat)
2. **Task 2: GetArticles watched tag filtering + relevance sorting** - `f1b7cfe` (feat)

## Files Created/Modified
- `backend-go/internal/domain/topicanalysis/watched_tags_service.go` — WatchTag/UnwatchTag/ListWatchedTags/GetWatchedTagIDsExpanded 业务逻辑
- `backend-go/internal/domain/topicanalysis/watched_tags_handler.go` — HTTP handlers + RegisterWatchedTagsRoutes
- `backend-go/internal/domain/models/topic_graph.go` — TopicTag 新增 IsWatched/WatchedAt 字段
- `backend-go/internal/domain/models/article.go` — Article 新增 RelevanceScore 字段 + ToDict 扩展
- `backend-go/internal/platform/database/postgres_migrations.go` — 迁移 20260415_0001 添加 is_watched/watched_at 列
- `backend-go/internal/domain/articles/handler.go` — GetArticles 支持 watched_tag_ids + sort_by + parseAndExpandWatchedTagIDs
- `backend-go/internal/app/router.go` — 注册 RegisterWatchedTagsRoutes

## Decisions Made
- 使用子查询计算 relevance_score 而非 GROUP BY，避免 GORM 扫描复杂度
- 关注标签展开查询在请求时实时查 topic_tag_relations，确保数据新鲜
- 为 watched tags 场景使用独立 COUNT(DISTINCT) 查询避免 JOIN 计数膨胀

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 后端 API 完整就绪，前端可直接对接
- Task 2 中 GetArticles 的 watched_tag_ids 为空时行为完全不变，不影响现有首页
- Plan 02 可基于这些 API 构建前端关注交互和首页推送 UI

---
*Phase: 02-watched-tags-homepage-feed*
*Completed: 2026-04-15*

## Self-Check: PASSED

- All 8 key files verified FOUND on disk
- Commits e583a1b and f1b7cfe verified in git log
- `go build ./...` passes cleanly
