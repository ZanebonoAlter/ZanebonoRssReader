---
phase: 06-tag-merge-ui
verified: 2026-04-13T15:30:00Z
status: gaps_found
score: 8/14 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Frontend can call preview API and receive typed candidate pairs"
    status: failed
    reason: "TypeScript types use camelCase but backend sends snake_case. apiClient does no key transformation. Runtime data will have source_tag_id/source_label/etc. but component reads sourceTagId/sourceLabel/etc. — all undefined."
    artifacts:
      - path: "front/app/types/tagMerge.ts"
        issue: "Types defined in camelCase (sourceTagId, sourceLabel) but backend JSON uses snake_case (source_tag_id, source_label)"
      - path: "front/app/api/tagMergePreview.ts"
        issue: "No mapping layer between snake_case response and camelCase types; raw apiClient.get passes data through unchanged"
      - path: "front/app/features/topic-graph/components/TagMergePreview.vue"
        issue: "Template accesses candidate.sourceLabel, candidate.similarity, etc. — single-word fields work but all multi-word fields return undefined"
    missing:
      - "Either change types to snake_case matching existing codebase pattern (see topicGraph.ts), or add a mapping function in tagMergePreview.ts that transforms response keys"
  - truth: "Types correctly represent API response shape with camelCase mapping"
    status: failed
    reason: "No camelCase mapping exists. The project's apiClient (client.ts) passes JSON data through without transformation. Existing pattern in codebase uses snake_case response types (e.g., topicGraph.ts uses source_id/target_id/target_label)."
    artifacts:
      - path: "front/app/types/tagMerge.ts"
        issue: "All multi-word properties use camelCase (sourceTagId, sourceArticles, sourceArticleTitles) but backend sends snake_case"
      - path: "front/app/api/client.ts"
        issue: "request() method returns data.data as-is, no key transformation"
    missing:
      - "Fix types to use snake_case matching backend JSON tags, or add mapping in API layer"
  - truth: "User can trigger scan and see candidate merge pairs in cards (D-01)"
    status: partial
    reason: "UI component exists with card layout and all visual elements, but card text (sourceLabel, targetLabel, article counts) will be blank at runtime due to snake_case/camelCase mismatch. Cards render structurally but display no meaningful data."
    artifacts:
      - path: "front/app/features/topic-graph/components/TagMergePreview.vue"
        issue: "Template {{ candidate.sourceLabel }} reads undefined because runtime property is source_label"
    missing:
      - "Fix data mapping so card content renders from API response"
  - truth: "User can expand cards to see article titles (D-01)"
    status: failed
    reason: "Double mismatch: backend sends source_article_list/target_article_list with article_id, but frontend expects sourceArticleTitles/targetArticleTitles with articleId. Both key names AND property names differ."
    artifacts:
      - path: "front/app/types/tagMerge.ts"
        issue: "sourceArticleTitles/targetArticleTitles — wrong key names AND wrong case for sub-properties"
      - path: "backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go"
        issue: "Returns source_article_list/target_article_list (different name from what frontend expects)"
    missing:
      - "Align article list property names between frontend types and backend JSON tags"
  - truth: "User can edit merge name inline on each card (D-02)"
    status: partial
    reason: "Inline edit UI exists and saves to customNames Map correctly. mergeSingle sends the custom name via API (POST body uses correct snake_case). However, initial display of targetLabel will be blank."
    artifacts:
      - path: "front/app/features/topic-graph/components/TagMergePreview.vue"
        issue: "startEdit defaults to candidate.targetLabel which is undefined at runtime"
    missing:
      - "Fix data mapping so target label displays correctly as edit default"
  - truth: "User sees summary after merge completion (D-04)"
    status: partial
    reason: "Summary state and buildSummary logic are correctly implemented, but mergedDetails entries source sourceLabel from candidates array which has undefined values. Detail list would show blank source labels."
    artifacts:
      - path: "front/app/features/topic-graph/components/TagMergePreview.vue"
        issue: "buildSummary reads c.sourceLabel which would be undefined"
    missing:
      - "Fix data mapping so summary detail entries show correct labels"
