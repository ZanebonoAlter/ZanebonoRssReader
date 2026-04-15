<!-- generated-by: gsd-doc-writer -->

# Testing

This project has three independent test suites covering the Go backend, the Nuxt frontend, and cross-system integration flows.

## Test Frameworks and Setup

| Suite | Framework | Location | Language |
|-------|-----------|----------|----------|
| Backend unit tests | Go `testing` + `testify` | `backend-go/**/*_test.go` | Go |
| Frontend unit tests | Vitest + Vue Test Utils | `front/app/**/*.test.ts` | TypeScript |
| Frontend E2E tests | Playwright | `front/tests/e2e/*.spec.ts` | TypeScript |
| Integration tests | pytest | `tests/workflow/` | Python |
| Firecrawl check | Python script | `tests/firecrawl/` | Python |

### Backend (Go)

Tests use the standard `testing` package with `github.com/stretchr/testify` for assertions. Each test file lives alongside the source code as `*_test.go`. Most tests create an in-memory SQLite database via `gorm.Open(sqlite.Open("file:...?mode=memory&cache=shared"))` and auto-migrate the required models, so no external database is needed.

### Frontend Unit (Vitest)

Vitest runs with `happy-dom` as the DOM environment. Configuration lives in `front/vitest.config.ts`. Tests are co-located with source files using the `*.test.ts` naming convention under `front/app/`.

### Frontend E2E (Playwright)

Playwright is configured in `front/playwright.config.ts`. It starts the Nuxt dev server on `http://localhost:3000`, then runs browser tests against it. Tests are serialized (`fullyParallel: false`, `workers: 1`) for cold-startup stability.

### Integration (pytest)

Integration tests in `tests/workflow/` verify end-to-end scheduler behavior and data flows. They require the Go backend running on `http://localhost:5000` and access the SQLite database directly. Configuration is in `tests/workflow/config.py`.

## Running Tests

### Backend

```bash
# All backend tests
cd backend-go
go test ./...

# Single package (verbose)
go test ./internal/domain/feeds -v

# Single test by name
go test ./internal/domain/feeds -run TestBuildArticleFromEntryTracksOnlyRunnableStates -v
```

### Frontend Unit Tests

```bash
cd front

# Run all unit tests
pnpm test:unit

# Single test file
pnpm test:unit -- app/utils/articleContentSource.test.ts

# Single test by name pattern
pnpm test:unit -- app/utils/articleContentSource.test.ts -t "prefers firecrawl"
```

### Frontend E2E Tests

```bash
cd front

# Run all E2E tests (starts dev server automatically)
pnpm test:e2e

# Run with Playwright UI
pnpm test:e2e:ui

# List tests without running
pnpm test:e2e:list

# Pass extra Playwright arguments
pnpm test:e2e:args -- --grep "topic-graph"
```

### Python Integration Tests

```bash
# Set up environment (first time)
cd tests/workflow
uv venv
.venv\Scripts\activate    # Windows
uv pip install -r requirements.txt

# Run all integration tests
pytest test_*.py -v

# Single file
pytest test_schedulers.py -v

# Single test
pytest test_schedulers.py::TestAutoRefreshScheduler::test_scheduler_exists -v

# With coverage
pytest --cov=. --cov-report=html
```

The integration tests require the Go backend running at `http://localhost:5000`. Start it from a separate terminal:

```bash
cd backend-go
go run cmd/server/main.go
```

### Firecrawl Integration Check

A standalone script that verifies Firecrawl service connectivity, scrape functionality, article content updates, and AI configuration:

```bash
cd backend-go
go run cmd/server/main.go    # Start backend in a separate terminal

cd tests/firecrawl
python test_firecrawl_integration.py
```

