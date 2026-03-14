# Topic Graph 年轮树干改版计划

## TL;DR

> **Quick Summary**: 保留现有 `3d-force-graph` 交互能力，重做 `topic-graph` 的页面层级、视觉语言和历史表达，让当前焦点主题成为可感知的“树干主轴”，同时补齐 Playwright 浏览器级关键路径验证。
>
> **Deliverables**:
> - `front/app/features/topic-graph/` 视觉与布局重构
> - 焦点主题“树干化”表达与历史脉络增强
> - Playwright 基础配置与 topic-graph 关键路径用例
> - 可执行验证命令：typecheck、Vitest、build、Playwright
>
> **Estimated Effort**: Medium
> **Parallel Execution**: YES - 3 waves
> **Critical Path**: Task 1 -> Task 2 -> Task 4 -> Task 8

---

## Context

### Original Request
用户认为 `front/app/features/topic-graph/` 当前前端样式过于丑陋，希望改成更有科技感且具备历史脉络感的实现，并明确要求当前聚焦主题具有“树干 / 主轴”的存在感。

### Interview Summary
**Key Discussions**:
- 视觉主隐喻已确定为“年轮树干”
- 必须保留原有 3D 图谱能力，不改成 2D 或静态页
- 历史表达需要从“信息存在”升级为“结构可感知”
- 需要新增 Playwright
- Playwright 范围限定为 topic-graph 关键路径
- 自动化测试策略已确定为“测试后补”

**Research Findings**:
- `front/app/features/topic-graph/components/TopicGraphPage.vue` 负责数据请求、状态编排、模态预览与子组件组合
- `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue` 已承担 3D 图谱引擎边界，当前活跃节点主要通过球体、halo、边高亮和 camera focus 表达
- `front/app/features/topic-graph/components/TopicGraphSidebar.vue` 目前是偏浅色纸面风格，与主图深色技术风不一致
- `front/app/features/topic-graph/components/TopicGraphFooterPanels.vue` 的历史表达只有条形图，叙事强度不足
- `front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts` 是最合适的衍生状态边界，可承载 trunk/history 的可测试派生逻辑

### Metis Review
**Identified Gaps** (addressed):
- 需要显式锁定范围：不改后端 API、不换图谱库、不扩成全站 redesign
- 需要为 Playwright 增加稳定选择器与 graph-ready 标记，避免 WebGL 场景下测试不稳定
- 需要把“树干感”转成可验证的 UI 结构结果，而不是主观审美描述
- 需要覆盖空状态、弱数据、长文本、慢加载和 resize 等边界情况

---

## Work Objectives

### Core Objective
在不改变 topic-graph 核心数据流和 3D 引擎的前提下，重构页面为统一的深色科技叙事界面，让当前焦点主题成为视觉主轴，并让历史与关联内容围绕该主轴展开。

### Concrete Deliverables
- 改造 `front/app/features/topic-graph/components/TopicGraphPage.vue`
- 改造 `front/app/features/topic-graph/components/TopicGraphHeader.vue`
- 改造 `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue`
- 改造 `front/app/features/topic-graph/components/TopicGraphSidebar.vue`
- 改造 `front/app/features/topic-graph/components/TopicGraphFooterPanels.vue`
- 扩展 `front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts` 及其测试
- 新增 Playwright 配置、脚本和 `topic-graph` 关键路径测试

### Definition of Done
- [ ] `/topics` 页面呈现统一的年轮树干科技风格，而非深浅割裂的拼贴式风格
- [ ] 当前焦点主题在页面主结构中具备明确主轴地位，且能通过 DOM 状态或结构标记被验证
- [ ] 历史脉络不再仅以简单条形图出现，而是形成可读的时间/层级叙事结构
- [ ] `pnpm test:unit -- app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts` 通过
- [ ] `pnpm exec nuxi typecheck` 通过
- [ ] `pnpm build` 通过
- [ ] Playwright topic-graph 关键路径测试通过

### Must Have
- 保留 `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue` 的 3D 图谱能力
- 保留 `/topics` 入口和现有主要信息块：图谱、热点主题、历史、相关文章、文章预览、站内/外入口
- 为 Playwright 提供稳定选择器或 readiness 标记
- 对空状态、慢加载、无历史数据、无相关文章状态保持安全回退

### Must NOT Have (Guardrails)
- 不替换 `3d-force-graph`、`three`、`three-spritetext`
- 不引入后端接口改造，除非执行阶段证明当前 payload 无法表达所需历史结构
- 不把本次工作扩展为全站视觉系统重构
- 不把“历史感”做成纯装饰背景，必须承载真实数据或状态
- 不新增与 topic-graph 无关的 Playwright 全站覆盖

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — 所有验收都必须由代理执行，不能要求用户手动点击确认“好看了”。

