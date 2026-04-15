# Topic Graph 前端说明

## 这份文档讲什么

这份文档只讲前端的 Topic Graph 页面，也就是 `/topics` 这条路由背后的页面结构、数据流和与后端的对接方式。

它回答这些问题：

- 主题图谱页由哪些组件组成
- 页面初始加载时会拉哪些接口
- 选中 topic、热点标签、digest、文章预览时状态怎么流转
- AI analysis 面板是怎么轮询和回填的

## 页面入口

- 路由页：`front/app/pages/topics.vue`
- 页面主容器：`front/app/features/topic-graph/components/TopicGraphPage.vue`
- API 边界：`front/app/api/topicGraph.ts`

`topics.vue` 本身很薄，只负责挂载 `TopicGraphPage`。

## 当前页面结构

Topic Graph 页面当前不是单一图谱画布，而是一个多区块工作台。

### 主体组件

- `TopicGraphPage.vue`：页面状态中心，负责 orchestration
- `TopicGraphHeader.vue`：顶部控制区，切换 `daily/weekly`、日期、刷新
- `TopicGraphCanvas.client.vue`：3D 图谱画布，只负责展示和节点点击事件
- `TopicGraphSidebar.vue`：右侧 topic 详情栏
- `TopicTimeline.vue`：中下方时间线 / digest 列表
- `TopicGraphFooterPanels.vue`：底部分析与历史面板
- `ArticleContentView.vue`：文章预览弹层内容，直接复用主阅读组件

### 相关子组件

- `TopicAnalysisPanel.vue`
- `TopicAnalysisTabs.vue`
- `TopicAIAnalysisPanel.vue`
- `EventAnalysisView.vue`
- `PersonAnalysisView.vue`
- `KeywordAnalysisView.vue`
- `KeywordCloud.vue`
- `TopicTimeline.vue` / `TimelineItem.vue` / `TimelineHeader.vue`

## 当前数据来源

Topic Graph 的前端数据面主要集中在 `useTopicGraphApi()`。

### 图谱与详情

- `getGraph(type, date)` -> `/topic-graph/:type`
- `getTopicDetail(slug, type, date)` -> `/topic-graph/topic/:slug`
- `getTopicsByCategory(type, date)` -> `/topic-graph/by-category`
- `getDigestsByArticleTag(slug, type, date, limit)` -> `/topic-graph/tag/:slug/digests`

### analysis

- `getTopicAnalysis(...)` -> `/topic-graph/analysis`
- `getAnalysisStatus(...)` -> `/topic-graph/analysis/status`
- `rebuildTopicAnalysis(...)` -> `/topic-graph/analysis/rebuild`
- `retryTopicAnalysis(...)` -> `/topic-graph/analysis/retry`

### 相关文章

- `getTopicArticles(...)` -> `/topic-graph/topic/:slug/articles`

另外文章预览不会走 topic graph API，而是复用 `useArticlesApi().getArticle(articleId)` 拉标准 article 详情。

补充：article tags 的生成已经改成“主路径 + 兜底”模式：普通 refresh 文章会尽快打标签，`Firecrawl + 自动补全` 的文章会在补全完成后打标签，summary 阶段只补漏。

## 首次进入页面时会发生什么

场景：用户第一次打开 `/topics`。

链路：

1. `TopicGraphPage` 初始化默认状态：
   - `selectedType = daily`
   - `selectedDate = 今天`
2. 调用 `loadGraph()`
3. `getGraph(selectedType, selectedDate)` 拉图谱主体数据
4. 图谱成功后：
   - `graphPayload` 写入
   - 默认选中 `top_topics[0]`
   - 以第一个 topic 的 `slug` 继续加载详情
5. 同时并行触发 `loadHotspots()`
6. `getTopicsByCategory(...)` 拉热点标签分组数据
7. 如果默认 topic 存在，再调用 `loadTopicDetail(slug)` 拉右侧详情与时间线数据

也就是说，首屏其实至少有两组请求并行：

- 图谱主体
- 热点分类

然后再串行补 topic detail。

## 页面核心状态怎么分层

Topic Graph 当前的状态主源放在 `TopicGraphPage.vue`，不是 Pinia。

### 图谱主状态

- `selectedType`
- `selectedDate`
- `graphPayload`
- `loadingGraph`
- `notice`

### 当前选中对象

- `selectedTopicSlug`
- `selectedCategory`
- `selectedKeywordSlug`
- `detail`

