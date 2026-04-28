# 代码库关注点

**分析日期:** 2026-04-10

## PostgreSQL 迁移遗留问题

### SQLite 代码残留 (HIGH)

**问题描述:**
项目已从 SQLite 迁移到 PostgreSQL（`backend-go/configs/config.yaml` driver 设置为 "postgres"），但大量 SQLite 相关代码仍然存在，可能造成混淆和维护负担。

**残留文件:**
- `backend-go/internal/platform/database/sqlite_legacy_migrations.go` (581行) - SQLite 专用迁移逻辑，包含 `sqliteColumnExists`, `sqliteTableExists`, `sqliteIndexExists` 等函数
- `backend-go/internal/platform/database/connect_sqlite.go` - SQLite 连接器仍保留
- `backend-go/internal/platform/database/datamigrate/reader_sqlite.go` - SQLite 数据读取器（用于迁移工具）
- `backend-go/go.mod` 第7行: `github.com/glebarez/sqlite v1.11.0` - SQLite 驱动依赖仍声明
- `backend-go/go.mod` 第89行: `modernc.org/sqlite v1.23.1` - SQLite 纯 Go 实现

**默认配置仍指向 SQLite:**
- `backend-go/internal/platform/config/config.go` 第60-61行: 默认 driver 为 "sqlite"，默认 DSN 为 "rss_reader.db"
- `backend-go/internal/platform/database/db.go` 第23行: `const defaultDatabaseDriver = "sqlite"`
- `.env.example` 第3行: `SQLITE_DB_FILE=rss_reader.db` - 无 PostgreSQL 相关示例

**影响范围:**
- 新开发者可能误认为 SQLite 是默认数据库
- 配置加载失败时会回退到 SQLite 默认值
- 依赖管理存在无用包

**修复建议:**
1. 评估是否需要保留 SQLite 支持用于本地开发/测试
2. 若决定移除 SQLite 支持，删除上述文件并更新默认配置
3. 更新 `.env.example` 为 PostgreSQL 示例
4. 若保留 SQLite 支持，明确文档说明用途（仅用于测试/归档分支）

### 数据文件残留 (MEDIUM)

**问题描述:**
SQLite 数据文件仍存在于 data 目录，与 PostgreSQL 生产配置并存。

**文件位置:**
- `data/rss_reader.db` (4096 bytes)
- `data/rss_reader.db-shm` (32768 bytes) - WAL 共享内存文件
- `data/rss_reader.db-wal` (3736872 bytes) - WAL 日志文件（约3.7MB）

**影响:**
- 可能造成数据混淆
- 占用磁盘空间
- 暗示 SQLite 仍在使用

**修复建议:**
1. 确认 PostgreSQL 迁移完成后，可删除 SQLite 数据文件
2. 或将其移动到 `data/archive/` 目录作为备份保留
3. 更新 `.gitignore` 明确排除 SQLite 文件（已部分实现）

### Docker 配置并存 (MEDIUM)

**问题描述:**
存在两个 Docker Compose 配置文件，分别对应 SQLite 和 PostgreSQL。

**文件位置:**
- `docker-compose.sqlite.yml` (42行) - SQLite 版本容器配置
- `docker-compose.pgvector.yml` (24行) - PostgreSQL + pgvector 配置
- 无默认 `docker-compose.yml` 文件

**影响:**
- 部署时需明确指定配置文件
- 可能误用 SQLite 配置启动服务

**修复建议:**
1. 明确文档说明两种配置用途
2. 或将 `docker-compose.pgvector.yml` 重命名为 `docker-compose.yml` 作为默认
3. SQLite 配置可移至 `docker/archive/` 或明确标记为归档版本

### 测试代码仍使用 SQLite (MEDIUM)

**问题描述:**
后端测试代码普遍使用 SQLite 内存数据库，与生产 PostgreSQL 不一致。

**文件位置:**
- `backend-go/internal/platform/database/db_test.go` 全部测试使用 `sqlite.Open(...)` 内存数据库
- 多处测试代码包含 `gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())))`

**影响:**
- 测试不能验证 PostgreSQL 特定功能（如 pgvector）
- 某些 SQL 语法差异可能被测试遗漏

**修复建议:**
1. 建立测试数据库工厂，支持通过环境变量切换 SQLite/PostgreSQL（已在迁移计划中提及）
2. 关键功能测试应覆盖 PostgreSQL 路径
3. 保持 SQLite 内存测试用于快速迭代，但关键集成测试用 PostgreSQL

---

## 文档状态问题