### Test Decision
- **Infrastructure exists**: YES
- **Automated tests**: Tests-after
- **Framework**: Vitest + Playwright
- **Playwright target**: Chromium only (default for first pass; can expand later)

### QA Policy
每个任务都必须包含 agent-executed QA 场景，并在需要时通过稳定选择器、data attributes、显式空状态文案或 readiness 状态进行验证。

- **Frontend/UI structure**: 通过 Playwright 验证关键结构、交互和错误态
- **Derived state / transforms**: 通过 Vitest 验证 view-model 派生逻辑
- **Build safety**: 通过 `pnpm exec nuxi typecheck` 与 `pnpm build` 验证
- **Evidence**: Playwright 失败时保存 trace/screenshot 到 `.sisyphus/evidence/`

---

## Execution Strategy

### Parallel Execution Waves

Wave 1 (Start Immediately — boundaries + derived state):
├── Task 1: Stabilize topic-graph test hooks and state contract [quick]
├── Task 2: Extend view-model for trunk/history emphasis [quick]
├── Task 3: Redesign hero shell and left rail hierarchy [visual-engineering]
└── Task 4: Rework 3D canvas emphasis while preserving engine [visual-engineering]

Wave 2 (After Wave 1 — supporting panels, parallel):
├── Task 5: Redesign right sidebar into unified dark-tech lineage rail [visual-engineering]
├── Task 6: Rebuild footer/history panels around chronology narrative [visual-engineering]
└── Task 7: Add Playwright config and frontend e2e command [quick]

Wave 3 (After Wave 2 — integrated verification):
└── Task 8: Add topic-graph Playwright critical-path coverage and finalize verification [unspecified-high]

Wave FINAL (After ALL tasks — independent review, 4 parallel):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real browser QA replay (unspecified-high)
└── Task F4: Scope fidelity check (deep)

Critical Path: Task 1 -> Task 2 -> Task 4 -> Task 8
Parallel Speedup: ~55% faster than sequential
Max Concurrent: 4

### Dependency Matrix

- **1**: None -> 4, 7, 8
- **2**: None -> 4, 5, 6, 8
- **3**: None -> 5, 6, 8
- **4**: 1, 2 -> 8
- **5**: 2, 3 -> 8
- **6**: 2, 3 -> 8
- **7**: 1 -> 8
- **8**: 1, 2, 3, 4, 5, 6, 7 -> F1-F4

### Agent Dispatch Summary

- **Wave 1**: **4** — T1 → `quick`, T2 → `quick`, T3 → `visual-engineering`, T4 → `visual-engineering`
- **Wave 2**: **3** — T5 → `visual-engineering`, T6 → `visual-engineering`, T7 → `quick`
- **Wave 3**: **1** — T8 → `unspecified-high`
- **FINAL**: **4** — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [x] 1. Stabilize topic-graph verification hooks and page state contract

  **What to do**:
  - Add stable DOM markers for the topic-graph page shell, active topic area, history area, article preview trigger/container, and graph-ready state.
  - Ensure the page exposes deterministic empty/loading/error states without changing the route or API flow.
  - Keep selectors local to topic-graph components; do not introduce global testing abstractions yet.

  **Must NOT do**:
  - Do not alter backend requests or response shapes.
  - Do not add generic app-wide test helpers beyond what Playwright setup requires.

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: small but important structural changes for testability and state exposure.
  - **Skills**: [`vue-best-practices`, `vue-testing-best-practices`]
    - `vue-best-practices`: keep changes aligned with Vue 3 component conventions.
    - `vue-testing-best-practices`: make selectors and state markers test-friendly.
  - **Skills Evaluated but Omitted**:
    - `ui-ux-pro-max`: visual design is not the primary goal of this task.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3)
  - **Blocks**: 4, 7, 8
  - **Blocked By**: None

  **References**:
  - `front/app/features/topic-graph/components/TopicGraphPage.vue:185` - Main page shell where stable page-level markers should live.
  - `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue:189` - Canvas container where a graph-ready or mounted marker can be surfaced.
  - `front/app/features/topic-graph/components/TopicGraphSidebar.vue:40` - Sidebar already contains distinct empty/loading/detail branches; expose them predictably.
  - `front/app/pages/topics.vue:1` - Route entry confirms `/topics` should remain unchanged.

  **Acceptance Criteria**:
  - [ ] Topic graph page exposes stable selectors/readiness markers for Playwright.
  - [ ] Empty/loading/detail/article-preview states are distinguishable by DOM state, not timing guesses.
  - [ ] `pnpm exec nuxi typecheck` passes after selector/state additions.

  **QA Scenarios**:
  ```
  Scenario: topic-graph route exposes stable markers on load
    Tool: Playwright
    Preconditions: Frontend app running and `/topics` accessible
    Steps:
      1. Open `/topics`
      2. Wait for the page-level topic-graph root marker
      3. Wait for the graph-ready marker or explicit loading marker transition
      4. Assert the active-topic panel container and history panel container exist
    Expected Result: Route loads with deterministic markers that tests can wait on
    Failure Indicators: No root marker, no graph-ready/loading state, missing active-topic/history containers
    Evidence: .sisyphus/evidence/task-1-route-markers.png

  Scenario: topic-graph empty or pre-focus state is explicit
    Tool: Playwright
    Preconditions: Page loaded before any manual topic change
    Steps:
      1. Open `/topics`
      2. Inspect the sidebar state container
      3. Assert it contains either focused-topic data or a known fallback/empty-state message
    Expected Result: Sidebar never renders as an ambiguous blank container
    Failure Indicators: Empty container, unstable text, missing state wrapper
    Evidence: .sisyphus/evidence/task-1-pre-focus-state.png
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-1-route-markers.png`
  - [ ] `.sisyphus/evidence/task-1-pre-focus-state.png`

  **Commit**: YES
  - Message: `test(front): stabilize topic-graph verification hooks`
  - Files: `front/app/features/topic-graph/components/TopicGraphPage.vue`, `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue`, `front/app/features/topic-graph/components/TopicGraphSidebar.vue`
  - Pre-commit: `pnpm exec nuxi typecheck`

