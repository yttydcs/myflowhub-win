# Plan - MCP Remote Docker Auth

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-Win`
- Branch: `feat/mcp-remote-docker-auth`
- Base: local `main` at `906bdd1 feat: 增加 MCP ensure-running 启动模式`
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-mcp-remote-docker-auth`
- Current Stage: 3.1 planning

## Stage Records

### Initialization

- `$m-autoflow` is active because this changes runtime security, deployment packaging, requirements/specs, and archive docs.
- `$m-docs` is active for requirements/specs impact checks and final change/lesson routing.
- `guide.md` read from `D:/project/MyFlowHub3/guide.md`.
- Worktree rule satisfied: all implementation must stay in `D:/project/MyFlowHub3/worktrees/feat-mcp-remote-docker-auth`.
- Main repo current status before worktree creation: `main...origin/main [ahead 3]`, clean.
- Participating repo: `MyFlowHub-Win` only.
- Participating modules: `cmd/myflowhub-mcp`, `internal/mcp`, install script, Docker deployment files, README, requirements/specs/change docs.

### Stage 1 - Requirements Analysis

#### Goal

Make `myflowhub-mcp` deployable as a Dockerized remote HTTP MCP server while preserving local-safe defaults. Remote exposure must be explicit and must require an access key so the service is not callable by arbitrary public clients.

#### Scope

Must:

- Preserve existing `stdio`, local HTTP, and `-EnsureRunning` behavior.
- Keep local HTTP default as `127.0.0.1:17688`.
- Add an explicit remote HTTP mode that allows non-loopback listen addresses only when the user opts in.
- Require a bearer token when remote mode is enabled.
- Use one fixed static bearer token for the remote MCP endpoint; do not add token issuing, refresh, rotation, or multi-token management in this workflow.
- Accept the token through `Authorization: Bearer <token>`.
- Allow the token to be configured through an environment variable and/or CLI flag.
- Reject requests without the expected token when auth is configured.
- Add Docker build/runtime entrypoints for `myflowhub-mcp`.
- Add compose example with persistent config volume and required token environment.
- Update Codex install script so remote HTTP config can include `http_headers.Authorization`.
- Update stable requirements/specs before or alongside code.
- Keep write tools behind existing `--allow-write`; remote access key is transport access control, not Hub authorization.

Optional:

- Accept raw `Authorization: <token>` only if it does not weaken the Bearer contract. Default should document Bearer.
- Add a small health/help note in README; no separate health endpoint unless needed for Docker validation.

Out of scope:

- No TLS termination in the Go process.
- No user database, OAuth, session cookie, or token rotation service.
- No token issuing, refresh, rotation, or multi-token management flow.
- No public deployment to a live server in this workflow unless the user explicitly asks after implementation.
- No reverse proxy configuration beyond documenting that production should use HTTPS/VPN/firewall.
- No change to Hub protocol wire or NotifyNode behavior.

#### Use Cases

- User deploys `myflowhub-mcp` on a cloud server with Docker and connects Codex over HTTP.
- Multiple Codex sessions share one remote MCP runtime, Hub connection, config directory, node keys, and login state.
- User protects the MCP endpoint with a bearer token stored in Codex `http_headers`.
- User keeps `--allow-write` disabled for read-only deployments, or explicitly enables it when TopicBus publish / flow / varstore writes are intended.

#### Functional Requirements

- CLI adds an explicit remote enable switch, proposed as `--allow-remote`.
- CLI adds an access token flag, proposed as `--auth-token`.
- CLI can read access token from `MYFLOWHUB_MCP_AUTH_TOKEN`.
- If `--allow-remote` is false and `--listen` is non-loopback, startup must fail as today.
- If `--allow-remote` is true and `--auth-token` / env token is empty, startup must fail.
- HTTP request authorization must run before JSON-RPC body processing.
- Requests must pass when `Authorization` exactly matches `Bearer <token>`.
- Unauthorized requests must return HTTP `401`; malformed wrong-token requests must not reveal the expected token.
- Origin checks should remain strict for local mode. In remote mode, Origin protection is not a substitute for bearer auth; token auth is mandatory.
- Docker image must run the headless MCP binary, not the Wails GUI.
- Docker runtime must expose `/mcp` on a configurable port and store config under a mounted volume.
- Compose example must make the token required via `.env` or environment substitution.
- Codex install script must support remote URL + authorization header output for HTTP transport.

#### Non-functional Requirements

