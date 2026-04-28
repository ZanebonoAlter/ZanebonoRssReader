# 编码约定

**分析日期:** 2026-04-10

## 命名模式

### 前端文件命名

**组件：**
- Vue 组件使用 PascalCase：`FeedIcon.vue`、`ArticleContentView.vue`、`AISummary.vue`
- 对应目录结构：`front/app/components/` 和 `front/app/features/*/components/`

**Composables：**
- 使用 camelCase 带 `use` 前缀：`useAI.ts`、`useApiStore.ts`
- 存放在：`front/app/composables/` 或 feature 内的 `composables/` 目录

**Stores：**
- 使用 camelCase 带 `use` 前缀：`useApiStore.ts`
- 存放在：`front/app/stores/`

**API 文件：**
- 使用 camelCase：`feeds.ts`、`articles.ts`、`client.ts`
- 存放在：`front/app/api/`

**类型文件：**
- 使用 camelCase 或描述性名称：`article.ts`、`types/index.ts`
- 存放在：`front/app/types/`

### 后端命名

**导出符号：**
- 结构体、函数、方法使用 PascalCase：`FeedService`、`RefreshFeed`、`GetArticles`
- 位置：`backend-go/internal/domain/*/handler.go`、`backend-go/internal/domain/*/service.go`

**私有符号：**
- 使用 lowerCamelCase：`loadArticleWithTagCount`、`cleanupOldArticles`、`shouldDelayArticleTagging`

**包名：**
- 简短、描述性名称：`feeds`、`articles`、`digest`、`models`

**JSON 字段标签：**
- 统一使用 snake_case：
```go
type Article struct {
    ID        uint      `json:"id"`
    FeedID    uint      `json:"feed_id"`     // snake_case
    CreatedAt time.Time `json:"created_at"`  // snake_case
    SummaryStatus string `json:"summary_status"` // snake_case
}
```

### 前端类型命名

**Props 接口：**
- 常用命名 `Props`：
```typescript
interface Props {
  article: Article | null
  articles?: Article[]
  onClose?: () => void
}
```

**Emits 类型：**
- 使用 `defineEmits<...>()` 类型化：
```typescript
const emit = defineEmits<{
  favorite: [id: string]
  navigate: [article: Article]
  articleUpdate: [id: string, updates: Partial<Article>]
}>()
```

## 代码风格

### 前端格式化

**TypeScript/Vue：**
- 大部分前端文件省略分号
- 保持 UTF-8 编码，严禁 ANSI、GBK、UTF-16
- 不配置专用 lint 工具（`front/package.json` 中无 lint script）
- 主要质量门：`pnpm exec nuxi typecheck` 和 `pnpm build`

### 后端格式化

**Go：**
- 使用 `gofmt` 格式化
- 无额外格式化工具配置
- 保持标准 Go 格式规范

## 导入组织

### 前端导入顺序

**推荐顺序：**
1. Vue/Nuxt 核心
2. 第三方库
3. 内部模块（使用 `~/` 别名）
4. 类型导入（使用 `import type`）

**实际示例（来自 `ArticleContentView.vue`）：**
```typescript
// 1. 第三方库
import { Icon } from '@iconify/vue'
import { marked } from 'marked'

// 2. 类型导入
import type { Article, RssFeed } from '~/types'

// 3. 内部模块 - 子组件
import ArticleTagList from './ArticleTagList.vue'

// 4. 内部模块 - API
import { useArticlesApi } from '~/api/articles'

// 5. 内部模块 - composables
import { useReadingTracker, useScrollDepthTracker } from '~/features/preferences/composables/useReadingTracker'
import { useContentCompletion, type ContentCompletionStatus } from '~/features/articles/composables/useContentCompletion'

// 6. 内部模块 - 工具函数
import { shouldShowArticleDescription } from '~/utils/articleContentGuards'
import { getArticleContentSources, resolveArticleContentBySource, type ArticleContentSource } from '~/utils/articleContentSource'
```

**路径别名：**
- 使用 `~` 别名表示 app 根路径
- 示例：`import { useApiStore } from '~/stores/api'`

