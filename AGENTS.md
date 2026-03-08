# AGENTS.md - Agent Coding Guidelines

This document provides guidance to agentic coding assistants working in this repository.

## Project Overview

RSS Reader application with dual backend options (Python Flask / Go Gin) and Nuxt 4 frontend. Features AI-powered summarization, automatic feed refreshing, and a three-column FeedBro-style layout.

**Note**: No user authentication - designed for personal/single-user deployment.

---

## 角色设定

你是一位资深独立设计师，专注于「反主流」的网页美学。  
你鄙视千篇一律的 SaaS 模板，追求每个像素都有温度。

## ❌ 绝对禁止项

### 配色禁止

- 紫色/靛蓝色/蓝紫渐变（#6366F1、#8B5CF6）
- 纯平背景色（必须有噪点纹理或渐变）
- Tailwind 默认色板

### 布局禁止

- Hero + 三卡片布局
- 完美居中对齐
- 等宽多栏（必须不对称）

### 文案禁止

- 高深的专业名词和无意义的空话
- Lorem Ipsum 占位文本
- 被动语态和长句

### 组件禁止

- Shadcn/Material UI 默认组件（必须深度定制）
- Emoji 作为功能图标
- 线性动画（ease-in-out）

## ✅ 必须遵守项

### 文案风格

- 口语化，像朋友聊天
- 具体化，有数字和场景
- 可以幽默、自嘲、甚至挑衅
- 每句话不超过 15 个字

### 图片系统

- 图标：使用 Iconify 图标库（<https://iconify.design>）
- 占位图：使用 Picsum Photos（<https://picsum.photos>）
- 真实图片：使用 Pexels 搜索（<https://www.pexels.com>）
- 插画：使用 unDraw（<https://undraw.co>）

## Development Commands

### Frontend (Nuxt 4 + TypeScript)

```bash
cd front

# Install dependencies
pnpm install

# Development server (port 3001)
pnpm dev

# Production build
pnpm build

# Preview production build
pnpm preview

# Type checking (use nuxi typecheck if available)
npx nuxi typecheck
```

**Frontend Environment**: Node.js 18+, pnpm 10.15.0+

### Backend - Go (Gin + GORM)

```bash
cd backend-go

# Download dependencies
go mod tidy

# Run server (port 5000)
go run cmd/server/main.go

# Build binary
go build -o rss-server cmd/server/main.go

# Run with air hot reload (if installed)
air

# Database migration commands
go run cmd/migrate/main.go check    # Check database connection
go run cmd/migrate/main.go migrate  # Run migrations (with confirmation)
go run cmd/migrate/main.go fresh     # Rebuild all tables (destructive)

# Database utility commands
go run cmd/list-tables/main.go      # List all tables in database
go run cmd/test-init/main.go        # Test database initialization

# Note: Database initialization is automatic on server startup
# The server will automatically create missing tables via EnsureTables()
```

**Go Environment**: Go 1.21+

### Quick Start (Both Frontend + Backend)

From project root on Windows:
```bash
start-all.bat
```

---

## Code Style Guidelines

### Frontend (Vue 3 / Nuxt 4 / TypeScript)

#### Component Style
- **Use Composition API with `<script setup>`** (default, not Options API)
- **TypeScript required** for all new code
- **File naming**: PascalCase for components (`FeedLayout.vue`, `ArticleCard.vue`)
- **Component organization**: Group by feature in `components/` (layout/, article/, ai/, dialog/)

```vue
<script setup lang="ts">
// Import type-only dependencies first
import type { Category, Article } from '~/types'
import { Icon } from '@iconify/vue'

// Props with TypeScript interface
interface Props {
  categoryId?: string
  readonly?: boolean
}
const props = withDefaults(defineProps<Props>(), {
  categoryId: '',
  readonly: false,
})

// Emits with TypeScript
const emit = defineEmits<{
  update: [value: string]
  delete: [id: string]
}>()

// Composables
const apiStore = useApiStore()
const { categories } = storeToRefs(apiStore)
</script>
```

