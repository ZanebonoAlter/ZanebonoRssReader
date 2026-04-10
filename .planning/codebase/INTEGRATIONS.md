# 外部集成

**分析日期:** 2026-04-10

## Web 内容抓取

### Firecrawl
- **用途:** 文章全文内容抓取和 Markdown 转换
- **SDK/客户端:** 自研 HTTP client (`backend-go/internal/domain/contentprocessing/firecrawl_service.go`)
- **认证:** Bearer token via `API_KEY` 配置
- **关键文件:**
  - `backend-go/internal/domain/contentprocessing/firecrawl_service.go` - Firecrawl 服务封装
  - `backend-go/internal/domain/contentprocessing/firecrawl_config.go` - 配置管理
  - `backend-go/internal/domain/contentprocessing/firecrawl_job_queue.go` - 任务队列
  - `backend-go/internal/jobs/firecrawl.go` - 调度器实现
- **API 端点:** `/v1/scrape`
- **输出格式:** Markdown + HTML
- **前端集成:** `front/app/api/firecrawl.ts`, `front/app/features/articles/composables/useContentCompletion.ts`

### Crawl4AI
- **用途:** 备选内容抓取服务
- **SDK/客户端:** 自研 HTTP client (`backend-go/internal/domain/contentprocessing/crawl4ai_client.go`)
- **默认地址:** `http://localhost:11235` (通过 `CRAWL_SERVICE_URL` 配置)
- **超时:** 60秒

## AI 智能服务

### AI 摘要生成
- **用途:** 文章内容智能摘要和重写
- **SDK/客户端:** 自研 OpenAI-compatible client (`backend-go/internal/platform/ai/service.go`)
- **API 格式:** OpenAI Chat Completions API (`/chat/completions`)
- **认证:** Bearer token
- **超时:** 120秒
- **最大 Token:** 16000
- **关键文件:**
  - `backend-go/internal/domain/summaries/ai_handler.go` - 摘要处理
  - `backend-go/internal/domain/summaries/ai_prompt_builder.go` - Prompt 构建
  - `backend-go/internal/jobs/auto_summary.go` - 自动摘要调度

### AI Router (多模型路由)
- **用途:** 支持多个 AI provider 的路由和 fallback
- **Provider 类型:** OpenAI-compatible, Ollama
- **关键文件:** `backend-go/internal/platform/airouter/router.go`
- **功能:** 自动 fallback、延迟日志、调用统计

### 主题分析 AI
- **用途:** 文章主题提取、标签生成、embedding
- **关键文件:**
  - `backend-go/internal/domain/topicextraction/` - 主题提取
  - `backend-go/internal/domain/topicanalysis/` - 分析服务
  - `backend-go/internal/domain/topicgraph/` - 知识图谱

### OpenNotebook
- **用途:** AI 摘要增强和远程 notebook 同步
- **客户端:** `backend-go/internal/platform/opennotebook/client.go`
- **API 端点:** `/api/transformations`, `/api/transformations/execute`

## 实时通信

### WebSocket
- **用途:** 摘要进度推送、Firecrawl 进度推送
- **实现:** Gorilla WebSocket (`backend-go/internal/platform/ws/hub.go`)
- **端点:** `ws://localhost:5000/ws`
- **消息类型:**
  - `progress` - 摘要进度
  - `firecrawl_progress` - 抓取进度
- **前端连接:** `front/app/features/summaries/composables/useSummaryWebSocket.ts`
- **重连策略:** 最大 5 次，延迟 3000ms

## 数据导出

### Obsidian
- **用途:** 每日/每周 digest 导出到 Obsidian vault
- **导出器:** `backend-go/internal/domain/digest/obsidian.go`
- **导出格式:** Markdown 文件
- **目录结构:** `vault/Daily/{category}/{date}-日报.md`

### Feishu (飞书)
- **用途:** digest 通知推送到飞书群
- **通知器:** `backend-go/internal/domain/digest/feishu.go`
- **消息类型:** text, interactive (卡片)
- **认证:** Webhook URL

## RSS 数据源

### RSS 解析
- **用途:** 订阅源内容解析
- **库:** mmcdole/gofeed (`backend-go/internal/domain/feeds/rss_parser.go`)
- **支持格式:** RSS, Atom
- **OPML:** `backend-go/internal/domain/feeds/opml.go`

## HTTP API 规范

### 前端 API Client
- **位置:** `front/app/api/client.ts`
- **封装:** `ApiClient` class，统一返回 `{ success, data, error, message, pagination }`
- **Trace 支持:** W3C traceparent header
- **方法:** get, post, put, delete, upload, download

### 后端响应格式
- **Handler 模式:** `gin.H{"success": bool, "data"|"error"|"message": ...}`
- **JSON 字段命名:** snake_case
- **路由入口:** `backend-go/internal/app/router.go`

## 调度任务

### 后台调度器
- **框架:** robfig/cron v3
- **调度器列表:**
  - AutoRefresh - 自动刷新订阅源
  - AutoSummary - 自动生成摘要
  - PreferenceUpdate - 用户偏好更新
  - ContentCompletion - 内容补全
  - Firecrawl - 全文抓取
  - Digest - 日报/周报生成
- **入口:** `backend-go/internal/app/runtime.go`

## 数据存储

### PostgreSQL
- **迁移路径:** SQLite → PostgreSQL
- **迁移工具:** `backend-go/cmd/migrate-db/main.go`
- **配置参数:**
  - `--postgres-dsn` 或环境变量 `DATABASE_DSN`
  - 连接池: max_idle=5, max_open=25, lifetime=60min

### Redis (可选)
- **用途:** 主题分析任务队列
- **环境变量:** `REDIS_URL`
- **回退:** 内存队列 (Redis 不可用时)

## 环境变量清单

**必需:**
- `DATABASE_DRIVER` - 数据库驱动 (sqlite/postgres)
- `DATABASE_DSN` - 数据库连接字符串

**可选:**
- `SERVER_PORT` - 服务端口 (默认 5000)
- `SERVER_MODE` - Gin 模式 (debug/release)
- `CORS_ORIGINS` - CORS 允许来源
- `REDIS_URL` - Redis 连接 (用于分析队列)
- `CRAWL_SERVICE_URL` - Crawl4AI 服务地址
- `NUXT_PUBLIC_API_ORIGIN` - 前端 API origin
- `NUXT_PUBLIC_API_BASE` - 前端 API base path
- `API_INTERNAL_BASE` - 内部 API base

---

*集成审计: 2026-04-10*