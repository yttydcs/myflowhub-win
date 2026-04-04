# Plan - Win P0 Authoring Closure

## Workflow Information
- Repo: `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps`
- Branch: `feat/win-p0-authoring-gaps`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`:
  - repository root has no `guide.md`
  - `docs/README.md` read as entry point
  - `$m-autoflow` initialization rules confirmed before any implementation
- base/worktree confirmation:
  - control repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - control branch: `main`
  - control repo has user-owned dirty changes and remains control-plane only:
    - `go.mod`
    - `myflowhub-mcp.exe`
  - active execution worktree: `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps`
  - active execution branch: `feat/win-p0-authoring-gaps`
  - this workflow only modifies `MyFlowHub-Win`

### Stage 1 - Requirements Analysis
#### Goal
- Close the remaining Win-side P0 authoring gaps so the editor and project deploy UI can express the existing server `flow` contract without forcing users back to raw JSON for core scenarios.

#### Scope
- Must:
  - add Win-side `loop_item / loop_index` source support for store, inspectors, field binding dialog, and `foreach.body` session wiring
  - add Win-side authoring for `retry_backoff_ms`, `max_active_runs`, and `dedup_window_ms`
  - add Win-side `subflow` self-call / recursion guard on strict save paths
  - update tests and Win requirement/spec docs
- Optional:
  - tighten wording and help text where needed for the new fields
- Not in scope:
  - no runtime / server behavior changes
  - no recursive transform tree visual editor
  - no recursive nested `foreach.body` visual session

#### Use Cases
- author a `transform` or `compose` inside `foreach.body` that reads the current item or index directly
- configure fixed retry backoff for a node without editing JSON by hand
- configure `max_active_runs` at flow level from Win
- configure trigger dedup for `event` / `var_changed` flows from Win
- reject obviously invalid `subflow` self-calls before the payload reaches the executor

#### Functional Requirements
- `FlowBindingSourceKind`, `FlowSourceDraft`, and `FlowInputBindingDraft` must support `loop_item` and `loop_index`
- top-level inspector, body inspector, field binding dialog, and body window wiring must expose the new source kinds consistently
- node authoring must support `retry_backoff_ms`
- flow authoring must support `max_active_runs`
- trigger authoring must support `dedup_window_ms` for `event` and `var_changed`
- strict save must reject:
  - `loop_item / loop_index` outside `foreach.body`
  - `loop_index` with a path
  - negative numeric control fields
  - `dedup_window_ms > 0` on `interval` or `cron`
  - `subflow.flow_id == current flow_id`
  - any recursion chain that can be detected from the locally known graph context

#### Non-functional Requirements
- keep `foreachBodyJson` as the single persisted truth for body graphs
- keep changes minimal and explicit; no silent fallback or silent coercion
- preserve existing form/json mode gating semantics
- avoid widening the write surface beyond the P0 closure

#### Inputs / Outputs
- Inputs:
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
- Outputs:
  - updated Win authoring code
  - updated Win tests
  - updated Win requirements/specs
  - this worktree `plan.md`
  - stage `4` archive under `docs/change/`

#### Edge Cases
- `loop_item / loop_index` are only valid inside `foreach.body`
- `loop_index` does not support JSON Pointer path access
- `retry_backoff_ms`, `max_active_runs`, and `dedup_window_ms` must reject negative values
- `cron` and `interval` do not allow `dedup_window_ms > 0`
- `subflow` must fail when it points to the current `flow_id`
- if a loaded advanced spec contains extra fields beyond current form coverage, Win must keep existing explicit failure / JSON-only behavior

#### Acceptance Criteria
- Win can round-trip the P0 fields and source kinds through form mode and strict save
- `foreach.body` can author `loop_item / loop_index` via the same UI family as other bindings
- invalid `subflow` self-reference is rejected before save / deploy
- targeted Vitest coverage exists for store, inspector, window, and project deploy paths
- Win docs reflect the new stable authoring contract

#### Risks
- `frontend/src/stores/flow.ts` is the central coupling point for source normalization, strict save, payload export, and tests
- source-kind UI currently exists in multiple components and can drift if not updated coherently
- trigger authoring is modeled in both `flow.ts` and `flowProjects.ts`; partial updates could cause deploy drift

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- Keep the current Win architecture and patch the missing contract paths in place rather than introducing a new abstraction layer or state model.
- Extend the existing source contract once and thread it through:
  - store normalization
  - strict serialization
  - top-level inspector
  - body inspector
  - field binding dialog
  - body session window wiring
- Extend existing node / flow / trigger draft models for the new control fields instead of inventing parallel config objects.
- Add `subflow` guard in the strict save path so Win catches obvious invalid references early while still leaving the server as the final enforcement layer.

#### Alternatives Considered
- Patch each duplicated switch / select independently:
  - rejected because the current problem is cross-layer drift; repeating that pattern would likely reintroduce gaps
- Large shared-component refactor for all inspector source-kind UI:
  - rejected because the change surface is too large for a P0 closure task
- Leave these capabilities JSON-only:
  - rejected because the user explicitly asked to close P0 authoring gaps

#### Module Responsibilities
- `frontend/src/stores/flow.ts`
  - extend graph state, flow state, normalization, strict save, and payload export
  - validate new node / flow / trigger fields
  - implement Win-side `subflow` self-reference / recursion checks where locally knowable
- `frontend/src/stores/flowProjects.ts`
  - extend project trigger draft normalization, wire conversion, and deploy persistence
- `frontend/src/pages/Flow.vue`
  - expose UI for deploy trigger dedup fields
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - expose top-level node and flow authoring for new fields and sources
- `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - expose body-session authoring for new fields and sources
- `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - expose the new source kinds for schema-driven field binding
- `frontend/src/windows/FlowEditorWindow.vue`
  - carry body-session source-kind defaults and binding wiring through to persisted `foreachBodyJson`
- docs / tests
  - keep Win requirement/spec truth aligned with implementation
  - prove round-trip and failure behavior

#### Data / Call Flow
1. Load an existing flow or project payload into Win draft state.
2. Normalize node / flow / trigger fields into `flow.ts` or `flowProjects.ts`.
3. User edits the fields in inspectors, body window, field binding dialog, or deploy UI.
4. Strict save path converts draft state back into `flow` wire payload.
5. Win rejects invalid local states before calling backend bindings.

#### Interface Drafts
- `FlowBindingSourceKind += "loop_item" | "loop_index"`
- `FlowNodeDraft += retryBackoffMs`
- `FlowDraftSnapshot`, `FlowState`, and `FlowPayload` gain flow-level `maxActiveRuns`
- `FlowState` and `FlowTriggerDraft` gain `dedupWindowMs`
- node wire output gains optional `retry_backoff_ms`
- flow wire output gains optional `max_active_runs`
- trigger wire output gains optional `dedup_window_ms` for `event` and `var_changed`

#### Error Handling and Safety
- reject all negative numeric control values
- reject `loop_item / loop_index` in invalid scope
- reject `loop_index` path usage
- reject unsupported trigger dedup combinations
- reject direct `subflow` self-reference and locally detectable recursion chains
- preserve explicit JSON-only fallback rules when form mode cannot represent loaded advanced payloads

#### Performance and Testing Strategy
- avoid new stores or duplicated persisted state
- keep `foreachBodyJson` as the only persisted body-graph source
- targeted tests:
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/stores/flowProjects.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`

