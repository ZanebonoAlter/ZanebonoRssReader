# Phase 1: 并发控制修复 - Context

**Gathered:** 2026-04-11
**Status:** Ready for planning

<domain>
## Phase Boundary

scheduler并发执行控制：确保触发操作不丢失任务、不重复执行，前端能感知执行结果。

具体范围：
- CONC-01: Auto-refresh scheduler在feed刷新完成后正确触发auto-summary
- CONC-02: Firecrawl scheduler TriggerNow()返回实际执行状态
- CONC-03: 所有scheduler TriggerNow()锁定失败时返回一致错误格式
- CONC-04: Auto-refresh异步刷新feed时每个goroutine独立panic recovery
- CONC-05: Digest scheduler reload时不丢失pending定时任务

</domain>

<decisions>
## Implementation Decisions

### Auto-refresh等待机制 (CONC-01)
- **D-01:** TriggerNow()采用异步+完成通知模式，立即返回"触发成功"，不阻塞等待feeds刷新完成
- **D-02:** 完成通知机制：通过WebSocket或新增状态API让前端感知feeds刷新何时完成
- **D-03:** triggerAutoSummaryAfterRefreshes保持现有行为：独立goroutine等待feeds完成后触发

### Firecrawl TriggerNow返回值 (CONC-02)
- **D-04:** TriggerNow()采用异步+batch查询模式，立即返回，包含batch_id
- **D-05:** 新增batch状态查询API，前端通过batch_id查询执行结果（completed/failed计数）
- **D-06:** 保持现有WebSocket firecrawl_progress推送作为实时进度补充

### TriggerNow格式一致性 (CONC-03)
- **D-07:** 统一必填字段：`accepted`, `started`, `reason`, `message`, `status_code`（失败时）
- **D-08:** 成功返回可选扩展字段：`effectful`, `summary`, `batch_id`等根据scheduler需要添加
- **D-09:** status_code统一使用http常量（http.StatusConflict等），不硬编码

### Panic Recovery (CONC-04)
- **D-10:** 现有实现已正确：refreshFeedAsync已有defer/recover，无需修改
- **D-11:** panic记录到feed的refresh_error字段，其他feeds继续刷新

### Digest Reload优雅停止 (CONC-05)
- **D-12:** 现有实现已正确：cron.Stop().Done()等待执行任务完成
- **D-13:** AddFunc重新添加时保持原有schedule时间，无需额外处理

### Agent's Discretion
- 完成通知WebSocket消息格式细节
- batch状态查询API具体字段设计
- 前端如何监听/调用这些机制

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` §CONC-01~05 — 并发控制需求详细定义

### Source Files (Affected)
- `backend-go/internal/jobs/auto_refresh.go` — Auto-refresh scheduler实现
- `backend-go/internal/jobs/firecrawl.go` — Firecrawl scheduler实现
- `backend-go/internal/jobs/handler.go` — Scheduler HTTP handler
- `backend-go/internal/domain/digest/scheduler.go` — Digest scheduler实现
- `backend-go/internal/jobs/auto_summary.go` — Auto-summary scheduler实现（参考TriggerNow格式）
- `backend-go/internal/jobs/content_completion.go` — Content completion scheduler实现（参考TriggerNow格式）
- `backend-go/internal/jobs/preference_update.go` — Preference update scheduler实现（参考TriggerNow格式）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `executionMutex.TryLock()` 模式：所有scheduler已使用，防止重复执行
- WebSocket Hub (`ws.GetHub()`)：Firecrawl已用于broadcastProgress，可复用
- SchedulerTask数据库表：auto_refresh已用于持久化状态，可复用

### Established Patterns
- TriggerNow()返回格式：`{accepted, started, reason, message, status_code}`
- Panic recovery pattern：defer/recover + log.Printf + 状态更新
- Cron schedule pattern：`cron.New()` + `AddFunc` + `Start()` + `Stop().Done()`

### Integration Points
- HTTP Handler: `handler.go` `TriggerScheduler()` 调用各scheduler TriggerNow()
- WebSocket: `ws.GetHub().BroadcastRaw()` 推送进度消息
- 前端API: `front/app/api/scheduler.ts` 调用trigger API

</code_context>

<specifics>
## Specific Ideas

- Auto-refresh完成通知：可参考Firecrawl的WebSocket broadcastProgress模式，在feeds刷新完成后推送消息
- Firecrawl batch查询：batchID已在runCrawlCycle中生成（line 163），可新增 `/api/schedulers/firecrawl/batch/:id` API查询该batch的状态
- TriggerNow格式统一：最小修改，只统一status_code使用http常量而非硬编码

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-并发控制修复*
*Context gathered: 2026-04-11*