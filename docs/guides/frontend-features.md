# 前端功能说明

## 页面入口

- 主阅读页：`/`
- Digest 总览：`/digest`
- Digest 单视图：`/digest/daily`、`/digest/weekly`

## 主阅读页

主阅读页是三栏结构。

- 左栏：导航、分类、feed、快捷入口
- 中栏：文章列表或 AI 总结列表
- 右栏：文章正文或 AI 总结详情

### 顶部工具栏

支持这些动作：

- 刷新当前 feed 或全部 feed
- 全部标记为已读
- 新增订阅源
- 新增分类
- 导入 OPML
- 打开全局设置

### 左侧导航

支持这些入口：

- 全部文章
- 收藏
- AI 总结
- Digest
- 分类树
- feed 列表

分类和 feed 的点击会驱动主壳重新拉取对应文章。

## Feed 与分类管理

### 分类

- 新增分类
- 编辑分类
- 删除分类
- 分类可配置名称、图标、颜色、描述

删除分类时，主界面文案明确说明“不删除该分类下订阅源”。

### Feed

- 新增 feed
- 编辑 feed
- 删除 feed
- 单 feed 手动刷新
- 全量刷新
- OPML 导入
- OPML 导出

feed 还带这些能力开关：

- `ai_summary_enabled`
- `content_completion_enabled`
- `completion_on_refresh`
- `max_completion_retries`
- `firecrawl_enabled`

## 文章阅读

正文区由 `ArticleContentView.vue` 驱动。

### 基础阅读能力

- 展示文章标题、时间、作者、封面
- 收藏 / 取消收藏
- 打开原文
- 上一篇 / 下一篇
- 全屏阅读
- 预览模式和 iframe 模式切换

### 已读与阅读行为

- 打开文章时自动标记已读
- 跟踪打开、关闭、滚动、收藏、取消收藏
- 30 秒批量上传一次阅读行为
- 累积 10 条事件也会触发上传

### 内容增强

如果后端已配置内容增强能力，正文区会展示状态面板。

- Firecrawl 抓取状态
- AI 整理状态
- 抓取时间、总结时间、尝试次数
- 手动抓取全文
- 手动生成整理稿

### 内容源切换

当原始内容和 Firecrawl 全文都存在时，正文区支持切换：

- 原始内容
- Firecrawl 全文

如果已经有 `aiContentSummary`，会优先展示 AI 整理稿。

## AI 总结

AI 总结列表在主阅读页中栏展示，详情在右栏展示。

### 列表能力

- 按分类过滤
- 按 feed 过滤
- 按日期过滤
- 快捷日期范围筛选
- 分页
- 删除总结

### 生成能力

- 可选择时间窗口
- 发起多分类批量总结任务
- 通过 WebSocket 实时显示队列进度
- 支持失败任务错误展开

### 依赖

AI 总结依赖全局 AI 设置中的：

- `baseURL`
- `apiKey`
- `model`

如果没有配置 API Key，列表顶部会给出提示。

## Digest

Digest 是独立页面，不嵌在主阅读壳里。

### Digest 总览

- 支持日报 / 周报切换
- 支持按日期切换锚点
- 支持刷新当前版面
- 支持立即执行
- 支持查看任务状态

### Digest 详情

- 左栏看分类与运行状态
- 中栏看 feed 级 AI 总结
- 右栏看总结正文

### 关联文章

每条 digest summary 都可以拉取关联文章。

- 点击关联文章后弹窗阅读
- 弹窗里复用 `ArticleContentView`
- 仍保留收藏、抓取、整理等动作

### 设置项

Digest 设置支持：

- 日报开关和时间
- 周报开关、星期和时间
- 飞书推送开关与 webhook
- Obsidian 导出开关与 vault 路径
- 测试飞书
- 测试 Obsidian

## 视觉与交互约束

前端当前不是标准 SaaS 模板风格，文档和实现要保持一致。

- 主色为 Ink Blue，不用紫色
- 强调色为 Print Red
- 背景带纸张质感和渐变
- 交互动效以短促、克制为主
- Digest 页面允许更强烈的版式设计

## 当前已落地但容易忽略的点

- `apiStore` 是主数据源，不要再写副本同步逻辑
- `feedsStore`、`articlesStore` 是派生视图
- WebSocket 只用于 AI 总结队列进度
- Digest 详情里的文章弹窗直接复用主阅读组件
- 文章内容源切换和 Firecrawl 状态已经进主阅读链路
