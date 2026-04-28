# MILESTONES

## Completed Milestones

---

### v1.1 业务漏洞修复

**Shipped:** 2026-04-11
**Phases:** 6 phases, 10 plans
**Status:** Complete

修复代码审查发现的 6 类业务漏洞（状态一致性、级联删除、阻塞恢复等）。

---

### v1.2 标签智能收敛与关注推送

**Shipped:** 2026-04-15
**Phases:** 8 phases (5 completed, 3 skipped)
**Plans:** 20 completed + 6 quick tasks
**Timeline:** 3 days (2026-04-13 → 2026-04-15)
**Commits:** 122

**Key Accomplishments:**
1. pgvector 向量搜索基础设施 + SQL 级余弦距离搜索
2. 三级标签匹配 + 新文章入库实时自动收敛
3. 关注标签 CRUD + 首页关注文章推送（相关度排序）
4. 标签合并交互界面（扫描预览、自定义名称、批量确认）
5. LLM 抽象标签提取 + 递归层级树
6. 标签树增强（Description、时间筛选、图谱发光、节点归类）

**Skipped (设计方向变更):** Phase 3 日报周报重构, Phase 4 标签历史趋势, Phase 5 相关标签推荐

**Archive:** → [v1.2-ROADMAP.md](./milestones/v1.2-ROADMAP.md), [v1.2-REQUIREMENTS.md](./milestones/v1.2-REQUIREMENTS.md)

---

*Updated: 2026-04-15*
