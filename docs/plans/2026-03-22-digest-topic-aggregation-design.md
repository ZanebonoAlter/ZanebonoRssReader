# Digest Topic Aggregation Design

**Date:** 2026-03-22
**Status:** Approved

## Goal

让日报/周报不再拥有独立的 topic tag 语义。

Digest 在主题图谱和 digest 详情中的 tag，统一改为“该 digest 覆盖 article 的聚合 tag 索引”。
同时把 article tags 接入标准文章详情接口，并在前端做通用可视化展示，供主阅读页、digest 弹窗和 topic graph 复用。

## Why This Route

- 现在 digest 本质是按时间窗聚合 `ai_summaries`，不是独立 topic 实体
- 真正稳定、可复用的标签来源已经是 `article_topic_tags`
- 保留 digest 自己的 tag 会长期造成两套语义并存：`digest own tags` vs `article aggregate tags`
- 把 digest 定义为“文章集合的聚合视图”后，topic graph、digest 页面、article 详情三处语义就统一了

## Current Architecture Fit

现有关键链路：

- article tags 来源：`backend-go/internal/domain/topicextraction/article_tagger.go`
- digest 聚合：`backend-go/internal/domain/digest/generator.go`
- topic graph digest / topic detail：`backend-go/internal/domain/topicgraph/`
- article 详情接口：`backend-go/internal/domain/articles/handler.go`
- 通用文章预览：`front/app/features/articles/components/ArticleContentView.vue`
- topic graph 页面：`front/app/features/topic-graph/components/TopicGraphPage.vue`
- digest 文章弹窗：`front/app/features/digest/components/DigestDetail.vue`

## Recommended Architecture

### 1. Source Of Truth

digest 的 tag 不落独立持久化表，也不再把 `ai_summary.summary_topics` 视为 digest 自身标签。

统一规则：

- article 的标签来源是 `article_topic_tags`
- digest 的标签来源是 digest 覆盖 article 的 tag 聚合结果
- topic graph 中 digest 相关节点和卡片展示，统一使用 digest 聚合 tags

`ai_summary.summary_topics` 继续保留给摘要自身链路兼容使用，但不再作为 digest tags 主来源。

### 2. Runtime Aggregation Instead Of New Storage

不新增 `digest_topic_tags` 之类的新表。

原因：

- digest 不是独立知识实体
- 聚合索引天然依赖 article 集合，运行时计算更贴近真实语义
- 避免数据冗余、回填、增量同步和历史一致性问题

聚合结果建议结构：

- `label`
- `slug`
- `category`
- `kind`（可选，若前端高亮/归一化需要）
- `score` 或 `weight`
- `article_count`

其中 `article_count` 用于表达该 tag 在当前 digest 中命中了多少篇 article，前端可直接显示为索引强度。

### 3. Standard Article Detail Must Carry Tags

标准文章接口 `GET /api/articles/:id` 增加 `tags` 字段。

这样一来：

- 主阅读页文章详情能展示 tags
- digest 弹窗复用 `ArticleContentView` 时自动展示 tags
- topic graph 文章预览也自动展示 tags

前端不需要在多个页面分别拼接 article tag 查询逻辑。

### 4. Topic Graph Must Switch To Aggregated Digest Tags

topic graph 当前 digest 展示存在两种不一致：

- `detail.summaries` 走 summary topics
- hotspot digests 前端甚至有 `tags: []` 占位

这次统一改成：

- topic detail 里的 digest summary 返回 `aggregated_tags`
- hotspot digests 返回 `aggregated_tags`
- timeline、digest modal、sidebar 全部消费同一字段

这样 `/topics` 页面才真正体现“日报展示其覆盖 article 的所有 tag 作为索引”。

## Backend Design

### 1. Aggregation Service Shape

建议在 digest 域补一层聚合工具，职责是：

- 输入：digest 关联 article IDs
- 查询：`article_topic_tags -> topic_tags`
- 输出：去重后的聚合 tags，附带 `article_count`

同一套聚合逻辑需要被两处复用：

- digest 页面相关接口
- topic graph 返回 digest 卡片时

### 2. API Contract Changes

#### `GET /api/articles/:id`

新增：

```json
{
  "success": true,
  "data": {
    "id": 123,
    "title": "...",
    "tags": [
      {
        "slug": "iran",
        "label": "伊朗",
        "category": "keyword",
        "score": 0.92
      }
    ]
  }
}
```

#### Digest preview / detail response

每个 digest summary 补：

- `aggregated_tags`

建议前端以后统一只消费这个字段作为 digest tags。

#### Topic graph digest responses

