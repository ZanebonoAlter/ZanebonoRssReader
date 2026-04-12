# REQUIREMENTS: Milestone v1.2 标签智能收敛与关注推送

**Defined:** 2026-04-12
**Core Value:** 通过智能标签系统帮助用户高效消费信息

## v1.2 Requirements

### INFRA — 基础设施

- [ ] **INFRA-01**: topic_tag_embeddings 向量存储从 JSON text 迁移到 pgvector vector 列，相似度搜索用 SQL `<=>` 运算符替代 Go 侧循环计算
- [ ] **INFRA-02**: getEmbeddingModel 从 provider 配置动态读取 embedding 模型名，不硬编码 ada-002
- [ ] **INFRA-03**: 收敛阈值（HighSimilarity/LowSimilarity）做成可配置项，支持通过 API 或数据库配置

### CONV — 标签自动收敛

- [ ] **CONV-01**: 新文章入库调用 findOrCreateTag 时，集成 EmbeddingService.TagMatch 做三级匹配（exact → alias → embedding），高相似度自动复用已有标签
- [ ] **CONV-02**: 标签合并时在事务内迁移 article_topic_tags 等关联记录到目标标签，防止引用悬空
- [ ] **CONV-03**: AI judgment 中间地带（LowSimilarity ~ HighSimilarity）跳过 AI 判定，降级为创建新标签；中间地带阈值可调整
- [ ] **CONV-04**: 合并后的旧标签标记为 merged 状态（非物理删除），保留合并历史可追溯

### WATCH — 关注标签

- [ ] **WATCH-01**: 用户可在标签列表页面勾选/取消关注标签（is_watched 开关）
- [ ] **WATCH-02**: 后端提供关注标签 CRUD API：列出关注标签、设置关注、取消关注
- [ ] **WATCH-03**: 关注标签变更时记录 watched_at 时间，用于日报周报的时间范围判断

### FEED — 首页关注文章推送

- [ ] **FEED-01**: 首页展示关注标签关联的文章流，按时间倒序排列
- [ ] **FEED-02**: 支持按单个关注标签筛选文章
- [ ] **FEED-03**: 文章列表支持按相关度排序（关注标签匹配数量、embedding 距离加权）

### DIGEST — 日报周报重构

- [ ] **DIGEST-01**: 完全替换现有日报/周报逻辑，从按分类聚合改为按关注标签聚合文章
- [ ] **DIGEST-02**: 日报/周报支持手动触发（不限于定时任务）
- [ ] **DIGEST-03**: 日报/周报适配所有导出通道：前端展示、飞书、Obsidian、OpenNotebook
- [ ] **DIGEST-04**: 无关注标签时有合理的降级提示（而非空白或报错）

### TRENDS — 标签历史分析

- [ ] **TRENDS-01**: 用户可指定关注标签或手动选择标签，生成该标签的主题叙事总结（AI 生成）
- [ ] **TRENDS-02**: 主题叙事包含：事件来龙去脉、人物/实体时间线、综合评价总结
- [ ] **TRENDS-03**: 支持选择时间范围限定叙事内容范围

### REC — 相关标签推荐

- [ ] **REC-01**: 基于关注标签推荐相关标签，综合 embedding 相似度和同文章共现频次
- [ ] **REC-02**: 推荐结果在关注标签管理页面或标签详情页展示

## Future Requirements

### Deferred

- **AI judgment 中间地带处理**: 未来可实现 AI 辅助判定 0.78-0.97 相似度的标签是否合并
- **标签手动合并 UI**: 管理界面支持用户主动选择合并目标
- **趋势可视化图表**: 文章数量、时间分布的图表化展示
- **多标签对比分析**: 同时对比多个标签的历史叙事

## Out of Scope

| Feature | Reason |
|---------|--------|
| 全量标签聚类展示 | 本次目标是自动合并，不是展示分组 |
| 多用户/协作系统 | 单用户部署 |
| 标签自动删除 | 只做合并，不做删除 |
| 自定义 embedding 模型训练 | 使用现成 API 即可 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| INFRA-01 | — | Pending |
| INFRA-02 | — | Pending |
| INFRA-03 | — | Pending |
| CONV-01 | — | Pending |
| CONV-02 | — | Pending |
| CONV-03 | — | Pending |
| CONV-04 | — | Pending |
| WATCH-01 | — | Pending |
| WATCH-02 | — | Pending |
| WATCH-03 | — | Pending |
| FEED-01 | — | Pending |
| FEED-02 | — | Pending |
| FEED-03 | — | Pending |
| DIGEST-01 | — | Pending |
| DIGEST-02 | — | Pending |
| DIGEST-03 | — | Pending |
| DIGEST-04 | — | Pending |
| TRENDS-01 | — | Pending |
| TRENDS-02 | — | Pending |
| TRENDS-03 | — | Pending |
| REC-01 | — | Pending |
| REC-02 | — | Pending |

**Coverage:**
- v1.2 requirements: 22 total
- Mapped to phases: 0 ⚠️
- Unmapped: 22 ⚠️

---
*Requirements defined: 2026-04-12*
*Last updated: 2026-04-12 after initial definition*
