# 后端架构

## 先说结论

这份文档只描述当前 `backend-go/` 已经落地的真实结构，不再沿用旧的预期分层。

当前后端可以直接按四层理解：

- `cmd/`：启动入口和辅助命令
- `internal/app/`：应用装配、路由注册、运行时启动与退出
- `internal/platform/`：数据库、配置、AI 路由、WebSocket、共享基础设施
- `internal/domain/` + `internal/jobs/`：业务域能力与后台调度执行壳

如果你发现文档和代码不一致，优先相信源码入口：`backend-go/cmd/server/main.go`、`backend-go/internal/app/router.go`、`backend-go/internal/app/runtime.go`。

## 技术栈

- Go 1.21
- Gin
- GORM
- SQLite
- Viper
- Gorilla WebSocket
- robfig/cron

## 当前真实入口

- 服务入口：`backend-go/cmd/server/main.go`
- 路由装配：`backend-go/internal/app/router.go`
- 运行时启动：`backend-go/internal/app/runtime.go`
- 运行时共享引用：`backend-go/internal/app/runtimeinfo/schedulers.go`
- 配置加载：`backend-go/internal/platform/config/config.go`
- 数据库初始化与表补丁：`backend-go/internal/platform/database/db.go`
- 配置文件：`backend-go/configs/config.yaml`

## 当前目录现实

```text
backend-go/
├── cmd/
│   ├── migrate-digest/
│   ├── migrate-tags/
│   ├── server/
│   └── test-digest/
├── configs/
├── internal/
│   ├── app/
│   │   └── runtimeinfo/
│   ├── domain/
│   │   ├── aiadmin/
│   │   ├── articles/
│   │   ├── categories/
│   │   ├── contentprocessing/
│   │   ├── digest/
│   │   ├── feeds/
│   │   ├── models/
│   │   ├── preferences/
│   │   ├── summaries/
│   │   ├── topicanalysis/
│   │   ├── topicextraction/
│   │   ├── topicgraph/
│   │   └── topictypes/
│   ├── jobs/
│   └── platform/
│       ├── ai/
│       ├── airouter/
│       ├── aisettings/
│       ├── config/
│       ├── database/
│       ├── middleware/
│       ├── opennotebook/
│       └── ws/
```

## 分层职责

### `cmd/`

- `server/`：HTTP 服务真实入口
- `migrate-digest/`：digest 配置/表迁移命令
- `migrate-tags/`：主题标签相关迁移命令
- `test-digest/`：digest 联调入口

### `internal/app/`

这是应用壳层，负责把平台能力、业务域和后台任务接起来。

- `router.go`：注册 HTTP API 和 WebSocket 路由
- `runtime.go`：启动 scheduler、初始化内容补全服务、注册优雅退出
- `runtimeinfo/`：临时保存运行时实例，给 handler 查询状态或触发任务

这里要注意：`runtimeinfo` 还是过渡方案，它不是完整的 runtime container。

### `internal/platform/`

这是共享基础设施层，不承载具体业务语义。

- `config/`：读取 `configs/config.yaml`
- `database/`：初始化 SQLite、建表、索引、字段补丁
- `middleware/`：Gin 中间件，例如 CORS
- `ws/`：WebSocket hub，给前端推送异步任务状态
- `ai/`：AI 调用封装
- `airouter/`：AI provider、capability route、failover 路由
- `aisettings/`：兼容旧配置表的 AI / Firecrawl / Open Notebook 配置读写
- `opennotebook/`：Open Notebook 客户端能力

### `internal/domain/`

业务能力按域组织，handler 和 service 主要都放在域目录里。