#### File Encoding
- **All frontend source files must be UTF-8**. This includes `.vue`, `.ts`, `.js`, `.css`, `.json`, and config files.
- **Never save frontend files as GBK, ANSI, UTF-16, or system-default encodings**. Nuxt/Vite toolchains will crash on mixed encodings.
- **When editing files from PowerShell, always specify UTF-8 explicitly**. Use `Set-Content -Encoding utf8` or another explicit UTF-8 write path.
- **If a file is rewritten by script, preserve UTF-8 on write-back**. Do not rely on editor or shell defaults.
- **If the frontend build throws a UTF-8 parse or Vue preprocessor panic, check file encoding first before debugging component logic**.

#### Type Definitions
- **Centralized types** in `app/types/` - separate files by domain (category.ts, feed.ts, article.ts)
- **Use `interface` for object shapes**, `type` for unions/aliases
- **Backend IDs**: Backend uses `number`, frontend stores use `string` - convert at API boundary
- **API responses**: Use `ApiResponse<T>` wrapper

```typescript
// app/types/category.ts
export interface Category {
  id: string           // Frontend uses string
  name: string
  slug: string
  icon: string
  color: string
  description: string
  feedCount: number
}

// app/types/api.ts
export interface ApiResponse<T = any> {
  success: boolean
  data?: T
  pagination?: PaginationMeta
  message?: string
  error?: string
}
```

#### State Management (Pinia)
- **Use `defineStore()` with setup syntax**
- **Store naming**: `use` prefix + camelCase (`useApiStore`, `useFeedsStore`)
- **Data flow**: Backend → `apiStore` → local stores → components

```typescript
// stores/api.ts
export const useApiStore = defineStore('api', () => {
  const loading = ref(false)
  const error = ref<string | null>(null)
  const categories = ref<Category[]>([])

  async function fetchCategories() {
    loading.value = true
    const response = await getCategories()
    if (response.success) {
      categories.value = response.data.map(cat => ({
        ...cat,
        id: String(cat.id)  // Convert number to string
      }))
    }
    loading.value = false
    return response
  }

  return { loading, error, categories, fetchCategories }
})
```

#### API Layer
- **Use `ApiClient` class** for all HTTP requests (`composables/api/client.ts`)
- **API modules** in `composables/api/` (categories.ts, feeds.ts, articles.ts)
- **Handle errors** with `ApiResponse<T>` wrapper - check `response.success`

```typescript
// composables/api/categories.ts
export function useCategoriesApi() {
  return {
    async getCategories() {
      return apiClient.get<Category[]>('/categories')
    },
    async createCategory(data: CreateCategoryDto) {
      return apiClient.post<Category>('/categories', data)
    }
  }
}
```

#### Styling (Tailwind CSS v4)
- **Utility-first**: Use Tailwind classes, avoid custom CSS when possible
- **Component-specific CSS**: Import at end of `<script setup>` with `import './Component.css'`
- **Color palette**: Use semantic names (primary/amber/purple) with Tailwind arbitrary values

#### Imports
- **Group by type**: Vue → external → internal (composables, stores, types, utils)
- **Use `~` alias** for app root (`~/components/...`, `~/composables/...`)
- **Type-only imports**: `import type { X }` for types-only

```typescript
// Preferred import order
import { ref, computed } from 'vue'
import { Icon } from '@iconify/vue'
import { useApiStore } from '~/stores/api'
import type { Category } from '~/types'
```

### Backend - Go (Gin + GORM)

#### File Organization
- **Package structure**: `cmd/` (entry points), `internal/` (private), `pkg/` (public)
- **Handler naming**: PascalCase, verb-first (`GetCategories`, `CreateFeed`, `RefreshFeed`)
- **Model methods**: PascalCase for exported, camelCase for internal

```go
// internal/handlers/category.go
func GetCategories(c *gin.Context) {
    var categories []models.Category
    if err := database.DB.Order("name ASC").Find(&categories).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error":   err.Error(),
        })
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "data": categories})
}
```

