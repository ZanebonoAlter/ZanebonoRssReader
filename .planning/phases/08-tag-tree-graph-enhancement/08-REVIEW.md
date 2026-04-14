---
phase: 08-tag-tree-graph-enhancement
reviewed: 2026-04-14T23:30:00Z
depth: quick
files_reviewed: 8
findings:
  critical: 0
  warning: 2
  info: 2
  total: 4
status: issues_found
files_reviewed_list:
  - backend-go/internal/domain/topictypes/types.go
  - backend-go/internal/domain/topicgraph/service.go
  - front/app/features/topic-graph/components/TagMergePreview.vue
  - front/app/components/dialog/GlobalSettingsDialog.vue
  - backend-go/internal/domain/topicanalysis/abstract_tag_service.go
  - backend-go/internal/domain/topicanalysis/abstract_tag_handler.go
  - front/app/features/topic-graph/components/TagHierarchy.vue
  - front/app/features/topic-graph/components/TagHierarchyRow.vue
---

# Phase 8: Code Review Report (Gap Closure 08-07, 08-08, 08-09)

**Reviewed:** 2026-04-14T23:30:00Z
**Depth:** quick
**Files Reviewed:** 8
**Status:** issues_found

## Summary

Reviewed 8 source files changed across three gap-closure plans: abstract tag annotation (08-07), TagMergePreview inline rendering (08-08), custom date range + inactive tag sorting (08-09). The changes are well-structured and follow existing patterns. No critical security issues found — GORM parameterized queries protect against SQL injection in the custom date range path. Found 2 warnings: a logic bug in `resolveActiveTagIDs` that marks all tags inactive when `custom:` prefix is malformed, and unchecked DB errors in `findAbstractSlugs`.

## Warnings

### WR-01: `resolveActiveTagIDs` returns all-inactive for malformed `custom:` prefix

**File:** `backend-go/internal/domain/topicanalysis/abstract_tag_service.go:564-587`
**Issue:** When `timeRange` starts with `"custom:"` but has fewer than 3 colon-separated parts (e.g., `"custom:"`, `"custom:2024-01-01"`), the `if len(parts) == 3` guard fails, `activeIDs` stays nil, and the function returns an empty result map. This means **all tags render as inactive** instead of falling back to "all active" like other invalid-format cases do.

The `default:` branch and the invalid-date branches both correctly fall back to marking all tags active, but the `len(parts) != 3` path inside the `custom:` case does not.

**Fix:**
```go
case strings.HasPrefix(timeRange, "custom:"):
    parts := strings.SplitN(timeRange, ":", 3)
    if len(parts) != 3 {
        // Malformed custom range — treat as no filter
        for id := range candidateIDs {
            result[id] = true
        }
        return result
    }
    startDate := parts[1]
    endDate := parts[2]
    // ... rest of validation unchanged
```

### WR-02: `findAbstractSlugs` silently ignores DB errors

**File:** `backend-go/internal/domain/topicgraph/service.go:970-979`
**Issue:** Both the `Pluck` call (line 970-972) and the `Find` call (line 979) discard their error returns. If the database is unreachable or the query fails, the function silently returns with no abstract annotations rather than logging or propagating the error. Consistent with project pattern of logging warnings for non-critical DB failures.

**Fix:**
```go
func findAbstractSlugs(db *gorm.DB, topicNodes map[string]*topictypes.GraphNode) {
    var abstractParentIDs []uint
    if err := db.Model(&models.TopicTagRelation{}).
        Select("DISTINCT parent_id").
        Pluck("parent_id", &abstractParentIDs).Error; err != nil {
        fmt.Printf("Warning: findAbstractSlugs pluck failed: %v\n", err)
        return
    }

    if len(abstractParentIDs) == 0 {
        return
    }

    var parentTags []models.TopicTag
    if err := db.Where("id IN ?", abstractParentIDs).Find(&parentTags).Error; err != nil {
        fmt.Printf("Warning: findAbstractSlugs find failed: %v\n", err)
        return
    }
    // ... rest unchanged
```

## Info

### IN-01: TagHierarchyRow local `editingValue` duplicates parent state

**File:** `front/app/features/topic-graph/components/TagHierarchyRow.vue:22`
**Issue:** The component maintains its own `editingValue` ref (line 22) while the parent `TagHierarchy.vue` also maintains an `editingValue` ref. The two are synced via `handleInput` → `emit('update:editing-value')` and `handleStartEdit` → `emit('start-edit')`. This works but creates a dual-source-of-truth pattern. The local ref is written to but only read for initial value in `handleStartEdit` — the template reads from the parent via the `editingId` check and `handleInput` emit.

**Fix:** Non-urgent. The current sync mechanism works correctly — this is a maintainability suggestion for a future refactor.

### IN-02: `showCustomRange` button active state decoupled from actual `timeRange`

**File:** `front/app/features/topic-graph/components/TagHierarchy.vue:349`
**Issue:** The "自定义" button's `--active` class is bound to `showCustomRange` (the toggle for the date picker panel), but the time range filter button states are bound to `timeRange`. After applying a custom range and then switching to another preset (e.g., "7天"), the date picker panel closes (via the `timeRange` watch on line 250-251), but if the user re-opens the custom panel, the button lights up even though `timeRange` is currently `"7d"`. Minor visual inconsistency — not a functional bug.

**Fix:** Non-urgent. Could bind the active state to `timeRange.startsWith('custom:')` instead of `showCustomRange` for the button highlight, while still using `showCustomRange` for the panel visibility.

---

_Reviewed: 2026-04-14T23:30:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: quick_
