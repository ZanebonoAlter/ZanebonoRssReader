---
phase: quick
plan: 01
subsystem: backend-jobs
tags: [scheduler, tag-merge, pgvector, embedding]
dependency_graph:
  requires: [topicanalysis.MergeTags, topicanalysis.DefaultThresholds, pgvector]
  provides: [AutoTagMergeScheduler, auto_tag_merge scheduler endpoint]
  affects: [runtime.go, handler.go, runtimeinfo/schedulers.go]
tech_stack:
  added: [pgvector cross-join SQL, cosine distance threshold]
  patterns: [AutoSummaryScheduler pattern]
key_files:
  created:
    - backend-go/internal/jobs/auto_tag_merge.go
  modified:
    - backend-go/internal/app/runtime.go
    - backend-go/internal/app/runtimeinfo/schedulers.go
    - backend-go/internal/jobs/handler.go
decisions:
  - Used pgvector SQL cross-join for efficient pair discovery instead of O(n²) Go loop
  - Hourly default interval (3600s), capped at 50 pairs per cycle
  - Distance threshold derived from DefaultThresholds.HighSimilarity (0.97 → distance < 0.03)
  - Tag with more articles always kept as target; smaller ID breaks ties
metrics:
  duration: ~5min
  completed: 2026-04-13
---

# Quick Task 260413-r4v: Auto Tag Merge Scheduler Summary

Auto-merge scheduler that periodically scans active tags via pgvector cross-join SQL and merges high-similarity pairs (>=0.97) using existing MergeTags function.

## Changes Made

### Task 1: Create AutoTagMergeScheduler (050a10d)
- Created `backend-go/internal/jobs/auto_tag_merge.go`
- Follows AutoSummaryScheduler pattern exactly: cron, mutex, GetStatus, TriggerNow, UpdateInterval, ResetStats
- `scanAndMergeTags()` uses efficient pgvector SQL cross-join to find similar pairs
- Only merges within same category, keeps tag with more articles as target
- Skips pairs where either tag was already merged in the current cycle
- RunSummary with MergeDetails provides full audit trail

### Task 2: Wire into runtime (245370e)
- Added `AutoTagMerge` field to Runtime struct
- StartRuntime: creates scheduler with 3600s interval, starts after BlockedArticleRecovery
- SetupGracefulShutdown: stops AutoTagMerge scheduler
- runtimeinfo: added `AutoTagMergeSchedulerInterface` variable
- handler: added `auto_tag_merge` entry to schedulerDescriptors
- Existing routes `/api/schedulers/auto_tag_merge/status` and `/api/schedulers/auto_tag_merge/trigger` work automatically

## Deviations from Plan

None — plan executed exactly as written.

## Verification

- `go build ./...` — compiles ✓
- `go vet ./internal/jobs/ ./internal/app/ ./internal/app/runtimeinfo/` — no issues ✓

## Self-Check: PASSED

- [x] `backend-go/internal/jobs/auto_tag_merge.go` exists
- [x] `backend-go/internal/app/runtime.go` modified with AutoTagMerge field
- [x] `backend-go/internal/app/runtimeinfo/schedulers.go` modified
- [x] `backend-go/internal/jobs/handler.go` modified
- [x] Commit 050a10d exists
- [x] Commit 245370e exists