### 热点与 digest 选择

- `hotspotData`
- `selectedHotspotTag`
- `hotspotDigests`
- `selectedDigestId`
- `previewDigestId`
- `showLowQualityTags`

### 文章预览

- `selectedPreviewArticle`
- `previewArticles`
- `loadingPreviewArticle`

### AI analysis 状态

页面内和底部面板内都各自维护了 analysis 相关状态。

- `TopicGraphPage.vue` 里有一组页面级 analysis 状态
- `TopicGraphFooterPanels.vue` 里还有按类型分开的 `analysisDataByType / statusByType / progressByType`

当前真实实现里，底部面板这套是更贴近实际展示链路的主实现。

## 视图模型层做了什么

原始图谱 payload 不直接喂给画布，而是先经过：

- `front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts`

它主要做这些转换：

- 归一化 topic category
- 给节点补 `size` 和 `accent`
- 优先根据 `quality_score` 推导节点大小和透明度，质量分缺失时再回退旧的 `weight`
- 过滤掉权重过低的 edge（`weight < 0.35`）
- 推导 `featuredNodeIds`
- 推导 trunk / branch / peripheral 三层视觉层级
- 生成页面头部统计文案

因此 `TopicGraphCanvas.client.vue` 更像是一个纯显示组件，视觉分层逻辑已经前移到 view model。

## 具体交互用例

### 用例 1：点击图谱节点

场景：用户在 3D 图谱里点了一个 topic node。

链路：

1. `TopicGraphCanvas.client.vue` 触发 `nodeClick`
2. `TopicGraphPage.handleNodeClick()` 接收 node
3. 只处理 `kind === 'topic'` 的节点
4. 更新 `selectedCategory`
5. 调用 `loadTopicDetail(node.slug)`
6. 右侧详情栏、时间线、底部分析面板一起切到这个 topic

这说明画布只负责“发出点击事件”，真正的状态切换都在页面容器层完成。

### 用例 2：点击热点标签

场景：用户从热点分类区点一个标签。

链路：

1. `handleTagSelect(slug, category)` 更新：
   - `selectedCategory`
   - `selectedTopicSlug`
   - `selectedHotspotTag`
2. 调用 `loadHotspotDigests(slug)`
3. 通过 `/topic-graph/tag/:slug/digests` 走反查链路：`tag -> articles -> digests`
4. 同时调用 `loadTopicDetail(slug)` 更新 sidebar 详情
5. 时间线区域优先展示 `hotspotDigests` 转换后的 `hotspotTimelineItems`

这里的 digest tags 现在不是 digest 自己的 summary topics，而是 digest 覆盖 article 的 `aggregated_tags`。

补充：热点标签分组和 fallback topic 列表现在都优先按 `quality_score` 排序。普通标签如果带 `is_low_quality: true`，默认会被隐藏；抽象标签不受这个默认过滤影响，用户可以通过页面开关显式显示全部标签。

所以热点标签点击后，页面会同时切换两套内容：

- 右侧还是 topic detail
- 中下区域则优先展示“包含这个标签文章的 digest 列表”

补充：标签层级视图里的 tag row 现在也会复用同一条 `handleTagSelect(slug, category)` 链路，所以从层级树里点标签时，也会同步驱动右侧详情栏和下方 timeline，而不是只停留在层级视图内部。

### 用例 3：选择 digest，再打开文章预览

场景：用户从时间线选中一条 digest，再点里面的文章。

链路：

1. `selectedDigestId` 指向当前 digest
2. `selectedDigest` 计算属性会把 digest 转成 `TimelineDigestSelection`
3. 如果当前来自热点 digests，会优先用 `matched_articles`
4. 如果当前来自 topic detail digests，会和 `detail.articles` 做一次 article id 交集
5. 用户点文章后调用 `openArticlePreview(articleId)`
6. 通过 `articlesApi.getArticle(articleId)` 拉完整 article
7. 弹出 `ArticleContentView`，复用主阅读链路里的：
   - Firecrawl 状态
   - AI 内容整理状态
   - 内容源切换
   - 手动抓取 / 手动整理动作
   - article tags 通用展示

这也是 topic graph 页面和主阅读链路复用最深的一段。

### 用例 4：底部 AI analysis 面板

场景：用户选中 topic 后，希望看人物/事件/关键词分析。

链路：

