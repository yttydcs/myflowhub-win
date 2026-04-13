# Local Hub Parent Admission

## Summary

- `Session -> Local Hub` must support joining a parent network that runs in approval mode without requiring the user to switch to a manual shell or hidden env workflow.
- The page must expose a stable `self-id` input and a one-shot `parent join permit` input.
- Existing Local Hub start, stop, restart, and advanced argument flows must remain backward compatible.

## User Problem

- A fresh local `hub_server` cannot complete admission into an approval-mode parent network with only the current structured fields.
- `self-id` is effectively hidden inside `extraArgs`, and `parent join permit` has no Win UI path at all.
- Users currently have to remember launch-time env details for a path that should be supported by the Local Hub page.

## Scope

- In scope:
  - expose `self-id` as a structured Local Hub config field
  - expose `parent join permit` as a structured launch-time field
  - validate unsupported approval-mode combinations before process launch
  - preserve current host, port, node ID, parent, auth, and `extraArgs` behavior
- Out of scope:
  - changing Server-side approval semantics
  - permit issuance and approval management UI
  - broad Local Hub redesign outside the admission path

## Stable Requirements

### Config Model

- `self-id` is a stable Local Hub configuration field and must be persisted with the other saved Local Hub settings.
- `parent join permit` is not a persisted config field. It is a one-shot launch input that applies only to the next `Start` or `Restart`.

### Validation

- If `parent join permit` is provided, Local Hub must reject launch unless:
  - parent link is enabled
  - parent address is non-empty
  - `self-id` is non-empty
- Validation failures must stop before `hub_server` is spawned and must surface an explicit user-visible error.

### Launch Behavior

- `Start` and `Restart` must both accept the one-shot admission input path.
- If a launch succeeds, the UI must clear the one-shot `parent join permit` field so the page does not imply that the same permit is reusable.
- Existing users who do not fill the new fields must see no behavior change in normal Local Hub flows.

## Acceptance Criteria

- A user can configure a fresh Local Hub for an approval-mode parent network directly from `Session -> Local Hub`.
- Approval-mode launch failures caused by missing `self-id`, disabled parent link, or missing parent address are rejected before process launch.
- Existing stored configs without `self-id` continue to load successfully.
- Existing non-approval Local Hub flows remain compatible.
