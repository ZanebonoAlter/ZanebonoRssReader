# Technology Stack — v1.2 标签智能收敛与关注推送

**Project:** RSS Reader — v1.2 Milestone
**Researched:** 2026-04-12
**Mode:** Ecosystem (targeted to new capabilities only)

## Executive Summary

This milestone's new features require **zero new dependencies**. The existing stack — Go/Gin/GORM/PostgreSQL + pgvector + airouter embedding infrastructure + Nuxt 4/Vue 3/Pinia — already provides everything needed. The work is entirely about new domain logic, schema additions, and API/UI endpoints layered on existing infrastructure.

The one infrastructure change worth noting: the codebase already has a staged pgvector column (`topic_tag_embeddings.embedding vector(1536)`) that was prepared in migration `20260403_0003`. This milestone should **complete the cutover** from the legacy JSON text payload to native pgvector queries for similarity search, replacing the Go-side `CosineSimilarity()` loop with SQL `1 - (embedding <=> $1)` operations.

## Recommended Stack

### Backend — No New Libraries

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| pgvector (PostgreSQL extension) | Already installed | Native vector similarity search via `<=>` operator | Replaces Go-side cosine similarity loop in `FindSimilarTags`; migration `20260403_0003` already added the `vector(1536)` column — just need to write to it and query it |
| GORM AutoMigrate | Already in use | Schema migrations for new columns/tables | New: `topic_tags.is_watched`, `tag_trend_snapshots` table, `watched_tag_digest_runs` table |
| robfig/cron/v3 | v3.0.1 (existing) | Digest scheduler — extend with watched-tag mode | Already drives daily/weekly digest; add new cron entries for trend snapshot and watched-tag digest |
| airouter.EmbeddingClient | Existing | Generate embeddings for new tags | Already works; no changes needed |
| airouter.CosineSimilarity | Existing (legacy) | Fallback similarity when pgvector not available | Keep as safety net, but primary path should use pgvector SQL |

### Frontend — No New Libraries

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Vue 3 Composition API | Existing | UI components for watched tags, trends, digest redesign | Standard approach per project conventions |
| Pinia | Existing | State management for watched tags list, trend data | Derive from existing `useApiStore` pattern |
| Chart.js or lightweight alternative | **New** | Tag trend line charts (historical article counts) | See analysis below |

### New Chart Library Decision

**Recommendation: Use CSS-only sparklines / simple SVG for trend visualization, NOT a full charting library.**

Rationale:
- The trend data is simple: 7-30 data points per tag, line chart only
- Project avoids generic SaaS aesthetics; custom SVG/CSS sparklines match the editorial feel better
- Chart.js adds ~70KB; Chart.js + vue-chartjs wrapper is overkill for this
- Alternative: A minimal inline SVG component using `<polyline>` + `<path>` — ~50 lines per component, zero dependency
- If more chart types are needed later (bar charts for co-occurrence matrix), re-evaluate at that point

**If a chart library is still preferred:** Use `chart.js@4` + `vue-chartjs@5` — most widely used, good TypeScript support, tree-shakeable. But the editorial/magazine design philosophy of this project favors custom-built visuals.

## New Database Schema

### New Columns (existing tables)

```sql
-- topic_tags: watched/followed flag
ALTER TABLE topic_tags ADD COLUMN IF NOT EXISTS is_watched BOOLEAN DEFAULT false;
CREATE INDEX IF NOT EXISTS idx_topic_tags_is_watched ON topic_tags(is_watched) WHERE is_watched = true;

-- topic_tags: last trend snapshot timestamp (avoid redundant recomputation)
ALTER TABLE topic_tags ADD COLUMN IF NOT EXISTS trend_updated_at TIMESTAMPTZ;
```

### New Tables