以下返回中的 digest 卡片统一补 `aggregated_tags`：

- `/api/topic-graph/tag/:slug/digests`
- `/api/topic-graph/topic/:slug`

### 3. Compatibility Rule

过渡期可以保留后端现有 `topics` 字段，避免一次性打断老前端映射。

但新前端逻辑应遵循：

- digest tags 优先读 `aggregated_tags`
- `topics` 只作为兼容字段，不再代表 digest 自身标签语义

后续稳定后可以逐步弱化 `topics` 的展示用途。

## Frontend Design

### 1. Reusable Article Tag UI

新增一个通用组件，例如：

- `front/app/features/articles/components/ArticleTagList.vue`

输入建议：

- `tags`
- `highlightedSlugs?`
- `compact?`
- `grouped?`
- `maxVisible?`

组件职责：

- 支持轻量 chip 展示
- 支持按 category 分组或混排
- 支持命中高亮
- 支持折叠/展开，避免标签墙

### 2. ArticleContentView As Universal Entry

`ArticleContentView.vue` 增加 article tags 展示区。

推荐位置：

- 标题下方 meta 区
- 阅读动作区上方

这样以下场景自动通用：

- 主阅读页正文
- digest 文章弹窗
- topic graph 文章预览

### 3. Digest UI Changes

digest 详情和弹窗中的 digest 标签区改为显示 `aggregated_tags`。

推荐文案风格：

- `索引标签`
- `来自 12 篇文章`
- tag chip 上可附 `x3` / `3 篇` 之类的命中提示

重点是让用户理解：

- 这不是 digest 自己被单独打的标签
- 这是 digest 覆盖内容的索引

### 4. Topic Graph UI Changes

`/topics` 页面需要同步调整：

- timeline 中每条 digest 显示 `aggregated_tags`
- hotspot digests 不再显示空 tag 区
- topic detail 对应的 digest summary 统一显示 `aggregated_tags`
- 打开 digest modal 时，先展示 digest-level 聚合 tags，再展示 article list
- 打开 article preview 时展示 article-level tags

建议做两层语义：

- `digest-level aggregated tags`：表示日报/摘要覆盖主题索引
- `article-level tags`：表示单篇 article 归属

### 5. Topic-Aware Highlighting

在 topic graph 页面中，如果当前已选中某个 topic slug：

- digest 的 `aggregated_tags` 中命中该 slug 的 chip 做高亮
- article 的 `tags` 中命中该 slug 的 chip 也做高亮

这样用户能直观看到：

- 当前 topic 如何落在这份 digest 上
- 又具体落在哪些文章上

## UX Principles

- 不把 digest 伪装成独立 topic 对象
- 不让标签信息只停留在 `/topics`
- 同一篇 article 在任何入口打开，看到的 tag 结果都一致
- 默认轻量展示，避免标签过多时压垮页面

## Testing Strategy

### Backend

- article detail 返回 tags
- digest 聚合 tags 去重正确
- 同一 tag 命中多篇 article 时 `article_count` 正确
- topic graph 的 hotspot digests 和 topic detail digests 都返回聚合 tags

### Frontend

- `ArticleContentView` 在有/无 tags 场景都正常
- digest 详情 / 弹窗能显示聚合 tags
- topic graph timeline / digest modal / article preview 都能显示 tags
- 当前 topic 命中高亮正常

### Manual Verification

- 主阅读页打开一篇已打标签文章
- digest 页面打开一条 digest，再点文章弹窗
- `/topics` 中点击热点 tag，查看 digests 和 article preview

## Risks And Guardrails

- 若直接删掉所有旧 `topics` 使用点，回归面会偏大；应先补 `aggregated_tags` 再逐步切前端消费
- digest 聚合 tag 数量可能很多，前端必须默认截断
- article 详情接口新增 `tags` 后，前端类型映射必须同步，不然容易出现空展示或类型不匹配

## Success Criteria

- digest 不再被产品和代码语义当成“单独打 tag 的对象”
- topic graph 中 digest 的 tag 来源统一为 article 聚合 tags
- 标准 article 详情接口带 tags
- 主阅读页、digest 弹窗、topic graph 文章预览都能通用显示 article tags
- `/topics` 页面 digest 区和 article 区能同时表达聚合索引与单篇归属

## 后续补充：文章打标签主路径

后续实现将文章打标签策略进一步固定为：

- 普通 refresh：文章入库后立即打标签
- `Firecrawl + 自动补全`：补全成功后重打标签
- summary 阶段：只对没有 article tags 的文章做兜底补齐
- 前端文章详情：提供“手动打标签 / 重新打标签”按钮，并显示进行中的状态文案
