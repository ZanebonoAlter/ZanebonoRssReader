# Architecture: 标签智能收敛与关注推送

**Domain:** RSS Reader 标签系统增强 + 日报周报重构
**Researched:** 2026-04-12
**Overall confidence:** HIGH

## Executive Summary

在现有 domain-driven 架构上扩展四个核心能力：标签自动收敛、关注标签、关注视角日报周报、标签趋势分析。经过完整代码审查，所有集成点明确，不需要新建 domain 包，而是在 `topicextraction`（标签收敛钩子）、`topicanalysis`（趋势分析扩展）、`digest`（完全替换生成逻辑）三个现有包内修改，仅新增 `watched_tags` 表和对应 handler。

**关键架构决策：**
1. **收敛在 `findOrCreateTag` 中触发** — 这是标签创建的唯一入口，hook 在此处可确保每条新标签都经过相似度检查
2. **关注标签轻量化** — 仅一个新表 `watched_tags`（topic_tag_id + watched_at），不引入复杂关系
3. **日报周报完全替换** — 新生成逻辑按关注标签聚合文章，替换现有按 category 聚合的 `DigestGenerator`
4. **趋势分析复用 `topicanalysis`** — 新增 `trend` analysis_type，复用现有 AnalysisService + Queue 架构

## 现有架构全景

### Domain 包依赖关系（当前）

```
topicextraction ──→ models (TopicTag, ArticleTopicTag, AISummaryTopic)
       ↓
topicanalysis ──→ models (TopicTagEmbedding, TopicTagAnalysis)
       ↓              ↓
topicgraph ──→ topicextraction, topictypes
       ↓
digest ──→ models (AISummary), topicextraction
```

### 关键集成点

| 集成点 | 位置 | 作用 |
|--------|------|------|
| 标签创建入口 | `topicextraction.findOrCreateTag()` | 所有标签（article/summary）创建/匹配的唯一路径 |
| 标签向量存储 | `topicanalysis.EmbeddingService` | 生成 + 查询 embedding，三级阈值匹配 |
| 日报生成 | `digest.DigestGenerator` | 按 category 聚合 AISummary → 需替换为按 watched tag 聚合 article |
| 定时调度 | `digest.DigestScheduler` | cron 调度 + 飞书/Obsidian 推送 → 保留调度框架，替换生成内容 |
| 文章-标签关联 | `models.ArticleTopicTag` | 文章与标签的多对多关联，所有查询基础 |
| 摘要-标签关联 | `models.AISummaryTopic` | 摘要与标签的多对多关联 |
| 向量搜索 | `topic_tag_embeddings` 表 | 存储 JSON 格式向量（非 pgvector），用 Go 内存 cosine 计算 |

## 推荐架构

### 总体组件图

```
┌─────────────────────────────────────────────────────────────┐
│                    Backend (Go)                              │
│                                                             │
│  ┌──────────────────┐    ┌──────────────────┐              │
│  │  topicextraction │    │  topicanalysis   │              │
│  │                  │    │                  │              │
│  │  findOrCreateTag │───→│  EmbeddingService│              │
│  │    + 收敛 hook   │    │  .FindSimilarTags│              │
│  │                  │    │  + 收敛逻辑       │              │
│  └──────────────────┘    └──────────────────┘              │
│          │                        │                         │
│          ↓                        ↓                         │
│  ┌──────────────────┐    ┌──────────────────┐              │
│  │  watchedtags     │    │  tagtrends       │              │
│  │  (新 handler +   │    │  (扩展           │              │
│  │   model)         │    │   topicanalysis)  │              │
│  └──────────────────┘    └──────────────────┘              │
│          │                        │                         │
│          ↓                        ↓                         │
│  ┌──────────────────────────────────────────┐              │
│  │           digest (重写生成逻辑)            │              │
│  │                                          │              │
│  │  DigestGenerator → WatchedTagDigestGen   │              │
│  │  scheduler 保留，generateDaily/Weekly 替换│              │
│  └──────────────────────────────────────────┘              │
│                                                             │
│  ┌──────────────────┐                                      │
│  │   tagrecommend   │  关注标签相关推荐                      │
│  │   (topicanalysis │  embedding相似 + 共现                  │
│  │    扩展)          │                                      │
│  └──────────────────┘                                      │
└─────────────────────────────────────────────────────────────┘
         │
         ↓ WebSocket (tag_converged, watched_articles)
┌─────────────────────────────────────────────────────────────┐
│              Frontend (Nuxt 4)                               │
│                                                             │
│  新 API client 方法 + 新 Pinia store (watchedTags)          │
│  新页面/组件：关注标签设置、关注文章流、趋势图表               │
│  修改现有 digest 页面：按标签聚合视图                         │
└─────────────────────────────────────────────────────────────┘
```

