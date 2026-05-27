# Plan - MyFlowHub-Win MCP TopicBus Publish and Operational Write Tools

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/mcp-topicbus-publish`
- Base: `main` at `794c10a`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish`
- Current Stage: `4 - Change Archive / Closeout`

## Stage Records

### Initialization
- `guide.md`: none in `repo/MyFlowHub-Win`.
- Base/worktree confirmation:
  - Main repo path is control-plane only: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`.
  - Active implementation worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish`.
  - Dedicated branch: `feat/mcp-topicbus-publish`.
  - Participating repo: only `MyFlowHub-Win`.
  - Participating modules: headless MCP runtime/tools, MCP docs, focused MCP tests.
- Current baseline:
  - Main repo `main` was clean before worktree creation.
  - New worktree inherited an unrelated older root `plan.md`; it was replaced with this workflow plan so execution cannot follow the wrong task.
  - Shell may emit unrelated conda hook noise after successful commands; rely on command exit code and primary output.

### Stage 1 - Requirements Analysis
#### Goal
Expose a minimal TopicBus publish capability through the existing headless `myflowhub-mcp` client so Codex or another MCP host can publish a notification event that `MyFlowHub-MetricsNode` NotifyNode already subscribes to and displays as a system notification. During live setup, also keep the two small write-gated operational MCP tools needed to configure authority routing and sync runtime permission snapshots.

#### Scope
Must:
- Add `myflowhub_topicbus_publish` to the MCP tool list.
- Reuse existing `internal/services/topicbus.TopicBusService.Publish`.
- Keep MCP as an independent headless client with isolated config and node identity.
- Preserve existing MCP `stdio` contract: stdout remains JSON-RPC only, logs remain stderr/service logs.
- Require connected and authenticated/default identity state before publish, using the same source/target fallback model as management/varstore tools.
- Validate `topic` and `name` locally before sending.
- Accept notification-friendly payload inputs such as `title`, `body`, `level`, `source`, `url`, and optional structured `payload` / `meta`.
- Gate publish behind `allow_write`, because it emits an externally visible event.
- Add `myflowhub_management_config_set` behind `allow_write` so MCP can update management config such as authority routing in controlled setup flows.
- Add `myflowhub_auth_push_perms_snapshot` behind `allow_write` so MCP can push a validated runtime permission snapshot to the authority.
- Update stable MCP requirement/spec docs because the formal tool set changes.
- Add focused tests for tool registration, validation, routing defaults, write gate, payload shaping, management config set, and permission snapshot push.

Optional:
- Add `topicbus_subscribe`, `topicbus_unsubscribe`, or `topicbus_list_subs` MCP tools later.
- Add smoke-script coverage later when a real Hub and NotifyNode test rig are required.

Not in scope:
- Do not modify `MyFlowHub-Proto`.
- Do not modify `MyFlowHub-MetricsNode` NotifyNode behavior.
- Do not add wildcard topics, offline replay, ack, durable queue, or app-market abstractions.
- Do not modify Win GUI TopicBus pages or Wails bindings.
- Do not add a native OS notification presenter to `MyFlowHub-Win`; display remains owned by MetricsNode.

#### Use Cases
- Codex finishes a coding task and calls MCP with topic `codex/task/done`; a MetricsNode subscribed to that exact topic pops a system notification.
- Codex reports a failed verification with topic `codex/task/failed` and a body containing the failed command summary.
- A user manually calls the MCP tool to verify the end-to-end NotifyNode route before wiring automatic agent behavior.
- A future automation layer can reuse the same publish tool as the stable MCP emission point.

#### Functional Requirements
1. `tools/list` must include `myflowhub_topicbus_publish`.
2. The tool must accept:
   - `topic` required non-empty string
   - `name` optional string, defaulting to a stable value such as `mcp.topicbus.publish` or a notification-specific name
   - `title` optional string
   - `body` optional string
   - `level` optional string
   - `source` optional string
   - `url` optional string
   - `payload` optional JSON object for caller-controlled payload fields
   - `meta` optional JSON object for extra metadata
   - `source_id` optional node ID
   - `target_id` optional node ID
