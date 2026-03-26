# Plan - Win Flow Detail Merge Regression

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-flow-detail-merge-regression`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-flow-detail-merge-regression`
- Current Stage: `4 archived / awaiting workflow end confirmation`

## Stage Records

### Initialization
- guide.md:
  - `D:\project\MyFlowHub3\guide.md` exists and requires Chinese commit messages, Chrome DevTools for UI tests when relevant, server docs under `repo\MyFlowHub-Server\docs`, and all worktrees under `D:\project\MyFlowHub3\worktrees`.
- base/worktree confirmation:
  - Control-plane repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - Active execution worktree: `D:\project\MyFlowHub3\worktrees\fix-win-flow-detail-merge-regression`
  - Base branch: `main`
  - Current main HEAD already contains merge result `4f273ad` that breaks `wails generate module` and `go test ./...`.

### Stage 1 - Requirements Analysis
#### Goal
- Repair the current Win flow detail merge regression so Wails bindings and Go validation pass again without changing the stable `flow.detail` JSON contract.

#### Scope
- Must:
  - restore a single canonical definition source for `actionDetail`, `actionDetailResp`, `DetailReq`, and `DetailResp`
  - keep `FlowService.Detail` / `DetailSimple` signatures aligned with the stable spec
  - restore `wails generate module` and `GOWORK=off` Go validation
  - keep the frontend-facing `flow.detail` payload shape unchanged
- Optional:
  - add a focused regression test if needed to lock the canonical type path
  - refresh generated Wails bindings if the fix changes generated output
- Not in scope:
  - changing runtime `flow.detail` protocol behavior
  - upgrading shared proto
  - expanding detail UI behavior or store semantics

#### Use Cases
- Developers run `wails generate module` after backend changes and need bindings generation to succeed.
- Developers run `GOWORK=off; go test ./...` and expect the repo to compile end to end.
- Frontend callers continue to invoke `window.go.flow.FlowService.DetailSimple(...)` with the same JSON fields.

#### Functional Requirements
1. `internal/services/flow` must compile without duplicate declarations.
2. `Detail` and `DetailSimple` must continue using Win-local typed payloads rather than missing shared proto detail types.
3. `extractCodeMsg(...)` must still recognize `*DetailResp`.
4. The canonical local detail type source must match the stable spec and current frontend expectations.
5. `wails generate module` must succeed from the worktree with `GOWORK=off`.

#### Non-functional Requirements
- Keep the change set minimal and localized.
- Preserve current JSON field names and optionality unless a build-only fix is impossible.
- Avoid introducing a second competing definition source for detail payloads.
- Use `GOWORK=off` for Go and Wails verification in the worktree.

#### Inputs / Outputs
- Inputs:
  - current broken merge state on `main`
  - existing local detail types in `internal/services/flow/detail_types.go`
  - existing flow detail tests and Wails bindings generation
- Outputs:
  - deduplicated flow detail type ownership
  - passing Go tests / build
  - passing Wails bindings generation
  - archived change notes for the merge regression

#### Edge Cases
- keeping the wrong `DetailResp` shape and silently drifting from the spec
- fixing compile errors but leaving `frontend/wailsjs` stale
- running validation without `GOWORK=off` and masking the true state

#### Acceptance Criteria
1. `$env:GOWORK='off'; go test ./internal/services/flow` passes.
2. `$env:GOWORK='off'; go test ./...` passes.
3. `$env:GOWORK='off'; wails generate module` passes.
4. `internal/services/flow/service.go` no longer declares duplicate detail action constants or detail payload types.
5. The resulting `DetailReq` / `DetailResp` source remains consistent with `docs/specs/flow-editor-run-detail.md`.

#### Risks
- Choosing the wrong canonical `DetailResp` variant could reintroduce Wails or frontend shape drift.
- Generated bindings may change and need to be included deliberately.
- Future merges can reintroduce the regression unless the lesson/index is updated with a merge-regression cue.

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- Keep `internal/services/flow/detail_types.go` as the canonical source for detail payloads and action constants, and remove the duplicate `Detail*` declarations that were reintroduced into `internal/services/flow/service.go` by a later merge path.

