---
phase: 260411-oto
plan: 01
type: execute
wave: 1
depends_on: []
files_modified: [backend-go/internal/jobs/auto_refresh.go]
autonomous: true
requirements: [REC-01]
key_decisions:
  - "Query stale feeds before batch reset to enable per-feed logging"
  - "Use [STALE] prefix for log messages to distinguish from other logs"
tech_stack:
  added: []
  patterns: ["query-first-then-update pattern for visibility"]
key_files:
  created: []
  modified: ["backend-go/internal/jobs/auto_refresh.go"]
---

# Phase 260411-oto Plan 01: Stale Feed Recovery Logging Summary

## One-Liner

Added per-feed logging in `resetStaleRefreshingFeeds()` to log each feed's ID and stale duration before batch reset.

## Changes Made

### Task 1: Add per-feed logging in resetStaleRefreshingFeeds

Modified `resetStaleRefreshingFeeds()` function in `backend-go/internal/jobs/auto_refresh.go`:

**Before:** Batch update only, logging count at end.

**After:** 
1. Query stale feeds matching criteria first
2. Log each feed with `[STALE] Feed X stuck for Y minutes, resetting`
3. Then perform batch reset update
4. Keep existing summary log with count

**Implementation Details:**
- Added `staleFeeds []models.Feed` query before batch update
- Loop through each stale feed and log with `staleDuration.Minutes()` calculation
- Preserve existing batch update logic and count logging
- Early return on query error with log message

## Verification

- **Build:** `go build ./internal/jobs` - PASSED (no errors)
- **Impact analysis:** LOW risk, only one direct caller in same file

## Deviations from Plan

None - plan executed exactly as written.

## Threat Flags

None - internal maintenance function, no external input.

## Known Stubs

None - no stubs introduced.

## Self-Check

- [x] `backend-go/internal/jobs/auto_refresh.go` modified as specified
- [x] Commit `08dfd6b` exists in git history
- [x] Code compiles successfully

## Metrics

| Metric | Value |
|--------|-------|
| Duration | ~5 minutes |
| Tasks Completed | 1/1 |
| Files Modified | 1 |
| Commit Hash | 08dfd6b |

## Commit

```
08dfd6b fix(260411-oto): add per-feed logging in resetStaleRefreshingFeeds
```