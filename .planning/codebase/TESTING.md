# 测试模式

**分析日期:** 2026-04-10

## 测试框架

### 前端测试框架

**运行器：**
- Vitest
- 配置文件：`front/vitest.config.ts`

**断言库：**
- Vitest 内置 `expect`

**运行命令：**
```bash
pnpm test:unit                          # 运行所有单元测试
pnpm test:unit -- app/utils/*.test.ts   # 指定文件
pnpm test:unit -- -t "prefers firecrawl" # 指定测试名称
```

### 后端测试框架

**运行器：**
- Go 标准 `testing` 包
- 部分测试使用 `testify`（`assert`、`require`）

**运行命令：**
```bash
go test ./...                           # 运行所有测试
go test ./internal/domain/feeds -v      # 指定包
go test ./internal/domain/feeds -run TestBuildArticleFromEntryTracksOnlyRunnableStates -v  # 指定测试
```

### 集成测试框架

**Python 集成测试：**
- pytest
- 位置：`tests/workflow/`、`tests/firecrawl/`

**运行命令：**
```bash
cd tests/workflow
uv venv
.venv\Scripts\activate
uv pip install -r requirements.txt
pytest test_*.py -v                      # 运行所有测试
pytest test_schedulers.py -v            # 指定文件
pytest test_schedulers.py::TestAutoRefreshScheduler::test_name -v  # 指定测试
pytest --cov=. --cov-report=html        # 覆盖率报告
```

**前提条件：**
- Go 后端运行在 `localhost:5000`

## 测试文件组织

### 前端测试位置

**单元测试：**
- 与源文件同级放置
- 命名模式：`*.test.ts`

**目录结构：**
```
front/app/
├── utils/
│   ├── articleContentSource.ts
│   └── articleContentSource.test.ts    # 同级测试
├── features/articles/components/
│   ├── ArticleTagList.vue
│   └── ArticleTagList.test.ts          # 同级测试
├── features/digest/components/
│   └── digestLayout.vue
│   └── digestLayout.test.ts            # 同级测试
```

### 后端测试位置

**单元测试：**
- 与源文件同级放置
- 命名模式：`*_test.go`

**目录结构：**
```
backend-go/internal/
├── domain/feeds/
│   ├── service.go
│   └── service_test.go                 # 同级测试
├── domain/articles/
│   ├── handler.go
│   └── handler_test.go                 # 同级测试
├── domain/digest/
│   ├── generator.go
│   ├── generator_test.go               # 同级测试
│   └── integration_test.go             # 集成测试
├── platform/database/
│   ├── db.go
│   └── db_test.go                      # 同级测试
```

### 集成测试位置

**Python 测试：**
```
tests/
├── workflow/
│   ├── test_schedulers.py              # 调度器测试
│   ├── test_workflow_integration.py   # 工作流集成测试
│   ├── test_error_handling.py         # 错误处理测试
│   ├── utils/
│   │   ├── api_client.py              # API 客户端工具
│   │   ├── database.py                # 数据库工具
│   │   └ mock_services.py             # Mock 服务
│   └── config.py                      # 测试配置
├── firecrawl/
│   ├── test_firecrawl_integration.py  # Firecrawl 流程测试
│   └── config.py
```

## 测试结构

### 前端测试套件

**组织模式：**
```typescript
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ArticleTagList from './ArticleTagList.vue'

describe('ArticleTagList', () => {
  it('renders aggregated article tags and highlights matched slugs', () => {
    const wrapper = mount(ArticleTagList, {
      props: {
        tags: [
          { slug: 'ai-agent', label: 'AI Agent', category: 'keyword', articleCount: 2 },
          { slug: 'sam-altman', label: 'Sam Altman', category: 'person', articleCount: 1 },
        ],
        highlightedSlugs: ['ai-agent'],
      },
    })

    expect(wrapper.text()).toContain('AI Agent')
    expect(wrapper.text()).toContain('Sam Altman')
    expect(wrapper.find('[data-tag-slug="ai-agent"]').classes()).toContain('article-tag--highlighted')
  })

  it('truncates long tag lists in compact mode', () => {
    // 测试代码...
  })
})
```

