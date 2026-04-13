# ROADMAP: Milestone v1.2 标签智能收敛与关注推送

## Overview

| Metric | Value |
|--------|-------|
| Milestone | v1.2 标签智能收敛与关注推送 |
| Phases | 6 |
| Requirements | 22+ |
| Coverage | 100% ✓ |

## Phases

| # | Phase | Goal | Requirements | Success Criteria |
|---|-------|------|--------------|------------------|
| 1 | 基础设施与标签收敛 | pgvector 迁移 + embedding 配置 + 标签自动收敛 | INFRA-01~03, CONV-01~04 | 4 |
| 2 | 关注标签与首页推送 | 关注标签 CRUD + 首页关注文章推送 | WATCH-01~03, FEED-01~03 | 4 |
| 3 | 日报周报重构 | 从分类聚合改为关注标签视角，适配所有导出通道 | DIGEST-01~04 | 4 |
| 4 | 标签历史趋势 | AI 生成标签主题叙事分析 | TRENDS-01~03 | 3 |
| 5 | 相关标签推荐 | 基于关注标签推荐相关标签 | REC-01~02 | 2 |
| 6 | 标签合并交互界面 | 手动触发全量标签合并、预览结果、修改合并名称、查看合并前后差异 | CONV-02 | TBD |

## Phase Details

### Phase 1: 基础设施与标签收敛

**Goal**: 标签系统具备 pgvector 向量搜索能力，新文章入库时语义相近标签自动合并，标签空间不再碎片化

**Requirements**: INFRA-01, INFRA-02, INFRA-03, CONV-01, CONV-02, CONV-03, CONV-04

**Success Criteria:**
1. 新文章入库时，如果已存在语义相近标签（相似度 ≥ 阈值），自动复用已有标签而非创建新标签
2. 标签合并后，关联文章的标签引用正确迁移到目标标签，旧标签标记为 merged 状态保留历史可追溯
3. 相似度搜索通过 pgvector SQL `<=>` 运算符完成，Go 侧不再循环遍历全表计算余弦距离
4. embedding 模型名从 provider 配置动态读取，收敛阈值可通过 API 调整无需重启

**Plans:** 4 plans in 3 waves

Plans:
- [x] 01-01-PLAN.md — pgvector 迁移 + embedding 配置表 + API (INFRA-01~03)
- [x] 01-02-PLAN.md — TagMatch 集成 findOrCreateTag 三级匹配 (CONV-01, CONV-03)
- [x] 01-03-PLAN.md — 标签合并事务 + merged 状态保留 (CONV-02, CONV-04)
- [ ] 01-04-PLAN.md — Embedding 配置前端界面 (INFRA-03, gap closure)

**Files affected:**
- `backend-go/internal/platform/database/postgres_migrations.go`
- `backend-go/internal/domain/models/topic_graph.go`
- `backend-go/internal/domain/models/embedding_config.go`
- `backend-go/internal/domain/topicanalysis/embedding.go`
- `backend-go/internal/domain/topicanalysis/config_service.go`
- `backend-go/internal/domain/topicanalysis/handler.go`
- `backend-go/internal/domain/topicextraction/tagger.go`
- `backend-go/internal/app/router.go`

---

### Phase 2: 关注标签与首页推送

**Goal**: 用户可以关注特定标签，首页看到关注标签关联的文章推送

**Depends on**: Phase 1 (收敛完成后的干净标签空间)

**Requirements**: WATCH-01, WATCH-02, WATCH-03, FEED-01, FEED-02, FEED-03

**Success Criteria:**
1. 用户可以在标签列表页勾选/取消关注标签，关注状态持久化并记录 watched_at 时间
2. 首页展示关注标签关联的文章流，按时间倒序排列
3. 用户可按单个关注标签筛选文章，文章列表支持按相关度排序（匹配标签数、embedding 距离加权）
4. 无关注标签时首页回退到完整时间线，不显示空白

**Plans:**
- [ ] 02-01: [待规划]

**Files affected:**
- `backend-go/internal/domain/models/topic_tag.go` (新增 watched 字段)
- `backend-go/internal/domain/topicextraction/handler.go` (关注 API)
- `front/app/pages/` (首页关注推送)
- `front/app/api/tags.ts` (前端 API)
- `front/app/stores/` (关注标签 store)

---

### Phase 3: 日报周报重构

**Goal**: 日报周报从分类聚合完全替换为关注标签视角，所有导出通道正确输出

**Depends on**: Phase 2 (关注标签数据)

**Requirements**: DIGEST-01, DIGEST-02, DIGEST-03, DIGEST-04

