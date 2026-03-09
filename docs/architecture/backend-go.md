# 后端架构

## 先说结论

后端现在已经完成了一轮目录归位。

当前真实结构已经是：

- `backend-go/internal/app/`
- `backend-go/internal/platform/`
- `backend-go/internal/domain/`
- `backend-go/internal/jobs/`

所以这份文档现在主要讲两件事：

1. 当前真实结构是什么
2. 这套结构各自负责什么

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
- 运行时共享状态：`backend-go/internal/app/runtimeinfo/schedulers.go`
- 配置加载：`backend-go/internal/platform/config/config.go`
- 数据库初始化：`backend-go/internal/platform/database/db.go`
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
│   │   └── runtimeinfo/
│   ├── domain/
│   │   ├── articles/
│   │   ├── categories/
│   │   ├── contentprocessing/
│   │   ├── digest/
│   │   ├── feeds/
│   │   ├── models/
│   │   ├── preferences/
│   │   └── summaries/
│   ├── jobs/
│   └── platform/
│       ├── ai/
│       ├── aisettings/
│       ├── config/
│       ├── database/
│       ├── middleware/
│       └── ws/
```

## 这些目录现在各管什么

### `cmd/`

- `server/` - HTTP 服务入口
- `migrate-digest/` - digest 表迁移工具
- `test-digest/` - digest 联调和测试入口

### `internal/app/`

这是应用壳层。

- `router.go` - 统一注册 HTTP 和 WebSocket 路由
- `runtime.go` - 启动 jobs、装配运行时、处理优雅退出
- `runtimeinfo/` - 暂存 scheduler 运行时引用，避免 job 和 domain 互相咬住

### `internal/platform/`

平台能力统一归这层。

- `config/` - 读取 `configs/config.yaml` 和默认值
- `database/` - 数据库初始化、建表、字段补丁
- `middleware/` - Gin 中间件
- `ws/` - WebSocket hub
- `ai/` - OpenAI 风格调用封装
- `aisettings/` - 共享 AI / Firecrawl 配置读写

### `internal/domain/`

业务域都收进这里。一个功能需要的 handler、service、model helper，优先都在同域里找。

- `categories/` - 分类管理
- `feeds/` - 订阅管理、刷新、OPML
- `articles/` - 文章列表、详情、状态更新
- `summaries/` - AI 设置、自动摘要配置、摘要队列
- `preferences/` - 阅读行为和偏好分析
- `contentprocessing/` - 内容补全、抓取、Firecrawl 相关处理
- `digest/` - digest 配置、生成器、导出器、手动运行
- `models/` - 共享 GORM 模型和公共格式化 helper

### `internal/jobs/`

定时任务执行壳统一放这里。

- `auto_refresh.go`
- `auto_summary.go`
- `content_completion.go`
- `firecrawl.go`
- `preference_update.go`
- `handler.go` - scheduler 状态和手动 trigger 接口

## 当前主要业务域

现在领域边界已经比之前清楚很多：

- `categories` - 分类管理
- `feeds` - 订阅管理和刷新
- `articles` - 文章读取、状态更新、统计
- `summaries` - AI 摘要和摘要队列
- `preferences` - 阅读行为与偏好分析
- `contentprocessing` - 内容补全、抓取、Firecrawl 开关与正文处理
- `digest` - 每日/每周汇总、飞书、Obsidian
- `platform/ws` - WebSocket 进度推送

## 当前结构的问题

当前主要问题已经从“目录混乱”变成“边界还不够干净”：

- `domain/models` 还是共享模型桶，后续可以继续按域拆细
- `runtimeinfo` 目前还是过渡层，后续可以继续收敛成更明确的 runtime container
- `aisettings` 仍在承载跨域配置，后续可以继续拆 ownership
- digest 测试已经过时，文档和验证时要单独看待

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

当前已经落到 `app / platform / domain / jobs` 四层结构。

### 后续优化方向

目录目标已经落地，后续更像是边界清理：

```text
backend-go/
├── cmd/
├── internal/
│   ├── app/
│   ├── platform/
│   ├── domain/
│   └── jobs/
```

职责保持成这样：

- `app/` - 服务装配、路由注册、启动流程、运行时
- `platform/` - 配置、数据库、中间件、WebSocket
- `domain/` - 按业务域收拢 handler、service、test，必要时共享 `models`
- `jobs/` - 定时任务执行壳

## 推荐的领域切分

当前已经这样切：

- `categories`
- `feeds`
- `articles`
- `summaries`
- `preferences`
- `contentprocessing`
- `digest`

`firecrawl` 目前仍保留在 `contentprocessing`，因为它还是内容增强链路的一段，不是单独业务面。

## 迁移原则

- 先让文档说真话
- 先保 API 路径不变
- 先保入口和路由稳定
- 再清理共享配置和 runtime 边界
- 每次结构调整都同步更新 `docs/`

## 建议阅读顺序

- 先看 `docs/architecture/backend-runtime.md`
- 再看 `docs/guides/content-processing.md`
- 再看 `docs/guides/digest.md`
- 最后回到具体源码目录