- Secure by default: no remote listen and no unauthenticated remote mode.
- Avoid logging secrets.
- Keep auth implementation inside HTTP transport boundary, not in tool handlers.
- Use constant-time comparison for token equality.
- Keep Docker image minimal and avoid frontend/Wails build dependencies for MCP-only image.
- Preserve Windows PowerShell 5.1 script compatibility.

#### Inputs / Outputs

- Inputs:
  - CLI flags: `--transport`, `--listen`, `--mcp-path`, `--allow-remote`, `--auth-token`, existing identity/config flags.
  - Environment: `MYFLOWHUB_MCP_AUTH_TOKEN`, existing runtime environment.
  - HTTP: `Authorization` header.
- Outputs:
  - HTTP JSON-RPC responses on authorized requests.
  - HTTP 401 for missing/invalid token.
  - Explicit startup error when remote mode is unsafe.
  - Docker image and compose configuration.

#### Edge Cases

- `--allow-remote` set but token missing.
- Token env variable is whitespace.
- `--listen 0.0.0.0:17688` without `--allow-remote`.
- `Authorization` missing.
- `Authorization: Bearer` with empty token.
- `Authorization` has the wrong scheme.
- Correct token but non-POST method.
- Docker container starts with missing token environment.
- Codex config contains HTTP URL but no `http_headers.Authorization`.
- Existing local `-EnsureRunning` path must not suddenly require token.

#### Acceptance Criteria

- `go test ./internal/mcp ./internal/mcpapp -count=1` passes.
- `go build ./cmd/myflowhub-mcp` passes.
- Local HTTP without token remains usable on loopback.
- Non-loopback listen without explicit remote switch fails.
- Remote switch without token fails.
- Remote switch with token accepts requests carrying `Authorization: Bearer <token>`.
- Remote switch with token rejects missing/wrong token.
- Docker image builds for MCP.
- Compose config is syntactically valid and documents required env.
- Install script `-WhatIf` can output remote HTTP Codex config with `http_headers.Authorization`.
- `git diff --check` passes.

#### Risks

- Public MCP exposure is high risk because tools can mutate Hub state when `--allow-write` is enabled.
- Codex HTTP MCP must be configured with headers; otherwise remote endpoint will reject all calls.
- Docker build may require Go 1.25 image availability. If unavailable locally, use the closest official Go image only after verifying the toolchain requirement.

### Stage 2 - Architecture Design

#### Overall Solution

Keep the local HTTP server behavior unchanged, then add a remote-auth layer at the HTTP transport boundary:

```text
HTTP request
  -> remote/listen safety check configured at startup
  -> bearer token check when token configured
  -> Origin check for local mode
  -> method/body validation
  -> shared JSON-RPC HandleRequest
```

Remote mode is explicit because it changes the security boundary. Bearer token is mandatory in remote mode because exposing MCP without authentication would allow arbitrary JSON-RPC tool calls.

#### Alternatives Considered

- Reverse proxy only:
  - Pros: mature TLS/auth can be delegated.
  - Cons: Go service would still be unsafe if exposed directly; not enough for a reusable Docker image.
- Query token:
  - Pros: simplest for ad hoc clients.
  - Cons: leaks through logs/history and is weaker than headers; not preferred.
- Basic auth:
  - Pros: common.
  - Cons: no need for username/password model here; bearer token matches Codex `http_headers` better.
- Mutual TLS:
  - Pros: strongest.
  - Cons: too heavy for this workflow and not needed for first remote Docker support.

#### Module Responsibilities

- `cmd/myflowhub-mcp/main.go`
  - Parse `--allow-remote` and `--auth-token`.
  - Read `MYFLOWHUB_MCP_AUTH_TOKEN`.
  - Validate remote mode before server start.
  - Pass remote/auth config into `mcp.HTTPServerConfig`.
- `internal/mcp/http_server.go`
  - Enforce remote listen and token policy.
  - Check `Authorization: Bearer <token>` before dispatch.
  - Preserve local Origin behavior.
- `internal/mcp/http_server_test.go`
  - Cover local/no-token compatibility, remote requires token, auth success/failure, non-loopback validation.
- `scripts/install-codex-myflowhub-mcp.ps1`
  - Add optional auth token/header output for HTTP config.
- `Dockerfile.mcp`
  - Build `cmd/myflowhub-mcp` into a Linux binary.
  - Run it with HTTP transport and environment-driven config.
- `docker/mcp-entrypoint.sh`
  - Map simple Docker environment variables to CLI flags without logging secrets.
- `docker-compose.mcp.yml`
  - Provide a minimal service with port mapping, config volume, token env, Hub endpoint env, and optional write gate.
- `README.md`
  - Add Docker and remote Codex examples.