- `aiadmin/`：AI provider 与 capability route 管理
- `categories/`：分类 CRUD
- `feeds/`：订阅 CRUD、刷新、OPML、RSS 解析
- `articles/`：文章列表、详情、状态更新、统计
- `summaries/`：摘要列表、单篇摘要、自动摘要配置、摘要队列
- `preferences/`：阅读行为记录与偏好分析
- `contentprocessing/`：内容补全、Firecrawl 配置与抓取、文章正文处理
- `digest/`：digest 配置、预览、手动运行、导出、定时调度
- `topictypes/`：主题图谱共享类型和窗口工具
- `topicextraction/`：摘要/文章标签提取
- `topicanalysis/`：主题分析任务与分析结果 API
- `topicgraph/`：主题图谱、主题详情、主题相关文章查询
- `models/`：共享 GORM 模型和部分格式化 helper

### `internal/jobs/`

这里是调度外壳，不放完整业务，只负责定时触发和运行状态记录。

- `auto_refresh.go`：扫描到点 feed 并异步触发刷新
- `auto_summary.go`：按 feed 聚合近时间窗文章并生成 `ai_summaries`
- `content_completion.go`：对 `firecrawl completed + summary incomplete` 的文章做内容补全
- `firecrawl.go`：轮询待抓取文章并执行 Firecrawl
- `preference_update.go`：阅读偏好更新任务
- `handler.go`：部分 scheduler 状态查询与手动触发 API

## 当前主要子系统

### 订阅与文章

`feeds` 和 `articles` 是基础数据面。

- feed 刷新负责拉 RSS、去重、入库 article
- article 记录承接后续 Firecrawl、内容补全、摘要、主题分析
- feed 上的 `firecrawl_enabled`、`article_summary_enabled` 会直接影响文章入库后的状态初始化

### AI 与内容增强

这部分不再只是一个“AI 摘要开关”，而是三层叠加：

- `platform/airouter`：管理 provider 和 capability route
- `domain/contentprocessing`：正文抓取、内容补全、Firecrawl 配置
- `domain/summaries` + `jobs/auto_summary.go`：按订阅批量生成摘要

### Digest

digest 现在已经是正式子系统，不是边角工具。

- 支持 daily / weekly 两类时间窗
- 支持配置查询、预览、手动执行、定时执行
- 支持 Feishu、Obsidian、Open Notebook 三条输出链路
- digest 配置更新后会尝试热重载 `DigestScheduler`

### 主题图谱

主题能力已经拆成四个包：

- `topictypes`：共享类型和窗口解析
- `topicextraction`：从摘要/文章提取 topic tag
- `topicanalysis`：生成并查询 topic analysis
- `topicgraph`：返回图谱节点边、详情、相关文章、相关 digest

依赖方向大致是：

```text
topictypes
    ↑
    ├── topicgraph
    ├── topicanalysis
    └── topicextraction -> topicanalysis
```

## 数据模型重点

旧文档只写 feed/article 基础字段已经不够，当前后端的数据面至少包含这些正式能力。

### `feeds`

- `article_summary_enabled`
- `completion_on_refresh`
- `max_completion_retries`
- `firecrawl_enabled`
- `refresh_interval`
- `refresh_status`

### `articles`

- `image_url`
- `summary_status`
- `summary_generated_at`
- `ai_content_summary`
- `completion_attempts`
- `completion_error`
- `firecrawl_status`
- `firecrawl_content`
- `firecrawl_error`
- `firecrawl_crawled_at`

### 其他关键表/模型

- `ai_settings`：兼容旧配置存储
- `ai_providers` / `ai_routes` / `ai_route_providers`：AI 路由配置
- `ai_summaries`：按 feed/分类聚合后的摘要
- `scheduler_tasks`：scheduler 最近执行状态、耗时、错误、结果摘要
- `digest_configs`：digest 配置
- 主题图谱相关模型：`topic_tags`、`topic_tag_analyses`、`topic_tag_embeddings` 等

## 真实 API 面

`internal/app/router.go` 当前已经注册这些主路由组：

- `/api/categories`
- `/api/feeds`
- `/api/articles`
- `/api/ai`
- `/api/summaries`
- `/api/schedulers`
- `/api/reading-behavior`
- `/api/user-preferences`
- `/api/content-completion`
- `/api/firecrawl`
- `/api/topic-graph`
- `/api/digest`
- `/api/import-opml` / `/api/export-opml`
- `/ws`

