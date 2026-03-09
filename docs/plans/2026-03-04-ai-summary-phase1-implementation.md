# AI总结增强功能 - Phase 1 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 构建日报/周报自动生成、飞书推送和Obsidian导出功能，提升AI总结的消费体验

**Architecture:** 基于现有AutoSummaryScheduler，新增DigestScheduler模块处理日报/周报生成，通过FeishuNotifier推送消息，通过ObsidianExporter写入Markdown文件到Obsidian vault

**Tech Stack:**
- Go 1.21+ (Gin, GORM, Cron)
- Vue 3 + Nuxt 4 + TypeScript
- Tailwind CSS v4
- SQLite数据库
- OpenAI API (用于二次汇总)

---

## Phase 1 任务列表 (预计2周)

### Task 1: 创建数据库模型和迁移

**Files:**
- Create: `backend-go/internal/digest/models.go`
- Create: `backend-go/internal/digest/migration.go`
- Modify: `backend-go/cmd/server/main.go:1-50`

**Step 1: 创建digest配置模型**

```go
// internal/digest/models.go
package digest

import (
    "time"
)

type DigestConfig struct {
    ID                       uint       `gorm:"primaryKey" json:"id"`
    DailyEnabled             bool      `gorm:"default:true" json:"daily_enabled"`
    DailyTime                string    `gorm:"size:5;default:'09:00'" json:"daily_time"`
    WeeklyEnabled            bool      `gorm:"default:true" json:"weekly_enabled"`
    WeeklyDay                string    `gorm:"size:10;default:'Monday'" json:"weekly_day"`
    WeeklyTime               string    `gorm:"size:5;default:'09:00'" json:"weekly_time"`
    FeishuEnabled            bool      `gorm:"default:true" json:"feishu_enabled"`
    FeishuWebhookURL         string    `gorm:"size:500" json:"feishu_webhook_url"`
    FeishuPushSummary        bool      `gorm:"default:true" json:"feishu_push_summary"`
    FeishuPushDetails        bool      `gorm:"default:true" json:"feishu_push_details"`
    ObsidianEnabled          bool      `gorm:"default:true" json:"obsidian_enabled"`
    ObsidianVaultPath        string    `gorm:"size:1000" json:"obsidian_vault_path"`
    ObsidianDailyDigest      bool      `gorm:"default:true" json:"obsidian_daily_digest"`
    ObsidianWeeklyDigest     bool      `gorm:"default:true" json:"obsidian_weekly_digest"`
    CreatedAt                time.Time `json:"created_at"`
    UpdatedAt                time.Time `json:"updated_at"`
}

func (DigestConfig) TableName() string {
    return "digest_configs"
}
```

**Step 2: 创建数据库迁移**

```go
// internal/digest/migration.go
package digest

import (
    "log"
    "my-robot-backend/pkg/database"
)

func Migrate() {
    err := database.DB.AutoMigrate(&DigestConfig{})
    if err != nil {
        log.Printf("Failed to migrate digest models: %v", err)
    } else {
        log.Println("Digest models migrated successfully")
    }

    // 创建默认配置
    var count int64
    database.DB.Model(&DigestConfig{}).Count(&count)
    if count == 0 {
        defaultConfig := DigestConfig{
            DailyEnabled:         true,
            DailyTime:            "09:00",
            WeeklyEnabled:        true,
            WeeklyDay:            "Monday",
            WeeklyTime:           "09:00",
            FeishuEnabled:        true,
            FeishuPushSummary:    true,
            FeishuPushDetails:    true,
            ObsidianEnabled:      true,
            ObsidianDailyDigest:  true,
            ObsidianWeeklyDigest: true,
        }
        database.DB.Create(&defaultConfig)
        log.Println("Default digest config created")
    }
}
```

**Step 3: 在server启动时调用迁移**

修改 `cmd/server/main.go`，在启动函数中添加:

```go
import (
    "my-robot-backend/internal/digest"
)

func main() {
    // ... 现有代码 ...

    // 运行digest迁移
    digest.Migrate()

    // ... 现有代码 ...
}
```

**Step 4: 运行迁移验证**

```bash
cd backend-go
go run cmd/server/main.go
```

Expected output: "Digest models migrated successfully" 和 "Default digest config created"

**Step 5: 验证数据库表**

```bash
sqlite3 rss_reader.db ".schema digest_configs"
```

Expected: 看到digest_configs表结构

**Step 6: 提交**

```bash
git add backend-go/internal/digest/ backend-go/cmd/server/main.go
git commit -m "feat(digest): add database models and migration for digest configs"
```

---

### Task 2: 实现DigestScheduler定时任务

**Files:**
- Create: `backend-go/internal/digest/scheduler.go`
- Modify: `backend-go/cmd/server/main.go`

**Step 1: 创建DigestScheduler结构**

