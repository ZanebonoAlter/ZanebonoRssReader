---
status: testing
phase: 02-watched-tags-homepage-feed
source: [02-01-SUMMARY.md, 02-02-SUMMARY.md]
started: 2026-04-15T12:00:00Z
updated: 2026-04-15T12:00:00Z
---

## Current Test

number: 1
name: Cold Start Smoke Test
expected: |
  Kill any running Go backend. Clear ephemeral state if needed. Start backend from backend-go/cmd/server/main.go. Server boots without errors. GET /api/topic-tags/watched returns empty array or valid list (not error). Frontend homepage loads without console errors.
awaiting: user response

## Tests

### 1. Cold Start Smoke Test
expected: Kill backend, restart from scratch. Server boots clean, GET /api/topic-tags/watched returns valid response, frontend homepage loads.
result: [pending]

### 2. Watch a Tag (Backend API)
expected: POST /api/topic-tags/:tag_id/watch returns success. Subsequent GET /api/topic-tags/watched includes the tag with is_watched=true and watched_at timestamp.
result: [pending]

### 3. Unwatch a Tag (Backend API)
expected: POST /api/topic-tags/:tag_id/unwatch returns success. Tag removed from GET /api/topic-tags/watched list.
result: [pending]

### 4. GetArticles with Watched Tags Filter
expected: GET /api/articles?watched_tag_ids=1,2 returns only articles tagged with those tags. Empty watched_tag_ids returns all articles (unchanged behavior).
result: [pending]

### 5. Relevance Sort
expected: GET /api/articles?watched_tag_ids=1,2&sort_by=relevance returns articles sorted by matched tag count descending. Articles matching both tags appear before articles matching only one.
result: [pending]

### 6. Abstract Tag Expansion
expected: Watch an abstract tag (has children in topic_tag_relations). GET /api/articles?watched_tag_ids=[abstract_id] returns articles tagged with ANY child tag. Relevance score weights abstract tag matches 2x.
result: [pending]

### 7. Heart Icon Toggle (Frontend)
expected: In TagHierarchy, click heart icon on a tag row. Icon immediately fills/unfills (optimistic UI). If API succeeds, state persists. If API fails, icon reverts and shows error toast.
result: [pending]

### 8. Sidebar Watched Tags Section
expected: Sidebar shows "关注标签" group between topic-graph button and categories. Contains "全部关注" item and individual watched tag items. Empty state shows guidance banner with link to /topics.
result: [pending]

### 9. FeedLayoutShell Watched Tag Filtering
expected: Click watched tag in sidebar. Feed shows only articles with that tag. Click "全部关注" shows articles matching ANY watched tag with relevance sort.
result: [pending]

### 10. Empty Watched Tags Guidance
expected: When no tags are watched, sidebar shows guidance banner encouraging user to visit /topics to watch tags. Homepage shows default timeline (all articles, no filter).
result: [pending]

## Summary

total: 10
passed: 0
issues: 0
pending: 10
skipped: 0
blocked: 0

## Gaps

[none yet]