#### Alternatives Considered
- Option A: keep `detail_types.go` canonical and delete duplicate declarations from `service.go`
  - Pros:
    - matches the documented fix from `2026-03-26_win-flow-detail-bindings`
    - keeps a single exported local payload definition source
    - smallest safe repair
  - Cons:
    - still leaves long-term local/shared proto convergence as future work
- Option B: delete `detail_types.go` and keep the inlined `service.go` types
  - Pros:
    - one less file
  - Cons:
    - conflicts with the archived fix and stable spec clarification
    - drops `ExecutorNode` and changes the `Node` model away from the documented canonical local type
- Decision:
  - adopt Option A

#### Module Responsibilities
- `internal/services/flow/detail_types.go`
  - canonical local `DetailReq` / `DetailResp`
  - canonical `detail` / `detail_resp` action constants
- `internal/services/flow/service.go`
  - transport, validation, await, and error handling logic
  - references canonical local detail types without redefining them
- `internal/services/flow/service_test.go`
  - regression guard for detail validation and code/msg extraction
- generated `frontend/wailsjs/*`
  - refreshed only if Wails generation updates generated bindings

#### Data / Call Flow
1. frontend calls `window.go.flow.FlowService.DetailSimple(...)`
2. `FlowService.DetailSimple` delegates to `Detail`
3. `Detail` validates fields, encodes `actionDetail`, and awaits `actionDetailResp`
4. response unmarshals into canonical `DetailResp`
5. `extractCodeMsg` handles `*DetailResp`
6. Wails module generation reflects the deduplicated public method signature and models

#### Interface Drafts
- Keep:
  - `Detail(ctx, sourceID, targetID uint32, req DetailReq) (DetailResp, error)`
  - `DetailSimple(sourceID, targetID uint32, req DetailReq) (DetailResp, error)`
- Keep canonical payload fields from `detail_types.go` unchanged.

#### Error Handling and Safety
- Do not change validation rules for `req_id`, `flow_id`, or `node_id`.
- Do not change transport error wrapping.
- Fail fast on duplicate-definition compile issues through tests and Wails generation.

#### Performance and Testing Strategy
- Minimal code-motion fix only.
- Validation:
  - `$env:GOWORK='off'; go test ./internal/services/flow`
  - `$env:GOWORK='off'; go test ./...`
  - `$env:GOWORK='off'; wails generate module`
  - `git diff --check`
- If generated bindings change, inspect only the expected `flow.Detail*` binding/model sections.

