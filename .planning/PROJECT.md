# PROJECT: RSS Reader

## What This Is

RSS Reader 应用，Go 后端 + Nuxt 4 前端，PostgreSQL 存储，单用户部署。核心功能包括 feed 订阅、文章阅读、AI 摘要、Firecrawl 内容增强、日报周报导出。已完成 v1.1 业务漏洞修复。

## Core Value

**通过智能标签系统帮助用户高效消费信息**

标签不是附属功能，是信息组织的核心。语义收敛减少噪音，关注机制聚焦兴趣，历史趋势辅助反思。

## Current Milestone: v1.2 标签智能收敛与关注推送

**Goal:** 通过 embedding 语义相似度自动合并相近标签，建立关注标签机制，重构日报周报为关注视角，增加首页关注文章推送与标签历史趋势分析

**Target features:**
- 标签自动收敛：新文章入库时 embedding 匹配，高相似度自动合并
- Embedding 模型配置：复用已有 airouter provider 框架（CapabilityEmbedding）
- 关注标签：标签列表勾选关注
- 日报周报重构：完全替换为基于关注标签的每日/每周总结，支持手动触发
- 首页关注文章推送：关注标签关联文章，支持按标签筛选
- 关注标签相关度推送：推荐与关注标签高相关的其他标签（embedding 相似或同文章共现）
- 标签历史趋势分析：指定关注标签或手动选标签的历史维度分析

## Key Decisions

| Decision | Reason | Alternatives Considered |
|----------|--------|-------------------------|
| 复用 airouter provider 框架 | CapabilityEmbedding 路由已存在，EmbeddingClient 已实现 | 自建 embedding 模块（重复造轮子） |
| 自动合并标签（非聚类展示） | 减少标签碎片，从根本上简化标签空间 | 展示时聚类（不解决数据冗余） |
| 新文章入库时实时触发收敛 | 及时收敛，避免累积大量重复标签 | 定时批量合并（延迟高） |
| 完全替换日报周报 | 关注标签视角是核心体验，旧逻辑不保留 | 新增视图（两套逻辑维护成本高） |

## Requirements

### Validated

**v1.1 Phase 03 (状态一致性修复):**
- ✓ STAT-01: Feed删除时文章级联删除 — v1.1/Phase 3
- ✓ STAT-02: 文章清理不误删活跃文章 — v1.1/Phase 3
- ✓ STAT-03: Summary-only feed文章summary_status初始化为pending — v1.1/Phase 3
- ✓ STAT-04: 阻塞文章自动恢复机制 — v1.1/Phase 3
- ✓ STAT-05: 阻塞数量超过阈值时WARN告警 — v1.1/Phase 3

### Active (v1.2)

See `.planning/REQUIREMENTS.md` for current milestone requirements.

**Validated in Phase 1 (基础设施与标签收敛):**
- INFRA-01: pgvector 向量列替代 JSON 文本存储
- INFRA-02: Embedding 模型名从 provider 动态读取
- INFRA-03: 相似度阈值可通过 API 配置
- CONV-01: findOrCreateTag 集成 TagMatch 三级匹配
- CONV-02: MergeTags 事务安全合并标签
- CONV-03: 中间地带跳过 AI 判定创建新标签
- CONV-04: 合并标签标记 merged 状态保留历史

**Validated in Phase 8 (标签树图谱增强):**
- TopicTag Description 字段 + LLM 生成标签描述
- 抽象标签 Description 生成（ExtractAbstractTag 扩展）
- 后端时间筛选 API + 前端时间筛选 UI
- 图谱抽象标签可视化（3D 发光效果）+ 点击详情面板
- TagMergePreview 迁移至设置页 + 合并后重建提示
- 标签树节点手动归类（后端 API + 前端弹窗）

### Out of Scope

| Requirement | Reason |
|-------------|--------|
| 全量标签聚类展示 | 本次目标是自动合并，不是展示分组 |
| 多用户系统 | 单用户部署，不需要 |
| 标签手动合并 UI | 本次用自动合并，手动合并可后续迭代 |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition:**
1. Requirements invalidated → Move to Out of Scope with reason
2. Requirements validated → Move to Validated with phase reference
3. New requirements emerged → Add to Active
4. Decisions to log → Add to Key Decisions

**After each milestone:**
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

## Context

**Codebase:** Go + Nuxt 4, PostgreSQL, 单用户部署

**已有的 embedding 基础设施:**
- pgvector `vector(1536)` 列 + HNSW 索引，SQL `<=>` 余弦距离搜索
- `topicanalysis.EmbeddingService` — embedding 生成 + 相似度匹配 + TagMatch 三级逻辑
- `topicanalysis.EmbeddingConfigService` — 配置 CRUD，阈值/模型可 API 调整
- `airouter.EmbeddingClient` — OpenAI 兼容 embedding API 调用
- `airouter.Store` — provider/route 管理，支持 CapabilityEmbedding
- `topic_tag_embeddings` 表 — pgvector 向量列 + 遗留 JSON 字段双写
- 三级匹配阈值: HighSimilarity ≥ 0.97 自动复用, LowSimilarity < 0.78 新建, 中间地带创建新标签
- 标签合并: MergeTags 事务迁移 article_topic_tags + ai_summary_topics，merged 状态保留

**现有标签系统:**
- `topic_tags` 表 (slug, label, aliases, category)
- `topicextraction` 包: TagArticle, RetagArticle, TagQueue, TagJobQueue
- `topicgraph` 包: BuildTopicGraph
- v1.1 已修复标签流程统一（全部走 TagJobQueue）

**现有日报周报:**
- `digest` 包: DigestConfig, scheduler (daily/weekly), Obsidian 导出
- 前端 `digest.ts` API + 对应页面

---

*Last updated: 2026-04-14 (Phase 8 complete — v1.2 milestone finished)*
