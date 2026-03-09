# 数据库说明

## 当前数据库

- 数据库类型：SQLite
- 默认文件：`backend-go/rss_reader.db`

## 初始化方式

后端启动时会自动初始化缺失表。核心逻辑在：

- `backend-go/pkg/database/db.go`

## 当前核心表

- `categories`
- `feeds`
- `articles`
- `ai_summaries`
- `ai_summary_feeds`
- `ai_summary_queue`
- `scheduler_tasks`
- `ai_settings`
- `reading_behaviors`
- `user_preferences`
- `digest_configs`

其中：

- 核心表和补充字段保证逻辑在 `backend-go/pkg/database/db.go`
- digest 相关表迁移在 `backend-go/internal/digest/`

## 当前 schema 特点

这个项目现在不是只靠单次 `AutoMigrate`。

实际流程是：

1. 启动时初始化数据库连接
2. 执行 `EnsureTables()` 保证缺表存在
3. 对旧表补充新增字段
4. 再由 digest 迁移补 digest 自己的表

也就是说，数据库演进是“GORM + 手写 SQL 保底 + 独立子系统迁移”三种方式并存。

## 当前新增能力相关字段

### `feeds`

- `content_completion_enabled`
- `completion_on_refresh`
- `max_completion_retries`
- `firecrawl_enabled`

### `articles`

- `image_url`
- `content_status`
- `full_content`
- `content_fetched_at`
- `completion_attempts`
- `completion_error`
- `ai_content_summary`
- `firecrawl_status`
- `firecrawl_error`
- `firecrawl_content`
- `firecrawl_crawled_at`

## 常用命令

```bash
cd backend-go
go run cmd/server/main.go
go run cmd/migrate-digest/main.go
go run cmd/test-digest/main.go
```

## 说明

- 当前项目以 Go 后端为主
- 文档以当前 checkout 里的真实数据库逻辑为准
- 不再把不存在的历史后端当作正式运行依赖描述
- 数据库文档以 `docs/operations/database.md` 为准，不再依赖 `backend-go/DATABASE.md` 的旧叙述