1. `TopicGraphFooterPanels.vue` 监听 `detail`
2. 基于 `detail.topic.category + kind` 计算当前 `selectedAnalysisType`
3. `loadAnalysis(type)` 先查 `/topic-graph/analysis`
4. 如果已存在结果，直接 parse `payload_json`
5. 如果没有现成结果，则调用 `getAnalysisStatus()`
6. 当状态是：
   - `pending` / `processing`：启动轮询
   - `ready`：重新拉完整 analysis
   - `missing`：必要时触发 `rebuildTopicAnalysis()`
7. 轮询间隔目前是 `1800ms`
8. 完成后把结果按类型写回 `analysisDataByType`

这里有一个当前实现上的现实细节：`TopicGraphFooterPanels.vue` 里分析请求的 `windowType` 目前固定写成了 `daily`，不是完全跟随页面顶部的 `selectedType`。

## 当前 UI 分区职责

### Header

- 切换 `daily/weekly`
- 选择锚点日期
- 刷新当前图谱

### Canvas

- 展示 topic/feed 节点和 topic_topic / topic_feed 边
- 根据 `highlightedNodeIds` 和 `relatedEdgeIds` 做高亮
- 不自己持有业务状态

### Sidebar

- 展示当前 topic 的：
  - 基础信息
  - 相关文章
  - 相关标签
  - 相关 topic
  - search/app links

### Timeline

- 展示 topic 关联 digest 或热点反查 digest
- 支持 timeline filters
- 选择 digest 后驱动底部与预览区域
- digest 卡片里的 tags 使用 `aggregated_tags`，表达“这份 digest 覆盖到的 article tag 索引”
- digest 下单篇文章打开后，文章弹窗展示 article 自身 tags，并对当前选中 topic 做高亮

### Footer Panels

- 当前显示占位信息，分析功能已禁用
- 后续将提供独立的主题分析看板入口
- 分析相关 API 和逻辑保留在后端，前端入口待后续实现

## 当前与 Pinia 的关系

Topic Graph 页面目前主要是页面内状态管理，不是 store 驱动页面。

但仓库里还有一个：

- `front/app/stores/aiAnalysis.ts`

它也封装了 topic analysis 的缓存和轮询逻辑，不过当前 `TopicGraphPage` / `TopicGraphFooterPanels` 这条主渲染链并没有直接依赖它。

也就是说，现在 topic analysis 在前端存在两套实现思路：

- 页面组件内直接管理
- `aiAnalysis` store 形式管理

如果后续继续收敛，这会是一个值得清理的边界点。

## 当前实现里容易忽略的点

- `/topics` 已经是独立页面，不属于主阅读页三栏壳
- 图谱页不是只看图，还串了热点标签、digest 时间线、analysis 和文章预览
- 文章预览复用 `ArticleContentView`，所以 topic graph 自动继承了 Firecrawl / 内容补全能力
- 热点标签点击后，中下区域优先显示反查 digest，不再只显示当前 topic detail 自带 summaries
- view model 会过滤低权重边，因此后端返回的 edge 不一定都会进最终画布

## 标签质量评分与低质量标签

### 概述

每个标签有一个 0–1 的 `quality_score`，由后端定时任务 `tag_quality_score` 计算。当 `quality_score < 0.3` 且标签非抽象时，该标签被标记为"低质量"（`is_low_quality = true`）。

### 质量分数计算流程

代码位置：`backend-go/internal/domain/topicextraction/quality_score.go`

#### 第一步：采集原始指标

对每个 `status = 'active'` 的标签，通过 SQL 聚合四项原始指标：

| 指标 | SQL 来源 | 含义 |
|------|----------|------|
| `article_count` | `COUNT(DISTINCT article_topic_tags.article_id)` | 该标签关联了多少篇文章 |
| `feed_diversity` | `COUNT(DISTINCT articles.feed_id)` | 这些文章来自几个不同的 feed |
| `avg_cooccurrence` | `AVG(同一篇文章上的其他标签数)` | 该标签平均和多少其他标签共现 |
| `semantic_match_avg` | 当前固定为 0（预留） | 语义匹配置信度 |

注意：`semantic_match_avg` 当前 SQL 硬编码为 `0 AS semantic_match_avg`，后续代码中如果该值为 0 则回退使用默认常量 `0.7`。

#### 第二步：百分位排名（percentile ranking）

对前三项指标（article_count、avg_cooccurrence、feed_diversity），分别在全量标签中计算百分位排名：