```go
// internal/digest/scheduler.go
package digest

import (
    "log"
    "time"
    "github.com/robfig/cron/v3"
)

type DigestScheduler struct {
    cron        *cron.Cron
    isRunning   bool
}

func NewDigestScheduler() *DigestScheduler {
    return &DigestScheduler{
        cron:      cron.New(),
        isRunning: false,
    }
}

func (s *DigestScheduler) Start() error {
    if s.isRunning {
        log.Println("Digest scheduler already running")
        return nil
    }

    // 加载配置
    config, err := s.LoadConfig()
    if err != nil {
        log.Printf("Failed to load digest config: %v", err)
        return err
    }

    // 添加每日任务
    if config.DailyEnabled {
        dailyExpr := fmt.Sprintf("0 %s * * *", config.DailyTime)
        if _, err := s.cron.AddFunc(dailyExpr, s.generateDailyDigest); err != nil {
            return err
        }
        log.Printf("Daily digest scheduled at %s", config.DailyTime)
    }

    // 添加每周任务
    if config.WeeklyEnabled {
        weeklyExpr := fmt.Sprintf("0 %s * * %s", config.WeeklyTime, s.weekdayToNumber(config.WeeklyDay))
        if _, err := s.cron.AddFunc(weeklyExpr, s.generateWeeklyDigest); err != nil {
            return err
        }
        log.Printf("Weekly digest scheduled at %s on %s", config.WeeklyTime, config.WeeklyDay)
    }

    s.cron.Start()
    s.isRunning = true
    log.Println("Digest scheduler started")
    return nil
}

func (s *DigestScheduler) Stop() {
    if !s.isRunning {
        return
    }
    s.cron.Stop()
    s.isRunning = false
    log.Println("Digest scheduler stopped")
}

func (s *DigestScheduler) LoadConfig() (*DigestConfig, error) {
    var config DigestConfig
    err := database.DB.First(&config).Error
    return &config, err
}

func (s *DigestScheduler) weekdayToNumber(day string) string {
    weekdays := map[string]string{
        "Monday":    "1",
        "Tuesday":   "2",
        "Wednesday": "3",
        "Thursday":  "4",
        "Friday":    "5",
        "Saturday":  "6",
        "Sunday":    "0",
    }
    return weekdays[day]
}

func (s *DigestScheduler) generateDailyDigest() {
    log.Println("Starting daily digest generation")
    // Task 3实现
}

func (s *DigestScheduler) generateWeeklyDigest() {
    log.Println("Starting weekly digest generation")
    // Task 3实现
}
```

**Step 2: 在server中集成scheduler**

修改 `cmd/server/main.go`:

```go
import (
    "my-robot-backend/internal/digest"
)

var digestScheduler *digest.DigestScheduler

func main() {
    // ... 现有代码 ...

    // 启动digest scheduler
    digestScheduler = digest.NewDigestScheduler()
    if err := digestScheduler.Start(); err != nil {
        log.Printf("Failed to start digest scheduler: %v", err)
    }
    defer digestScheduler.Stop()

    // ... 现有代码 ...
}
```

**Step 3: 运行server验证scheduler启动**

```bash
cd backend-go
go run cmd/server/main.go
```

Expected: "Digest scheduler started" 和 "Daily digest scheduled at 09:00"

**Step 4: 提交**

```bash
git add backend-go/internal/digest/scheduler.go backend-go/cmd/server/main.go
git commit -m "feat(digest): add DigestScheduler for daily/weekly digest generation"
```

---

### Task 3: 实现日报/周报内容生成器

**Files:**
- Create: `backend-go/internal/digest/generator.go`
- Create: `backend-go/internal/digest/generator_test.go`

**Step 1: 创建generator结构和方法**

```go
// internal/digest/generator.go
package digest

import (
    "fmt"
    "time"
    "my-robot-backend/internal/models"
    "my-robot-backend/pkg/database"
)

type DigestGenerator struct {
    config *DigestConfig
}

func NewDigestGenerator(config *DigestConfig) *DigestGenerator {
    return &DigestGenerator{config: config}
}

type CategoryDigest struct {
    CategoryName  string
    CategoryID    uint
    FeedCount     int
    AISummaries   []models.AISummary
}

func (g *DigestGenerator) GenerateDailyDigest(date time.Time) ([]CategoryDigest, error) {
    startTime := date.Truncate(24 * time.Hour)
    endTime := startTime.Add(24 * time.Hour)

    var summaries []models.AISummary
    err := database.DB.Where("created_at >= ? AND created_at < ?", startTime, endTime).
        Preload("Feed").
        Preload("Category").
        Find(&summaries).Error

    if err != nil {
        return nil, err
    }

    return g.groupByCategory(summaries), nil
}

func (g *DigestGenerator) GenerateWeeklyDigest(date time.Time) ([]CategoryDigest, error) {
    // 找到本周一
    weekday := int(date.Weekday())
    if weekday == 0 {
        weekday = 7
    }
    monday := date.AddDate(0, 0, -weekday+1)
    monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.Local)
    sunday := monday.AddDate(0, 0, 7)

    var summaries []models.AISummary
    err := database.DB.Where("created_at >= ? AND created_at < ?", monday, sunday).
        Preload("Feed").
        Preload("Category").
        Find(&summaries).Error

    if err != nil {
        return nil, err
    }

    return g.groupByCategory(summaries), nil
}

func (g *DigestGenerator) groupByCategory(summaries []models.AISummary) []CategoryDigest {
    categoryMap := make(map[uint]*CategoryDigest)

    for _, summary := range summaries {
        categoryID := uint(0) // 默认"未分类"
        categoryName := "未分类"

        if summary.Category != nil {
            categoryID = summary.Category.ID
            categoryName = summary.Category.Name
        }

        if _, exists := categoryMap[categoryID]; !exists {
            categoryMap[categoryID] = &CategoryDigest{
                CategoryName: categoryName,
                CategoryID:   categoryID,
                FeedCount:    0,
                AISummaries:  []models.AISummary{},
            }
        }

        categoryMap[categoryID].AISummaries = append(categoryMap[categoryID].AISummaries, summary)
        categoryMap[categoryID].FeedCount++
    }

    result := make([]CategoryDigest, 0, len(categoryMap))
    for _, digest := range categoryMap {
        result = append(result, *digest)
    }

    return result
}
```

**Step 2: 编写单元测试**

```go
// internal/digest/generator_test.go
package digest

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
)

func TestGroupByCategory(t *testing.T) {
    config := &DigestConfig{}
    generator := NewDigestGenerator(config)

    summaries := []models.AISummary{
        {
            ID:     1,
            Title:  "Test Summary 1",
            Category: &Category{ID: 1, Name: "AI技术"},
        },
        {
            ID:     2,
            Title:  "Test Summary 2",
            Category: &Category{ID: 1, Name: "AI技术"},
        },
        {
            ID:     3,
            Title:  "Test Summary 3",
            Category: &Category{ID: 2, Name: "前端开发"},
        },
    }

    result := generator.groupByCategory(summaries)

    assert.Equal(t, 2, len(result))
    assert.Equal(t, "AI技术", result[0].CategoryName)
    assert.Equal(t, 2, result[0].FeedCount)
    assert.Equal(t, "前端开发", result[1].CategoryName)
    assert.Equal(t, 1, result[1].FeedCount)
}
```

