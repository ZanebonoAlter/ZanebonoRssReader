# Phase 1: 并发控制修复 - Research

**Research Date:** 2026-04-11
**Phase:** 01-并发控制修复
**Researcher:** Claude (opus)

---

## Research Summary

**Question:** What do I need to know to PLAN Phase 1 well?

**Answer:** Phase 1 fixes concurrency control issues in Go backend schedulers. The codebase already has partial implementations - CONTEXT.md correctly identifies what works and what needs modification. The main work involves:
1. Adding completion notification mechanisms (WebSocket + batch status API)
2. Standardizing TriggerNow return format (http constants)
3. Verifying existing panic recovery and reload patterns work correctly

---

## Codebase Analysis

### TriggerNow Pattern Inventory

| Scheduler | Location | Current Format | status_code |
|-----------|----------|----------------|-------------|
| auto_refresh | auto_refresh.go:305-354 | `{accepted, started, reason, message, summary}` | `http.StatusConflict` (correct) |
| firecrawl | firecrawl.go:60-77 | `{accepted, started, message}` | `409` (hardcoded - needs fix) |
| auto_summary | auto_summary.go:704-746 | `{accepted, started, reason, message}` | `http.StatusConflict` (correct) |
| content_completion | content_completion.go:98-115 | `{accepted, started, message}` | `409` (hardcoded - needs fix) |
| preference_update | preference_update.go:127-149 | `{accepted, started, reason, message}` | `409` (hardcoded - needs fix) |

**Finding:** 3 schedulers use hardcoded `409` instead of `http.StatusConflict`. Handler.go:respondTriggerResult correctly extracts and uses status_code.

### Auto-Refresh Completion Mechanism

**Current implementation (auto_refresh.go:248-264):**
```go
func (s *AutoRefreshScheduler) triggerAutoSummaryAfterRefreshes(wg *sync.WaitGroup) {
    wg.Wait()
    // ... triggers auto-summary
}
```

**Analysis:**
- `wg.Wait()` correctly blocks until all goroutines complete (line 249)
- Called in separate goroutine (line 198): `go s.triggerAutoSummaryAfterRefreshes(&refreshWG)`
- Auto-summary TriggerNow() is async (line 728-738 in auto_summary.go)
- No completion notification to frontend

**CONTEXT.md Decision D-02:** Add completion notification via WebSocket or status API

**Implementation approach:**
1. Add WebSocket broadcast after feeds complete (reuse firecrawl pattern)
2. New message type: `auto_refresh_complete`
3. Include: triggered_count, stale_reset_count, duration

### Firecrawl Batch Status Query

**Current implementation (firecrawl.go:163):**
```go
batchID := time.Now().Format("20060102150405")
```

**Analysis:**
- batchID is generated but not stored persistently
- No batch status tracking table exists
- WebSocket broadcasts progress in real-time
- TriggerNow() returns immediately without batch_id

**CONTEXT.md Decision D-04/D-05:** Async + batch status API

**Implementation approach:**
1. Return batch_id in TriggerNow response
2. Track batch status in memory (completed/failed counts)
3. Add GET `/api/schedulers/firecrawl/batch/:id` endpoint
4. Or: reuse WebSocket progress message (already broadcasts final status)

**Recommendation:** WebSocket approach is simpler - frontend already receives `firecrawl_progress` with status="completed". Just need to ensure TriggerNow returns batch_id so frontend knows which batch to watch.

### Panic Recovery Verification

**auto_refresh.go:225-237:**
```go
func (s *AutoRefreshScheduler) refreshFeedAsync(ctx context.Context, feedID uint) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("[PANIC] refreshFeedAsync for feed %d: %v", feedID, r)
            s.resetFeedStatus(feedID, fmt.Sprintf("panic: %v", r))
        }
    }()
    // ...
}
```

**Verdict:** Already implemented correctly (D-10 confirmed)

**firecrawl.go runCrawlCycle:** NO panic recovery (missing!)
**auto_summary.go checkAndGenerateSummaries:** Has recovery (lines 256-263)

**Note:** Panic recovery for firecrawl/preference_update is Phase 5 scope (ERR-01, ERR-02), not Phase 1.

### Digest Scheduler Reload

**digest/scheduler.go:72-76:**
```go
func (s *DigestScheduler) reloadLocked() error {
    if s.cron != nil && s.isRunning {
        ctx := s.cron.Stop()
        <-ctx.Done()  // Wait for running jobs to complete
    }
    s.cron = cron.New()
    // ... re-add jobs with AddFunc
}
```

**Verdict:** Already implemented correctly (D-12 confirmed)
- `cron.Stop()` returns context
- `<-ctx.Done()` waits for executing jobs
- New cron instance created, jobs re-added with same schedule

---

## Integration Points

### WebSocket Infrastructure

