# 项目结构

**分析日期:** 2026-04-10

## 目录布局

```
my-robot/
├── front/                    # Nuxt 4 前端应用
│   ├── app/                  # 应用核心代码
│   │   ├── api/              # HTTP API 客户端层
│   │   ├── components/       # 共享UI组件
│   │   ├── composables/      # Vue composables
│   │   ├── features/         # 领域特性模块
│   │   ├── pages/            # Nuxt 路由页面
│   │   ├── stores/           # Pinia 状态存储
│   │   ├── types/            # TypeScript 类型定义
│   │   ├── utils/            # 工具函数
│   │   └── app.vue           # 应用入口组件
│   ├── public/               # 静态资源
│   ├── nuxt.config.ts        # Nuxt 配置
│   ├── vitest.config.ts      # Vitest 测试配置
│   ├── tsconfig.json         # TypeScript 配置
│   └── package.json          # 前端依赖
│
├── backend-go/               # Go 后端应用
│   ├── cmd/                  # 命令入口点
│   │   ├── server/           # HTTP服务主入口
│   │   ├── migrate-digest/   # Digest迁移工具
│   │   ├── test-digest/      # Digest测试工具
│   │   ├── migrate-db/       # 数据库迁移工具
│   │   └── migrate-tags/     # 标签迁移工具
│   ├── internal/             # 内部代码
│   │   ├── app/              # 应用层
│   │   │   ├── router.go     # HTTP路由定义
│   │   │   ├── runtime.go    # 调度器启动/关闭
│   │   │   └── runtimeinfo/  # 运行时信息
│   │   ├── domain/           # 领域层
│   │   │   ├── models/       # GORM数据模型
│   │   │   ├── feeds/        # RSS订阅领域
│   │   │   ├── articles/     # 文章领域
│   │   │   ├── summaries/    # AI摘要领域
│   │   │   ├── digest/       # 摘要导出领域
│   │   │   ├── contentprocessing/ # 内容处理领域
│   │   │   ├── topicgraph/   # 主题图谱领域
│   │   │   ├── topicextraction/ # 标签提取领域
│   │   │   ├── topicanalysis/ # 主题分析领域
│   │   │   ├── topictypes/   # 主题类型定义
│   │   │   ├── preferences/  # 用户偏好领域
│   │   │   ├── categories/   # 分类领域
│   │   │   └── aiadmin/      # AI管理领域
│   │   ├── jobs/             # 调度器任务
│   │   └── platform/         # 平台基础设施
│   │       ├── database/     # 数据库连接/迁移
│   │       ├── ws/           # WebSocket Hub
│   │       ├── tracing/      # OpenTelemetry追踪
│   │       ├── airouter/     # AI提供商路由
│   │       ├── config/       # 配置管理
│   │       ├── middleware/   # HTTP中间件
│   │       ├── aisettings/   # AI设置存储
│   │       ├── opennotebook/ # OpenNotebook客户端
│   │       └── ai/           # AI服务抽象
│   ├── configs/              # 配置文件目录
│   ├── go.mod                # Go模块定义
│   └── README.md             # 后端文档
│
├── tests/                    # 测试目录
│   ├── workflow/             # 工作流集成测试 (Python)
│   │   ├── utils/            # 测试工具
│   │   ├── test_*.py         # pytest测试文件
│   │   ├── requirements.txt  # Python依赖
│   │   └── README.md         # 测试文档
│   ├── firecrawl/            # Firecrawl集成测试
│   │   ├── test_firecrawl_integration.py
│   │   └── config.py
│
├── docker/                   # Docker配置
│   └ postgres/
│   │   └ init/               # PostgreSQL初始化脚本
│   │       └ 01-enable-pgvector.sql
│
├── docs/                     # 文档目录 (注意: 混合旧/新文档)
│   ├── architecture/         # 架构文档
│   ├── operations/           # 运维文档
│   └── *.md                  # 其他文档
│
├── data/                     # 数据存储目录
│   └ rss_reader.db           # SQLite遗留文件
│
├── docker-compose.pgvector.yml    # PostgreSQL容器配置
├── docker-compose.sqlite.yml      # SQLite遗留配置
├── .env                       # 环境变量 (不读取内容)
├── .env.example               # 环境变量示例
├── AGENTS.md                  # Agent开发指南
├── README.md                  # 项目主文档
└── CLAUDE.md                  # Claude AI指南
```

