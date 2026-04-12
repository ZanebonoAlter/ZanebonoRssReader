# 主题图谱 Topic Graph

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/topic-graph/:type` | 获取图谱 |
| GET | `/api/topic-graph/topic/:slug` | 主题详情 |
| GET | `/api/topic-graph/by-category` | 按类别分组 |
| GET | `/api/topic-graph/topic/:slug/articles` | 主题关联文章 |
| GET | `/api/topic-graph/tag/:slug/digests` | 标签关联 Digest |
| GET | `/api/topic-graph/tag/:slug/pending-articles` | 未收录文章 |

---

### GET /api/topic-graph/:type

`type` 如 `daily`。

| 参数 | 类型 | 说明 |
|------|------|------|
| `date` | string | `YYYY-MM-DD` |
| `category_id` | uint | 按分类 |
| `feed_id` | uint | 按订阅源 |

```json
{
  "success": true,
  "data": {
    "type": "daily",
    "anchor_date": "2024-01-15",
    "period_label": "2024-01-15 当日",
    "topic_count": 10,
    "summary_count": 25,
    "feed_count": 5,
    "top_topics": [
      { "slug": "ai-agent", "label": "AI Agent", "category": "keyword", "score": 5.2 }
    ],
    "nodes": [...],
    "edges": [...]
  }
}
```

### GET /api/topic-graph/topic/:slug

查询参数：`type`（默认 `daily`）、`date`、`category_id`、`feed_id`

```json
{
  "success": true,
  "data": {
    "topic": { "slug": "ai-agent", "label": "AI Agent", "category": "keyword" },
    "articles": [...],
    "total_articles": 50,
    "related_tags": [...],
    "summaries": [...],
    "history": [...],
    "related_topics": [...],
    "search_links": { "youtube_videos": "...", "youtube_live": "..." },
    "app_links": { "digest_view": "/digest/daily", "topic_graph": "/topics" }
  }
}
```

### GET /api/topic-graph/by-category

返回标签按 event/person/keyword 分组。

查询参数：`type`、`date`、`category_id`、`feed_id`

```json
{
  "success": true,
  "data": {
    "events": [{ "slug": "...", "label": "...", "category": "event", "score": 3.5 }],
    "people": [{ "slug": "...", "label": "...", "category": "person", "score": 2.8 }],
    "keywords": [{ "slug": "...", "label": "...", "category": "keyword", "score": 5.2 }]
  }
}
```

### GET /api/topic-graph/topic/:slug/articles

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `type` | string | daily | 窗口类型 |
| `date` | string | - | 锚点日期 |
| `page` | int | 1 | 页码 |
| `page_size` | int | 15 | 上限 100 |

```json
{
  "success": true,
  "data": {
    "articles": [
      {
        "id": "123",
        "title": "文章标题",
        "summary": "...",
        "pub_date": "2024-01-15T10:30:00Z",
        "feed_name": "Feed名称",
        "feed_id": "1",
        "link": "https://...",
        "tags": [{ "slug": "ai-agent", "label": "AI Agent", "category": "keyword" }],
        "image_url": "https://..."
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 15
  }
}
```

### GET /api/topic-graph/tag/:slug/digests

查询参数：`type`（默认 `daily`）、`date`、`limit`（默认 `20`，上限 100）

### GET /api/topic-graph/tag/:slug/pending-articles

指定标签下未收录到任何 Digest 的文章。

---

## 主题分析 Topic Analysis

路由注册在 `/api/topic-graph/analysis` 下：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/topic-graph/analysis` | 按查询参数获取分析 |
| GET | `/api/topic-graph/analysis/status` | 按查询参数获取状态 |
| POST | `/api/topic-graph/analysis/rebuild` | 按查询参数重建 |
| POST | `/api/topic-graph/analysis/retry` | 同 rebuild |
| GET | `/api/topic-graph/analysis/:tagID/:analysisType` | 获取指定分析 |
| POST | `/api/topic-graph/analysis/:tagID/:analysisType/rebuild` | 重建指定分析 |
| GET | `/api/topic-graph/analysis/:tagID/:analysisType/status` | 获取状态 |

### 查询参数方式

GET `/api/topic-graph/analysis`：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tag_id` | uint | 是 | 标签 ID |
| `analysis_type` | string | 是 | 分析类型 |
| `windowType` | string | 否 | 窗口类型 |
| `anchorDate` | string | 否 | `YYYY-MM-DD` |

### 路径参数方式

GET `/api/topic-graph/analysis/:tagID/:analysisType`：

| 参数 | 说明 |
|------|------|
| `tagID` | 标签 ID |
| `analysisType` | 分析类型 |

查询参数：`windowType`、`anchorDate`（或 `date`）

### 分析状态响应

```json
{
  "success": true,
  "data": {
    "status": "processing",
    "progress": 65,
    "error": null,
    "result": null
  }
}
```
