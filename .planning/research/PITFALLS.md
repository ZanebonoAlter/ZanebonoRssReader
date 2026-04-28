# Domain Pitfalls

**Domain:** 标签智能收敛与关注推送 (v1.2 milestone for RSS Reader)
**Researched:** 2026-04-12
**Confidence:** HIGH (codebase-specific analysis from reading existing embedding/tag/digest infrastructure)

---

## Critical Pitfalls

Mistakes that cause rewrites or major issues.

### Pitfall 1: 标签合并级联 — 收敛后旧标签的引用悬空

**What goes wrong:** 收敛时将标签 A 合并到标签 B，但 `article_topic_tags`、`ai_summary_topics` 中的 `topic_tag_id` 仍指向被合并掉的标签 A。后续查询关注标签列表、首页文章推送、日报周报生成都无法找到这些关联文章。

**Why it happens:** 现有 `findOrCreateTag`（`tagger.go:101`）只做 slug 匹配创建，不做合并迁移。`EmbeddingService.TagMatch`（`embedding.go:180`）返回 `ShouldCreate: false` 和 `ExistingTag`，但调用方（tagger）拿到后直接关联新 tag，不会迁移旧的关联记录。

**Consequences:**
- 用户关注 "Kubernetes" 但 "k8s" 已合并到 "Kubernetes"，旧文章仍关联 "k8s" 的 tag ID → 关注推送漏掉这些文章
- 日报周报中关注标签统计不完整
- 标签趋势分析漏掉历史数据

**Prevention:**
1. 合并操作必须迁移所有 `article_topic_tags` 和 `ai_summary_topics` 中旧标签的 `topic_tag_id` 到新标签
2. 在同一个数据库事务中完成迁移+删除旧标签
3. 维护 `aliases` 字段记录被合并的标签原名，便于反向查询
4. `findOrCreateTag` 应改为先调 `EmbeddingService.TagMatch`，在匹配到已有标签时做合并迁移而非仅跳过

**Detection:** 合并后检查被合并标签是否仍有 `article_topic_tags` 引用；单元测试覆盖合并场景。

**Phase mapping:** 标签收敛核心 Phase — 必须在收敛逻辑实现时同步实现。

---

### Pitfall 2: Embedding 模型切换导致相似度阈值失效

**What goes wrong:** 现有 `DefaultThresholds`（`embedding.go:33`）硬编码 `HighSimilarity: 0.97`、`LowSimilarity: 0.78`，这些阈值是为 `text-embedding-ada-002` 校准的。如果用户切换到 `text-embedding-3-small` 或其他模型（如 BGE、Cohere），同一对标签的 cosine similarity 会显著不同。`text-embedding-3-small` 的相似度分数整体偏低，导致大量标签无法匹配高阈值而误创建新标签。

**Why it happens:** OpenAI 第三代 embedding 模型的 cosine similarity 分布与 ada-002 不同。社区报告 `text-embedding-3-small` 在相同文档上的相似度分数普遍低于 ada-002（参见 OpenAI community 讨论 #873048）。阈值写死在 Go 常量中，不随模型配置变化。

**Consequences:**
- 模型切换后标签收敛功能完全失效 — 所有标签都被判定为"不相似"而创建新标签
- 已有 embedding 数据与新模型不兼容但 `TextHash` 检测不到模型变化
- 每次切换模型都需要手动重新校准阈值

**Prevention:**
1. 阈值应存储在数据库中，按 embedding 模型版本关联
2. `getEmbeddingModel`（`embedding.go:320`）目前硬编码返回 `text-embedding-ada-002`，必须从 provider 配置中读取实际模型
3. 当检测到 embedding 模型变化时，标记所有已有 embedding 为 stale 并触发重新生成
4. 提供阈值校准工具：给定一组已知相似/不相似的标签对，计算最佳阈值

**Detection:** 切换模型后监控新标签创建率；如果创建率飙升则阈值可能失效。

**Phase mapping:** 标签收敛 Phase 或 Embedding 配置 Phase — 必须在收敛逻辑之前解决模型配置读取问题。