**Step 3: 运行测试**

```bash
cd backend-go
go test ./internal/digest/generator_test.go ./internal/digest/generator.go -v
```

Expected: 所有测试通过

**Step 4: 提交**

```bash
git add backend-go/internal/digest/generator.go backend-go/internal/digest/generator_test.go
git commit -m "feat(digest): add digest generator for grouping summaries by category"
```

---

### Task 4: 实现飞书推送功能

**Files:**
- Create: `backend-go/internal/digest/feishu.go`
- Create: `backend-go/internal/digest/feishu_test.go`

**Step 1: 创建飞书推送结构**

```go
// internal/digest/feishu.go
package digest

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type FeishuNotifier struct {
    webhookURL string
    client     *http.Client
}

type FeishuMessage struct {
    MsgType string              `json:"msg_type"`
    Content map[string]interface{} `json:"content"`
}

func NewFeishuNotifier(webhookURL string) *FeishuNotifier {
    return &FeishuNotifier{
        webhookURL: webhookURL,
        client:     &http.Client{Timeout: 10 * time.Second},
    }
}

func (f *FeishuNotifier) SendSummary(title, content string) error {
    message := FeishuMessage{
        MsgType: "text",
        Content: map[string]interface{}{
            "text": fmt.Sprintf("%s\n\n%s", title, content),
        },
    }

    return f.send(message)
}

func (f *FeishuNotifier) SendCard(title, content string) error {
    message := FeishuMessage{
        MsgType: "interactive",
        Content: map[string]interface{}{
            "config": map[string]interface{}{
                "wide_screen_mode": true,
            },
            "elements": []map[string]interface{}{
                {
                    "tag":  "div",
                    "text": map[string]interface{}{
                        "tag":  "lark_md",
                        "content": content,
                    },
                },
            },
        },
    }

    return f.send(message)
}

func (f *FeishuNotifier) send(message FeishuMessage) error {
    body, err := json.Marshal(message)
    if err != nil {
        return err
    }

    req, err := http.NewRequest("POST", f.webhookURL, bytes.NewReader(body))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")

    resp, err := f.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("feishu API returned status %d", resp.StatusCode)
    }

    return nil
}
```

**Step 2: 编写测试**

```go
// internal/digest/feishu_test.go
package digest

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestNewFeishuNotifier(t *testing.T) {
    notifier := NewFeishuNotifier("https://test.webhook.url")
    assert.NotNil(t, notifier)
    assert.Equal(t, "https://test.webhook.url", notifier.webhookURL)
}

// 可以添加mock server测试实际发送逻辑
```

**Step 3: 运行测试**

```bash
cd backend-go
go test ./internal/digest/feishu_test.go ./internal/digest/feishu.go -v
```

**Step 4: 提交**

```bash
git add backend-go/internal/digest/feishu.go backend-go/internal/digest/feishu_test.go
git commit -m "feat(digest): add Feishu notifier for push notifications"
```

---

### Task 5: 实现Obsidian导出功能

**Files:**
- Create: `backend-go/internal/digest/obsidian.go`
- Create: `backend-go/internal/digest/obsidian_test.go`

**Step 1: 创建Obsidian导出结构**

```go
// internal/digest/obsidian.go
package digest

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
)

type ObsidianExporter struct {
    vaultPath string
}

func NewObsidianExporter(vaultPath string) *ObsidianExporter {
    return &ObsidianExporter{
        vaultPath: vaultPath,
    }
}

func (e *ObsidianExporter) ExportDailyDigest(date time.Time, digests []CategoryDigest) error {
    for _, digest := range digests {
        if err := e.exportCategoryDaily(date, digest); err != nil {
            return err
        }
    }
    return nil
}

func (e *ObsidianExporter) ExportWeeklyDigest(weekStart time.Time, digests []CategoryDigest) error {
    weekNum := e.getWeekNumber(weekStart)
    for _, digest := range digests {
        if err := e.exportCategoryWeekly(weekStart, weekNum, digest); err != nil {
            return err
        }
    }
    return nil
}

func (e *ObsidianExporter) exportCategoryDaily(date time.Time, digest CategoryDigest) error {
    dirPath := filepath.Join(e.vaultPath, "Daily", digest.CategoryName)
    if err := os.MkdirAll(dirPath, 0755); err != nil {
        return err
    }

    filename := fmt.Sprintf("%s-日报.md", date.Format("2006-01-02"))
    filePath := filepath.Join(dirPath, filename)

    content := e.generateDailyMarkdown(date, digest)

    return os.WriteFile(filePath, []byte(content), 0644)
}

func (e *ObsidianExporter) exportCategoryWeekly(weekStart time.Time, weekNum int, digest CategoryDigest) error {
    dirPath := filepath.Join(e.vaultPath, "Weekly", digest.CategoryName)
    if err := os.MkdirAll(dirPath, 0755); err != nil {
        return err
    }

    filename := fmt.Sprintf("%d-W%d-周报.md", weekStart.Year(), weekNum)
    filePath := filepath.Join(dirPath, filename)

    weekEnd := weekStart.AddDate(0, 0, 6)
    content := e.generateWeeklyMarkdown(weekStart, weekEnd, digest)

    return os.WriteFile(filePath, []byte(content), 0644)
}

func (e *ObsidianExporter) generateDailyMarkdown(date time.Time, digest CategoryDigest) string {
    return fmt.Sprintf(`---
category: %s
date: %s
type: daily-digest
tags: [daily, %s, %s]
---

# %s - %s日报

## 本日概要

本日共生成%d份订阅源总结。

## 订阅源总结

%s
`,
        digest.CategoryName,
        date.Format("2006-01-02"),
        digest.CategoryName,
        date.Format("2006-01"),
        digest.CategoryName,
        date.Format("2006年1月2日"),
        digest.FeedCount,
        e.generateSummariesSection(digest.AISummaries),
    )
}

func (e *ObsidianExporter) generateWeeklyMarkdown(weekStart, weekEnd time.Time, digest CategoryDigest) string {
    weekNum := e.getWeekNumber(weekStart)
    return fmt.Sprintf(`---
