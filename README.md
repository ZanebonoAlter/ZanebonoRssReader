# RSS Reader with AI Content Completion

基于 Go + Nuxt 4 的智能 RSS 阅读器，支持 AI 自动补全和总结文章内容。

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D?logo=vue.js)](https://vuejs.org/)
[![Nuxt](https://img.shields.io/badge/Nuxt-4.0-00DC82?logo=nuxt.js)](https://nuxt.com/)
[![Python](https://img.shields.io/badge/Python-3.10+-3776AB?logo=python)](https://www.python.org/)

## ✨ 特性

- 🚀 **高性能后端** - Go + Gin + SQLite，快速响应
- 🎨 **现代界面** - Vue 3 + Nuxt 4，三栏 FeedBro 风格布局
- 🤖 **AI 内容补全** - 自动抓取完整内容并生成总结
- 📰 **智能汇总** - 按分类自动生成新闻汇总
- 🔄 **自动刷新** - 定时自动更新订阅源
- 📊 **阅读偏好** - 追踪阅读习惯，智能排序

## 🏗️ 架构

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Frontend  │      │   Backend   │      │Crawl Service│
│  (Nuxt 4)   │◄────►│  (Go + Gin) │◄────►│  (Python)   │
│   :3001     │      │   :5000     │      │   :11235     │
└─────────────┘      └─────────────┘      └─────────────┘
                            │
                            ▼
                     ┌─────────────┐
                     │  SQLite DB  │
                     └─────────────┘
```

## 📋 前置要求

- **Node.js** 18+ - 前端开发
- **Go** 1.21+ - 后端开发
- **Python** 3.10+ - 爬虫服务
- **pnpm** - 前端包管理器
- **uv** - Python 包管理器

## 🚀 快速开始

### 方式 1：一键启动（推荐）

**Windows:**
```bash
scripts/start.bat
```

**Linux/Mac:**
```bash
chmod +x scripts/start.sh
./scripts/start.sh
```

这会自动：
1. 安装 Playwright 浏览器（首次运行）
2. 启动爬虫服务
3. 启动后端服务
4. 启动前端服务

### 方式 2：手动启动

#### 1. 安装 Playwright 浏览器（仅首次）

```bash
scripts/install-playwright.bat
```

#### 2. 启动服务

```bash
# Terminal 1: 爬虫服务
cd crawl-service
uv run python main.py

# Terminal 2: 后端
cd backend-go
go run cmd/server/main.go

# Terminal 3: 前端
cd front
pnpm dev
```

#### 3. 访问应用

- 前端: http://localhost:3001
- 后端 API: http://localhost:5000
- 爬虫服务: http://localhost:11235/health

## 📁 项目结构

```
my-robot/
├── backend-go/              # Go 后端
│   ├── cmd/                # 命令行工具
│   ├── internal/           # 私有代码
│   │   ├── handlers/       # HTTP 处理器
│   │   ├── services/       # 业务逻辑
│   │   ├── models/         # 数据模型
│   │   └── schedulers/     # 后台任务
│   └── pkg/                # 公共代码
│
├── front/                  # Nuxt 4 前端
│   ├── app/
│   │   ├── components/     # Vue 组件
│   │   ├── composables/    # 组合式函数
│   │   ├── stores/         # Pinia 状态管理
│   │   └── types/          # TypeScript 类型
│   └── nuxt.config.ts
│
├── crawl-service/          # Python 爬虫服务
│   ├── main.py             # FastAPI 应用
│   └── pyproject.toml      # 依赖配置
│
├── scripts/                # 工具脚本
│   ├── start.bat           # Windows 启动脚本
│   ├── start.sh            # Linux/Mac 启动脚本
│   └── install-playwright.bat
│
├── docs/                   # 文档
│   ├── CONTENT_COMPLETION.md
│   ├── QUICKSTART.md
│   └── READING_PREFERENCES.md
│
├── AGENTS.md               # AI Agent 指南
├── CLAUDE.md               # Claude 使用指南
└── README.md               # 本文件
```

## 🎯 核心功能

### AI 内容补全

自动补全 RSS 中不完整的文章内容：

1. 编辑 Feed，启用"内容补全"
2. 刷新时自动检测不完整内容
3. 使用 Crawl4AI 抓取完整内容
4. AI 生成结构化总结

详细文档: [Content Completion](docs/CONTENT_COMPLETION.md)

### AI 智能总结

- **单文章总结**: 手动触发 AI 总结
- **批量汇总**: 按分类自动生成新闻汇总
- **调度任务**: 每小时自动生成汇总

### 阅读偏好追踪

- 自动记录阅读行为
- 分析阅读习惯
- 智能推荐排序

## 🛠️ 开发

### 后端开发

```bash
cd backend-go

# 运行服务器
go run cmd/server/main.go

# 运行测试
go test ./...

# 数据库迁移
go run cmd/migrate/main.go
go run cmd/migrate-content-completion/main.go
```

### 前端开发

```bash
cd front

# 开发模式
pnpm dev

# 类型检查
npx nuxi typecheck

# 构建生产版本
pnpm build
```

### 爬虫服务开发

```bash
cd crawl-service

# 开发模式
uv run python main.py

# 安装依赖
uv sync

# 安装浏览器
uv run playwright install chromium
```

## 📚 文档

- [快速启动](docs/QUICKSTART.md) - 5 分钟上手
- [Content Completion](docs/CONTENT_COMPLETION.md) - 内容补全功能
- [Reading Preferences](docs/READING_PREFERENCES.md) - 阅读偏好追踪
- [Backend Development](backend-go/README.md) - 后端开发指南
- [Frontend Development](front/README.md) - 前端开发指南

## 🔧 配置

### 后端配置

编辑 `backend-go/configs/config.yaml`:

```yaml
server:
  port: 5000
  mode: debug  # debug | release

database:
  dsn: rss_reader.db

crawl_service:
  url: http://localhost:11235
```

### 爬虫服务配置

编辑 `crawl-service/.env`:

```bash
PORT=11235
CRAWL_TIMEOUT=30
CRAWL_ONLY_MAIN_CONTENT=true
LOG_LEVEL=info
```

## 🐛 故障排查

### 爬虫服务启动失败

```bash
# Windows
scripts/install-playwright.bat

# Linux/Mac
cd crawl-service && uv run playwright install chromium
```

### 数据库错误

```bash
cd backend-go
go run cmd/migrate/main.go
go run cmd/migrate-content-completion/main.go
```

### 前端无法连接后端

检查 `front/app/utils/constants.ts` 中的 `API_BASE_URL`

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

**需要帮助？** 查看 [文档](docs/) 或提交 Issue
