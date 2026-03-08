# 后端架构

## 技术栈

- Go
- Gin
- GORM
- SQLite

## 当前入口

- 服务入口：`backend-go/cmd/server/main.go`
- 数据库初始化：`backend-go/pkg/database/db.go`
- 配置：`backend-go/configs/config.yaml`

## 当前问题

- 代码按 `handlers/services/models/schedulers` 横向切开
- 理解一个功能需要跨很多目录
- 启动装配和路由注册都堆在主入口附近

## 目标结构

```text
backend-go/
├── cmd/
├── internal/
│   ├── app/
│   ├── platform/
│   ├── domain/
│   └── jobs/
```

## 目录规则

- `app/` - 服务装配、路由注册、启动流程
- `platform/` - 配置、数据库、中间件、WebSocket
- `domain/` - 业务域代码
- `jobs/` - 定时任务执行壳

## 主要业务域

- `categories`
- `feeds`
- `articles`
- `summaries`
- `preferences`
- `content-processing`
- `digest`

## 迁移原则

- 先抽 `internal/app`
- 再搬平台代码
- 再按领域收拢 handler、service、model、test
- API 路径先保持不变
