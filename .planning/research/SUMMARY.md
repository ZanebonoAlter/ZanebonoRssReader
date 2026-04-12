# Project Research Summary

**Project:** RSS Reader — v1.2 标签智能收敛与关注推送
**Domain:** Tag-intelligent RSS reader with AI-powered entity resolution and personalized digest
**Researched:** 2026-04-12
**Confidence:** HIGH

## Executive Summary

This milestone adds intelligent tag convergence and watched-tag push capabilities to an existing RSS Reader built on Go/Gin/GORM + Nuxt 4/Vue 3. The research reveals that **zero new dependencies** are needed — all required infrastructure (pgvector extension, airouter embedding client, cron scheduler, digest export pipeline) already exists in the codebase. The work is purely domain logic, schema additions, and API/UI endpoints layered on proven foundations.

The recommended approach follows a strict dependency chain: **(1) complete pgvector cutover + tag auto-convergence** → **(2) watched tags CRUD** → **(3) watched article push + digest redesign** → **(4) trend analysis** → **(5) related tag recommendations**. The convergence phase is the critical dependency — all downstream "watched tag" features are meaningless if the tag space remains fragmented. The convergence hook must be inserted inside `findOrCreateTag()` (the sole tag creation entry point), and the pgvector SQL `<=>` operator should replace the Go-side cosine similarity loop for performance.

The highest-risk areas are: (1) tag merge cascading — when tags merge, all `article_topic_tags` references must be migrated within a single transaction, or watched-tag queries silently miss articles; (2) embedding model switching invalidates hardcoded similarity thresholds (0.97/0.78 calibrated for ada-002); (3) digest redesign must update all three export channels (Feishu, Obsidian, Open Notebook) simultaneously — the old `CategoryDigest` struct is deeply embedded in every exporter. Each of these has clear prevention strategies documented below.

## Key Findings

### Recommended Stack

No new packages required. The existing stack provides everything:

- **pgvector `<=>` operator** — native SQL cosine distance, replacing Go-side loop. Column `vector(1536)` already staged in migration `20260403_0003`; needs dual-write + read cutover.
- **robfig/cron/v3** (existing) — extend scheduler with watched-tag digest mode and daily trend snapshot jobs.
- **airouter.EmbeddingClient** (existing) — generate embeddings for new tags; no API changes needed.
- **Custom SVG/CSS sparklines** — for trend visualization, avoiding Chart.js (~70KB). Editorial design philosophy favors bespoke visuals; trend data is simple (7-30 data points, line chart only).

### Expected Features

**Must have (table stakes):**
- **标签自动收敛** — embedding similarity matching inside `findOrCreateTag()`, high-similarity auto-merge with alias tracking. This is the foundation; without it, the tag space stays fragmented and all downstream features degrade.
- **关注标签勾选** — `watched_tags` table + CRUD API + tag list UI with toggle. Low complexity but blocks all other watched-tag features.
- **关注文章推送** — multi-tag OR query with article deduplication, cold-start fallback to full timeline, homepage feed section.
- **日报周报关注视角** — complete replacement of `DigestGenerator` from category-based to watched-tag-based aggregation, all three export channels adapted.

**Should have (differentiators):**
- **相关标签推荐** — dual-signal fusion (0.6× embedding similarity + 0.4× normalized co-occurrence), excludes already-watched tags.
- **标签历史趋势分析** — `article_topic_tags` time-bucketed aggregation with delta calculation, backend aggregation to keep frontend lightweight.

**Defer (v2+):**
- **Embedding 模型可配置 UI** — framework exists (`airouter.Router.ResolvePrimaryProvider(CapabilityEmbedding)`), but threshold re-calibration when switching models is a non-trivial problem. Quick win to expose in settings, but the threshold management is the hard part.
- **AI judgment tier** — the third tier of `TagMatch` (embedding similarity falls in the 0.78–0.97 middle ground). Can be deferred; current plan treats middle-ground as "create new tag."

### Architecture Approach

Extend three existing domain packages (`topicextraction`, `topicanalysis`, `digest`) rather than creating new ones. Only one new lightweight package (`watchedtags` — handler + service, model goes in shared `models/`). The convergence hook goes inside `findOrCreateTag()` because it's the sole tag creation path — this avoids missing convergence points.