#### Extensibility Design Points
- once the source contract is fully closed, future body-scoped sources can reuse the same path
- flow / trigger control fields remain plain numeric options, leaving room for later richer policy objects if needed
- keeping validation in strict save paths avoids coupling UI rendering rules too tightly to future runtime expansion

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Current state:
  - core `flow` runtime and protocol already support the target P0 contract
  - Win already supports most advanced-node authoring and `foreach.body` visual sessions
  - Win still lacks:
    - `loop_item / loop_index`
    - `retry_backoff_ms / max_active_runs / dedup_window_ms`
    - front-end `subflow` recursion guard
- Project goal:
  - make Win-side authoring match the current P0 server contract closely enough that the remaining gaps are no longer on the critical path for normal use

#### Docs Governance Routing Decision
- using `$m-docs` to validate routing and impact
- docs tree is already present and healthy; no bootstrap required
- canonical destinations:
  - stable Win authoring truth -> `docs/requirements/flow-editor-visual-form.md`
  - stable Win technical contract -> `docs/specs/flow-editor-visual-form.md`
  - workflow control document -> worktree-root `plan.md`
  - completed result archive -> `docs/change/YYYY-MM-DD_topic.md`
- Requirements impact: `updated`
- Specs impact: `updated`
- Lessons impact: `none` unless implementation reveals a reusable new pitfall