**Success Criteria:**
1. 日报/周报按关注标签聚合文章，不再按分类聚合
2. 用户可通过前端手动触发生成日报/周报
3. 所有导出通道（前端展示、飞书、Obsidian、OpenNotebook）正确输出关注标签视角内容
4. 无关注标签时显示降级提示信息，不报错或空白

**Plans:**
- [ ] 03-01: [待规划]

**Files affected:**
- `backend-go/internal/domain/digest/` (生成逻辑重写)
- `front/app/api/digest.ts`
- `front/app/pages/digest/`

---

### Phase 4: 标签历史趋势

**Goal**: 用户可选择标签查看 AI 生成的主题叙事分析，了解标签下的信息脉络

**Depends on**: Phase 2 (关注标签数据)

**Requirements**: TRENDS-01, TRENDS-02, TRENDS-03

**Success Criteria:**
1. 用户可选择关注标签或手动选择任意标签，指定时间范围，生成该标签的主题叙事总结（AI 生成）
2. 主题叙事包含：事件来龙去脉、人物/实体时间线、综合评价总结
3. 时间范围限定可控制叙事内容的边界

**Plans:**
- [ ] 04-01: [待规划]

**Files affected:**
- `backend-go/internal/domain/topicanalysis/` (叙事生成)
- `backend-go/internal/domain/articles/` (标签关联文章查询)
- `front/app/pages/` (趋势分析页面)

---

### Phase 5: 相关标签推荐

**Goal**: 基于关注标签推荐语义相近和共现频率高的相关标签

**Depends on**: Phase 1 (embedding 基础设施), Phase 2 (关注标签列表)

**Requirements**: REC-01, REC-02

**Success Criteria:**
1. 关注标签管理页面或标签详情页展示推荐的相关标签，综合 embedding 相似度和同文章共现频次
2. 推荐结果排除已关注标签，点击推荐标签可查看详情或直接关注

**Plans:**
- [ ] 05-01: [待规划]

**Files affected:**
- `backend-go/internal/domain/topicanalysis/embedding.go` (相似度查询)
- `backend-go/internal/domain/topicgraph/` (共现计算)
- `front/app/pages/` (推荐展示)

---

## Dependencies

```
Phase 1 (INFRA+CONV) ──┬── Phase 2 (WATCH+FEED) ──┬── Phase 3 (DIGEST)
                       │                          │
                       │                          ├── Phase 4 (TRENDS)
                       │                          │
                       └──────────────────────────┴── Phase 5 (REC)

Phase 1 ── Phase 6 (标签合并交互界面)
```

执行顺序: 1 → 2 → 3 → 4 → 5 (Phases 3/4 可并行，Phase 5 需 1+2)
Phase 6 可在 Phase 1 之后任意时间执行

---

### Phase 6: 标签合并交互界面

**Goal**: 用户可手动触发全量标签合并扫描，预览待合并标签对（源→目标），修改合并后标签名称，确认后执行合并，查看合并前后差异

**Depends on**: Phase 1 (embedding 基础设施 + MergeTags)

**Requirements**: CONV-02

**Success Criteria:**
1. 用户可在前端手动触发全量标签相似度扫描，系统返回所有高相似度标签对（>= 0.97）
2. 预览界面展示每对标签的源名称、目标名称、相似度、各自关联文章数
3. 用户可修改合并后的标签名称（不限于源或目标的名称）
4. 用户可逐对确认或跳过合并，也可一键全部合并
5. 合并完成后展示结果汇总：哪些标签被合并、合并后的新名称、迁移的文章数

**Plans:** 3 plans in 3 waves

Plans:
- [ ] 06-01-PLAN.md — Backend scan-preview API + merge-with-name API (CONV-02)
- [ ] 06-02-PLAN.md — Frontend API layer for preview and custom merge
- [ ] 06-03-PLAN.md — TagMergePreview.vue component (cards, inline edit, summary)

**Files affected:**
- `backend-go/internal/domain/topicanalysis/tag_merge_preview.go` (scan logic extracted)
- `backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go` (preview & custom merge APIs)
- `backend-go/internal/app/router.go` (route registration)
- `front/app/api/tagMergePreview.ts` (frontend API)
- `front/app/types/tagMerge.ts` (type definitions)
- `front/app/features/topic-graph/components/TagMergePreview.vue` (UI component)
- `front/app/pages/topics.vue` (entry point)

---

## Verification

**After all phases:**
1. 新文章入库标签自动收敛，无重复标签
2. 关注标签推送文章准确、筛选正常
3. 日报周报按关注标签输出，4 通道正确
4. 标签叙事分析内容合理、时间范围可控
5. 相关标签推荐有意义且不重复
6. 标签合并交互界面功能完整，用户可预览、修改、确认合并

---

*Generated by GSD roadmap workflow*
*Updated: 2026-04-13*
