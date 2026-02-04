# Backend-Go 项目架构

基于 **Go** + **Gin** + **GORM** + **SQLite** 的高性能 RSS 阅读器后端，与 Python 后端 100% API 兼容，共享数据库。

## 1. 项目概述

### 技术栈
- **框架**: Gin v1.10.0 (HTTP Web Framework)
- **ORM**: GORM v1.25.12
- **数据库**: SQLite (共享 `rss_reader.db`)
- **RSS 解析**: gofeed v1.3.0
- **定时任务**: cron v3
- **配置管理**: Viper v1.19.0

### 与 Python 后端关系
- ✅ **100% API 兼容** - 前端无需修改即可切换
- ✅ **共享 SQLite 数据库** - `rss_reader.db`
- ✅ **功能完全对等** - 所有功能均已实现
- ⚡ **更高性能** - Go 并发处理

---

## 2. 项目结构

```
backend-go/
├── cmd/                        # 入口命令
│   ├── create-behavior-tables/ # 行为追踪表迁移工具
│   │   └── main.go
│   ├── migrate/                # 数据库迁移工具
│   │   └── main.go
│   └── server/                 # HTTP 服务主入口
│       └── main.go
├── configs/                    # 配置文件
│   └── config.yaml
├── internal/                   # 内部包 (不对外暴露)
│   ├── config/                 # 配置管理
│   │   └── config.go
│   ├── handlers/               # HTTP 处理器 (API 路由)
│   │   ├── ai.go               # AI 相关接口
│   │   ├── article.go          # 文章接口
│   │   ├── category.go         # 分类接口
│   │   ├── feed.go             # 订阅接口
│   │   ├── opml.go             # OPML 导入导出
│   │   ├── reading_behavior.go # 阅读行为追踪接口
│   │   ├── scheduler.go        # 调度器管理
│   │   └── summary.go          # AI 摘要接口
│   ├── middleware/             # 中间件
│   │   └── cors.go             # CORS 跨域
│   ├── models/                 # 数据模型 (GORM)
│   │   ├── ai_models.go        # AI 模型 (摘要/调度器/设置)
│   │   ├── article.go          # 文章模型
│   │   ├── category.go         # 分类模型
│   │   ├── feed.go             # 订阅模型
│   │   ├── reading_behavior.go # 阅读行为模型
│   │   ├── user_preference.go  # 用户偏好模型
│   │   └── utils.go            # 工具函数
│   ├── schedulers/             # 定时任务
│   │   ├── auto_refresh.go     # 自动刷新订阅
│   │   ├── auto_summary.go     # 自动生成摘要
│   │   └── preference_update.go # 偏好数据更新
│   └── services/               # 业务逻辑服务
│       ├── ai_prompt_builder.go # AI 个性化提示词
│       ├── ai_service.go       # AI 服务 (OpenAI API)
│       ├── feed_service.go     # 订阅业务逻辑
│       ├── preference_service.go # 偏好分析服务
│       └── rss_parser.go       # RSS 解析器
├── pkg/                        # 公共包 (可复用)
│   └── database/
│       └── db.go               # 数据库连接
├── go.mod                      # Go 模块定义
├── go.sum                      # 依赖校验
├── README.md                   # 项目文档
├── test_api.go                 # API 测试工具
└── rss_reader.db               # SQLite 数据库 (与 Python 共享)
```

---

## 3. 入口点详解

### 3.1 HTTP 服务 (`cmd/server/main.go`)

**启动流程**:
1. 加载配置 (`configs/config.yaml`)
2. 初始化数据库连接
3. 设置 API 路由
4. 启动定时调度器
5. 启动 HTTP 服务 (默认 5000 端口)
6. 优雅关机处理

**关键代码**:
```go
func main() {
    // 1. 加载配置
    cfg := config.Load()
    
    // 2. 初始化数据库
    db, err := database.InitDB(cfg.Database.DSN)
    
    // 3. 设置路由
    router := gin.Default()
    handlers.SetupRoutes(router, db)
    
    // 4. 启动调度器
    schedulers.Start()
    
    // 5. 启动服务
    srv := &http.Server{Addr: ":" + cfg.Server.Port, Handler: router}
    go srv.ListenAndServe()
    
    // 6. 优雅关机
    gracefulShutdown(srv, db)
}
```

### 3.2 数据库迁移工具 (`cmd/migrate/main.go`)

