---
phase: 01-并发控制修复
verified: 2026-04-11T12:29:39+08:00
status: gaps_found
score: 6/8 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Frontend receives notification when all feeds refresh complete"
    status: failed
    reason: "后端已广播 auto_refresh_complete，但前端代码中没有任何该事件的 WebSocket 消费方，因此用户侧无法实际收到完成通知。"
    artifacts:
      - path: "backend-go/internal/jobs/auto_refresh.go"
        issue: "已发送 auto_refresh_complete 广播，但仅完成后端侧接线。"
      - path: "front/app/features/summaries/composables/useSummaryWebSocket.ts"
        issue: "仅解析 summary progress 消息，没有 auto_refresh_complete 处理逻辑。"
      - path: "front/app/components/dialog/GlobalSettingsDialog.vue"
        issue: "调度器面板只做轮询与触发反馈，没有订阅 auto_refresh_complete。"
    missing:
      - "新增前端 WebSocket 监听并消费 auto_refresh_complete 事件"
      - "在调度器 UI 中展示 auto-refresh 完成反馈"
  - truth: "Firecrawl batch_id can be used to track same execution in progress channel"
    status: partial
    reason: "后端已返回 batch_id 并在 firecrawl_progress 广播中复用，但前端没有 firecrawl_progress 消费逻辑，SchedulerTriggerResult 类型也未暴露 batch_id。"
    artifacts:
      - path: "backend-go/internal/jobs/firecrawl.go"
        issue: "batch_id 在后端链路中已贯通，但没有前端消费者。"
      - path: "front/app/types/scheduler.ts"
        issue: "SchedulerTriggerResult 未声明 batch_id 字段。"
      - path: "front/app/components/dialog/GlobalSettingsDialog.vue"
        issue: "保存了 trigger 返回值，但未使用 batch_id 建立 Firecrawl 进度跟踪。"
    missing:
      - "扩展前端调度器触发结果类型以包含 batch_id"
      - "新增 firecrawl_progress 前端消费与按 batch_id 关联展示"
---

# Phase 01: 并发控制修复 Verification Report

**Phase Goal:** scheduler并发执行不丢失任务、不重复执行
**Verified:** 2026-04-11T12:29:39+08:00
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Auto-refresh waits for all feed refresh goroutines before auto-summary | ✓ VERIFIED | `backend-go/internal/jobs/auto_refresh.go:177-199,249-280` 使用 `sync.WaitGroup`，`wg.Wait()` 后才调用 `triggerable.TriggerNow()`；`go test ./internal/jobs -run "...TestAutoRefresh..." -v` 通过。 |
| 2 | Frontend receives notification when all feeds refresh complete | ✗ FAILED | 后端已在 `auto_refresh.go:252-265` 广播 `auto_refresh_complete`，但 `front/app` 下 `grep "auto_refresh_complete"` 无匹配，前端无消费代码。 |
| 3 | Firecrawl trigger returns clear execution result for success/failure/already_running | ✓ VERIFIED | `backend-go/internal/jobs/firecrawl.go:61-80` 返回 accepted/started/message；锁冲突返回 `already_running`；`front/app/components/dialog/GlobalSettingsDialog.vue:350-388,437-438` 会展示成功/失败/已在运行反馈。 |
| 4 | Firecrawl batch_id can be used to track same execution in progress channel | ✗ FAILED | `firecrawl.go:72-80,168,281` 已复用同一 `batchID`，但 `front/app` 下无 `firecrawl_progress` 消费；`front/app/types/scheduler.ts:76-84` 也无 `batch_id`。 |
| 5 | Repeated scheduler triggers do not start duplicate runs and report already_running consistently | ✓ VERIFIED | `auto_refresh.go:321-345`、`firecrawl.go:61-69`、`content_completion.go:99-107`、`preference_update.go:128-138` 均有拒绝重复执行逻辑；`TestTriggerNowStatusCode` 与 `TestDigestSchedulerSkipsOverlappingDailyGeneration` 通过。 |
| 6 | Auto-refresh per-feed goroutine panic is isolated and recorded | ✓ VERIFIED | `auto_refresh.go:189-194` 为每个 feed 单独起 goroutine；`refreshFeedAsync:226-238` 含独立 `defer recover`，并写入 `resetFeedStatus(feedID, fmt.Sprintf("panic: %v", r))`。 |
| 7 | Digest scheduler reload waits for running jobs and preserves schedules | ✓ VERIFIED | `backend-go/internal/domain/digest/scheduler.go:72-118` 在 reload 时先 `ctx := s.cron.Stop(); <-ctx.Done()`，再 `cron.New()` + `AddFunc(...)` 重新注册；`handler.go:792-801` 实际调用 `Reload()`。 |
| 8 | TriggerNow lock-failure responses use consistent structure and http.StatusConflict | ✓ VERIFIED | `firecrawl.go:63-69`、`content_completion.go:101-107`、`preference_update.go:132-138` 均返回 `accepted/started/reason/message/status_code`；`handler.go:178-194` 提取 `status_code`；`TestTriggerNowStatusCode` 通过。 |

