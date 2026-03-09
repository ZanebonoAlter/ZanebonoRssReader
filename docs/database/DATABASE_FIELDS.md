# 数据库字段说明文档

本文档详细说明了 RSS Reader 项目中所有数据库表的字段用途、数据流向和工作流程。

---

## 核心表结构

### 1. Articles 表（文章表）

存储 RSS 文章的核心数据。

#### 内容相关字段

| 字段名 | 类型 | 用途 | 来源 | 格式 |
|--------|------|------|------|------|
| `content` | TEXT | RSS 原始内容（HTML 片段） | RSS Feed 解析 | HTML |
| `firecrawl_content` | TEXT | Firecrawl 抓取的完整网页内容 | Firecrawl Scheduler | Markdown |
| `ai_content_summary` | TEXT | AI 生成的优化总结内容 | AI Summary Scheduler | Markdown |
| `full_content` | TEXT | （保留字段，向后兼容） | - | - |

#### 状态字段

| 字段名 | 类型 | 用途 | 可选值 |
|--------|------|------|--------|
| `content_status` | VARCHAR(20) | AI 总结状态 | `incomplete` / `pending` / `complete` / `failed` |
| `firecrawl_status` | VARCHAR(20) | Firecrawl 抓取状态 | `pending` / `processing` / `completed` / `failed` |

#### 其他字段

| 字段名 | 用途 |
|--------|------|
| `completion_attempts` | AI 总结重试次数 |
| `completion_error` | AI 总结错误信息 |
| `firecrawl_error` | Firecrawl 抓取错误信息 |
| `firecrawl_crawled_at` | Firecrawl 抓取时间 |

---

### 2. Feeds 表（订阅源表）

存储 RSS 订阅源配置。

#### 功能开关字段

| 字段名 | 类型 | 用途 | 说明 |
|--------|------|------|------|
| `firecrawl_enabled` | BOOLEAN | 是否启用 Firecrawl 抓取完整内容 | 需要全局配置 Firecrawl API |
| `content_completion_enabled` | BOOLEAN | 是否启用 AI 内容总结 | **命名说明**：虽然名字是"内容补全"，但实际功能是"AI 内容总结" |
| `max_completion_retries` | INTEGER | AI 总结最大重试次数 | 默认 3 次 |

**重要说明**：
- `content_completion_enabled` 的名称源于历史原因，容易误解为"补全缺失内容"
- 实际功能：对 Firecrawl 抓取的完整内容进行 AI 优化总结
- 建议理解方式：`content_completion_enabled` = `ai_summary_enabled`（文章级别）

---

### 3. scheduler_tasks 表（调度任务表）

存储定时任务的状态信息。

| 任务名 | 描述 | 执行间隔 |
|--------|------|----------|
| `auto_refresh` | 自动刷新 RSS 订阅源 | 60 秒 |
| `auto_summary` | 生成分类级别的 AI 总结 | 3600 秒（1小时）|
| `ai_summary` | AI 智能总结文章内容（基于 Firecrawl）| 3600 秒（1小时）|

**命名变更**：
- 原名：`content_completion`
- 新名：`ai_summary`
- 变更原因：更准确反映功能实际用途

---

### 4. ai_summary_queue 表（AI 总结队列表）

存储待处理的 AI 总结任务（预留，当前未使用）。

| 字段名 | 用途 |
|--------|------|
| `article_id` | 待处理的文章 ID |
| `status` | 任务状态（`pending` / `processing` / `completed` / `failed`）|
| `retry_count` | 重试次数 |
| `error_message` | 错误信息 |

---

## 工作流程

### 完整的内容处理流程

```
┌─────────────────────────────────────────────────────────────┐
│  1. Feed Refresh Scheduler（每 60 秒）                      │
│     ↓                                                        │
│     解析 RSS Feed                                            │
│     ↓                                                        │
│     创建新文章                                               │
│     - content = RSS 原始内容                                 │
│     - firecrawl_status = 'pending'                          │
│     - content_status = 'incomplete'（如果 feed 启用）       │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│  2. Firecrawl Scheduler（每 5 分钟）                        │
│     ↓                                                        │
│     查询条件：                                               │
│     - firecrawl_status = 'pending'                          │
│     - feed.firecrawl_enabled = true                         │
│     ↓                                                        │
│     抓取完整网页内容                                         │
│     ↓                                                        │
│     更新文章：                                               │
│     - firecrawl_status = 'completed'                        │
│     - firecrawl_content = 完整内容（Markdown）              │
│     - content_status = 'incomplete' ← 标记需要 AI 总结     │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│  3. AI Summary Scheduler（每 60 分钟）                      │
│     ↓                                                        │
│     查询条件：                                               │
│     - firecrawl_status = 'completed'                        │
│     - content_status = 'incomplete'                         │
│     - feed.content_completion_enabled = true                │
│     ↓                                                        │
│     读取 firecrawl_content                                   │
│     ↓                                                        │
│     调用 AI 生成优化总结                                     │
│     ↓                                                        │
│     更新文章：                                               │
│     - ai_content_summary = AI 总结（Markdown）              │
│     - content_status = 'complete'                           │
└─────────────────────────────────────────────────────────────┘
```

### 状态流转图

#### Firecrawl 状态流转

```
pending → processing → completed
                     ↘ failed
```

- `pending`：初始状态，等待抓取
- `processing`：正在抓取（中间状态，通常很短）
- `completed`：抓取成功
- `failed`：抓取失败