**工具函数测试：**
```typescript
import { describe, expect, it } from 'vitest'
import { getArticleContentSources, resolveArticleContentBySource } from './articleContentSource'

describe('articleContentSource', () => {
  it('prefers firecrawl when both firecrawl and original content exist', () => {
    const sources = getArticleContentSources({
      firecrawlContent: '# Firecrawl body',
      content: '<p>Original fallback</p>',
    })

    expect(sources.available).toEqual(['firecrawl', 'original'])
    expect(sources.defaultSource).toBe('firecrawl')
    expect(resolveArticleContentBySource(sources, 'firecrawl')).toBe('# Firecrawl body')
    expect(resolveArticleContentBySource(sources, 'original')).toBe('<p>Original fallback</p>')
  })
})
```

### 后端测试套件

**表格驱动测试（推荐）：**
```go
func TestBuildArticleFromEntryTracksOnlyRunnableStates(t *testing.T) {
    service := NewFeedService()
    entry := ParsedEntry{
        Title:       "Fresh News",
        Description: "desc",
        Content:     "content",
        Link:        "https://example.com/article",
        Author:      "bot",
    }

    tests := []struct {
        name               string
        feed               models.Feed
        wantFirecrawl      string
        wantSummary        string
    }{
        {
            name:          "full pipeline",
            feed:          models.Feed{ID: 1, FirecrawlEnabled: true, ArticleSummaryEnabled: true},
            wantFirecrawl: "pending",
            wantSummary:   "incomplete",
        },
        {
            name:          "manual only",
            feed:          models.Feed{ID: 2, FirecrawlEnabled: false, ArticleSummaryEnabled: true},
            wantFirecrawl: "",
            wantSummary:   "complete",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            article := service.buildArticleFromEntry(tt.feed, entry)
            if article.FirecrawlStatus != tt.wantFirecrawl {
                t.Fatalf("firecrawl status = %q, want %q", article.FirecrawlStatus, tt.wantFirecrawl)
            }
            if article.SummaryStatus != tt.wantSummary {
                t.Fatalf("summary status = %q, want %q", article.SummaryStatus, tt.wantSummary)
            }
        })
    }
}
```

**Testify 模式：**
```go
func TestGroupByCategory(t *testing.T) {
    config := &DigestConfig{}
    generator := NewDigestGenerator(config)

    summaries := []models.AISummary{
        {ID: 1, FeedID: uintPtr(1), Title: "Test Summary 1", Category: &models.Category{ID: 1, Name: "AI技术"}},
        {ID: 2, FeedID: uintPtr(2), Title: "Test Summary 2", Category: &models.Category{ID: 1, Name: "AI技术"}},
        {ID: 3, FeedID: uintPtr(3), Title: "Test Summary 3", Category: &models.Category{ID: 2, Name: "前端开发"}},
    }

    result := generator.groupByCategory(summaries)

    assert.Equal(t, 2, len(result))

    categoryMap := make(map[string]CategoryDigest)
    for _, cat := range result {
        categoryMap[cat.CategoryName] = cat
    }

    aiTech, exists := categoryMap["AI技术"]
    assert.True(t, exists)
    assert.Equal(t, 2, aiTech.FeedCount)
    assert.Equal(t, 2, len(aiTech.AISummaries))
}
```

### Python 集成测试

**pytest Fixture 模式：**
```python
class TestAutoRefreshScheduler:
    """自动刷新调度器测试"""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """测试初始化"""
        self.db = DatabaseHelper(TestConfig.DATABASE_PATH)
        self.api = APIClient(TestConfig.BACKEND_BASE_URL, TestConfig.BACKEND_TIMEOUT)
        
        # 创建测试数据
        self.test_feed_id = self.db.create_test_feed(
            title=DatabaseConfig.TEST_FEED['title'],
            url=DatabaseConfig.TEST_FEED['url'],
            refresh_interval=60
        )
        
        yield
        
        # 清理测试数据
        self.db.cleanup_test_data()
    
    def test_scheduler_exists(self):
        """测试调度器任务是否存在"""
        task = self.db.get_scheduler_task('auto_refresh')
        assert task is not None, "auto_refresh 调度器任务不存在"
        assert task['status'] in ['idle', 'running'], f"调度器状态异常: {task['status']}"
    
    def test_scheduler_can_be_triggered(self):
        """测试调度器可以手动触发"""
        result = self.api.trigger_scheduler('auto_refresh')
        assert result.get('success'), f"触发调度器失败: {result}"
```