3. If `payload` is supplied, the handler must preserve it as structured JSON rather than stringifying it unnecessarily.
4. If `title/body/level/source/url/meta` are supplied, the handler must merge them into the outgoing payload object in a predictable way.
5. If neither structured notification fields nor payload are supplied, the outgoing TopicBus payload may be omitted or default to a small object containing source metadata.
6. The handler must trim and validate `topic` and `name` before sending.
7. The handler must resolve source/target using the same identity/default fallback behavior as existing management tools.
8. The handler must fail locally with structured errors for invalid arguments, missing identity, not connected, and write disabled.
9. The runtime must own a `TopicBusService` instance and close it during runtime shutdown.
10. The runtime must expose a `TopicBusPublish` method to the MCP tools backend interface.
11. The runtime must expose `ConfigSet` and `PushPermsSnapshot` methods to the MCP tools backend interface.
12. `management_config_set` must reject empty keys, require `allow_write`, and use the same management source/target fallback as config reads.
13. `auth_push_perms_snapshot` must reject an empty snapshot, require `allow_write`, and resolve the authority using explicit `authority_id`, `authority.node_id`, or hub fallback.

#### Non-functional Requirements
- Minimal change surface: keep implementation inside existing MCP runtime/tool patterns.
- Backward compatibility: existing MCP tools and GUI behavior must not change.
- Safety: TopicBus publish is gated by `allow_write`.
- Observability: existing TopicBus service logging remains the send-side trace.
- Maintainability: do not duplicate transport encoding; route through `TopicBusService.Publish`.
- Payload handling: use `encoding/json` and typed maps/raw JSON, not ad hoc string concatenation.

#### Inputs / Outputs
- Inputs:
  - MCP tool call JSON arguments.
  - Runtime auth/default state.
  - Existing Hub session.
- Output to MCP caller:
  - Structured success object containing resolved `source_id`, `target_id`, `topic`, `name`, and normalized payload preview when applicable.
  - Existing structured error shape on failures.
- Output to Hub:
  - TopicBus `publish` frame containing `topicbus.PublishReq{Topic, Name, TS, Payload}`.
- Output to NotifyNode:
  - Live exact-topic event only if NotifyNode is online and subscribed to the same topic.

#### Edge Cases
- Empty or whitespace-only `topic`.
- Empty or whitespace-only custom `name`.
- Caller passes `payload` that is not a JSON object.
- Caller passes notification fields that conflict with `payload` keys.
- Session is disconnected.
- Runtime has no usable source identity.
- Runtime has no usable target ID.
- `allow_write=false`.
- Empty `management_config_set.key`.
- Empty `auth_push_perms_snapshot.snapshot`.
- Current authority role lacks permission to write config or push permission snapshots.
- Hub send fails or route is unavailable.
- NotifyNode is offline or subscribed to a different topic; MCP publish still succeeds because TopicBus publish has no delivery ack.

#### Acceptance Criteria
1. `go test ./internal/mcp ./internal/mcpapp ./internal/services/topicbus -count=1` passes with `GOWORK=off`.
2. `go build -o .\build\bin\myflowhub-mcp.exe .\cmd\myflowhub-mcp` passes with `GOWORK=off`.
3. `myflowhub_topicbus_publish` appears in `tools/list`.
4. With `allow_write=false`, the tool returns `write_disabled` before sending.
5. With missing/invalid `topic`, the tool returns `invalid_arguments`.
6. With connected/authenticated fake backend and `allow_write=true`, the tool calls backend publish with expected source, target, topic, name, and JSON payload.
7. `management_config_set` and `auth_push_perms_snapshot` are write-gated and have focused tests.
8. Stable MCP requirements and specs list TopicBus publish plus the two operational write tools and their write-gate behavior.

#### Risks
- Existing MCP tool file is large; duplicate function blocks or stale generated sections may create accidental edits. Use focused patches and targeted tests.
- TopicBus publish is fire-and-forget, so a successful MCP call does not prove a NotifyNode displayed the notification.
- Adding a required method to the MCP backend interface requires updating all test fakes.

#### Issue List
- None currently.

### Stage 2 - Architecture Design
#### Overall Solution
Add TopicBus as a first-class service inside the existing MCP runtime, then expose a single write-gated MCP tool that publishes a TopicBus event. This keeps MetricsNode as the subscriber/display endpoint and makes `myflowhub-mcp` the publisher endpoint for Codex.

