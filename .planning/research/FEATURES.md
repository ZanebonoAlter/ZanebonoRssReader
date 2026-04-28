# Feature Landscape

**Domain:** RSS Reader — 标签智能收敛与关注推送 (v1.2)
**Researched:** 2026-04-12
**Confidence:** HIGH (based on deep codebase analysis + domain expertise)

## Table Stakes

Features users expect from a tag-intelligent RSS reader. Missing = product feels incomplete.

| Feature | Why Expected | Complexity | Dependencies | Notes |
|---------|--------------|------------|--------------|-------|
| **标签自动收敛** | 标签碎片化是 AI 提取标签的通病。用户看到的应该是 "AI" 而不是 "Artificial Intelligence"、"artificial-intelligence"、"AI技术" 三个标签 | **High** | 现有 `EmbeddingService.TagMatch` 三级逻辑、`airouter.EmbeddingClient`、`topic_tag_embeddings` 表 | 核心改动在 `findOrCreateTag()` 中，将现有的 slug-only 匹配升级为 embedding 相似度匹配。需扩展 `TagMatch` 以支持自动合并（当前只做复用/创建判定） |
| **关注标签勾选** | 信息过载是 RSS 阅读器的核心痛点，"关注"是标准过滤机制（Feedly Boards、Inoreader Rules、Readwise Reader Folders） | **Low** | `topic_tags` 表需新增 `is_watched` 字段 | 单用户应用，布尔字段足够。不需要关注时间线、关注来源追踪等多用户场景的复杂设计 |
| **关注文章推送** | 关注了标签却看不到关联文章 = 功能没有闭环 | **Medium** | 关注标签勾选、`article_topic_tags` / `ai_summary_topics` 关联表 | 需新的 API 端点按 watched tag 聚合文章，前端需新的首页 feed 区域 |
| **日报周报关注视角** | 已有日报周报功能但按分类组织，缺少用户个性化视角 | **Medium** | 关注标签勾选、现有 `digest` 包、现有前端 `DigestListView.vue` | PROJECT.md 决策是"完全替换"旧逻辑，不是新增视图。需重构 `DigestGenerator` 从分类维度改为标签维度 |

## Differentiators

Features that set product apart. Not expected, but valued.

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| **相关标签推荐** | 帮用户发现感兴趣的关联话题，扩大关注范围而不增加噪音 | **Medium** | 现有 `fetchCoOccurrence` 和 `fetchRelatedTopicLabels` 逻辑、`EmbeddingService.FindSimilarTags` | 两种信号源：(1) 同文章共现（已有 `fetchCoOccurrence`），(2) embedding 语义相似（已有 `FindSimilarTags`）。推荐算法可从共现频次加权开始，后续叠加 embedding 相似度 |
| **标签历史趋势分析** | 让用户感知"这个话题最近变热了/冷了"，是信息消费的 meta 认知工具 | **High** | `article_topic_tags` 按时间分桶、关注标签、现有 `buildTrendData` 逻辑 | 现有 `buildTrendData` 已做按天计数的简单趋势。需要扩展为：(1) 可指定时间范围（7天/30天/90天），(2) 多标签对比，(3) 趋势方向判定（上升/下降/稳定）。PostgreSQL 时序查询可直接用 `date_trunc` |
| **Embedding 模型可配置** | 不同 embedding 模型（text-embedding-ada-002 vs text-embedding-3-small）对中文标签的区分度不同，用户应能切换 | **Low** | 现有 `airouter.Store` provider 框架、`CapabilityEmbedding` 路由 | 框架已存在（`airouter.Router.ResolvePrimaryProvider(CapabilityEmbedding)`），只需在 UI 上暴露配置入口 |

## Anti-Features

Features to explicitly NOT build.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **标签聚类展示（分组不合并）** | 不解决标签碎片化根本问题，只是视觉上分组，数据库里仍然是 N 个标签 | 自动合并到 canonical tag，从数据层面消除碎片 |
| **手动标签合并 UI** | 增加用户负担，且已有自动合并机制。本次聚焦自动化 | 自动收敛 + 收敛日志（可后续迭代添加手动复核） |
| **多用户关注/分享标签** | 单用户应用，不需要社交特性 | 简单布尔标记 `is_watched` |
| **标签别名管理界面** | 维护成本高，用户不关心别名映射 | 自动合并时维护 aliases 字段，用户无需感知 |
| **实时标签推送通知** | RSS 阅读器不是聊天工具，文章入库到用户查看有天然延迟 | 日报/周报关注视角推送 + 首页关注 feed |
| **机器学习预测趋势** | 过度工程，单用户场景数据量不足以训练可靠模型 | 简单的计数时序 + 移动平均即可满足"这个话题热不热"的需求 |

