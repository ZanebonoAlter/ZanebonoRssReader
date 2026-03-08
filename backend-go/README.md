# Backend Go

后端基于 Go、Gin、GORM 和 SQLite。

## 当前入口

- 服务入口：`backend-go/cmd/server/main.go`
- 路由装配：`backend-go/internal/app/router.go`
- 运行时装配：`backend-go/internal/app/runtime.go`
- 配置文件：`backend-go/configs/config.yaml`
- 数据库逻辑：`backend-go/pkg/database/db.go`

## 开发命令

```bash
go mod tidy
go run cmd/server/main.go
go test ./...
go run cmd/migrate-digest/main.go
go run cmd/test-digest/main.go
```

## 架构文档

- 后端架构：`docs/architecture/backend-go.md`
- 后端运行与接口：`docs/architecture/backend-runtime.md`
- 数据流：`docs/architecture/data-flow.md`
- 数据库说明：`docs/operations/database.md`
- 开发流程：`docs/operations/development.md`

## 说明

- `docs/` 里的文档现在是正式维护入口
- `backend-go/ARCHITECTURE.md` 和 `backend-go/DATABASE.md` 适合当历史参考，不再当作现状真相
- 后端目录会继续从横向分层，逐步迁向按职责和领域组织
- 当前方向见 `docs/architecture/backend-go.md`