<!-- VERIFY: Firecrawl service URL (http://192.168.5.27:3002) is environment-specific and configured in tests/firecrawl/config.py -->

## Writing New Tests

### Backend

- Place test files next to the source as `*_test.go` in the same package.
- Use table-driven tests when multiple cases share the same logic.
- For tests needing a database, create an in-memory SQLite instance and call `AutoMigrate` on the required models. See `backend-go/internal/domain/feeds/service_test.go` for the `setupFeedsTestDB` pattern.
- Import `github.com/stretchr/testify` only when a file already uses it; otherwise use the standard `testing` package directly.
- Use `t.Helper()` in setup functions for cleaner error traces.

### Frontend Unit Tests

- Co-locate test files with source: `front/app/<path>/file.test.ts`.
- Use `describe`/`it` blocks from Vitest:
  ```typescript
  import { describe, expect, it } from 'vitest'
  import { myFunction } from './myFunction'

  describe('myFunction', () => {
    it('does the expected thing', () => {
      expect(myFunction('input')).toBe('expected')
    })
  })
  ```
- Tests run in `happy-dom` environment — no real browser needed.
- E2E test files in `front/tests/e2e/` are excluded from Vitest via `vitest.config.ts`.

### Frontend E2E Tests

- Place spec files in `front/tests/e2e/*.spec.ts`.
- Use Playwright's `test` and `expect` imports:
  ```typescript
  import { test, expect } from '@playwright/test'

  test('page loads', async ({ page }) => {
    await page.goto('/some-page')
    await expect(page.locator('body')).toBeVisible()
  })
  ```
- Tests run sequentially against Chromium. The dev server starts automatically via the `webServer` config in `playwright.config.ts`.

### Integration Tests

- Test files live in `tests/workflow/test_*.py`.
- Use the shared helpers from `tests/workflow/utils/`:
  - `DatabaseHelper` — direct SQLite access for asserting database state
  - `APIClient` — HTTP client for calling backend API endpoints
  - `MockFirecrawl` / `MockAIService` — mock external service responses
- Each test class uses `pytest.fixture(autouse=True)` for setup/teardown that creates test data and cleans up afterward.
- See `tests/workflow/config.py` for `TestConfig` and `DatabaseConfig` defaults.

## Coverage

No minimum coverage thresholds are configured in any test framework.

- **Frontend**: Vitest does not have coverage settings in `vitest.config.ts`. Run `pnpm test:unit` for pass/fail results only.
- **Backend**: No `cover` profile or threshold flags in Go test commands.
- **Integration**: `pytest-cov` is listed as a dependency in `tests/workflow/requirements.txt`. Generate a report with `pytest --cov=. --cov-report=html`.

## CI Integration

No CI/CD pipeline is currently configured. There are no `.github/workflows/` files in the repository.

All tests are run locally:

- **Backend**: `go test ./...` from `backend-go/`
- **Frontend unit**: `pnpm test:unit` from `front/`
- **Frontend type check**: `pnpm exec nuxi typecheck` from `front/`
- **Frontend E2E**: `pnpm test:e2e` from `front/`
- **Integration**: `pytest test_*.py -v` from `tests/workflow/` (requires running backend)

The recommended pre-push verification sequence is:

```bash
cd backend-go && go test ./... && go build ./...
cd front && pnpm exec nuxi typecheck && pnpm test:unit && pnpm build
```

## Test Organization Summary

```
backend-go/
├── internal/domain/feeds/service_test.go                 # Feed parsing, article creation
├── internal/domain/articles/handler_test.go              # Article HTTP handlers
├── internal/domain/summaries/summary_queue_test.go
├── internal/domain/summaries/ai_prompt_builder_test.go
├── internal/domain/digest/generator_test.go              # Digest generation
├── internal/domain/digest/scheduler_test.go
├── internal/domain/digest/handler_test.go
├── internal/domain/digest/integration_test.go
├── internal/domain/digest/obsidian_test.go
├── internal/domain/digest/feishu_test.go
├── internal/domain/contentprocessing/firecrawl_job_queue_test.go
├── internal/domain/contentprocessing/content_completion_service_test.go
├── internal/domain/contentprocessing/content_completion_handler_test.go
├── internal/domain/topicextraction/tag_job_queue_test.go
├── internal/domain/topicextraction/metadata_test.go
├── internal/domain/topicextraction/extractor_test.go
├── internal/domain/topicgraph/handler_test.go
├── internal/domain/topicanalysis/analysis_queue_test.go
├── internal/domain/preferences/handler_test.go
├── internal/domain/aiadmin/handler_test.go
├── internal/jobs/auto_refresh_test.go                    # Background job processors
├── internal/jobs/firecrawl_test.go
├── internal/jobs/content_completion_test.go
├── internal/jobs/preference_update_test.go
├── internal/jobs/auto_summary_test.go
├── internal/jobs/handler_test.go
├── internal/platform/database/db_test.go                 # DB and migration
├── internal/platform/database/datamigrate/writer_postgres_test.go
├── internal/platform/database/datamigrate/verify_test.go
├── internal/platform/config/config_test.go
├── internal/platform/airouter/router_test.go             # AI model routing
├── internal/platform/airouter/store_test.go
├── internal/platform/airouter/migration_test.go
├── internal/platform/opennotebook/client_test.go
├── internal/platform/aisettings/config_store_test.go
└── cmd/migrate-db/main_test.go

front/
├── app/utils/api.test.ts                                 # API client utilities
├── app/utils/articleContentSource.test.ts                # Content source resolution
├── app/utils/articleContentGuards.test.ts
├── app/utils/schedulerMeta.test.ts
├── app/features/articles/components/ArticleTagList.test.ts
├── app/features/digest/components/digestLayout.test.ts
├── app/features/topic-graph/utils/buildDisplayedTopicGraph.test.ts
├── app/features/topic-graph/utils/topicGraphCanvasLinks.test.ts
├── app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts
├── app/features/topic-graph/components/TopicTimeline.test.ts
└── tests/e2e/
    ├── baseline.spec.ts                                  # Smoke tests
    └── topic-graph.spec.ts                               # Topic graph E2E

tests/
├── workflow/
│   ├── test_schedulers.py                                # Scheduler unit tests
│   ├── test_workflow_integration.py                      # End-to-end workflow tests
│   ├── test_error_handling.py                            # Error handling tests
│   ├── utils/                                            # Shared test helpers
│   │   ├── database.py                                   # DatabaseHelper
│   │   ├── api_client.py                                 # APIClient
│   │   └── mock_services.py                              # MockFirecrawl, MockAIService
│   └── config.py                                         # Test configuration
└── firecrawl/
    └── test_firecrawl_integration.py                     # Firecrawl integration check
```