## Feature Dependencies

```
标签自动收敛 ──────────────────────────────────────────┐
   │                                                   │
   │ (收敛后的标签空间更干净，下游功能才有意义)            │
   ▼                                                   │
关注标签勾选 ──► 关注文章推送 (首页 feed)                │
   │                                                   │
   ├──► 日报周报关注视角重构                             │
   │                                                   │
   ├──► 相关标签推荐                                    │
   │                                                   │
   └──► 标签历史趋势分析                                │
                                                       │
Embedding 模型可配置 ──► (收敛质量的基础设施)            │
                                                       │
(所有关注相关功能都依赖收敛后的干净标签空间) ◄────────────┘
```

### Critical Path

1. **Embedding 模型可配置** — 基础设施，可快速完成
2. **标签自动收敛** — 核心依赖，必须先于所有关注功能
3. **关注标签勾选** — 基础数据层，Low 复杂度但下游阻塞
4. **关注文章推送** — 关注功能的第一用户价值
5. **日报周报关注视角重构** — 中等复杂度，依赖关注数据
6. **相关标签推荐** — 锦上添花，可最后
7. **标签历史趋势分析** — 独立性最强，可并行

## Existing Codebase Leverage

| Feature | Existing Infrastructure | Gap to Fill |
|---------|------------------------|-------------|
| 标签自动收敛 | `EmbeddingService.TagMatch()` 三级匹配（exact → alias → embedding）已有完整实现 | `findOrCreateTag()` 目前只做 slug 匹配，需改为调用 `TagMatch`；高相似度时自动合并 aliases 而非仅复用 |
| 关注标签勾选 | `topic_tags` 表结构完整，GORM 迁移简单 | 新增 `is_watched bool` 字段 + 列表 API 返回 watched 状态 |
| 关注文章推送 | `article_topic_tags` 表 + `GetArticlesByTag()` 已实现按标签查文章 | 需新增"按 watched tags 批量查询"API，前端需新的 feed 组件 |
| 日报周报重构 | `digest` 包完整（Generator、Scheduler、Obsidian、飞书） | `DigestGenerator` 从按分类分组改为按 watched tag 分组；保持 Obsidian/飞书导出兼容 |
| 相关标签推荐 | `fetchCoOccurrence()` + `fetchRelatedTopicLabels()` + `FindSimilarTags()` 已实现 | 组合两种信号源的推荐逻辑，按 watched tag 触发 |
| 标签趋势分析 | `buildTrendData()` 已有按天计数逻辑 | 扩展时间范围、多标签对比、趋势方向判定 |

## MVP Recommendation

Prioritize (first 3 phases):

1. **Embedding 模型可配置** — Quick win，解锁收敛质量
2. **标签自动收敛** — 核心价值，消除碎片
3. **关注标签勾选** — Low effort，High dependency value

Defer:
- **标签历史趋势分析**: High complexity，独立性强，可在核心关注功能稳定后单独迭代
- **相关标签推荐**: 需要一定量的 watched tags 数据才有推荐价值，过早上线推荐结果稀疏

## Complexity Assessment Summary

| Feature | Backend | Frontend | Data Model | Overall |
|---------|---------|----------|------------|---------|
| 标签自动收敛 | High (修改核心 `findOrCreateTag` 流程) | Low (对用户透明) | Medium (aliases 维护) | **High** |
| 关注标签勾选 | Low (CRUD + boolean field) | Medium (标签列表勾选 UI) | Low (一字段) | **Low** |
| 关注文章推送 | Medium (新 API + 聚合查询) | High (首页新 feed 区域 + 筛选) | Low | **Medium** |
| 日报周报重构 | Medium (重构分组逻辑) | Medium (保持现有 UI 结构，换数据源) | Low | **Medium** |
| 相关标签推荐 | Low (组合已有逻辑) | Medium (推荐展示 UI) | Low | **Medium** |
| 标签趋势分析 | Medium (时序聚合查询) | High (趋势图表组件) | Low | **High** |
| Embedding 可配置 | Low (框架已存在) | Low (设置面板暴露) | None | **Low** |

## Sources

- Codebase analysis: `topicanalysis/embedding.go` (TagMatch 三级逻辑), `topicextraction/tagger.go` (findOrCreateTag), `digest/generator.go` (按分类分组), `topicanalysis/analysis_service.go` (co-occurrence + trend data)
- Feedly topic tracking patterns: docs.feedly.com
- Entity resolution for tag matching: semantic ER patterns from arxiv/html/2506.02509v1
- Trend analysis methods: Stack Overflow z-score approach (stackoverflow.com/questions/787496)
- PROJECT.md v1.2 decisions: 完全替换日报周报、自动合并（非聚类）、实时触发收敛