Selected approach:
- Runtime assembly: instantiate `topicbussvc.New(session, logs, bus)` alongside `varpool`.
- Runtime API: add `TopicBusPublish(ctx, sourceID, targetID uint32, topic, name, payloadText string) error`, `ConfigSet(...)`, and `PushPermsSnapshot(...)`.
- MCP backend: extend `Backend` with `TopicBusPublish`, `ConfigSet`, and `PushPermsSnapshot`.
- Tool surface: add `myflowhub_topicbus_publish`, `myflowhub_management_config_set`, and `myflowhub_auth_push_perms_snapshot`.
- Payload builder: tool layer builds a JSON object from `payload` plus notification convenience fields, marshals it once, and passes the JSON string to `TopicBusService.Publish`.

#### Alternatives Considered
- Publish directly from MetricsNode:
  - Rejected for this workflow because MetricsNode is already the subscriber/display node; Codex needs a publisher exposed through MCP.
- Add subscribe/unsubscribe/list tools now:
  - Deferred. The immediate notification use case only needs publish; adding subscription control increases permission and lifecycle surface.
- Add a dedicated notification protocol:
  - Rejected. Existing TopicBus publish envelope already satisfies live notification fanout.
- Add app-market manifest/handler abstractions now:
  - Rejected for scope. This workflow should only create the lowest stable publish capability.

#### Module Responsibilities
- `internal/mcpapp/runtime.go`
  - Own TopicBus service lifecycle.
  - Provide timeout-wrapped `TopicBusPublish`, `ConfigSet`, and `PushPermsSnapshot`.
- `internal/mcp/tools.go`
  - Declare tool schema.
  - Decode/validate args.
  - Enforce connected/auth/write-gate checks.
  - Resolve source/target route.
  - Build normalized JSON payload.
  - Return structured success or structured MCP errors.
- `internal/mcp/tools_test.go`
  - Extend fake backend.
  - Lock registration, write-gate, validation, route, payload behavior, config set, and permission snapshot behavior.
- `docs/requirements/mcp-client.md`
  - Add TopicBus publish and operational write tools to stable MCP capability requirements.
- `docs/specs/mcp-client.md`
  - Add tool contracts, write-gate classification, and runtime assembly boundary.

#### Data / Call Flow
```text
MCP host / Codex
  -> tools/call myflowhub_topicbus_publish
  -> internal/mcp tool handler validates args and write gate
  -> resolve source_id / target_id from explicit args or auth/defaults
  -> build JSON payload
  -> mcpapp.Runtime.TopicBusPublish
  -> internal/services/topicbus.TopicBusService.Publish
  -> session.SendCommand(SubProtoTopicBus, source, target, publish envelope)
  -> Hub TopicBus live forwarding
  -> MetricsNode NotifyNode exact-topic subscriber
  -> OS notification presenter
```

#### Interface Drafts
MCP tool args:
```go
type topicBusPublishArgs struct {
    Topic    string          `json:"topic"`
    Name     string          `json:"name,omitempty"`
    Title    string          `json:"title,omitempty"`
    Body     string          `json:"body,omitempty"`
    Level    string          `json:"level,omitempty"`
    Source   string          `json:"source,omitempty"`
    URL      string          `json:"url,omitempty"`
    Payload  json.RawMessage `json:"payload,omitempty"`
    Meta     json.RawMessage `json:"meta,omitempty"`
    SourceID *uint32         `json:"source_id,omitempty"`
    TargetID *uint32         `json:"target_id,omitempty"`
}
```

Runtime method:
```go
func (r *Runtime) TopicBusPublish(ctx context.Context, sourceID, targetID uint32, topic, name, payloadText string) error
func (r *Runtime) ConfigSet(ctx context.Context, sourceID, targetID uint32, key, value string) (protomanagement.ConfigResp, error)
func (r *Runtime) PushPermsSnapshot(ctx context.Context, sourceID, targetID uint32, snapshot coreperm.Snapshot) error
```

Suggested MCP call:
```json
{
  "topic": "codex/task/done",
  "name": "codex.done",
  "title": "代码写完了",
  "body": "已推送，CI 通过",
  "level": "info",
  "source": "codex"
}
```

#### Error Handling and Safety
- `invalid_arguments`: empty topic/name, invalid payload/meta JSON, unsupported JSON shape if a strict object is required.
- `not_connected`: session not connected.
- `missing_identity`: source or target cannot be resolved.
- `write_disabled`: `allow_write=false`.
- `invalid_arguments`: empty config key or empty permission snapshot.
- `upstream_error`: `TopicBusService.Publish` or session send fails.
- Publish remains no-ack; the success response should not claim notification delivery.