其中 `topic-graph` 组下面还挂了 `analysis` 子路由，AI 管理则已经扩展到 provider 和 route 级别，而不是只有“摘要设置”一个入口。

## 具体数据链路示例

下面这几条链路是当前代码里真实存在、而且最值得在阅读代码时重点跟的主线。

### 用例 1：自动刷新 feed -> 新文章入库

场景：用户给某个 feed 配了刷新间隔，或者手动触发 `/api/schedulers/auto_refresh/trigger`。

链路：

1. `internal/jobs/auto_refresh.go` 扫描 `refresh_interval > 0` 的 feed
2. 到点 feed 调用 `feeds.FeedService.RefreshFeed`
3. `RefreshFeed` 通过 RSS parser 拉取源站内容并更新 feed 元信息
4. 新 entry 去重后写入 `articles`
5. `buildArticleFromEntry` 按 feed 配置初始化文章状态：
   - 默认 `summary_status = complete`
   - 如果 feed 开启 `firecrawl_enabled`，则文章先标记 `firecrawl_status = pending`
   - 如果同时开启 `article_summary_enabled`，则文章再标记 `summary_status = incomplete`
6. `cleanupOldArticles` 按 `max_articles` 清理旧文章（收藏文章跳过）

这个链路的关键点是：feed 刷新不只是在“加文章”，它还会把后续 Firecrawl / 内容补全链路需要的状态位一起种进去。

### 用例 2：Firecrawl 抓正文 -> 内容补全生成文章摘要

场景：某个 feed 开启了 Firecrawl，前面的刷新流程已经把文章打成 `firecrawl_status = pending`。

链路：

1. `jobs.FirecrawlScheduler` 轮询待抓取文章
2. Firecrawl 成功后，文章被更新为：
   - `firecrawl_status = completed`
   - `firecrawl_content` 写入抓取正文
   - `summary_status = incomplete`
3. `jobs.ContentCompletionScheduler` 定时查询：
   - `articles.firecrawl_status = completed`
   - `articles.summary_status = incomplete`
   - `feeds.article_summary_enabled = true`
4. `contentprocessing.ContentCompletionService.CompleteArticle` 基于 Firecrawl 正文生成 `ai_content_summary`
5. 文章最终更新为完成态，并记录失败次数、错误信息、最近处理文章等状态
6. 前端可通过 `/api/content-completion/overview` 和 `/api/content-completion/articles/:article_id/status` 看到结果

这条链路说明：运行时对外现在用 `content_completion` 作为规范 scheduler 名字，但仍兼容旧别名 `ai_summary`；它对应的是“文章级内容补全”，不是 `ai_summaries` 表里的 feed 聚合摘要。

### 用例 3：自动摘要 -> 主题标签 -> 主题分析

场景：某个 feed 开启了 `ai_summary_enabled`，系统按时间窗自动生成订阅摘要。

链路：

1. `jobs.AutoSummaryScheduler` 扫描 `ai_summary_enabled = true` 的 feed
2. 按 `time_range` 取最近文章，并按最多 20 篇分 batch
3. `summaries.NewAISummaryPromptBuilder` 结合偏好服务组装 prompt
4. 通过 `airouter` 调用 summary capability 对应 provider
5. 结果写入 `ai_summaries`
6. 新摘要写入后调用：
   - `topicextraction.TagSummary(&aiSummary)`
   - `topicextraction.BackfillArticleTags(batch, feedName, categoryName)`（仅兜底补齐 article tags）
7. topic extraction 产出的标签会继续驱动 `topicanalysis` 的分析任务
8. 前端再通过 `/api/topic-graph/*` 和 `/api/topic-graph/analysis/*` 读取图谱和分析结果