---

### Pitfall 3: Embedding 存储为 JSON text 而非 pgvector — 性能瓶颈

**What goes wrong:** 现有 `TopicTagEmbedding.Vector` 字段（`topic_graph.go:72`）是 `type:text`，存储 JSON 序列化的 `[]float64`。`FindSimilarTags`（`embedding.go:119`）加载所有同 category 的 embedding 到 Go 内存做 cosine similarity 计算。当标签数量增长到数千级别时，每次新文章入库都需加载全部向量做全表扫描。

**Why it happens:** 项目已有 pgvector 扩展（README 明确提到 `pgvector/pgvector:pg18-trixie` Docker 镜像），但 embedding 存储尚未迁移到 `vecotr` 类型（注意模型注释 "Legacy JSON text payload kept until the staged pgvector runtime cutover lands"）。

**Consequences:**
- 标签收敛变成 O(N) 全表扫描，N = 同 category 标签数 × 向量维度
- 新文章入库时阻塞，因为 tag 流程在 `TagJobQueue` worker 中同步执行
- 随着标签增长，embedding API 调用 + 全量向量比较的时间不可接受

**Prevention:**
1. 在标签收敛 Phase 之前或同步完成 pgvector 迁移：`Vector` 字段改为 `vector(1536)` 类型
2. 使用 pgvector 的 `<=>` 操作符做近似最近邻搜索（ANN），代替全量加载
3. 添加 `(category, vector)` 复合索引或分区
4. 如果不迁移 pgvector，至少在 Go 侧做相似度缓存，避免每篇文章都重新计算

**Detection:** 当标签数量超过 500 时，测量收敛耗时；如果 > 2 秒则需优化。

**Phase mapping:** 标签收敛 Phase — 收敛的实时性要求决定了必须在此 Phase 解决向量搜索性能。

---

### Pitfall 4: 日报周报替换破坏现有导出通道

**What goes wrong:** PROJECT.md 决定"完全替换日报周报"，但现有 digest 系统有多个导出通道：飞书推送（`feishu.go`）、Obsidian 导出（`obsidian.go`）、Open Notebook（`scheduler.go:197`）。如果仅替换 `DigestGenerator` 的数据源（从按分类改为按关注标签），但不更新这些导出通道的格式，会导致：
- 飞书卡片格式错乱（原格式按 `CategoryDigest` 渲染）
- Obsidian markdown 模板不匹配新数据结构
- Open Notebook 发送的数据缺少关注标签维度

**Why it happens:** `DigestScheduler` 直接依赖 `CategoryDigest` 结构体（`generator.go:21`），所有导出方法（`sendFeishuDigest`、`exportToObsidian`、`autoSendToOpenNotebook`）都假定数据是分类维度的。重构数据源但不重构导出 = 编译通过但运行时格式错误。

**Consequences:**
- 用户已配置的飞书/Obsidian 推送突然输出乱格式内容
- 前端 digest 页面无数据显示（API 响应格式变了但前端未同步更新）

**Prevention:**
1. 定义新的 `WatchedTagDigest` 结构体，同时保留 `CategoryDigest` 作为 fallback
2. 每个导出通道单独适配新格式，逐一测试
3. 在数据库中增加配置项让用户选择 digest 模式（分类 / 关注标签），平滑过渡
4. 前后端同步修改：后端 API 响应格式变更时，前端 digest 页面必须同步更新

**Detection:** 替换后运行每个导出通道的集成测试。

**Phase mapping:** 日报周报重构 Phase — 必须逐一验证所有导出通道。

---

### Pitfall 5: 关注标签首页推送 — 无限加载与冷启动

**What goes wrong:** 首页按关注标签过滤文章，如果用户没有关注任何标签（新用户/冷启动），首页为空，体验极差。如果用户关注了大量标签（20+），关联文章太多，需要分页但分页策略可能按标签分（同一文章多标签重复出现）或按文章分（复杂 JOIN）。