### 后端导入分组

**分组规则：**
1. Go 标准库（无空行分隔）
2. 空行
3. 第三方库
4. 空行
5. 本地包

**实际示例（来自 `service.go`）：**
```go
import (
    "context"
    "fmt"
    "time"

    "go.opentelemetry.io/otel"
    otelCodes "go.opentelemetry.io/otel/codes"
    "gorm.io/gorm"

    "my-robot-backend/internal/domain/contentprocessing"
    "my-robot-backend/internal/domain/models"
    "my-robot-backend/internal/domain/topicextraction"
    "my-robot-backend/internal/platform/database"
)
```

**别名使用：**
- 仅在冲突或可读性需要时使用别名
- 示例：`otelCodes "go.opentelemetry.io/otel/codes"`

## 错误处理

### 前端错误处理

**API 响应模式：**
- 所有 HTTP 调用通过 `ApiClient` 返回统一格式
- 返回 `{ success, data, error, message }` 结构
- 不在组件中直接抛出异常

**ApiClient 模式（来自 `client.ts`）：**
```typescript
class ApiClient {
  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
    try {
      const response = await fetch(url, { ...options })
      const data = await response.json()

      if (!response.ok) {
        return {
          success: false,
          error: data.error || data.message || '请求失败',
          message: data.message,
        }
      }

      return {
        success: true,
        data: data.data,
        pagination: data.pagination,
        message: data.message,
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : '网络错误',
      }
    }
  }
}
```

**组件错误处理：**
- 使用 try-catch 包裹异步操作
- 捕获后显示友好消息，保留 `console.error` 用于调试
- 使用防御性 null 检查

**实际示例（来自 `ArticleContentView.vue`）：**
```typescript
async function handleManualFirecrawl() {
  if (!props.article || manualFirecrawlLoading.value) return

  manualFirecrawlLoading.value = true
  manualActionError.value = null

  try {
    const response = await crawlArticle(Number(props.article.id))
    if (!response.success) {
      throw new Error(response.error || '手动抓取失败')
    }
    // 成功处理...
    manualActionError.value = null
  } catch (error) {
    const message = error instanceof Error ? error.message : '手动抓取失败'
    manualActionError.value = message
    syncCurrentArticle({
      firecrawlStatus: 'failed',
      firecrawlError: message,
    })
  } finally {
    manualFirecrawlLoading.value = false
  }
}
```

### 后端错误处理

**Handler 模式：**
- 验证失败和数据库错误使用早返回
- 返回 `gin.H{"success": bool, "data"|"error"|"message": ...}` 结构

**实际示例（来自 `handler.go`）：**
```go
func GetArticle(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("article_id"), 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error":   "Invalid article ID",
        })
        return
    }

    article, err := loadArticleWithTagCount(uint(id))
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, gin.H{
                "success": false,
                "error":   "Article not found",
            })
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{
                "success": false,
                "error":   err.Error(),
            })
        }
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    article.ToDict(),
    })
}
```

**错误包装：**
- 使用 `fmt.Errorf(... %w ...)` 包装底层错误
- 不使用 panic 处理错误

**实际示例：**
```go
if err := database.DB.First(&feed, feedID).Error; err != nil {
    if err == gorm.ErrRecordNotFound {
        return fmt.Errorf("feed not found")
    }
    return err
}
```

## 日志记录

### 前端日志

**调试日志：**
- 使用 `console.error` 记录调试上下文
- 保留在组件中用于开发调试

**实际示例：**
```typescript
} catch {
  liveStatus.value = null
}
```

### 后端日志

**追踪系统：**
- 使用 OpenTelemetry 进行分布式追踪
- 在关键操作中添加 span

**实际示例（来自 `service.go`）：**
```go
func (s *FeedService) RefreshFeed(ctx context.Context, feedID uint) (err error) {
    ctx, span := otel.Tracer("rss-reader-backend").Start(ctx, "FeedService.RefreshFeed")
    defer span.End()
    defer func() {
        if err != nil {
            span.SetStatus(otelCodes.Error, "error")
            span.RecordError(err)
        }
    }()
    // ...
}
```

