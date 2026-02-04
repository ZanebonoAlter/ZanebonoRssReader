# Front (Nuxt 4) 项目架构

## 1. 项目概述

基于 **Nuxt 4** + **Vue 3** + **TypeScript** + **Tailwind CSS v4** 的 RSS 阅读器前端，采用三栏式 FeedBro 风格布局。

### 技术栈
- **框架**: Nuxt 4.2.2 (Vue 3.5.26)
- **样式**: Tailwind CSS v4 + @tailwindcss/vite
- **UI 组件**: @nuxt/ui ^4.3.0
- **状态管理**: Pinia ^3.0.4 + @pinia/nuxt
- **工具库**: VueUse, Day.js, Iconify, Marked, motion-v
- **HTTP**: 原生 fetch (封装 ApiClient)

---

## 2. 项目结构

```
front/
├── app/
│   ├── app.vue                 # 根组件 - 初始化数据，渲染 FeedLayout
│   ├── components/             # Vue 组件
│   │   ├── FeedLayout.vue          # 主布局容器（三栏式）
│   │   ├── layout/
│   │   │   ├── AppHeader.vue       # 顶部工具栏
│   │   │   ├── AppSidebar.vue      # 左侧边栏（分类/订阅）
│   │   │   └── ArticleListPanel.vue # 中间文章列表面板
│   │   ├── article/
│   │   │   ├── ArticleCard.vue     # 文章列表项
│   │   │   └── ArticleContent.vue  # 文章阅读区（集成行为追踪）
│   │   ├── category/
│   │   │   └── CategoryCard.vue    # 分类展示
│   │   ├── feed/
│   │   │   ├── FeedIcon.vue        # 订阅图标
│   │   │   └── RefreshStatusIcon.vue # 刷新状态
│   │   ├── ai/
│   │   │   ├── AISummary.vue       # AI 摘要卡片
│   │   │   ├── AISummaryDetail.vue # 摘要详情
│   │   │   └── AISummariesList.vue # 摘要列表
│   │   └── dialog/
│   │       ├── AddCategoryDialog.vue   # 添加分类
│   │       ├── EditCategoryDialog.vue  # 编辑分类
│   │       ├── AddFeedDialog.vue       # 添加订阅
│   │       ├── EditFeedDialog.vue      # 编辑订阅
│   │       ├── ImportOpmlDialog.vue    # OPML 导入
│   │       └── GlobalSettingsDialog.vue # 全局设置（集成偏好面板）
│   ├── composables/            # 组合式函数
│   │   ├── api/                    # API 层
│   │   │   ├── client.ts           # 基础 HTTP 客户端
│   │   │   ├── index.ts            # 统一导出
│   │   │   ├── categories.ts       # 分类 API
│   │   │   ├── feeds.ts            # 订阅 API
│   │   │   ├── articles.ts         # 文章 API
│   │   │   ├── summaries.ts        # AI 摘要 API
│   │   │   ├── opml.ts             # OPML API
│   │   │   └── reading_behavior.ts # 阅读行为 API
│   │   ├── useAI.ts                # AI 摘要功能
│   │   ├── useAutoRefresh.ts       # 定时刷新
│   │   ├── useRefreshPolling.ts    # 状态轮询
│   │   ├── useReadingTracker.ts    # 阅读行为追踪
│   │   └── useRssParser.ts         # RSS 解析
│   ├── stores/                 # Pinia 状态管理
│   │   ├── api.ts                  # API Store（数据源）
│   │   ├── feeds.ts                # 订阅 Store
│   │   ├── articles.ts             # 文章 Store
│   │   └── preferences.ts          # 用户偏好 Store
│   ├── types/                  # TypeScript 类型
│   │   ├── index.ts                # 统一导出
│   │   ├── api.ts                  # API 响应类型
│   │   ├── article.ts              # 文章类型
│   │   ├── ai.ts                   # AI 类型
│   │   ├── category.ts             # 分类类型
│   │   ├── common.ts               # 通用类型
│   │   ├── feed.ts                 # 订阅类型
│   │   └── reading_behavior.ts     # 阅读行为类型
│   ├── utils/                  # 工具函数
│   │   ├── index.ts                # 统一导出
│   │   ├── constants.ts            # 常量定义
│   │   ├── text.ts                 # 文本处理
│   │   ├── date.ts                 # 日期处理
│   │   └── storage.ts              # 存储工具
│   ├── plugins/                # Nuxt 插件
│   │   └── dayjs.ts                # Day.js 配置
│   └── assets/
│       └── css/
│           └── main.css            # 主样式文件
├── nuxt.config.ts              # Nuxt 配置
├── package.json                # 依赖管理
├── tsconfig.json               # TS 配置
└── ARCHITECTURE.md             # 本文件
```

