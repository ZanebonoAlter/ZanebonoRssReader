# AGENTS.md

Agent guide for coding assistants working in `D:\project\my-robot`.

## Rule Sources
- Primary source of truth: this file, `README.md`, and docs under `docs/`.
- Subdirectory guides: `front/AGENTS.md`, `backend-go/AGENTS.md` for domain-specific conventions.
- Checked for Cursor rules: no `.cursorrules` and no `.cursor/rules/` directory found.
- Checked for Copilot rules: no `.github/copilot-instructions.md` found.
- If new rule files appear later, merge their guidance here before making broad changes.

## Project Snapshot
- RSS Reader app with a Nuxt 4 frontend and a Go backend.
- Main features: feed subscriptions, article reading, AI summaries, Firecrawl enrichment, digest export, schedulers.
- Personal/single-user deployment only; there is no auth system.
- Frontend API: `http://localhost:5000/api`; WebSocket: `ws://localhost:5000/ws`.
- Backend persistence is SQLite.

## Repo Layout
- `front/`: Nuxt 4, Vue 3, TypeScript, Pinia, Tailwind CSS v4.
- `backend-go/`: Gin, GORM, SQLite, schedulers, digest jobs.
- `docs/`: architecture and workflow docs.
- `tests/workflow/`: Python integration tests for scheduler and workflow behavior.
- `tests/firecrawl/`: Python integration check for Firecrawl flow.

## First Files To Read
- `README.md` for product scope.
- `front/app/app.vue` for frontend entry.
- `front/app/api/client.ts` for HTTP conventions.
- `front/app/stores/api.ts` for data mapping and main store usage.
- `backend-go/cmd/server/main.go` for backend entry.
- `backend-go/internal/app/router.go` for route layout.
- `backend-go/internal/app/runtime.go` for scheduler/runtime wiring.

## Build, Test, And Verify

### Frontend
Run from `front/`:
```bash
pnpm install
pnpm dev
pnpm build
pnpm generate
pnpm preview
pnpm exec nuxi typecheck
pnpm test:unit
```
- Single test file: `pnpm test:unit -- app/utils/articleContentSource.test.ts`
- Single test by name: `pnpm test:unit -- app/utils/articleContentSource.test.ts -t "prefers firecrawl"`
- No dedicated lint script is configured in `front/package.json`.
- Main quality gates: `pnpm exec nuxi typecheck` and `pnpm build`.

### Backend Go
Run from `backend-go/`:
```bash
go mod tidy
go run cmd/server/main.go
go build ./...
go test ./...
go run cmd/migrate-digest/main.go
go run cmd/test-digest/main.go
```
- Single package: `go test ./internal/domain/feeds -v`
- Single test: `go test ./internal/domain/feeds -run TestBuildArticleFromEntryTracksOnlyRunnableStates -v`
- Prefer targeted package tests first, then `go test ./...` for broader coverage.

### Python Integration Tests
Run from `tests/workflow/`:
```bash
uv venv
.venv\Scripts\activate
uv pip install -r requirements.txt
pytest test_*.py -v
```
- Single file: `pytest test_schedulers.py -v`
- Single test: `pytest test_schedulers.py::TestAutoRefreshScheduler::test_name -v`
- Coverage: `pytest --cov=. --cov-report=html`
- These tests expect the Go backend on `localhost:5000`.

### Firecrawl Check
- Start backend from `backend-go/` with `go run cmd/server/main.go`.
- Then run `python test_firecrawl_integration.py` from `tests/firecrawl/`.

## Frontend Conventions
- Use Vue 3 Composition API with `<script setup lang="ts">` for new Vue files.
- Use TypeScript across frontend code.
- Keep route pages thin; move business logic into `front/app/features/` or composables.
- Put network calls in `front/app/api/`, not directly in components.
- `useApiStore` is the primary data source; other stores should be derived UI state.
- Keep shared types in `front/app/types/`.
- Convert backend numeric IDs to frontend strings at the API/store boundary.
- Keep `snake_case -> camelCase` mapping in API/store code, never in templates.
- Reuse `ApiResponse<T>` for request results.

### Frontend Imports, Formatting, Naming
- Preferred import order: Vue/Nuxt, third-party, internal modules, then type-only imports.
- Use `import type` for type-only dependencies.
- Use `~` alias imports for app-root paths.
- Follow existing file-local formatting; do not reformat unrelated lines.
- Most frontend files omit semicolons; preserve surrounding style.
- Frontend files must remain UTF-8; never rewrite them as ANSI, GBK, or UTF-16.
- Components: PascalCase, e.g. `ArticleContentView.vue`.
- Composables and stores: camelCase with `use` prefix, e.g. `useSummaryWebSocket`.
- Utility files: descriptive camelCase names.
- Props interfaces are commonly named `Props`; emits should be typed with `defineEmits<...>()`.

### Frontend Error Handling
- Wrap HTTP access behind `ApiClient`.
- Return `{ success, data, error, message }` shaped results instead of throwing into the UI.
- In components, show friendly messages and keep `console.error` for debugging context.
- Prefer defensive null checks around API data.

## Backend Go Conventions
- Keep HTTP routes in `internal/app/router.go`; keep business logic in `internal/domain/*`.
- Use focused domain packages such as `feeds`, `digest`, `summaries`, and `contentprocessing`.
- Use PascalCase for exported symbols and lowerCamelCase for private helpers.
- Keep JSON fields snake_case via struct tags.
- Use `fmt.Errorf(... %w ...)` when wrapping lower-level errors.
- Prefer early returns for validation failures and DB errors.
- Handlers should return `gin.H{"success": bool, "data"|"error"|"message": ...}`.
- Validate params and request bodies before touching the database.
- Keep GORM models in `internal/domain/models` and shared infrastructure in `internal/platform/*`.