## Mocking

### 前端 Mock

**Vue Test Utils：**
- 使用 `mount()` 并传入 props
- 不深度 Mock 子组件

**实际示例：**
```typescript
import { mount } from '@vue/test-utils'
import ArticleTagList from './ArticleTagList.vue'

const wrapper = mount(ArticleTagList, {
  props: {
    tags: [
      { slug: 'ai-agent', label: 'AI Agent', category: 'keyword', articleCount: 2 },
    ],
    highlightedSlugs: ['ai-agent'],
  },
})

expect(wrapper.text()).toContain('AI Agent')
```

**什么不 Mock：**
- Vitest 配置使用 `happy-dom` 环境
- 不 Mock 基础 DOM API

### 后端 Mock

**数据库 Mock：**
- 使用内存 SQLite：`sqlite.Open("file:%s?mode=memory&cache=shared")`
- 不 Mock GORM

**实际示例：**
```go
func setupFeedsTestDB(t *testing.T) {
    t.Helper()

    db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
    if err != nil {
        t.Fatalf("open sqlite: %v", err)
    }

    database.DB = db
    if err := database.DB.AutoMigrate(&models.Feed{}, &models.Article{}, ...); err != nil {
        t.Fatalf("migrate test db: %v", err)
    }
}
```

**HTTP Mock：**
- 使用 `httptest.NewServer` 模拟外部 RSS 服务

**实际示例：**
```go
func TestRefreshFeedEnqueuesTagJobWhenCompletionDisabled(t *testing.T) {
    setupFeedsTestDB(t)

    rssServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/rss+xml")
        _, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>OpenAI Feed</title>
    <item>
      <title>OpenAI launches new AI agent runtime</title>
      <link>https://example.com/openai-agent</link>
    </item>
  </channel>
</rss>`))
    }))
    defer rssServer.Close()

    feed := models.Feed{
        Title: "Seed Feed",
        URL:   rssServer.URL,
        MaxArticles: 10,
        FirecrawlEnabled: false,
    }
    // ...
}
```

## 测试数据和工厂

### 前端测试数据

**内联数据：**
- 测试中直接定义数据对象
- 不使用外部 fixtures 文件

```typescript
const wrapper = mount(ArticleTagList, {
  props: {
    tags: [
      { slug: 'ai-agent', label: 'AI Agent', category: 'keyword', articleCount: 2 },
    ],
  },
})
```

### 后端测试数据

**内联创建：**
- 在测试中直接创建 model 对象
- 使用 helper 函数简化创建

```go
func setupDigestGeneratorTestDB(t *testing.T) {
    t.Helper()

    db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:digest_generator_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
    require.NoError(t, err)
    
    category := models.Category{Name: "AI", Slug: "ai", Color: "#3b6b87", Icon: "mdi:brain"}
    require.NoError(t, database.DB.Create(&category).Error)
    
    feed := models.Feed{Title: "OpenAI Blog", URL: "https://example.com/openai", CategoryID: &category.ID}
    require.NoError(t, database.DB.Create(&feed).Error)
}
```

### Python 测试数据

**工厂方法：**
- 使用 `DatabaseHelper` 类创建测试数据
- 位置：`tests/workflow/utils/database.py`

```python
self.test_feed_id = self.db.create_test_feed(
    title=DatabaseConfig.TEST_FEED['title'],
    url=DatabaseConfig.TEST_FEED['url'],
    refresh_interval=60
)