**命令**:
```bash
go run cmd/migrate/main.go check    # 检查数据库连接
go run cmd/migrate/main.go migrate  # 执行迁移 (带确认)
go run cmd/migrate/main.go fresh     # 重建所有表 (危险)
```

---

## 4. 配置管理

### 4.1 配置文件 (`configs/config.yaml`)

```yaml
server:
  port: "5000"        # HTTP 端口
  mode: "debug"       # debug / release / test

database:
  driver: "sqlite"
  dsn: "rss_reader.db"  # 数据库文件

cors:
  origins:
    - "http://localhost:3001"  # 前端地址
    - "http://localhost:3000"
  methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
  allow_headers: ["Content-Type", "Authorization"]
```

### 4.2 配置结构 (`internal/config/config.go`)

```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    CORS     CORSConfig
}

type ServerConfig struct {
    Port string
    Mode string
}
```

**Viper 集成**: 支持环境变量覆盖，配置文件热加载

---

## 5. 数据库模型 (GORM)

### 5.1 模型关系图

```
Category (1) ←── (N) Feed (1) ←── (N) Article
     ↑                                    
     └──────────────────────────────────── AISummary
```

### 5.2 Category (`internal/models/category.go`)

```go
type Category struct {
    ID          uint      `gorm:"primaryKey"`
    Name        string    `gorm:"uniqueIndex;size:100;not null"`
    Slug        string    `gorm:"uniqueIndex;size:50"`
    Icon        string    `gorm:"size:50;default:folder"`
    Color       string    `gorm:"size:20;default:#6366f1"`
    Description string    `gorm:"type:text"`
    CreatedAt   time.Time
    Feeds       []Feed    `gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE"`
}
```

**字段说明**:
- `Name`: 分类名称 (唯一)
- `Slug`: URL 标识符
- `Icon`: Iconify 图标名
- `Color`: 十六进制颜色值
- 级联删除: 删除分类时删除其订阅

### 5.3 Feed (`internal/models/feed.go`)

```go
type Feed struct {
    ID               uint       `gorm:"primaryKey"`
    Title            string     `gorm:"size:200;not null"`
    Description      string     `gorm:"type:text"`
    URL              string     `gorm:"uniqueIndex;size:500;not null"`
    CategoryID       *uint      `gorm:"index"`
    Icon             string     `gorm:"size:50;default:rss"`
    Color            string     `gorm:"size:20;default:#8b5cf6"`
    LastUpdated      *time.Time
    CreatedAt        time.Time
    MaxArticles      int        `gorm:"default:100"`
    RefreshInterval  int        `gorm:"default:60"`  // 分钟
    RefreshStatus    string     `gorm:"size:20;default:idle"`
    RefreshError     string     `gorm:"type:text"`
    LastRefreshAt    *time.Time
    AISummaryEnabled bool       `gorm:"default:true"`
    Articles         []Article  `gorm:"foreignKey:FeedID;constraint:OnDelete:CASCADE"`
}
```

**刷新状态**: `idle` | `refreshing` | `success` | `error`

### 5.4 Article (`internal/models/article.go`)

```go
type Article struct {
    ID          uint       `gorm:"primaryKey"`
    FeedID      uint       `gorm:"index;not null"`
    Title       string     `gorm:"size:500;not null"`
    Description string     `gorm:"type:text"`
    Content     string     `gorm:"type:text"`
    Link        string     `gorm:"size:1000"`
    PubDate     *time.Time
    Author      string     `gorm:"size:200"`
    Read        bool       `gorm:"default:false"`
    Favorite    bool       `gorm:"default:false"`
    CreatedAt   time.Time
    Feed        Feed       `gorm:"foreignKey:FeedID"`
}
```

### 5.5 AI 模型 (`internal/models/ai_models.go`)

**AISummary** (AI 摘要)
```go
type AISummary struct {
    ID           uint      `gorm:"primaryKey"`
    CategoryID   *uint     `gorm:"index"`
    Title        string    `gorm:"size:200;not null"`
    Summary      string    `gorm:"type:text;not null"`
    KeyPoints    string    `gorm:"type:text"`    // JSON 数组
    Articles     string    `gorm:"type:text"`    // JSON 文章 ID 列表
    ArticleCount int       `gorm:"default:0"`
    TimeRange    int       `gorm:"default:180"`  // 分钟
    CreatedAt    time.Time
    UpdatedAt    time.Time
    Category     *Category `gorm:"foreignKey:CategoryID"`
}
```

**SchedulerTask** (调度器任务)
```go
type SchedulerTask struct {
    ID                    uint       `gorm:"primaryKey"`
    Name                  string     `gorm:"uniqueIndex;size:50;not null"`
    Description           string     `gorm:"size:200"`
    CheckInterval         int        `gorm:"default:60"`  // 秒
    LastExecutionTime     *time.Time
    NextExecutionTime     *time.Time
    Status                string     `gorm:"size:20;default:idle"`
    LastError             string     `gorm:"type:text"`
    LastErrorTime         *time.Time
    TotalExecutions       int        `gorm:"default:0"`
    SuccessfulExecutions  int        `gorm:"default:0"`
    FailedExecutions      int        `gorm:"default:0"`
    ConsecutiveFailures   int        `gorm:"default:0"`
    LastExecutionDuration *float64   // 秒
    LastExecutionResult   string     `gorm:"type:text"`
    CreatedAt             time.Time
    UpdatedAt             time.Time
}
```

**AISettings** (AI 配置)
```go
type AISettings struct {
    ID          uint      `gorm:"primaryKey"`
    Key         string    `gorm:"uniqueIndex;size:100;not null"`
    Value       string    `gorm:"type:text"`
    Description string    `gorm:"size:200"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 5.6 级联删除

| 父表 | 子表 | 行为 |
|------|------|------|
| Category | Feed | `OnDelete:CASCADE` |
| Feed | Article | `OnDelete:CASCADE` |

---

## 6. API 路由设计

### 6.1 路由总览

```go
func SetupRoutes(r *gin.Engine, db *gorm.DB) {
    // 基础路由
    r.GET("/", homeHandler)
    r.GET("/health", healthHandler)
    
    // API 路由组
    api := r.Group("/api")
    {
        // 分类
        api.GET("/categories", GetCategories)
        api.POST("/categories", CreateCategory)
        api.PUT("/categories/:id", UpdateCategory)
        api.DELETE("/categories/:id", DeleteCategory)
        
        // 订阅
        api.GET("/feeds", GetFeeds)
        api.POST("/feeds", CreateFeed)
        api.PUT("/feeds/:id", UpdateFeed)
        api.DELETE("/feeds/:id", DeleteFeed)
        api.POST("/feeds/:id/refresh", RefreshFeed)
        api.POST("/feeds/fetch", FetchFeed)
        api.POST("/feeds/refresh-all", RefreshAllFeeds)
        
        // 文章
        api.GET("/articles", GetArticles)
        api.GET("/articles/stats", GetArticlesStats)
        api.GET("/articles/:id", GetArticle)
        api.PUT("/articles/:id", UpdateArticle)
        api.PUT("/articles/bulk-update", BulkUpdateArticles)
        
        // AI
        api.POST("/ai/summarize", SummarizeArticle)
        api.POST("/ai/test", TestAIConnection)
        api.GET("/ai/settings", GetAISettings)
        api.POST("/ai/settings", SaveAISettings)
        
        // OPML
        api.POST("/import-opml", ImportOPML)
        api.GET("/export-opml", ExportOPML)
        
        // 调度器
        api.GET("/schedulers/status", GetSchedulersStatus)
        api.POST("/schedulers/:name/trigger", TriggerScheduler)
        api.POST("/schedulers/:name/reset", ResetSchedulerStats)
        api.PUT("/schedulers/:name/interval", UpdateSchedulerInterval)
        
        // AI 摘要
        api.GET("/summaries", GetSummaries)
        api.POST("/summaries/generate", GenerateSummary)
        api.POST("/summaries/auto-generate", AutoGenerateSummary)
        api.GET("/summaries/:id", GetSummary)
        api.DELETE("/summaries/:id", DeleteSummary)
        
        // 自动摘要配置
        api.GET("/auto-summary/status", GetAutoSummaryStatus)
        api.POST("/auto-summary/config", UpdateAutoSummaryConfig)
    }
}
```

### 6.2 详细端点

#### 分类 API (`/api/categories`)

| 方法 | 端点 | 处理器 | 描述 |
|------|------|--------|------|
| GET | `/api/categories` | `GetCategories` | 获取所有分类 |
| POST | `/api/categories` | `CreateCategory` | 创建分类 |
| PUT | `/api/categories/:id` | `UpdateCategory` | 更新分类 |
| DELETE | `/api/categories/:id` | `DeleteCategory` | 删除分类 |

#### 订阅 API (`/api/feeds`)

| 方法 | 端点 | 处理器 | 描述 |
|------|------|--------|------|
| GET | `/api/feeds` | `GetFeeds` | 获取订阅列表 |
| POST | `/api/feeds` | `CreateFeed` | 创建订阅 |
| PUT | `/api/feeds/:id` | `UpdateFeed` | 更新订阅 |
| DELETE | `/api/feeds/:id` | `DeleteFeed` | 删除订阅 |
| POST | `/api/feeds/:id/refresh` | `RefreshFeed` | 刷新单个订阅 (异步) |
| POST | `/api/feeds/fetch` | `FetchFeed` | 预览订阅元数据 |
| POST | `/api/feeds/refresh-all` | `RefreshAllFeeds` | 刷新所有订阅 |

#### 文章 API (`/api/articles`)

| 方法 | 端点 | 处理器 | 描述 |
|------|------|--------|------|
| GET | `/api/articles/stats` | `GetArticlesStats` | 获取统计信息 |
| GET | `/api/articles` | `GetArticles` | 获取文章列表 (支持过滤) |
| GET | `/api/articles/:id` | `GetArticle` | 获取单篇文章 |
| PUT | `/api/articles/:id` | `UpdateArticle` | 更新文章 (已读/收藏) |
| PUT | `/api/articles/bulk-update` | `BulkUpdateArticles` | 批量更新 |

**查询参数**:
- `feed_id`: 按订阅过滤
- `category_id`: 按分类过滤
- `read`: 按已读状态过滤 (true/false)
- `favorite`: 按收藏过滤 (true/false)
- `search`: 搜索标题
- `page`, `page_size`: 分页

#### AI API (`/api/ai`)

| 方法 | 端点 | 处理器 | 描述 |
|------|------|--------|------|
| POST | `/api/ai/summarize` | `SummarizeArticle` | 生成文章摘要 |
| POST | `/api/ai/test` | `TestAIConnection` | 测试 AI API 连接 |
| GET | `/api/ai/settings` | `GetAISettings` | 获取 AI 配置 |
| POST | `/api/ai/settings` | `SaveAISettings` | 保存 AI 配置 |

#### OPML API

| 方法 | 端点 | 处理器 | 描述 |
|------|------|--------|------|
| POST | `/api/import-opml` | `ImportOPML` | 导入 OPML 文件 |
| GET | `/api/export-opml` | `ExportOPML` | 导出 OPML 文件 |

#### 调度器 API (`/api/schedulers`)

| 方法 | 端点 | 处理器 | 描述 |
|------|------|--------|------|
| GET | `/api/schedulers/status` | `GetSchedulersStatus` | 获取所有调度器状态 |
| POST | `/api/schedulers/:name/trigger` | `TriggerScheduler` | 手动触发调度器 |
| POST | `/api/schedulers/:name/reset` | `ResetSchedulerStats` | 重置统计 |
| PUT | `/api/schedulers/:name/interval` | `UpdateSchedulerInterval` | 更新间隔 |

#### 摘要 API (`/api/summaries`)

| 方法 | 端点 | 处理器 | 描述 |
|------|------|--------|------|
| GET | `/api/summaries` | `GetSummaries` | 获取摘要列表 |
| POST | `/api/summaries/generate` | `GenerateSummary` | 生成摘要 |
| POST | `/api/summaries/auto-generate` | `AutoGenerateSummary` | 异步生成摘要 |
| GET | `/api/summaries/:id` | `GetSummary` | 获取单个摘要详情 |
| DELETE | `/api/summaries/:id` | `DeleteSummary` | 删除摘要 |

#### 阅读行为追踪 API (`/api/reading-behavior`)

| 方法 | 端点 | 处理器 | 描述 |
|------|------|--------|------|
| POST | `/api/reading-behavior/track` | `TrackReadingBehavior` | 记录单个行为事件 |
| POST | `/api/reading-behavior/track-batch` | `BatchTrackReadingBehavior` | 批量记录行为事件 |
| GET | `/api/reading-behavior/stats` | `GetReadingStats` | 获取阅读统计信息 |

#### 用户偏好 API (`/api/user-preferences`)

| 方法 | 端点 | 处理器 | 描述 |
|------|------|--------|------|
| GET | `/api/user-preferences` | `GetUserPreferences` | 获取用户偏好列表 |
| POST | `/api/user-preferences/update` | `TriggerPreferenceUpdate` | 手动触发偏好更新 |

---

## 7. 服务层 (Services)

### 7.1 RSS 解析器 (`internal/services/rss_parser.go`)

**功能**:
- 使用 `gofeed` 解析 RSS/Atom 源
- 提取: 标题、描述、内容、发布时间、作者、图片、标签
- 获取站点 favicon
- URL 验证

**核心函数**:
```go
func ParseFeed(url string) (*gofeed.Feed, error)
func ExtractImage(html string) string
func FetchFavicon(url string) string
```

### 7.2 订阅服务 (`internal/services/feed_service.go`)

**功能**:
- 刷新订阅源并添加新文章
- 预览订阅元数据
- 重复检测 (按链接)
- 文章数量限制 (清理旧文章)
- 异步错误处理

**核心函数**:
```go
func (s *FeedService) RefreshFeed(feedID uint) error
func (s *FeedService) FetchFeedPreview(url string) (*FeedPreview, error)
func (s *FeedService) RefreshAllFeeds() error
```

### 7.3 AI 服务 (`internal/services/ai_service.go`)

**功能**:
- OpenAI 兼容 API 集成
- 文章摘要生成
- API 连接测试
- 中英文提示词支持
- 结构化摘要解析

**核心函数**:
```go
func (s *AIService) SummarizeArticle(article *models.Article, settings *AISettings) (*SummaryResult, error)
func (s *AIService) TestConnection(settings *AISettings) error
func (s *AIService) GenerateCategorySummary(categoryID uint, timeRange int) (*AISummary, error)
```

**摘要格式**:
```json
{
  "one_sentence_summary": "一句话总结",
  "key_points": ["要点1", "要点2"],
  "takeaways": ["收获1", "收获2"],
  "tags": ["标签1", "标签2"]
}
```

### 7.4 偏好分析服务 (`internal/services/preference_service.go`)

**功能**:
- 分析阅读行为数据计算用户偏好
- 按订阅源和分类聚合偏好
- 时间衰减算法（30天半衰期）
- 支持手动和自动更新

**核心函数**:
```go
func (s *PreferenceService) UpdateAllPreferences() error
func (s *PreferenceService) GetUserFeedPreferences() ([]UserPreference, error)
func (s *PreferenceService) GetUserCategoryPreferences() ([]UserPreference, error)
func (s *PreferenceService) calculatePreferenceScore(...) float64
```

**偏好分数计算**:
- 滚动深度权重：40%
- 阅读时长权重：30%
- 互动频率权重：30%
- 时间衰减因子：exp(-距今天数/30)

### 7.5 AI 提示词构建器 (`internal/services/ai_prompt_builder.go`)

**功能**:
- 构建个性化 AI 摘要提示词
- 注入用户偏好背景（Top 订阅源、分类）
- 根据阅读习惯调整风格（详细/简洁）
- 优化摘要重点

**核心函数**:
```go
func (b *AISummaryPromptBuilder) BuildPersonalizedPrompt(
    categoryName string,
    articlesText string,
    articleCount int,
    language string,
) (string, error)
```

---

## 8. 定时调度器 (Schedulers)

### 8.1 自动刷新 (`internal/schedulers/auto_refresh.go`)

**功能**:
- 每 60 秒检查一次 (可配置)
- 检查每个订阅的 `refresh_interval`
- 异步刷新到期的订阅
- 状态追踪

**配置项**:
- `CheckInterval`: 60 秒 (检查周期)
- `Feed.RefreshInterval`: 订阅级别刷新间隔 (分钟)

### 8.2 自动摘要 (`internal/schedulers/auto_summary.go`)

**功能**:
- 每 3600 秒 (1小时) 运行一次
- 为所有分类生成 AI 摘要
- 需要 AI 配置 (base_url, api_key, model)
- 结果持久化到数据库
- Mutex 互斥锁防止并发执行

**配置项**:
- `CheckInterval`: 3600 秒
- `TimeRange`: 180 分钟 (处理最近 3 小时文章)

### 8.3 偏好更新 (`internal/schedulers/preference_update.go`)

**功能**:
- 每 1800 秒 (30分钟) 运行一次
- 聚合阅读行为数据生成偏好
- 调用 PreferenceService 分析计算
- 支持手动触发更新

**配置项**:
- `CheckInterval`: 1800 秒

### 8.4 调度器启动

```go
func Start() {
    // 启动自动刷新调度器
    autoRefresh := NewAutoRefreshScheduler(db)
    autoRefresh.Start()
    
    // 启动自动摘要调度器
    autoSummary := NewAutoSummaryScheduler(db)
    autoSummary.Start()
    
    // 启动偏好更新调度器
    preferenceUpdate := NewPreferenceUpdateScheduler(1800)
    preferenceUpdate.Start()
}

func Stop() {
    // 优雅停止所有调度器
    autoRefresh.Stop()
    autoSummary.Stop()
    preferenceUpdate.Stop()
}
```

---

## 9. 中间件

### 9.1 CORS 中间件 (`internal/middleware/cors.go`)

```go
func CORSMiddleware(config *config.CORSConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", strings.Join(config.Origins, ", "))
        c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(config.Methods, ", "))
        c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        
        c.Next()
    }
}
```

---

## 10. 数据库连接

### 10.1 初始化 (`pkg/database/db.go`)

```go
func InitDB(dsn string) (*gorm.DB, error) {
    db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, err
    }
    
    // 自动迁移
    db.AutoMigrate(
        &models.Category{},
        &models.Feed{},
        &models.Article{},
        &models.AISummary{},
        &models.SchedulerTask{},
        &models.AISettings{},
    )
    
    return db, nil
}
```

### 10.2 与 Python 后端兼容

- 共享同一个 `rss_reader.db` 文件
- 使用相同的数据库 schema
- GORM AutoMigrate 确保表结构一致

---

## 11. 开发指南

### 11.1 环境要求

- Go 1.23+
- SQLite3

### 11.2 启动命令

```bash
cd backend-go

