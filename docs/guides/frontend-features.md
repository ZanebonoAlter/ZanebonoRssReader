# 前端功能说明

## 入口

- 首页入口：`front/app/pages/index.vue`
- Digest 入口：`front/app/pages/digest/index.vue`
- 主阅读壳：`front/app/features/shell/components/FeedLayoutShell.vue`

## 主阅读器

- 顶部栏负责刷新、全部已读、添加 feed、添加分类、导入 OPML、打开全局设置
- 左侧栏负责全部文章、收藏夹、AI 总结、Digest、分类树和 feed 列表切换
- 中间栏负责文章列表、日期筛选、分页、feed 状态展示
- 右侧栏负责正文阅读、原文 iframe、收藏、上下篇切换、Firecrawl 全文和 AI 整理稿切换

## Feed 与分类管理

- 分类和 feed 的读取都走 `front/app/stores/api.ts`
- 分类点击会重新拉取该分类下的 feeds 和 articles
- feed 点击会拉取该 feed 的文章，并触发单 feed 刷新
- 分类编辑、删除、feed 编辑、OPML 导入都由壳层弹窗触发

## 文章内容增强

- 正文阅读由 `front/app/features/articles/components/ArticleContentView.vue` 负责
- 组件会跟踪已读、滚动深度、收藏、上下篇跳转
- 内容源支持原始内容和 Firecrawl 全文切换
- 如果 feed 开启内容补全或 Firecrawl，面板会显示处理状态、错误和手动触发按钮

## AI 总结

- 总结列表由 `front/app/features/summaries/components/AISummariesListView.vue` 负责
- 支持分类或 feed 过滤、日期过滤、分页、删除总结
- 生成总结时会提交队列任务，并通过 WebSocket 更新进度
- 选中总结后，详情区切到 `front/app/features/summaries/components/AISummaryDetailView.vue`

## Digest

- Digest 页面由 `front/app/features/digest/components/DigestListView.vue` 负责
- 支持日报、周报切换，日期跳转，刷新当前版和立即执行
- 左栏看分类，中栏看总结列表，右栏看总结正文和相关文章弹窗阅读
- 设置抽屉由 `front/app/features/digest/components/DigestSettings.vue` 负责

## 状态与数据流

- `front/app/api/` 是唯一 HTTP 边界
- `front/app/stores/api.ts` 是后端数据的单一来源
- `front/app/stores/feeds.ts` 和 `front/app/stores/articles.ts` 只暴露派生视图
- `syncToLocalStores()` 已移除，前端不再维护本地复制数组
