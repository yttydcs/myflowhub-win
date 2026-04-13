# Todo - MyFlowHub-Win Thesis Code Comments

## Workflow Information
- Repo: `MyFlowHub-Win`
- Branch: `chore/thesis-code-comments`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\chore-thesis-code-comments-win`
- Current Stage: `3.2`
- Control Plan: `D:\project\MyFlowHub3\worktrees\chore-thesis-code-comments-control\todo.md`

## Baseline Notes
- 已同步未提交基线: `myflowhub-mcp.exe`
- 该二进制不属于本轮注释目标，只保留基线，不参与修改。

## Task Mapping
- Task ID: `WIN-1`
- Local Scope:
  - `app.go`
  - `cmd/**`
  - `internal/**`
  - `frontend/src/**`
  - `scripts/**`
- Exclusions:
  - `frontend/wailsjs/**`
  - `frontend/src/generated/**`
  - `myflowhub-mcp.exe`
  - `docs/**`
  - `node_modules/**`
  - `dist/**`
- Acceptance:
  - 关键 backend / frontend / MCP 文件已补足帮助理解的注释
- Validation:
  - diff 审查
- Rollback:
  - 回退 Win worktree 改动