**hub.go provides:**
- `GetHub()` singleton
- `BroadcastRaw(data []byte)` - firecrawl uses this
- `FirecrawlProgressMessage` type already defined

**Pattern for auto-refresh completion:**
```go
msg := ws.AutoRefreshCompleteMessage{
    Type:           "auto_refresh_complete",
    TriggeredFeeds: summary.TriggeredFeeds,
    StaleResetFeeds: summary.StaleResetFeeds,
    Duration:       duration,
}
data, _ := json.Marshal(msg)
ws.GetHub().BroadcastRaw(data)
```

### Handler Response Pattern

**handler.go:178-194:**
```go
func respondTriggerResult(c *gin.Context, name string, result map[string]interface{}) {
    statusCode := http.StatusOK
    if rawCode, ok := result["status_code"].(int); ok {
        statusCode = rawCode
    }
    delete(result, "status_code")  // Remove from response body
    result["name"] = name
    // ...
}
```

**Key insight:** status_code is extracted for HTTP status, then deleted from response. All schedulers must include it for proper HTTP response codes.

---

## Validation Architecture

### Nyquist Dimensions for Phase 1

| Dimension | Requirement | Implementation |
|-----------|-------------|----------------|
| D1: Input Validation | HTTP params validated | handler.go validates scheduler name exists |
| D2: Output Validation | TriggerNow format consistent | Standardize across all schedulers |
| D3: State Machine | executionMutex TryLock prevents double-run | Already implemented |
| D4: Error Propagation | panic recovered, logged, status updated | auto_refresh has it; firecrawl missing (Phase 5) |
| D5: Concurrency | goroutine wg.Wait for completion | Already implemented |
| D6: Idempotency | TryLock prevents duplicate triggers | Already implemented |
| D7: Observability | SchedulerTask table records execution | Already implemented |
| D8: Testability | WebSocket/batch status testable | Need test endpoints |

---

## Technical Recommendations

### For CONC-01 (Auto-refresh completion notification)

**Option A (WebSocket):** Simplest - reuse existing infrastructure
- Add `AutoRefreshCompleteMessage` type in ws/hub.go
- Broadcast after `triggerAutoSummaryAfterRefreshes` completes
- Frontend listens for `auto_refresh_complete` message

**Option B (Status API):** More explicit polling
- Add `GetLastRunSummary()` method
- Frontend polls until `is_executing=false`

**Recommendation:** WebSocket (D-02 said "WebSocket or status API") - simpler, already has infrastructure.

### For CONC-02 (Firecrawl batch status)

**Current WebSocket flow:**
1. TriggerNow() returns immediately
2. runCrawlCycle broadcasts `firecrawl_progress` with batchID
3. Frontend receives real-time updates
4. Final message: status="completed"

**Modification needed:**
- TriggerNow() should return batch_id so frontend knows which batch to watch
- No new API needed - WebSocket already provides status

### For CONC-03 (TriggerNow format consistency)

**Changes needed:**
- firecrawl.go:67: `409` → `http.StatusConflict`
- content_completion.go:106: `409` → `http.StatusConflict`
- preference_update.go:137: `409` → `http.StatusConflict`

**No changes:**
- auto_refresh.go:312 - already uses `http.StatusConflict`
- auto_summary.go:711 - already uses `http.StatusConflict`

### For CONC-04/CONC-05

**Verified:** Already implemented correctly per CONTEXT.md D-10/D-12

---

## Files Modified Summary

| File | Change Type | Requirements Addressed |
|------|-------------|------------------------|
| ws/hub.go | Add message type | CONC-01 |
| auto_refresh.go | Add completion broadcast | CONC-01 |
| firecrawl.go | Return batch_id, fix status_code | CONC-02, CONC-03 |
| content_completion.go | Fix status_code | CONC-03 |
| preference_update.go | Fix status_code | CONC-03 |

---

## Dependencies

**No external dependencies needed.** All changes use existing:
- WebSocket Hub infrastructure
- HTTP constants from net/http
- sync.WaitGroup pattern
- cron library patterns

---

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| WebSocket message format mismatch | Define explicit struct, match existing pattern |
| Frontend not listening for new message type | Document message format, frontend change tracked separately |
| Hardcoded status_code typo | Use `http.StatusConflict` constant, not manual number |

---

## Security Threat Model

**Trust Boundaries:**
- Client → API (untrusted input via HTTP params)
- API → Scheduler (trusted internal call)

**STRIDE Analysis:**

| Threat | Category | Disposition | Mitigation |
|--------|----------|-------------|------------|
| Malicious scheduler name injection | Tampering | mitigate | handler.go validates name exists before calling TriggerNow |
| WebSocket message spoofing | Spoofing | accept | Internal broadcast only, no external input |
| Concurrent trigger race condition | Tampering | mitigate | executionMutex.TryLock prevents double execution |

---

*Research complete. Ready for planning.*