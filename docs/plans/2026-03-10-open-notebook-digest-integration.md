# Open Notebook Digest Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a new digest-side integration that sends existing digest Markdown to open-notebook for a second-pass summary without disturbing the current AI summary pipeline.

**Architecture:** Reuse `buildPreview()` in the digest domain as the single source of input Markdown, add a dedicated `platform/opennotebook` client plus isolated config storage, then expose digest-scoped config and send endpoints to the frontend digest view. The first version is manual-only with inline result display; no persistence table is required yet.

**Tech Stack:** Go 1.21, Gin, GORM, SQLite, Nuxt 4, Vue 3, TypeScript, Pinia

---

### Task 1: Define open-notebook config model in frontend and backend contract

**Files:**
- Modify: `front/app/api/digest.ts`
- Modify: `front/app/features/digest/components/DigestSettings.vue`
- Optional reference: `backend-go/internal/platform/aisettings/config_store.go`

**Step 1: Add frontend config types**

Add `OpenNotebookConfig` and `OpenNotebookRunResult` to `front/app/api/digest.ts`.

Fields:

```ts
export interface OpenNotebookConfig {
  enabled: boolean
  base_url: string
  api_key: string
  model: string
  target_notebook: string
  prompt_mode: 'digest_summary'
  auto_send_daily: boolean
  auto_send_weekly: boolean
  export_back_to_obsidian: boolean
}

export interface OpenNotebookRunResult {
  digest_type: DigestType
  anchor_date: string
  source_markdown: string
  summary_markdown: string
  remote_id?: string
  remote_url?: string
}
```

**Step 2: Add API methods in `front/app/api/digest.ts`**

Add:

```ts
async getOpenNotebookConfig()
async updateOpenNotebookConfig(config: OpenNotebookConfig)
async sendToOpenNotebook(type: DigestType, date?: string)
```

**Step 3: Keep naming aligned with backend JSON**

Use snake_case in API layer only.

**Step 4: Commit**

```bash
git add front/app/api/digest.ts front/app/features/digest/components/DigestSettings.vue
git commit -m "feat(digest): define open-notebook config contract"
```

---

### Task 2: Add backend config helpers for open-notebook

**Files:**
- Modify: `backend-go/internal/platform/aisettings/config_store.go`
- Test: `backend-go/internal/platform/aisettings/config_store_test.go`

**Step 1: Write the failing test**

Create tests for loading and saving `open_notebook_config` without affecting `summary_config`.

Cases:

- loading missing config returns empty map
- saving config creates new `AISettings` row
- saving again updates existing row

**Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/aisettings -run OpenNotebook -v`

Expected: fail because helpers do not exist yet.

**Step 3: Write minimal implementation**

Add helpers parallel to the existing summary config helpers:

```go
const openNotebookConfigKey = "open_notebook_config"

func LoadOpenNotebookConfig() (map[string]interface{}, *models.AISettings, error)
func SaveOpenNotebookConfig(config map[string]interface{}, description string) error
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/aisettings -run OpenNotebook -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend-go/internal/platform/aisettings/config_store.go backend-go/internal/platform/aisettings/config_store_test.go
git commit -m "feat(ai): add open-notebook config storage helpers"
```

---

### Task 3: Build the open-notebook platform client

**Files:**
- Create: `backend-go/internal/platform/opennotebook/client.go`
- Create: `backend-go/internal/platform/opennotebook/client_test.go`

**Step 1: Write the failing test**

Use `httptest.Server` to verify:

- request method/path/body are correct
- auth header is sent
- success response is parsed
- non-200 returns a readable error

**Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/opennotebook -v`

Expected: fail because package does not exist.

**Step 3: Write minimal implementation**

Create a client like:

```go
type Client struct {
  BaseURL string
  APIKey string
  Model string
  HTTPClient *http.Client
}

type SummarizeDigestRequest struct {
  Title string `json:"title"`
  Content string `json:"content"`
  TargetNotebook string `json:"target_notebook,omitempty"`
  PromptMode string `json:"prompt_mode,omitempty"`
}

type SummarizeDigestResponse struct {
  SummaryMarkdown string `json:"summary_markdown"`
  RemoteID string `json:"remote_id,omitempty"`
  RemoteURL string `json:"remote_url,omitempty"`
}

func (c *Client) SummarizeDigest(req SummarizeDigestRequest) (*SummarizeDigestResponse, error)
```

Keep endpoint path isolated in one place so it is easy to change.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/opennotebook -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend-go/internal/platform/opennotebook/client.go backend-go/internal/platform/opennotebook/client_test.go
git commit -m "feat(platform): add open-notebook client"
```

---

### Task 4: Add digest-domain config and send handlers

**Files:**
- Modify: `backend-go/internal/domain/digest/handler.go`
- Modify: `backend-go/internal/app/router.go`
- Test: `backend-go/internal/domain/digest/handler_test.go`

**Step 1: Write the failing test**

Add handler tests for:

- `GET /api/digest/open-notebook/config`
- `PUT /api/digest/open-notebook/config`
- `POST /api/digest/open-notebook/daily?date=2026-03-10`

Mock or inject the open-notebook client behavior.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/digest -run OpenNotebook -v`