## 目录用途详解

### `front/app/`

**api/:**
- 用途: HTTP请求层，所有后端API调用入口
- 核心文件: `client.ts` (ApiClient类)
- 包含: `categories.ts`, `feeds.ts`, `articles.ts`, `opml.ts`, `summaries.ts` 等

**components/:**
- 用途: 共享UI组件，跨页面复用
- 结构:
  - `dialog/`: 对话框组件 (AddFeedDialog, EditFeedDialog等)
  - `feed/`: Feed相关组件 (FeedIcon, FeedActionMenu等)
  - `category/`: 分类组件 (CategoryCard)
  - `ai/`: AI相关组件 (AISummary)
  - `common/`: 通用组件 (AppTooltip)

**composables/:**
- 用途: Vue composables，可复用的响应式逻辑
- 特点: 不包含领域特定逻辑，通用工具

**features/:**
- 用途: 领域特性模块，按业务领域组织
- 结构:
  - `articles/`: 文章展示、内容补全
  - `digest/`: 摘要列表、详情、设置
  - `summaries/`: AI摘要WebSocket、列表、详情
  - `topic-graph/`: 主题图谱可视化、分析面板
  - `feeds/`: Feed刷新轮询
  - `ai/`: AI路由设置面板
  - `shell/`: 应用骨架 (侧边栏、头部、列表面板)
  - `preferences/`: 阅读追踪

**pages/:**
- 用途: Nuxt文件路由页面
- 关键文件:
  - `index.vue`: 主页面
  - `topics.vue`: 主题图谱页
  - `digest/index.vue`: 摘要列表页
  - `digest/[id].vue`: 摘要详情页

**stores/:**
- 用途: Pinia状态存储
- 关键文件:
  - `api.ts`: 主数据存储 (`useApiStore`)
  - `articles.ts`: 文章派生状态
  - `feeds.ts`: Feed派生状态
  - `preferences.ts`: 用户偏好
  - `aiAnalysis.ts`: AI分析状态

**types/:**
- 用途: TypeScript类型定义
- 包含: Article, RssFeed, Category, ApiResponse等类型

**utils/:**
- 用途: 工具函数
- 关键文件: `api.ts` (API基础URL获取)

### `backend-go/cmd/`

**server/:**
- 用途: HTTP服务主入口
- 关键文件: `main.go`
- 启动流程: 配置→数据库→路由→调度器→服务

**migrate-*/:**
- 用途: 数据迁移/测试工具
- 包含: digest迁移、数据库迁移、标签迁移、digest测试

### `backend-go/internal/`

**app/:**
- 用途: 应用层，路由和运行时
- 关键文件:
  - `router.go`: 所有HTTP路由定义
  - `runtime.go`: 调度器启动、优雅关闭

**domain/:**
- 用途: 业务逻辑层，按领域划分
- 模型层 `models/`:
  - `feed.go`: Feed GORM模型
  - `article.go`: Article GORM模型
  - `category.go`: Category模型
  - `ai_models.go`: AI配置模型
  - `topic_graph.go`: 主题图谱模型
  - `job_queue.go`: 任务队列模型
  - `user_preference.go`: 用户偏好
  - `reading_behavior.go`: 阅读行为

**jobs/:**
- 用途: 调度器定义
- 关键文件:
  - `auto_refresh.go`: RSS刷新调度器
  - `auto_summary.go`: AI摘要调度器
  - `preference_update.go`: 偏好更新调度器
  - `content_completion.go`: 内容补全调度器
  - `firecrawl.go`: Firecrawl调度器
  - `handler.go`: 调度器API handlers

**platform/:**
- 用途: 基础设施层，共享服务
- `database/`:
  - `db.go`: 数据库初始化入口
  - `connect_postgres.go`: PostgreSQL连接
  - `connect_sqlite.go`: SQLite连接 (遗留)
  - `postgres_migrations.go`: PostgreSQL迁移
  - `datamigrate/`: SQLite→PostgreSQL数据迁移工具
- `ws/`: WebSocket Hub实现
- `tracing/`: OpenTelemetry追踪
- `airouter/`: AI提供商路由 (多AI切换)
- `config/`: Viper配置管理
- `middleware/`: CORS等中间件

### `tests/`