- [x] 2. Extend view-model for trunk emphasis and chronology-friendly derived state

  **What to do**:
  - Expand `buildTopicGraphViewModel.ts` to derive explicit active/focus-friendly metadata that downstream UI can use for trunk styling, chronology labels, history emphasis, and spotlight grouping.
  - Add or update Vitest coverage for the new derived behavior, including empty payload safety and edge filtering continuity.
  - Keep this logic deterministic and independent from runtime DOM concerns.

  **Must NOT do**:
  - Do not move rendering logic or API calls into the view-model layer.
  - Do not introduce `any` or couple tests to exact CSS class names.

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: this is derived-state logic with targeted tests.
  - **Skills**: [`vue-best-practices`, `vue-testing-best-practices`]
    - `vue-best-practices`: maintain typed boundaries for frontend state.
    - `vue-testing-best-practices`: shape clean Vitest assertions around UI-facing derivations.
  - **Skills Evaluated but Omitted**:
    - `ui-ux-pro-max`: not necessary for pure derivation logic.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3)
  - **Blocks**: 4, 5, 6, 8
  - **Blocked By**: None

  **References**:
  - `front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts:35` - Existing derivation boundary for stats, node sizing, and featured nodes.
  - `front/app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts:5` - Existing local test style and baseline behaviors to preserve.
  - `front/app/features/topic-graph/components/TopicGraphPage.vue:36` - View model consumption point; new derived fields should support page composition without extra ad-hoc logic.

  **Acceptance Criteria**:
  - [ ] View-model exposes the additional derived state needed for trunk/chronology presentation.
  - [ ] Empty payloads and weak-edge cases remain safe.
  - [ ] `pnpm test:unit -- app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts` passes.

  **QA Scenarios**:
  ```
  Scenario: derived state supports trunk-focused presentation
    Tool: Bash (pnpm/vitest)
    Preconditions: Updated test file includes new trunk/chronology assertions
    Steps:
      1. Run `pnpm test:unit -- app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts`
      2. Assert the suite reports PASS with zero failures
      3. Inspect that tests cover hero/topic ordering, edge filtering, and new chronology/trunk metadata
    Expected Result: Derived state is verified by targeted unit tests
    Failure Indicators: Failing assertions, missing test coverage for new metadata
    Evidence: .sisyphus/evidence/task-2-vitest.txt

  Scenario: empty payload remains safe after derivation changes
    Tool: Bash (pnpm/vitest)
    Preconditions: Empty-state test case exists in the same spec
    Steps:
      1. Run `pnpm test:unit -- app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts -t "empty state"`
      2. Assert the empty-state test passes
    Expected Result: New derivations do not break the existing empty fallback behavior
    Failure Indicators: Empty-state test failure or thrown runtime error
    Evidence: .sisyphus/evidence/task-2-empty-state.txt
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-2-vitest.txt`
  - [ ] `.sisyphus/evidence/task-2-empty-state.txt`

  **Commit**: YES
  - Message: `feat(front): derive trunk and chronology state for topic graph`
  - Files: `front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts`, `front/app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts`
  - Pre-commit: `pnpm test:unit -- app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts`