self.test_article_id = self.db.create_test_article(
    feed_id=self.test_feed_id,
    title="Firecrawl测试文章",
    firecrawl_status='pending'
)
```

## 测试覆盖率

### 前端覆盖率

**要求：**
- 未强制覆盖率目标

**查看覆盖率：**
- Vitest 未配置覆盖率输出
- 主要依赖 `pnpm exec nuxi typecheck` 和 `pnpm build`

### 后端覆盖率

**要求：**
- 未强制覆盖率目标
- 测试分布在各 domain 包

**查看覆盖率：**
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Python 覆盖率

**要求：**
- 未强制覆盖率目标

**查看覆盖率：**
```bash
pytest --cov=. --cov-report=html
```

## 测试类型

### 前端单元测试

**范围：**
- 工具函数测试：`front/app/utils/*.test.ts`
- 组件渲染测试：`front/app/features/*/components/*.test.ts`
- 不涉及真实 HTTP 请求

**示例：**
- `articleContentSource.test.ts` - 内容源选择逻辑
- `ArticleTagList.test.ts` - 标签列表渲染和截断
- `digestLayout.test.ts` - Digest 布局测试

### 后端单元测试

**范围：**
- Service 方法测试：业务逻辑
- Handler 测试：参数验证和响应格式
- Generator 测试：数据转换逻辑

**示例：**
- `service_test.go` - Feed 刷新逻辑、文章创建状态流转
- `generator_test.go` - Digest 分类聚合逻辑
- `handler_test.go` - HTTP handler 参数验证

### 后端集成测试

**范围：**
- 多模块协作测试
- 数据库操作集成

**示例：**
- `digest/integration_test.go` - Digest 生成完整流程
- `cmd/migrate-db/main_test.go` - 数据库迁移工具测试

### Python 集成测试

**范围：**
- 调度器状态和行为
- Firecrawl 流程验证
- 端到端工作流

**示例：**
- `test_schedulers.py` - 调度器健康检查、触发、状态流转
- `test_firecrawl_integration.py` - Firecrawl 抓取流程
- `test_workflow_integration.py` - 完整工作流测试

## 常见测试模式

### 异步测试

**前端：**
```typescript
it('handles async content resolution', async () => {
  const sources = await getArticleContentSources({ ... })
  expect(sources.available).toEqual(['firecrawl', 'original'])
})
```

**后端：**
```go
func TestRefreshFeedEnqueuesTagJobWhenCompletionDisabled(t *testing.T) {
    // ...
    if err := service.RefreshFeed(context.Background(), feed.ID); err != nil {
        t.Fatalf("refresh feed: %v", err)
    }
}
```

### 错误测试

**前端：**
```typescript
it('handles error response', () => {
  const result = { success: false, error: '网络错误' }
  expect(result.success).toBe(false)
  expect(result.error).toBe('网络错误')
})
```

**后端：**
```go
func TestGroupByCategory_WithNilCategory(t *testing.T) {
    // 测试 nil category 处理
    result := generator.groupByCategory(summaries)
    
    uncategorized, exists := categoryMap["未分类"]
    assert.True(t, exists)
    assert.Equal(t, uint(0), uncategorized.CategoryID)
}
```

### 状态流转测试

**后端：**
```go
func TestCleanupOldArticlesKeepsActiveCompletionArticles(t *testing.T) {
    setupFeedsTestDB(t)
    
    // 创建不同状态的文章
    articles := []models.Article{
        {Title: "new complete", SummaryStatus: "complete", FirecrawlStatus: "completed"},
        {Title: "old incomplete", SummaryStatus: "incomplete", FirecrawlStatus: "completed"},
    }
    
    service.cleanupOldArticles(&feed)
    
    // 验证清理后保留正确的文章
    if !titles["old incomplete"] {
        t.Fatalf("expected incomplete article to be preserved")
    }
}
```

**Python：**
```python
def test_firecrawl_status_transition(self):
    """测试 Firecrawl 状态流转"""
    article = self.db.get_article(self.test_article_id)
    assert article['firecrawl_status'] == 'pending', "初始状态应该是 pending"
    
    # 模拟状态变更
    self.db.update('articles', {'firecrawl_status': 'processing'}, 'id = ?', (self.test_article_id,))
    
    article = self.db.get_article(self.test_article_id)
    assert article['firecrawl_status'] == 'processing', "状态应该变为 processing"
```

---

*测试分析：2026-04-10*