```
percentileRank = 该标签超过的标签数 / 总标签数
```

例如有 100 个标签，某标签的 article_count 排在第 80 名，则 `freqPct = 0.80`。

- 标签总数 < 3 时，所有百分位直接返回 0.5
- 标签不在统计列表中时，也返回 0.5

#### 第三步：加权合成质量分数

```go
// 实际权重
quality_score = 0.4 * freqPct + 0.2 * coocPct + 0.2 * feedDivPct + 0.2 * semanticPct
```

| 维度 | 权重 | 原始数据 | 排名方式 |
|------|------|----------|----------|
| 频率 | **40%** | `article_count` | 百分位排名 |
| 共现度 | 20% | `avg_cooccurrence` | 百分位排名 |
| Feed 多样性 | 20% | `feed_diversity` | 百分位排名 |
| 语义质量 | 20% | 当前固定 `0.7` | 直接使用 |

频率权重最高，意味着被更多文章引用的标签质量更高。

#### 第四步：抽象标签的分数继承

抽象标签（`is_abstract = true`）不直接计算分数，而是由其子标签加权平均得出：

```
abstract_score = Σ(子标签.quality_score × 子标签.article_count) / Σ(子标签.article_count)
```

子标签文章数作为权重，文章越多的子标签对父标签分数贡献越大。无文章记录的子标签保底权重为 1。

#### 第五步：无文章的标签

如果一个标签没有任何关联文章（`article_count = 0`），其 `quality_score` 直接设为 `0`。

### 低质量标签判定标准

```go
// 后端在构建 API 响应时动态标记，不持久化到数据库
IsLowQuality = (Source != "abstract") && (QualityScore < 0.3)
```

即同时满足两个条件：
1. **非抽象标签** — 标签来源不是 abstract（抽象标签豁免）
2. **质量分数低于 0.3** — 综合排名在 30% 以下

典型的低质量标签特征：文章关联少、feed 来源单一、很少和其他标签共现。

### 前端行为

- **3D 拓扑图**：低质量节点默认不渲染，用户可通过节点旁的眼睛图标手动显示
- **热点题材列表**：低质量标签排在底部，显示"低质量"文字标记
- **标签层级视图**：低质量标签显示橙色"低质量"徽章

### 相关代码

| 文件 | 职责 |
|------|------|
| `backend-go/internal/domain/topicextraction/quality_score.go` | 质量分数计算核心逻辑 |
| `backend-go/internal/jobs/tag_quality_score.go` | 定时任务调度，默认每小时运行一次 |
| `backend-go/internal/domain/topicgraph/service.go` | 在 API 响应中标记 `is_low_quality` |
| `backend-go/internal/domain/topicanalysis/abstract_tag_service.go` | 层级 API 响应中标记 `is_low_quality` |
| `front/app/features/topic-graph/components/TopicGraphPage.vue` | 拓扑图默认隐藏、热点列表排序 |
| `front/app/features/topic-graph/components/TagHierarchyRow.vue` | 标签层级行中的"低质量"徽章 |

## 标签匹配与抽象层级

### 概述

新标签从文章/摘要中提取后，经过 `findOrCreateTag` 的三阈值匹配流程决定是复用、归入抽象、还是新建。抽象标签建立后还会触发层级匹配，形成多级分类树。整个过程围绕 embedding 相似度驱动。

核心代码：`backend-go/internal/domain/topicextraction/tagger.go`、`backend-go/internal/domain/topicanalysis/abstract_tag_service.go`、`backend-go/internal/domain/topicanalysis/embedding.go`

### 三阈值匹配流程

`findOrCreateTag` 对每个提取出的标签执行以下匹配：

```
新标签 → TagMatch() 生成 embedding → FindSimilarTags 搜索
    ↓
┌─────────────────────────────────────────────────────────┐
│ exact（slug 完全匹配）      → 复用现有标签，更新元信息    │
│ >= 0.97 + 普通标签          → 复用，更新别名              │
│ >= 0.97 + 抽象标签          → 创建子标签，建立父子关系    │
│ 0.78~0.97 + 普通标签        → LLM 抽象提取，两个子标签    │
│ 0.78~0.97 + 抽象标签        → 创建子标签，建立父子关系    │
│ < 0.78                     → 全新标签，生成 embedding     │
└─────────────────────────────────────────────────────────┘
```

阈值可在 `embedding_config` 表配置，默认值：

