# 技术栈

**分析日期:** 2026-04-10

## 编程语言

**主要:**
- TypeScript 5.x - 前端全栈使用，Vue 3 Composition API + `<script setup lang="ts">`
- Go 1.25.0 - 后端核心语言，domain-driven design 架构

**辅助:**
- Python 3.x - 集成测试 (`tests/workflow/`, `tests/firecrawl/`)

## 运行环境

**前端:**
- Node.js (pnpm 10.15.0 作为包管理器)
- Nuxt 4.2.2 作为 SSR/SSG 框架
- Vite 作为开发构建工具

**后端:**
- Go runtime，默认端口 5000
- Gin Web Framework 模式
- 环境变量通过 Viper 配置

## 框架

**前端核心:**
- Nuxt 4.2.2 - SSR/SSG 框架
- Vue 3.5.26 - UI 框架，Composition API
- Pinia 3.0.4 - 状态管理
- Vue Router 4.6.4 - 路由
- Tailwind CSS 4.1.18 - CSS 框架
- @nuxt/ui 4.3.0 - UI 组件库

**前端辅助:**
- motion-v 1.7.6 - 动画库
- @vueuse/core 14.1.0 - Vue 工具库
- dayjs 1.11.19 - 日期处理
- marked 17.0.1 - Markdown 解析
- @iconify/vue 5.0.0 - 图标库
- 3d-force-graph 1.79.1 + three 0.183.2 - 3D 图可视化

**后端核心:**
- Gin 1.12.0 - HTTP Web 框架
- GORM 1.25.12 - ORM
- Gorilla WebSocket 1.5.3 - WebSocket 支持
- robfig/cron v3.0.1 - 任务调度

**后端辅助:**
- mmcdole/gofeed 1.3.0 - RSS 解析
- spf13/viper 1.19.0 - 配置管理
- go.uber.org/zap 1.27.1 - 高性能日志
- stretchr/testify 1.11.1 - 测试断言

## 数据库

**主数据库:**
- PostgreSQL (近期从 SQLite 迁移)
  - 驱动: `gorm.io/driver/postgres`
  - 连接池配置: max_open_conns=25, max_idle_conns=5
  - 迁移脚本: `backend-go/internal/platform/database/postgres_migrations.go`
  - 数据迁移工具: `backend-go/cmd/migrate-db/main.go`

**SQLite (遗留支持):**
- 仍保留 SQLite 支持 (`glebarez/sqlite v1.11.0`)
- 可通过配置切换 `database.driver`

**缓存/队列:**
- Redis (可选，用于主题分析队列)
  - 驱动: `github.com/redis/go-redis/v9 v9.18.0`
  - 通过 `REDIS_URL` 环境变量启用
  - 回退策略: Redis 不可用时使用内存队列

## 测试框架

**前端:**
- Vitest 3.2.4 - 单元测试
- @vue/test-utils 2.4.6 - Vue 组件测试
- happy-dom 20.8.4 - DOM 环境
- Playwright 1.53.2 - E2E 测试

**后端:**
- go test - Go 标准测试框架
- stretchr/testify - 断言和 mock

**集成测试:**
- pytest 8.0.0 - Python 测试框架
- pytest-cov 4.1.0 - 覆盖率报告
- requests 2.31.0 - HTTP 客户端

## 构建工具

**前端:**
- pnpm 10.15.0 - 包管理 (lockfile: `pnpm-lock.yaml`)
- Nuxt CLI (`nuxi`) - 构建命令
- Vite - 开发服务器和打包

**后端:**
- Go modules (`go.mod`, `go.sum`)
- gofmt - 代码格式化

## 监控与追踪

**OpenTelemetry:**
- go.opentelemetry.io/otel v1.42.0
- 自定义 SQLite Span Exporter (`backend-go/internal/platform/tracing/`)
- Gin 中间件集成 (`otelgin`)
- Trace context propagation (W3C traceparent)

**日志:**
- go.uber.org/zap - 结构化日志

## 配置管理

**前端:**
- `front/nuxt.config.ts` - Nuxt 配置
- `front/vitest.config.ts` - Vitest 配置
- `front/playwright.config.ts` - Playwright 配置
- 环境变量: `NUXT_PUBLIC_API_ORIGIN`, `NUXT_PUBLIC_API_BASE`, `API_INTERNAL_BASE`

**后端:**
- YAML 配置文件 (`config.yaml`)
- Viper 支持环境变量覆盖
- 主要环境变量: `SERVER_PORT`, `SERVER_MODE`, `DATABASE_DRIVER`, `DATABASE_DSN`, `CORS_ORIGINS`, `REDIS_URL`

## 平台要求

**开发环境:**
- Node.js 18+
- Go 1.25+
- PostgreSQL 12+ (推荐)
- Redis (可选)
- pnpm 10+

**生产部署:**
- 单用户部署，无需认证系统
- 默认前端: `localhost:3000`
- 默认后端 API: `localhost:5000/api`
- WebSocket: `ws://localhost:5000/ws`

---

*技术栈分析: 2026-04-10*