- [x] 3. Redesign hero shell and left rail into a unified trunk-first stage

  **What to do**:
  - Rework the page shell and header so the experience reads as one cohesive dark-tech observatory surface instead of separate cards.
  - Make the top hero, left rail stats, and hot-topic navigation support the “current trunk / outer rings” hierarchy.
  - Preserve existing controls (date/type/refresh) and existing information blocks while improving typography, spacing, and contrast.

  **Must NOT do**:
  - Do not remove date/type/refresh controls.
  - Do not create a generic SaaS dashboard look or purple-biased palette.

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: heavy UI composition and visual hierarchy work.
  - **Skills**: [`frontend-design`, `ui-ux-pro-max`, `vue-best-practices`]
    - `frontend-design`: shape a production-quality, non-generic visual system.
    - `ui-ux-pro-max`: enforce contrast, hierarchy, and responsive UI quality.
    - `vue-best-practices`: keep component updates idiomatic.
  - **Skills Evaluated but Omitted**:
    - `web-artifacts-builder`: not needed for existing Nuxt component edits.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2)
  - **Blocks**: 5, 6, 8
  - **Blocked By**: None

  **References**:
  - `front/app/features/topic-graph/components/TopicGraphPage.vue:185` - Existing stage and left-rail layout to reshape without changing page responsibilities.
  - `front/app/features/topic-graph/components/TopicGraphHeader.vue:29` - Hero and control toolbar layout to redesign while preserving interactions.
  - `front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts:66` - Hero label/subline data source that should still drive the top narrative.

  **Acceptance Criteria**:
  - [ ] Header, left rail, and stage background share one coherent visual language.
  - [ ] Hot topics and stats reinforce focus hierarchy instead of reading as detached chips/cards.
  - [ ] `pnpm exec nuxi typecheck` passes after shell/header refactor.

  **QA Scenarios**:
  ```
  Scenario: topic-graph shell preserves controls and structure after redesign
    Tool: Playwright
    Preconditions: Frontend app running with updated shell styles
    Steps:
      1. Open `/topics`
      2. Assert the graph type toggle, date input, and refresh button are visible
      3. Assert the hot-topics region and stats region are visible within the same main stage
    Expected Result: Redesign improves presentation without removing primary controls or overview blocks
    Failure Indicators: Missing controls, hidden hot-topics region, broken stage hierarchy
    Evidence: .sisyphus/evidence/task-3-shell-structure.png

  Scenario: narrower viewport keeps hero and rail readable
    Tool: Playwright
    Preconditions: Frontend app running
    Steps:
      1. Set viewport to 1024x900
      2. Open `/topics`
      3. Assert there is no clipped toolbar and no horizontal overflow on the main stage
    Expected Result: Layout remains readable at a narrower desktop/tablet width
    Failure Indicators: Horizontal scrolling, clipped controls, overlapped sections
    Evidence: .sisyphus/evidence/task-3-shell-responsive.png
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-3-shell-structure.png`
  - [ ] `.sisyphus/evidence/task-3-shell-responsive.png`

  **Commit**: NO
  - Message: `feat(front): redesign topic-graph shell and canvas emphasis`
  - Files: `front/app/features/topic-graph/components/TopicGraphPage.vue`, `front/app/features/topic-graph/components/TopicGraphHeader.vue`
  - Pre-commit: `pnpm exec nuxi typecheck`