- `docs/requirements/mcp-client.md` / `docs/specs/mcp-client.md`
  - Record stable remote HTTP and access-key contract.
- `docs/change/*`
  - Archive implementation and validation.
- `docs/lessons/*`
  - Add only if validation exposes a reusable deployment/auth pitfall.

#### Interface Draft

CLI:

```powershell
myflowhub-mcp --transport http --listen 127.0.0.1:17688 --mcp-path /mcp
myflowhub-mcp --transport http --allow-remote --listen 0.0.0.0:17688 --mcp-path /mcp --auth-token <token>
```

Environment:

```powershell
$env:MYFLOWHUB_MCP_AUTH_TOKEN = "<token>"
```

HTTP:

```http
Authorization: Bearer <token>
```

Codex config:

```toml
[mcp_servers.myflowhub]
type = "http"
url = "https://example.com/mcp"

[mcp_servers.myflowhub.http_headers]
Authorization = "Bearer <token>"
```

Docker compose env draft:

```env
MYFLOWHUB_MCP_AUTH_TOKEN=change-me
MYFLOWHUB_MCP_ENDPOINT=10.3.3.5:9000
MYFLOWHUB_MCP_DEVICE_ID=ai-node
MYFLOWHUB_MCP_DISPLAY_NAME=AI MCP
MYFLOWHUB_MCP_ALLOW_WRITE=false
```

#### Error Handling and Safety

- Startup rejects remote mode without token.
- Startup rejects non-loopback listen without remote mode.
- HTTP unauthorized response is `401 Unauthorized`.
- Auth comparison uses constant-time equality.
- Token is never included in logs or structured errors.
- `--allow-write` remains a separate explicit gate; bearer auth does not imply write permission.
- Docs must state that public deployment should sit behind HTTPS, firewall, VPN, or reverse proxy. The built-in token prevents casual arbitrary access but does not replace transport encryption.

#### Performance and Testing Strategy

- Auth check is one header parse and constant-time compare per request.
- No additional Hub calls for request authentication.
- Tests:
  - Go unit tests around `internal/mcp`.
  - Go build for `cmd/myflowhub-mcp`.
  - Docker build if Docker is available.
  - Compose config validation if Docker Compose is available.
  - Script `-WhatIf` validation for remote config output.

#### Extensibility Design Points

- Token auth is transport-level and can later be replaced or supplemented by reverse proxy auth without touching tool handlers.
- Future independent `MyFlowHub-MCP` repo can keep the same CLI/env contract.
- Docker compose can later grow TLS/reverse-proxy profiles without changing the binary.
- The first remote auth contract intentionally stays simple: one fixed static bearer token configured by CLI/env and sent by Codex `http_headers`.

### Stage 3.1 - Planning

#### Project Goal and Current State

Current `main` already supports local shared HTTP MCP and `-EnsureRunning`, but it intentionally rejects non-loopback listen addresses. There is no Docker deployment entrypoint and no HTTP access-key authentication. This workflow adds a controlled remote Docker deployment mode.

#### Docs Governance Routing Decision

Using `$m-docs` for routing and impact checks.

- Requirements impact: clarify/add
  - Remote HTTP and access-key behavior extends the stable MCP client requirement.
- Specs impact: clarify/add
  - CLI/env/header/Docker contracts must be recorded.
- Lessons impact: none currently known
  - Add a lesson only if implementation/validation exposes a reusable pitfall.

Related requirements:

- `docs/requirements/mcp-client.md`

Related specs:

- `docs/specs/mcp-client.md`

Related lessons:

- `docs/lessons/powershell-args-automatic-variable.md`
- `docs/lessons/powershell-utf8-nobom-parse.md`

#### Executable Task List

- `MCP-REMOTE-1`: add remote HTTP auth and CLI/env config.
- `MCP-REMOTE-2`: add Docker image and compose deployment entrypoints.
- `MCP-REMOTE-3`: update install script and documentation.
- `MCP-REMOTE-4`: validate, review, and archive.

#### Task Details

##### MCP-REMOTE-1 - Remote HTTP Auth

- Owner: Codex
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-mcp-remote-docker-auth`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-mcp-remote-docker-auth/plan.md`
- Goal: allow remote HTTP only with explicit opt-in and bearer token.
- Files / Modules:
  - `cmd/myflowhub-mcp/main.go`
  - `internal/mcp/http_server.go`
  - `internal/mcp/http_server_test.go`
- Write Set: CLI and HTTP transport only.
- Acceptance:
  - local loopback works without token.
  - non-loopback without remote switch fails.
  - remote switch without token fails.
  - remote switch with token enforces `Authorization: Bearer <token>`.
