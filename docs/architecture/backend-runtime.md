# 后端运行与接口

## 启动主线

当前 Go 后端的真实启动顺序在 `backend-go/cmd/server/main.go`：

1. 加载配置 `backend-go/internal/config/config.go`
2. 初始化数据库 `backend-go/pkg/database/db.go`
3. 执行 digest 迁移 `backend-go/internal/digest`
4. 创建 Gin 实例并挂载 CORS
5. 注册路由 `backend-go/internal/app/router.go`
6. 启动运行时 `backend-go/internal/app/runtime.go`
7. 注册优雅退出
8. 监听 `:5000`

这说明 `cmd/server` 现在更像一个薄入口，真正的装配工作已经开始往 `internal/app/` 收口。

## 当前运行时职责

`backend-go/internal/app/runtime.go` 负责把后台任务真正拉起来，并把运行时实例注入给 handler。

当前会启动 6 类后台任务：

- 自动刷新 `auto_refresh`
- 自动摘要 `auto_summary`
- 偏好更新 `preference_update`
- 内容补全 `content_completion`
- Firecrawl 抓取 `firecrawl`
- digest 定时任务 `digest`

其中内容补全还会先初始化抓取服务地址，默认读取 `CRAWL_SERVICE_URL`，未设置时回落到 `http://localhost:11235`。

## 路由分层

当前 HTTP 与 WebSocket 接口主要集中在 `backend-go/internal/app/router.go`。

### 基础路由

- `GET /` - API 简要说明
- `GET /health` - 健康检查
- `GET /api/tasks/status` - 当前任务队列占位状态
- `GET /ws` - WebSocket 连接入口

### API 路由组

- `/api/categories` - 分类 CRUD
- `/api/feeds` - 订阅 CRUD、单条查询、刷新、批量刷新、预览拉取
- `/api/articles` - 文章列表、详情、统计、状态更新、批量更新
- `/api/ai` - AI 设置、连通性测试、单篇摘要
- `/api/summaries` - 摘要列表、详情、删除、队列提交、队列状态、任务查询
- `/api/reading-behavior` - 阅读行为记录与统计
- `/api/user-preferences` - 偏好查询与手动更新
- `/api/content-completion` - 单文章补全、整 feed 补全、补全状态、总览
- `/api/firecrawl` - 单文章抓取、feed 级启用、状态、配置保存
- `/api/digest` - digest 配置、状态、预览、手动运行、飞书测试、Obsidian 测试

## WebSocket 位置

WebSocket 逻辑在 `backend-go/internal/ws/hub.go`。

它现在属于一类明显的“平台能力”：

- 给前端推送异步任务进度
- 服务摘要和抓取类后台任务
- 不属于某个单独业务域，但会被多个业务域复用

这也是后续把它收进 `internal/platform/ws/` 的直接理由。

## 后台任务现状

当前后台任务代码仍散落在多个位置：

- `backend-go/internal/schedulers/` - 多数定时任务外壳
- `backend-go/internal/services/` - 任务实际业务逻辑
- `backend-go/internal/digest/` - digest 自带调度与生成逻辑
- `backend-go/internal/handlers/scheduler.go` - 手动触发与状态查询接口

这里的主要问题不是“不能跑”，而是理解成本高：

- 看一个任务要在 scheduler、service、handler 之间来回跳
- digest 拥有半独立实现，和其他任务的风格不完全一致
- scheduler handler 中仍有部分占位实现

## 现在能看到什么状态

`auto_refresh` 和 `auto_summary` 现在都会把最近一轮执行结果写回 `scheduler_tasks`。

这几类信息可以直接从后端状态接口拿到：

- 上次执行时间
- 执行耗时
- 下次执行时间
- 最近一轮摘要
- 手动 trigger 是真的开始了，还是被拒绝了

其中：

- `auto_refresh` 会记录扫描了多少 feed、多少 feed 到点、真正触发了多少刷新、多少 feed 已经在刷新中
- `auto_summary` 会记录这轮看了多少 feed、产出多少总结、跳过多少、失败多少

## 当前限制

现在有几块必须在文档里讲清：

- scheduler 接口不是全部完整实现
- `ResetSchedulerStats` 还是 placeholder
- `UpdateSchedulerInterval` 还是 placeholder

已经补齐的部分：

- `auto_summary` 的手动触发现在会真实启动一轮执行，或者明确返回为什么没启动
- `auto_refresh` 的状态不再只是“看起来在跑”，而是会持续更新最近一轮执行摘要

所以文档里不能再写“功能完全对等”或“全部能力已闭环”。

## 目录重组怎么落

当前推荐把后端重组理解成四层，而不是一次大搬家：

- `internal/app/` - 装配、路由、运行时
- `internal/platform/` - 配置、数据库、中间件、WebSocket
- `internal/domain/` - 按业务域收拢 handler、service、model、test
- `internal/jobs/` - 定时任务执行外壳

这个结构现在还没完全实现，但 `internal/app/` 已经是第一步。

## 读代码建议

如果你是第一次进这个后端，建议按这个顺序读：

1. `backend-go/cmd/server/main.go`
2. `backend-go/internal/app/router.go`
3. `backend-go/internal/app/runtime.go`
4. `backend-go/internal/handlers/*`
5. `backend-go/internal/services/*`
6. `backend-go/internal/models/*`
7. `backend-go/internal/schedulers/*`
8. `backend-go/internal/digest/*`