#### Performance and Testing Strategy
- Payload is small and marshaled once per call.
- No new goroutines or background subscriptions are needed for MCP publish.
- Targeted tests:
  - tool list contains `myflowhub_topicbus_publish`
  - write gate rejects before backend call
  - invalid topic rejects
  - default route resolution works
  - payload merge preserves structured fields
  - upstream error maps to structured error
  - management config set write gate and route/key/value behavior
  - auth permission snapshot route resolution and non-empty snapshot behavior
- Validation commands:
  - `$env:GOWORK='off'; go test ./internal/mcp ./internal/mcpapp ./internal/services/topicbus -count=1`
  - `$env:GOWORK='off'; go build -o .\build\bin\myflowhub-mcp.exe .\cmd\myflowhub-mcp`

#### Extensibility Design Points
- Tool name and payload model leave room for future `myflowhub_topicbus_subscribe/list_subs`.
- Notification-friendly fields are plain payload keys, so MetricsNode NotifyNode can consume them without protocol changes.
- Future app-market handlers can standardize topic names and payload conventions above TopicBus without altering this tool.
- Operational write tools remain generic and write-gated, so they can support MCP setup without adding environment-specific hard-coded config.

#### Issue List
- None currently.

### Stage 3.1 - Planning
#### Project Goal and Current State
- Current NotifyNode can display system notifications from subscribed TopicBus topics.
- Current `MyFlowHub-Win` MCP client exposes session/auth/management/exec/flow/varstore but not TopicBus publish, management config write, or permission snapshot push.
- Current Win TopicBus service already implements `Publish`; the missing layer is MCP runtime/tool exposure.

#### Docs Governance Routing Decision
Using `$m-docs`:
- Docs tree exists and is healthy; no bootstrap needed.
- Requirements impact: update existing `docs/requirements/mcp-client.md` because the stable MCP tool set changes from "reserved for topicbus later" to "publish supported now" and now includes two controlled operational write helpers.
- Specs impact: update existing `docs/specs/mcp-client.md` because the MCP interface contract, runtime methods, and write-gate classification change.
- Related requirements:
  - `docs/requirements/mcp-client.md`
- Related specs:
  - `docs/specs/mcp-client.md`
- Related lessons:
  - none currently. This is a straightforward capability addition; create a lesson only if implementation exposes a recurring trap.
- Change archive destination after implementation:
  - `docs/change/2026-05-27_mcp-topicbus-publish.md`

#### Executable Task List
- [x] `BASE-1` Repair pre-existing MCP tools.go duplicate residue that blocks compilation.
- [x] `DOC-1` Update stable MCP requirement/spec docs for TopicBus publish and operational write tools.
- [x] `RT-1` Wire TopicBus service and operational wrappers into MCP runtime.
- [x] `TOOL-1` Add `myflowhub_topicbus_publish` tool contract, validation, routing, write gate, and payload builder.
- [x] `TOOL-2` Add `myflowhub_management_config_set` and `myflowhub_auth_push_perms_snapshot`.
- [x] `TEST-1` Add focused MCP tool tests and update fake backend.
- [x] `VERIFY-1` Run targeted tests/build and perform self review.
- [ ] `ARCHIVE-1` Create change archive and record docs impact.

#### Task Details
##### DOC-1 - Stable MCP Docs
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish\plan.md`
- Goal: Make the stable MCP requirement/spec list `myflowhub_topicbus_publish`, `myflowhub_management_config_set`, and `myflowhub_auth_push_perms_snapshot`, and define their safety behavior.
- Files / Modules:
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
- Write Set: docs only.
- Acceptance: Requirements/specs mention TopicBus publish, operational write helpers, args, write gate, and no-delivery-ack boundary.
- Test Points: docs review.
- Rollback: Revert the two docs files.

##### BASE-1 - Pre-existing MCP Tool File Compile Repair
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish\plan.md`
- Goal: Remove the already-present duplicate/orphaned `internal/mcp/tools.go` residue that causes `go test ./internal/mcp` to fail before TopicBus changes.
- Files / Modules:
  - `internal/mcp/tools.go`
- Write Set: remove duplicate stale handler/helper block only.
- Acceptance: `go test ./internal/mcp -count=1` can progress past the syntax error at the pre-existing orphaned code.
- Test Points: `GOWORK=off go test ./internal/mcp -count=1`.
- Rollback: Restore removed duplicate block if a later diff shows it was not redundant.

