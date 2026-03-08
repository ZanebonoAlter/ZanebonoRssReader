# 前端组件分工

## 路由入口

- `front/app/pages/index.vue` 只挂主阅读壳
- `front/app/pages/digest/index.vue` 和 `front/app/pages/digest/[id].vue` 只挂 Digest 视图

## Shell

- `front/app/features/shell/components/FeedLayoutShell.vue` 是主阅读器编排层
- `front/app/features/shell/components/AppHeaderView.vue` 只负责顶部操作按钮和刷新提示
- `front/app/features/shell/components/AppSidebarView.vue` 只负责导航、分类树、feed 列表和侧栏宽度交互
- `front/app/features/shell/components/ArticleListPanelView.vue` 只负责文章列表、筛选、分页和 feed 状态条

## Articles

- `front/app/features/articles/components/ArticleCardView.vue` 是文章列表卡片
- `front/app/features/articles/components/ArticleContentView.vue` 是正文阅读器，也是文章增强能力的主入口
- `front/app/features/articles/components/ContentCompletionView.vue` 负责单文章补全过程展示
- `front/app/features/articles/composables/useContentCompletion.ts` 负责内容补全接口和状态查询
- `front/app/features/articles/composables/useArticleProcessingStatus.ts` 负责 Firecrawl 和 AI 状态文案

## Summaries

- `front/app/features/summaries/components/AISummariesListView.vue` 负责 AI 总结列表和生成队列
- `front/app/features/summaries/components/AISummaryDetailView.vue` 负责总结正文和相关文章展开阅读
- `front/app/features/summaries/composables/useSummaryWebSocket.ts` 负责实时队列进度

## Digest

- `front/app/features/digest/components/DigestListView.vue` 是 Digest 页面编排层
- `front/app/features/digest/components/DigestDetail.vue` 负责单条 digest summary 正文和相关文章弹窗
- `front/app/features/digest/components/DigestSettings.vue` 负责 digest 配置抽屉

## Feeds 与 Preferences

- `front/app/features/feeds/composables/useAutoRefresh.ts` 负责全局自动刷新
- `front/app/features/feeds/composables/useRefreshPolling.ts` 负责刷新轮询和当前选择态
- `front/app/features/preferences/composables/useReadingTracker.ts` 负责阅读时长和滚动埋点

## 共享层

- `front/app/api/` 只做 HTTP 请求和接口边界转换
- `front/app/components/common/`、`front/app/components/feed/`、`front/app/components/dialog/`、`front/app/components/category/` 保留真正的通用组件
- `front/app/components/` 不再承载主业务实现，也不再保留旧入口兼容壳