---

# Phase 6: Tag Merge Preview UI — Verification Report

**Phase Goal:** 用户可手动触发全量标签合并扫描，预览待合并标签对（源→目标），修改合并后标签名称，确认后执行合并，查看合并前后差异
**Verified:** 2026-04-13T15:30:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | GET /api/topic-tags/merge-preview returns candidate pairs without auto-executing | ✓ VERIFIED | `ScanSimilarTagPairs` returns `[]TagMergeCandidate` without calling `MergeTags`; handler returns JSON without side effects |
| 2 | Each candidate includes source/target names, similarity score, article counts | ✓ VERIFIED | Backend struct has all fields: SourceLabel, TargetLabel, Similarity, SourceArticles, TargetArticles with JSON tags |
| 3 | POST /api/topic-tags/merge-with-name accepts custom name and merges correctly | ✓ VERIFIED | Handler validates inputs, renames target via Slugify, calls MergeTags transaction, returns result |
| 4 | Both APIs follow existing handler pattern from tag_management_handler.go | ✓ VERIFIED | Uses gin.H responses, early-return validation, error handling matching existing handlers |
| 5 | Frontend can call preview API and receive typed candidate pairs | ✗ FAILED | TypeScript types use camelCase but runtime data is snake_case — properties like sourceTagId/sourceLabel are undefined |
| 6 | Frontend can call merge-with-name API with custom label | ⚠️ PARTIAL | POST request body correctly uses snake_case; response type uses camelCase (newLabel vs backend's new_label) |
| 7 | Types correctly represent API response shape with camelCase mapping | ✗ FAILED | No mapping layer exists; apiClient passes data through raw; existing codebase pattern uses snake_case response types |
| 8 | User can trigger scan and see candidate merge pairs in cards (D-01) | ⚠️ PARTIAL | Component structure and card layout exist; all card text (labels, counts) renders blank due to data mapping mismatch |
| 9 | User can expand cards to see article titles (D-01) | ✗ FAILED | Double mismatch: backend sends source_article_list, frontend expects sourceArticleTitles (different name + different case) |
| 10 | User can edit merge name inline on each card (D-02) | ⚠️ PARTIAL | Edit UI exists, saves to customNames correctly; but default value from candidate.targetLabel is undefined |
| 11 | User can merge or skip each pair individually (D-03) | ✓ VERIFIED | mergeSingle() and skipCandidate() functions correctly implemented with loading states |
| 12 | User can batch merge all non-skipped pairs (D-03) | ✓ VERIFIED | batchMerge() iterates visibleCandidates, calls mergeSingle for each, builds summary |
| 13 | User sees summary after merge completion (D-04) | ⚠️ PARTIAL | Summary state with stats and detail list exists; but detail entries read sourceLabel from candidates which is undefined |

**Score:** 6/13 truths fully verified, 4 partial, 3 failed

### ROADMAP Success Criteria

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| SC1 | 用户可在前端手动触发全量标签相似度扫描 | ⚠️ PARTIAL | Scan triggered correctly via `startScan()`, API called, but candidate data fields blank at runtime |
| SC2 | 预览界面展示每对标签的源名称、目标名称、相似度、各自关联文章数 | ✗ FAILED | Labels and article counts undefined — card content blank |
| SC3 | 用户可修改合并后的标签名称 | ⚠️ PARTIAL | Edit UI works, POST body correct, but default display blank |
| SC4 | 用户可逐对确认或跳过合并，也可一键全部合并 | ✓ VERIFIED | Per-card merge/skip buttons functional, batch merge implemented |
| SC5 | 合并完成后展示结果汇总 | ⚠️ PARTIAL | Summary renders but detail list labels blank |

**Score:** 1/5 criteria fully met, 3 partial, 1 failed

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `backend-go/internal/domain/topicanalysis/tag_merge_preview.go` | ScanSimilarTagPairs + GetCandidateArticleTitles | ✓ VERIFIED | 176 lines, exports ScanSimilarTagPairs, GetCandidateArticleTitles, TagMergeCandidate, CandidateArticle |
| `backend-go/internal/domain/topicanalysis/tag_merge_preview_handler.go` | HTTP handlers for preview and custom merge | ✓ VERIFIED | 151 lines, exports ScanMergePreviewHandler, MergeTagsWithCustomNameHandler, RegisterTagMergePreviewRoutes |
| `backend-go/internal/app/router.go` | Route registration | ✓ VERIFIED | Line 168: `topicanalysisdomain.RegisterTagMergePreviewRoutes(api)` |
| `front/app/types/tagMerge.ts` | Type definitions for merge preview | ⚠️ ORPHANED | File exists (50 lines, 6 interfaces) but types use camelCase while runtime data is snake_case |
| `front/app/api/tagMergePreview.ts` | API functions | ⚠️ ORPHANED | File exists (24 lines, useTagMergePreviewApi) but no data mapping; raw pass-through |
| `front/app/features/topic-graph/components/TagMergePreview.vue` | Complete UI component | ⚠️ HOLLOW | 817 lines, all states/features implemented, but data display broken due to mapping issue |
| `front/app/features/topic-graph/components/TopicGraphPage.vue` | Entry point button | ✓ VERIFIED | Line 27: import TagMergePreview; Line 81: showMergePreview ref; Lines 866-873: trigger button; Lines 1150-1154: component instance |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| tag_merge_preview_handler.go | tag_merge_preview.go | ScanSimilarTagPairs call | ✓ WIRED | Line 36: `ScanSimilarTagPairs(limit)` |
| tag_merge_preview_handler.go | embedding.go | MergeTags call | ✓ WIRED | Line 128: `MergeTags(body.SourceTagID, body.TargetTagID)` |
| router.go | tag_merge_preview_handler.go | RegisterTagMergePreviewRoutes | ✓ WIRED | Line 168: registered in api group |
| tagMergePreview.ts | /api/topic-tags/merge-preview | apiClient.get | ✓ WIRED | Line 12: GET endpoint |
| tagMergePreview.ts | /api/topic-tags/merge-with-name | apiClient.post | ✓ WIRED | Line 17: POST endpoint |
| TagMergePreview.vue | useTagMergePreviewApi | scanMergePreview, mergeTagsWithCustomName | ✓ WIRED | Lines 46, 104: API calls |
| TopicGraphPage.vue | TagMergePreview.vue | component import + :visible prop | ✓ WIRED | Lines 27, 81, 866-873, 1150-1154 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| TagMergePreview.vue → candidates | `candidates.value` | API response `response.data.candidates` | Yes (backend returns real data) | ⚠️ HOLLOW — backend sends snake_case keys, frontend reads camelCase keys |
| TagMergePreview.vue → sourceArticleTitles | `candidate.sourceArticleTitles` | API response `source_article_list` | Name mismatch + case mismatch | ✗ DISCONNECTED |
| TagMergePreview.vue → sourceLabel display | `candidate.sourceLabel` | API response `source_label` | Case mismatch | ✗ DISCONNECTED |
| TagMergePreview.vue → similarity display | `candidate.similarity` | API response `similarity` | Single word — matches | ✓ FLOWING |
| TagMergePreview.vue → mergeSummary | `mergeSummary.value` | Computed from candidates array | Built from candidate fields | ⚠️ HOLLOW — source labels from candidates are undefined |

**Root Cause Analysis:** The project's `ApiClient` (`front/app/api/client.ts`) passes response JSON through without any key transformation. The established pattern in this codebase is to use **snake_case** in response types (see `topicGraph.ts` line 355: `{ source_id: number; target_id: number; target_label: string }`). The Phase 06 types and component incorrectly use **camelCase** without adding a mapping layer.

**Evidence of existing pattern:**
- `TopicGraphPage.vue` line 853: `graphPayload?.period_label` (snake_case)
- `topicGraph.ts` line 355: `{ source_id: number; target_id: number; target_label: string }` (snake_case)
- `TagMergePreview.vue` line 248: `candidate.sourceLabel` (camelCase — BREAKS PATTERN)

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Backend compiles | `cd backend-go && go build ./...` | No errors | ✓ PASS |
| Backend route registered | grep RegisterTagMergePreviewRoutes router.go | Found at line 168 | ✓ PASS |

Step 7b: SKIPPED (no running server available for endpoint testing)

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-----------|-------------|--------|----------|
| CONV-02 | 06-01, 06-02, 06-03 | 标签合并时在事务内迁移 article_topic_tags 等关联记录到目标标签，防止引用悬空 | ⚠️ PARTIAL | Backend MergeTags transaction is correct (verified in Phase 01). Phase 06 adds UI — backend APIs work, but frontend data mapping prevents UI from functioning correctly |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| front/app/types/tagMerge.ts | 7-19 | camelCase types for snake_case API data | 🛑 Blocker | All card text renders blank |
| front/app/api/tagMergePreview.ts | 7-23 | No data mapping between API response and types | 🛑 Blocker | Runtime data shape doesn't match TypeScript expectations |
| front/app/features/topic-graph/components/TopicGraphPage.vue | 757 | `handleMergeComplete(_summary)` — unused summary parameter | ℹ️ Info | Prefixed with `_` to satisfy lint; not a real issue |

**Note on stub classification:** The empty implementations check (return null/return {}/return []) found zero matches. The console.log check found legitimate error logging, not stub handlers. The component is fully implemented — the issue is purely a data mapping gap.

### Human Verification Required

### 1. UI Data Display Test

**Test:** Run frontend + backend, navigate to topics page, click "标签合并预览" button
**Expected:** Cards display source label → target label with similarity badge and article counts
**Why human:** Need to visually confirm whether the snake_case/camelCase mismatch actually causes blank rendering (our analysis says yes, but runtime behavior could differ if there's an unexpected transformation layer)

### 2. Article Title Expansion Test

**Test:** Click expand ("查看文章") on a candidate card
**Expected:** Two-column layout showing article titles for source and target tags
**Why human:** The article list property name mismatch (source_article_list vs sourceArticleTitles) needs runtime verification

### 3. Merge Execution Test

**Test:** Click merge ("合并") on a candidate card
**Expected:** Card disappears, merge API succeeds, card reappears in summary
**Why human:** The POST body uses correct snake_case so merge should work, but the UI feedback (loading states, success/failure) needs visual confirmation

### Gaps Summary

**One root cause with cascading impact:** The frontend types (`tagMerge.ts`) and component (`TagMergePreview.vue`) use camelCase property names (sourceTagId, sourceLabel, etc.) but the backend API returns snake_case JSON (source_tag_id, source_label, etc.). The project's ApiClient does not transform keys. The existing codebase pattern (topicGraph.ts, TopicGraphPage.vue) uses snake_case response types.

**Impact:** The entire preview UI renders structurally but displays blank text for all card data — labels, article counts, and article titles. The merge/skip actions still work (POST body is correct snake_case), but the user experience is broken because the preview information is invisible.

**Fix required:**
1. **Option A (recommended, consistent with codebase):** Change `tagMerge.ts` types to use snake_case, update `TagMergePreview.vue` template to read snake_case properties. Align with existing pattern in `topicGraph.ts`.
2. **Option B:** Add a mapping function in `tagMergePreview.ts` that transforms the API response from snake_case to camelCase before returning.

**Estimated fix scope:** ~30 lines changed across 3 files (tagMerge.ts types, tagMergePreview.ts mapping, TagMergePreview.vue template accessors).

---

_Verified: 2026-04-13T15:30:00Z_
_Verifier: the agent (gsd-verifier)_
