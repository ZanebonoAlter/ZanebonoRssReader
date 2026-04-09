# 开发指南

## 环境要求

- Node.js 18+
- pnpm 10.15.0+
- Go 1.21+

## 启动顺序

1. 启动 `backend-go`
2. 启动 `front`
3. 确认前端能连到 `http://localhost:5000/api`

## 前端开发

```bash
cd front
pnpm install
pnpm dev
pnpm build
pnpm exec nuxi typecheck
pnpm test:unit
```

默认地址：`http://localhost:3001`

## 后端开发

```bash
cd backend-go
go mod tidy
go run cmd/server/main.go
go build ./...
go test ./...
```

默认地址：`http://localhost:5000`

## 常用后端命令

```bash
cd backend-go

# 启动服务
go run cmd/server/main.go

# 运行全部 Go 测试
go test ./...

# digest 迁移
go run cmd/migrate-digest/main.go

# digest 测试入口
go run cmd/test-digest/main.go

# scheduler 相关验证
go test ./internal/schedulers ./internal/handlers
```

## 后端阅读入口

- 入口：`backend-go/cmd/server/main.go`
- 路由：`backend-go/internal/app/router.go`
- 运行时：`backend-go/internal/app/runtime.go`
- 数据库：`backend-go/pkg/database/db.go`

如果你要整理后端结构，先从 `internal/app/` 开始，不要直接把全部注意力丢进 `handlers/`。

## 一键启动

Windows 下可以直接运行：

```bash
start-all.bat
```

也可以直接使用 Docker Compose：

```bash
cp .env.example .env
docker compose up --build
```

- 前端端口通过 `FRONT_PORT` 配置
- 后端端口通过 `BACKEND_PORT` 配置
- SQLite 文件名通过 `SQLITE_DB_FILE` 配置
- 数据文件会写到仓库根目录 `data/`
- Docker 默认前端端口是 `3000`
- 构建代理可通过 `.env` 里的 `GOPROXY`、`NPM_CONFIG_REGISTRY`、`HTTP_PROXY`、`HTTPS_PROXY` 配置

## 前后端联调约定

- 前端 API 基础地址：`http://localhost:5000/api`
- AI 总结 WebSocket：默认 `ws://localhost:5000/ws`
- 后端 ID 是数字
- 前端 store 内统一转成字符串
- feed 文章总结开关统一使用 `article_summary_enabled` / `articleSummaryEnabled`
- 文章总结状态统一使用 `summary_status` / `summaryStatus`

## 前端开发约束

### 目录约束

- HTTP 逻辑只放 `front/app/api`
- 业务实现优先放 `front/app/features`
- 通用组件放 `front/app/components`
- 路由页只做挂载，不写大段业务

### 状态约束

- `useApiStore` 是主数据源
- `useFeedsStore` 和 `useArticlesStore` 只做派生视图
- 不再新增 `syncToLocalStores()` 一类的副本同步逻辑

### 代码约束

- 新增前端代码默认使用 `<script setup lang="ts">`
- 类型定义集中在 `front/app/types`
- API 返回值统一通过 `ApiResponse<T>` 包装
- `snake_case -> camelCase` 的映射集中在 API 或 store 层
- 字段重命名不在组件里做兼容，直接在类型和 store 映射层切换

### 样式约束

- 保持 editorial / magazine 主题
- 不回退到蓝紫 SaaS 视觉
- 尽量复用 `main.css` 里的主题变量
- 对话框、卡片、状态标签优先沿用现有语义类

## 编码与文件写入

- 前端源码必须使用 UTF-8
- PowerShell 改写文件时要显式保持 UTF-8
- 如果构建报 Vue / Vite 编码错误，先查文件编码，不要先怀疑业务逻辑

## 提交前最低检查

前端改动至少执行其中一项：

- `pnpm build`
- `pnpm exec nuxi typecheck`
- `pnpm test:unit`

后端改动至少执行其中一项：

- `go build ./...`
- 对应范围的 Go 测试

文档改动如果涉及功能、接口、结构变化，也要同步更新：

- `docs/architecture/frontend.md`
- `docs/architecture/frontend-components.md`
- `docs/architecture/backend-go.md`
- `docs/architecture/backend-runtime.md`
- `docs/architecture/data-flow.md`
- `docs/guides/frontend-features.md`
- `docs/operations/database.md`
