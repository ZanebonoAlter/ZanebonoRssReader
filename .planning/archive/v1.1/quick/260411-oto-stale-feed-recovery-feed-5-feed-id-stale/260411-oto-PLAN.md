---
phase: 260411-oto
plan: 01
type: execute
wave: 1
depends_on: []
files_modified: [backend-go/internal/jobs/auto_refresh.go]
autonomous: true
requirements: [REC-01]
must_haves:
  truths:
    - "Feeds stuck in refreshing state for > 5 minutes are reset to runnable state"
    - "Each stale feed's ID and duration is logged"
  artifacts:
    - path: "backend-go/internal/jobs/auto_refresh.go"
      provides: "Stale feed recovery with detailed logging"
  key_links:
    - from: "resetStaleRefreshingFeeds()"
      to: "log.Printf"
      pattern: "log\\.Printf.*stale.*feed"
---

<objective>
Fix stale feed recovery to log feed_id and stale duration instead of just the count.

Purpose: Operators need visibility into which feeds are stuck and how long they were stuck.
Output: Modified resetStaleRefreshingFeeds() with per-feed logging.
</objective>

<execution_context>
@$HOME/.config/opencode/get-shit-done/workflows/execute-plan.md
@$HOME/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@backend-go/internal/jobs/auto_refresh.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add per-feed logging in resetStaleRefreshingFeeds</name>
  <files>backend-go/internal/jobs/auto_refresh.go</files>
  <action>
Modify the `resetStaleRefreshingFeeds()` function (lines 403-416) to:
1. First QUERY feeds matching the stale criteria (refresh_status = 'refreshing' AND last_refresh_at < cutoff)
2. For each stale feed, log: feed_id and stale duration (now - last_refresh_at)
3. Then perform the batch UPDATE to reset their status

Current implementation does batch update without logging individual feeds.
New implementation should:
- Query `SELECT id, last_refresh_at FROM feeds WHERE refresh_status = 'refreshing' AND last_refresh_at < cutoff`
- For each result: `log.Printf("[STALE] Feed %d stuck for %.1f minutes, resetting", feed.ID, staleDuration.Minutes())`
- Then execute the batch reset update

Keep the existing summary log at the end with count.
  </action>
  <verify>
    <automated>go test ./internal/jobs -run TestResetStaleRefreshingFeeds -v 2>/dev/null || go build ./internal/jobs</automated>
  </verify>
  <done>resetStaleRefreshingFeeds() logs each stale feed_id and duration before resetting</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Internal | This is internal maintenance logic, no external input |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-260411-01 | N/A | Internal function | accept | No external input, purely internal state management |
</threat_model>

<verification>
- Code compiles: `go build ./internal/jobs`
- Function logs per-feed details when stale feeds detected
- Existing batch reset behavior preserved
</verification>

<success_criteria>
- resetStaleRefreshingFeeds() logs "[STALE] Feed X stuck for Y minutes" for each stale feed
- Batch reset still executed correctly
- Code compiles and tests pass
</success_criteria>

<output>
After completion, create `.planning/quick/260411-oto-stale-feed-recovery-feed-5-feed-id-stale/260411-oto-SUMMARY.md`
</output>