### 文档与实际不符 (HIGH)

**问题描述:**
大量文档仍描述 SQLite 为唯一数据库，与当前 PostgreSQL 生产配置不符。

**需更新文档:**
- `README.md` 第76行: "后端 | Go + Gin + GORM + SQLite"
- `README.md` 第90行: "SQLite 文件默认落在仓库根目录 `data/rss_reader.db`"
- `docs/architecture/backend-go.md` 第21、99行: SQLite 描述
- `docs/operations/database.md` 第5-6行: "数据库类型：SQLite"
- `docs/architecture/backend-runtime.md` 第8行: SQLite 初始化描述
- `docs/architecture/tracing.md` 第7、21、165、222、291行: SQLite exporter 描述
- `docs/operations/development.md` 第87行: SQLite 配置说明
- `AGENTS.md` 第17、21行: SQLite 持久化描述

**影响:**
- 新开发者获取错误架构信息
- 可能做出与实际不符的决策

**修复建议:**
1. 系统性更新所有架构文档，反映 PostgreSQL + pgvector 当前状态
2. 明确说明 SQLite 为历史版本或归档用途
3. 更新 AGENTS.md 的项目快照描述

### 迁移计划文档残留 (LOW)

**问题描述:**
`docs/superpowers/plans/2026-04-03-postgres-pgvector-single-database-architecture-migration.md` 是详细的迁移计划文档，但迁移已完成，该文档可能造成混淆。

**文件位置:**
- 693行详细迁移计划文档

**影响:**
- 可能误认为迁移仍在进行中

**修复建议:**
1. 将该文档移动到 `docs/history/` 作为历史记录
2. 或添加标记说明迁移已完成
3. 保留为参考但明确状态

---

## 安全考量

### 无认证系统 (LOW - 已明确)

**问题描述:**
项目设计为单用户部署，无认证系统。所有 API 端点无需认证即可访问。

**确认位置:**
- CORS 中间件允许 Authorization 头，但不验证（`backend-go/internal/platform/middleware/cors.go`）
- 所有 Authorization 头使用仅用于外部 API（OpenAI、Firecrawl 等）
- 无 JWT、session 或用户表

**影响:**
- 单机部署场景下可接受
- 多用户/云端部署时存在安全风险

**当前缓解措施:**
- 项目明确为个人/单用户部署
- AGENTS.md 已说明 "no auth system"

**建议:**
- 若未来需要多用户，需设计认证系统
- 当前保持现状，但文档明确安全边界

### API Key 管理 (MEDIUM)

**问题描述:**
API Key 通过配置和环境变量管理，但部分硬编码在测试代码中。

**位置:**
- 多处测试使用 `"token"` 作为硬编码 API Key
- 无 API Key 加密存储机制
- 依赖环境变量传递敏感信息

**影响:**
- 测试代码不影响生产安全
- 生产环境依赖正确的环境变量配置

**建议:**
- 生产部署文档明确 API Key 配置方式
- 考虑添加配置验证，确保必要 API Key 存在

---

## 测试覆盖差距

### 前端 E2E 测试依赖后端 (LOW)

**问题描述:**
前端 E2E 测试 (`front/tests/e2e/*.spec.ts`) 需要后端服务运行，测试隔离性有限。

**文件位置:**
- `front/playwright.config.ts`
- `front/tests/e2e/baseline.spec.ts`
- `front/tests/e2e/topic-graph.spec.ts`

**影响:**
- E2E 测试需要完整环境启动
- CI/CD 流程复杂化

**建议:**
- 保持现有集成测试模式（已有效）
- 考虑添加更多纯前端单元测试

### 部分模块缺少测试 (MEDIUM)

**问题描述:**
大型业务模块缺少对应测试文件。

**无测试的大型文件:**
- `backend-go/internal/domain/topicgraph/service.go` (839行) - 无对应 `*_test.go`
- `backend-go/internal/jobs/auto_summary.go` (775行) - 有测试但覆盖率需验证
- `front/app/features/topic-graph/components/AIAnalysisPanel.vue` (808行) - 无测试
- `front/app/features/summaries/components/AISummaryDetailView.vue` (875行) - 无测试

**影响:**
- 核心业务逻辑变更可能引入未检测问题

**建议:**
- 为 `topicgraph/service.go` 添加关键路径测试
- 前端大型组件考虑添加交互测试

---

## 代码复杂度问题

### 大型文件需关注 (MEDIUM)

**问题描述:**
部分文件行数较多，可能需要拆分重构。