category: %s
week: %d-W%d
date_range: %s ~ %s
type: weekly-digest
tags: [weekly, %s]
---

# %s - 第%d周周报 (%s-%s)

## 本周概览

本周共生成%d份订阅源总结。

## 按订阅源汇总

%s
`,
        digest.CategoryName,
        weekStart.Year(),
        weekNum,
        weekStart.Format("2006-01-02"),
        weekEnd.Format("2006-01-02"),
        digest.CategoryName,
        digest.CategoryName,
        weekNum,
        weekStart.Format("1.2"),
        weekEnd.Format("1.2"),
        digest.FeedCount,
        e.generateSummariesSection(digest.AISummaries),
    )
}

func (e *ObsidianExporter) generateSummariesSection(summaries []models.AISummary) string {
    var content string
    for _, summary := range summaries {
        feedName := "未知订阅源"
        if summary.Feed != nil {
            feedName = summary.Feed.Title
        }
        content += fmt.Sprintf("### %s\n\n%s\n\n", feedName, summary.Summary)
    }
    return content
}

func (e *ObsidianExporter) getWeekNumber(date time.Time) int {
    _, week := date.ISOWeek()
    return week
}
```

**Step 2: 编写测试**

```go
// internal/digest/obsidian_test.go
package digest

import (
    "os"
    "path/filepath"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
)

func TestObsidianExporter(t *testing.T) {
    // 创建临时目录
    tempDir := t.TempDir()
    exporter := NewObsidianExporter(tempDir)

    date := time.Date(2026, 3, 4, 0, 0, 0, 0, time.Local)
    digests := []CategoryDigest{
        {
            CategoryName: "AI技术",
            CategoryID:   1,
            FeedCount:    2,
            AISummaries: []models.AISummary{
                {ID: 1, Title: "Test 1", Summary: "Content 1"},
            },
        },
    }

    err := exporter.ExportDailyDigest(date, digests)
    assert.NoError(t, err)

    // 验证文件创建
    expectedPath := filepath.Join(tempDir, "Daily", "AI技术", "2026-03-04-日报.md")
    _, err = os.Stat(expectedPath)
    assert.NoError(t, err, "Daily digest file should be created")
}
```

**Step 3: 运行测试**

```bash
cd backend-go
go test ./internal/digest/obsidian_test.go ./internal/digest/obsidian.go -v
```

**Step 4: 提交**

```bash
git add backend-go/internal/digest/obsidian.go backend-go/internal/digest/obsidian_test.go
git commit -m "feat(digest): add Obsidian exporter for markdown export"
```

---

### Task 6: 集成所有模块到scheduler

**Files:**
- Modify: `backend-go/internal/digest/scheduler.go`

**Step 1: 完善scheduler实现**

修改 `internal/digest/scheduler.go`:

```go
func (s *DigestScheduler) generateDailyDigest() {
    log.Println("Starting daily digest generation")

    config, err := s.LoadConfig()
    if err != nil {
        log.Printf("Failed to load config: %v", err)
        return
    }

    generator := NewDigestGenerator(config)
    date := time.Now()

    digests, err := generator.GenerateDailyDigest(date)
    if err != nil {
        log.Printf("Failed to generate daily digest: %v", err)
        return
    }

    log.Printf("Generated daily digest for %d categories", len(digests))

    // 飞书推送
    if config.FeishuEnabled {
        s.sendFeishuDigest("daily", date, digests, config)
    }

    // Obsidian导出
    if config.ObsidianEnabled && config.ObsidianDailyDigest {
        s.exportToObsidian("daily", date, digests, config)
    }
}

func (s *DigestScheduler) generateWeeklyDigest() {
    log.Println("Starting weekly digest generation")

    config, err := s.LoadConfig()
    if err != nil {
        log.Printf("Failed to load config: %v", err)
        return
    }

    generator := NewDigestGenerator(config)
    date := time.Now()

    digests, err := generator.GenerateWeeklyDigest(date)
    if err != nil {
        log.Printf("Failed to generate weekly digest: %v", err)
        return
    }

    log.Printf("Generated weekly digest for %d categories", len(digests))

    // 飞书推送
    if config.FeishuEnabled {
        s.sendFeishuDigest("weekly", date, digests, config)
    }

    // Obsidian导出
    if config.ObsidianEnabled && config.ObsidianWeeklyDigest {
        s.exportToObsidian("weekly", date, digests, config)
    }
}

func (s *DigestScheduler) sendFeishuDigest(digestType string, date time.Time, digests []CategoryDigest, config *DigestConfig) {
    notifier := NewFeishuNotifier(config.FeishuWebhookURL)

    if config.FeishuPushSummary {
        // 发送汇总通知
        title, content := s.generateSummaryMessage(digestType, date, digests)
        if err := notifier.SendCard(title, content); err != nil {
            log.Printf("Failed to send Feishu summary: %v", err)
        }
    }

    if config.FeishuPushDetails {
        // 发送细节通知
        for _, digest := range digests {
            for _, summary := range digest.AISummaries {
                title := fmt.Sprintf("%s 今日新闻", digest.CategoryName)
                if err := notifier.SendCard(title, summary.Summary); err != nil {
                    log.Printf("Failed to send Feishu details: %v", err)
                }
            }
        }
    }
}

func (s *DigestScheduler) exportToObsidian(digestType string, date time.Time, digests []CategoryDigest, config *DigestConfig) {
    exporter := NewObsidianExporter(config.ObsidianVaultPath)

    var err error
    if digestType == "daily" {
        err = exporter.ExportDailyDigest(date, digests)
    } else {
        err = exporter.ExportWeeklyDigest(date, digests)
    }

    if err != nil {
        log.Printf("Failed to export to Obsidian: %v", err)
    }
}

func (s *DigestScheduler) generateSummaryMessage(digestType string, date time.Time, digests []CategoryDigest) (string, string) {
    title := fmt.Sprintf("今日新闻摘要 📋", date)
    if digestType == "weekly" {
        title = fmt.Sprintf("本周新闻摘要 📋")
    }

    var content string
    for _, digest := range digests {
        content += fmt.Sprintf("### %s\n\n%d个订阅源\n\n", digest.CategoryName, digest.FeedCount)
    }

    return title, content
}
```