## 注释规范

**何时注释：**
- 复杂业务逻辑需要解释
- 公共 API 和工具函数需要用途说明
- 非直观的状态流转逻辑

**JSDoc/TSDoc：**
- 部分公共函数使用类型注释而非 JSDoc
- 类型定义本身即文档

**实际示例：**
```typescript
// Extract domain and build favicon URL
function getFaviconFromUrl(url: string): string | null {
  try {
    const urlObj = new URL(url)
    // ...
  } catch {
    return null
  }
}
```

## 函数设计

### 前端函数

**大小指南：**
- Composables 和 store 函数通常 10-30 行
- 组件内事件处理函数保持简洁

**参数模式：**
- Props 使用 TypeScript interface 定义
- 函数参数使用可选参数和默认值

**返回值：**
- API 函数返回 `ApiResponse<T>` 结构
- Composables 返回 reactive 引用和方法对象

**实际示例（来自 `useAI.ts`）：**
```typescript
export const useAI = () => {
  const settingsState = useState<AISettingsState>('ai-settings', createDefaultSettings)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function loadSettings(force = false) {
    // ...
  }

  const summarizeArticle = async (title: string, content: string, language: string = 'zh'): Promise<ApiResponse<AISummaryData>> => {
    // ...
  }

  return {
    loading,
    error,
    aiSettings,
    isAIEnabled,
    loadSettings,
    summarizeArticle,
    testConnection,
  }
}
```

### 后端函数

**大小指南：**
- Handler 函数保持 30-60 行
- Service 方法可更长但需结构清晰

**参数模式：**
- Handler 接收 `gin.Context`
- Service 方法接收具体参数和 `context.Context`

**返回值：**
- Handler 返回 JSON 响应
- Service 方法返回业务数据或 error

## 模块设计

### 前端模块

**导出模式：**
- API 文件导出函数工厂：`export function useFeedsApi()`
- Store 使用 Pinia defineStore：`export const useApiStore = defineStore('api', () => { ... })`
- Composables 导出函数：`export const useAI = () => { ... }`

**Barrel 文件：**
- `front/app/api/index.ts` 用于统一导出 API 模块
- 类型文件集中导出

### 后端模块

**包结构：**
- 每个领域域单独包：`feeds`、`articles`、`digest`、`summaries`
- Handler 和 Service 分离文件
- Models 统一放在 `internal/domain/models/`

**导出：**
- 包级导出关键结构和函数
- 内部实现保持私有

## Vue 特定约定

### Composition API

**必须使用：**
- 所有 Vue 文件使用 `<script setup lang="ts>`
- 严禁使用 Options API（除非项目明确要求）

**Props 定义：**
```typescript
const props = withDefaults(defineProps<Props>(), {
  article: null,
  articles: () => [],
  onClose: () => {},
  highlightedTagSlugs: () => [],
})
```

**响应式声明：**
- 使用 `ref()` 和 `computed()` 
- Store 使用 Pinia

**实际示例：**
```typescript
<script setup lang="ts">
const articlesStore = useArticlesStore()

const fallbackIcon = computed(() => {
  if (props.icon) return null
  // ...
})

function getFaviconFromUrl(url: string): string | null {
  // ...
}
</script>
```

## 数据映射

### 前端数据转换

**ID 转换：**
- 在 API/Store 边界将后端数值 ID 转为前端字符串
- 示例：`id: String(feed.id)` 或 `Number(id)` 反向转换

**snake_case → camelCase：**
- 仅在 Store/API 层进行映射
- 模板中不做映射，直接使用 camelCase 属性

**实际示例（来自 `api.ts`）：**
```typescript
const mappedFeeds = items.map((feed: any) => ({
  id: String(feed.id),
  title: feed.title,
  lastUpdated: feed.last_updated || new Date().toISOString(), // snake_case → camelCase
  articleCount: feed.article_count || 0,
  unreadCount: feed.unread_count || 0,
}))
```

---

*约定分析：2026-04-10*