# PROJECT: RSS Reader

## What This Is

RSS Reader 应用，Go 后端 + Nuxt 4 前端，PostgreSQL 存储，单用户部署。核心功能包括 feed 订阅、文章阅读、AI 摘要、Firecrawl 内容增强、智能标签系统（embedding 语义收敛、关注推送、抽象标签层级）、日报周报导出。

## Core Value

**通过智能标签系统帮助用户高效消费信息**

标签是信息组织的核心。语义收敛减少噪音，关注机制聚焦兴趣，抽象标签层级帮助从碎片走向结构。

## Current State: v1.2 Shipped (2026-04-15)

**已完成:**
- pgvector 向量搜索基础设施 + 三级标签匹配 + 自动收敛
- 关注标签 CRUD + 首页关注文章推送（心形图标、侧边栏、相关度排序）
- 标签合并交互界面（扫描预览、自定义名称、批量操作）
- 抽象标签提取 + 递归层级树
- 标签树增强：Description 生成、时间筛选、图谱发光可视化、节点手动归类
- 标签 quality_score 方案
- 后端日志分流（info → stdout, error → stderr）

**跳过 (设计方向变更):**
- Phase 3: 日报周报重构为关注标签视角
- Phase 4: 标签历史趋势分析
- Phase 5: 相关标签推荐

## Key Decisions

| Decision | Reason | Outcome |
|----------|--------|---------|
| 复用 airouter provider 框架 | CapabilityEmbedding 路由已存在 | ✓ 避免重复造轮子 |
| 自动合并标签（非聚类展示） | 减少标签碎片 | ✓ 标签空间大幅收敛 |
| 新文章入库时实时触发收敛 | 及时收敛 | ✓ 无累积 |
| 完全替换日报周报 | 关注标签视角是核心 | ⚠️ 因设计方向变更跳过 |
| Middle band 跳过 AI 判定 | 简化流程，后续由抽象标签处理 | ✓ |
| Phase 3/4/5 跳过 | 设计方向变更 | — 待新里程碑定义 |

## Requirements

### Validated

**v1.1:**
- ✓ STAT-01~05: 状态一致性修复 — v1.1

**v1.2:**
- ✓ INFRA-01~03: pgvector 基础设施 + 动态 embedding 配置 — v1.2/Phase 1
- ✓ CONV-01~04: 三级标签匹配 + 自动收敛 + 合并事务 — v1.2/Phase 1
- ✓ WATCH-01~03: 关注标签 CRUD — v1.2/Phase 2
- ✓ FEED-01~03: 首页关注文章推送 — v1.2/Phase 2
- ✓ NEW-01~08: 抽象标签 + 标签树增强 + quality_score — v1.2/Phase 7, 8

### Active

(None — awaiting next milestone definition via `/gsd-new-milestone`)

### Out of Scope

| Requirement | Reason |
|-------------|--------|
| 全量标签聚类展示 | 自动合并已解决碎片问题 |
| 多用户系统 | 单用户部署 |
| 标签自动删除 | 只做合并，不做删除 |

## Evolution

This document evolves at phase transitions and milestone boundaries.

## Context

**Codebase:** Go + Nuxt 4, PostgreSQL + pgvector, 单用户部署
**Timeline:** v1.1 (2026-04-11) → v1.2 (2026-04-15)
**LOC:** ~31K lines added in v1.2

---
*Last updated: 2026-04-15 after v1.2 milestone completion*
