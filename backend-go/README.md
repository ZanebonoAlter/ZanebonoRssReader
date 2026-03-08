# Backend Go

后端基于 Go、Gin、GORM 和 SQLite。

## 开发命令

```bash
go mod tidy
go run cmd/server/main.go
go test ./...
```

## 当前入口

- 服务入口：`backend-go/cmd/server/main.go`
- 配置文件：`backend-go/configs/config.yaml`
- 数据库逻辑：`backend-go/pkg/database/db.go`

## 架构文档

- 后端架构：`docs/architecture/backend-go.md`
- 数据流：`docs/architecture/data-flow.md`
- 数据库说明：`docs/operations/database.md`
- 开发流程：`docs/operations/development.md`

## 目录重组方向

后端将从 `handlers/services/models/schedulers` 横向分层，逐步迁移到：

- `internal/app/`
- `internal/platform/`
- `internal/domain/`
- `internal/jobs/`

目标是让一个功能的 handler、service、model、test 尽量待在同一领域目录里。