---

## 3. 配置详解

### 3.1 Nuxt 配置 (`nuxt.config.ts`)

```typescript
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  vite: {
    plugins: [tailwindcss()],     // Tailwind CSS v4 Vite 插件
  },
  css: ['~/assets/css/main.css'],  // 全局样式
  modules: ['motion-v/nuxt', '@pinia/nuxt'], // 动画 + 状态管理
})
```

**关键配置**:
- 使用 Tailwind CSS v4 (@tailwindcss/vite)
- motion-v: 动画库
- @pinia/nuxt: Pinia 集成

### 3.2 依赖列表 (`package.json`)

| 依赖 | 版本 | 用途 |
|------|------|------|
| nuxt | ^4.2.2 | 核心框架 |
| vue | ^3.5.26 | Vue 核心 |
| vue-router | ^4.6.4 | 路由 |
| tailwindcss | ^4.1.18 | CSS 框架 |
| @tailwindcss/vite | ^4.1.18 | Tailwind Vite 插件 |
| @nuxt/ui | ^4.3.0 | UI 组件库 |
| pinia | ^3.0.4 | 状态管理 |
| @pinia/nuxt | ^0.11.3 | Pinia Nuxt 集成 |
| @vueuse/core | ^14.1.0 | Vue 工具集 |
| @iconify/vue | ^5.0.0 | 图标组件 |
| dayjs | ^1.11.19 | 日期处理 |
| marked | ^17.0.1 | Markdown 渲染 |
| motion-v | ^1.7.6 | 动画库 |
| tw-animate-css | ^1.4.0 | Tailwind 动画 |

---

## 4. API 层架构

### 4.1 目录结构

```
app/composables/api/
├── client.ts       # 基础 HTTP 客户端类
├── index.ts        # API 模块统一导出
├── categories.ts   # 分类 API (CRUD)
├── feeds.ts        # 订阅源 API
├── articles.ts     # 文章 API
├── summaries.ts    # AI 摘要 API
└── opml.ts         # OPML 导入导出
```

### 4.2 基础客户端 (`client.ts`)

**ApiClient 类**
- 封装 fetch API，统一错误处理
- Base URL: `http://localhost:5000/api`
- 标准方法: `get()`, `post()`, `put()`, `delete()`, `upload()`, `download()`
- 响应格式: `{ success: boolean, data: T, message?: string, error?: string }`

```typescript
// 使用示例
const client = new ApiClient()
const response = await client.get<Category[]>('/categories')
```

### 4.3 API 模块

| 模块 | 文件 | 主要方法 | 后端端点 |
|------|------|----------|----------|
| Categories | `categories.ts` | getCategories, createCategory, updateCategory, deleteCategory | GET/POST/PUT/DELETE /api/categories |
| Feeds | `feeds.ts` | getFeeds, createFeed, updateFeed, deleteFeed, refreshFeed, previewFeed | /api/feeds |
| Articles | `articles.ts` | getArticles, markAsRead, markAsFavorite, bulkMarkAsRead, getStats | /api/articles |
| Summaries | `summaries.ts` | getSummaries, createSummary, deleteSummary, generateSummary | /api/summaries |
| OPML | `opml.ts` | importOPML, exportOPML | POST /api/import-opml, GET /api/export-opml |
| ReadingBehavior | `reading_behavior.ts` | trackBehavior, trackBehaviorBatch, getReadingStats, getUserPreferences | /api/reading-behavior, /api/user-preferences |

