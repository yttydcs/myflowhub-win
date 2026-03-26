# Plan - win-authority-override-removal

## Workflow Information
- Repo: `MyFlowHub-Win`
- Branch: `refactor/win-authority-override-removal`
- Base: `main`
- Worktree: `D:/project/MyFlowHub3/worktrees/win-authority-override-removal`
- Current Stage: `4`

## Stage 1 - Requirements Analysis
### Goal
- Remove outdated Authority Override UI and legacy `authority.node_id` fallback wording from the Win authority admin console after the semi-central authority rollout.

### Scope
- Must:
  - remove manual `Authority Override` input from `Access Policy` / `Registration Approvals` / `Permit Issuance`
  - stop exposing `authorityReason` / legacy resolve-rule wording in the GUI
  - keep authority resolution automatic for the three pages
  - preserve existing policy / approvals / permit workflows
  - update long-lived `authority-admin-console` requirement/spec docs
- Optional:
  - keep a lightweight manual refresh action if it still improves usability
- Not in scope:
  - do not remove MCP-side `authority_id` override capability
  - do not redesign backend authority routing or add remote approval forwarding
  - do not change `PermissionService.ResolveAuthority` compatibility behavior unless required by the GUI refactor

### Use Cases
- authority admin opens one of the three authority pages after login and sees the effective authority directly, without needing to understand old `authority.node_id` fallback rules
- page actions still resolve authority automatically before load / save / approve / reject / permit issue / permit revoke
- semi-central deployments no longer show users an outdated manual override control that suggests persistent authority selection still drives the runtime

### Functional Requirements
- the three authority pages must no longer render the `Authority Override` input
- the pages must no longer render the legacy resolve rule text `manual override -> authority.node_id -> hubId fallback`
- the shared authority store must resolve authority without requiring page-level manual override input
- page actions must continue to work when identity is ready and must still fail explicitly when identity / authority is missing

### Non-functional Requirements
- keep the change surface minimal and localized to Win GUI + related docs/tests
- do not silently break non-root authority admin flows by removing backend compatibility logic that current tools still rely on
- keep naming and visual behavior consistent across the three pages

### Inputs / Outputs
- Input:
  - current auth snapshot (`nodeId`, `hubId`)
  - existing `PermissionService.ResolveAuthority`
- Output:
  - read-only authority context in the GUI
  - automatic authority resolution before authority operations

### Edge Cases
- user is disconnected or not logged in
- authority cannot be resolved
- authority changes when `hubId` or `nodeId` changes
- approvals / permit actions must still hit the resolved authority without a manual override field

### Acceptance Criteria
- no authority page shows `Authority Override` or the legacy resolve-rule copy
- authority pages still load / save / approve / reject / issue / revoke successfully through the shared authority store
- authority admin requirement/spec docs match the new read-only authority context
- frontend tests pass for the updated authority store behavior

### Risks
- if the GUI removes override state but still depends on it implicitly, page actions may stop resolving authority
- removing visible resolution details without keeping automatic resolution could make failures harder to understand
- overreaching into MCP authority routing would expand the task beyond the requested cleanup

## Stage 2 - Architecture Design
### Overall Solution
- Keep backend compatibility routing in `PermissionService.ResolveAuthority`, but stop exposing manual override and internal resolution reasons in the authority admin GUI.
- The shared authority store becomes a read-only authority context holder:
  - identity in
  - automatic `ResolveAuthority(...)` call out
  - resolved `authorityId` cached for page actions
- Page-level outdated controls are replaced by passive authority display plus the existing action buttons for real work.

### Module Responsibilities
- `frontend/src/stores/authority.ts`
  - owns auth snapshot-backed authority resolution
  - drops manual override state from the GUI-facing store
- `frontend/src/pages/AccessPolicy.vue`
  - shows current authority context and policy workflow
- `frontend/src/pages/RegistrationApprovals.vue`
  - shows current authority context and approval workflow
- `frontend/src/pages/PermitIssuance.vue`
  - shows current authority context and permit workflow
- `docs/requirements/authority-admin-console.md`
  - records user-facing console behavior
- `docs/specs/authority-admin-console.md`
  - records store/page contract changes

### Data / Call Flow
- login/session snapshot updates `sourceId` + `hubId`
- authority store clears stale authority when identity changes
- page load or action path calls `requireAuthority()`
- store auto-resolves authority through `PermissionService.ResolveAuthority(sourceId, hubId, 0)`
- page action proceeds with resolved `authorityId`

