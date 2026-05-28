# MyFlowHub-Win (Wails)

## Prerequisites
- Go (per `go.mod`)
- Node.js + npm
- Wails CLI (`wails version`)

## Fresh Worktree Bootstrap (PowerShell)
When a new worktree is created, `frontend/wailsjs/` is not present because it is generated and ignored by git.
Before running frontend-only validation in a fresh worktree, use this bootstrap sequence from the repo root:

```powershell
wails version
$env:GOWORK='off'; wails generate module
cd frontend
npm ci
npm run build
```

Notes:
- `wails generate module` restores `frontend/wailsjs/**`, which the frontend build depends on.
- `wails dev` / `wails build` may also regenerate bindings, but the explicit generate step is the recommended preflight for fresh worktrees.
- Re-run `$env:GOWORK='off'; wails generate module` after backend binding changes or if `frontend/wailsjs/**` is missing.

## Dev (recommended)
1) Start server (defaults to `:9000`):
   - `cd ../MyFlowHub-Server`
   - `go run ./cmd/hub_server`
2) Start the app:
   - `cd ../MyFlowHub-Win`
   - `wails dev`
3) Smoke test:
   - Address: `127.0.0.1:9000`
   - Device ID: any non-empty string (e.g. `dev-1`)
   - Click **Connect**
   - Go to **Presets** → **Node Echo** → click **Send**
   - Expect: toast shows success (e.g. “Node echo sent.”) and no obvious errors in Logs.

## Build (Windows)
- `wails build -platform windows/amd64`

Output:
- `build/bin/myflowhub-win.exe`

Notes:
- This build embeds `frontend/dist`. The build command runs `npm install` + `npm run build` automatically (per `wails.json`).

## MCP CLI
- Build:
  - `$env:GOWORK='off'; go build -o .\build\bin\myflowhub-mcp.exe ./cmd/myflowhub-mcp`
