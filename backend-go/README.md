# RSS Reader Backend (Go)

> **注意**：Go 后端与 Python 后端使用相同的数据库，100% 兼容，可以直接切换使用！

## 📋 目录

- [快速开始](#快速开始)
- [数据库兼容性](#数据库兼容性)
- [常见问题](#常见问题)
- [API 文档](#api-文档)
- [开发指南](#开发指南)

## 🚀 快速开始

### 前提条件

- Go 1.21 或更高版本
- 现有的 RSS Reader 数据库（由 Python 后端创建）

### 安装和运行

```bash
# 1. 进入目录
cd backend-go

# 2. 下载依赖
go mod download

# 3. 运行服务器
go run cmd/server/main.go
```

服务器将在 `http://localhost:5000` 启动。

### 构建可执行文件

```bash
# 构建
go build -o bin/server.exe cmd/server/main.go

# 运行
./bin/server.exe
```

## 📦 数据库兼容性

### ⚠️ 重要：数据库迁移错误

如果您看到以下错误：
```
SQL logic error: table categories__temp has no column named UNIQUE (1)
```

**原因**：Go 后端默认不运行自动迁移，直接使用 Python 后端创建的数据库。

**解决方案**：默认配置已正确，无需修改。直接运行即可：
```bash
go run cmd/server/main.go
```

### 数据库管理工具

检查数据库状态：
```bash
go run cmd/migrate/main.go check
```

其他命令：
```bash
# 运行迁移（谨慎使用）
go run cmd/migrate/main.go migrate

# 全新开始（删除所有数据）
go run cmd/migrate/main.go fresh
```

详细说明：[DATABASE_COMPATIBILITY.md](DATABASE_COMPATIBILITY.md)

## 🔧 配置

编辑 `configs/config.yaml`：

```yaml
server:
  port: "5000"        # 服务端口
  mode: "debug"       # debug, release, test

database:
  driver: "sqlite"
  dsn: "rss_reader.db"  # 数据库文件路径

cors:
  origins:
    - "http://localhost:3001"
    - "http://localhost:3000"
```

## 📡 API 文档

### 基础端点
- `GET /` - API 信息
- `GET /health` - 健康检查
- `GET /api/tasks/status` - 任务状态

### Categories
- `GET /api/categories` - 获取所有分类
- `POST /api/categories` - 创建分类
- `PUT /api/categories/:id` - 更新分类
- `DELETE /api/categories/:id` - 删除分类

### Feeds
- `GET /api/feeds` - 获取订阅源
- `POST /api/feeds` - 创建订阅源
- `PUT /api/feeds/:id` - 更新订阅源
- `DELETE /api/feeds/:id` - 删除订阅源
- `POST /api/feeds/:id/refresh` - 刷新订阅源
- `POST /api/feeds/fetch` - 预览订阅源

### Articles
- `GET /api/articles/stats` - 获取统计
- `GET /api/articles` - 获取文章列表
- `GET /api/articles/:id` - 获取文章详情
- `PUT /api/articles/:id` - 更新文章
- `PUT /api/articles/bulk-update` - 批量更新

### AI
- `POST /api/ai/summarize` - AI 总结
- `POST /api/ai/test` - 测试连接
- `GET /api/ai/settings` - 获取配置
- `POST /api/ai/settings` - 保存配置

### OPML
- `POST /api/import-opml` - 导入 OPML
- `GET /api/export-opml` - 导出 OPML

### Schedulers
- `GET /api/schedulers/status` - 获取状态
- `POST /api/schedulers/:name/trigger` - 触发调度器
- `PUT /api/schedulers/:name/interval` - 更新间隔

## ❓ 常见问题

### 1. 端口被占用

**错误**：`bind: Only one usage of each socket address`

**解决**：
- 方案 A：修改 `configs/config.yaml` 中的端口
- 方案 B：停止占用 5000 端口的进程

### 2. CORS 错误

**解决**：确保 `configs/config.yaml` 中的 CORS 配置包含前端地址。

### 3. 找不到包

**解决**：
```bash
go mod tidy
go mod download
```

更多问题：[TROUBLESHOOTING.md](TROUBLESHOOTING.md)

## 📁 项目结构

```
backend-go/
├── cmd/
│   ├── server/main.go      # 主服务器
│   └── migrate/main.go     # 数据库工具
├── internal/
│   ├── config/             # 配置管理
│   ├── handlers/           # HTTP 处理器
│   ├── models/             # 数据模型
│   ├── services/           # 业务逻辑
│   ├── schedulers/         # 调度器
│   └── middleware/         # 中间件
├── pkg/database/           # 数据库连接
├── configs/config.yaml     # 配置文件
├── bin/server.exe          # 可执行文件
└── go.mod
```

## 🔨 开发命令

```bash
# 运行
go run cmd/server/main.go

# 构建
go build -o bin/server.exe cmd/server/main.go

# 依赖管理
go mod tidy
go mod download

# 检查数据库
go run cmd/migrate/main.go check

# 运行测试
go test ./...
```

## 🎯 完成度

| 功能 | 状态 |
|------|------|
| 核心 CRUD | ✅ 100% |
| RSS 解析 | ✅ 100% |
| AI 功能 | ✅ 100% |
| OPML | ✅ 100% |
| 调度器 | ✅ 90% |
| 测试 | 🚧 70% |

## 📚 更多文档

- [DATABASE_COMPATIBILITY.md](DATABASE_COMPATIBILITY.md) - 数据库兼容性详解
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - 故障排除
- [PROGRESS.md](PROGRESS.md) - 开发进度
- [COMPLETION.md](COMPLETION.md) - 完成总结

## 🙏 致谢

本项目基于 Python Flask 版本重写，感谢原始版本的设计和实现。

## 📄 许可证

MIT License