**使用模式**:
```typescript
// 获取 API 方法
const { getCategories, createCategory } = useCategoriesApi()

// 调用
const response = await getCategories()
if (response.success) {
  // 处理 response.data
}
```

---

## 5. 界面布局架构

### 5.1 整体布局 (FeedLayout.vue)

**三栏式设计 (FeedBro 风格)**

```
┌─────────────────────────────────────────────────────────────┐
│  AppHeader (顶部工具栏: 刷新 / 添加订阅 / 导入 / 设置)       │
├────────────────┬──────────────────┬─────────────────────────┤
│                │                  │                         │
│  AppSidebar    │ ArticleListPanel │    ArticleContent     │
│  (左侧边栏)    │   (中间文章列表)  │    (右侧阅读区)        │
│                │                  │                         │
│  • 分类树      │  • 搜索栏        │    • 文章标题          │
│  • 订阅列表    │  • 筛选排序      │    • 元信息            │
│  • 操作菜单    │  • 文章卡片列表  │    • 正文内容          │
│                │                  │    • AI 摘要           │
│                │                  │                         │
└────────────────┴──────────────────┴─────────────────────────┘
```

**布局特点**:
- 左侧边栏可调整宽度 (resizable)
- 移动端响应式适配
- 支持键盘快捷键导航

### 5.2 组件层级

```
app.vue (根组件 - 初始化数据)
└── FeedLayout.vue (主布局容器)
    ├── layout/
    │   ├── AppHeader.vue        # 顶部工具栏
    │   │   └── 操作按钮组 (刷新/添加/导入/设置)
    │   ├── AppSidebar.vue       # 左侧边栏
    │   │   ├── 分类分组渲染
    │   │   ├── 订阅列表
    │   │   └── 操作菜单 (编辑/删除)
    │   └── ArticleListPanel.vue # 中间面板
    │       ├── 搜索栏
    │       ├── 筛选排序
    │       └── ArticleCard[] (文章列表)
    ├── article/
    │   └── ArticleContent.vue   # 右侧阅读区
    └── ai/
        ├── AISummary.vue        # AI 摘要卡片
        ├── AISummaryDetail.vue  # 摘要详情
        └── AISummariesList.vue  # 摘要列表
```

### 5.3 对话框组件

```
components/dialog/
├── AddCategoryDialog.vue    # 添加分类
├── EditCategoryDialog.vue   # 编辑分类
├── AddFeedDialog.vue        # 添加订阅 (带预览)
├── EditFeedDialog.vue       # 编辑订阅
├── ImportOpmlDialog.vue     # OPML 导入
└── GlobalSettingsDialog.vue # 全局设置 (AI 配置)
```

---

## 6. 状态管理架构

### 6.1 多 Store 模式

**数据流向**: 后端 → API Store → Local Stores → UI

| Store | 文件 | 职责 | 数据来源 |
|-------|------|------|----------|
| **useApiStore** | `stores/api.ts` | 与后端通信，数据源 | 后端 API |
| **useFeedsStore** | `stores/feeds.ts` | 本地订阅/分类状态 | apiStore 同步 |
| **useArticlesStore** | `stores/articles.ts` | 本地文章列表 + 筛选 | apiStore 同步 |
| **usePreferencesStore** | `stores/preferences.ts` | 用户偏好与阅读统计 | 后端行为追踪 API |

### 6.2 useApiStore (数据源)