#### Related Requirements / Specs / Lessons
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps\docs\requirements\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- Related lessons:
  - `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps\docs\lessons\flow-body-spec-mode-normalization.md`

#### Executable Task List
- [x] `WIN-P0-1` close source-kind support for `loop_item / loop_index` across store and editor surfaces
- [x] `WIN-P0-2` add Win authoring for `retry_backoff_ms`, `max_active_runs`, and `dedup_window_ms`
- [x] `WIN-P0-3` add Win-side `subflow` self-call / recursion guard on strict save paths
- [x] `WIN-P0-4` update docs and targeted tests for the new stable authoring contract

#### Task Details
##### `WIN-P0-1` - Close Loop Source Authoring
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps\plan.md`
- Goal:
  - let Win author, read, and save `loop_item / loop_index` in the same authoring surfaces that already support other source kinds
- Files / Modules:
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write Set:
  - the files above
- Acceptance:
  - source dropdowns expose `loop_item / loop_index` where valid
  - strict save emits correct wire and rejects invalid scope / path combinations
  - `foreach.body` session can persist these sources back into parent JSON
- Test Points:
  - store round-trip and error tests
  - top-level/body inspector rendering tests
  - window-level body wiring test
- Rollback:
  - revert the files above and associated tests

##### `WIN-P0-2` - Add Run Control and Trigger Authoring
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps\plan.md`
- Goal:
  - expose `retry_backoff_ms`, `max_active_runs`, and `dedup_window_ms` in Win authoring and persist them correctly
- Files / Modules:
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
- Write Set:
  - the files above
- Acceptance:
  - node backoff and flow active-run cap round-trip through Win
  - trigger dedup round-trips in both direct flow authoring and project deploy authoring
  - invalid numeric values and unsupported trigger combinations fail explicitly
- Test Points:
  - `flow.test.ts`
  - `flowProjects.test.ts`
  - targeted UI tests where needed
- Rollback:
  - revert the files above and associated tests

##### `WIN-P0-3` - Add Frontend Subflow Guard
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps\plan.md`
- Goal:
  - reject obvious invalid `subflow` references in Win before save / deploy
- Files / Modules:
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write Set:
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
- Acceptance:
  - direct self-call is rejected
  - recursion chain is rejected when it can be determined from the current local graph / locally known project graphs
  - no false positive against valid unrelated subflows
- Test Points:
  - targeted `flow.test.ts` strict-save failures
- Rollback:
  - revert `frontend/src/stores/flow.ts` and related tests

##### `WIN-P0-4` - Sync Docs and Regression Coverage
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps\plan.md`
- Goal:
  - align Win requirement/spec docs with the completed P0 authoring contract and prove the behavior with targeted tests
- Files / Modules:
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - targeted test files under `frontend/src/...`
- Write Set:
  - the files above
- Acceptance:
  - docs mention the new stable contract and boundaries accurately
  - targeted Vitest suite passes
- Test Points:
  - run the planned Vitest subset and record results for stage `4`
- Rollback:
  - revert the docs and tests above

#### Dependencies
- `WIN-P0-1` and `WIN-P0-2` both depend on consistent updates in `frontend/src/stores/flow.ts`
- `WIN-P0-3` depends on the final `subflow` serialization path in `flow.ts`
- `WIN-P0-4` depends on the final implementation result so docs and tests match the shipped behavior

#### Risks and Notes
- `frontend/src/stores/flow.ts` is a shared write set for three tasks; implementation order must stay deliberate
- existing body-mode lesson about `specEditorMode` drift must be respected when touching body-session defaults
- targeted tests are the primary verification path; full frontend build may still be affected by existing environment issues unrelated to this workflow

#### Parallelism Assessment
- Safe implementation parallelism is limited.
- Reason:
  - `WIN-P0-1`, `WIN-P0-2`, and `WIN-P0-3` all converge on the same central write set:
    - `frontend/src/stores/flow.ts`
    - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
    - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
    - `frontend/src/stores/flow.test.ts`
  - splitting those tasks across workers would create high merge-conflict and semantic drift risk
- Stage `3.2` decision:
  - main agent keeps implementation local
- Stage `3.3` opportunity:
  - after the local patch lands, a read-only sub-agent review / verification pass is safe and useful for quality

#### Issue List
- none

阻塞：否
进入 3.2