**Step 2: 运行server验证**

```bash
cd backend-go
go run cmd/server/main.go
```

Expected: "Digest scheduler started"

**Step 3: 提交**

```bash
git add backend-go/internal/digest/scheduler.go
git commit -m "feat(digest): integrate generator, feishu notifier and obsidian exporter into scheduler"
```

---

### Task 7: 实现API配置端点

**Files:**
- Create: `backend-go/internal/handlers/digest.go`
- Modify: `backend-go/internal/routes/routes.go`

**Step 1: 创建digest handlers**

```go
// internal/handlers/digest.go
package handlers

import (
    "net/http"
    "my-robot-backend/internal/digest"
    "my-robot-backend/pkg/database"
    "github.com/gin-gonic/gin"
)

func GetDigestConfig(c *gin.Context) {
    var config digest.DigestConfig
    if err := database.DB.First(&config).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "success": false,
            "error":   "Config not found",
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    config,
    })
}

func UpdateDigestConfig(c *gin.Context) {
    var req digest.DigestConfig
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error":   err.Error(),
        })
        return
    }

    var config digest.DigestConfig
    if err := database.DB.First(&config).Error; err != nil {
        // 创建新配置
        if err := database.DB.Create(&req).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "success": false,
                "error":   err.Error(),
            })
            return
        }
    } else {
        // 更新现有配置
        req.ID = config.ID
        if err := database.DB.Save(&req).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "success": false,
                "error":   err.Error(),
            })
            return
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Config updated successfully",
        "data":    req,
    })
}

func TestFeishuPush(c *gin.Context) {
    var config digest.DigestConfig
    if err := database.DB.First(&config).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "success": false,
            "error":   "Config not found",
        })
        return
    }

    notifier := digest.NewFeishuNotifier(config.FeishuWebhookURL)
    if err := notifier.SendSummary("测试消息", "这是一条测试消息"); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error":   err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Test message sent successfully",
    })
}

func TestObsidianWrite(c *gin.Context) {
    var config digest.DigestConfig
    if err := database.DB.First(&config).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "success": false,
            "error":   "Config not found",
        })
        return
    }

    exporter := digest.NewObsidianExporter(config.ObsidianVaultPath)
    testDigest := []digest.CategoryDigest{
        {
            CategoryName: "测试分类",
            FeedCount:    1,
        },
    }

    if err := exporter.ExportDailyDigest(c.Request.Context().Value("now").(time.Time), testDigest); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "success": false,
            "error":   err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Test file written successfully",
    })
}
```

**Step 2: 注册路由**

找到并修改 `internal/routes/routes.go`:

```go
import (
    "my-robot-backend/internal/handlers"
)

func SetupRoutes(r *gin.Engine) {
    // ... 现有路由 ...

    // Digest API routes
    digestGroup := api.Group("/digest")
    {
        digestGroup.GET("/config", handlers.GetDigestConfig)
        digestGroup.PUT("/config", handlers.UpdateDigestConfig)
        digestGroup.POST("/test-feishu", handlers.TestFeishuPush)
        digestGroup.POST("/test-obsidian", handlers.TestObsidianWrite)
    }
}
```

**Step 3: 测试API**

```bash
# 获取配置
curl http://localhost:5000/api/digest/config

# 测试飞书推送
curl -X POST http://localhost:5000/api/digest/test-feishu
```

**Step 4: 提交**

```bash
git add backend-go/internal/handlers/digest.go backend-go/internal/routes/routes.go
git commit -m "feat(digest): add API endpoints for digest configuration"
```

---

### Task 8: 前端 - 添加侧边栏菜单

**Files:**
- Modify: `front/app/components/layout/SidebarContent.vue`

**Step 1: 添加日报周报菜单项**

在 `SidebarContent.vue` 的菜单列表中添加:

```vue
<script setup lang="ts">
// ... 现有代码 ...

const menuItems = [
  // ... 现有菜单项 ...
  {
    icon: 'mdi:newspaper-variant-multiple',
    label: '日报周报',
    route: '/digest'
  }
]
</script>
```

**Step 2: 测试菜单显示**

```bash
cd front
pnpm dev
```

访问 http://localhost:3001，验证侧边栏显示"日报周报"菜单

**Step 3: 提交**

```bash
git add front/app/components/layout/SidebarContent.vue
git commit -m "feat(digest): add digest menu item to sidebar"
```

---

### Task 9: 前端 - 创建日报周报列表页面

**Files:**
- Create: `front/app/pages/digest/index.vue`
- Create: `front/app/components/digest/DigestList.vue`
- Create: `front/app/composables/api/digest.ts`

**Step 1: 创建digest API composable**

```typescript
// composables/api/digest.ts
import { apiClient } from './client'

export interface DigestConfig {
  daily_enabled: boolean
  daily_time: string
  weekly_enabled: boolean
  weekly_day: string
  weekly_time: string
  feishu_enabled: boolean
  feishu_webhook_url: string
  feishu_push_summary: boolean
  feishu_push_details: boolean
  obsidian_enabled: boolean
  obsidian_vault_path: string
  obsidian_daily_digest: boolean
  obsidian_weekly_digest: boolean
}

export function useDigestApi() {
  return {
    async getConfig() {
      return apiClient.get<DigestConfig>('/digest/config')
    },

    async updateConfig(config: DigestConfig) {
      return apiClient.put<DigestConfig>('/digest/config', config)
    },

    async testFeishu() {
      return apiClient.post('/digest/test-feishu', {})
    },

    async testObsidian() {
      return apiClient.post('/digest/test-obsidian', {})
    }
  }
}
```

