# Win MCP Auth Perms And Smoke

## 变更背景 / 目标

- Win MCP client 已经具备基础 `session/auth/management/varstore` 链路，但这一轮还缺两块关键补足：
  - AI 无法直接通过 MCP 查询“当前节点到底拿到了哪些权限”
  - 仓内缺少一个可直接连真实 Hub 的无界面 smoke 入口，导致每次都要手工拼 JSON-RPC 或依赖 GUI 侧验证
- 本轮目标是在不新增本地权限层的前提下：
  - 暴露 `auth_get_perms` / `auth_list_roles`
  - 提供独立 `config_dir` 的真实 Hub smoke 脚本
  - 把这两部分写入稳定 requirements/specs/README

## 具体变更内容

- 更新 `internal/mcpapp/runtime.go`
  - 新增 `GetPerms` / `ListRoles` runtime wrapper
  - 继续复用现有 auth service 与 timeout 约束
- 更新 `internal/mcp/tools.go`
  - 新增 `myflowhub_auth_get_perms`
  - 新增 `myflowhub_auth_list_roles`
  - 为 `auth_get_perms` 增加默认 `node_id` 回退逻辑
  - 为 `auth_list_roles` 增加 `offset/limit/role/node_ids` 参数透传与本地校验
  - 继续沿用现有结构化错误模型和 Hub-side RBAC 边界
- 更新 `internal/mcp/tools_test.go`
  - 为 fake backend 补 `GetPerms` / `ListRoles`
  - 新增默认身份回退、过滤参数透传、非法 `node_ids` 校验测试
- 新增 `scripts/test-myflowhub-mcp-smoke.ps1`
  - 复用 `scripts/start-myflowhub-mcp.ps1` 拉起 MCP 进程
  - 通过 line-delimited JSON-RPC 驱动：
    - `initialize`
    - `tools/list`
    - `myflowhub_session_connect`
    - `myflowhub_auth_register` 或 `myflowhub_auth_login`
    - `myflowhub_auth_get_perms`
    - `myflowhub_auth_list_roles`
    - `myflowhub_management_list_nodes`
  - `login` 模式要求显式 `-ConfigDir`
  - `register` 返回 `pending` 时显式失败，并提示保留 config dir 以便后续 login
- 更新稳定文档
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`

## Requirements impact: updated

## Specs impact: updated

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke\docs\requirements\mcp-client.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke\docs\specs\mcp-client.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `DOCS-1`
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
- `IMPL-1`
  - `internal/mcpapp/runtime.go`
  - `internal/mcp/tools.go`
  - `internal/mcp/tools_test.go`
- `IMPL-2`
  - `scripts/test-myflowhub-mcp-smoke.ps1`
- `IMPL-3`
  - README 对齐
  - 本地脚本帮助与参数失败路径验证
  - Go 回归与构建验证
- `ARCHIVE-1`
  - `docs/change/README.md`
  - `docs/change/2026-03-26_win-mcp-auth-perms-smoke.md`

## 经验 / 教训摘要

- 面向 AI 的 MCP client，权限自检不应只靠 `session_status` 猜测；直接提供 `get_perms/list_roles` 能明显降低调用前的不确定性。
- 真实 Hub smoke 若要可复用，必须自己驱动 JSON-RPC，而不是把“如何手动点一遍”留在 README 里。
- 审批制 register 流如果不在脚本层显式失败，AI 或调用者很容易把 `pending` 误判成“注册成功但还没刷新状态”。

## 可复用排查线索

- 症状：
  - AI 已登录，但不知道自己是否具备 `auth.list_roles` 或其他 authority 权限
  - 需要在无 GUI 情况下验证 MyFlowHub MCP 到真实 Hub 的基础链路
  - `login` smoke 失败，但不确定是缺 node keys 还是缺 `node_id`
- 触发条件：
  - MCP tool 集合缺少 auth 权限查询工具
  - 认证脚本直接复用 GUI 配置目录或空目录
  - Hub 开启 `register.require_approval`
- 关键词：
  - `myflowhub_auth_get_perms`
  - `myflowhub_auth_list_roles`
  - `test-myflowhub-mcp-smoke.ps1`
  - `pending approval`
  - `ConfigDir`
- 快速检查：
  - `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Help`
  - `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login`
  - `tools/list` 中确认新增 auth 工具
  - 登录后直接调 `myflowhub_auth_get_perms`

## 关键设计决策与权衡

- 决策：不增加新的本地 owner/target 白名单
  - 原因：用户已明确 Hub-side RBAC 才是授权真相，本地只保留既有 `allow_write` gate。
- 决策：`auth_list_roles` 直接暴露完整过滤面
  - 原因：避免脚本或 host 端再拼接多轮查询，保持与协议能力一致。
- 决策：真实 Hub smoke 复用 `start-myflowhub-mcp.ps1`
  - 原因：避免二进制优先级、源码 fallback、环境参数在两套脚本里分叉。
- 决策：`login` smoke 强制要求 `-ConfigDir`
  - 原因：登录必须依赖已有 node keys，默认回到 GUI 配置目录或空目录都会制造误判。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke`
  - 结果：通过
- `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Help`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke`
  - 结果：通过
- `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke`
  - 结果：按预期失败，明确提示 `Login mode requires -ConfigDir...`
- `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke`
  - 结果：通过
- `$env:GOWORK='off'; go test ./... -count=1`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke`
  - 结果：通过
- 真实 Hub 端到端 smoke
  - 结果：本轮未在当前环境实际连真实 Hub 执行，需用户在目标 Hub 环境下运行脚本确认

## 潜在影响与回滚方案

- 潜在影响：
  - 依赖旧工具集合的 host 现在会看到两个新增 auth 工具
  - 若当前角色不具备 authority 查询权限，`auth_list_roles` 会返回 Hub 侧权限失败
  - smoke 脚本把 `pending` 视为失败，调用方需要按审批制流程处理，而不是把它当成功
- 回滚方案：
  - 回退 `internal/mcpapp/runtime.go`、`internal/mcp/tools.go`、`internal/mcp/tools_test.go`
  - 删除 `scripts/test-myflowhub-mcp-smoke.ps1`
  - 回退 requirements/specs/README 中新增 auth 工具与 smoke 描述

## 子Agent执行轨迹

- none
