# Win MCP Authority Flow And Config Read

## 变更背景 / 目标

- 在前一轮 `auth_get_perms/auth_list_roles` 和真实 Hub smoke 落地后，MCP client 仍缺两块关键能力：
  - AI 无法直接处理 authority 审批流，无法查看 pending register 或签发 permit
  - AI 无法读取 management config，因此 authority 不是 hub 本身时，路由选择依赖外部知识
- 本轮目标是在不接入 `localhub`、不开放 `config_set`、不增加新的本地授权层的前提下：
  - 暴露 authority 审批流工具
  - 暴露只读 management config 工具
  - 让 authority 类 auth 工具支持 `authority_id -> authority.node_id -> hub target` 的解析顺序

## 具体变更内容

- 更新 `internal/mcpapp/runtime.go`
  - 新增 auth authority wrapper：
    - `ListPendingRegisters`
    - `ApproveRegister`
    - `RejectRegister`
    - `IssueRegisterPermit`
    - `RevokeRegisterPermit`
  - 新增 management wrapper：
    - `ConfigGet`
    - `ConfigList`
- 更新 `internal/mcp/tools.go`
  - 新增工具：
    - `myflowhub_auth_list_pending_registers`
    - `myflowhub_auth_approve_register`
    - `myflowhub_auth_reject_register`
    - `myflowhub_auth_issue_register_permit`
    - `myflowhub_auth_revoke_register_permit`
    - `myflowhub_management_config_get`
    - `myflowhub_management_config_list`
  - 为 authority 类 auth 工具新增 authority 路由解析：
    - 显式 `authority_id`
    - 否则尝试 `config_get("authority.node_id")`
    - 再回退 hub target
  - 让 `auth_get_perms` / `auth_list_roles` 也复用同一 authority 路由解析
  - 为 pending list / permit expiry 增加本地参数校验
- 更新 `internal/mcp/tools_test.go`
  - 扩展 fake backend 覆盖 authority/config 方法
  - 新增 authority 解析、显式 authority override、config read、参数校验测试
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
- `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `DOCS-2`
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
- `IMPL-4`
  - `internal/mcp/tools.go`
  - `internal/mcp/tools_test.go`
  - `internal/mcpapp/runtime.go`
- `IMPL-5`
  - `internal/mcp/tools.go`
  - `internal/mcp/tools_test.go`
  - `internal/mcpapp/runtime.go`
- `TEST-2`
  - `internal/mcp/tools_test.go`
  - 全量 `go test` / `go build`
- `ARCHIVE-2`
  - `docs/change/README.md`
  - `docs/change/2026-03-26_win-mcp-authority-config-read.md`

## 经验 / 教训摘要

- authority 类 auth 工具如果只沿用 `hub_id` 作为 target，在 authority 独立部署时会让 AI 很难稳定命中真实 authority。
- `config_get` 只读能力对 AI 来说不是“锦上添花”，它直接决定后续 auth / management 路由是否能自洽。
- `config_set` 即便底层 service 已有，也不应该因为加了 `config_get` 就顺手开放；读写边界必须单独评估。

## 可复用排查线索

- 症状：
  - AI 能登录，但看不到 pending register，也无法走 approval/permit 流
  - `auth_get_perms` / `auth_list_roles` 在 authority 独立部署时命中错误目标
  - 需要读取 `authority.node_id`、`auth.*` 等配置键来辅助决策
- 触发条件：
  - MCP 工具只支持基础 auth，不支持 authority 管理动作
  - authority 不是 hub 自身
  - host 只知道 hub target，不知道 authority target
- 关键词：
  - `myflowhub_auth_list_pending_registers`
  - `myflowhub_auth_issue_register_permit`
  - `myflowhub_management_config_get`
  - `authority.node_id`
  - `authority_id`
- 快速检查：
  - `tools/list` 中确认 7 个新工具已出现
  - 先调 `myflowhub_management_config_get {"key":"authority.node_id"}`
  - 再调 `myflowhub_auth_list_pending_registers`
  - 若显式知道 authority，直接传 `authority_id`

## 关键设计决策与权衡

- 决策：不接入 `localhub`
  - 原因：这是宿主机控制面，不是当前 AI 作为 Hub 节点工作流的刚需。
- 决策：authority 路由解析放在 MCP tool 层
  - 原因：复用现有 session/auth/management 服务即可，不需要新增一层本地授权系统。
- 决策：`auth_get_perms` / `auth_list_roles` 也切到 authority 路由解析
  - 原因：避免旧 auth 查询工具和新 authority 工具在目标选择上出现双重语义。
- 决策：`management_config_set` 继续关闭
  - 原因：本轮只补观测能力，不扩大到远程配置写入。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke`
  - 结果：通过
- `$env:GOWORK='off'; go test ./... -count=1`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke`
  - 结果：通过
- `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke`
  - 结果：通过
- `git diff --check`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-perms-smoke`
  - 结果：通过

## 潜在影响与回滚方案

- 潜在影响：
  - Host 现在会看到 7 个新增工具
  - `auth_get_perms` / `auth_list_roles` 的目标解析行为从“直接 hub target”变为“优先 authority”
  - 当角色缺少 `config_get` 权限时，authority 自动解析会回退到 hub target
- 回滚方案：
  - 回退 `internal/mcp/tools.go`、`internal/mcp/tools_test.go`、`internal/mcpapp/runtime.go`
  - 回退 requirements/specs/README 中新增 authority/config read 契约
  - 删除 `docs/change/2026-03-26_win-mcp-authority-config-read.md`

## 子Agent执行轨迹

- none