**Why it happens:** 现有 `GetArticlesByTag`（`article_tagger.go:270`）只能按单个 slug 查询，不支持多标签 OR 查询。首页需要的是"关注标签中任意一个关联的文章"，需要新的查询逻辑。

**Consequences:**
- 冷启动用户看到空白首页 → 流失
- 多标签分页时同一文章因多标签关联而重复出现
- 大量 JOIN 查询（articles × article_topic_tags × topic_tags × watched_tags）性能差

**Prevention:**
1. 冷启动方案：未关注标签时回退到现有首页逻辑（按时间线展示全部文章）
2. 多标签查询用 `IN` 子查询而非多次单标签查询合并
3. 文章去重：以 article ID 为分页游标，而非 tag+article 联合游标
4. 添加 Redis 缓存或 PostgreSQL 物化视图缓存关注标签的文章列表

**Detection:** 首页加载性能监控；关注 0 个标签和 30+ 标签两种边界测试。

**Phase mapping:** 首页关注推送 Phase — 查询逻辑设计必须在实现前明确冷启动和分页策略。

---

## Moderate Pitfalls

### Pitfall 6: 标签趋势分析的时间窗口与已有 AnalysisService 冲突

**What goes wrong:** 现有 `AnalysisService`（`analysis_service.go`）按 tag+analysisType+windowType+anchorDate 组合缓存分析结果，使用 `TopicAnalysisCursor` 做增量更新。新的标签趋势分析如果复用这套机制但不遵守 cursor 约定（只查新 summary），会导致趋势数据不完整或重复计数。

**Prevention:** 标签趋势分析应直接查 `article_topic_tags` 按时间聚合，而非走 `AnalysisService` 的 summary 聚合路径。趋势分析的数据源是文章标签关联，不是 AI summary。

**Phase mapping:** 趋势分析 Phase — 明确数据源选择。

---

### Pitfall 7: 相关标签推荐 — 共现统计的偏差

**What goes wrong:** 共现推荐（`fetchCoOccurrence`，`analysis_service.go:555`）用 `COUNT(*)` 做归一化。高频标签（如"AI"出现在 50% 的文章中）会与所有标签共现，推荐结果全是高频标签，无信息量。

**Prevention:**
1. 使用 PMI（Pointwise Mutual Information）或 TF-IDF 风格的权重，降低高频标签的推荐分
2. 结合 embedding 相似度（已有 `FindSimilarTags`）和共现频率做加权排序
3. 过滤掉共现次数低于阈值的噪声标签

**Phase mapping:** 相关标签推荐 Phase — 推荐算法必须在实现时考虑频率偏差。

---

### Pitfall 8: Embedding 调用频率控制 — 新文章入库时的 API 限流

**What goes wrong:** 标签收敛在新文章入库时实时触发。如果一次 RSS 刷新拉取 100 篇文章，每篇文章平均 5 个标签，每个新标签需要 1 次 embedding API 调用（`GenerateEmbedding`）+ 1 次全量相似度比较，可能触发 API 速率限制。

**Why it happens:** `TagJobQueue` worker 串行处理标签任务，但每篇文章的每个标签都独立调用 embedding API。没有批量 embedding 或缓存机制。

**Prevention:**
1. 使用 OpenAI embedding API 的批量模式（单次请求多个文本），减少 API 调用次数
2. 缓存已计算过的标签 embedding（查 `topic_tag_embeddings` 表已有记录）
3. 对于已存在的 slug 精确匹配，直接跳过 embedding 计算
4. 添加 embedding API 调用的 rate limiter

**Detection:** 监控 `ai_call_logs` 中 embedding 类型的调用频率和错误率。

**Phase mapping:** 标签收敛 Phase — 收敛逻辑实现时必须处理 API 限流。

---

### Pitfall 9: 收敛方向选择错误 — 保留哪个标签

**What goes wrong:** 当两个标签合并时（如 "GPT-4" 和 "gpt4"），保留哪个？现有逻辑通过 slug 精确匹配找已有标签（`tagger.go:108`），先创建的标签被保留。但如果先创建的是 "gpt4"（来自 heuristic），后创建的是 "GPT-4"（来自 LLM），保留 heuristic 标签可能不如保留 LLM 标签准确。