#### Extensibility Design Points
- Keep detail payload ownership isolated in one file so future shared proto convergence only touches one source.
- Extend the lesson doc with merge-regression symptoms so future integration work checks for duplicate local type declarations as well as missing proto symbols.

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - restore the current broken `main` merge state to a compiling state without changing stable `flow.detail` behavior.
- Current State:
  - `main` fails on duplicate declarations between `internal/services/flow/detail_types.go` and `internal/services/flow/service.go`.
  - root cause points to merge result `4f273ad` combining the archived fix with earlier inlined detail types from `31cf9ee`.

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: none
- Specs impact: none
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\fix-win-flow-detail-merge-regression\docs\requirements\flow-editor-run-detail.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\fix-win-flow-detail-merge-regression\docs\specs\flow-editor-run-detail.md`
- Related lessons:
  - `D:\project\MyFlowHub3\worktrees\fix-win-flow-detail-merge-regression\docs\lessons\wails-binding-proto-drift.md`
  - `D:\project\MyFlowHub3\worktrees\fix-win-flow-detail-merge-regression\docs\change\2026-03-26_win-flow-detail-bindings.md`
- Stable truth routing:
  - stable behavior and contract remain in `docs/requirements` and `docs/specs`
  - this regression fix result goes to `docs/change`
  - reusable merge-regression troubleshooting goes to `docs/lessons`

#### Related Requirements / Specs / Lessons
- Requirement:
  - `docs/requirements/flow-editor-run-detail.md`
- Spec:
  - `docs/specs/flow-editor-run-detail.md`
- Lessons:
  - `docs/lessons/wails-binding-proto-drift.md`
- Related prior archive:
  - `docs/change/2026-03-26_win-flow-detail-bindings.md`

#### Executable Task List
- [x] FIX-1 remove duplicate detail action/type declarations from `service.go` and keep `detail_types.go` canonical
- [x] TEST-1 run Go, Wails, and diff validation; refresh generated bindings only if generation updates files
- [x] REVIEW-1 perform 3.3 review against requirements, architecture, stability, and test coverage
- [x] ARCHIVE-1 archive the regression fix and update the lesson/index if merge-regression cues are worth preserving

#### Task Details
##### FIX-1 - Canonical Detail Type Deduplication
- Owner:
  - main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\fix-win-flow-detail-merge-regression`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\fix-win-flow-detail-merge-regression\todo.md`
- Goal:
  - remove merge-reintroduced duplicate detail declarations from `service.go` and keep the archived `detail_types.go` fix as the canonical source
- Files / Modules:
  - `internal/services/flow/service.go`
  - `internal/services/flow/detail_types.go`
- Write Set:
  - `internal/services/flow/service.go`
  - `internal/services/flow/detail_types.go` only if clarification is required
- Acceptance:
  - `internal/services/flow` compiles without duplicate declarations
  - `FlowService.Detail` / `DetailSimple` keep using local typed payloads
  - no stable JSON contract drift is introduced
- Test Points:
  - `$env:GOWORK='off'; go test ./internal/services/flow`
  - inspect resulting `DetailResp` use path
- Rollback:
  - revert this task's diff and return to the previous merge state

##### TEST-1 - Validation And Binding Refresh
- Owner:
  - main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\fix-win-flow-detail-merge-regression`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\fix-win-flow-detail-merge-regression\todo.md`
- Goal:
  - verify the repo and Wails bindings path recover end to end
- Files / Modules:
  - generated `frontend/wailsjs/*` if changed by Wails generation
- Write Set:
  - generated bindings only if the tool updates them
- Acceptance:
  - `GOWORK=off` Go tests pass
  - `GOWORK=off` Wails module generation passes
  - generated binding changes, if any, are limited to expected detail models/signatures
- Test Points:
  - `$env:GOWORK='off'; go test ./...`
  - `$env:GOWORK='off'; wails generate module`
  - `git diff --check`
- Rollback:
  - revert generated bindings if generation proves unrelated or unstable

##### ARCHIVE-1 - Change And Lesson Archive
- Owner:
  - main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\fix-win-flow-detail-merge-regression`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\fix-win-flow-detail-merge-regression\todo.md`
- Goal:
  - archive this merge regression and preserve reusable troubleshooting cues
- Files / Modules:
  - `docs/change/YYYY-MM-DD_*.md`
  - `docs/lessons/wails-binding-proto-drift.md`
  - `docs/lessons/README.md` if needed
- Write Set:
  - archive and lesson files only
- Acceptance:
  - archive documents the merge regression, verification, and rollback
  - lesson mentions duplicate local-type merge regressions if warranted
- Test Points:
  - manual doc consistency check
- Rollback:
  - revert the new archive and lesson edits

#### Dependencies
- `FIX-1` before `TEST-1`
- `TEST-1` before `REVIEW-1` and `ARCHIVE-1`

#### Risks and Notes
- `frontend/wailsjs` in `main` may currently be stale because bindings generation is blocked.
- The canonical source decision must follow the prior archived fix and current spec, not whichever declaration is shorter.
- Validation must be run from the worktree with `GOWORK=off`.

#### Parallelism Assessment
- No sub-agent dispatch.
- Reason:
  - write set is tightly coupled around `internal/services/flow/service.go`, canonical type ownership, and a single validation pipeline
  - this fix is small enough that delegation would add coordination overhead without reducing critical-path time

### Stage 3.3 - Code Review
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过

### Stage 4 - Archive
- `docs/change/2026-03-26_win-flow-detail-merge-regression.md` 已创建
- `docs/lessons/wails-binding-proto-drift.md` 已补充 merge 回归线索
- `docs/change/README.md`、`docs/lessons/README.md` 已更新

#### Issue List
- none

阻塞：否
进入 3.2
