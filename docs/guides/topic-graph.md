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

所以热点标签点击后，页面会同时切换两套内容：

- 右侧还是 topic detail
- 中下区域则优先展示“包含这个标签文章的 digest 列表”

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

### Footer Panels

- 展示 history
- 展示 analysis tabs 与分析结果
- 提供分析刷新 / 重建入口

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