**State**:
```typescript
{
  loading: boolean,      // 全局加载状态
  error: string | null, // 错误信息
  categories: Category[],
  feeds: Feed[],
  articles: Article[],
  allFeeds: Feed[]      // 侧边栏缓存
}
```

**核心方法**:
- `initialize()` - 应用启动时加载所有数据
- `fetchCategories()`, `fetchFeeds()`, `fetchArticles()` - 数据获取
- `create/update/delete Category/Feed` - CRUD 操作
- `syncToLocalStores()` - 同步数据到本地 stores

### 6.3 useFeedsStore (本地订阅状态)

**State**:
```typescript
{
  feeds: RssFeed[],
  categories: Category[] // 含 6 个默认分类
}
```

**计算属性**:
- `feedCount` - 订阅总数
- `categorizedFeeds` - 按分类分组
- `unreadCountsByFeed` - 各订阅未读数

### 6.4 useArticlesStore (本地文章状态)

**State**:
```typescript
{
  articles: Article[],
  filters: FilterState,    // 当前筛选条件
  currentArticle: Article | null
}
```

**计算属性**:
- `filteredArticles` - 应用筛选后的文章
- `unreadCount`, `favoriteCount` - 统计
- `articlesByFeed`, `unreadCountByFeed` - 分组统计

### 6.5 完整数据流

```
后端 API
    ↓ (fetch)
useApiStore (categories, feeds, articles)
    ↓ (syncToLocalStores)
useFeedsStore / useArticlesStore
    ↓ (reactive)
UI 组件 (FeedLayout → 子组件)
    ↓ (用户操作)
apiStore.createXxx() / updateXxx() / deleteXxx()
    ↓ (成功后)
重新 fetch → sync → UI 更新
```

### 6.6 阅读行为追踪数据流

```
用户阅读文章 (ArticleContent.vue)
    ↓
useReadingTracker 自动追踪
    ↓ (每 30 秒或关闭时)
批量上传到 /api/reading-behavior/track-batch
    ↓
后端存储到 reading_behaviors 表
    ↓ (每 30 分钟)
定时任务聚合到 user_preferences 表
    ↓
usePreferencesStore 获取偏好数据
    ↓
GlobalSettingsDialog 可视化展示
```

**追踪内容**:
- 文章打开/关闭事件
- 滚动深度 (0-100%)
- 阅读时长 (秒)
- 收藏/取消收藏操作

**偏好计算**:
- 滚动深度权重：40%
- 阅读时长权重：30%
- 互动频率权重：30%
- 时间衰减：30 天半衰期

---

## 7. 工具函数

### 7.1 目录结构

```
app/utils/
├── index.ts        # 统一导出
├── constants.ts    # 常量定义
├── text.ts         # 文本处理
├── date.ts         # 日期处理
└── storage.ts      # 存储工具
```

### 7.2 常量定义 (`constants.ts`)

| 常量 | 值 | 说明 |
|------|-----|------|
| `API_BASE_URL` | `'http://localhost:5000/api'` | API 地址 |
| `DEFAULT_PAGE_SIZE` | `10` | 默认分页大小 |
| `MAX_PAGE_SIZE` | `10000` | 最大分页 |
| `REFRESH_POLLING_INTERVAL` | `2000` | 刷新轮询间隔 (ms) |
| `MAX_POLLING_TIME` | `60000` | 最大轮询时间 (ms) |
| `AUTO_REFRESH_MINUTES` | `60` | 自动刷新间隔 (分钟) |
| `SIDEBAR_DEFAULT_WIDTH` | `256` | 侧边栏默认宽度 (px) |
| `SIDEBAR_MIN_WIDTH` | `200` | 最小宽度 (px) |
| `SIDEBAR_MAX_WIDTH` | `500` | 最大宽度 (px) |
| `AI_GENERATION_TIMEOUT` | `120000` | AI 生成超时 (ms) |
| `COLOR_OPTIONS` | 颜色数组 | 分类/订阅颜色选项 |
| `ICON_OPTIONS` | 图标数组 | 分类图标选项 |
| `TIME_RANGE_OPTIONS` | 时间数组 | AI 摘要时间范围 |

