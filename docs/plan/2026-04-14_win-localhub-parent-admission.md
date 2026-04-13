# Plan - Win Local Hub Approval Join UX

## Workflow Information
- Repo: `MyFlowHub-Win`
- Branch: `feat/win-localhub-approval-ui`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui`
- Current Stage: `4 Change Archive`

## Stage Records

### Initialization
- guide.md:
  - Read `D:\project\MyFlowHub3\guide.md`.
  - Confirmed repo workflow rules: worktrees must live under `D:\project\MyFlowHub3\worktrees\`; implementation must not happen in the main repo path; commit messages should use Chinese text with optional English prefixes.
- base/worktree confirmation:
  - Control-plane repo path `D:\project\MyFlowHub3\repo\MyFlowHub-Win` is dirty and must remain read-only for this workflow.
  - Created dedicated worktree `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui` on branch `feat/win-localhub-approval-ui` from `main`.
  - `MyFlowHub-Server` was reviewed as an external contract source only; no Server worktree is planned unless the approved design expands into Server CLI changes.

### Stage 1 - Requirements Analysis
#### Goal
- Improve `Session -> Local Hub` so a user can connect a fresh local `hub_server` to a parent network that requires approval without dropping to a manual shell/env workflow.

#### Scope
- Must:
  - Add an explicit Local Hub input path for parent self-registration identity (`self-id`).
  - Add an explicit Local Hub input path for the one-time parent join permit required by approval-mode parent networks.
  - Keep the existing host/port/parent/auth/extra-args flow working for current users.
  - Validate invalid combinations before launch and fail with explicit UI/backend errors.
  - Keep the implementation bounded to `MyFlowHub-Win` if current `hub_server` env/flag contracts already suffice.
- Optional:
  - Improve Local Hub page copy so approval-mode requirements are visible before launch.
  - Add a small status hint for the approval-mode bootstrap path in the Local Hub page.
- Not in scope:
  - Changing auth admission semantics in `MyFlowHub-Server`.
  - Changing permit issuance / approval pages.
  - Adding new long-lived Server CLI flags if a Win-only launch/env solution is sufficient.
  - Broader Local Hub redesign unrelated to approval-mode parent join.

#### Use Cases
- A user has a fresh local Hub and a parent Hub with `require_approval=true`, plus a one-time permit issued for the local Hub device.
- A user wants to start the local Hub from the Win page by filling structured fields instead of remembering extra args and hidden env vars.
- A user wants existing non-approval Local Hub flows to behave exactly as before.

#### Functional Requirements
- The Local Hub page must expose a structured `self-id` field instead of requiring `extraArgs` for that value.
- The Local Hub launch flow must support passing a one-time `parent join permit` to `hub_server`.
- The Local Hub backend must map the structured fields to the current Server contract:
  - `self-id` -> current `hub_server` launch input
  - `parent join permit` -> current `hub_server` launch input
- Validation must reject at least:
  - join permit present while parent link is disabled
  - join permit present while parent address is empty
  - join permit present while `self-id` is empty
- Existing `extraArgs` must remain available as an advanced escape hatch.
- Start / Restart must use the new launch inputs consistently.

#### Non-functional Requirements
- Prefer the smallest safe change surface in Win.
- Do not force a Server code change when the current env-based contract already supports the feature.
- Avoid exposing a one-time join permit through a permanent or overly broad interface unless the plan explicitly chooses that tradeoff.
- Add regression tests for changed launch/validation behavior.
- Preserve backward compatibility for stored Local Hub configs that do not contain the new fields.

#### Inputs / Outputs
- Inputs:
  - Local Hub page form values
  - current installed binary path
  - existing Local Hub config store
  - optional approval-mode permit issued by authority
- Outputs:
  - persisted Local Hub config for stable fields
  - launch-time process args/env for `hub_server`
  - Local Hub page status / validation feedback

#### Edge Cases
- Existing config with no new fields should still load and start.
- Parent link may be enabled for non-approval use; `self-id` should not silently break that path.
- One-time join permits are sensitive and stale-prone; the handling model must be deliberate.
- Local Hub may be restarted after the permit has already been consumed; the UX must not suggest that the same permit is reusable.

#### Acceptance Criteria
- A user can configure a fresh Local Hub for an approval-mode parent network directly from `Session -> Local Hub` without manual shell env setup.
- Starting or restarting with an invalid approval-mode combination fails before process launch with an explicit error.
- Existing Local Hub users who do not use approval-mode parent join see no regression in install/start/stop/restart behavior.
- Targeted tests cover the new validation and launch mapping.

#### Risks
- There is currently no stable Local Hub requirement/spec document in the Win repo; this workflow should add stable docs rather than leaving the feature only in `docs/change`.
- Join permit persistence is a security/UX tradeoff and should not be locked in casually.
- There are currently no focused Local Hub tests; a small refactor may be needed to make launch mapping testable.

#### Issue List
- Decision needed at plan approval: should `parent join permit` be treated as a one-shot non-persistent launch input, or as a persisted Local Hub config field?

### Stage 2 - Architecture Design
#### Overall Solution
- Recommended option: implement the improvement in `MyFlowHub-Win` only.
  - Persist stable launch fields such as `self-id`.
  - Treat `parent join permit` as a launch-time override rather than a durable config secret.
  - Map the page inputs to the existing Server contract by injecting process env values for `HUB_SELF_ID` and `HUB_PARENT_JOIN_PERMIT`.
- Rationale:
  - `hub_server` already supports `HUB_SELF_ID`, `HUB_PARENT_JOIN_PERMIT`, and `HUB_WORKDIR` via env.
  - `cmd/hub_server` does not expose a `-parent-join-permit` flag today.
  - A Win-only solution avoids widening the public Server CLI/API surface for a feature the runtime already supports.

#### Alternatives Considered
- Alternative A: persist `parent join permit` in Local Hub config and inject it on every Start.
  - Smaller implementation.
  - Worse secret hygiene and more stale-permit confusion.
- Alternative B: add new Server CLI flags such as `-parent-join-permit`.
  - More uniform “all fields are flags”.
  - Cross-repo scope increase; exposes permit via process args; not justified while env support already exists.
- Recommended choice:
  - Use Win-only env injection.
  - Keep permit non-persistent by default unless the user explicitly prefers the simpler persisted model.

#### Module Responsibilities
- `docs/requirements/*` / `docs/specs/*`
  - Add stable truth for Local Hub approval-mode join behavior and launch contract.
- `internal/services/localhub/types.go`
  - Extend Local Hub config / launch input types.
- `internal/services/localhub/service.go`
  - Validate new fields.
  - Build launch args/env.
  - Keep existing install/start/stop/restart behavior compatible.
- `frontend/src/pages/LocalHub.vue`
  - Expose the new fields and wire them into save/start/restart.
  - Surface validation and approval-mode guidance.
- Tests
  - Cover launch mapping and validation.

#### Data / Call Flow
- UI loads `Snapshot()`.
- User edits Local Hub config fields.
- `SaveConfig(...)` persists stable values.
- `Start(...)` / `Restart(...)` receives any one-shot launch override, validates the combined launch request, then:
  - keeps existing args for `-addr`, `-node-id`, `-parent`, auth flags, and `extraArgs`
  - adds env overrides for `HUB_SELF_ID` and `HUB_PARENT_JOIN_PERMIT`
  - starts `hub_server`
- UI reloads snapshot and shows current run state.

#### Interface Drafts
- Win service:
  - Extend Local Hub config with a stable `selfId` field.
  - Add a launch-request shape or equivalent override path so `Start` / `Restart` can receive a non-persistent `parentJoinPermit`.
- Frontend:
  - Add `Self ID` field under `Hub Params`.
  - Add `Parent Join Permit` field with explicit one-time / approval-mode wording.
- Docs:
  - New requirement/spec leaf docs for Local Hub approval-mode join support.

#### Error Handling and Safety
- Reject approval-mode combinations that cannot work before spawning the process.
- Avoid logging the permit value directly in Win logs or user-facing status text.
- Do not broaden Server CLI surface unless the approved design changes.

#### Performance and Testing Strategy
- No meaningful performance impact is expected.
- Add focused Go tests around Local Hub validation and launch construction.
- Add a small frontend test only if needed to cover the new save/start payload path; prioritize backend launch correctness first.

#### Extensibility Design Points
- Keep `extraArgs` as an escape hatch for less common flags.
- Model one-shot launch overrides separately from persisted config so future sensitive launch inputs can reuse the same path.
- Preserve room for optional future `workdir` structuring without forcing it into this workflow.

#### Issue List
- User approval needed on the recommended permit-handling model:
  - recommended: non-persistent one-shot launch input
  - simpler but weaker: persisted config field

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - Make `Session -> Local Hub` directly usable for approval-mode parent join.
- Current state:
  - The page already supports parent link config and advanced args.
  - `self-id` is only available indirectly through `extraArgs`.
  - `parent join permit` is not represented in the Win launch model even though `hub_server` already accepts `HUB_PARENT_JOIN_PERMIT`.

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Canonical routing:
  - Stable user-facing behavior -> `docs/requirements/`
  - Stable technical launch contract -> `docs/specs/`
  - Workflow execution -> this worktree-root `plan.md`
  - Completed results -> `docs/change/`
- Requirements impact: `add`
- Specs impact: `add`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui\docs\requirements\localhub-parent-admission.md` (new, planned)
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui\docs\specs\localhub-parent-admission.md` (new, planned)
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md` (existing upstream auth/runtime contract)
- Related lessons:
  - `none currently`

#### Executable Task List
- `DOC-1` add stable Local Hub approval-mode requirement/spec docs and update indexes.
- `IMPL-1` extend LocalHub backend launch model and validation for approval-mode parent join.
- `IMPL-2` extend the Local Hub page to expose the new fields and launch path.
- `QA-1` run targeted Go/frontend validation.

#### Task Details
##### DOC-1 - Local Hub Stable Docs
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui\plan.md`
- Goal:
  - Add stable Win-side requirement/spec truth for Local Hub approval-mode parent join support.
- Files / Modules:
  - `docs/requirements/README.md`
  - `docs/specs/README.md`
  - `docs/requirements/localhub-parent-admission.md` (new)
  - `docs/specs/localhub-parent-admission.md` (new)
- Write Set:
  - docs only
- Acceptance:
  - Stable requirement/spec docs exist and are indexed.
- Test Points:
  - manual docs review
- Rollback:
  - revert the docs commit

##### IMPL-1 - LocalHub Launch Contract
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui\plan.md`
- Goal:
  - Add structured Local Hub launch support for `self-id` and approval-mode parent join permit without expanding Server scope.
- Files / Modules:
  - `internal/services/localhub/types.go`
  - `internal/services/localhub/service.go`
  - `internal/services/localhub/*test*.go` (new or updated)
- Write Set:
  - Local Hub backend only
- Acceptance:
  - backend validates unsupported approval-mode combinations
  - backend can inject the correct env values for launch
  - current launch behavior remains backward compatible
- Test Points:
  - Go tests for validation
  - Go tests for launch args/env construction
- Rollback:
  - revert Local Hub backend changes

##### IMPL-2 - Local Hub Page UX
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui\plan.md`
- Goal:
  - Expose approval-mode Local Hub fields in the page and wire them to the backend launch path.
- Files / Modules:
  - `frontend/src/pages/LocalHub.vue`
  - `frontend/src/i18n/messages/session.ts`
  - optional focused frontend test file if needed
- Write Set:
  - Local Hub frontend only
- Acceptance:
  - user can fill `self-id` and permit from the page
  - page surfaces validation / one-time permit guidance
  - existing Local Hub actions still work
- Test Points:
  - targeted frontend test if practical
  - manual start/restart flow review
- Rollback:
  - revert Local Hub frontend changes

##### QA-1 - Targeted Validation
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui\plan.md`
- Goal:
  - Verify the bounded Local Hub approval-mode changes without running broad unrelated suites.
- Files / Modules:
  - touched backend/frontend tests only
- Write Set:
  - none unless test fixes are needed
- Acceptance:
  - changed tests pass
  - manual verification steps are documented for the Local Hub page
- Test Points:
  - `go test` for Local Hub service package or targeted package scope
  - targeted frontend test command for Local Hub page if added
- Rollback:
  - revert test-only changes or feature changes together

#### Dependencies
- `MyFlowHub-Win` only for the current recommended scope.
- Existing upstream auth/runtime contract from `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`.

#### Risks and Notes
- If the approved design insists on a persisted permit or new CLI flags, the plan must return to `3.1` and expand scope.
- No sub-agent use is planned.
- The current recommendation is to keep `parent join permit` non-persistent.

#### Parallelism Assessment
- No parallel execution planned.
- The scope is small, tightly coupled around one page and one backend service, and does not justify sub-agents.

#### Issue List
- 用户已在本轮回复 `请继续`，视为批准推荐的“permit 非持久化、仅下一次 Start/Restart 生效”方案，并允许进入 `3.2`。

阻塞：否
进入 3.2
禁止派发子Agent

### Stage 3.2 - Implementation
#### Execution Summary
- `DOC-1`
  - Added stable requirement/spec docs:
    - `docs/requirements/localhub-parent-admission.md`
    - `docs/specs/localhub-parent-admission.md`
  - Updated indexes:
    - `docs/requirements/README.md`
    - `docs/specs/README.md`
- `IMPL-1`
  - Extended Local Hub config with persisted `selfId`.
  - Added launch-only `LaunchRequest` with `parentJoinPermit`.
  - Refactored launch construction into testable helpers:
    - input normalization
    - approval-mode validation
    - env override construction for `HUB_SELF_ID` / `HUB_PARENT_JOIN_PERMIT`
  - Ensured inherited env is explicitly overridden even when the new values are blank, to avoid stale approval-mode env leaking into later launches.
  - Added focused Go tests in `internal/services/localhub/process_test.go`.
- `IMPL-2`
  - Added `Self ID` and one-shot `Parent Join Permit` fields to `frontend/src/pages/LocalHub.vue`.
  - Kept permit state transient in the page and cleared it after successful start/restart.
  - Wired `Start` / `Restart` to pass the launch request.
  - Added client-side validation hints before launch.
  - Tightened the existing flow so `saveConfig(true)` failure no longer falls through into `Start` / `Restart`.

#### Validation Results
- `DOC-1`
  - stable requirement/spec docs exist and are indexed
- `IMPL-1`
  - `GOWORK=off go test ./internal/services/localhub -count=1`
  - result: pass
- `IMPL-2`
  - no existing focused Local Hub frontend test file was present in the worktree
  - page wiring was reviewed against the new backend method signatures and transient-permit design

#### Issue List
- no known implementation blockers remain inside the approved Win-only scope

### Stage 3.3 - Code Review
- Requirements coverage: pass
  - The page now exposes `self-id` and one-shot `parent join permit`, and both `Start` and `Restart` flow through the same bounded launch request.
- Architecture fit: pass
  - Scope stayed inside `MyFlowHub-Win`; Server CLI surface was not expanded.
- Validation / safety: pass
  - Approval-mode invalid combinations are rejected before process start.
  - Sensitive permit values are passed via env and are not logged directly by the new code paths.
- Backward compatibility: pass
  - Stored configs without `selfId` still load.
  - Existing non-approval flows keep the previous args behavior.
- Test coverage: partial but acceptable
  - Focused Go tests cover the critical launch/env mapping and validation path.
  - No focused frontend automated test was added in this iteration; residual risk is limited to page-to-binding wiring and should be covered with manual UI smoke in Stage 4 or user QA.
- Sub-agent governance: pass
  - no sub-agents used

#### Issue List
- residual risk: Local Hub frontend behavior was not covered by a dedicated automated UI test in this iteration

### Stage 4 - Change Archive
#### Archive Outputs
- `docs/change/2026-04-14_win-localhub-parent-admission.md`
- Updated indexes:
  - `docs/change/README.md`
- Lessons:
  - no new lesson promoted in this iteration

#### Impact Record
- Requirements impact: `added`
- Specs impact: `added`
- Lessons impact: `none`

#### Ready State
- Implementation, targeted backend verification, code review, and change archive are complete in this worktree.
- The workflow is ready for user QA, follow-up edits, or explicit `结束workflow` closeout.
