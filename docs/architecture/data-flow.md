# 数据流

## 主链路

```text
RSS 源
  -> backend-go 拉取和解析
  -> SQLite 持久化
  -> 可选全文抓取 / 内容补全 / AI 总结 / Digest 聚合
  -> 前端通过 app/api 拉取
  -> apiStore 映射为前端模型
  -> 派生 store 和 feature 组件消费
  -> UI 渲染
```

## 前端数据流

```text
page
  -> feature shell / feature view
  -> app/api/*
  -> backend API
  -> useApiStore
  -> useFeedsStore / useArticlesStore / usePreferencesStore
  -> 组件渲染
```

## 前端状态职责

### `useApiStore`

主数据源。

- 拉分类
- 拉 feed
- 拉文章
- 执行分类、feed、文章相关 CRUD
- 处理 OPML 导入导出
- 处理 AI 总结接口
- 初始化应用启动数据

### `useFeedsStore`

派生订阅视图。

- feed 分组
- 分类视图
- feed 未读数

### `useArticlesStore`

派生文章视图。

- 当前筛选条件
- 当前文章
- 已读 / 收藏统计
- 文章列表排序与过滤

### `usePreferencesStore`

阅读偏好相关状态。

- 读取偏好分数
- 读取阅读统计
- 手动触发偏好更新

## 字段映射规则

- 后端响应保留 `snake_case`
- 前端内部统一 `camelCase`
- 前端的 `id` 统一转成 `string`
- 转换集中在 API 模块和 `useApiStore`
- 组件层不应散落字段映射逻辑

## 主阅读页交互流

### 应用启动

```text
app.vue mounted
  -> apiStore.initialize()
  -> Promise.all(fetchCategories, fetchFeeds, fetchArticles)
  -> FeedLayoutShell 渲染
```

### 切分类

```text
AppSidebar
  -> FeedLayoutShell.handleCategoryClick()
  -> apiStore.fetchFeeds(...)
  -> apiStore.fetchArticles(...)
  -> 列表栏和正文区响应更新
```

### 切 feed

```text
AppSidebar
  -> FeedLayoutShell.handleFeedClick()
  -> apiStore.fetchArticles(feed_id)
  -> apiStore.refreshFeed(feed_id)
  -> 轮询 refresh_status
  -> 刷新完成后再拉文章
```

### 打开文章

```text
ArticleListPanel
  -> ArticleContentView
  -> apiStore.markAsRead()
  -> useReadingTracker 记录 open / scroll / close / favorite
  -> reading_behavior 接口批量上报
```

## 文章内容增强流

### Firecrawl / 内容补全状态

```text
ArticleContentView
  -> useContentCompletion.getCompletionStatus(articleId)
  -> /content-completion/articles/:id/status
  -> UI 展示抓取状态、整理状态、错误信息
```

### 手动抓取全文

```text
ArticleContentView
  -> useFirecrawlApi.crawlArticle(articleId)
  -> 后端执行抓取
  -> 再次查询 completion status
  -> 更新 article.firecrawlContent / firecrawlStatus
```

### 手动生成整理稿

```text
ArticleContentView
  -> completeArticle(articleId, { force: true })
  -> 后端生成 ai_content_summary
  -> 再次查询 completion status
  -> UI 渲染整理稿
```

## AI 总结流

```text
AISummariesListView
  -> apiStore.submitQueueSummary()
  -> backend 创建批次任务
  -> useSummaryWebSocket.connect()
  -> /ws 推送进度
  -> 批次完成后 fetchSummaries()
  -> 右栏显示 AISummaryDetailView
```

## Digest 流

```text
DigestListView
  -> getStatus()
  -> getPreview(daily|weekly, date)
  -> 左栏分类 + 中栏 summary 列表 + 右栏详情
  -> runNow() 可立即生成新版本
  -> DigestDetail 按 article_ids 拉关联文章
  -> 关联文章在弹窗中复用 ArticleContentView
```

## 定时任务链路

- feed 自动刷新
- Firecrawl / 内容补全处理
- AI 总结批量生成
- Digest 日报 / 周报生成
- 阅读偏好聚合任务

### scheduler 状态回传

```text
GlobalSettingsDialog.schedulers tab
  -> useSchedulerApi.getSchedulersStatus()
  -> /api/schedulers/status
  -> backend 返回 database_state + last_run_summary + is_executing
  -> UI 渲染 auto_refresh / auto_summary / ai_summary / firecrawl 状态卡
```

### 手动 trigger 链路

```text
GlobalSettingsDialog.schedulers tab
  -> useSchedulerApi.triggerScheduler(name)
  -> POST /api/schedulers/:name/trigger
  -> backend 判断 accepted / started / reason / message
  -> 前端显示真实反馈，不再只看 HTTP 200
  -> 短周期轮询刷新最新状态
```

### `auto_refresh` 状态流

```text
auto_refresh scheduler
  -> 扫描 refresh_interval > 0 的 feed
  -> 判断是否到点
  -> 标记 feed.refresh_status=refreshing
  -> 异步调用 feedService.RefreshFeed()
  -> 把扫描数 / 到点数 / 触发数 / 已在刷新数写回 scheduler_tasks.last_execution_result
```

### `auto_summary` 状态流

```text
auto_summary scheduler
  -> 读取 AI 配置
  -> 扫描 ai_summary_enabled=true 的 feed
  -> 聚合近 time_range 内文章
  -> 调 AI 生成 summary
  -> 把 feed 数 / 生成数 / 跳过数 / 失败数写回 scheduler_tasks.last_execution_result
  -> 手动 trigger 时也走同一套执行链路
```

## 约束

- 不再维护本地镜像数组同步链
- 不再使用 `syncToLocalStores()`
- 组件层优先消费已映射好的前端模型
- 与后端交互的细节只应停留在 `app/api` 和 store