### 新增/修改文件清单

#### Phase 1: 标签自动收敛

**修改文件：**

| 文件 | 修改内容 | 原因 |
|------|---------|------|
| `internal/domain/topicextraction/tagger.go` | `findOrCreateTag()` 新增收敛分支 | 这是标签创建的唯一入口，embedding 匹配在此触发 |
| `internal/domain/topicanalysis/embedding.go` | 新增 `FindAndMergeSimilarTag()` 方法 | 封装收敛逻辑：查找 → 决策 → 合并别名 → 更新关联 |
| `internal/domain/models/topic_graph.go` | `TopicTag` 新增 `MergedFrom` 字段（可选） | 追踪合并历史，前端可展示 |

**数据流变更：**
```
Before: findOrCreateTag() → slug lookup → found? reuse : create
After:  findOrCreateTag() → slug lookup → found? reuse
                                        → not found? EmbeddingService.TagMatch()
                                           → high_similarity: 合并到 existing tag
                                           → low_similarity: 创建新 tag
                                           → ai_judgment: AI 判定（暂不实现，降级为创建）
```

**收敛策略（在 `findOrCreateTag` 内）：**
```go
// 伪代码 - 插入在 slug lookup 失败后
candidates, err := embeddingService.FindSimilarTags(ctx, &candidateTag, category, 3)
if err == nil && len(candidates) > 0 && candidates[0].Similarity >= convergenceThreshold {
    // 收敛：将新标签作为别名加入已有标签
    existing := candidates[0].Tag
    mergedAliases := append(existing.Aliases, candidateTag.Label)
    // 更新 existing tag 的 aliases
    // 返回 existing tag（而不是创建新 tag）
    return existing, nil
}
// 否则正常创建
```

#### Phase 2: 关注标签

**新增文件：**

| 文件 | 内容 |
|------|------|
| `internal/domain/models/watched_tag.go` | `WatchedTag` 模型定义 |
| `internal/domain/watchedtags/handler.go` | HTTP handler: 列表、添加、删除 |
| `internal/domain/watchedtags/service.go` | 关注/取关、获取关注标签文章、推荐相关标签 |

**修改文件：**

| 文件 | 修改内容 |
|------|---------|
| `internal/app/router.go` | 新增 `/api/watched-tags` 路由组 |

**新模型：**
```go
// internal/domain/models/watched_tag.go
type WatchedTag struct {
    ID         uint      `gorm:"primaryKey" json:"id"`
    TopicTagID uint      `gorm:"uniqueIndex;not null" json:"topic_tag_id"`
    WatchedAt  time.Time `gorm:"autoCreateTime" json:"watched_at"`
    
    TopicTag   *TopicTag `gorm:"foreignKey:TopicTagID" json:"topic_tag,omitempty"`
}
```

**新 API 端点：**
```
GET    /api/watched-tags                    — 获取关注标签列表（含标签详情）
POST   /api/watched-tags                    — 添加关注（body: {topic_tag_id})
DELETE /api/watched-tags/:topic_tag_id      — 取消关注
GET    /api/watched-tags/articles           — 获取关注标签关联文章（支持分页、按标签筛选）
GET    /api/watched-tags/recommendations    — 推荐相关标签（embedding相似 + 共现）
```

#### Phase 3: 日报周报重构

**修改文件：**

| 文件 | 修改内容 |
|------|---------|
| `internal/domain/digest/generator.go` | 新增 `WatchedTagDigestGenerator`，按关注标签聚合文章替换按 category 聚合 AISummary |
| `internal/domain/digest/scheduler.go` | `generateDailyDigest()` / `generateWeeklyDigest()` 使用新生成器 |
| `internal/domain/digest/handler.go` | `GetDigestPreview()` 适配新数据结构，`RunDigestNow()` 同步 |
| `internal/domain/digest/models.go` | 可能需要扩展 DigestConfig（如关注标签优先级配置） |

