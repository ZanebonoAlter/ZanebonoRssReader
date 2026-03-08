# 后端架构

## 先说结论

后端现在已经不是一套“纯横向分层”的老结构了。

`backend-go/internal/app/` 已经开始承担启动装配、路由注册和运行时管理，但大部分业务代码仍主要分散在：

- `backend-go/internal/handlers/`
- `backend-go/internal/services/`
- `backend-go/internal/models/`
- `backend-go/internal/schedulers/`

所以这份文档要同时讲两件事：

1. 当前真实结构是什么
2. 后续要往什么结构迁

## 技术栈

- Go 1.21
- Gin
- GORM
- SQLite
- Viper
- Gorilla WebSocket

## 当前真实入口

- 服务入口：`backend-go/cmd/server/main.go`
- 路由装配：`backend-go/internal/app/router.go`
- 运行时装配：`backend-go/internal/app/runtime.go`
- 配置加载：`backend-go/internal/config/config.go`
- 数据库初始化：`backend-go/pkg/database/db.go`
- 配置文件：`backend-go/configs/config.yaml`

如果你要看“服务怎么启动起来”，优先看 `cmd/server` 和 `internal/app`，不要再只盯着 `main.go`。

## 当前目录现实

```text
backend-go/
├── cmd/
│   ├── migrate-digest/
│   ├── server/
│   └── test-digest/
├── configs/
├── internal/
│   ├── app/
│   ├── config/
│   ├── digest/
│   ├── handlers/
│   ├── middleware/
│   ├── models/
│   ├── schedulers/
│   ├── services/
│   └── ws/
└── pkg/
    └── database/
```

## 这些目录现在各管什么

### `cmd/`

- `server/` - HTTP 服务入口
- `migrate-digest/` - digest 表迁移工具
- `test-digest/` - digest 联调和测试入口

### `internal/app/`

这是已经开始成型的应用壳层。

- `router.go` - 统一注册 HTTP 和 WebSocket 路由
- `runtime.go` - 启动 scheduler、注入 runtime 依赖、处理优雅退出

### `internal/config/`

负责读取 `configs/config.yaml` 和默认值。

当前是“配置文件 + 默认值”模式，不要把它写成支持完整环境变量覆盖或热重载。

### `internal/handlers/`

HTTP 接口层。现在大部分业务功能都从这里暴露：

- 分类
- 订阅
- 文章
- AI 设置与摘要
- 阅读行为
- 内容补全
- Firecrawl
- digest
- scheduler 管理

### `internal/services/`

主要业务逻辑层。当前已经不是只做 RSS 解析，还包括：

- feed 刷新
- AI 摘要
- 偏好分析
- 内容补全
- Firecrawl 集成
- 摘要队列

### `internal/models/`

GORM 模型定义。

除了基础的 `Category`、`Feed`、`Article`，现在还包含：

- `AISummary`
- `AISummaryFeed`
- `SchedulerTask`
- `AISettings`
- `ReadingBehavior`
- `UserPreference`

### `internal/schedulers/`

多数后台任务的执行壳在这里。

当前已覆盖：

- 自动刷新
- 自动摘要
- 偏好更新
- 内容补全
- Firecrawl

### `internal/digest/`

digest 是一个半独立子系统，自己带：

- 模型
- 迁移
- 生成器
- 调度器
- Feishu / Obsidian 输出

### `internal/ws/`

WebSocket hub，主要服务异步任务进度广播。

### `pkg/database/`

数据库初始化和表保证逻辑。

这个位置现在还能用，但从结构上看，后续更适合被收进 `internal/platform/database/`。

## 当前主要业务域

虽然代码还没完全按领域收口，但现在实际已经能看出这些业务块：

- `categories` - 分类管理
- `feeds` - 订阅管理和刷新
- `articles` - 文章读取、状态更新、统计
- `summaries` - AI 摘要和摘要队列
- `preferences` - 阅读行为与偏好分析
- `content-processing` - 内容补全与正文处理
- `firecrawl` - 抓取全文和 feed 级能力开关
- `digest` - 每日/每周汇总、飞书、Obsidian
- `realtime` - WebSocket 进度推送

## 当前结构的问题

当前痛点不是“完全没层次”，而是“层次和领域混在一起”：

- 看一个功能仍要跨 handler、service、model、scheduler 几层跳
- digest 形成了自己的小体系，和其他能力的组织方式不一致
- `internal/app/` 已经出现，但平台层还没跟上
- 数据库、配置、WebSocket 这些平台能力还没统一归位

## 数据模型已经变了

旧文档只写基础 feed/article 字段已经不够了。

当前模型里已经明确出现这些新增能力字段：

- `feeds.content_completion_enabled`
- `feeds.completion_on_refresh`
- `feeds.max_completion_retries`
- `feeds.firecrawl_enabled`
- `articles.image_url`
- `articles.content_status`
- `articles.full_content`
- `articles.ai_content_summary`
- `articles.firecrawl_status`
- `articles.firecrawl_content`

同时数据库里还存在：

- `ai_summary_queue`
- `digest_configs`

所以后端文档必须把“内容处理链路”和“digest 子系统”当成正式能力，而不是边角说明。

## 当前状态和目标状态要分开写

### 当前状态

当前是“开始往 app 壳层收拢，但业务仍以横向目录为主”的混合结构。

### 目标状态

后续目录重组的目标仍然是：

```text
backend-go/
├── cmd/
├── internal/
│   ├── app/
│   ├── platform/
│   ├── domain/
│   └── jobs/
```

建议职责保持成这样：

- `app/` - 服务装配、路由注册、启动流程、运行时
- `platform/` - 配置、数据库、中间件、WebSocket
- `domain/` - 按业务域收拢 handler、service、model、test
- `jobs/` - 定时任务执行壳

## 推荐的领域切分

未来可以按这些 domain 收：

- `categories`
- `feeds`
- `articles`
- `summaries`
- `preferences`
- `content-processing`
- `digest`

如果 `firecrawl` 继续扩张，可以独立成域；如果仍然只是内容链路的一段，就保留在 `content-processing`。

## 迁移原则

- 先让文档说真话
- 先保 API 路径不变
- 先收平台能力，再收业务域
- 先做目录归位，再做深层逻辑拆分
- 每迁一块，就同步更新 `docs/`

## 建议阅读顺序

- 先看 `docs/architecture/backend-runtime.md`
- 再看 `docs/guides/content-processing.md`
- 再看 `docs/guides/digest.md`
- 最后回到具体源码目录
