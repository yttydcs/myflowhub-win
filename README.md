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
  - `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
- Start via script:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 --endpoint 127.0.0.1:9000 --device-id ai-node --display-name "AI MCP"`
- Run directly:
  - `.\myflowhub-mcp.exe --endpoint 127.0.0.1:9000 --config-dir "$env:APPDATA\\MyFlowHub\\mcp-client" --device-id ai-node --display-name "AI MCP" --allow-write`
- Typical MCP host config:

```json
{
  "mcpServers": {
    "myflowhub": {
      "command": "powershell.exe",
      "args": [
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        "D:\\project\\MyFlowHub3\\worktrees\\win-mcp-ai-client\\scripts\\start-myflowhub-mcp.ps1",
        "--endpoint",
        "127.0.0.1:9000",
        "--config-dir",
        "C:\\Users\\<user>\\AppData\\Roaming\\MyFlowHub\\mcp-client",
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
- `scripts/start-myflowhub-mcp.ps1` is a thin wrapper around `go run ./cmd/myflowhub-mcp` and forwards any extra CLI flags unchanged.
- Use a dedicated `--config-dir` so the MCP node keeps separate `settings.json` and node keys from the GUI client.
- Write tools such as `myflowhub_varstore_set` and `myflowhub_varstore_revoke` stay disabled unless `--allow-write` is set.