```sql
-- Historical trend snapshots for watched tags
-- Populated by a daily scheduled job
CREATE TABLE IF NOT EXISTS tag_trend_snapshots (
    id SERIAL PRIMARY KEY,
    topic_tag_id INTEGER NOT NULL REFERENCES topic_tags(id) ON DELETE CASCADE,
    snapshot_date DATE NOT NULL,
    article_count INTEGER NOT NULL DEFAULT 0,
    feed_count INTEGER NOT NULL DEFAULT 0,
    avg_score FLOAT NOT NULL DEFAULT 0,
    UNIQUE(topic_tag_id, snapshot_date)
);
CREATE INDEX IF NOT EXISTS idx_tag_trend_snapshots_tag_date ON tag_trend_snapshots(topic_tag_id, snapshot_date DESC);

-- Watched-tag digest run history
CREATE TABLE IF NOT EXISTS watched_tag_digest_runs (
    id SERIAL PRIMARY KEY,
    digest_type VARCHAR(10) NOT NULL, -- 'daily' or 'weekly'
    anchor_date DATE NOT NULL,
    tag_count INTEGER NOT NULL DEFAULT 0,
    article_count INTEGER NOT NULL DEFAULT 0,
    markdown TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(digest_type, anchor_date)
);
```

### pgvector Column Activation

The `topic_tag_embeddings.embedding vector(1536)` column already exists (migration `20260403_0003`). Changes needed:

1. **Write path**: When `SaveEmbedding()` is called, also populate the `embedding` column with the same vector data
2. **Read path**: Replace `FindSimilarTags` Go-side loop with:
   ```sql
   SELECT topic_tag_id, 1 - (embedding <=> $1) AS similarity
   FROM topic_tag_embeddings
   JOIN topic_tags ON topic_tags.id = topic_tag_embeddings.topic_tag_id
   WHERE topic_tags.category = $2
     AND topic_tag_id != $3
     AND embedding IS NOT NULL
   ORDER BY embedding <=> $1
   LIMIT $4;
   ```
3. **Keep `Vector` JSON column**: Don't drop it yet — it's the fallback if pgvector isn't available

## Integration Points with Existing Code

### Tag Convergence — Where to Hook

| Existing Code | Integration Point | What to Change |
|---------------|-------------------|----------------|
| `topicanalysis.EmbeddingService.FindSimilarTags` | Replace with pgvector SQL query | Core similarity search; currently loads all embeddings into Go memory |
| `topicanalysis.EmbeddingService.SaveEmbedding` | Also write to `embedding vector(1536)` column | Dual-write during cutover |
| `topicanalysis.EmbeddingService.TagMatch` | Adjust thresholds per convergence design doc | Already has 3-tier logic (exact → high_sim → ai_judgment) |
| `topicextraction.extractor_enhanced.resolveCandidate` | Unified scoring: `similarity × ln(feed_count + 1)` | Per `docs/plans/2026-03-23-tag-weight-convergence-design.md` |
| `topicextraction.article_tagger.tagArticle` | Call `RecalculateFeedCount` after tag create/delete | FeedCount already added to model |

### Watched Tags — New Domain

| New Component | Location | Depends On |
|---------------|----------|------------|
| `WatchedTagService` | `backend-go/internal/domain/watchedtags/` | `models.TopicTag`, GORM |
| `WatchedTagHandler` | Same package | Gin routes |
| Watched tag API routes | `router.go` → `/api/watched-tags` | Handler |
| Frontend `watchedTags.ts` API | `front/app/api/` | `apiClient` |
| Frontend composable `useWatchedTags` | `front/app/composables/` | API layer |
| Pinia store (optional) | `front/app/stores/` | If watched tags need global reactivity |

### Digest Redesign — Extend Existing

| Existing Code | Integration Point | What to Change |
|---------------|-------------------|----------------|
| `digest.Generator.GenerateDailyDigest` | Add watched-tag variant `GenerateWatchedDigest` | Filter summaries by watched tags; new grouping logic |
| `digest.handler.buildPreview` | New preview mode: `?mode=watched` | Keep existing mode working, add parallel mode |
| `digest.Scheduler` | Add watched-tag cron entry | Uses same scheduler infrastructure |
| `front/app/api/digest.ts` | Add `mode: 'watched'` query param | Extend existing API types |
| Digest pages | Redesign layout for tag-centric view | Biggest UI change |

