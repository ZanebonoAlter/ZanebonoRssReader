---
phase: 01
slug: 并发控制修复
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-11
---

# Phase 01 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (testing package) |
| **Config file** | none — existing Go test infrastructure |
| **Quick run command** | `go test ./internal/jobs -v -run TestTrigger` |
| **Full suite command** | `go test ./internal/jobs ./internal/domain/digest -v` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/jobs -v -run TestTrigger`
- **After every plan wave:** Run `go test ./internal/jobs ./internal/domain/digest -v`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | CONC-01 | — | WebSocket broadcast only internal | unit | `go test ./internal/jobs -run TestAutoRefreshComplete -v` | ❌ W0 | ⬜ pending |
| 01-01-02 | 01 | 1 | CONC-01 | — | N/A | unit | `go test ./internal/platform/ws -run TestAutoRefreshMessage -v` | ❌ W0 | ⬜ pending |
| 01-02-01 | 02 | 1 | CONC-02 | — | N/A | unit | `go test ./internal/jobs -run TestFirecrawlTriggerNowBatchID -v` | ❌ W0 | ⬜ pending |
| 01-03-01 | 03 | 1 | CONC-03 | T-01-01 | status_code via http constant, not hardcoded | unit | `go test ./internal/jobs -run TestTriggerNowStatusCode -v` | ❌ W0 | ⬜ pending |
| 01-04-01 | 04 | 1 | CONC-04 | — | Panic recovered, feed continues | unit | Existing test passes | ✅ | ⬜ pending |
| 01-04-02 | 04 | 1 | CONC-05 | — | Cron.Stop().Done() waits | unit | Existing test passes | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `backend-go/internal/jobs/auto_refresh_test.go` — TestAutoRefreshComplete, TestTriggerNowStatusCode stubs
- [ ] `backend-go/internal/jobs/firecrawl_test.go` — TestFirecrawlTriggerNowBatchID stub
- [ ] `backend-go/internal/platform/ws/hub_test.go` — TestAutoRefreshMessage stub (if not exists)

*Note: CONC-04/CONC-05 are verification-only tasks (existing code confirmed correct).*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| WebSocket frontend receives completion | CONC-01 | Requires frontend integration | Open browser DevTools → Network → WS tab → Trigger auto-refresh → Verify `auto_refresh_complete` message received |
| Batch ID propagation to frontend | CONC-02 | Requires frontend integration | Trigger firecrawl → Verify response contains batch_id → Verify WebSocket messages have matching batch_id |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending