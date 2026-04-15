# Phase 1: 基础设施与标签收敛 - Context

**Gathered:** 2026-04-13
**Status:** Ready for planning

<domain>
## Phase Boundary

标签系统具备 pgvector 向量搜索能力，新文章入库时语义相近标签自动合并，标签空间不再碎片化。包含：pgvector 迁移、embedding 模型动态配置、阈值可调、findOrCreateTag 集成三级匹配、合并事务迁移引用、中间地带降级创建、旧标签 merged 状态保留。

</domain>

<decisions>
## Implementation Decisions

### 存量 embedding 迁移
- **D-01:** 当前无存量 embedding 数据（功能未生效），不需要数据格式迁移
- **D-02:** 直接建立 pgvector `vector` 列，从零开始生成 embedding
- **D-03:** 旧文章标签支持后续批量重新计算，不阻塞本阶段交付

### 合并历史追溯
- **D-04:** TopicTag 模型新增 `status` 字段，值为 `active` 或 `merged`
- **D-05:** TopicTag 模型新增 `merged_into_id` 指针，指向合并目标标签
- **D-06:** 合并时不建立额外的 merge_events 日志表，简单标记即可
- **D-07:** 合并后的旧标签保留（不物理删除），查询时通过 status 过滤

### 阈值配置
- **D-08:** 建立独立的 `embedding_config` 数据库表，存储 HighSimilarity、LowSimilarity 阈值及其他 embedding 相关配置
- **D-09:** 配置项包含：阈值、模型名、维度等，方便未来扩展
- **D-10:** 后端启动时从表内读取配置，提供 API 端点供修改

### embedding 模型切换
- **D-11:** 模型切换后，现有标签 embedding 标记为过期（stale），不立即重算
- **D-12:** 后台任务异步重算过期 embedding，切换期间相似度匹配降级为创建新标签
- **D-13:** 切换期间不影响标签创建流程，只影响收敛（合并）能力

### the agent's Discretion
- pgvector 列的具体维度和索引参数选择
- 嵌入生成的批处理策略
- 后台重算任务的并发和速率控制
- API 端点的具体请求/响应结构

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 基础设施
- `backend-go/internal/domain/models/topic_graph.go` — TopicTag、TopicTagEmbedding、ArticleTopicTag 模型定义
- `backend-go/internal/domain/topicanalysis/embedding.go` — EmbeddingService、TagMatch 三级匹配逻辑、阈值定义
- `backend-go/internal/platform/airouter/embedding.go` — EmbeddingClient、CosineSimilarity 实现
- `backend-go/internal/platform/airouter/router.go` — ResolvePrimaryProvider、CapabilityEmbedding 路由
- `backend-go/internal/platform/airouter/store.go` — Store 定义、Capability 常量

### 标签收敛
- `backend-go/internal/domain/topicextraction/article_tagger.go` — TagArticle 入口、findOrCreateTag 调用点
- `backend-go/internal/domain/topicextraction/tagger.go` — findOrCreateTag 当前实现（slug+category 精确匹配，无 embedding）

No external specs — requirements fully captured in REQUIREMENTS.md (INFRA-01~03, CONV-01~04).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `topicanalysis.EmbeddingService.TagMatch`: 三级匹配逻辑（exact → alias → embedding）已完整实现，但未连接到 findOrCreateTag
- `topicanalysis.EmbeddingService.FindSimilarTags`: 语义相似度搜索已实现，当前在 Go 侧遍历全表计算余弦距离（待替换为 pgvector SQL）
- `airouter.EmbeddingClient.Embed`: OpenAI 兼容 embedding API 调用已实现
- `airouter.Router.ResolvePrimaryProvider(CapabilityEmbedding)`: provider 路由查找已实现
- `topicanalysis.DefaultThresholds`: HighSimilarity=0.97, LowSimilarity=0.78 默认值已定义

### Established Patterns
- **airouter 模式**: provider 通过数据库配置，Router 解析主 provider，支持 capability 路由
- **findOrCreateTag 模式**: slug+category 查找 → 更新/创建，在 topicextraction 包内
- **GORM 模型**: snake_case JSON tag，tableName() 方法，GORM struct tag
- **事务模式**: database.DB 全局变量直接使用 GORM，简单场景不用 Repository 模式

### Integration Points
- `findOrCreateTag` (tagger.go:101) — 这是收敛集成的关键接入点，需要在此调用 TagMatch
- `TagArticle` (article_tagger.go:20) — 新文章入库时的标签入口，调用 findOrCreateTag
- `TagSummary` (tagger.go:20) — AI summary 标签入口，也调用 findOrCreateTag
- `topic_tag_embeddings` 表 — 需要从 text 列迁移到 pgvector vector 列
- `article_topic_tags` 表 — 合并时需要迁移引用（UPDATE topic_tag_id）

### Key Technical Observations
- `getEmbeddingModel` (embedding.go:320) 硬编码返回 "text-embedding-ada-002"，需从 provider 配置读取
- `FindSimilarTags` (embedding.go:119) 加载全表到 Go 内存计算余弦距离，pgvector SQL 替代后性能大幅提升
- `TopicTag.IsCanonical` 字段已存在但未充分使用，merged 标签可复用此字段语义或新增 status
- `TopicTagEmbedding.Vector` 当前是 `string` (text) 类型，需迁移为 pgvector `vector` 类型

</code_context>

<specifics>
## Specific Ideas

- 目前 embedding 功能未实际生效，数据库中没有存量 embedding 数据，迁移无需考虑格式转换
- 用户倾向独立配置表而非融入 preferences 系统，因为 embedding 相关配置（模型、维度、阈值）是一个独立的配置域
- 模型切换是低频操作，切换期间短暂降级可接受

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 01-infrastructure-tag-convergence*
*Context gathered: 2026-04-13*