##### RT-1 - MCP Runtime TopicBus and Operational Services
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish\plan.md`
- Goal: Instantiate and expose TopicBus publish plus timeout-wrapped management/auth write wrappers in the headless MCP runtime.
- Files / Modules:
  - `internal/mcpapp/runtime.go`
- Write Set: runtime assembly and method wrappers only.
- Acceptance: runtime has a TopicBus service, closes it, and exposes timeout-wrapped publish/config/snapshot methods.
- Test Points: `go test ./internal/mcpapp -count=1`.
- Rollback: Revert runtime.go changes.

##### TOOL-1 - MCP TopicBus Publish Tool
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish\plan.md`
- Goal: Add the MCP tool with validation, payload shaping, route resolution, and write-gate protection.
- Files / Modules:
  - `internal/mcp/tools.go`
- Write Set: MCP backend interface, args type, tool registration, handler/helper functions.
- Acceptance: tool appears in tools list and calls backend publish only after local validation/write-gate checks pass.
- Test Points: `go test ./internal/mcp -count=1`.
- Rollback: Revert tools.go changes.

##### TOOL-2 - MCP Operational Write Tools
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish\plan.md`
- Goal: Add the controlled setup tools needed by MCP-driven authority/config workflows.
- Files / Modules:
  - `internal/mcp/tools.go`
- Write Set: MCP backend interface, args types, tool registrations, handlers, snapshot validation, and status hint update.
- Acceptance: `management_config_set` and `auth_push_perms_snapshot` require `allow_write`, validate local inputs, and route through existing management/authority fallback helpers.
- Test Points: `go test ./internal/mcp -count=1`.
- Rollback: Revert operational write tool changes.

##### TEST-1 - Focused MCP Tests
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish\plan.md`
- Goal: Lock the new tool behavior without requiring a live Hub.
- Files / Modules:
  - `internal/mcp/tools_test.go`
- Write Set: test fake backend and new tests.
- Acceptance: tests cover registration, invalid topic, write gate, successful publish payload/routing, upstream error, config set write gate/route, and auth snapshot route/snapshot behavior.
- Test Points: `go test ./internal/mcp -count=1`.
- Rollback: Revert tools_test.go changes.

##### VERIFY-1 - Validation and Review
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish\plan.md`
- Goal: Verify implementation and perform Stage 3.3 review.
- Files / Modules:
  - no planned source edits unless review finds issues.
- Write Set: none unless returning to Stage 3.2 for fixes.
- Acceptance:
  - `$env:GOWORK='off'; go test ./internal/mcp ./internal/mcpapp ./internal/services/topicbus -count=1`
  - `$env:GOWORK='off'; go build -o .\build\bin\myflowhub-mcp.exe .\cmd\myflowhub-mcp`
  - Stage 3.3 checklist all pass.
- Test Points: targeted Go tests/build.
- Rollback: Revert failed task changes by task ID if needed.

##### ARCHIVE-1 - Change Archive
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-mcp-topicbus-publish\plan.md`
- Goal: Archive workflow results and docs impact.
- Files / Modules:
  - `docs/change/2026-05-27_mcp-topicbus-publish.md`
  - `docs/change/README.md` if index maintenance is required by local style
  - `docs/lessons/*` only if a reusable implementation lesson emerges
- Write Set: docs/change and optional docs/lessons.
- Acceptance: Archive records task mapping, tests, decisions, rollback, docs impact, and sub-agent trace.
- Test Points: docs review.
- Rollback: Remove archive entry and index change.

#### Dependencies
- Existing `TopicBusService.Publish` must remain compatible.
- Existing MCP route/default helpers must be reused; if no suitable helper exists, add a small local helper inside `tools.go`.
- Operational write tools must remain generic and write-gated; do not hard-code cloud node IDs, roles, topics, or user-specific config.
- No live Hub is required for unit tests; live end-to-end notification verification can be manual after merge.

#### Risks and Notes
- `TopicBusService.Publish` validates `topic` and `name`; tool layer still validates first so MCP errors are structured.
- Publish success means "frame sent", not "notification displayed".
- MetricsNode subscription is exact topic. User-facing examples should use exactly matching topic strings, e.g. `codex/task/done`.

#### Parallelism Assessment
- Potentially separable write sets exist (`docs`, `runtime`, `tools/tests`), but the changes are small and the tool/test changes share the same interface/fake backend.
- Sub-agents are not used for this round because the implementation is tightly coupled through the `Backend` interface and payload contract, and host policy has not introduced a separate sub-agent execution channel in this turn.

#### Issue List
- None currently.