# 下载依赖
go mod tidy

# 运行服务
go run cmd/server/main.go

# 或使用 air 热重载
air
```

### 11.3 数据库迁移

```bash
# 检查数据库
go run cmd/migrate/main.go check

# 执行迁移
go run cmd/migrate/main.go migrate

# 重建数据库 (危险)
go run cmd/migrate/main.go fresh
```

### 11.4 API 测试

```bash
# 运行测试工具
go run test_api.go
```

---

## 12. 生产部署

### 12.1 构建

```bash
# 编译
go build -o rss-server cmd/server/main.go

# 运行
./rss-server
```

### 12.2 Docker (可选)

```dockerfile
FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o rss-server cmd/server/main.go
EXPOSE 5000
CMD ["./rss-server"]
```

### 12.3 与前端集成

前端配置 (`front/app/utils/constants.ts`):
```typescript
export const API_BASE_URL = 'http://localhost:5000/api'
```

**启动顺序**:
```bash
# 1. 启动后端
cd backend-go && go run cmd/server/main.go

# 2. 启动前端
cd front && pnpm dev
```

---

## 13. 与 Python 后端对比

| 特性 | Python (Flask) | Go (Gin) |
|------|----------------|----------|
| 性能 | 中等 | 高 (并发) |
| 启动时间 | 较慢 | 快 |
| 内存占用 | 较高 | 低 |
| 开发效率 | 高 | 中等 |
| 类型安全 | 运行时 | 编译时 |
| 部署 | 需要解释器 | 单二进制文件 |
| 数据库 | 共享 SQLite | 共享 SQLite |
| API 兼容 | 基准 | 100% 兼容 |

---

## 14. 关键设计模式

### 14.1 依赖注入

```go
// 处理器依赖数据库连接
func NewCategoryHandler(db *gorm.DB) *CategoryHandler {
    return &CategoryHandler{db: db}
}
```

### 14.2 服务层模式

```go
// 业务逻辑封装在服务层
func (s *FeedService) RefreshFeed(feedID uint) error {
    // 获取订阅
    // 解析 RSS
    // 保存文章
    // 更新状态
}
```

### 14.3 异步处理

```go
// 异步刷新
func RefreshFeed(c *gin.Context) {
    go func() {
        feedService.RefreshFeed(feedID)
    }()
    c.JSON(202, gin.H{"message": "refreshing"})
}
```

---

**版本**: 1.0  
**更新日期**: 2026-02-03