**Prevention:**
1. 合并时保留 `source = "llm"` 的标签（更准确），将 heuristic 标签的 label 和 aliases 合并进来
2. 或保留 `is_canonical = true` 的标签
3. 合并后更新 `aliases` 字段记录被合并标签的原始名称

**Phase mapping:** 标签收敛 Phase — 合并策略在收敛逻辑中明确。

---

### Pitfall 10: 关注标签的数据模型 — 单用户简化过度

**What goes wrong:** 假设单用户就直接用内存或简单标记（如 `TopicTag.IsWatched` 字段），后续如果要支持多用户或导入导出关注列表就会需要数据库迁移。

**Prevention:** 即使单用户，也应创建 `watched_tags` 关联表（`tag_id`, `watched_at`, `priority`），而非在 `topic_tags` 表加 `is_watched` 字段。理由：
1. 关注是用户行为，不是标签固有属性
2. `watched_at` 支持按关注时间排序和趋势分析起点
3. `priority` 支持用户自定义排序权重
4. 未来多用户时只需加 `user_id` 字段

**Phase mapping:** 关注标签 Phase — 数据模型设计时独立建表。

---

### Pitfall 11: 日报周报的时间窗口边界 — CST 时区处理不一致

**What goes wrong:** 现有 digest 用 `digestCST = time.FixedZone("CST", 8*3600)`（`generator.go:10`），topicgraph 用 `topictypes.TopicGraphCST`。如果新的关注标签日报使用不同的时区变量或 `time.Now()` 不带时区，会导致跨日期的文章漏算或重复计算。

**Prevention:** 统一使用一个时区常量（如 `topictypes.TopicGraphCST`），所有涉及时间窗口计算的代码都引用同一个变量。新 Phase 的代码不应定义新的时区变量。

**Phase mapping:** 日报周报重构 Phase — 检查所有时间窗口计算。

---

### Pitfall 12: 前后端 ID 类型映射 — 关注标签 API 的 string/uint 混乱

**What goes wrong:** AGENTS.md 明确要求 "Convert backend numeric IDs to frontend strings at the API/store boundary"。但关注标签功能涉及大量 tag ID 传递（关注/取消关注、过滤查询、趋势分析）。如果前端传 string ID 而后端期望 uint，或 JSON 序列化丢失精度（JavaScript number 精度问题 > 2^53），会导致关注操作静默失败。

**Prevention:**
1. 后端 API 统一将 tag ID 序列化为 string（使用 `json:"id"` 自定义 marshaler）
2. 前端 `useApiStore` 的 ID 映射逻辑必须在关注标签 API 中同步处理
3. 关注标签的 API 请求/响应中明确 ID 类型，用 TypeScript 类型约束

**Phase mapping:** 贯穿所有 Phase — 每次 API 变更都需检查。

---

## Minor Pitfalls

### Pitfall 13: 标签列表页性能 — 大量标签的分页与搜索

**What goes wrong:** 关注标签选择 UI 需要展示所有可用标签供用户勾选。如果直接加载全部 `topic_tags`，标签数量大时（1000+）前端渲染卡顿。

**Prevention:** 后端标签列表 API 支持分页 + 搜索（按 label 模糊匹配），前端用虚拟滚动或分页加载。

**Phase mapping:** 关注标签 Phase — 前端 UI 实现时处理。

---

### Pitfall 14: 关注标签与已有 `is_canonical` 字段混淆

**What goes wrong:** `TopicTag` 有 `IsCanonical` 字段（`topic_graph.go:49`），但这是标记"是否为合并后的标准标签"，不是"用户是否关注"。新增关注功能时可能误用此字段。

**Prevention:** 关注功能必须用独立的 `watched_tags` 表或字段，不与 `is_canonical` 混用。文档和代码注释明确区分两者含义。

**Phase mapping:** 关注标签 Phase — 数据模型设计时明确字段语义。

---

### Pitfall 15: 标签趋势图的前端图表渲染 — 大量数据点