### Tag Trend Analysis — New Scheduled Job

| Component | Location | Depends On |
|-----------|----------|------------|
| `TagTrendService` | `backend-go/internal/domain/tagtrends/` | `article_topic_tags`, `tag_trend_snapshots` |
| Daily snapshot job | Integrated into existing scheduler pattern | `robfig/cron` |
| Trend query API | `router.go` → `/api/tag-trends/:slug` | Handler |
| Related tag recommendations | Extend `topicgraph.getRelatedTags` | Combine embedding similarity + co-occurrence |

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Vector search | pgvector SQL `<=>` operator | Keep Go-side CosineSimilarity loop | Go-side loads ALL embeddings into memory per query; pgvector does it in SQL with index support. For ~1000 tags it works either way, but pgvector is the intended architecture (migration already prepared) |
| Trend storage | Daily snapshot table | Compute on-the-fly from `article_topic_tags` | On-the-fly requires scanning all articles per request; snapshot is O(1) for reads, O(watched_tags) for daily write |
| Chart rendering | Custom SVG/CSS sparklines | Chart.js + vue-chartjs | Zero new deps, matches editorial design philosophy, trend data is simple enough |
| Watched tags storage | Column on `topic_tags` | Separate `watched_tags` table | Column is simpler (no join needed), boolean flag with partial index is efficient for the single-user case |
| Digest architecture | Parallel mode in existing package | New separate package | Avoids code duplication; the scheduler, exporters, and config infrastructure are all reusable |

## What NOT to Add

| Library/Tool | Why Not |
|-------------|---------|
| Any Go embedding library (e.g., `pgx` vector types) | pgvector SQL works through raw GORM queries; no special Go types needed — just pass the vector as a string literal `[0.1, 0.2, ...]` |
| Redis for tag caching | Single-user app, PostgreSQL handles this fine |
| Message queue for tag events | TagJobQueue already exists; watched tag operations are synchronous DB writes |
| Elasticsearch / Meilisearch | pgvector handles vector search; full-text search isn't needed for tags |
| GraphQL | REST API with Gin is the established pattern |
| WebSocket for trend data | Trends update daily; no real-time need |
| Time-series database (TimescaleDB) | 7-30 data points per tag, daily granularity — trivial for PostgreSQL |
| Any new Go dependency | Zero new `go get` needed |

## Installation

```bash
# Backend — NO new packages needed
cd backend-go
# Just run existing migrations (GORM AutoMigrate handles new columns)
go run cmd/server/main.go

# Frontend — NO new packages needed (unless choosing Chart.js)
cd front
# If Chart.js is chosen:
pnpm add chart.js vue-chartjs
# Otherwise: zero installs
```

## pgvector Migration Strategy

The cutover from JSON vectors to pgvector is the most impactful infrastructure change:

1. **Phase A: Dual-write** — `SaveEmbedding()` writes to both `Vector` (JSON text) and `embedding` (vector column)
2. **Phase B: Read from pgvector** — `FindSimilarTags()` uses SQL `<=>` operator, falls back to Go-side loop if `embedding IS NULL`
3. **Phase C: Backfill** — One-time migration to populate `embedding` column from existing `Vector` JSON
4. **Phase D: Remove Go-side fallback** — After confirming all rows have `embedding` populated

This staged approach means zero downtime and zero risk of data loss.

## Sources

- Codebase analysis: `topicanalysis/embedding.go`, `airouter/embedding.go`, `digest/` package, `topicgraph/service.go`
- Migration `20260403_0003`: pgvector column already staged
- `docs/plans/2026-03-23-tag-weight-convergence-design.md`: convergence scoring formula
- pgvector cosine distance operator `<=>` — documented at github.com/pgvector/pgvector
- Confidence: **HIGH** — all findings verified by direct code reading, no external library claims
