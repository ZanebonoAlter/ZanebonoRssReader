---
gsd_state_version: 1.0
milestone: v1.1
milestone_name: milestone
status: executing
last_updated: "2026-04-11T05:50:00.000Z"
last_activity: 2026-04-11 -- Phase 02 context gathered
progress:
  total_phases: 6
  completed_phases: 0
  total_plans: 3
  completed_plans: 0
  percent: 0
---

# STATE: Milestone v1.1 业务漏洞修复

## Current Position

Phase: 02 (标签流程统一) — CONTEXT READY
Plan: 0 of N
Status: Context gathered for Phase 02
Last activity: 2026-04-11 -- Phase 02 context gathered

## Blocked

(None)

## Accumulated Context

### 代码审查发现

**定时任务问题:**

- auto_refresh.go:190-193 goroutine异步刷新，triggerAutoSummaryAfterRefreshes等待完成
- firecrawl.go:60-77 TriggerNow()直接调用runCrawlCycle，无结果返回
- staleRefreshingTimeout=5分钟重置，无后续处理

**标签流程问题:**

- firecrawl.go:238-248 异步enqueue，但文档说"直接调用RetagArticle"
- content_completion_service.go:198-205 同样异步enqueue
- articles/handler.go:216 手动API直接调用RetagArticle，绕过队列
- tag_queue.go:67-82 Start失败后无自动恢复

**状态一致性:**

- feeds/service.go:172-193 buildArticleFromEntry状态转换遗漏
- cleanupOldArticles依赖feed存在，feed删除后失效
- blocked文章无恢复机制

**API问题:**

- scheduler.ts:6-37 trigger用fetch而非apiClient
- api.ts:275-289 updateArticle不刷新unreadCount

**错误处理:**

- firecrawl.go缺少panic recovery
- preference_update.go缺少panic recovery
- digest/scheduler.go不记录执行状态

### 关键文件

| 文件 | 漏洞类别 |
|------|----------|
| auto_refresh.go | 并发、恢复 |
| firecrawl.go | 并发、错误、恢复、标签 |
| content_completion.go | 状态、错误、恢复 |
| content_completion_service.go | 标签、状态 |
| articles/handler.go | 标签 |
| tag_queue.go | 标签、恢复 |
| feeds/service.go | 状态 |
| scheduler.ts | API |
| api.ts | API |

---

*Updated: 2026-04-11*