### Interface Drafts
- authority store state keeps:
  - `sourceId`
  - `hubId`
  - `authorityId`
  - `resolving`
- remove GUI-facing:
  - `authorityOverride`
  - `authorityReason`

### Error Handling and Safety
- preserve explicit local failures for missing `sourceId`, `hubId`, or unresolved authority
- do not remove `PermissionService.ResolveAuthority` itself, because backend and MCP still use that compatibility path
- keep page state stable on failed authority resolution

### Performance and Testing Strategy
- no new polling
- resolve authority on demand and cache per identity
- validate via:
  - frontend vitest authority admin store tests
  - targeted Go tests if service signatures change
  - frontend build if needed

### Extensibility Design Points
- future runtime `effective_authority_id` exposure can plug into the same authority store without bringing back manual override UI
- MCP authority routing can be migrated separately without coupling to this GUI cleanup

## Docs Governance Routing Decision
- Using `$m-docs` for routing and impact checks.
- Requirements impact: `clarify`
- Specs impact: `clarify`
- Related requirements:
  - `D:/project/MyFlowHub3/worktrees/win-authority-override-removal/docs/requirements/authority-admin-console.md`
- Related specs:
  - `D:/project/MyFlowHub3/worktrees/win-authority-override-removal/docs/specs/authority-admin-console.md`
  - `D:/project/MyFlowHub3/repo/MyFlowHub-Server/docs/specs/auth.md`
- Related lessons: `none`

## Executable Task List
- `AUTHUI-1`
- `AUTHUI-2`
- `DOC-1`
- `TEST-1`
- `REV-1`
- `ARC-1`

## Task Details
### AUTHUI-1 - Simplify Shared Authority Store
- Goal: remove GUI-facing manual override / reason state and keep automatic authority resolution only
- Files:
  - `frontend/src/stores/authority.ts`
  - `frontend/src/stores/authority_admin.test.ts`
- Acceptance:
  - store resolves authority with `override=0`
  - state resets cleanly on identity change
- Tests:
  - `npm test -- authority_admin`
- Rollback:
  - restore removed authority store fields and expectations

### AUTHUI-2 - Remove Outdated Page Controls
- Goal: remove outdated authority override widgets and legacy copy from the three authority pages
- Files:
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/pages/RegistrationApprovals.vue`
  - `frontend/src/pages/PermitIssuance.vue`
  - `frontend/src/i18n/messages/operations.ts`
- Acceptance:
  - no page renders the outdated input or legacy fallback text
  - authority display stays readable and actions still work
- Tests:
  - frontend tests and build
- Rollback:
  - restore removed widgets/copy

### DOC-1 - Align Authority Console Docs
- Goal: update long-lived requirement/spec docs to the new read-only authority context
- Files:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Acceptance:
  - docs no longer describe manual authority override as part of the GUI contract
- Tests:
  - docs consistency review
- Rollback:
  - revert doc updates

### TEST-1 - Run Validation
- Goal: verify the GUI cleanup does not break authority admin flows
- Files:
  - tests only
- Acceptance:
  - relevant tests pass
- Tests:
  - `npm test -- authority_admin`
  - `npm run build`
  - optional `GOWORK=off go test ./internal/services/permission -count=1`
- Rollback:
  - investigate failing area before revert

## Parallelism Assessment
- No sub-agents.
- Reason:
  - the write set is small and tightly coupled across store/pages/docs/tests

阻塞：否
进入 3.2

## Stage 3.3 - Review
### Checklist
- Scope kept to Win GUI, tests, and long-lived docs
- Backend `ResolveAuthority` compatibility path left intact
- Outdated Authority Override controls and wording removed from the three authority pages
- Shared authority store no longer exposes GUI-side override / reason state

### Validation
- `npm test -- authority_admin`
  - result: passed
- `npm run build`
  - result: passed
  - note: the worktree lacked generated `frontend/wailsjs`, so validation used a temporary junction to the main repo copy and removed it after build
- `git diff --check`
  - result: passed

## Stage 4 - Archive
### Outputs
- `docs/change/2026-03-26_win-authority-override-ui-removal.md`
- `docs/change/README.md`

### Workflow End State
- Archive complete
- Waiting for user confirmation before ending the workflow
