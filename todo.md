# Plan - MyFlowHub-Win MCP Authority Flow And Config Read

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `main`
- Source Branch: `feat/win-mcp-perms-smoke` (merged and deleted)
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke` (removed after merge)
- Current Stage: `workflow ended / merged to main / worktree removed`

## Goal And Current State
- Goal:
  - 为 Win MCP client 增加 authority 审批流工具，并补充只读 management config 查询。
- Current State:
  - 上一轮已完成 `auth_get_perms/auth_list_roles` 与真实 Hub smoke 脚本。
  - 当前 MCP 仍缺 authority 管理动作，AI 无法直接查看 pending register 或发放 permit。
  - 当前 `management` 只暴露 `list_nodes/node_info`，还不能直接查询 `authority.node_id` 等配置。
  - `localhub` 已明确排除，不纳入本轮范围。

## Docs Governance
- Requirements impact: updated
- Specs impact: updated
- Related requirements:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\mcp-client.md`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\mcp-client.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
  - `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
- Related lessons:
  - none

## Scope
- Must:
  - 新增 `myflowhub_auth_list_pending_registers`
  - 新增 `myflowhub_auth_approve_register`
  - 新增 `myflowhub_auth_reject_register`
  - 新增 `myflowhub_auth_issue_register_permit`
  - 新增 `myflowhub_auth_revoke_register_permit`
  - 新增 `myflowhub_management_config_get`
  - 新增 `myflowhub_management_config_list`
  - 为 authority 类 auth 工具增加 authority 路由解析
  - 更新 requirements/specs/README
  - 补相关测试
- Optional:
  - 让既有 `auth_get_perms/auth_list_roles` 也复用 authority 路由解析
- Out of scope:
  - `localhub` 接入 MCP
  - `management_config_set`
  - `topicbus` / `flow` / `file` 接入
  - 新增本地额外权限层

## Parallelism Assessment
- 写集集中在 `internal/mcp/*`、`internal/mcpapp/*`、`docs/*`、`README.md`，并且 authority 路由设计会影响多个 auth 工具，不派发子 Agent。
- Owner: 主 Agent

## Executable Checklist
- [x] DOCS-2 更新 requirements/specs/README，补 authority 工具与 config read 契约
- [x] IMPL-4 暴露 authority 审批流 MCP 工具
- [x] IMPL-5 暴露 management config 只读 MCP 工具并补 authority 路由解析
- [x] TEST-2 补工具测试并完成回归验证
- [x] REVIEW-2 完成 3.3 代码复核
- [x] ARCHIVE-2 归档到 `docs/change`

## Task Details
### DOCS-2 - 稳定文档更新
- Goal:
  - 将 authority 审批流和 management config read 写入 requirements/specs/README。
- Files:
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
- Acceptance:
  - 文档能描述新工具、参数、authority 路由回退和不开放 `config_set` 的边界
- Tests:
  - 文档自检
- Rollback:
  - 回退文档更新

### IMPL-4 - Authority 审批流工具
- Goal:
  - 让 AI 能查看 pending register、批准/拒绝注册、发放/撤销 permit。
- Files:
  - `internal/mcp/tools.go`
  - `internal/mcp/tools_test.go`
  - `internal/mcpapp/runtime.go`
- Acceptance:
  - `tools/list` 包含 5 个新的 authority auth 工具
  - 支持参数校验与结构化错误
  - 支持显式 `authority_id` 或 authority 自动解析
- Tests:
  - `go test ./internal/mcp -count=1`
- Rollback:
  - 回退新增工具与 runtime wrapper

### IMPL-5 - Management Config Read And Authority Routing
- Goal:
  - 让 AI 能读取目标节点 config，并让 auth authority 工具能优先命中真实 authority。
- Files:
  - `internal/mcp/tools.go`
  - `internal/mcp/tools_test.go`
  - `internal/mcpapp/runtime.go`
- Acceptance:
  - `tools/list` 包含 `myflowhub_management_config_get` / `myflowhub_management_config_list`
  - authority 路由优先 `authority_id`，否则尝试读取 `authority.node_id`，失败时回退 hub target
  - 不开放 `management_config_set`
- Tests:
  - `go test ./internal/mcp -count=1`
- Rollback:
  - 回退 management config 工具与 authority 路由改动

### TEST-2 - 回归验证
- Goal:
  - 确保新增 authority/config 工具不破坏现有 MCP 行为。
- Files:
  - `internal/mcp/tools_test.go`
  - 如有必要的 README 验证说明
- Acceptance:
  - 单测覆盖 authority 参数校验与 authority 解析路径
  - 构建与全量回归通过
- Tests:
  - `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - `$env:GOWORK='off'; go test ./... -count=1`
  - `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
- Rollback:
  - 回退新增测试与验证说明

## Risks And Notes
- authority 可能不是 hub 自身；路由回退必须清晰，避免把 authority 类请求错误打到普通 hub target。
- `config_get` 可能返回本地 404 或上游权限错误；MCP 错误提示要保留下一步动作。
- `config_set` 虽然 Win service 已支持，但本轮继续保持关闭，避免把读配置和写配置一起放开。

## Stage 3.3 Review
- 需求覆盖：通过
  - authority 审批流工具、management config read、authority 路由回退、README/requirements/specs 更新均已落地。
- 架构合理性：通过
  - 继续复用现有 auth/management/runtime，不引入本地额外权限层。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - authority 自动解析最多增加一次 `config_get`，其余为薄封装。
- 可读性与一致性：通过
  - 新工具命名、错误结构、返回模型与既有 MCP 工具保持一致。
- 可扩展性与配置化：通过
  - authority 支持显式 override，也支持自动读取 `authority.node_id`；`config_set` 仍保持关闭。
- 稳定性与安全：通过
  - `localhub` 未接入；`config_get/list` 为只读；真实授权边界仍在 Hub。
- 测试覆盖情况：通过
  - `go test ./internal/mcp -count=1`
  - `$env:GOWORK='off'; go test ./... -count=1`
  - `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - `git diff --check`
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子 Agent。

## Archive Output
- Change:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\change\2026-03-26_win-mcp-auth-perms-smoke.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\change\2026-03-26_win-mcp-authority-config-read.md`
- Index updated:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\change\README.md`
- Lessons impact:
  - none

## Workflow End Record
- User confirmation:
  - `2026-03-26` confirmed ending the workflow
- Merge result:
  - source branch fast-forwarded into `main` at commit `4643062`
- Cleanup:
  - removed worktree `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke`
  - ran `git worktree prune`
  - deleted local branch `feat/win-mcp-perms-smoke`

阻塞：否
Workflow 已结束并已合并到 `main`
worktree 已移除并完成清理