- [x] 4. Rework 3D canvas emphasis so the active topic reads as the trunk axis

  **What to do**:
  - Preserve the current `3d-force-graph` runtime and click/focus behavior, but strengthen the active node’s presence through materials, link hierarchy, camera feel, label treatment, and scene atmosphere.
  - Ensure the active topic visually dominates as the trunk/main axis while adjacent nodes/edges behave like secondary branches.
  - Keep graph resize behavior and one-hop relationship highlighting intact or stronger.

  **Must NOT do**:
  - Do not replace the graph engine or remove click-to-focus behavior.
  - Do not make the scene so decorative that labels and relations become less readable.

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: this is visual interaction tuning inside an existing 3D scene.
  - **Skills**: [`frontend-design`, `ui-ux-pro-max`, `vue-best-practices`]
    - `frontend-design`: helps shape a deliberate visual language in the graph scene.
    - `ui-ux-pro-max`: ensures contrast and focus hierarchy remain legible.
    - `vue-best-practices`: keeps client-only component behavior stable.
  - **Skills Evaluated but Omitted**:
    - `webapp-testing`: useful later in verification, but not central to implementation guidance here.

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 1 sequential tail
  - **Blocks**: 8
  - **Blocked By**: 1, 2

  **References**:
  - `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue:27` - Graph setup boundary; preserve imports, instance creation, and force-graph lifecycle.
  - `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue:74` - Node object builder where active/focused visual treatment is currently defined.
  - `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue:122` - Link width/color logic where trunk-vs-branch distinction can be reinforced.
  - `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue:152` - Camera focus behavior that should continue guiding users toward the active trunk.
  - `front/app/features/topic-graph/utils/buildTopicGraphViewModel.ts:57` - Graph-derived metadata source for stronger focus semantics.

  **Acceptance Criteria**:
  - [ ] Active topic remains clickable and focusable, with stronger visual dominance than surrounding nodes.
  - [ ] One-hop relation emphasis still works and is visually clearer than non-focused edges.
  - [ ] Canvas remains responsive to mount and resize without losing graph rendering.
  - [ ] `pnpm exec nuxi typecheck` passes after canvas updates.

  **QA Scenarios**:
  ```
  Scenario: clicking a topic updates focused graph state and active detail region
    Tool: Playwright
    Preconditions: `/topics` page loaded and graph-ready marker visible
    Steps:
      1. Open `/topics`
      2. Click a hot-topic control or graph-linked topic trigger
      3. Assert the active-topic region updates with the selected topic label
      4. Assert the graph container exposes an active-node marker/value that changed accordingly
    Expected Result: Active topic change drives both graph focus and surrounding detail state
    Failure Indicators: Click does nothing, active label unchanged, graph state marker unchanged
    Evidence: .sisyphus/evidence/task-4-active-focus.png

  Scenario: graph remains mounted after viewport resize
    Tool: Playwright
    Preconditions: `/topics` page loaded
    Steps:
      1. Open `/topics`
      2. Resize viewport from 1440x960 to 1100x900
      3. Assert the graph container remains visible and the page still shows an active topic/detail state
    Expected Result: Resize does not collapse or blank the canvas area
    Failure Indicators: Canvas disappears, graph-ready marker vanishes permanently, page sections overlap
    Evidence: .sisyphus/evidence/task-4-resize.png
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-4-active-focus.png`
  - [ ] `.sisyphus/evidence/task-4-resize.png`

  **Commit**: NO
  - Message: `feat(front): redesign topic-graph shell and canvas emphasis`
  - Files: `front/app/features/topic-graph/components/TopicGraphCanvas.client.vue`, `front/app/features/topic-graph/components/TopicGraphPage.vue`
  - Pre-commit: `pnpm exec nuxi typecheck`

- [x] 5. Redesign right sidebar into a dark-tech lineage rail

  **What to do**:
  - Rework the sidebar to align visually with the main stage instead of the current light paper panel.
  - Preserve current content responsibilities: active topic header, related articles, related topics.
  - Make the focused topic feel like the trunk origin and related content feel like branches or linked strata.

  **Must NOT do**:
  - Do not remove article preview entry points.
  - Do not collapse all information into one long undifferentiated card.

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: this is a dense information layout redesign task.
  - **Skills**: [`frontend-design`, `ui-ux-pro-max`, `vue-best-practices`]
    - `frontend-design`: for stronger information architecture and visual rhythm.
    - `ui-ux-pro-max`: for readability and accessible contrast in a dark UI.
    - `vue-best-practices`: for clean component updates.
  - **Skills Evaluated but Omitted**:
    - `web-artifacts-builder`: not needed for local component work.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 6, 7)
  - **Blocks**: 8
  - **Blocked By**: 2, 3

  **References**:
  - `front/app/features/topic-graph/components/TopicGraphSidebar.vue:40` - Existing state branches and content groups that must remain present.
  - `front/app/features/topic-graph/components/TopicGraphPage.vue:263` - Sticky right-rail integration point that must still host the sidebar cleanly.
  - `front/app/features/topic-graph/components/TopicGraphPage.vue:269` - Article preview modal flow; sidebar article triggers must continue to feed this.

  **Acceptance Criteria**:
  - [ ] Sidebar matches the main dark-tech visual language.
  - [ ] Focused topic, related articles, and related topics remain clearly separated and readable.
  - [ ] Article buttons still open the preview flow via the existing event contract.

  **QA Scenarios**:
  ```
  Scenario: sidebar shows focused topic and related article list after topic selection
    Tool: Playwright
    Preconditions: `/topics` page loaded with at least one selectable topic
    Steps:
      1. Open `/topics`
      2. Trigger a topic selection
      3. Assert the sidebar focused-topic heading is visible
      4. Assert the related-articles region exists and is not empty when data is available
    Expected Result: Sidebar updates coherently around the selected topic
    Failure Indicators: Sidebar remains blank, headings missing, articles region not rendered
    Evidence: .sisyphus/evidence/task-5-sidebar-focus.png

  Scenario: related article click still opens preview modal
    Tool: Playwright
    Preconditions: Sidebar contains at least one article card/button
    Steps:
      1. Click the first related article trigger
      2. Wait for the article preview modal container
      3. Assert modal body renders and close button exists
    Expected Result: Existing article preview flow remains intact after sidebar redesign
    Failure Indicators: No modal, modal opens empty, click not wired
    Evidence: .sisyphus/evidence/task-5-article-preview.png
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-5-sidebar-focus.png`
  - [ ] `.sisyphus/evidence/task-5-article-preview.png`

  **Commit**: NO
  - Message: `feat(front): redesign topic-graph shell and canvas emphasis`
  - Files: `front/app/features/topic-graph/components/TopicGraphSidebar.vue`, `front/app/features/topic-graph/components/TopicGraphPage.vue`
  - Pre-commit: `pnpm exec nuxi typecheck`

