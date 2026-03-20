# 热点题材反向追溯日报 - 实现总结

## 实现概述

已成功实现方案二：**从文章标签反向追溯到包含这些文章的日报**。

## 核心功能

### 1. 后端 API 实现

**新增文件：**
- `backend-go/internal/domain/topicgraph/hotspot_digests.go` - 核心业务逻辑

**新增 API 端点：**
```
GET /api/topic-graph/tag/:slug/digests
```

**查询参数：**
- `type`: 时间窗口类型 (daily/weekly)
- `date`: 锚点日期 (YYYY-MM-DD)
- `limit`: 返回数量限制 (默认20)

**反向追溯逻辑：**
```
Tag Slug → Topic Tag ID → Article IDs (via ArticleTopicTag) 
  → Digests containing those articles (via AISummary.articles JSON field)
```

### 2. 前端实现

**API 层更新：**
- `front/app/api/topicGraph.ts`
  - 新增 `HotspotDigestCard` 类型
  - 新增 `HotspotDigestsResponse` 类型
  - 新增 `getDigestsByArticleTag()` API 方法

**组件层更新：**
- `front/app/features/topic-graph/components/TopicGraphPage.vue`
  - 新增热点题材日报状态管理
  - 新增 `loadHotspotDigests()` 方法
  - 更新 `handleTagSelect()` 方法加载关联日报

### 3. 路由配置

**更新文件：**
- `backend-go/internal/app/router.go`
  - 新增路由：`/topic-graph/tag/:slug/digests`

## 数据结构

### HotspotDigestCard
```typescript
{
  id: number
  title: string
  summary: string
  feed_name: string
  feed_color: string
  category_name: string
  article_count: number
  created_at: string
  matched_articles?: Array<{
    id: number
    title: string
  }>
}
```

## 使用流程

1. **用户点击热点题材标签**
   - 触发 `handleTagSelect()`
   - 设置选中的标签状态

2. **加载关联日报**
   - 调用 `loadHotspotDigests()`
   - 发送请求到 `GET /api/topic-graph/tag/{slug}/digests`
   - 后端执行反向追溯查询

3. **展示结果**
   - 在热点题材区域下方或侧边展示关联日报列表
   - 每个日报显示标题、摘要、来源Feed、匹配的文章数等信息

## 优势

1. **精准关联**：基于文章级别的标签关联，确保日报与标签的相关性
2. **细粒度**：可以展示具体匹配的文章，而不仅仅是日报
3. **可追溯**：用户可以了解为什么某个日报会出现在特定标签下
4. **符合初衷**：保持文章标签的细粒度关联，而非仅基于日报标签

## 测试验证

- ✅ 后端编译成功
- ✅ 前端构建成功
- ✅ TypeScript 类型检查通过

## 后续优化建议

1. **缓存优化**：对热点标签的日报结果进行缓存
2. **分页加载**：当日报数量较多时支持分页
3. **排序选项**：支持按时间、匹配文章数等排序
4. **UI优化**：添加加载状态和空状态提示