**workflow/:**
- 用途: Python集成测试，验证调度器和API
- 特点: 需要 Go 后端运行在 localhost:5000
- 结构:
  - `utils/`: mock服务、API客户端、数据库工具
  - `test_schedulers.py`: 调度器测试
  - `test_workflow_integration.py`: 工作流集成测试
  - `test_error_handling.py`: 错误处理测试

**firecrawl/:**
- 用途: Firecrawl集成验证
- 特点: 需要 Go 后端运行

## 关键文件位置

### 入口点

| 文件 | 用途 |
|------|------|
| `backend-go/cmd/server/main.go` | Go后端服务入口 |
| `front/app/app.vue` | Vue前端入口 |
| `front/nuxt.config.ts` | Nuxt配置 |
| `backend-go/internal/app/router.go` | HTTP路由定义 |
| `backend-go/internal/app/runtime.go` | 调度器运行时 |

### 配置

| 文件 | 用途 |
|------|------|
| `backend-go/internal/platform/config/config.go` | Go配置管理 |
| `front/nuxt.config.ts` | Nuxt运行时配置 |
| `.env` | 环境变量 (不读取内容) |
| `docker-compose.pgvector.yml` | PostgreSQL容器配置 |

### 核心逻辑

| 文件 | 用途 |
|------|------|
| `front/app/stores/api.ts` | 前端主数据存储 |
| `front/app/api/client.ts` | HTTP客户端封装 |
| `backend-go/internal/domain/feeds/service.go` | Feed业务逻辑 |
| `backend-go/internal/domain/digest/generator.go` | Digest生成逻辑 |
| `backend-go/internal/platform/ws/hub.go` | WebSocket Hub |

### 测试

| 文件 | 用途 |
|------|------|
| `front/vitest.config.ts` | Vitest配置 |
| `tests/workflow/pytest` | Python集成测试 |
| `backend-go/**/*_test.go` | Go单元测试 |

## 命名约定

### 前端文件

- Vue组件: PascalCase (如 `ArticleCardView.vue`)
- Composables: camelCase + `use`前缀 (如 `useSummaryWebSocket.ts`)
- Store文件: camelCase (如 `api.ts`)
- 类型文件: camelCase (如 `article.ts`)
- 测试文件: `.test.ts`后缀 (如 `normalizeArticle.test.ts`)

### 后端文件

- Go文件: 小写，单词连接 (如 `auto_refresh.go`)
- Handler: `handler.go`
- Service: `service.go`
- 测试: `_test.go`后缀 (如 `service_test.go`)
- Model: 小写 (如 `feed.go`, `article.go`)
- Domain包: 小写单词 (如 `feeds`, `articles`, `digest`)

## 新增代码放置指南

### 新功能

**前端:**
- 组件: `front/app/features/{domain}/components/`
- Composable: `front/app/features/{domain}/composables/`
- API调用: `front/app/api/{domain}.ts`
- 类型: `front/app/types/{domain}.ts`

**后端:**
- Handler: `backend-go/internal/domain/{domain}/handler.go`
- Service: `backend-go/internal/domain/{domain}/service.go`
- Model: `backend-go/internal/domain/models/{entity}.go`
- 路由: 添加到 `backend-go/internal/app/router.go`

### 新调度器

- 定义: `backend-go/internal/jobs/{name}.go`
- 注册: `backend-go/internal/app/runtime.go`
- Handler: `backend-go/internal/jobs/handler.go`

### 新测试

**前端:**
- 单元测试: 与源文件同目录，`.test.ts`后缀
- E2E测试: `front/tests/e2e/`

**后端:**
- 单元测试: 与源文件同包，`_test.go`后缀
- 集成测试: `tests/workflow/` (Python)

### 工具类

**前端:**
- 工具函数: `front/app/utils/`
- 领域工具: `front/app/features/{domain}/utils/`

**后端:**
- 平台服务: `backend-go/internal/platform/{service}/`

## 特殊目录

### `.planning/`
- 用途: GSD工作流规划目录
- 内容: ROADMAP.md, phase目录, codebase分析
- 生成: 不手动编辑

### `docs/`
- 用途: 项目文档
- 注意: 混合旧文档和新文档，使用代码实际状态而非仅依赖文档

### `data/`
- 用途: 数据存储
- 内容: SQLite遗留文件
- PostgreSQL: 使用Docker volume

### `.gitnexus/`
- 用途: GitNexus代码索引
- 生成: 自动生成，不手动编辑

---

*结构分析: 2026-04-10*