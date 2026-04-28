---
phase: 07-middle-band
plan: 01
subsystem: api, database
tags: [gorm, pgvector, llm, airouter, gin, abstract-tags, hierarchy]

requires:
  - phase: 01
    provides: embedding infrastructure, TagMatch, airouter, Slugify
provides:
  - topic_tag_relations table and TopicTagRelation model
  - ExtractAbstractTag: LLM-based abstract concept extraction from middle-band candidates
  - GetTagHierarchy: recursive tag hierarchy tree query
  - UpdateAbstractTagName: rename abstract tag with slug dedup + async re-embedding
  - DetachChildTag: remove child from parent without deleting parent
  - 3 HTTP endpoints: GET hierarchy, PUT abstract-name, POST detach
  - tagger.go ai_judgment branch integration with graceful fallback
affects: [07-02-frontend, tag-matching, topic-extraction]

tech-stack:
  added: []
  patterns: [llm-abstract-extraction, graceful-degradation, async-embedding, hierarchy-tree]

key-files:
  created:
    - backend-go/internal/domain/models/topic_tag_relation.go
    - backend-go/internal/domain/topicanalysis/abstract_tag_service.go
    - backend-go/internal/domain/topicanalysis/abstract_tag_service_test.go
    - backend-go/internal/domain/topicanalysis/abstract_tag_handler.go
  modified:
    - backend-go/internal/platform/database/postgres_migrations.go
    - backend-go/internal/domain/topicextraction/tagger.go
    - backend-go/internal/app/router.go

key-decisions:
  - "Reuse CapabilityTopicTagging for LLM abstract extraction (no new capability needed)"
  - "Articles associate with child tags, abstract tags only for aggregation (per D-03)"
  - "Abstract tag category inherited from first candidate tag (per Claude's Discretion)"
  - "LLM failure gracefully degrades to creating normal tag (per D-02)"

patterns-established:
  - "Abstract tag extraction: middle-band candidates → LLM → abstract_name → tag+relation creation"
  - "Handler pattern: validate → service call → status-coded response"
  - "Async re-embedding on abstract tag rename"

requirements-completed: [CONV-03, NEW-01, NEW-02]

duration: 5min
completed: 2026-04-14
---

# Phase 07 Plan 01: Backend Abstract Tag Service Summary

**Topic_tag_relations table + LLM abstract concept extraction + hierarchy API + tagger.go middle-band integration**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-14T00:34:59Z
- **Completed:** 2026-04-14T00:40:00Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- TopicTagRelation model + PostgreSQL migration for parent-child tag hierarchy
- ExtractAbstractTag service using LLM (airouter) to extract common abstract concepts from middle-band candidates
- Graceful degradation: LLM failure falls back to normal tag creation
- 3 API endpoints registered: GET /topic-tags/hierarchy, PUT /topic-tags/:id/abstract-name, POST /topic-tags/:id/detach
- tagger.go ai_judgment branch now attempts abstract tag extraction before creating new tags
- Unit tests for prompt construction and JSON parsing

## Task Commits

1. **Task 1: Data model + abstract tag extraction service** - `b952e21` (feat)
2. **Task 2: Handler + routes + tagger.go integration** - `eeaf253` (feat)

## Files Created/Modified
- `backend-go/internal/domain/models/topic_tag_relation.go` - TopicTagRelation GORM model
- `backend-go/internal/platform/database/postgres_migrations.go` - topic_tag_relations migration
- `backend-go/internal/domain/topicanalysis/abstract_tag_service.go` - Core service: ExtractAbstractTag, GetTagHierarchy, UpdateAbstractTagName, DetachChildTag
- `backend-go/internal/domain/topicanalysis/abstract_tag_service_test.go` - Unit tests
- `backend-go/internal/domain/topicanalysis/abstract_tag_handler.go` - HTTP handlers + RegisterAbstractTagRoutes
- `backend-go/internal/domain/topicextraction/tagger.go` - ai_judgment branch integration
- `backend-go/internal/app/router.go` - Route registration

## Decisions Made
- Reused CapabilityTopicTagging for abstract tag LLM calls (same topic_tagging route, no new AI provider needed)
- Articles associate with the highest-similarity child tag, not the abstract tag (per D-03)
- Abstract tag inherits category from first candidate tag
- Abstract name max length 160 chars to prevent LLM abuse (per T-07-02)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## Next Phase Readiness
- Backend APIs ready for frontend consumption (Plan 07-02)
- GET /api/topic-tags/hierarchy returns TagHierarchyNode tree
- PUT /api/topic-tags/:id/abstract-name updates name+slug+async re-embedding
- POST /api/topic-tags/:id/detach removes child relation

---
*Phase: 07-middle-band*
*Completed: 2026-04-14*
