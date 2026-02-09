# MyFlowHub-Win (Wails)

## Prerequisites
- Go (per `go.mod`)
- Node.js + npm
- Wails CLI (`wails version`)

## Dev (recommended)
1) Start server (defaults to `:9000`):
   - `cd ../MyFlowHub-Server`
   - `go run ./cmd/hub_server`
2) Start the app:
   - `cd ../myflowhub-win_remove-fyne`
   - `wails dev`
3) Smoke test (in the app Home page):
   - Address: `127.0.0.1:9000`
   - Device ID: any non-empty string (e.g. `dev-1`)
   - Click **Connect**
   - Click **Register** (NodeID=0) or **Login** (if NodeID already assigned)
   - Expect: status shows **Connected** and no obvious errors in Logs.

## Build (Windows)
- `cd d:\project\MyFlowHub3\worktrees\myflowhub-win_remove-fyne`
- `wails build -platform windows/amd64`

Output:
- `build/bin/myflowhub-win.exe`

Notes:
- This build embeds `frontend/dist`. The build command runs `npm install` + `npm run build` automatically (per `wails.json`).