**Step 2: 创建digest列表组件**

```vue
<!-- components/digest/DigestList.vue -->
<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { ref, onMounted } from 'vue'
import { useDigestApi } from '~/composables/api/digest'

const digestApi = useDigestApi()
const loading = ref(false)
const digests = ref([])

onMounted(async () => {
  loading.value = true
  // TODO: 获取日报周报列表
  loading.value = false
})
</script>

<template>
  <div class="digest-list">
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-xl font-bold">日报周报</h2>
      <div class="flex gap-2">
        <button class="px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700">
          刷新
        </button>
      </div>
    </div>

    <div v-if="loading" class="text-center py-12">
      <Icon icon="mdi:loading" width="48" class="animate-spin" />
    </div>

    <div v-else class="space-y-4">
      <!-- TODO: 显示日报周报列表 -->
      <div class="text-center text-ink-light py-12">
        暂无日报周报
      </div>
    </div>
  </div>
</template>
```

**Step 3: 创建页面**

```vue
<!-- pages/digest/index.vue -->
<script setup lang="ts">
import DigestList from '~/components/digest/DigestList.vue'
</script>

<template>
  <div class="h-full">
    <DigestList />
  </div>
</template>
```

**Step 4: 测试页面**

访问 http://localhost:3001/digest

**Step 5: 提交**

```bash
git add front/app/pages/digest/index.vue front/app/components/digest/DigestList.vue front/app/composables/api/digest.ts
git commit -m "feat(digest): add digest list page and API composable"
```

---

### Task 10: 前端 - 创建配置表单

**Files:**
- Create: `front/app/components/digest/DigestSettings.vue`
- Modify: `front/app/components/digest/DigestList.vue`

**Step 1: 创建配置表单组件**

```vue
<!-- components/digest/DigestSettings.vue -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useDigestApi, type DigestConfig } from '~/composables/api/digest'

const digestApi = useDigestApi()
const loading = ref(false)
const saving = ref(false)
const config = ref<DigestConfig>({
  daily_enabled: true,
  daily_time: '09:00',
  weekly_enabled: true,
  weekly_day: 'Monday',
  weekly_time: '09:00',
  feishu_enabled: true,
  feishu_webhook_url: '',
  feishu_push_summary: true,
  feishu_push_details: true,
  obsidian_enabled: true,
  obsidian_vault_path: '',
  obsidian_daily_digest: true,
  obsidian_weekly_digest: true,
})

onMounted(async () => {
  await loadConfig()
})

async function loadConfig() {
  loading.value = true
  try {
    const response = await digestApi.getConfig()
    if (response.success && response.data) {
      config.value = response.data
    }
  } catch (error) {
    console.error('Failed to load config:', error)
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  saving.value = true
  try {
    const response = await digestApi.updateConfig(config.value)
    if (response.success) {
      alert('配置已保存')
    }
  } catch (error) {
    console.error('Failed to save config:', error)
    alert('保存失败')
  } finally {
    saving.value = false
  }
}

async function testFeishu() {
  try {
    const response = await digestApi.testFeishu()
    if (response.success) {
      alert('测试消息已发送，请检查飞书')
    }
  } catch (error) {
    console.error('Failed to test Feishu:', error)
    alert('测试失败')
  }
}

async function testObsidian() {
  try {
    const response = await digestApi.testObsidian()
    if (response.success) {
      alert('测试文件已写入，请检查Obsidian vault')
    }
  } catch (error) {
    console.error('Failed to test Obsidian:', error)
    alert('测试失败')
  }
}
</script>

<template>
  <div class="digest-settings">
    <h3 class="text-lg font-bold mb-4">日报周报设置</h3>

    <div v-if="loading" class="text-center py-12">
      <Icon icon="mdi:loading" width="48" class="animate-spin" />
    </div>

    <div v-else class="space-y-6">
      <!-- 基础设置 -->
      <div class="space-y-4">
        <h4 class="font-semibold">基础设置</h4>

        <div class="flex items-center justify-between">
          <label>启用日报</label>
          <input
            v-model="config.daily_enabled"
            type="checkbox"
            class="w-5 h-5"
          >
        </div>

        <div v-if="config.daily_enabled" class="pl-4">
          <label class="block text-sm mb-1">日报时间</label>
          <input
            v-model="config.daily_time"
            type="time"
            class="border rounded px-3 py-2"
          >
        </div>

        <div class="flex items-center justify-between">
          <label>启用周报</label>
          <input
            v-model="config.weekly_enabled"
            type="checkbox"
            class="w-5 h-5"
          >
        </div>

        <div v-if="config.weekly_enabled" class="pl-4 space-y-2">
          <div>
            <label class="block text-sm mb-1">周报星期</label>
            <select v-model="config.weekly_day" class="border rounded px-3 py-2">
              <option value="Monday">周一</option>
              <option value="Tuesday">周二</option>
              <option value="Wednesday">周三</option>
              <option value="Thursday">周四</option>
              <option value="Friday">周五</option>
              <option value="Saturday">周六</option>
              <option value="Sunday">周日</option>
            </select>
          </div>
          <div>
            <label class="block text-sm mb-1">周报时间</label>
            <input
              v-model="config.weekly_time"
              type="time"
              class="border rounded px-3 py-2"
            >
          </div>
        </div>
      </div>

      <!-- 飞书设置 -->
      <div class="space-y-4">
        <h4 class="font-semibold">飞书推送</h4>

        <div class="flex items-center justify-between">
          <label>启用飞书推送</label>
          <input
            v-model="config.feishu_enabled"
            type="checkbox"
            class="w-5 h-5"
          >
        </div>

        <div v-if="config.feishu_enabled" class="pl-4 space-y-2">
          <div>
            <label class="block text-sm mb-1">Webhook URL</label>
            <input
              v-model="config.feishu_webhook_url"
              type="text"
              placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..."
              class="w-full border rounded px-3 py-2"
            >
          </div>

          <div class="flex items-center justify-between">
            <label>推送汇总通知</label>
            <input
              v-model="config.feishu_push_summary"
              type="checkbox"
              class="w-5 h-5"
            >
          </div>

          <div class="flex items-center justify-between">
            <label>推送细节通知</label>
            <input
              v-model="config.feishu_push_details"
              type="checkbox"
              class="w-5 h-5"
            >
          </div>

          <button
            @click="testFeishu"
            class="px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700"
          >
            测试推送
          </button>
        </div>
      </div>

      <!-- Obsidian设置 -->
      <div class="space-y-4">
        <h4 class="font-semibold">Obsidian导出</h4>

        <div class="flex items-center justify-between">
          <label>启用Obsidian导出</label>
          <input
            v-model="config.obsidian_enabled"
            type="checkbox"
            class="w-5 h-5"
          >
        </div>

        <div v-if="config.obsidian_enabled" class="pl-4 space-y-2">
          <div>
            <label class="block text-sm mb-1">Vault路径</label>
            <input
              v-model="config.obsidian_vault_path"
              type="text"
              placeholder="/path/to/ObsidianVault"
              class="w-full border rounded px-3 py-2"
            >
          </div>

          <div class="flex items-center justify-between">
            <label>导出日报</label>
            <input
              v-model="config.obsidian_daily_digest"
              type="checkbox"
              class="w-5 h-5"
            >
          </div>

          <div class="flex items-center justify-between">
            <label>导出周报</label>
            <input
              v-model="config.obsidian_weekly_digest"
              type="checkbox"
              class="w-5 h-5"
            >
          </div>

          <button
            @click="testObsidian"
            class="px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700"
          >
            测试写入
          </button>
        </div>
      </div>

      <!-- 保存按钮 -->
      <div class="pt-4 border-t">
        <button
          @click="saveConfig"
          :disabled="saving"
          class="w-full px-4 py-3 bg-amber-600 text-white rounded-lg hover:bg-amber-700 disabled:opacity-50"
        >
          {{ saving ? '保存中...' : '保存配置' }}
        </button>
      </div>
    </div>
  </div>
</template>
```

