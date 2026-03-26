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
- Smoke against a real Hub with `register`:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode register`
- Smoke against a real Hub with `login`:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir "$env:APPDATA\\myflowhub\\mcp-codex" -NodeID 7`
- Force source mode:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 -PreferSource --version`
- Run directly:
  - `.\build\bin\myflowhub-mcp.exe --endpoint 127.0.0.1:9000 --config-dir "$env:APPDATA\\myflowhub\\mcp-codex" --device-id ai-node --display-name "AI MCP" --allow-write`
- Install into Codex (recommended):
  - `powershell -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -Endpoint 127.0.0.1:9000`
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
- The MCP process uses `stdio`; `stdout` is reserved for JSON-RPC and logs go to `stderr`.
- `scripts/start-myflowhub-mcp.ps1` first checks `MYFLOWHUB_MCP_EXE`, `build/bin/myflowhub-mcp.exe`, `.\myflowhub-mcp.exe`, and `.\bin\myflowhub-mcp.exe`; if none exist, it falls back to `go run ./cmd/myflowhub-mcp`.
- `scripts/install-codex-myflowhub-mcp.ps1` updates `~/.codex/config.toml` in place and supports `-WhatIf`.
- `scripts/test-myflowhub-mcp-smoke.ps1` drives the MCP process over line-delimited JSON-RPC and verifies `connect -> auth -> auth_get_perms -> auth_list_roles -> management_list_nodes`.
- `myflowhub_auth_get_perms` and `myflowhub_auth_list_roles` expose Hub-side permission self-check and role listing to AI hosts.
- `myflowhub_auth_list_pending_registers`, `myflowhub_auth_approve_register`, `myflowhub_auth_reject_register`, `myflowhub_auth_issue_register_permit`, and `myflowhub_auth_revoke_register_permit` expose authority approval flow to AI hosts.
- `myflowhub_management_config_get` and `myflowhub_management_config_list` expose read-only management config lookup; `config_set` remains intentionally unavailable.
- Use a dedicated `--config-dir` so the MCP node keeps separate `settings.json` and node keys from the GUI client.
- `login` smoke should use an existing dedicated `-ConfigDir`; `register` smoke will preserve any generated config dir so the same node keys can be reused after approval.
- Authority auth tools accept explicit `authority_id`; if omitted, the MCP layer first tries `authority.node_id` and then falls back to the hub target.
- Write tools such as `myflowhub_varstore_set` and `myflowhub_varstore_revoke` stay disabled unless `--allow-write` is set.
- `myflowhub_session_status` returns auth/defaults/config plus `permissions`, `readiness`, and `hints`.
- Tool failures return structured `code` / `message` / `hint` / `details`, which is the preferred machine-readable contract for AI hosts.

