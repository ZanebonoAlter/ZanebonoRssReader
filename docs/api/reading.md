# 阅读行为与用户偏好

## 阅读行为 Reading Behavior

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/reading-behavior/track` | 记录单条 |
| POST | `/api/reading-behavior/track-batch` | 批量记录 |
| GET | `/api/reading-behavior/stats` | 阅读统计 |

### POST /api/reading-behavior/track

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `article_id` | uint | 是 | 文章 ID |
| `feed_id` | uint | 是 | 订阅源 ID |
| `session_id` | string | 是 | 会话 ID |
| `event_type` | string | 是 | open, close, scroll, favorite 等 |
| `category_id` | uint | 否 | 留空自动填充 |
| `scroll_depth` | int | 否 | 滚动深度 |
| `reading_time` | int | 否 | 秒 |

### POST /api/reading-behavior/track-batch

```json
{ "events": [ { ...同 track 格式... }, ... ] }
```

### GET /api/reading-behavior/stats

```json
{
  "success": true,
  "data": {
    "total_articles": 200,
    "total_reading_time": 18000,
    "avg_reading_time": 90.5,
    "avg_scroll_depth": 72.3,
    "most_active_feed_id": 3,
    "most_active_category": 1
  }
}
```

---

## 用户偏好 User Preferences

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/user-preferences` | 偏好列表 |
| POST | `/api/user-preferences/update` | 触发偏好重算 |

### GET /api/user-preferences

| 参数 | 类型 | 说明 |
|------|------|------|
| `type` | string | `feed`/`category`，留空返回全部 |

按偏好分数降序，含关联 Feed/Category 信息。

### POST /api/user-preferences/update

后台执行偏好重算。
