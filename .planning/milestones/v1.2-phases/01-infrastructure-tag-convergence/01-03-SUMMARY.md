---
phase: 01-infrastructure-tag-convergence
plan: 03
subsystem: database, api
tags: [tag-merge, transaction, gorm, status-field, reference-migration]

requires:
  - phase: 01-infrastructure-tag-convergence
    provides: "pgvector SQL similarity, EmbeddingService.TagMatch three-level matching, findOrCreateTag integration"
provides:
  - TopicTag Status/MergedIntoID fields with migration
  - MergeTags atomic transaction function (5-step reference migration)
  - activeTagFilter scope for excluding merged tags from queries
  - TagMatch and FindSimilarTags filter out merged tags
affects: [tag-convergence, topic-analysis, topic-graph, tag-merge-ui]

tech-stack:
  added: []
  patterns: [transaction-safe-merge, dedup-before-update, active-tag-scope]

key-files:
  created: []
  modified:
    - backend-go/internal/domain/models/topic_graph.go
    - backend-go/internal/domain/topicanalysis/embedding.go
    - backend-go/internal/platform/database/postgres_migrations.go

key-decisions:
  - "Merged tags marked status='merged' with merged_into_id pointer, never physically deleted (per D-04, D-05, D-07)"
  - "Dedup-before-update pattern: check for existing (article_id, target_tag_id) before updating source links to avoid unique constraint violations"
  - "activeTagFilter includes OR status='' to handle rows created before migration ran"

patterns-established:
  - "Transaction-safe merge: 5-step atomic operation (article refs → summary refs → mark merged → delete embedding → recalculate feed_count)"
  - "Dedup-before-update: check existing link before migration, delete source link if target already covers it"
  - "Active tag scope: GORM scope filtering status=active OR status=empty for backward compat"

requirements-completed: [CONV-02, CONV-04]

duration: 3min
completed: 2026-04-13
---

# Phase 1 Plan 3: Tag Merge with Transaction-Safe Reference Migration Summary

**Atomic tag merge with Status/MergedIntoID fields, transaction-safe article_topic_tags and ai_summary_topics reference migration, and active-tag filtering across all match queries**

## Performance

- **Duration:** 3 min
- **Started:** 2026-04-13T00:01:42Z
- **Completed:** 2026-04-13T00:04:40Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- TopicTag model gains Status (active/merged) and MergedIntoID fields with migration
- MergeTags function performs 5-step atomic transaction: migrate article refs, migrate summary refs, mark merged, delete embedding, recalculate feed_count
- Dedup-before-update pattern handles unique constraint violations on idx_article_topic_tags_link
- TagMatch exact and alias match steps filter out merged tags via activeTagFilter scope
- FindSimilarTags SQL JOIN excludes merged tags (status = active OR empty)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add merged status fields to TopicTag model and create migration** - `9391987` (feat)
2. **Task 2: Implement MergeTags with transaction-safe reference migration** - `97bf8f5` (feat)

## Files Created/Modified
- `backend-go/internal/domain/models/topic_graph.go` - Added Status, MergedIntoID, MergedInto fields to TopicTag
- `backend-go/internal/domain/topicanalysis/embedding.go` - Added MergeTags function, activeTagFilter helper, filtered TagMatch/FindSimilarTags
- `backend-go/internal/platform/database/postgres_migrations.go` - Migration 20260413_0003 adds status + merged_into_id columns with indexes

## Decisions Made
- Merged tags retained with status='merged' and merged_into_id pointer, never physically deleted (per D-04, D-05, D-07)
- Dedup-before-update pattern: check existing (article_id, target_tag_id) before migrating source link — avoids unique constraint violations without ON CONFLICT complexity
- activeTagFilter includes `OR status = ''` for rows created before migration (edge case where status might be empty string)
- Feed_count recalculated via subquery join through article_topic_tags → articles → article_feeds

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Tag merge infrastructure complete — MergeTags available for future manual merge UI and batch operations
- All match queries (TagMatch, FindSimilarTags) filter merged tags by default
- Phase 01 (Infrastructure Tag Convergence) fully complete
- Ready for downstream phases that depend on tag convergence (关注标签, 日报重构, etc.)

---
*Phase: 01-infrastructure-tag-convergence*
*Completed: 2026-04-13*

## Self-Check: PASSED
- All 3 key files verified on disk
- 2 task commits verified: 9391987, 97bf8f5
- Go build passes, all tests pass