**What goes wrong:** 标签历史趋势分析如果按天统计且时间范围长（3 个月 = 90 个数据点），前端图表渲染可能卡顿，尤其是多个标签叠加对比时。

**Prevention:**
1. 后端做数据聚合（周/月粒度），不返回原始日数据
2. 前端使用轻量图表库（如 lightweight-charts），不用重量级库
3. 限制同时对比的标签数量（如最多 5 个）

**Phase mapping:** 趋势分析 Phase — 前端图表实现时处理。

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| 标签收敛（embedding 匹配 + 合并） | 合并后引用悬空 (Pitfall 1) | 事务内迁移所有关联记录 |
| 标签收敛（embedding 匹配 + 合并） | 模型切换阈值失效 (Pitfall 2) | 阈值按模型存储，检测模型变化 |
| 标签收敛（embedding 匹配 + 合并） | 向量搜索性能 (Pitfall 3) | 迁移 pgvector 或缓存相似度 |
| 标签收敛（embedding 匹配 + 合并） | API 限流 (Pitfall 8) | 批量 embedding + 缓存 |
| 关注标签（数据模型 + UI） | 单用户过度简化 (Pitfall 10) | 独立 watched_tags 表 |
| 关注标签（数据模型 + UI） | 前后端 ID 映射 (Pitfall 12) | 统一 string ID 策略 |
| 日报周报重构 | 导出通道破坏 (Pitfall 4) | 逐一适配每个导出通道 |
| 日报周报重构 | 时区不一致 (Pitfall 11) | 统一时区常量 |
| 首页关注推送 | 冷启动空白 (Pitfall 5) | 未关注时回退到全量首页 |
| 首页关注推送 | 分页重复 (Pitfall 5) | 以 article ID 为唯一游标 |
| 相关标签推荐 | 高频标签偏差 (Pitfall 7) | PMI/TF-IDF 加权 |
| 趋势分析 | 数据源选错 (Pitfall 6) | 用 article_topic_tags 而非 summary |
| 趋势分析 | 图表性能 (Pitfall 15) | 后端聚合 + 限制对比数量 |

## Integration-Level Warnings

### 标签收敛 × TagJobQueue 交互

现有 `TagJobQueue`（`tag_job_queue.go`）是串行 worker，每篇文章一个 job。标签收敛如果嵌入 tag 流程中，收敛逻辑的耗时（embedding API 调用 + 全量相似度比较）会拖慢整个 tag 队列处理速度。

**建议:** 收敛逻辑异步化 — tag worker 只做 tag 创建/关联，收敛逻辑作为独立的后台任务（或 `TagJobQueue` 的第二个阶段）。

### 关注标签 × 前端 Store

现有 `useApiStore` 是前端主数据源。关注标签数据如果放入单独的 store，需要在首页文章推送时跨 store 查询（关注标签 store → 文章过滤）。建议关注标签列表直接放入 `useApiStore` 的扩展字段中，保持单一数据源。

### 日报周报 × 定时调度器

现有 `DigestScheduler`（`scheduler.go`）使用 `robfig/cron` 做定时调度。重构时不要替换调度器本身（它是稳定的），只替换 `generateDailyDigest`/`generateWeeklyDigest` 方法内的数据生成逻辑。

## Sources

- **Codebase analysis** (HIGH confidence): `topicanalysis/embedding.go`, `topicextraction/tagger.go`, `digest/scheduler.go`, `digest/generator.go`, `topicgraph/service.go`, `models/topic_graph.go`
- **OpenAI embedding threshold discussion** (MEDIUM confidence): OpenAI Community #873048 — `text-embedding-3-small` produces lower cosine similarity scores than `ada-002`
- **pgvector documentation** (HIGH confidence): PostgreSQL extension for vector similarity search, `pgvector/pgvector:pg18-trixie` Docker image
- **AGENTS.md conventions** (HIGH confidence): ID type mapping, frontend API patterns, editorial UI direction

---

*Research informed by direct codebase reading of 15+ source files across topicanalysis, topicextraction, digest, topicgraph, airouter, and models packages.*