- [x] 6. Rebuild footer panels around chronology instead of simple bars

  **What to do**:
  - Redesign `TopicGraphFooterPanels.vue` so the history section conveys layered chronology/lineage rather than a generic bar list.
  - Preserve app-links and external-links, but align them visually with the same trunk/rings narrative.
  - Use real history data from `detail.history`; if chronology is sparse, provide a graceful fallback rather than decorative fake structure.

  **Must NOT do**:
  - Do not invent fake time data.
  - Do not remove app-link or external-link sections.

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: this task mixes data display redesign with narrative layout.
  - **Skills**: [`frontend-design`, `ui-ux-pro-max`, `vue-best-practices`]
    - `frontend-design`: for giving history a distinctive, non-generic visual treatment.
    - `ui-ux-pro-max`: for keeping data display legible and responsive.
    - `vue-best-practices`: for safe updates in a focused component.
  - **Skills Evaluated but Omitted**:
    - `vue-testing-best-practices`: browser verification is more central here than component-level test design.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 7)
  - **Blocks**: 8
  - **Blocked By**: 2, 3

  **References**:
  - `front/app/features/topic-graph/components/TopicGraphFooterPanels.vue:19` - Current footer grid and section split to preserve semantically.
  - `front/app/features/topic-graph/components/TopicGraphFooterPanels.vue:21` - History section entry point that needs stronger chronology expression.
  - `front/app/features/topic-graph/components/TopicGraphSidebar.vue:43` - Empty/fallback state wording style that can guide graceful no-data treatment.

  **Acceptance Criteria**:
  - [ ] History is presented as chronology/lineage structure rather than only simple bars.
  - [ ] App-links and external-links remain functional and visible.
  - [ ] Empty-history state remains graceful and explicit.

  **QA Scenarios**:
  ```
  Scenario: history panel renders chronology structure for a focused topic
    Tool: Playwright
    Preconditions: `/topics` page loaded and a topic with history data selected
    Steps:
      1. Open `/topics`
      2. Select a topic
      3. Assert the history panel container is visible
      4. Assert it renders multiple history entries/segments with labels and counts when history exists
    Expected Result: History is clearly structured and data-bearing after redesign
    Failure Indicators: Missing panel, flat empty block despite available history, unreadable collapsed content
    Evidence: .sisyphus/evidence/task-6-history-chronology.png

  Scenario: no-history state remains explicit and safe
    Tool: Playwright
    Preconditions: Use a topic with no history if available, or mock/fixture state in test harness if setup supports it
    Steps:
      1. Navigate to the no-history state
      2. Assert the history section renders a known fallback message/container
    Expected Result: Absence of history never produces a broken or misleading chronology UI
    Failure Indicators: Empty card, broken layout, fake data rendering
    Evidence: .sisyphus/evidence/task-6-history-empty.png
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-6-history-chronology.png`
  - [ ] `.sisyphus/evidence/task-6-history-empty.png`

  **Commit**: NO
  - Message: `feat(front): redesign topic-graph shell and canvas emphasis`
  - Files: `front/app/features/topic-graph/components/TopicGraphFooterPanels.vue`
  - Pre-commit: `pnpm exec nuxi typecheck`