- Test Points:
  - `go test ./internal/mcp -count=1`
  - targeted HTTP auth tests.
- Rollback:
  - revert CLI flags, env lookup, HTTP auth config, and tests.

##### MCP-REMOTE-2 - Docker Deployment Entry

- Owner: Codex
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-mcp-remote-docker-auth`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-mcp-remote-docker-auth/plan.md`
- Goal: provide repeatable Docker build and compose runtime for remote MCP.
- Files / Modules:
  - `Dockerfile.mcp`
  - `docker/mcp-entrypoint.sh`
  - `docker-compose.mcp.yml`
  - `.dockerignore`
- Write Set: Docker/deploy files only.
- Acceptance:
  - Dockerfile builds MCP binary only.
  - compose requires token env and mounts persistent config.
  - compose maps HTTP MCP port explicitly.
- Test Points:
  - `docker build -f Dockerfile.mcp .` if Docker is available.
  - `docker compose -f docker-compose.mcp.yml config` if Compose is available.
- Rollback:
  - remove Docker deployment files.

##### MCP-REMOTE-3 - Install Script And Docs

- Owner: Codex
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-mcp-remote-docker-auth`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-mcp-remote-docker-auth/plan.md`
- Goal: make remote deployment discoverable and Codex-configurable.
- Files / Modules:
  - `scripts/install-codex-myflowhub-mcp.ps1`
  - `README.md`
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `docs/change/README.md`
- Write Set: scripts and docs.
- Acceptance:
  - `-WhatIf` can show remote `type = "http"` config plus `http_headers.Authorization`.
  - README explains Docker env, token, HTTPS/firewall/VPN requirement, and `--allow-write` separation.
  - stable requirements/specs record remote auth behavior.
- Test Points:
  - install script `-Transport http -Url ... -AuthToken ... -WhatIf`.
  - `git diff --check`.
- Rollback:
  - revert script/docs changes.

##### MCP-REMOTE-4 - Validate, Review, Archive

- Owner: Codex
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-mcp-remote-docker-auth`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-mcp-remote-docker-auth/plan.md`
- Goal: prove the change and record it for closeout.
- Files / Modules:
  - `docs/change/2026-05-28_win-mcp-remote-docker-auth.md`
  - optional `docs/lessons/*`
- Write Set: archive docs only unless review finds a defect.
- Acceptance:
  - required validations pass or blockers are clearly recorded.
  - code review checklist passes.
  - change archive contains rollback and deployment caveats.
- Test Points:
  - Go tests/build.
  - Docker/Compose validation where available.
  - script preview.
  - `git diff --check`.
- Rollback:
  - revert all task-scoped files.

#### Dependencies

- Existing HTTP MCP server support from `1ecf3a2`.
- Existing ensure-running support from `906bdd1`.
- Codex HTTP MCP supports `[mcp_servers.<name>.http_headers]`, verified in local `~/.codex/config.toml`.
- Go toolchain matching `go 1.25.0`.
- Docker availability is not assumed; if unavailable locally, record Docker build as not run.

#### Risks and Notes

- Do not deploy an unauthenticated MCP endpoint.
- Do not conflate bearer token with Hub authorization. Hub role permissions and `--allow-write` still govern actual tool effects.
- If remote endpoint is public, TLS should be handled by reverse proxy or platform ingress.
- Worktree inherited a stale plan from the previous workflow and this document replaces it intentionally.

#### Parallelism Assessment

- No sub-agent delegation.
- The change spans one tightly coupled security/deployment surface; splitting would increase integration risk more than it saves time.

#### Issue List

- User confirmed that a fixed token is acceptable and that no complicated issuing/refresh flow should be introduced.

阻塞：否
进入 3.2
禁止派发子Agent

用户已确认固定 token 方案，进入实现。

### Stage 3.2 - Implementation Record

#### File-level Change Summary

- `MCP-REMOTE-1`
  - `cmd/myflowhub-mcp/main.go`
    - Added `--allow-remote` and `--auth-token`.
    - Reads `MYFLOWHUB_MCP_AUTH_TOKEN`.
    - Fails early when remote HTTP mode has no token.
  - `internal/mcp/http_server.go`
    - Added remote config and bearer-token enforcement.
    - Returns `401` for missing / malformed / wrong token before body handling.
    - Keeps loopback Origin check only for local mode.
  - `internal/mcp/http_server_test.go`
    - Added local/remote auth, missing token, wrong token, wrong scheme, and remote Origin coverage.