**Step 2: 在列表页面添加设置按钮**

修改 `components/digest/DigestList.vue`:

```vue
<template>
  <div class="digest-list">
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-xl font-bold">日报周报</h2>
      <button
        @click="showSettings = true"
        class="px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700"
      >
        设置
      </button>
    </div>

    <!-- 对话框 -->
    <div
      v-if="showSettings"
      class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      @click.self="showSettings = false"
    >
      <div class="bg-white rounded-lg p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-bold">设置</h3>
          <button @click="showSettings = false" class="text-ink-medium hover:text-ink-dark">
            <Icon icon="mdi:close" width="24" height="24" />
          </button>
        </div>
        <DigestSettings />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import DigestSettings from './DigestSettings.vue'

const showSettings = ref(false)
</script>
```

**Step 3: 测试配置功能**

访问 http://localhost:3001/digest，点击"设置"按钮

**Step 4: 提交**

```bash
git add front/app/components/digest/DigestSettings.vue front/app/components/digest/DigestList.vue
git commit -m "feat(digest): add digest settings form"
```

---

### Task 11: 添加日报周报详情页面

**Files:**
- Create: `front/app/components/digest/DigestDetail.vue`
- Modify: `front/app/pages/digest/[id].vue`

**Step 1: 创建详情组件**

```vue
<!-- components/digest/DigestDetail.vue -->
<script setup lang="ts">
import { ref, computed } from 'vue'
import { marked } from 'marked'

const props = defineProps<{
  digest: any
}>()

const renderedContent = computed(() => {
  if (!props.digest) return ''
  return marked(props.digest.content)
})
</script>

<template>
  <div class="digest-detail">
    <div class="prose max-w-none">
      <div v-html="renderedContent" class="markdown-content" />
    </div>
  </div>
</template>

<style scoped>
.markdown-content {
  color: var(--color-ink-dark);
  line-height: 1.75;
}

.markdown-content :deep(h1),
.markdown-content :deep(h2),
.markdown-content :deep(h3) {
  font-weight: 700;
  margin-top: 1.75em;
  margin-bottom: 0.75em;
}

.markdown-content :deep(p) {
  margin-bottom: 1.25em;
}

.markdown-content :deep(a) {
  color: var(--color-ink-500);
  text-decoration: none;
  border-bottom: 1px solid transparent;
}

.markdown-content :deep(a:hover) {
  border-bottom-color: var(--color-ink-500);
}
</style>
```

**Step 2: 创建动态路由页面**

```vue
<!-- pages/digest/[id].vue -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import DigestDetail from '~/components/digest/DigestDetail.vue'

const route = useRoute()
const digest = ref(null)
const loading = ref(true)

onMounted(async () => {
  const id = route.params.id
  // TODO: 获取digest详情
  loading.value = false
})
</script>

<template>
  <div class="h-full flex flex-col">
    <div v-if="loading" class="flex-1 flex items-center justify-center">
      <Icon icon="mdi:loading" width="48" class="animate-spin" />
    </div>
    <DigestDetail v-else-if="digest" :digest="digest" />
    <div v-else class="flex-1 flex items-center justify-center">
      <div class="text-center">
        <Icon icon="mdi:file-document-outline" width="64" class="text-ink-light" />
        <p class="mt-4 text-ink-light">未找到该日报周报</p>
      </div>
    </div>
  </div>
</template>
```

**Step 3: 测试详情页**

访问 http://localhost:3001/digest/1

**Step 4: 提交**

```bash
git add front/app/components/digest/DigestDetail.vue front/app/pages/digest/[id].vue
git commit -m "feat(digest): add digest detail page"
```

---

### Task 12: 添加数据库迁移脚本

**Files:**
- Create: `backend-go/cmd/migrate-digest/main.go`

**Step 1: 创建迁移脚本**

