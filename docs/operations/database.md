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
- `scheduler_tasks`
- `ai_settings`
- `reading_behaviors`
- `user_preferences`

Digest 相关表由独立迁移负责。

## 常用命令

```bash
cd backend-go
go run cmd/migrate/main.go check
go run cmd/migrate/main.go migrate
go run cmd/migrate-digest/main.go
```

## 说明

- 当前项目以 Go 后端为主
- 文档以当前 checkout 里的真实数据库逻辑为准
- 不再把不存在的历史后端当作正式运行依赖描述