**新数据流（关注标签视角）：**
```
1. 获取所有 watched_tags
2. 按标签聚合时间窗口内的文章（JOIN article_topic_tags）
3. 每个标签 → 文章列表 + AI 摘要（如果有）+ 摘要原文
4. 生成 Markdown：按标签维度而非分类维度
5. 推送到飞书/Obsidian/OpenNotebook（推送框架不变）
```

**关键设计：新 DigestItem 结构**
```go
type WatchedTagDigestItem struct {
    Tag        models.TopicTag
    Articles   []models.Article
    Summaries  []models.AISummary  // 关联的 AI 摘要
    TrendDelta int                  // 相比昨天文章数变化
}
```

#### Phase 4: 标签趋势分析

**修改文件：**

| 文件 | 修改内容 |
|------|---------|
| `internal/domain/topicanalysis/analysis_service.go` | 扩展支持 `trend` analysis_type，新增趋势数据构建方法 |
| `internal/domain/topicanalysis/analysis_handler.go` | 新增趋势查询路由 |
| `internal/domain/topictypes/types.go` | 新增 `TagTrendData` 等响应类型 |

**新增文件（可选，如果趋势逻辑复杂）：**

| 文件 | 内容 |
|------|------|
| `internal/domain/topicanalysis/trend.go` | 趋势分析专用逻辑（时间序列聚合、变化率计算） |

**趋势数据来源：**
- 文章数时间序列：`article_topic_tags` 按 `created_at` 分桶统计
- 活跃度变化：与前一天/上周同期对比
- 新标签检测：首次出现在关注时间窗口内的标签

**趋势查询 API（复用现有 analysis handler 模式）：**
```
GET /api/topic-graph/analysis/trend?tag_id=X&window_type=daily&anchor_date=2026-04-12
GET /api/topic-graph/analysis/trend/batch?tag_ids=1,2,3&window_type=weekly
```

#### Phase 5: 相关标签推荐

**修改文件：**

| 文件 | 修改内容 |
|------|---------|
| `internal/domain/watchedtags/service.go` | `GetRecommendations()` 方法 |
| `internal/domain/topicanalysis/embedding.go` | 暴露 `FindSimilarTags` 给 watchedtags 包使用 |

**推荐算法（双信号融合）：**
```
信号1: Embedding 相似度 — 复用 FindSimilarTags()，取 top 10
信号2: 共现频率 — 复用 topicgraph.getRelatedTags() 的 SQL 模式

融合：加权求分
  score = 0.6 * embedding_similarity + 0.4 * normalized_cooccurrence
  
过滤：排除已关注的标签
排序：按融合分数降序
```

## 组件边界与职责

| 组件 | 职责 | 依赖 | 新增/修改 |
|------|------|------|----------|
| `topicextraction` | 标签提取 + 创建 + 收敛 hook | `topicanalysis` (收敛时调用) | 修改 |
| `topicanalysis` | Embedding 生成/匹配 + 收敛执行 + 趋势分析 | `airouter`, `models` | 修改 + 扩展 |
| `watchedtags` | 关注标签 CRUD + 文章查询 + 推荐 | `topicanalysis`, `models` | **新增** |
| `digest` | 日报周报生成（关注标签视角） | `watchedtags`, `models`, `topicextraction` | 修改（重写生成） |
| `topicgraph` | 话题图谱 + 详情（不变） | 不变 | 不变 |
| `models` | 数据模型 | 无 | 新增 WatchedTag |
| `app/router` | 路由注册 | 各 domain handler | 修改（新增路由） |

## 数据流

### 收敛流程（实时，在 TagQueue worker 内）

```
文章入库 → TagJobQueue.Enqueue → TagQueue.processJob
  → TagArticle / RetagArticle
    → tagArticle()
      → findOrCreateTag() ← 【收敛 hook 插入点】
        → slug 查找
        → 未找到 → EmbeddingService.TagMatch()
          → high_similarity(≥0.97): 复用 existing tag
          → low_similarity(<0.78): 创建新 tag + 生成 embedding
          → 中间: 创建新 tag（AI 判定暂不实现）
        → 返回 tag
      → 创建 ArticleTopicTag 关联
  → WebSocket 广播 tag_completed
```