### 7.3 文本工具 (`text.ts`)

- `truncateText(text, maxLength)` - 截断文本
- `cleanHtml(html, maxLength)` - 清理 HTML 标签
- `extractFirstImage(html)` - 提取第一张图片
- `highlightKeyword(text, keyword)` - 高亮关键词
- `generateRandomColor()` - 随机颜色
- `getCategoryColor(categoryId)` - 获取分类颜色

### 7.4 日期工具 (`date.ts`)

- `formatRelativeTime(dateString)` - 相对时间 (刚刚/5分钟前/1小时前)
- `formatDate(dateString, format)` - 格式化日期
- `isToday(dateString)` - 判断是否为今天

### 7.5 插件 (`plugins/dayjs.ts`)

Day.js 配置:
- 插件: utc, timezone, relativeTime
- 语言: zh-cn
- 全局注入: `$dayjs`

---

## 8. 类型定义

### 8.1 目录结构

```
app/types/
├── index.ts        # 统一导出
├── api.ts          # API 响应类型
├── article.ts      # 文章类型
├── ai.ts           # AI 类型
├── category.ts     # 分类类型
├── common.ts       # 通用类型
├── feed.ts         # 订阅类型
└── reading_behavior.ts # 阅读行为类型
```

### 8.2 核心类型

**Category** (`types/category.ts`)
```typescript
interface Category {
  id: string           // 前端: string, 后端: number
  name: string
  slug: string
  icon: string         // Iconify 图标名
  color: string        // 十六进制颜色
  description: string
  feedCount: number
}
```

**RssFeed** (`types/feed.ts`)
```typescript
interface RssFeed {
  id: string
  title: string
  url: string
  category: string     // 分类 ID
  icon?: string
  color?: string
  lastUpdated: string
  articleCount: number
  unreadCount?: number
  refreshStatus?: 'idle' | 'refreshing' | 'success' | 'error'
  aiSummaryEnabled?: boolean
}
```

**Article** (`types/article.ts`)
```typescript
interface Article {
  id: string
  feedId: string
  title: string
  description: string
  content: string
  link: string
  pubDate: string
  author?: string
  read: boolean
  favorite: boolean
  imageUrl?: string
}
```

**AISummary** (`types/ai.ts`)
```typescript
interface AISummary {
  id: number
  category_id: number | null
  title: string
  summary: string
  key_points: string
  article_count: number
  time_range: number     // 分钟
  created_at?: string
  updated_at?: string
}
```

**ReadingBehaviorEvent** (`types/reading_behavior.ts`)
```typescript
type ReadingEventType = 'open' | 'close' | 'scroll' | 'favorite' | 'unfavorite'

interface ReadingBehaviorEvent {
  article_id: number
  feed_id: number
  category_id?: number
  session_id: string
  event_type: ReadingEventType
  scroll_depth?: number      // 0-100
  reading_time?: number      // 秒
}
```

**ReadingStats** (`types/reading_behavior.ts`)
```typescript
interface ReadingStats {
  total_articles: number
  total_reading_time: number
  avg_reading_time: number
  avg_scroll_depth: number
  most_active_feed_id: number
  most_active_category: number
}
```

**UserPreference** (`types/reading_behavior.ts`)
```typescript
interface UserPreference {
  id: number
  feed_id?: number
  category_id?: number
  preference_score: number  // 0-1
  avg_reading_time: number
  interaction_count: number
  scroll_depth_avg: number
  last_interaction_at?: string
  created_at: string
  updated_at: string
  feed_title?: string
  category_name?: string
}
```

### 8.3 ID 类型转换

**设计**: 后端使用 `number` 类型 ID，前端 Store 使用 `string` 类型