**Major components:**
1. **topicextraction (modified)** — `findOrCreateTag()` gains convergence branch: slug miss → `EmbeddingService.TagMatch()` → merge or create.
2. **topicanalysis (extended)** — new `FindAndMergeSimilarTag()` method, trend analysis as new `analysis_type`, pgvector read cutover.
3. **watchedtags (new)** — CRUD handler + service, multi-tag article query, related tag recommendations.
4. **digest (rewritten)** — `WatchedTagDigestGenerator` replaces category-based generator; scheduler and export channels preserved, data source changed.
5. **Frontend** — new watched tags composable/store, homepage watched feed section, digest page redesign, custom SVG trend charts.

### Critical Pitfalls

1. **Merge cascade — dangling references** — When tag A merges into tag B, all `article_topic_tags` and `ai_summary_topics` rows must be migrated in the same transaction. Without this, watched-tag queries miss articles silently.
2. **Embedding model switch invalidates thresholds** — `HighSimilarity: 0.97` / `LowSimilarity: 0.78` are calibrated for `text-embedding-ada-002`. `text-embedding-3-small` produces lower similarity scores. Thresholds must be stored per-model or auto-adjusted.
3. **pgvector performance — JSON text fallback** — Current `FindSimilarTags` loads all embeddings into Go memory. Must complete pgvector cutover before convergence goes live, or tag processing blocks on every article ingestion.
4. **Digest redesign breaks all export channels** — Feishu, Obsidian, and Open Notebook exporters all assume `CategoryDigest` struct. Every exporter must be adapted to `WatchedTagDigest` simultaneously. Frontend digest pages must update in lockstep.
5. **Cold-start blank homepage** — If user has zero watched tags, the homepage feed must fall back to the existing full timeline, not show an empty page.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: pgvector 迁移 + 标签自动收敛
**Rationale:** pgvector cutover is a prerequisite for performant convergence. Convergence is the critical dependency — all watched-tag features require a clean tag space.
**Delivers:** Dual-write to pgvector column, SQL-based similarity search, `findOrCreateTag()` convergence hook with merge-in-transaction logic.
**Addresses:** 标签自动收敛 (table stakes), pgvector activation (infrastructure).
**Avoids:** Pitfall 1 (dangling references via transactional merge), Pitfall 3 (pgvector perf via SQL `<=>` operator).
**Pitfalls to watch:** Pitfall 2 (threshold model lock-in), Pitfall 8 (embedding API rate limiting during bulk ingestion), Pitfall 9 (merge direction — prefer LLM-sourced labels).

### Phase 2: 关注标签 CRUD + 首页推送
**Rationale:** Watched tags are the data foundation for digest redesign, trends, and recommendations. Article push delivers the first tangible user value.
**Delivers:** `watched_tags` table, CRUD API, homepage watched-article feed with multi-tag OR query and deduplication.
**Uses:** Existing `article_topic_tags` joins, Pinia store extension.
**Implements:** `watchedtags` package (handler + service), frontend composable + feed component.
**Avoids:** Pitfall 5 (cold-start fallback to full timeline), Pitfall 10 (independent `watched_tags` table, not boolean on `topic_tags`), Pitfall 12 (string ID mapping at API boundary).
**Pitfalls to watch:** Pitfall 13 (tag list pagination for 1000+ tags).

### Phase 3: 日报周报重构
**Rationale:** Depends on watched tags (Phase 2) for data source. Complete replacement per PROJECT.md decision, not a new view alongside the old one.
**Delivers:** `WatchedTagDigestGenerator`, adapted Feishu/Obsidian/Open Notebook exporters, updated frontend digest pages.
**Uses:** Existing digest scheduler framework (`robfig/cron`), existing export channel plumbing.
**Implements:** Digest package rewrite — new `WatchedTagDigestItem` struct, tag-centric markdown generation.
**Avoids:** Pitfall 4 (all export channels tested individually), Pitfall 11 (unified CST timezone constant).

### Phase 4: 标签趋势分析
**Rationale:** Independent from digest redesign; depends only on watched tags (Phase 2). Can technically run in parallel with Phase 3.
**Delivers:** Trend API (time-bucketed `article_topic_tags` aggregation), custom SVG sparkline components, trend direction detection.
**Uses:** PostgreSQL `date_trunc` for time bucketing, `topicanalysis` extension.
**Avoids:** Pitfall 6 (data source: use `article_topic_tags`, not `AnalysisService` summary aggregation), Pitfall 15 (backend aggregation + max 5 tags comparison).