这条链路把“摘要生成”和“主题图谱”真正串起来了：图谱并不是独立系统，而是建立在摘要和文章标签结果之上的展示与分析层。

### 用例 4：digest 预览 / 手动运行 / 定时输出

场景：用户查看日报预览、手动执行日报，或者等系统到了定时点自动推送。

链路：

1. `/api/digest/preview/:type` 调用 `digest.buildPreview`
2. `DigestGenerator` 按 daily / weekly 时间窗聚合 `ai_summaries`
3. 后端组装：
   - 分类视图数据
   - markdown 预览正文
   - 默认选中的分类和 summary
   - 每条 digest summary 对应 article 的 `aggregated_tags` 索引
4. `/api/digest/run/:type` 会在预览结果基础上继续执行输出：
   - Feishu 推送
   - Obsidian 导出
   - Open Notebook 总结
5. `DigestScheduler` 则按 `digest_configs` 中的 daily / weekly 配置走同一类生成逻辑
6. `/api/schedulers/status` 已经会带上 digest 的统一状态视图，`/api/digest/status` 仍保留 digest 专用状态，`/api/digest/config` 和 `/api/digest/open-notebook/config` 返回配置状态

这条链路的重点是：digest 不是简单拼 markdown，而是一个“聚合 + 预览 + 多出口分发”的完整运行链。

补充：digest 不再被视为“单独打 tag 的对象”，topic graph 和 digest 页里展示的 digest tags 来自其覆盖 article 的 `article_topic_tags` 聚合结果；标准文章详情接口 `/api/articles/:id` 也会直接返回 article tags，供前端通用展示。

### Article 打标签时机

文章标签现在按以下规则运行：

1. 普通 refresh 新文章：入库后立即打标签（feed 未开启 Firecrawl 时）
2. 若 feed 开启了 `Firecrawl`：refresh 阶段先不打标签
	- Firecrawl 抓取完成后，写入 `tag_jobs` 队列，由 `TagQueue` worker 异步执行重新打标签
	- 若 feed 同时开启了 `自动补全`（`article_summary_enabled`），则由 ContentCompletion scheduler 在生成 `AIContentSummary` 后同样 enqueue `tag_jobs`
3. `auto_summary` / `summary_queue` 阶段：只做兜底补齐，只处理当前没有 article tags 的文章
4. 前端文章详情支持手动打标签 / 重新打标签，接口为 `POST /api/articles/:article_id/tags`
	- 手动接口现在只 enqueue 队列并返回 `job_id`
	- `TagQueue` 完成后通过 WebSocket 广播 `tag_completed`
5. `TagQueue.Start()` 首次启动失败时不会阻塞应用；它会后台按 30 秒间隔重试最多 10 次

当前正文提取优先级为：

- `AIContentSummary`
- `FirecrawlContent`
- `Content`
- `Description`

因此完整补全链路的文章会优先基于 AI 整理稿或 Firecrawl 全文得到标签，而 summary 阶段不再承担文章打标签主流程。

## 当前边界上的真实问题

当前问题已经不是“目录乱”，而是这些边界还在过渡：

- `runtimeinfo` 仍是全局变量式共享引用，适合过渡，不适合长期扩展
- `domain/models` 仍是共享模型桶，后续还可以继续收敛 ownership
- `aisettings` 同时承担兼容旧配置和新配置落库，职责偏宽
- `runtimeinfo` 仍是全局变量式共享引用，但当前至少已经把实际启动的 scheduler 全部挂进统一入口
- `/api/tasks/status` 现在是聚合视图，不是通用任务编排系统；它反映的是 summary queue、内容补全、firecrawl 三类后台工作

## 推荐阅读顺序

- 先看 `docs/architecture/backend-runtime.md`
- 再看 `backend-go/cmd/server/main.go`
- 再看 `backend-go/internal/app/router.go`
- 再看 `backend-go/internal/app/runtime.go`
- 再按用例追具体域：`feeds` -> `contentprocessing` -> `summaries` -> `digest` -> `topic*`