Expected: fail because routes and handlers do not exist.

**Step 3: Write minimal implementation**

In `handler.go`:

- add request/response structs for open-notebook config
- load/save config via `aisettings`
- add send handler that:
  - parses `type` and `date`
  - calls `buildPreview()`
  - sends `preview.Markdown` to client
  - returns `digest_type`, `anchor_date`, `source_markdown`, `summary_markdown`, `remote_id`, `remote_url`

Prefer function-level dependency injection for the client constructor so tests stay simple.

**Step 4: Register routes**

Add in `backend-go/internal/app/router.go`:

```go
digestGroup.GET("/open-notebook/config", digestdomain.GetOpenNotebookConfig)
digestGroup.PUT("/open-notebook/config", digestdomain.UpdateOpenNotebookConfig)
digestGroup.POST("/open-notebook/:type", digestdomain.SendDigestToOpenNotebook)
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/domain/digest -run OpenNotebook -v`

Expected: PASS

**Step 6: Commit**

```bash
git add backend-go/internal/domain/digest/handler.go backend-go/internal/domain/digest/handler_test.go backend-go/internal/app/router.go
git commit -m "feat(digest): add open-notebook config and send endpoints"
```

---

### Task 5: Extend digest settings UI for open-notebook config

**Files:**
- Modify: `front/app/features/digest/components/DigestSettings.vue`

**Step 1: Write the failing test**

If a frontend test setup exists for this feature, add a component test.
If not, document manual verification and skip automated test for this task.

Manual cases:

- loads open-notebook config on mount
- saves config successfully
- shows error notice on failure

**Step 2: Write minimal implementation**

Add a new settings group under existing Feishu / Obsidian blocks.

Fields:

- enabled
- base URL
- API key
- model
- target notebook
- auto daily
- auto weekly
- write back to Obsidian

Use existing visual patterns in this file.

**Step 3: Run manual verification**

Run: `pnpm dev`

Verify in browser:

- config can be loaded
- config can be saved
- notice text is correct

**Step 4: Commit**

```bash
git add front/app/features/digest/components/DigestSettings.vue
git commit -m "feat(digest): add open-notebook settings UI"
```

---

### Task 6: Add manual send action and result panel in digest view

**Files:**
- Modify: `front/app/features/digest/components/DigestListView.vue`
- Optional helper: `front/app/features/digest/components/DigestDetail.vue`

**Step 1: Write the failing test**

If no frontend tests exist here, define manual acceptance checks.

Checks:

- clicking send calls the endpoint for the selected type/date
- success result renders summary markdown
- loading state prevents duplicate clicks
- failure shows a notice

**Step 2: Write minimal implementation**

In `DigestListView.vue`:

- add local state for open-notebook result and loading flag
- add button near existing `runNow` action
- call `digestApi.sendToOpenNotebook(selectedType, selectedDates[selectedType])`
- render returned `summary_markdown` in a dedicated panel

Do not replace existing digest detail.
Show the second-pass result alongside it.

**Step 3: Run manual verification**

Run: `pnpm dev`

Verify:

- daily and weekly both send correctly
- returned content updates when date changes
- old result is cleared when the type/date changes

**Step 4: Commit**

```bash
git add front/app/features/digest/components/DigestListView.vue front/app/features/digest/components/DigestDetail.vue
git commit -m "feat(digest): add manual send-to-open-notebook flow"
```

---

### Task 7: Verify backend and frontend integration

**Files:**
- No code changes required unless issues are found

**Step 1: Run backend tests**

Run: `go test ./...`

Expected: all relevant backend tests pass

**Step 2: Run frontend type/build checks**

Run: `pnpm build`

Expected: successful build

**Step 3: Run manual end-to-end check**

Run backend and frontend locally.

Verify complete flow:

- open digest page
- save open-notebook config
- preview a daily digest
- click send
- see second-pass summary result

**Step 4: Fix any failures and rerun checks**

Repeat until clean.

**Step 5: Commit**

```bash
git add .
git commit -m "test(digest): verify open-notebook integration flow"
```

---

### Task 8: Optional follow-up for auto-send hooks

**Files:**
- Modify: `backend-go/internal/domain/digest/handler.go`
- Modify: `backend-go/internal/domain/digest/scheduler.go`

**Step 1: Write the failing test**

Add tests for auto-send behavior after a successful digest run, guarded by config flags.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/digest -run AutoSendOpenNotebook -v`

**Step 3: Write minimal implementation**

- in `RunDigestNow()`, call open-notebook after digest success when enabled
- later mirror the same behavior in scheduler daily/weekly methods
- do not block Feishu/Obsidian success if open-notebook fails

**Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/digest -run AutoSendOpenNotebook -v`

**Step 5: Commit**

```bash
git add backend-go/internal/domain/digest/handler.go backend-go/internal/domain/digest/scheduler.go
git commit -m "feat(digest): add optional auto-send to open-notebook"
```