#### Content Status 状态流转

```
incomplete → pending → complete
                     ↘ failed
```

- `incomplete`：需要 AI 总结（由 Firecrawl 完成后设置）
- `pending`：正在生成 AI 总结（中间状态）
- `complete`：AI 总结完成
- `failed`：AI 总结失败（超过重试次数）

---

## 字段用途说明

### 三个内容字段的区别

#### `content`（RSS 原始内容）

- **来源**：RSS Feed 解析
- **格式**：HTML 片段
- **特点**：
  - 可能不完整（部分 RSS 只提供摘要）
  - 可能包含 HTML 标签
  - 长度不确定
- **用途**：作为基础内容展示

#### `firecrawl_content`（完整网页内容）

- **来源**：Firecrawl 抓取
- **格式**：Markdown
- **特点**：
  - 完整的网页内容
  - 保留图片、链接、布局
  - 过滤了广告和导航栏
- **用途**：作为 AI 总结的输入源

#### `ai_content_summary`（AI 优化总结）

- **来源**：AI 生成
- **格式**：Markdown
- **特点**：
  - 保留核心内容和重要图片
  - 移除冗余内容
  - 重新组织结构，更易读
- **用途**：前端默认展示的内容

---

## 前端显示逻辑

### ArticleContent.vue 组件

#### 默认显示优先级

```
ai_content_summary（AI 总结）
    ↓ 如果为空
判断 feed.firecrawl_enabled
    ↓
  - 未开启：显示提示"未开启内容总结功能"
  - 已开启：显示提示"正在生成总结..."
```

#### 内容切换选项

用户可以手动切换显示：
1. **总结内容**（默认）：`ai_content_summary`
2. **原始内容**：`content`

**注意**：`firecrawl_content` 不对用户展示，仅作为 AI 总结的输入源。

---

## 配置要求

### Firecrawl 功能

1. 全局配置（`ai_settings` 表）：
   - `summary_config.firecrawl.enabled` = true
   - `summary_config.firecrawl.api_url` 设置正确
   - `summary_config.firecrawl.api_key` 设置正确

2. Feed 级别配置：
   - `feed.firecrawl_enabled` = true

### AI 总结功能

1. 全局配置（`ai_settings` 表）：
   - `summary_config.base_url` 设置正确
   - `summary_config.api_key` 设置正确
   - `summary_config.model` 设置正确

2. Feed 级别配置：
   - `feed.content_completion_enabled` = true

**依赖关系**：
- AI 总结功能依赖 Firecrawl 先抓取完整内容
- 如果 Firecrawl 失败，AI 总结会被跳过

---

## 错误处理

### Firecrawl 失败

- `firecrawl_status` = 'failed'
- `firecrawl_error` 记录错误信息
- 不会触发 AI 总结

### AI 总结失败

- `content_status` = 'failed'
- `completion_error` 记录错误信息
- 会自动重试（最多 `max_completion_retries` 次）

---

## 性能优化

### 批处理

- Firecrawl：每次最多处理 50 篇文章
- AI Summary：每次最多处理 50 篇文章

### 并发控制

- Firecrawl：并发数 3（可配置）
- AI Summary：单线程处理（避免 AI API 并发限制）

---

## 数据迁移说明

### 从旧版本迁移

如果数据库中已有 `content_completion` 任务：

```sql
-- 更新任务名称和描述
UPDATE scheduler_tasks 
SET name = 'ai_summary', 
    description = 'AI 智能总结文章内容（基于 Firecrawl 抓取的完整内容）' 
WHERE name = 'content_completion';
```

### 兼容性

- `full_content` 字段保留，但不再使用
- 旧数据不受影响
- 新流程自动处理新文章

---

## 常见问题

### Q: 为什么 Firecrawl 抓取的内容不直接展示给用户？

A: Firecrawl 抓取的完整内容可能：
- 包含大量冗余信息
- 结构不够清晰
- 不适合直接阅读

AI 总结会优化这些内容，保留精华部分。

### Q: 如果 AI 服务不可用怎么办？

A: 系统会：
1. 记录错误信息
2. 自动重试（最多 3 次）
3. 最终标记为 `failed`
4. 用户仍可查看原始 `content`

### Q: 能否只启用 Firecrawl 而不启用 AI 总结？

A: 可以。设置 `feed.content_completion_enabled = false`。

此时：
- 会抓取完整内容（`firecrawl_content`）
- 不会生成 AI 总结（`ai_content_summary` 为空）
- 前端显示原始内容（`content`）

### Q: 为什么 content_status 默认值是 'incomplete'？

A: 这是历史遗留问题。现在：
- 新文章创建时不设置 `content_status`
- Firecrawl 完成后设置为 `'incomplete'`
- 表示"需要 AI 总结"

---

## 更新日志

### 2026-03-05

**重大变更**：
1. 将 `content_completion` 任务重命名为 `ai_summary`
2. 明确了 `content_completion_enabled` 的实际用途（AI 总结）
3. 修改了 Content Completion Service 查询逻辑
4. Firecrawl 完成后自动设置 `content_status = 'incomplete'`
5. 创建了本文档

**向后兼容**：
- 保留所有现有字段
- 旧数据不受影响
- 新流程自动处理

---

## 相关文档

- `docs/architecture/data-flow.md` - 详细工作流程说明
- `docs/history/lessons-learned.md` - 开发经验总结
- `AGENTS.md` - 项目开发指南