### Phase 5: 相关标签推荐
**Rationale:** Needs embedding similarity (Phase 1) + watched tag list (Phase 2). Best done last when there's enough watched-tag data to make recommendations meaningful.
**Delivers:** Dual-signal recommendation API, recommendation UI in tag detail/watched tags page.
**Uses:** `FindSimilarTags` (embedding signal) + `fetchCoOccurrence` (co-occurrence signal), weighted fusion.
**Avoids:** Pitfall 7 (PMI/TF-IDF weighting to suppress high-frequency tag bias).

### Phase Ordering Rationale

- **Phase 1 → 2**: Convergence produces a clean tag space; watched tags on a fragmented space means users watch 3 variants of the same topic.
- **Phase 2 → 3**: Digest needs watched tags to aggregate by; can't generate tag-centric digest without knowing which tags are watched.
- **Phase 2 → 4**: Trend analysis targets watched tags primarily; can work without it but the primary use case is watched-tag trends.
- **Phase 1+2 → 5**: Recommendations need both embedding infrastructure and watched tag list (to exclude already-watched).
- **Phases 3 and 4 are independent** of each other and could theoretically be parallelized if resources allow.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 1 (收敛):** Convergence threshold calibration is nuanced — needs testing against real tag data to find the right balance between over-merge and under-merge. The pgvector dual-write cutover strategy is well-documented but needs careful implementation to avoid data loss.
- **Phase 3 (日报重构):** Each export channel (Feishu, Obsidian, Open Notebook) has its own format expectations. Research the exact template structures during planning.
- **Phase 5 (推荐):** PMI/TF-IDF weighting needs tuning against actual tag distribution data; the 0.6/0.4 fusion weights are starting points.

Phases with standard patterns (skip research-phase):
- **Phase 2 (关注标签):** Straightforward CRUD + JOIN query; well-established patterns in the codebase.
- **Phase 4 (趋势):** SQL time-bucketed aggregation is a solved problem; `buildTrendData` already exists as a reference implementation.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Zero new dependencies; all infrastructure verified by direct code reading. pgvector column already staged in migration. |
| Features | HIGH | Clear table stakes and differentiators identified. Feature dependencies mapped. Existing codebase leverage documented for every feature. |
| Architecture | HIGH | All integration points identified with file+line references. Component boundaries align with existing domain packages. Only one new package needed. |
| Pitfalls | HIGH | 5 critical pitfalls with specific code references and prevention strategies. Phase-specific warnings map directly to implementation phases. |

**Overall confidence:** HIGH

### Gaps to Address

- **Convergence threshold calibration:** The 0.97 high-similarity threshold is reasonable for ada-002 but needs empirical validation against this codebase's actual tag distribution. During Phase 1 planning, define a test harness that runs convergence on existing tags and measures merge quality.
- **Embedding model change detection:** The `getEmbeddingModel()` function currently hardcodes `text-embedding-ada-002`. Need to wire it to the actual provider config during Phase 1, with a staleness marker for existing embeddings when the model changes.
- **Frontend digest page redesign scope:** ARCHITECTURE.md specifies "redesign layout for tag-centric view" but the exact UI components haven't been designed. Phase 3 planning should include a UI spec.
- **Watched tags empty-state UX:** What happens when a user has never watched any tags? The homepage fallback is defined, but the digest in this case needs a clear design — PROJECT.md says "complete replacement," so what does digest show with zero watched tags?

## Sources

### Primary (HIGH confidence)
- Codebase analysis of 15+ source files: `topicanalysis/embedding.go`, `topicextraction/tagger.go`, `digest/generator.go`, `digest/scheduler.go`, `topicgraph/service.go`, `models/topic_graph.go`, `airouter/embedding.go`
- Migration `20260403_0003`: pgvector column already staged
- `docs/plans/2026-03-23-tag-weight-convergence-design.md`: convergence scoring formula
- PROJECT.md v1.2 milestone definition and key decisions
- AGENTS.md project conventions

### Secondary (MEDIUM confidence)
- pgvector `<=>` cosine distance operator — documented at github.com/pgvector/pgvector
- OpenAI community discussion #873048 — `text-embedding-3-small` produces lower cosine similarity than `ada-002`
- Feedly topic tracking patterns — reference for watched-tag UX expectations
- Stack Overflow z-score approach for trend anomaly detection

### Tertiary (LOW confidence)
- Entity resolution patterns from arxiv/html/2506.02509v1 — academic reference for tag matching
- PMI weighting for co-occurrence bias correction — theoretical, needs empirical validation

---
*Research completed: 2026-04-12*
*Ready for roadmap: yes*