### 关注文章推送（前端请求）

```
前端请求 GET /api/watched-tags/articles?tag_id=5
  → watchedtags.handler.GetWatchedArticles()
    → watchedtags.service.GetWatchedTagArticles()
      → 查询 watched_tags 确认关注
      → JOIN article_topic_tags + articles
      → WHERE topic_tag_id IN (watched tag ids)
      → ORDER BY created_at DESC, LIMIT, OFFSET
    → 返回文章列表
```

### 日报生成（关注标签视角）

```
DigestScheduler cron 触发
  → generateDailyDigest()
    → 获取 watched_tags 列表
    → 按 tag 聚合当日文章
    → 为每个 tag 附加关联 AI 摘要
    → 生成 Markdown（按标签维度）
    → 推送到飞书/Obsidian/OpenNotebook（框架不变）
```

### 趋势分析

```
前端请求 GET /api/topic-graph/analysis/trend?tag_id=X
  → AnalysisHandler → AnalysisService
    → 查询 article_topic_tags 按日分桶
    → 计算与前日/上周的变化率
    → 返回 TagTrendData{dates, counts, delta, direction}
```

## API 端点设计

### 新增端点

```
# 关注标签
GET    /api/watched-tags                      ListWatchedTags
POST   /api/watched-tags                      AddWatchedTag          {topic_tag_id: uint}
DELETE /api/watched-tags/:topic_tag_id        RemoveWatchedTag

# 关注文章推送
GET    /api/watched-tags/articles             GetWatchedArticles     ?tag_id=&page=&page_size=

# 相关标签推荐
GET    /api/watched-tags/recommendations      GetRecommendations     ?limit=10

# 趋势分析（复用现有 analysis 路由）
GET    /api/topic-graph/analysis              GetTrendAnalysis       ?tag_id=&analysis_type=trend&window_type=daily
```

### 修改端点

```
# 日报周报 — 输出结构变化，路由不变
GET    /api/digest/preview/:type              → 返回 WatchedTagDigestItem 结构
POST   /api/digest/run/:type                  → 使用新生成逻辑
```

## 模式与规范

### 模式 1: Domain Service 单例

**复用现有模式** — 全局单例通过 `sync.Once` 初始化：

```go
// 复用 topicanalysis 包的模式
var (
    watchedServiceGlobal WatchedTagService
    watchedServiceOnce   sync.Once
)

func GetWatchedTagService(db *gorm.DB) WatchedTagService {
    watchedServiceOnce.Do(func() {
        watchedServiceGlobal = NewWatchedTagService(db)
    })
    return watchedServiceGlobal
}
```

### 模式 2: Handler → Service → DB 分层

**严格遵循** — handler 不访问 DB，service 不直接写 HTTP 响应：

```go
// handler 只做参数解析 + 调用 service + 格式化响应
func AddWatchedTag(c *gin.Context) {
    var req struct{ TopicTagID uint `json:"topic_tag_id"` }
    if err := c.ShouldBindJSON(&req); err != nil { ... }
    tag, err := service.Watch(req.TopicTagID)
    if err != nil { ... }
    c.JSON(200, gin.H{"success": true, "data": tag})
}
```

### 模式 3: 响应格式统一

**所有新端点遵循** `{success: bool, data/error/message: ...}` 格式。

## 反模式（避免）

### 反模式 1: 收敛逻辑放在 TagQueue worker 层

**错误：** 在 `TagQueue.processJob()` 或 `tagArticle()` 外部做收敛
**后果：** 收敛逻辑与标签创建脱节，可能漏掉收敛点
**正确：** 在 `findOrCreateTag()` 内部调用收敛，因为这是标签创建的唯一路径

### 反模式 2: 为关注标签创建完整的独立 domain 包

**错误：** 新建 `internal/domain/watchedtags/` 包含 models、service、handler 全套
**后果：** 过度设计，关注标签本质就是一个 join 表 + 简单 CRUD
**正确：** models 放在 `models/` 包，handler 和 service 放在 `watchedtags/` 包（轻量）

### 反模式 3: 日报周报保留旧逻辑并行

**错误：** 新增关注标签视图，保留按 category 聚合视图
**后果：** 两套生成逻辑维护成本高，数据结构不兼容
**正确：** 完全替换，旧逻辑删除。PROJECT.md 已明确此决策