**Score:** 6/8 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `backend-go/internal/jobs/auto_refresh.go` | 等待全部 feed 完成、广播完成消息、再触发 auto-summary | ✓ VERIFIED | `runRefreshCycle` + `triggerAutoSummaryAfterRefreshes` 已接通，测试通过。 |
| `backend-go/internal/platform/ws/hub.go` | 定义 `AutoRefreshCompleteMessage` | ✓ VERIFIED | `hub.go:84-91` 定义了显式消息结构。 |
| `backend-go/internal/jobs/firecrawl.go` | `TriggerNow` 返回 `batch_id`，并传给 `runCrawlCycle` | ✓ VERIFIED | `firecrawl.go:72-80,130-168,281`。 |
| `backend-go/internal/jobs/content_completion.go` | 锁冲突响应使用统一格式 | ✓ VERIFIED | `content_completion.go:99-107`。 |
| `backend-go/internal/jobs/preference_update.go` | 锁冲突响应使用统一格式 | ✓ VERIFIED | `preference_update.go:128-138`。 |
| `front/app/features/summaries/composables/useSummaryWebSocket.ts` | 前端消费 auto-refresh / firecrawl 相关进度事件 | ✗ ORPHANED | 仅处理 summary `progress` 消息，没有 `auto_refresh_complete` / `firecrawl_progress`。 |
| `front/app/types/scheduler.ts` | 调度器触发结果能表达 batch_id | ✗ MISSING | `SchedulerTriggerResult` 未包含 `batch_id`。 |
| `front/app/components/dialog/GlobalSettingsDialog.vue` | 将后端 trigger/WS 结果真正呈现给用户 | ⚠ PARTIAL | 能显示 accepted/error/already_running，但不能显示 auto-refresh 完成通知，也不能跟踪 firecrawl batch。 |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `handler.go` | `TriggerNow()` | `respondTriggerResult` 提取 `status_code` | ✓ WIRED | `backend-go/internal/jobs/handler.go:156-194`。 |
| `firecrawl.go TriggerNow()` | `runCrawlCycle()` | `batchID` 生成与传参 | ✓ WIRED | `firecrawl.go:72-79,130-168`；相关测试通过。 |
| `auto_refresh.go triggerAutoSummaryAfterRefreshes()` | `ws.GetHub().BroadcastRaw()` | 完成消息广播 | ✓ WIRED | `auto_refresh.go:252-265`；`TestAutoRefreshCompleteBroadcastSource` 通过。 |
| `auto_refresh_complete` WebSocket 事件 | 前端 UI | WebSocket 消费 | ✗ NOT_WIRED | `front/app` 内无 `auto_refresh_complete` 匹配。 |
| `firecrawl TriggerNow batch_id` | 前端 Firecrawl 跟踪 | `firecrawl_progress` / `batch_id` | ✗ NOT_WIRED | `front/app` 内无 `firecrawl_progress` 匹配，`SchedulerTriggerResult` 也无 `batch_id`。 |
| `digest config update` | `DigestScheduler.Reload()` | `Reload -> reloadLocked -> Stop().Done()` | ✓ WIRED | `backend-go/internal/domain/digest/handler.go:792-801` 与 `scheduler.go:72-118`。 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `backend-go/internal/jobs/auto_refresh.go` | `summary.TriggeredFeeds` / `summary.StaleResetFeeds` / `duration` | `runRefreshCycle()` 累计结果 + `time.Since(startTime)` | Yes | ✓ FLOWING（但前端未消费） |
| `backend-go/internal/jobs/firecrawl.go` | `batchID` | `TriggerNow()` 中 `time.Now().Format(...)`，同值传入 `runCrawlCycle(batchID)` | Yes | ✓ FLOWING（但前端未消费） |
| `backend-go/internal/domain/digest/scheduler.go` | `ctx.Done()` + 重建 cron entries | `cron.Stop()` 返回 context，`AddFunc(...)` 重建任务 | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| TriggerNow 状态码与 auto-refresh 完成链路 | `go test ./internal/jobs -run "TestTriggerNowStatusCode|TestFirecrawlTriggerNowBatchID|TestFirecrawlRunCrawlCycleUsesInjectedBatchID|TestAutoRefreshCompleteMessageJSON|TestAutoRefreshCompleteBroadcastSource|TestAutoRefreshTriggerNowUpdatesSchedulerTaskAndFeedState|TestAutoRefreshTriggerNowRunsAutoSummaryAfterTriggeredRefreshesFinish" -v` | PASS | ✓ PASS |
| Digest reload / overlap 基础行为 | `go test ./internal/domain/digest -run "TestDigestScheduler.*" -v` | PASS | ✓ PASS |
| 相关包可编译 | `go build ./internal/jobs ./internal/platform/ws ./internal/domain/digest` | PASS | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| CONC-01 | `01-03-PLAN.md` | Auto-refresh scheduler在feed刷新完成后正确等待所有goroutine结束再触发auto-summary | ✓ SATISFIED | `auto_refresh.go:177-199,249-280` + `TestAutoRefreshTriggerNowRunsAutoSummaryAfterTriggeredRefreshesFinish`。 |
| CONC-02 | `01-02-PLAN.md` | Firecrawl scheduler的TriggerNow()返回实际执行状态（成功/失败/已运行），前端可感知结果 | ✓ SATISFIED | 后端 `firecrawl.go:61-80` 返回明确结果；前端 `GlobalSettingsDialog.vue:350-388` 会展示 success/error/already_running。 |
| CONC-03 | `01-01-PLAN.md` | 所有scheduler的TriggerNow()方法在锁定失败时返回一致的错误格式 | ✓ SATISFIED | `firecrawl.go`、`content_completion.go`、`preference_update.go` + `handler.go:178-194` + `TestTriggerNowStatusCode`。 |
| CONC-04 | —（ORPHANED） | Auto-refresh异步刷新feed时，每个goroutine独立的panic recovery，不影响其他feed刷新 | ✓ SATISFIED / ORPHANED | `auto_refresh.go:189-194,226-238`。该 requirement 出现在 `REQUIREMENTS.md` Phase 1，但未被任何 PLAN frontmatter 声明。 |
| CONC-05 | —（ORPHANED） | Digest scheduler reload时不丢失pending定时任务（优雅停止再启动） | ✓ SATISFIED / ORPHANED | `digest/scheduler.go:72-118` 先等待 `ctx.Done()` 再重建 cron；`handler.go:792-801` 调用 `Reload()`。该 requirement 未被任何 PLAN frontmatter 声明。 |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| — | — | 未在本 phase 相关后端文件中发现 TODO/FIXME/placeholder stub 模式 | ℹ️ Info | 无阻断性代码味道证据 |

### Gaps Summary

后端并发控制主体基本到位：`TryLock` 防重入、auto-refresh 的 `WaitGroup` 等待、每 feed 独立 `recover`、digest reload 的 `Stop().Done()` 都存在，并且相关 Go 测试通过。

但本 phase 仍未达成“代码库层面的完整 must-have”——两条关键链路停在后端：

1. **auto_refresh_complete 只有后端广播，没有前端接收方**，因此“前端收到完成通知”并未真正成立。
2. **firecrawl 的 batch_id 只在后端贯通，没有前端类型/状态/UI 消费**，因此“前端可按 batch_id 跟踪同一执行”未成立。

结论：**并发控制核心后端实现已完成，但用户侧通知/跟踪接线未完成，Phase 01 不能判定为 passed。**

---

_Verified: 2026-04-11T12:29:39+08:00_
_Verifier: the agent (gsd-verifier)_