### Backend Imports, Formatting, Tests
- Let `gofmt` format Go files.
- Group imports as stdlib, blank line, third-party, blank line, local packages.
- Keep package names short; alias only when collision or readability requires it.
- Use `testing` directly; `testify` is acceptable if a file already uses it.
- Prefer table tests when many cases share behavior.
- Keep tests close to code as `*_test.go` files.

## UI And Content Direction
- Preserve the repo's editorial / magazine feel.
- Avoid generic SaaS layouts, especially centered hero-plus-cards pages.
- Do not introduce purple/indigo default SaaS palettes.
- Prefer textured, layered, or gradient backgrounds over flat fills.
- Use Iconify for icons.
- Keep copy short, concrete, and conversational; short Chinese UI text is common here.

## Docs And Architecture Notes
- Update docs when APIs, runtime flows, or major UI structure change.
- Relevant docs usually include `docs/architecture/frontend.md`, `docs/architecture/backend-go.md`, and `docs/operations/development.md`.
- If scheduler flow, digest flow, or data mapping changes, document it.

## GitNexus Workflow
- Repo is indexed in GitNexus as `my-robot`.
- Before editing a function, method, or class, run `gitnexus_impact` on that symbol.
- If impact risk is HIGH or CRITICAL, warn the user before proceeding.
- Use `gitnexus_query` to understand unfamiliar execution flows.
- Use `gitnexus_context` when you need callers, callees, and process participation.
- Before committing, run `gitnexus_detect_changes()` and confirm the affected scope matches intent.

## Agent Expectations
- Do not assume there is a Python backend; the product backend is Go.
- Do not add new linters, formatters, or tooling unless the user asks.
- Ignore unrelated dirty-worktree changes.
- Verify the smallest relevant command after edits, then broaden if needed.
- Frontend-only edits: prefer `pnpm exec nuxi typecheck`, `pnpm test:unit`, or `pnpm build`.
- Backend-only edits: prefer targeted `go test` first, then `go test ./...` or `go build ./...`.
- Docs-only edits usually only need consistency checks unless the docs describe changed behavior.

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **my-robot** (2264 symbols, 5081 relationships, 179 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## When Debugging

1. `gitnexus_query({query: "<error or symptom>"})` — find execution flows related to the issue
2. `gitnexus_context({name: "<suspect function>"})` — see all callers, callees, and process participation
3. `READ gitnexus://repo/my-robot/process/{processName}` — trace the full execution flow step by step
4. For regressions: `gitnexus_detect_changes({scope: "compare", base_ref: "main"})` — see what your branch changed

## When Refactoring

- **Renaming**: MUST use `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` first. Review the preview — graph edits are safe, text_search edits need manual review. Then run with `dry_run: false`.
- **Extracting/Splitting**: MUST run `gitnexus_context({name: "target"})` to see all incoming/outgoing refs, then `gitnexus_impact({target: "target", direction: "upstream"})` to find all external callers before moving code.
- After any refactor: run `gitnexus_detect_changes({scope: "all"})` to verify only expected files changed.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Tools Quick Reference

| Tool | When to use | Command |
|------|-------------|---------|
| `query` | Find code by concept | `gitnexus_query({query: "auth validation"})` |
| `context` | 360-degree view of one symbol | `gitnexus_context({name: "validateUser"})` |
| `impact` | Blast radius before editing | `gitnexus_impact({target: "X", direction: "upstream"})` |
| `detect_changes` | Pre-commit scope check | `gitnexus_detect_changes({scope: "staged"})` |
| `rename` | Safe multi-file rename | `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` |
| `cypher` | Custom graph queries | `gitnexus_cypher({query: "MATCH ..."})` |

## Impact Risk Levels

| Depth | Meaning | Action |
|-------|---------|--------|
| d=1 | WILL BREAK — direct callers/importers | MUST update these |
| d=2 | LIKELY AFFECTED — indirect deps | Should test |
| d=3 | MAY NEED TESTING — transitive | Test if critical path |

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/my-robot/context` | Codebase overview, check index freshness |
| `gitnexus://repo/my-robot/clusters` | All functional areas |
| `gitnexus://repo/my-robot/processes` | All execution flows |
| `gitnexus://repo/my-robot/process/{name}` | Step-by-step execution trace |

## Self-Check Before Finishing

Before completing any code modification task, verify:
1. `gitnexus_impact` was run for all modified symbols
2. No HIGH/CRITICAL risk warnings were ignored
3. `gitnexus_detect_changes()` confirms changes match expected scope
4. All d=1 (WILL BREAK) dependents were updated

## Keeping the Index Fresh

After committing code changes, the GitNexus index becomes stale. Re-run analyze to update it:

```bash
npx gitnexus analyze
```

If the index previously included embeddings, preserve them by adding `--embeddings`:

```bash
npx gitnexus analyze --embeddings
```

To check whether embeddings exist, inspect `.gitnexus/meta.json` — the `stats.embeddings` field shows the count (0 means no embeddings). **Running analyze without `--embeddings` will delete any previously generated embeddings.**

> Claude Code users: A PostToolUse hook handles this automatically after `git commit` and `git merge`.

## CLI

- Re-index: `npx gitnexus analyze`
- Check freshness: `npx gitnexus status`
- Generate docs: `npx gitnexus wiki`

<!-- gitnexus:end -->