阻塞：否
进入 3.2

### Stage 3.2 - Implementation Record
#### File-level Change Summary
- `BASE-1`
  - `internal/mcp/tools.go`
    - Removed a pre-existing duplicated/orphaned handler block that left non-declaration statements outside any function and blocked `go test ./internal/mcp`.
- `DOC-1`
  - `docs/requirements/mcp-client.md`
    - Added `topicbus publish` to the stable MCP capability list and acceptance path.
    - Recorded exact-topic, no replay, no delivery confirmation, and write-gate behavior.
  - `docs/specs/mcp-client.md`
    - Added `myflowhub_topicbus_publish` contract, runtime service boundary, write-gate classification, and error semantics.
- `RT-1`
  - `internal/mcpapp/runtime.go`
    - Wired `TopicBusService` into MCP runtime assembly and shutdown.
    - Added `TopicBusPublish(...)` timeout-wrapped runtime method.
    - Added timeout-wrapped `ConfigSet(...)` and `PushPermsSnapshot(...)` runtime methods.
- `TOOL-1`
  - `internal/mcp/tools.go`
    - Extended `Backend` with `TopicBusPublish`.
    - Added `topicBusPublishArgs`, tool registration, local validation, write gate, route resolution, payload merge, and structured success/error handling.
    - Added `defaultTopicBusPublishName`.
- `TOOL-2`
  - `internal/mcp/tools.go`
    - Extended `Backend` with `ConfigSet` and `PushPermsSnapshot`.
    - Added `myflowhub_management_config_set` with key validation, write gate, route resolution, and structured response.
    - Added `myflowhub_auth_push_perms_snapshot` with non-empty snapshot validation, write gate, authority route resolution, and structured response.
    - Updated `session_status` write-gate hint to list all write tools.
- `TEST-1`
  - `internal/mcp/tools_test.go`
    - Extended fake backend.
    - Added tests for registration, write-gate rejection, empty topic rejection, payload merge/routing, default name, and upstream error mapping.
    - Added tests for management config set write-gate and route/key/value behavior.
    - Added test for auth permission snapshot authority resolution and backend call.

#### Design Notes
- `myflowhub_topicbus_publish` is intentionally a write-gated MCP tool because it emits externally visible events.
- `myflowhub_management_config_set` and `myflowhub_auth_push_perms_snapshot` are intentionally write-gated because they alter runtime authority/config state.
- The tool uses existing management source/target fallback semantics.
- `payload` and `meta` are accepted only as JSON objects; convenience fields `title/body/level/source/url` override same-name payload keys.
- Successful publish means the frame was sent, not that any NotifyNode displayed a notification.

#### Validation Results
- `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - Result: passed.
- `$env:GOWORK='off'; go test ./internal/mcp ./internal/mcpapp ./internal/services/topicbus -count=1`
  - Result: passed.
- `$env:GOWORK='off'; go build -o .\build\bin\myflowhub-mcp.exe .\cmd\myflowhub-mcp`
  - Result: passed.

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - Publish tool, operational write tools, write gate, docs, tests, and runtime wiring are covered.
- 架构合理性：通过
  - MCP remains the publisher endpoint; MetricsNode remains the subscriber/display endpoint; existing TopicBus service owns protocol encoding.
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - One payload marshal per tool call; no new polling, goroutines, or subscription loops.
- 可读性与一致性：通过
  - Handler follows existing MCP tool patterns and structured error helpers.
- 可扩展性与配置化：通过
  - Tool naming and payload model leave room for later subscribe/list tools and app-market conventions above TopicBus.
  - Operational write helpers stay generic and do not encode user-specific cloud config.
- 稳定性与安全：通过
  - Local validation happens before sending; write gate prevents event emission unless explicitly enabled.
- 测试覆盖情况：通过
  - Focused unit tests cover registration, invalid input, write gate, happy path, upstream error, config set, and permission snapshot push.
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - No sub-agents used; write sets stayed within confirmed plan.

### Stage 4 - Archive Prep
- 使用 `$m-docs` 完成 requirement/spec/lesson 影响复核。
- Requirements impact: updated
  - `docs/requirements/mcp-client.md`
- Specs impact: updated
  - `docs/specs/mcp-client.md`
- Lessons impact: none
  - No recurring runtime/debugging lesson is required; the pre-existing duplicated MCP tool residue is recorded in this workflow archive.
- Change archive target:
  - `docs/change/2026-05-27_mcp-topicbus-publish.md`
