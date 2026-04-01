# Plan - Win MCP Full Chain Smoke

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Merge Source Branch: `feat/win-mcp-full-chain-smoke`
- Base Branch: `main`
- Former Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
- Current Stage: `workflow ended`

## Summary
- 目标：
  - 把 Win MCP 的真实 Hub smoke 从基础链路扩展成 staged full-chain 验证入口，同时保持默认安全。
- 结果：
  - 已合并 staged smoke 脚本、稳定文档更新和 change 归档。
  - authority / write 阶段默认关闭，只有显式 opt-in 时才执行。
  - write 阶段会自动追加 `--allow-write`，并对临时 flow / var 执行显式 cleanup。

## Key Outputs
- 脚本：
  - `scripts/test-myflowhub-mcp-smoke.ps1`
- 稳定文档：
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
- 归档：
  - `docs/change/2026-04-01_win-mcp-full-chain-smoke.md`
- 详细执行记录：
  - `todo.md`

## Stage Outcome
- Stage 1: completed
- Stage 2: completed
- Stage 3.1: completed
- Stage 3.2: completed
- Stage 3.3: completed
- Stage 4: completed

## Validation Snapshot
- `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - passed
- `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - passed
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Help`
  - passed
- key local negative-path checks
  - passed
- real Hub staged smoke
  - not executed in this environment

## Notes
- `todo.md` 已作为本轮 workflow 的完整执行记录保留在主仓。
- `docs/change/2026-04-01_win-mcp-full-chain-smoke.md` 是本轮结果归档，不作为长期 requirements/specs 真相来源。
- 当前主仓仍存在用户自己的未提交状态，未被本次 workflow 清理或回退。