- `MCP-REMOTE-2`
  - `Dockerfile.mcp`
    - Multi-stage Go build for `cmd/myflowhub-mcp`.
    - Runtime image only contains the MCP binary and CA certs.
  - `docker/mcp-entrypoint.sh`
    - Maps environment variables to CLI flags.
    - Requires `MYFLOWHUB_MCP_AUTH_TOKEN`.
    - Adds `--allow-write` only for true-like env values.
  - `docker-compose.mcp.yml`
    - Declares remote HTTP service, port mapping, required token env, and persistent `/data` volume.
  - `.dockerignore`
    - Excludes generated and frontend-heavy paths from Docker build context.
  - `.gitattributes`
    - Forces `docker/*.sh` to LF to keep the entrypoint executable in Linux containers.
- `MCP-REMOTE-3`
  - `scripts/install-codex-myflowhub-mcp.ps1`
    - Adds `-AuthToken`.
    - Emits `http_headers.Authorization` for HTTP Codex config.
    - Suppresses local `EnsureRunning` hint for explicit remote URLs.
    - Fixes block replacement so a previous HTTP server block is replaced cleanly.
  - `README.md`
    - Adds remote Docker deployment steps and Codex remote URL example.
    - States that bearer token is transport auth, not Hub authorization.
  - `docs/requirements/mcp-client.md`
    - Adds remote Docker MCP requirement and fixed-token boundary.
  - `docs/specs/mcp-client.md`
    - Adds remote auth, token handling, and Docker contract.
- `MCP-REMOTE-4`
  - `docs/change/2026-05-28_win-mcp-remote-docker-auth.md`
  - `docs/change/README.md`
    - Added archive index entry.

#### Design Notes

- Remote auth stays at the transport boundary so tool handlers stay unchanged.
- Fixed token is enough for the current use case and keeps the deploy path simple.
- Docker runtime is intentionally minimal and does not pull in Wails GUI or frontend build steps.
- `Authorization: Bearer <token>` is required before body parsing in remote mode, so unauthenticated calls fail cheaply.
- Remote mode does not replace Hub login or `allow_write`; it only protects access to the MCP endpoint.

#### Validation Results

- `$env:GOWORK='off'; go test ./internal/mcp ./internal/mcpapp -count=1`
  - Passed.
- `$env:GOWORK='off'; go build -o $env:TEMP\myflowhub-mcp-remote-auth-test.exe ./cmd/myflowhub-mcp`
  - Passed.
- `$env:GOWORK='off'; $env:GOOS='linux'; $env:GOARCH='amd64'; go build -o $env:TEMP\myflowhub-mcp-linux-amd64-test ./cmd/myflowhub-mcp`
  - Passed.
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -Transport http -Url https://example.com/mcp -AuthToken test-token -WhatIf`
  - Passed.
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -Transport http -Listen 127.0.0.1:17688 -McpPath /mcp -AuthToken local-token -WhatIf`
  - Passed.
- Remote HTTP smoke with a real process:
  - Missing `Authorization` returned `401`.
  - `Authorization: Bearer test-token` returned `200`.
- `git diff --check`
  - Passed.
- Docker build / compose validation
  - Not run: `docker` is not installed / not on PATH in this environment.

### Stage 3.3 - Code Review

- 需求覆盖：通过
  - Remote HTTP access-key behavior, Docker deployment, and Codex header wiring are implemented and documented.
- 架构合理性：通过
  - Token auth is isolated at the HTTP transport boundary; runtime and tool handlers remain shared.
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - One header parse and constant-time compare per request; no extra Hub calls for auth.
- 可读性与一致性：通过
  - CLI, HTTP server, scripts, and docs follow the existing local MCP patterns.
- 可扩展性与配置化：通过
  - Token is fixed but configured via CLI/env; Docker and Codex URLs are declarative.
- 稳定性与安全：通过
  - Remote mode requires explicit opt-in, bearer token, and returns `401` before request body handling.
- 测试覆盖情况：通过
  - Go tests, build, Linux cross-build, script `-WhatIf`, and process-level auth smoke all passed.
- 子Agent治理与审计：通过
  - No sub-agents used; task IDs were kept bounded.

### Stage 4 - Change Archive

使用 `$m-docs` 完成 requirement/spec/lesson 影响复核。

- Requirements impact: updated
  - `docs/requirements/mcp-client.md`
- Specs impact: updated
  - `docs/specs/mcp-client.md`
- Lessons impact: none
  - No new reusable troubleshooting rule emerged beyond the existing PowerShell lessons.
- Change archive:
  - `docs/change/2026-05-28_win-mcp-remote-docker-auth.md`
- Index updates:
  - `docs/change/README.md`

已完成归档，请确认是否结束 workflow。
