---
gsd_state_version: 1.0
milestone: v1.2
milestone_name: 标签智能收敛与关注推送
status: roadmap_created
last_updated: "2026-04-13T00:00:00.000Z"
last_activity: 2026-04-13 -- Roadmap created for v1.2 (5 phases, 22 requirements)
progress:
  total_phases: 5
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# STATE: Milestone v1.2 标签智能收敛与关注推送

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-12)

**Core value:** 通过智能标签系统帮助用户高效消费信息
**Current focus:** Phase 1 基础设施与标签收敛

## Current Position

Phase: 1 of 5 (基础设施与标签收敛)
Plan: — of —
Status: Roadmap created, ready to plan
Last activity: 2026-04-13 — Roadmap created for v1.2

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**
- Total plans completed: 0 (v1.2)
- v1.1 plans completed: 10

**By Phase:**

| Phase | Plans | Total | Status |
|-------|-------|-------|--------|
| 1. 基础设施与标签收敛 | 0/? | - | Not started |
| 2. 关注标签与首页推送 | 0/? | - | Not started |
| 3. 日报周报重构 | 0/? | - | Not started |
| 4. 标签历史趋势 | 0/? | - | Not started |
| 5. 相关标签推荐 | 0/? | - | Not started |

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions logged in PROJECT.md Key Decisions table. Recent:

- v1.2: 复用 airouter provider 框架 (CapabilityEmbedding)
- v1.2: 自动合并标签（非聚类展示），减少碎片
- v1.2: 新文章入库时实时触发收敛
- v1.2: 完全替换日报周报逻辑

### Pending Todos

None yet.

### Blockers/Concerns

- **CONV-02 风险**: 标签合并事务内迁移 article_topic_tags 引用是高风险操作，需事务完整性保障
- **INFRA-02 影响**: embedding 模型切换会导致现有阈值 (0.97/0.78) 失效，需考虑模型感知阈值
- **DIGEST-03 复杂度**: 4 个导出通道需同步适配 (前端/飞书/Obsidian/OpenNotebook)

### Research Notes

- Phase 1 (收敛) 需实际标签数据校验阈值，收敛质量取决于真实分布
- Phase 3 (日报重构) 各导出通道模板结构需在规划时研究
- Phase 5 (推荐) PMI/TF-IDF 权重需真实数据调参，0.6/0.4 融合权重为初始值

## Session Continuity

Last session: 2026-04-13
Stopped at: Roadmap created, ready for Phase 1 planning
Resume file: None

---

*Updated: 2026-04-13*
