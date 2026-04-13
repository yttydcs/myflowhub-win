# Local Hub Parent Admission

## Summary

- `MyFlowHub-Win` adds a stable `selfId` field to the Local Hub config snapshot and save path.
- `Start` and `Restart` accept a launch request carrying an optional one-shot `parentJoinPermit`.
- Win maps the approval-mode inputs onto the existing `hub_server` runtime contract through environment variables instead of adding new Server CLI flags.

## Technical Contract

### Frontend

- `Snapshot.config` includes `selfId`.
- `SaveConfig` persists `selfId` with the rest of the Local Hub config.
- The page keeps `parentJoinPermit` in transient UI state only; `loadSnapshot()` must not overwrite it from persisted config.
- `Start(req)` and `Restart(req)` send `{ parentJoinPermit?: string }` when the user provides a one-shot permit.

### Backend

- `Config` includes:
  - `SelfID string`
- A launch-only request type includes:
  - `ParentJoinPermit string`
- Launch validation combines persisted config plus the request override and rejects unsupported approval-mode combinations before `exec.Command(...).Start()`.

### Runtime Mapping

- Local Hub continues to pass existing CLI args for:
  - `-addr`
  - `-node-id`
  - parent link flags
  - auth flags
  - advanced extra args
- Local Hub injects approval-mode values through environment overrides:
  - `HUB_SELF_ID`
  - `HUB_PARENT_JOIN_PERMIT`
- The launch environment must override inherited values even when the current request leaves them blank, so stale parent admission env does not leak into a later launch.

## Constraints

- Keep the feature bounded to `MyFlowHub-Win` unless the approved design changes.
- Do not log the permit value directly.
- Preserve backward compatibility for stored configs that do not yet contain `selfId`.

## Verification

- Add focused Go tests for launch validation and env construction.
- Confirm `Start` and `Restart` both use the same launch-request path.
