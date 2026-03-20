# Topic Graph API

## 分析相关接口

### 获取分析结果

```http
GET /api/topic-graph/analysis/:tagID/:analysisType
```

参数:

- `tagID`: 标签 ID
- `analysisType`: 分析类型（event | person | keyword）
- `windowType`: 时间窗口类型（daily | weekly）
- `anchorDate`: 锚点日期（YYYY-MM-DD）

响应:

```json
{
  "success": true,
  "data": {
    "id": 1,
    "topic_tag_id": 123,
    "analysis_type": "event",
    "window_type": "daily",
    "anchor_date": "2024-01-01",
    "summary_count": 10,
    "payload_json": "{...}",
    "source": "ai",
    "version": 1
  }
}
```

### 触发重新分析

```http
POST /api/topic-graph/analysis/:tagID/:analysisType/rebuild
```

请求体:

```json
{
  "windowType": "daily",
  "anchorDate": "2024-01-01"
}
```

### 获取分析状态

```http
GET /api/topic-graph/analysis/:tagID/:analysisType/status
```

响应:

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

## 主题日报接口

### 获取主题日报列表

获取指定主题下的日报列表，支持分页和时间筛选。

```http
GET /api/topic-graph/topic/:slug/articles
```

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slug | string | 是 | 主题 slug，如 `ai-agent`、`openai` |

#### 查询参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | integer | 否 | 1 | 页码，从 1 开始 |
| page_size | integer | 否 | 15 | 每页数量，最大 100 |
| type | string | 否 | daily | 时间窗口类型：daily 或 weekly |
| date | string | 否 | 今天 | 锚点日期，格式 YYYY-MM-DD |

#### 响应

```json
{
  "success": true,
  "data": {
    "articles": [
      {
        "id": "123",
        "title": "文章标题",
        "summary": "文章摘要内容...",
        "pub_date": "2024-01-15T10:30:00Z",
        "feed_name": "Feed名称",
        "feed_id": "1",
        "link": "https://example.com/article",
        "tags": [
          {
            "slug": "ai-agent",
            "label": "AI Agent",
            "category": "keyword"
          }
        ],
        "image_url": "https://example.com/image.jpg"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 15
  }
}
```

#### 错误响应

**缺少 slug 参数**

```json
{
  "success": false,
  "error": "slug is required"
}
```

**主题不存在**

```json
{
  "success": false,
  "error": "topic not found: ..."
}
```

#### 示例

```bash
# 获取第一页
curl "http://localhost:5000/api/topic-graph/topic/ai-agent/articles?page=1&page_size=15&type=daily&date=2024-01-15"

# 获取第二页
curl "http://localhost:5000/api/topic-graph/topic/openai/articles?page=2&page_size=20&type=weekly&date=2024-01-15"
```

## 图谱接口

### 获取主题图谱

```http
GET /api/topic-graph/:type
```

参数:

- `type`: 图谱类型（daily | weekly）
- `date`: 锚点日期（YYYY-MM-DD）

响应:

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
      {
        "slug": "ai-agent",
        "label": "AI Agent",
        "category": "keyword",
        "score": 5.2
      }
    ],
    "nodes": [...],
    "edges": [...]
  }
}
```

### 获取主题详情

```http
GET /api/topic-graph/topic/:slug
```

参数:

- `slug`: 主题 slug
- `type`: 时间窗口类型（daily | weekly）
- `date`: 锚点日期（YYYY-MM-DD）

响应:

```json
{
  "success": true,
  "data": {
    "topic": {
      "slug": "ai-agent",
      "label": "AI Agent",
      "category": "keyword"
    },
    "articles": [...],
    "total_articles": 50,
    "related_tags": [...],
    "summaries": [...],
    "history": [...],
    "related_topics": [...],
    "search_links": {
      "youtube_videos": "https://...",
      "youtube_live": "https://..."
    },
    "app_links": {
      "digest_view": "/digest/daily",
      "topic_graph": "/topics"
    }
  }
}
```

### 获取分类主题列表

```http
GET /api/topic-graph/topics-by-category
```

参数:

- `type`: 时间窗口类型（daily | weekly）
- `date`: 锚点日期（YYYY-MM-DD）

响应:

```json
{
  "success": true,
  "data": {
    "events": [
      {
        "slug": "gpt-5-launch",
        "label": "GPT-5 发布",
        "category": "event",
        "score": 3.5
      }
    ],
    "people": [
      {
        "slug": "sam-altman",
        "label": "Sam Altman",
        "category": "person",
        "score": 2.8
      }
    ],
    "keywords": [
      {
        "slug": "ai-agent",
        "label": "AI Agent",
        "category": "keyword",
        "score": 5.2
      }
    ]
  }
}
```