- [x] 7. Add Playwright config, scripts, and minimal frontend e2e baseline

  **What to do**:
  - Install and configure Playwright for the frontend workspace.
  - Add an e2e script in `front/package.json` and baseline config/spec directory structure.
  - Keep the setup small and topic-graph-oriented; avoid building a full cross-app framework.

  **Must NOT do**:
  - Do not add a large helper abstraction layer before the first useful spec exists.
  - Do not configure a wide cross-browser matrix in this pass.

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: focused test-infra setup with limited scope.
  - **Skills**: [`vue-testing-best-practices`, `webapp-testing`]
    - `vue-testing-best-practices`: aligns browser test setup with frontend verification conventions.
    - `webapp-testing`: useful for validating realistic browser-driven workflows.
  - **Skills Evaluated but Omitted**:
    - `frontend-design`: no visual design work here.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 6)
  - **Blocks**: 8
  - **Blocked By**: 1

  **References**:
  - `front/package.json:5` - Existing frontend scripts where `test:e2e` should be added cleanly.
  - `front/app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts:1` - Existing test presence confirms local test culture but not e2e conventions.
  - `front/app/pages/topics.vue:1` - Stable route target for the first e2e suite.

  **Acceptance Criteria**:
  - [ ] Frontend workspace contains a working Playwright config.
  - [ ] `front/package.json` exposes an e2e command for topic-graph verification.
  - [ ] Playwright can launch Chromium against the frontend app in local/agent execution.

  **QA Scenarios**:
  ```
  Scenario: Playwright command boots and discovers the topic-graph spec
    Tool: Bash
    Preconditions: Playwright dependencies installed in `front/`
    Steps:
      1. Run the configured install command if needed for browsers
      2. Run `pnpm test:e2e -- --list` or equivalent discovery command
      3. Assert the topic-graph spec is discovered
    Expected Result: Playwright infrastructure is wired and the spec is visible to the runner
    Failure Indicators: Unknown script, missing config, no specs found
    Evidence: .sisyphus/evidence/task-7-playwright-discovery.txt

  Scenario: Playwright can start Chromium for the frontend app
    Tool: Bash
    Preconditions: Frontend dev server or configured webServer target available
    Steps:
      1. Run the configured topic-graph Playwright command
      2. Assert Chromium launches and the runner starts executing
    Expected Result: Browser automation environment is usable
    Failure Indicators: Browser missing, startup timeout, config crash
    Evidence: .sisyphus/evidence/task-7-playwright-runner.txt
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-7-playwright-discovery.txt`
  - [ ] `.sisyphus/evidence/task-7-playwright-runner.txt`

  **Commit**: YES
  - Message: `test(front): add topic-graph playwright setup`
  - Files: `front/package.json`, `front/playwright.config.*`, `front/tests/e2e/*`
  - Pre-commit: `pnpm test:e2e -- --list`