**转换点**:
- API 调用时: `string → number`
- 数据进入 Store 前: `number → string`

**原因**:
- 避免 JavaScript 大数精度问题
- 与 URL 参数、localStorage 兼容更好

---

## 9. 关键设计模式

### 9.1 API Layer Pattern

```
api/client.ts (基础客户端)
    ↓ 继承
api/categories.ts (领域 API)
    ↓ 使用
stores/api.ts (全局状态)
    ↓ 同步
stores/feeds.ts / stores/articles.ts (本地状态)
    ↓ 消费
UI 组件
```

### 9.2 响应式数据流

1. **初始化**: `app.vue` → `apiStore.initialize()` → `syncToLocalStores()`
2. **读取**: 组件读取 `feedsStore` / `articlesStore` (响应式)
3. **变更**: 用户操作 → `apiStore.xxx()` → 后端 API
4. **更新**: 成功 → 重新 fetch → 同步到本地 stores → UI 自动更新

### 9.3 文件组织约定

**Components**:
- 按功能域分组: `layout/`, `article/`, `ai/`, `dialog/`
- 大驼峰命名: `FeedLayout.vue`, `ArticleCard.vue`

**Composables**:
- 以 `use` 开头: `useAI.ts`, `useAutoRefresh.ts`
- API 层单独目录: `api/`

**Stores**:
- 以 `use` 开头: `useApiStore`, `useFeedsStore`

**Types**:
- 与模型名一致: `category.ts`, `feed.ts`

---

## 10. 开发指南

### 10.1 命令

```bash
cd front

# 安装依赖
pnpm install

# 启动开发服务器 (port 3001)
pnpm dev

# 生产构建
pnpm build

# 预览生产构建
pnpm preview
```

### 10.2 环境要求

- Node.js 18+
- pnpm 10.15.0+
- 后端服务运行于 `localhost:5000`

### 10.3 启动顺序

```bash
# 方式 1: 分别启动
cd backend && python app.py    # 后端: port 5000
cd front && pnpm dev           # 前端: port 3001

# 方式 2: 同时启动 (Windows)
start-all.bat
```

---

## 11. 与后端集成

### 11.1 API 地址

**Base URL**: `http://localhost:5000/api`

### 11.2 核心端点

| 功能 | 端点 | 方法 |
|------|------|------|
| 分类 | /api/categories | GET/POST/PUT/DELETE |
| 订阅 | /api/feeds | GET/POST/PUT/DELETE |
| 刷新订阅 | /api/feeds/:id/refresh | POST |
| 预览订阅 | /api/feeds/preview | POST |
| 文章 | /api/articles | GET/PUT |
| 批量标记已读 | /api/articles/bulk-read | POST |
| AI 摘要 | /api/summaries | GET/POST/DELETE |
| 生成摘要 | /api/summaries/generate | POST |
| OPML 导入 | /api/import-opml | POST |
| OPML 导出 | /api/export-opml | GET |

### 11.3 数据模型关系

```
Category (1) ←→ (N) Feed (1) ←→ (N) Article
     ↑                                      
     └────────────────────────────────────── AISummary
```

---

## 12. 扩展开发

### 12.1 添加新的 API 模块

1. 创建 `app/composables/api/newFeature.ts`
2. 导出 `useNewFeatureApi()` 函数
3. 在 `app/composables/api/index.ts` 添加导出
4. 在 `app/types/` 添加相关类型

### 12.2 添加新页面

1. 在 `app/pages/` 创建 `.vue` 文件
2. Nuxt 自动根据文件路径生成路由
3. 如需参数路由: `app/pages/article/[id].vue`

### 12.3 添加新 Store

1. 创建 `app/stores/newStore.ts`
2. 使用 `defineStore()` 定义
3. 在组件中使用 `useNewStore()`

---

**版本**: 1.0  
**更新日期**: 2026-02-03
