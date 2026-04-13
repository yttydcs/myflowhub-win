# 2026-04-14 Win Local Hub Parent Admission

## Background

- `Session -> Local Hub` could not directly launch a fresh local `hub_server` into an approval-mode parent network.
- The upstream runtime already supported `HUB_SELF_ID` and `HUB_PARENT_JOIN_PERMIT`, but the Win page had no structured way to provide them.
- Users had to fall back to manual env and extra-args workflows for a path that should be available from the Local Hub UI.

## Goal

- Add a stable `self-id` config field and a one-shot `parent join permit` launch field to Local Hub.
- Keep the scope inside `MyFlowHub-Win` and reuse the existing Server runtime contract instead of expanding CLI flags.
- Preserve current Local Hub behavior for users who do not use approval-mode parent admission.

## Changes

### Stable Docs

- Added stable requirement doc:
  - `docs/requirements/localhub-parent-admission.md`
- Added stable spec doc:
  - `docs/specs/localhub-parent-admission.md`
- Updated category indexes:
  - `docs/requirements/README.md`
  - `docs/specs/README.md`

### Backend

- `internal/services/localhub/types.go`
  - added persisted `Config.SelfID`
  - added launch-only `LaunchRequest.ParentJoinPermit`
- `internal/services/localhub/process.go`
  - added launch normalization and validation helpers
  - added launch env override builder for `HUB_SELF_ID` and `HUB_PARENT_JOIN_PERMIT`
  - ensured blank current values override inherited stale env
- `internal/services/localhub/service.go`
  - persisted `localhub.self_id`
  - changed `Start` / `Restart` to accept the launch request
  - routed process launch through the shared launch-spec builder
- `internal/services/localhub/process_test.go`
  - added focused tests for approval-mode env injection, validation, and stale-env clearing

### Frontend

- `frontend/src/pages/LocalHub.vue`
  - added structured `Self ID` field
  - added transient `Parent Join Permit` field
  - validated approval-mode launch combinations before start/restart
  - cleared the one-shot permit after successful start/restart
  - fixed the previous fall-through so `saveConfig(true)` failure no longer proceeds into `Start` / `Restart`
- `frontend/src/i18n/messages/session.ts`
  - added the new Local Hub validation and guidance copy

## Verification

### Automated

- `GOWORK=off go test ./internal/services/localhub -count=1`
  - result: pass
- `git diff --check`
  - result: pass
  - note: only existing LF/CRLF warnings were reported for the working copy

### Attempted But Blocked

- `npm exec vitest run src/pages/Stream.test.ts`
  - result: blocked by missing frontend test dependencies in this worktree
  - observed errors:
    - `Cannot find package 'vitest'`
    - `Could not resolve '@vitejs/plugin-vue'`

### Manual QA Suggestions

- Open `Session -> Local Hub` in worktree `D:\project\MyFlowHub3\worktrees\feat-win-localhub-approval-ui`.
- Fill:
  - `Parent link = on`
  - `Parent = <authority addr>`
  - `Self ID = <stable device id>`
  - `Parent Join Permit = <one-time permit>`
- Verify:
  - missing `Self ID` with permit blocks start before process launch
  - missing parent link with permit blocks start before process launch
  - successful start clears the permit input while preserving saved `Self ID`

## Requirement / Spec Impact

- Requirements impact: `added`
- Specs impact: `added`
- Lessons impact: `none`

## Rollback

- Revert the Local Hub docs, backend, frontend, and test changes in this workflow.
- Specifically remove:
  - `docs/requirements/localhub-parent-admission.md`
  - `docs/specs/localhub-parent-admission.md`
  - `internal/services/localhub/process_test.go`
- Restore previous versions of:
  - `internal/services/localhub/types.go`
  - `internal/services/localhub/process.go`
  - `internal/services/localhub/service.go`
  - `frontend/src/pages/LocalHub.vue`
  - `frontend/src/i18n/messages/session.ts`