- Start via script:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 --endpoint 127.0.0.1:9000 --device-id ai-node --display-name "AI MCP"`
- Start shared local HTTP MCP server:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 --transport http --listen 127.0.0.1:17688 --mcp-path /mcp --endpoint 127.0.0.1:9000 --device-id ai-node --display-name "AI MCP"`
- Ensure shared local HTTP MCP server is running:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 -EnsureRunning --endpoint 127.0.0.1:9000 --device-id ai-node --display-name "AI MCP"`
- Smoke against a real Hub with `register`:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode register`
- Smoke against a real Hub with `login`:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir "$env:APPDATA\\myflowhub\\mcp-codex" -NodeID 7`
- Extended read smoke:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir "$env:APPDATA\\myflowhub\\mcp-codex" -EnableExtendedRead -ExecutorNode 42`
- Authority smoke:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir "$env:APPDATA\\myflowhub\\mcp-codex" -EnableAuthoritySmoke -AuthorityID 7 -PermitDeviceID temp-device -PermitRole operator`
- Write smoke:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir "$env:APPDATA\\myflowhub\\mcp-codex" -EnableWriteSmoke -ExecutorNode 42 -FlowMethod demo::run`
- Force source mode:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 -PreferSource --version`
- Run directly:
  - `.\build\bin\myflowhub-mcp.exe --endpoint 127.0.0.1:9000 --config-dir "$env:APPDATA\\myflowhub\\mcp-codex" --device-id ai-node --display-name "AI MCP" --allow-write`
- Install into Codex (recommended):
  - `powershell -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -Endpoint 127.0.0.1:9000`
- Install shared HTTP endpoint into Codex:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -Transport http -Listen 127.0.0.1:17688 -McpPath /mcp`
- Preview the Codex config change:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -Endpoint 127.0.0.1:9000 -WhatIf`
- Manual MCP host config:

```json
{
  "mcpServers": {
    "myflowhub": {
      "command": "powershell.exe",
      "args": [
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        "D:\\path\\to\\MyFlowHub-Win\\scripts\\start-myflowhub-mcp.ps1",
        "--endpoint",
        "127.0.0.1:9000",
        "--config-dir",
        "C:\\Users\\<user>\\AppData\\Roaming\\myflowhub\\mcp-codex",
        "--device-id",
        "ai-node",
        "--display-name",
        "AI MCP"
      ]
    }
  }
}
```

Notes:
- The MCP process supports `stdio` for single-client hosts and local HTTP for shared multi-client hosts.
- In `stdio` mode, `stdout` is reserved for JSON-RPC and logs go to `stderr`.
- In HTTP mode, all Codex sessions that point at the same local URL share one process, one runtime, and one Hub connection.
- `scripts/start-myflowhub-mcp.ps1 -EnsureRunning` probes the HTTP MCP endpoint, reuses it when ready, and starts a hidden background server only when the endpoint is unavailable. It is a lightweight launcher mode, not a Windows service.
- `scripts/start-myflowhub-mcp.ps1` first checks `MYFLOWHUB_MCP_EXE`, `build/bin/myflowhub-mcp.exe`, `.\myflowhub-mcp.exe`, and `.\bin\myflowhub-mcp.exe`; if none exist, it falls back to `go run ./cmd/myflowhub-mcp`.
- `scripts/install-codex-myflowhub-mcp.ps1` updates `~/.codex/config.toml` in place, supports `-WhatIf`, and can generate either `stdio` or `http` MCP config.
- `scripts/test-myflowhub-mcp-smoke.ps1` drives the MCP process over line-delimited JSON-RPC with staged smoke:
  - default base chain: `connect -> auth -> auth_get_perms -> auth_list_roles -> management_list_nodes`
  - `-EnableExtendedRead`: management/config/exec/flow-read checks
  - `-EnableAuthoritySmoke`: pending list plus optional permit / approve / reject actions
  - `-EnableWriteSmoke`: varstore + flow write checks with temporary resources and cleanup
- `myflowhub_auth_get_perms` and `myflowhub_auth_list_roles` expose Hub-side permission self-check and role listing to AI hosts.
- `myflowhub_auth_list_pending_registers`, `myflowhub_auth_approve_register`, `myflowhub_auth_reject_register`, `myflowhub_auth_issue_register_permit`, and `myflowhub_auth_revoke_register_permit` expose authority approval flow to AI hosts.
- `myflowhub_management_node_echo` and `myflowhub_management_list_subtree` expose read-only management reachability and subtree discovery.
- `myflowhub_management_config_get` and `myflowhub_management_config_list` expose read-only management config lookup; `config_set` remains intentionally unavailable.
- `myflowhub_exec_cap_query` exposes read-only exec capability discovery; `exec.call` remains intentionally unavailable.
- `myflowhub_flow_list`, `myflowhub_flow_get`, `myflowhub_flow_status`, `myflowhub_flow_set`, `myflowhub_flow_run`, and `myflowhub_flow_delete` expose flow deployment and run control to AI hosts.
- For exec tools, `target_id` is the transport route target while `requester_node` is the identity written into the exec payload; omitted `requester_node` falls back to `source_id`.
- For flow tools, `target_id` is the transport route target while `executor_node` is the actual flow executor; omitted `executor_node` falls back to `target_id`.
- Use a dedicated `--config-dir` so the MCP node keeps separate `settings.json` and node keys from the GUI client.
- `login` smoke should use an existing dedicated `-ConfigDir`; `register` smoke will preserve any generated config dir so the same node keys can be reused after approval.
- Authority and write stages are opt-in because they mutate real Hub state.
- Extended read prefers `-ConfigKey` for `management_config_get`; if that key is unavailable it falls back to the first readable key from `config_list`, and if no readable key exists it reports `config_get` as skipped.
- Write smoke automatically starts MCP with `--allow-write`, but it still requires explicit `-ExecutorNode` and `-FlowMethod`.
- Write smoke cleans up the temporary flow and variable before exit; cleanup failures are reported explicitly so residual resources can be removed manually.
- Authority auth tools accept explicit `authority_id`; if omitted, the MCP layer first tries `authority.node_id` and then falls back to the hub target.
- Write tools such as `myflowhub_flow_set`, `myflowhub_flow_run`, `myflowhub_flow_delete`, `myflowhub_varstore_set`, and `myflowhub_varstore_revoke` stay disabled unless `--allow-write` is set.
- `myflowhub_session_status` returns MCP server transport info, auth/defaults/config, `permissions`, `readiness`, and `hints`.
- Tool failures return structured `code` / `message` / `hint` / `details`, which is the preferred machine-readable contract for AI hosts.