| 阈值 | 默认值 | 含义 |
|------|--------|------|
| `HighSimilarity` | 0.97 | 自动复用 |
| `LowSimilarity` | 0.78 | 自动新建 |

### 抽象标签创建（middle band + 普通标签）

当两个普通标签在 0.78~0.97 之间相似时，`ExtractAbstractTag` 通过 LLM 提取公共概念：

1. LLM 生成抽象名称和描述
2. 创建抽象标签（`source = "abstract"`），异步生成 embedding
3. 两个普通标签成为抽象的子标签
4. 触发 `MatchAbstractTagHierarchy` 搜索更高级抽象

### 抽象层级匹配

新抽象标签创建后，`MatchAbstractTagHierarchy` 立即搜索相似抽象标签，建立多级层级：

| 相似度 | 行为 |
|--------|------|
| >= 0.97 | 新抽象成为现有抽象的子标签 |
| 0.78~0.97 | AI 判断谁更宽泛（parent/child），建立关系 |
| < 0.78 | 无操作 |

这一步使得抽象标签可以形成树状层级，例如"编程语言 → 前端框架 → React"。

### Embedding 管理策略

Embedding 的保留/删除直接决定标签能否被未来的相似标签匹配到。策略遵循以下规则：

**抽象标签：始终保留 embedding**

抽象标签代表分类概念，必须保持可被匹配。`MatchAbstractTagHierarchy` 中建立抽象父子关系时**不删除**子抽象标签的 embedding。

**普通标签：按兄弟情况决定**

```
删除 embedding 的条件（同时满足）：
  1. 父标签是抽象标签
  2. 同级没有抽象兄弟标签

保留 embedding 的条件（满足其一）：
  - 父标签不是抽象标签（独立标签）
  - 同级有抽象兄弟标签（需要精确匹配锚点）
```

这个策略的理由：

- **无抽象兄弟时**：父抽象是唯一的匹配入口，普通子标签不需要独立匹配
- **有抽象兄弟时**：普通子标签必须保留 embedding，否则新标签只能匹配到父抽象或错误地匹配到抽象兄弟，导致横向扩容

示例：

```
编程语言（抽象，有 embedding）
├── 前端框架（抽象，有 embedding）     ← 抽象，保留
│   ├── React（普通，无 embedding）    ← 父抽象，无抽象兄弟 → 删除
│   └── Vue（普通，无 embedding）      ← 同上
├── Python（普通，有 embedding）       ← 父抽象，有抽象兄弟"前端框架" → 保留
└── Java（普通，有 embedding）         ← 同上
```

### Embedding 的动态补回

当抽象标签通过 `linkAbstractParentChild` 被链接到另一个抽象下时，父级的普通子标签可能突然获得了抽象兄弟。此时 `enqueueEmbeddingsForNormalChildren` 异步为这些缺少 embedding 的普通子标签补生成 embedding。

场景：Python 先成为"编程语言"的子标签（此时无抽象兄弟，不生成 embedding），之后"前端框架"被链接到"编程语言"下，Python 现在有了抽象兄弟，系统自动补生成 Python 的 embedding。

### 相关代码

| 文件 | 职责 |
|------|------|
| `backend-go/internal/domain/topicextraction/tagger.go` | `findOrCreateTag` 三阈值匹配主流程 |
| `backend-go/internal/domain/topicanalysis/embedding.go` | `TagMatch`、`FindSimilarTags`、`FindSimilarAbstractTags`、`DeleteTagEmbedding` |
| `backend-go/internal/domain/topicanalysis/abstract_tag_service.go` | `ExtractAbstractTag`、`MatchAbstractTagHierarchy`、`linkAbstractParentChild`、`enqueueEmbeddingsForNormalChildren` |

## 相关文件

- `front/app/pages/topics.vue`
- `front/app/api/topicGraph.ts`
- `front/app/features/topic-graph/components/TopicGraphPage.vue`
- `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue`
- `front/app/features/topic-graph/components/TopicGraphSidebar.vue`
- `front/app/features/topic-graph/components/TopicGraphFooterPanels.vue`
- `front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts`

## 建议阅读顺序

- 先看 `front/app/pages/topics.vue`
- 再看 `front/app/features/topic-graph/components/TopicGraphPage.vue`
- 再看 `front/app/api/topicGraph.ts`
- 再看 `front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts`
- 最后看 `front/app/features/topic-graph/components/TopicGraphFooterPanels.vue` 和 `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue`