#### Model Definitions (GORM)
- **Use struct tags** for GORM and JSON: `gorm:"..." json:"..."`
- **JSON naming**: snake_case (`feed_count`, `created_at`)
- **Relationships**: Define foreign keys with cascade delete

```go
type Category struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    Name        string    `gorm:"uniqueIndex;size:100;not null" json:"name"`
    Slug        string    `gorm:"uniqueIndex;size:50" json:"slug"`
    Icon        string    `gorm:"size:50;default:folder" json:"icon"`
    Color       string    `gorm:"size:20;default:#6366f1" json:"color"`
    Description string    `gorm:"type:text" json:"description"`
    CreatedAt   time.Time `json:"created_at"`
    Feeds       []Feed    `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE" json:"feeds,omitempty"`
}
```

#### Error Handling
- **HTTP status codes**: Use appropriate codes (400, 404, 409, 500)
- **Response format**: Always wrap in `{"success": bool, "data/error": ...}`
- **Log errors** at handler level before returning

```go
if err := database.DB.First(&category, uint(id)).Error; err != nil {
    if err == gorm.ErrRecordNotFound {
        c.JSON(http.StatusNotFound, gin.H{
            "success": false,
            "error":   "Category not found",
        })
    } else {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error":   err.Error(),
        })
    }
    return
}
```

#### Context and Dependencies
- **Use global database connection** via `database.DB`
- **Get URL params**: `c.Param("id")` - always validate/parse
- **Parse JSON**: `c.ShouldBindJSON(&req)` with validation tags

```go
type CreateCategoryRequest struct {
    Name        string `json:"name" binding:"required"`
    Icon        string `json:"icon"`
    Color       string `json:"color"`
    Description string `json:"description"`
}

func CreateCategory(c *gin.Context) {
    var req CreateCategoryRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
        return
    }
    // ... process request
}
```

#### Go Conventions
- **Exported names**: PascalCase (`GetCategories`, `Category`)
- **Private names**: camelCase (`feedCount`, `parseUrl`)
- **Package comments**: Add doc comments for exported functions
- **Error wrapping**: Use `fmt.Errorf` with `%w` for wrapping errors

---

## Testing

### Frontend Testing
- **No test framework currently configured** - add Vitest for unit tests when needed
- **Manual testing**: Use `pnpm dev` and test in browser at `http://localhost:3001`

### Backend Testing (Go)
- **API test tool**: `go run backend-go/test_api.go` tests all endpoints
- **Add unit tests** in `*_test.go` files using Go's `testing` package
- **Test naming**: `func TestFunctionName(t *testing.T)`

---

## Key Integration Points

1. **API Base URL**: Frontend → `http://localhost:5000/api` (defined in `app/utils/constants.ts`)
2. **ID Conversion**: Backend (number) → Frontend Store (string) - convert at API layer
3. **Data Flow**: Backend API → `useApiStore` → local stores → UI components
4. **CORS**: Backend must allow `http://localhost:3001` origins

---

## Common Patterns

### Adding New API Endpoint
1. **Backend (Go)**: Add handler in `internal/handlers/`, register route in `SetupRoutes()`
2. **Frontend**: Add API function in `composables/api/`, add types in `types/`
3. **Store**: Add fetch/CRUD methods to `useApiStore`

### Adding New Component
1. Create `.vue` file in appropriate `components/` subdirectory
2. Use `<script setup lang="ts">` with TypeScript
3. Import from `~/composables`, `~/stores`, `~/types`
4. Use Tailwind classes for styling

### Adding New Store
1. Create file in `app/stores/` with `defineStore()`
2. Export `useXxxStore()` function
3. Use in components with `const xxxStore = useXxxStore()`

---

## Important Notes

- **No authentication** - app assumes single-user personal deployment
- **Database**: SQLite (`rss_reader.db`)
- **Cascading deletes**: Category → Feed → Article
- **Default refresh interval**: 60 minutes for feeds
- **AI features**: Requires OpenAI-compatible API configuration (stored in database)
