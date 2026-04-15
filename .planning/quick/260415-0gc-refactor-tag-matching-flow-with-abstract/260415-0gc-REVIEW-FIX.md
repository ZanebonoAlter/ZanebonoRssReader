# Review Fix Summary: 260415-0gc

**Date:** 2026-04-15
**Review file:** 260415-0gc-REVIEW.md
**Fix scope:** MEDIUM + LOW (7 findings)

## Verification

- `go build ./...` — PASS
- `go test ./internal/domain/topicanalysis/... ./internal/domain/topicextraction/...` — ALL PASS

## Fixes Applied

### MEDIUM-1: Eliminated embedding race condition in `createChildOfAbstract`
**File:** `tagger.go:313-316` (removed)
**Fix:** Removed the `generateTagDescription` and `generateAndSaveEmbedding` calls from `createChildOfAbstract`. Child tags of abstract parents don't need embeddings since they're deleted immediately after the parent-child relation is created. The old code had a race where the async goroutine would re-create the embedding after deletion.

### MEDIUM-2: Filter candidates by similarity before abstract extraction
**File:** `tagger.go:221-237` (modified)
**Fix:** Before calling `ExtractAbstractTag`, candidates are now filtered to only those with `Similarity >= LowSimilarity`. This prevents low-similarity tags (below 0.78) from losing their embeddings or being incorrectly linked as children of an abstract tag. If no candidates pass the filter, falls back to the best match to preserve behavior.

Added `GetThresholds()` method to `EmbeddingService` (embedding.go) to expose thresholds to the `topicextraction` package.

### MEDIUM-3: Added category filter to `FindSimilarAbstractTags`
**File:** `embedding.go:287-316` (modified)
**Fix:** `FindSimilarAbstractTags` now accepts a `category` parameter. When non-empty, adds `AND t.category = ?` to the SQL query, preventing cross-category abstract tag matching (e.g., "person" tags matching "event" abstract tags).

Updated `MatchAbstractTagHierarchy` to load the abstract tag's category and pass it to `FindSimilarAbstractTags`.

### LOW-1: Use instance thresholds in `MatchAbstractTagHierarchy`
**File:** `abstract_tag_service.go:841,853` (modified)
**Fix:** Replaced hardcoded `DefaultThresholds.HighSimilarity` / `.LowSimilarity` with `es.GetThresholds()` so custom threshold configuration from `embedding_config` is respected.

### LOW-2: Preserve embedding when relation creation fails
**File:** `tagger.go:327-331` (modified)
**Fix:** When the parent-child relation creation fails in `createChildOfAbstract`, the code now enqueues embedding generation for the tag so it remains discoverable in future similarity searches. Without this, the tag would exist without a relation and without an embedding — invisible to all matching.

### LOW-3: Moved `MatchAbstractTagHierarchy` into embedding goroutine
**File:** `abstract_tag_service.go:91-103, 157-159` (modified)
**Fix:** `MatchAbstractTagHierarchy` is now called at the end of the embedding generation goroutine in `ExtractAbstractTag`, after `SaveEmbedding` succeeds. Previously it was a separate `go` call that could fire before the embedding was saved, causing `FindSimilarAbstractTags` to fail silently.

### LOW-4: Rune-based truncation for CJK text safety
**File:** `abstract_tag_service.go:958-963`, `tagger.go:96-98, 383-386` (modified)
**Fix:** Replaced `len(s)` / `s[:n]` byte-based truncation with `[]rune(s)` / `string(runes[:n])` in:
- `truncateStr` helper
- `generateTagDescription` description truncation (500 chars)
- `TagSummary` article context truncation (200 chars)

This prevents invalid UTF-8 sequences when truncating Chinese/Japanese/Korean text.

## Not Fixed (Deferred)

| Finding | Reason |
|---------|--------|
| INFO-1: Duplicate SQL patterns | Refactoring concern, not a bug. Low priority. |
| INFO-2: Missing test coverage for `MatchAbstractTagHierarchy` | Requires DB mocking infrastructure. Track separately. |

## Files Changed

| File | Changes |
|------|---------|
| `backend-go/internal/domain/topicextraction/tagger.go` | MEDIUM-1, MEDIUM-2, LOW-2, LOW-4 |
| `backend-go/internal/domain/topicanalysis/embedding.go` | MEDIUM-3, `GetThresholds()` method |
| `backend-go/internal/domain/topicanalysis/abstract_tag_service.go` | LOW-1, LOW-3, LOW-4 |