```go
// cmd/migrate-digest/main.go
package main

import (
    "fmt"
    "log"
    "my-robot-backend/internal/digest"
    "my-robot-backend/pkg/database"
)

func main() {
    // 初始化数据库连接
    if err := database.Init(); err != nil {
        log.Fatalf("Failed to init database: %v", err)
    }

    // 运行迁移
    digest.Migrate()

    fmt.Println("Digest migration completed successfully")
}
```

**Step 2: 测试迁移**

```bash
cd backend-go
go run cmd/migrate-digest/main.go
```

**Step 3: 提交**

```bash
git add backend-go/cmd/migrate-digest/main.go
git commit -m "feat(digest): add standalone migration script"
```

---

### Task 13: 添加测试数据

**Files:**
- Create: `backend-go/cmd/test-digest/main.go`

**Step 1: 创建测试数据脚本**

```go
// cmd/test-digest/main.go
package main

import (
    "log"
    "time"
    "my-robot-backend/pkg/database"
    "my-robot-backend/internal/models"
)

func main() {
    if err := database.Init(); err != nil {
        log.Fatalf("Failed to init database: %v", err)
    }

    // 创建测试分类
    category := models.Category{
        Name: "AI技术",
        Slug: "ai-tech",
        Icon: "robot",
        Color: "#6366f1",
    }
    database.DB.Create(&category)

    // 创建测试feed
    feed := models.Feed{
        Title:        "TechCrunch",
        URL:          "https://techcrunch.com/feed/",
        CategoryID:   &category.ID,
        AISummaryEnabled: true,
    }
    database.DB.Create(&feed)

    // 创建测试AI总结
    summary := models.AISummary{
        FeedID:       &feed.ID,
        CategoryID:   &category.ID,
        Title:        "TechCrunch - 2026年3月4日测试",
        Summary:      "## 核心主题\n\n这是一个测试总结...",
        ArticleCount: 5,
        TimeRange:    180,
        CreatedAt:    time.Now(),
    }
    database.DB.Create(&summary)

    log.Println("Test data created successfully")
}
```

**Step 2: 运行测试脚本**

```bash
cd backend-go
go run cmd/test-digest/main.go
```

**Step 3: 提交**

```bash
git add backend-go/cmd/test-digest/main.go
git commit -m "test(digest): add test data generation script"
```

---

### Task 14: 集成测试

**Files:**
- Create: `backend-go/internal/digest/integration_test.go`

**Step 1: 创建集成测试**

```go
// internal/digest/integration_test.go
package digest

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
)

func TestDigestWorkflow(t *testing.T) {
    // 1. 创建配置
    config := &DigestConfig{
        DailyEnabled:    true,
        DailyTime:       "09:00",
        FeishuEnabled:   false, // 不实际发送
        ObsidianEnabled: false, // 不实际写入
    }

    // 2. 测试generator
    generator := NewDigestGenerator(config)
    digests, err := generator.GenerateDailyDigest(time.Now())
    assert.NoError(t, err)
    assert.NotNil(t, digests)

    // 3. 验证数据结构
    // ...
}
```

**Step 2: 运行集成测试**

```bash
cd backend-go
go test ./internal/digest/... -v
```

**Step 3: 提交**

```bash
git add backend-go/internal/digest/integration_test.go
git commit -m "test(digest): add integration tests"
```

---

### Task 15: 更新文档

**Files:**
- Modify: `README.md`
- Create: `docs/digest-setup-guide.md`

**Step 1: 创建设置指南**

```markdown
# 日报周报功能使用指南

## 功能概述

日报周报功能可以自动生成每日/每周的新闻汇总，并支持飞书推送和Obsidian导出。

## 配置步骤

### 1. 基础设置

1. 进入应用，点击左侧"日报周报"菜单
2. 点击"设置"按钮
3. 配置日报/周报的生成时间

### 2. 飞书推送

1. 在飞书群组中添加自定义机器人
2. 复制Webhook URL
3. 在设置中粘贴URL并启用推送
4. 点击"测试推送"验证

### 3. Obsidian导出

1. 安装Obsidian并创建Vault
2. 在设置中填写Vault完整路径
3. 点击"测试写入"验证
4. 启用日报/周报导出

## 文件结构

导出到Obsidian的文件结构如下：

```
ObsidianVault/
├── Daily/
│   ├── AI技术/
│   │   └── 2026-03-04-日报.md
│   └── 前端开发/
│       └── 2026-03-04-日报.md
├── Weekly/
│   ├── AI技术/
│   │   └── 2026-W9-周报.md
│   └── 前端开发/
│       └── 2026-W9-周报.md
└── Feeds/
    ├── TechCrunch/
    │   └── 2026-03-04.md
    └── ...
```

## 常见问题

### 飞书推送失败怎么办？

1. 检查Webhook URL是否正确
2. 确认机器人是否被移除
3. 查看后端日志

### Obsidian写入失败怎么办？

1. 检查路径是否有写入权限
2. 确认路径是否存在
3. 尝试使用绝对路径
```

**Step 2: 更新主README**

在 `README.md` 中添加功能说明

**Step 3: 提交**

```bash
git add README.md docs/digest-setup-guide.md
git commit -m "docs(digest): add setup guide and update README"
```

---

## 验收标准

### 功能验收

- [ ] 每天9点自动生成日报
- [ ] 每周一9点自动生成周报
- [ ] 飞书推送成功（汇总通知 + 细节通知）
- [ ] Obsidian文件正确生成
- [ ] 前端配置界面正常工作
- [ ] 测试推送和测试写入功能正常

### 性能验收

- [ ] 日报生成时间 < 30秒
- [ ] 周报生成时间 < 60秒
- [ ] 飞书推送响应时间 < 5秒
- [ ] Obsidian写入时间 < 10秒

### 代码质量

- [ ] 所有测试通过
- [ ] 无Go编译错误
- [ ] 无TypeScript类型错误
- [ ] 代码提交规范

---

## 后续工作 (Phase 2)

1. 添加AI相似度匹配
2. 实现实体抽取
3. 构建知识图谱
4. 跨分类智能推荐

详见 `docs/plans/2026-03-04-ai-summary-enhancement-design.md`
