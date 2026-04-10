<!-- generated-by: gsd-doc-writer -->

# API Reference

Go 后端 REST API 完整参考文档。所有接口基路径为 `/api`，后端默认运行在 `http://localhost:5000`。

## 通用约定

### 响应格式

所有接口返回 JSON，统一信封结构：

```json
{
  "success": true,
  "data": { ... },
  "message": "操作描述（可选）"
}
```

错误响应：

```json
{
  "success": false,
  "error": "错误描述"
}
```

### 分页

列表接口支持分页，查询参数 `page`（默认 `1`）和 `per_page`（默认 `20`）。响应包含 `pagination` 对象：

```json
{
  "success": true,
  "data": [ ... ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "pages": 8
  }
}
```

### 认证

本项目为个人/单用户部署，不包含认证系统。

### WebSocket

实时通信端点：`ws://localhost:5000/ws`，用于 AI 总结进度推送等场景。

---

## 目录

- [系统信息](#系统信息)
- [分类 Categories](#分类-categories)
- [订阅 Feeds](#订阅-feeds)
- [文章 Articles](#订阅-articles)
- [AI 总结 Summaries](#ai-总结-summaries)
- [AI 管理 Admin](#ai-管理-admin)
- [OPML 导入导出](#opml-导入导出)
- [定时任务 Schedulers](#定时任务-schedulers)
- [任务状态 Tasks](#任务状态-tasks)
- [阅读行为 Reading Behavior](#阅读行为-reading-behavior)
- [用户偏好 User Preferences](#用户偏好-user-preferences)
- [内容补全 Content Completion](#内容补全-content-completion)
- [Firecrawl](#firecrawl)
- [自动总结 Auto Summary](#自动总结-auto-summary)
- [主题图谱 Topic Graph](#主题图谱-topic-graph)
- [主题分析 Topic Analysis](#主题分析-topic-analysis)
- [Digest 汇总](#digest-汇总)
- [链路追踪 Traces](#链路追踪-traces)

---

## 系统信息

### GET /

返回 API 名称和版本。

**响应** `200`：

```json
{
  "name": "RSS Reader API (Go)",
  "version": "1.0.0",
  "endpoints": { ... }
}
```

### GET /health

健康检查。

**响应** `200`：

```json
{
  "status": "healthy",
  "database": "connected"
}
```

---

## 分类 Categories

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/categories` | 获取所有分类 |
| POST | `/api/categories` | 创建分类 |
| PUT | `/api/categories/:category_id` | 更新分类 |
| DELETE | `/api/categories/:category_id` | 删除分类 |

### GET /api/categories

获取所有分类，按名称升序排列，附带每个分类下的订阅源数量。

**响应** `200`：

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "name": "技术",
      "slug": "a1b2c3d4",
      "icon": "folder",
      "color": "#6366f1",
      "description": "技术相关订阅",
      "created_at": "2025-01-15 10:30:00",
      "feed_count": 5
    }
  ]
}
```

### POST /api/categories

创建分类。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 分类名称（唯一） |
| `slug` | string | 否 | URL slug，留空自动生成 |
| `icon` | string | 否 | 图标，默认 `folder` |
| `color` | string | 否 | 颜色，默认 `#6366f1` |
| `description` | string | 否 | 描述 |

**响应** `201`：返回创建的分类对象。`409`：同名分类已存在。

### PUT /api/categories/:category_id

更新分类。只更新请求体中提供的字段。

**路径参数**：`category_id`（uint）

**请求体**：同创建，但所有字段可选。

**响应** `200`：返回更新后的分类。`404`：分类不存在。

### DELETE /api/categories/:category_id

删除分类。

**响应** `200`：

```json
{ "success": true, "message": "Category deleted successfully" }
```

---

## 订阅 Feeds

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/feeds` | 获取订阅列表 |
| GET | `/api/feeds/:feed_id` | 获取单个订阅 |
| POST | `/api/feeds` | 创建订阅 |
| PUT | `/api/feeds/:feed_id` | 更新订阅 |
| DELETE | `/api/feeds/:feed_id` | 删除订阅 |
| POST | `/api/feeds/:feed_id/refresh` | 刷新单个订阅 |
| POST | `/api/feeds/fetch` | 预览 Feed URL |
| POST | `/api/feeds/refresh-all` | 刷新所有订阅 |

### GET /api/feeds

**查询参数**：

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `page` | int | 1 | 页码 |
| `per_page` | int | 20 | 每页条数，≥10000 返回全部 |
| `category_id` | int | - | 按分类过滤 |
| `uncategorized` | string | - | 设为 `true` 查询未分类订阅 |

**响应** `200`：带分页的订阅列表，每个订阅包含 `article_count` 和 `unread_count`。

### GET /api/feeds/:feed_id

获取单个订阅详情，包含文章统计。

### POST /api/feeds

创建订阅。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `url` | string | 是 | RSS feed URL |
| `title` | string | 否 | 标题，默认 `Untitled Feed` |
| `description` | string | 否 | 描述 |
| `category_id` | uint\* | 否 | 分类 ID |
| `icon` | string | 否 | 图标，默认 `mdi:rss` |
| `color` | string | 否 | 颜色，默认 `#8b5cf6` |
| `max_articles` | int | 否 | 最大文章数，默认 `100` |
| `refresh_interval` | int | 否 | 刷新间隔（分钟），默认 `60` |
| `ai_summary_enabled` | bool | 否 | 启用 AI 总结 |
| `article_summary_enabled` | bool | 否 | 启用文章级总结 |
| `completion_on_refresh` | bool | 否 | 刷新时自动补全 |
| `max_completion_retries` | int | 否 | 补全最大重试次数 |
| `firecrawl_enabled` | bool | 否 | 启用 Firecrawl |

**响应** `201`：返回创建的订阅。`409`：URL 已存在。

### PUT /api/feeds/:feed_id

更新订阅。只有请求体中明确提供的字段会被更新，布尔字段需要显式包含才会从 `false` 改为 `true`。

### DELETE /api/feeds/:feed_id

删除订阅及其关联文章。

### POST /api/feeds/:feed_id/refresh

触发后台刷新指定订阅。立即返回 `202 Accepted`。

**响应** `202`：

```json
{ "success": true, "message": "Started refreshing feed in background" }
```

### POST /api/feeds/fetch

预览 RSS feed URL，返回标题和描述。

**请求体**：

```json
{ "url": "https://example.com/feed.xml" }
```

**响应** `200`：

```json
{
  "success": true,
  "data": { "title": "Example Blog", "description": "A blog about..." }
}
```

### POST /api/feeds/refresh-all

触发后台刷新所有订阅。

**响应** `202`：

```json
{
  "success": true,
  "message": "Started refreshing all feeds in background",
  "data": { "total_feeds": 15 }
}
```

---

## 文章 Articles

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/articles/stats` | 获取文章统计 |
| GET | `/api/articles` | 获取文章列表 |
| GET | `/api/articles/:article_id` | 获取单篇文章 |
| POST | `/api/articles/:article_id/tags` | 重新打标签 |
| PUT | `/api/articles/:article_id` | 更新文章 |
| PUT | `/api/articles/bulk-update` | 批量更新文章 |

### GET /api/articles/stats

**响应** `200`：

```json
{
  "success": true,
  "data": { "total": 1500, "unread": 320, "favorite": 45 }
}
```

### GET /api/articles

**查询参数**：

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `page` | int | 1 | 页码 |
| `per_page` | int | 20 | 每页条数，上限 100 |
| `feed_id` | int | - | 按订阅源过滤 |
| `category_id` | int | - | 按分类过滤 |
| `uncategorized` | string | - | 设为 `true` 过滤未分类 |
| `read` | string | - | `true`/`false` 按已读状态过滤 |
| `favorite` | string | - | `true`/`false` 按收藏状态过滤 |
| `search` | string | - | 按标题或描述模糊搜索 |
| `start_date` | string | - | 起始日期 `YYYY-MM-DD` |
| `end_date` | string | - | 截止日期 `YYYY-MM-DD` |

文章按发布日期降序排列。每篇文章包含 `tag_count` 字段。

### GET /api/articles/:article_id

获取单篇文章，附带标签列表。

**响应** `200`：

```json
{
  "success": true,
  "data": {
    "id": 42,
    "feed_id": 1,
    "category_id": 2,
    "title": "文章标题",
    "description": "...",
    "content": "...",
    "link": "https://...",
    "image_url": "https://...",
    "pub_date": "2025-03-10 08:00:00",
    "author": "...",
    "read": false,
    "favorite": false,
    "summary_status": "complete",
    "ai_content_summary": "...",
    "firecrawl_status": "completed",
    "firecrawl_content": "...",
    "tag_count": 3,
    "tags": [ ... ]
  }
}
```

### POST /api/articles/:article_id/tags

重新为文章生成标签。

**响应** `200`：

```json
{
  "success": true,
  "message": "文章标签已更新",
  "data": { "tag_count": 3, "tags": [ ... ] }
}
```

### PUT /api/articles/:article_id

更新文章的已读/收藏状态。

**请求体**：

```json
{ "read": true, "favorite": false }
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `read` | bool\* | 否 | 已读状态 |
| `favorite` | bool\* | 否 | 收藏状态 |

### PUT /api/articles/bulk-update

批量更新文章。至少提供一个更新字段和一个过滤条件。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `ids` | uint[] | 否 | 按文章 ID 列表过滤 |
| `feed_id` | uint\* | 否 | 按订阅源过滤 |
| `category_id` | uint\* | 否 | 按分类过滤 |
| `uncategorized` | bool\* | 否 | 过滤未分类 |
| `read` | bool\* | 否 | 设置已读状态 |
| `favorite` | bool\* | 否 | 设置收藏状态 |

过滤优先级：`ids` > `feed_id` > `category_id` > `uncategorized`。

**响应** `200`：`message` 字段为受影响的行数。

---

## AI 总结 Summaries

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/summaries` | 获取总结列表 |
| GET | `/api/summaries/:summary_id` | 获取单个总结 |
| DELETE | `/api/summaries/:summary_id` | 删除总结 |
| POST | `/api/summaries/queue` | 提交批量总结任务 |
| GET | `/api/summaries/queue/status` | 获取队列状态 |
| GET | `/api/summaries/queue/jobs/:job_id` | 获取队列任务详情 |

### GET /api/summaries

**查询参数**：

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `feed_id` | int | - | 按订阅源过滤 |
| `category_id` | int | - | 按分类过滤 |
| `page` | int | 1 | 页码 |
| `per_page` | int | 20 | 每页条数 |

### GET /api/summaries/:summary_id

获取单个总结详情，包含关联的 Feed 和 Category。

### DELETE /api/summaries/:summary_id

删除指定总结。

### POST /api/summaries/queue

提交批量 AI 总结任务。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `category_ids` | uint[] | 否 | 分类 ID 列表 |
| `feed_ids` | uint[] | 否 | 订阅源 ID 列表 |
| `time_range` | int | 否 | 时间范围（天） |
| `base_url` | string | 否 | AI 服务地址 |
| `api_key` | string | 否 | API Key |
| `model` | string | 否 | 模型名 |

`category_ids` 和 `feed_ids` 至少提供一个。

**响应** `202`：

```json
{ "success": true, "message": "Summary job queued successfully", "data": { ... } }
```

### GET /api/summaries/queue/status

获取当前队列批次状态。无活跃任务时 `data` 为 `null`。

### GET /api/summaries/queue/jobs/:job_id

获取指定队列任务详情。

---

## AI 管理 Admin

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/ai/settings` | 获取 AI 设置 |
| POST | `/api/ai/settings` | 保存 AI 设置 |
| POST | `/api/ai/summarize` | AI 总结文章 |
| POST | `/api/ai/test` | 测试 AI 连接 |
| GET | `/api/ai/providers` | 列出 AI 提供商 |
| POST | `/api/ai/providers` | 创建/更新提供商 |
| PUT | `/api/ai/providers/:provider_id` | 更新指定提供商 |
| DELETE | `/api/ai/providers/:provider_id` | 删除提供商 |
| GET | `/api/ai/routes` | 列出 AI 路由 |
| PUT | `/api/ai/routes/:capability` | 更新指定路由 |

### POST /api/ai/summarize

使用 AI 对文章内容生成总结。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `title` | string | 是 | 文章标题 |
| `content` | string | 是 | 文章内容 |
| `base_url` | string | 否 | 覆盖 AI 服务地址 |
| `api_key` | string | 否 | 覆盖 API Key |
| `model` | string | 否 | 覆盖模型名 |
| `language` | string | 否 | 语言，默认 `zh` |

若未提供 `base_url`/`api_key`/`model`，使用 AI Router 默认配置。

### POST /api/ai/test

测试 AI 连接。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `base_url` | string | 是 | 服务地址 |
| `model` | string | 是 | 模型名 |
| `api_key` | string | 否 | API Key（ollama 类型可省略） |
| `provider_type` | string | 否 | 提供商类型（如 `ollama`） |

### GET /api/ai/settings

获取当前 AI 设置（Provider/Router 配置）。

**响应** `200`：

```json
{
  "success": true,
  "data": {
    "base_url": "https://api.openai.com/v1",
    "model": "gpt-4o-mini",
    "provider_id": 1,
    "provider_name": "OpenAI",
    "route_name": "default",
    "time_range": 180,
    "api_key_configured": true
  }
}
```

### POST /api/ai/settings

保存 AI 设置。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `api_key` | string | 是 | API Key |
| `base_url` | string | 否 | 服务地址，默认 `https://api.openai.com/v1` |
| `model` | string | 否 | 模型名，默认 `gpt-4o-mini` |

### GET /api/ai/providers

列出所有 AI 提供商配置。

**响应** `200`：提供商列表，包含 `id`, `name`, `provider_type`, `base_url`, `model`, `enabled`, `timeout_seconds`, `max_tokens`, `temperature`, `api_key_configured` 等字段。

### POST /api/ai/providers

创建或更新 AI 提供商（按 name 匹配）。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 提供商名称 |
| `base_url` | string | 是 | 服务地址 |
| `model` | string | 是 | 模型名 |
| `api_key` | string | 否 | API Key |
| `provider_type` | string | 否 | 类型 |
| `enabled` | bool\* | 否 | 是否启用，默认 `true` |
| `timeout_seconds` | int | 否 | 超时秒数 |
| `max_tokens` | int\* | 否 | 最大 tokens |
| `temperature` | float64\* | 否 | 温度参数 |
| `metadata` | string | 否 | 附加元数据 |

### PUT /api/ai/providers/:provider_id

更新指定 ID 的提供商。请求体同上。`api_key` 仅在非空时更新。

### DELETE /api/ai/providers/:provider_id

删除提供商。若仍被路由引用，返回 `409 Conflict`。

### GET /api/ai/routes

列出所有 AI 路由及其关联的提供商。

### PUT /api/ai/routes/:capability

更新指定能力的路由。

**路径参数**：`capability`（如 `summary`, `article_completion`）

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `provider_ids` | uint[] | 是 | 关联的提供商 ID 列表 |
| `name` | string | 否 | 路由名称 |
| `enabled` | bool\* | 否 | 是否启用，默认 `true` |
| `description` | string | 否 | 路由描述 |

---

## OPML 导入导出

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/import-opml` | 导入 OPML 文件 |
| GET | `/api/export-opml` | 导出 OPML 文件 |

### POST /api/import-opml

上传 OPML 文件，自动创建分类和订阅。

**请求**：`multipart/form-data`，字段名 `file`，文件类型 `.opml` 或 `.xml`。

**响应** `200`：

```json
{
  "success": true,
  "message": "Imported successfully",
  "data": {
    "feeds_added": 10,
    "categories_added": 3,
    "errors": [],
    "async_update": true
  }
}
```

### GET /api/export-opml

导出所有订阅为 OPML XML 文件。

**响应** `200`：`Content-Type: text/xml`，`Content-Disposition: attachment; filename=feeds.opml`。

---

## 定时任务 Schedulers

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/schedulers/status` | 获取所有调度器状态 |
| GET | `/api/schedulers/:name/status` | 获取指定调度器状态 |
| POST | `/api/schedulers/:name/trigger` | 手动触发调度器 |
| POST | `/api/schedulers/:name/reset` | 重置调度器统计 |
| PUT | `/api/schedulers/:name/interval` | 更新调度器间隔 |

**支持的调度器名称**：

| 名称 | 别名 | 说明 |
|------|------|------|
| `auto_refresh` | - | 自动刷新 RSS 订阅 |
| `auto_summary` | - | 自动生成 AI 总结 |
| `preference_update` | - | 更新阅读偏好 |
| `content_completion` | `ai_summary` | 文章内容补全 |
| `firecrawl` | - | 自动 Firecrawl 全文抓取 |
| `digest` | - | Digest 日报/周报定时任务 |

### PUT /api/schedulers/:name/interval

**请求体**：

```json
{ "interval": 30 }
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `interval` | int | 是 | 新间隔（正整数，单位取决于调度器） |

---

## 任务状态 Tasks

### GET /api/tasks/status

获取所有后台任务的实时状态（总结队列、内容补全、Firecrawl）。

**响应** `200`：

```json
{
  "success": true,
  "data": {
    "queue_size": 3,
    "active_tasks": 2,
    "tasks": [
      { "type": "summary_queue", "status": "processing", "batch_id": "...", ... },
      { "type": "content_completion", "status": "processing", ... }
    ]
  }
}
```

---

## 阅读行为 Reading Behavior

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/reading-behavior/track` | 记录阅读行为 |
| POST | `/api/reading-behavior/track-batch` | 批量记录阅读行为 |
| GET | `/api/reading-behavior/stats` | 获取阅读统计 |

### POST /api/reading-behavior/track

记录单条阅读行为事件。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `article_id` | uint | 是 | 文章 ID |
| `feed_id` | uint | 是 | 订阅源 ID |
| `session_id` | string | 是 | 会话 ID |
| `event_type` | string | 是 | 事件类型（open, close, scroll, favorite 等） |
| `category_id` | uint\* | 否 | 分类 ID，留空自动填充 |
| `scroll_depth` | int | 否 | 滚动深度 |
| `reading_time` | int | 否 | 阅读时长（秒） |

### POST /api/reading-behavior/track-batch

批量记录。

**请求体**：

```json
{ "events": [ { ...同 track 格式... }, ... ] }
```

### GET /api/reading-behavior/stats

**响应** `200`：

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
| GET | `/api/user-preferences` | 获取偏好列表 |
| POST | `/api/user-preferences/update` | 触发偏好重算 |

### GET /api/user-preferences

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `type` | string | `feed` 仅订阅源偏好，`category` 仅分类偏好，留空返回全部 |

返回按偏好分数降序排列的列表，包含关联的 Feed/Category 信息。

### POST /api/user-preferences/update

触发偏好重算（后台执行）。

---

## 内容补全 Content Completion

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/content-completion/articles/:article_id/complete` | 补全单篇文章 |
| POST | `/api/content-completion/feeds/:feed_id/complete-all` | 补全订阅源下所有文章 |
| GET | `/api/content-completion/articles/:article_id/status` | 获取补全状态 |
| GET | `/api/content-completion/overview` | 获取补全总览 |

### POST /api/content-completion/articles/:article_id/complete

触发单篇文章内容补全（Firecrawl + AI 整理）。

**请求体**（可选）：

```json
{ "force": true }
```

### POST /api/content-completion/feeds/:feed_id/complete-all

补全指定订阅源下所有 `incomplete` 或 `failed` 状态的文章。

**响应** `200`：

```json
{
  "success": true,
  "completed": 5,
  "failed": 1,
  "total": 6
}
```

### GET /api/content-completion/articles/:article_id/status

**响应** `200`：

```json
{
  "success": true,
  "data": {
    "summary_status": "complete",
    "attempts": 1,
    "error": "",
    "summary_generated_at": "2025-03-10 10:00:00",
    "ai_content_summary": "...",
    "firecrawl_content": "...",
    "firecrawl_status": "completed",
    "firecrawl_error": "",
    "firecrawl_crawled_at": "2025-03-10 09:58:00"
  }
}
```

### GET /api/content-completion/overview

**响应** `200`：

```json
{
  "success": true,
  "data": {
    "pending_count": 10,
    "processing_count": 2,
    "completed_count": 500,
    "failed_count": 3,
    "blocked_count": 5,
    "total_count": 520,
    "ai_configured": true,
    "blocked_reasons": {
      "waiting_for_firecrawl_count": 3,
      "feed_disabled_count": 1,
      "ai_unconfigured_count": 0,
      "ready_but_missing_content_count": 1
    }
  }
}
```

---

## Firecrawl

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/firecrawl/article/:id` | 抓取单篇文章全文 |
| POST | `/api/firecrawl/feed/:id/enable` | 启用/禁用订阅源 Firecrawl |
| GET | `/api/firecrawl/status` | 获取 Firecrawl 全局状态 |
| POST | `/api/firecrawl/settings` | 保存 Firecrawl 设置 |

### POST /api/firecrawl/article/:id

抓取指定文章的全文内容（需要订阅源已启用 Firecrawl 且全局已开启）。

**响应** `200`：

```json
{
  "success": true,
  "data": {
    "firecrawl_content": "# markdown content...",
    "firecrawl_status": "completed",
    "summary_status": "incomplete"
  }
}
```

### POST /api/firecrawl/feed/:id/enable

**请求体**：

```json
{ "enabled": true }
```

启用时会将该订阅源下未抓取的文章标记为 `pending`。

### GET /api/firecrawl/status

**响应** `200`：

```json
{
  "success": true,
  "data": {
    "enabled": true,
    "api_url": "https://api.firecrawl.dev/v0",
    "mode": "scrape",
    "timeout": 60,
    "max_content_length": 50000,
    "api_key_configured": true
  }
}
```

### POST /api/firecrawl/settings

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `enabled` | bool | 是 | 是否启用 |
| `api_url` | string | 是 | API 地址 |
| `api_key` | string | 否 | API Key（留空保留现有） |
| `mode` | string | 否 | 模式，默认 `scrape` |
| `timeout` | int | 否 | 超时秒数，默认 `60` |
| `max_content_length` | int | 否 | 最大内容长度，默认 `50000` |

---

## 自动总结 Auto Summary

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/auto-summary/status` | 获取自动总结配置状态 |
| POST | `/api/auto-summary/config` | 更新自动总结配置 |

### GET /api/auto-summary/status

**响应** `200`：

```json
{
  "success": true,
  "data": {
    "enabled": true,
    "status": "configured",
    "base_url": "https://api.openai.com/v1",
    "model": "gpt-4o-mini",
    "provider_id": 1,
    "route_name": "default",
    "time_range": 180
  }
}
```

未配置时返回 `{ "enabled": false, "status": "not_configured" }`。

### POST /api/auto-summary/config

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `base_url` | string | 否 | AI 服务地址 |
| `api_key` | string | 否 | API Key |
| `model` | string | 否 | 模型名 |
| `time_range` | int | 否 | 时间范围（天），默认 `180` |

---

## 主题图谱 Topic Graph

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/topic-graph/:type` | 获取主题图谱 |
| GET | `/api/topic-graph/topic/:slug` | 获取主题详情 |
| GET | `/api/topic-graph/by-category` | 按类别获取标签分组 |
| GET | `/api/topic-graph/topic/:slug/articles` | 获取主题关联文章 |
| GET | `/api/topic-graph/tag/:slug/digests` | 获取标签关联 Digest |
| GET | `/api/topic-graph/tag/:slug/pending-articles` | 获取标签未收录文章 |

### GET /api/topic-graph/:type

**路径参数**：`type` — 图谱类型（如 `daily`）

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `date` | string | 锚点日期 `YYYY-MM-DD` |
| `category_id` | uint | 按分类过滤 |
| `feed_id` | uint | 按订阅源过滤 |

### GET /api/topic-graph/topic/:slug

**查询参数**：`type`（默认 `daily`）、`date`、`category_id`、`feed_id`

### GET /api/topic-graph/by-category

返回标签按类别（event, person, keyword）分组。

**查询参数**：`type`（默认 `daily`）、`date`、`category_id`、`feed_id`

### GET /api/topic-graph/topic/:slug/articles

**查询参数**：

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `type` | string | daily | 图谱类型 |
| `date` | string | - | 锚点日期 |
| `page` | int | 1 | 页码 |
| `page_size` | int | 15 | 每页条数，上限 100 |

### GET /api/topic-graph/tag/:slug/digests

**查询参数**：`type`（默认 `daily`）、`date`、`limit`（默认 `20`，上限 100）

### GET /api/topic-graph/tag/:slug/pending-articles

获取指定标签下未收录到任何 Digest 的文章。

---

## 主题分析 Topic Analysis

以下路由注册在 `/api/topic-graph/analysis` 下：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/topic-graph/analysis` | 按查询参数获取分析 |
| GET | `/api/topic-graph/analysis/status` | 按查询参数获取分析状态 |
| POST | `/api/topic-graph/analysis/rebuild` | 按查询参数重建分析 |
| POST | `/api/topic-graph/analysis/retry` | 同 rebuild |
| GET | `/api/topic-graph/analysis/:tagID/:analysisType` | 获取指定分析 |
| POST | `/api/topic-graph/analysis/:tagID/:analysisType/rebuild` | 重建指定分析 |
| GET | `/api/topic-graph/analysis/:tagID/:analysisType/status` | 获取指定分析状态 |

### 查询参数方式

**GET /api/topic-graph/analysis** 的查询参数：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tag_id` | uint | 是 | 标签 ID |
| `analysis_type` | string | 是 | 分析类型 |
| `windowType` | string | 否 | 窗口类型 |
| `anchorDate` | string | 否 | 锚点日期 `YYYY-MM-DD` |

### 路径参数方式

**GET /api/topic-graph/analysis/:tagID/:analysisType** 的路径参数：

| 参数 | 说明 |
|------|------|
| `tagID` | 标签 ID（uint） |
| `analysisType` | 分析类型 |

查询参数：`windowType`、`anchorDate`（或 `date`）

---

## Digest 汇总

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/digest/config` | 获取 Digest 配置 |
| PUT | `/api/digest/config` | 更新 Digest 配置 |
| GET | `/api/digest/status` | 获取 Digest 运行状态 |
| GET | `/api/digest/preview/:type` | 预览 Digest |
| POST | `/api/digest/run/:type` | 立即执行 Digest |
| GET | `/api/digest/open-notebook/config` | 获取 Open Notebook 配置 |
| PUT | `/api/digest/open-notebook/config` | 更新 Open Notebook 配置 |
| POST | `/api/digest/open-notebook/:type` | 发送到 Open Notebook |
| POST | `/api/digest/test-feishu` | 测试飞书推送 |
| POST | `/api/digest/test-obsidian` | 测试 Obsidian 写入 |

### GET /api/digest/config

**响应** `200`：

```json
{
  "success": true,
  "data": {
    "daily_enabled": false,
    "daily_time": "09:00",
    "weekly_enabled": false,
    "weekly_day": 1,
    "weekly_time": "09:00",
    "feishu_enabled": false,
    "feishu_webhook_url": "",
    "feishu_push_summary": true,
    "feishu_push_details": false,
    "obsidian_enabled": false,
    "obsidian_vault_path": "",
    "obsidian_daily_digest": true,
    "obsidian_weekly_digest": true
  }
}
```

### PUT /api/digest/config

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `daily_enabled` | bool | 是 | 启用日报 |
| `daily_time` | string | 是 | 日报时间 `HH:MM` |
| `weekly_enabled` | bool | 是 | 启用周报 |
| `weekly_day` | int | 是 | 周几（0-6，0=周日） |
| `weekly_time` | string | 是 | 周报时间 `HH:MM` |
| `feishu_enabled` | bool | 是 | 启用飞书推送 |
| `feishu_webhook_url` | string | 是 | 飞书 Webhook URL |
| `feishu_push_summary` | bool | 是 | 推送摘要 |
| `feishu_push_details` | bool | 是 | 推送详情卡片 |
| `obsidian_enabled` | bool | 是 | 启用 Obsidian 导出 |
| `obsidian_vault_path` | string | 是 | Vault 路径 |
| `obsidian_daily_digest` | bool | 是 | 导出日报 |
| `obsidian_weekly_digest` | bool | 是 | 导出周报 |

启用 `daily_enabled` 时 `daily_time` 必须为合法 `HH:MM` 格式；启用 `weekly_enabled` 时 `weekly_day` 须在 0-6。

### GET /api/digest/status

获取 Digest 调度器运行状态。

### GET /api/digest/preview/:type

**路径参数**：`type` — `daily` 或 `weekly`

**查询参数**：`date` — 锚点日期 `YYYY-MM-DD`（可选）

**响应** `200`：包含 `type`, `title`, `period_label`, `generated_at`, `anchor_date`, `category_count`, `summary_count`, `markdown`, `categories` 等。

### POST /api/digest/run/:type

**路径参数**：`type` — `daily` 或 `weekly`

**查询参数**：`date`（可选）

立即生成并推送 Digest（飞书、Obsidian、Open Notebook 按配置自动执行）。

**响应** `200`：

```json
{
  "success": true,
  "message": "已执行当前 digest 流程",
  "data": {
    "preview": { ... },
    "sent_to_feishu": true,
    "exported_to_obsidian": false,
    "sent_to_open_notebook": false
  }
}
```

### GET /api/digest/open-notebook/config

获取 Open Notebook 集成配置。

### PUT /api/digest/open-notebook/config

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `enabled` | bool | 是 | 是否启用 |
| `base_url` | string | 是 | Open Notebook 地址 |
| `api_key` | string | 否 | API Key |
| `model` | string | 否 | 模型名 |
| `target_notebook` | string | 否 | 目标笔记本 |
| `prompt_mode` | string | 否 | 提示模式，默认 `digest_summary` |
| `auto_send_daily` | bool | 是 | 自动发送日报 |
| `auto_send_weekly` | bool | 是 | 自动发送周报 |
| `export_back_to_obsidian` | bool | 是 | 导出到 Obsidian |

### POST /api/digest/open-notebook/:type

**路径参数**：`type` — `daily` 或 `weekly`

**查询参数**：`date`（可选）

手动发送 Digest 到 Open Notebook。

### POST /api/digest/test-feishu

测试飞书 Webhook 推送。

**请求体**（可选）：

```json
{ "webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/..." }
```

留空使用已保存的 Webhook URL。

### POST /api/digest/test-obsidian

测试 Obsidian 写入。

**请求体**（可选）：

```json
{ "vault_path": "/path/to/vault" }
```

留空使用已保存的路径。

---

## 链路追踪 Traces

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/traces` | 按 trace_id 查询 |
| GET | `/api/traces/recent` | 获取最近链路 |
| GET | `/api/traces/search` | 搜索链路 |
| GET | `/api/traces/stats` | 获取追踪统计 |
| GET | `/api/traces/:trace_id/timeline` | 获取链路时间线 |
| GET | `/api/traces/:trace_id/otlp` | 导出 OTLP 格式 |

### GET /api/traces

**查询参数**：`trace_id`（必填）

返回该 trace 下所有 span。

### GET /api/traces/recent

**查询参数**：`limit`（默认 `50`）

### GET /api/traces/search

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `operation` | string | 按操作名过滤 |
| `status` | string | `error` 查询错误链路 |
| `min_duration_ms` | int64 | 按最小耗时过滤（慢链路） |
| `limit` | int | 数量限制，默认 `50` |

优先级：`status=error` > `operation` > `min_duration_ms` > 默认 recent。

### GET /api/traces/stats

获取追踪统计汇总。

### GET /api/traces/:trace_id/timeline

返回 span 树形结构的时间线视图。

### GET /api/traces/:trace_id/otlp

以 OTLP JSON 格式导出指定 trace。