### 反模式 4: 收敛时迁移已有关联

**错误：** 收敛时把旧标签的 article_topic_tags 全部迁移到新标签
**后果：** 大量 UPDATE 操作，可能中断正在进行的查询，且无法回滚
**正确：** 收敛只做别名合并 — 新文章使用已有标签，旧关联保持不变。通过别名查询时自动包含

## 构建顺序（依赖约束）

```
Phase 1: 标签自动收敛
  ├── 依赖: 现有 EmbeddingService.FindSimilarTags（已就绪）
  ├── 修改: topicanalysis/embedding.go（新增收敛方法）
  ├── 修改: topicextraction/tagger.go（findOrCreateTag hook）
  └── 验证: 新文章入库时自动合并到高相似度标签

Phase 2: 关注标签（依赖 Phase 1 的收敛后标签空间）
  ├── 新增: models/watched_tag.go
  ├── 新增: watchedtags/handler.go, service.go
  ├── 修改: router.go（注册路由）
  └── 验证: 关注/取关 CRUD，获取关注标签文章

Phase 3: 日报周报重构（依赖 Phase 2 的关注标签列表）
  ├── 修改: digest/generator.go（新 WatchedTagDigestGenerator）
  ├── 修改: digest/scheduler.go（使用新生成器）
  ├── 修改: digest/handler.go（适配新数据结构）
  └── 验证: 日报/周报按关注标签维度生成

Phase 4: 标签趋势分析（依赖 Phase 2 的关注标签）
  ├── 修改: topicanalysis/analysis_service.go（支持 trend 类型）
  ├── 新增: topicanalysis/trend.go（趋势计算逻辑）
  ├── 修改: topicanalysis/analysis_handler.go（趋势路由）
  └── 验证: 指定标签的历史趋势数据返回正确

Phase 5: 相关标签推荐（依赖 Phase 1 的 embedding + Phase 2 的关注列表）
  ├── 修改: watchedtags/service.go（推荐方法）
  └── 验证: 推荐列表排除已关注标签，按融合分数排序
```

**依赖关系图：**
```
Phase 1 (收敛) ──→ Phase 2 (关注) ──→ Phase 3 (日报重构)
                       │                     │
                       └──→ Phase 4 (趋势)   └── 日报需要关注标签列表
                       │
                       └──→ Phase 5 (推荐) ──→ 需要 Phase 1 的 embedding
```

## 可扩展性考虑

| 关注点 | 100 标签 | 1K 标签 | 10K 标签 |
|--------|----------|---------|----------|
| 收敛匹配 | 现有 JSON 遍历即可 | 可接受 | 需要优化：按 category 预过滤 + 缓存热点 embedding |
| 关注文章查询 | 直接 JOIN 查询 | 直接 JOIN 查询 | 需要索引优化：`(topic_tag_id, created_at)` 复合索引 |
| 趋势聚合 | SQL COUNT 分组 | SQL COUNT 分组 | 需要预计算/物化视图 |
| 日报生成 | 单次查询 | 单次查询 | 可能需要分批查询 |

**索引建议（随 Phase 2 创建）：**
```sql
CREATE INDEX idx_watched_tags_topic_tag_id ON watched_tags(topic_tag_id);
CREATE INDEX idx_article_topic_tags_tag_created ON article_topic_tags(topic_tag_id, created_at);
```

## 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| 收敛阈值不合适导致过度合并 | 标签丢失语义区分 | 收敛阈值可配置，初期使用保守阈值（≥0.97） |
| embedding 服务不可用时收敛失败 | 新文章不触发收敛 | 降级为不收敛（现有行为），不阻塞标签创建 |
| 日报重构后旧数据不兼容 | 前端 digest 页面展示错误 | 同步更新前端，preview API 保持向后兼容字段 |
| 关注标签为空时日报无内容 | 用户未关注任何标签 | 日报降级：展示 top N 热门标签文章 + 提示关注 |

## Sources

- 代码审查: `backend-go/internal/domain/` 全部 13 个子包
- 代码审查: `backend-go/internal/platform/airouter/` embedding 实现
- 代码审查: `backend-go/internal/app/router.go`, `runtime.go`
- PROJECT.md 里程碑定义和关键决策
- AGENTS.md 项目规范