- [x] 8. Add topic-graph Playwright critical-path spec and run full verification

  **What to do**:
  - Implement the first `topic-graph` Playwright spec covering route load, graph readiness, topic selection, active-topic detail update, chronology panel rendering, and article preview modal flow.
  - Include at least one edge/failure scenario: explicit empty/pre-focus state, no uncaught console errors on load, or narrow viewport resilience.
  - Run the full frontend verification chain and fix any issues caused by the redesign or test setup.

  **Must NOT do**:
  - Do not write pixel-perfect assertions against the WebGL canvas.
  - Do not broaden the suite to unrelated pages or cross-browser matrices in this pass.

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: this task integrates UI behavior, reliability, and end-to-end verification.
  - **Skills**: [`vue-testing-best-practices`, `webapp-testing`, `verification-before-completion`]
    - `vue-testing-best-practices`: for robust browser assertions against Vue stateful UI.
    - `webapp-testing`: for realistic route-level interaction execution.
    - `verification-before-completion`: ensures final claims are evidence-based.
  - **Skills Evaluated but Omitted**:
    - `frontend-design`: design work should already be complete; this task is verification-heavy.

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3
  - **Blocks**: F1, F2, F3, F4
  - **Blocked By**: 1, 2, 3, 4, 5, 6, 7

  **References**:
  - `front/app/pages/topics.vue:1` - Stable route target for the spec.
  - `front/app/features/topic-graph/components/TopicGraphPage.vue:185` - End-to-end page composition that should now expose deterministic test markers.
  - `front/app/features/topic-graph/components/TopicGraphSidebar.vue:64` - Related article triggers that must still open preview flows.
  - `front/app/features/topic-graph/components/TopicGraphFooterPanels.vue:19` - Chronology/app-links/external-links region that must remain visible.
  - `front/package.json:5` - Script entry point for running the final verification chain.

  **Acceptance Criteria**:
  - [ ] Topic-graph Playwright spec covers the agreed critical path.
  - [ ] The spec includes at least one failure/edge scenario.
  - [ ] `pnpm test:unit -- app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts` passes.
  - [ ] `pnpm exec nuxi typecheck` passes.
  - [ ] `pnpm build` passes.
  - [ ] `pnpm test:e2e -- tests/e2e/topic-graph.spec.ts` passes.

  **QA Scenarios**:
  ```
  Scenario: topic-graph critical path works end-to-end
    Tool: Playwright
    Preconditions: Frontend app available through configured Playwright webServer or manual dev server
    Steps:
      1. Open `/topics`
      2. Wait for topic-graph root marker and graph-ready marker
      3. Assert hero, hot-topics region, active-topic region, and history region are visible
      4. Select a topic via the hot-topics control or graph-related trigger
      5. Assert the active-topic label changes to the chosen topic
      6. Assert the chronology/history region updates and remains visible
      7. Click a related article trigger
      8. Assert the article preview modal opens and close it successfully
    Expected Result: The redesigned page supports the core user journey without regressions
    Failure Indicators: Graph never becomes ready, topic selection does not update detail state, chronology missing, article modal broken
    Evidence: .sisyphus/evidence/task-8-topic-graph-critical-path.zip

  Scenario: route load is resilient in a narrower viewport with clean console
    Tool: Playwright
    Preconditions: Frontend app running
    Steps:
      1. Set viewport to 1100x900
      2. Open `/topics`
      3. Capture console messages during load
      4. Assert no uncaught error-level console messages appear
      5. Assert the topic-graph root, sidebar, and history region are still visible
    Expected Result: The redesigned route remains stable and usable in a narrower layout
    Failure Indicators: Error console logs, collapsed layout, missing major regions
    Evidence: .sisyphus/evidence/task-8-narrow-viewport.zip
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-8-topic-graph-critical-path.zip`
  - [ ] `.sisyphus/evidence/task-8-narrow-viewport.zip`

  **Commit**: YES
  - Message: `test(front): add topic-graph playwright coverage`
  - Files: `front/tests/e2e/topic-graph.spec.ts`, related Playwright fixtures/config updates, and any touched topic-graph files required to make tests pass
  - Pre-commit: `pnpm test:unit -- app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts && pnpm exec nuxi typecheck && pnpm build && pnpm test:e2e -- tests/e2e/topic-graph.spec.ts`

---

## Final Verification Wave

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the finished implementation and verify each Must Have / Must NOT Have item against the plan. Confirm the preserved 3D boundary, preserved route, preserved major content blocks, and presence of Playwright critical-path coverage.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `pnpm exec nuxi typecheck`, `pnpm test:unit`, `pnpm build`, and the Playwright command. Review changed frontend files for dead selectors, broken responsive assumptions, vague naming, commented-out code, and fragile test waits.
  Output: `Typecheck [PASS/FAIL] | Unit [PASS/FAIL] | Build [PASS/FAIL] | E2E [PASS/FAIL] | VERDICT`

- [x] F3. **Real Browser QA** — `unspecified-high`
  Execute the exact Playwright scenarios from Task 8, plus an explicit no-topic/empty-state run and a narrow viewport run. Save traces/screenshots into `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Responsive [PASS/FAIL] | Console [CLEAN/ISSUES] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  Compare final diff against this plan and ensure no unrelated shell/page/backend code was modified beyond the allowed topic-graph and Playwright setup scope.
  Output: `Tasks [N/N compliant] | Scope [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **1**: `test(front): stabilize topic-graph verification hooks` — topic-graph page/component selectors and readiness markers
- **2**: `feat(front): derive trunk and chronology state for topic graph` — view-model and unit tests
- **3**: `feat(front): redesign topic-graph shell and canvas emphasis` — page, header, canvas, sidebar, footer UI work
- **4**: `test(front): add topic-graph playwright coverage` — Playwright config, scripts, and spec

---

## Success Criteria

### Verification Commands
```bash
pnpm test:unit -- app/features/topic-graph/utils/buildTopicGraphViewModel.test.ts
pnpm exec nuxi typecheck
pnpm build
pnpm test:e2e -- tests/e2e/topic-graph.spec.ts
```

### Final Checklist
- [ ] 3D topic graph capability preserved
- [ ] Active topic has trunk-like structural emphasis
- [ ] History presentation is chronology-forward rather than simple bars
- [ ] Sidebar and footer visually align with the main dark-tech language
- [ ] Topic-graph critical path covered by Playwright
- [ ] No unrelated frontend shell/backend scope creep