**后端大型文件 (>400行):**
- `backend-go/internal/domain/topicgraph/service.go` (839行) - Topic Graph 核心服务
- `backend-go/internal/jobs/auto_summary.go` (775行) - 自动摘要调度器
- `backend-go/internal/platform/database/sqlite_legacy_migrations.go` (581行) - SQLite 迁移（可能可清理）
- `backend-go/internal/jobs/content_completion.go` (492行) - 内容补全调度器
- `backend-go/internal/domain/topicextraction/extractor_enhanced.go` (522行) - 增强提取器
- `backend-go/internal/jobs/auto_refresh.go` (444行) - 自动刷新调度器
- `backend-go/internal/platform/database/datamigrate/writer_postgres.go` (461行) - PostgreSQL 写入器

**前端大型文件 (>400行):**
- `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue` (1508行)
- `front/app/features/topic-graph/components/TopicTimeline.vue` (1366行)
- `front/app/features/topic-graph/components/TopicGraphSidebar.vue` (1328行)
- `front/app/features/summaries/components/AISummaryDetailView.vue` (875行)
- `front/app/features/topic-graph/components/AIAnalysisPanel.vue` (808行)
- `front/app/features/topic-graph/components/TopicAnalysisPanel.vue` (499行)
- `front/app/features/topic-graph/utils/buildDisplayedTopicGraph.ts` (476行)

**建议:**
- 评估是否需要拆分（当前快速迭代阶段可接受）
- 关注测试覆盖，确保变更可控
- 高优先级：SQLite legacy 文件可考虑清理

---

## 依赖管理

### SQLite 依赖残留 (MEDIUM)

**问题描述:**
`go.mod` 中 SQLite 相关依赖仍声明，与 PostgreSQL 生产配置并存。

**依赖列表:**
- `github.com/glebarez/sqlite v1.11.0` - CGO-free SQLite 驱动
- `modernc.org/sqlite v1.23.1` - 纯 Go SQLite 实现
- 多个 SQLite 相关间接依赖

**影响:**
- 增加构建依赖图复杂度
- 若不使用，属于无用依赖

**建议:**
- 若决定移除 SQLite 支持，清理相关依赖
- 若保留用于测试，明确注释用途

### PostgreSQL 驱动缺失声明 (LOW)

**问题描述:**
`go.mod` 未直接声明 `gorm.io/driver/postgres`，但代码中使用。

**位置:**
- `backend-go/cmd/migrate-db/main.go` 第13行: `gorm.io/driver/postgres`
- `backend-go/internal/platform/database/connect_postgres.go` 第7行: `gorm.io/driver/postgres`

**状态:**
- 可能通过间接依赖引入
- 应显式声明以确保版本控制

**建议:**
- 在 `go.mod` require 块显式添加 `gorm.io/driver/postgres`

---

## 运行时问题

### Tracing 模块 SQLite 描述残留 (MEDIUM)

**问题描述:**
Tracing 文档和代码注释仍暗示只支持 SQLite。

**位置:**
- `docs/architecture/tracing.md` 第21行: "SQLiteSpanExporter"
- `docs/architecture/tracing.md` 第165行: "所有 span 目前写入 SQLite 表"

**影响:**
- Tracing 实际写入 PostgreSQL 的 `otel_spans` 表，文档错误

**建议:**
- 更新 tracing 文档反映 PostgreSQL 表名
- 代码注释应改为通用数据库描述

---

## 快速迭代状态

### 代码可能处于活跃开发 (INFO)

**观察:**
- 最近提交涉及 PostgreSQL 迁移
- 存在 `.sisyphus/plans/` 目录表明有计划管理
- `.worktrees/` 目录表明使用 git worktrees 进行并行开发
- 大型功能文件表明功能快速扩展

**影响:**
- 部分代码可能仍在调整中
- 测试覆盖可能滞后于新功能

**建议:**
- 优先关注核心路径稳定性
- 添加测试后再重构大型文件

---

## 优先级总结

| 优先级 | 关注点 | 数量 |
|--------|--------|------|
| HIGH | PostgreSQL 迁移遗留问题、文档与实际不符 | 2 |
| MEDIUM | SQLite 代码残留、数据文件残留、Docker 配置并存、测试使用 SQLite、API Key 管理、大型文件复杂度、SQLite 依赖残留、Tracing 文档错误 | 8 |
| LOW | 迁移计划文档残留、无认证系统、E2E 测试依赖、PostgreSQL 驱动声明 | 4 |
| INFO | 快速迭代状态 | 1 |

---

**审计日期:** 2026-04-10