# MCP Client

## Background

- 当前 `MyFlowHub-Win` 只提供面向用户的 Wails GUI 客户端，没有面向 AI host 的无界面入口。
- 变量读写、节点查询、认证等协议能力已经在 Win 后端服务中存在，但 AI 若要复用这些能力，不能依赖“去操作一个已经打开的 GUI 窗口”。
- 用户已确认 AI 应以独立客户端身份接入 Hub，因此该能力需要独立节点身份、独立本地配置目录，以及与 GUI 进程隔离的生命周期。

## Goal

- 为 `MyFlowHub-Win` 提供一个可被 MCP host 以 `stdio` 方式拉起的无界面客户端。
- 该客户端以独立节点身份连接 Hub，并向 AI 暴露首版 `session`、`auth`、`management`、`flow`、`varstore` 工具。
- `auth` 工具除 `register/login` 外，还要支持权限自检与角色分布查询。
- 首版默认只开放读能力与受控写能力，不与现有 GUI 本地配置互相污染。
- MCP 返回结果应足够稳定，让 AI 能直接判断当前阻塞类型和下一步动作。

## Scope

### Must

- 提供无界面可执行入口，不依赖 Wails GUI。
- 支持 MCP `tools/list` / `tools/call`。
- 支持以下首版能力：
  - `session connect/disconnect/status`
  - `auth register/login/get_perms/list_roles/list_pending_registers/approve_register/reject_register/issue_register_permit/revoke_register_permit`
  - `management list_nodes/node_info/config_get/config_list`
  - `flow list/get/status/set/run/delete`
  - `varstore list/get/set/revoke`
- 作为独立节点身份接入 Hub。
- 使用独立本地配置目录保存 MCP 自己的 settings 与 node keys。
- 写工具默认受显式开关保护。
- 日志只能写 `stderr`，不得污染 MCP `stdout`。
- 仓内提供一个可直接连真实 Hub 的 smoke 脚本，覆盖 `initialize -> tools/list -> connect -> auth -> auth_get_perms -> auth_list_roles -> management_list_nodes`。

### Optional

- 允许通过启动参数设置默认 `endpoint`、`device_id`、`display_name`、`timeout`。
- 为后续接入 `topicbus`、`subscribe/unsubscribe` 预留扩展空间。

### Out of Scope

- 不复用现有 GUI Win 的会话、节点身份或配置目录。
- 首版不开放 `config_set`。
- 首版不开放 `ExecCapQuery`；若需要执行器能力发现，后续按 `exec` 能力单独规划。
- 首版不通过 UI 自动化去操作现有 Win 界面。
- 首版不实现“连接第三方 MCP Server”的通用 client。

## Scenarios

- AI host 启动 `myflowhub-mcp`，连接 Hub 并执行 `register/login`。
- AI 在完成 `register/login` 后，立即用 `auth_get_perms` / `auth_list_roles` 校验当前身份是否具备预期权限。
- AI 通过 `auth_list_pending_registers` 查看待审批注册，并用 `approve/reject/permit` 工具完成 authority 管理动作。
- AI 在调用写工具前先通过 `session_status` 判断是否已连接、是否已登录、写 gate 是否开启。
- AI 列出当前节点，定位 Hub 或目标节点。
- AI 读取 `authority.node_id` 等 management config，辅助决定 auth 工具应命中哪个 authority。
- AI 向某个执行节点下发 flow 定义，或读取 / 列出 / 运行 / 删除某个 flow。
- AI 读取某个变量、列出变量名、创建变量、修改变量、撤销变量。
- 用户同时运行 GUI Win 客户端和 MCP 客户端，两者互不干扰。
- 用户通过脚本把 MCP 安装到 Codex，而不是手动编辑 host 配置。
- 用户通过 smoke 脚本直连真实 Hub，验证独立 `config_dir`、auth、权限查询与 management 基础链路。

## Functional Requirements

1. MCP 客户端必须支持标准 `stdio` 传输。
2. 客户端必须维持长连接 session，而不是每次 tool 调用重新拨号。
3. 首版必须显式支持 `auth register/login`，因为未认证连接默认只能访问 auth 子协议。
4. 首版必须显式支持 `auth get_perms/list_roles`，用于 AI 在登录后自检当前节点权限和 authority 角色分布。
5. MCP 必须显式支持 authority 审批流工具：`list_pending_registers`、`approve_register`、`reject_register`、`issue_register_permit`、`revoke_register_permit`。
6. MCP 必须显式支持只读 management config 查询：`config_get`、`config_list`。
7. authority 类 auth 工具必须支持显式 `authority_id`，未传时优先尝试读取 `authority.node_id`，失败时再回退到 hub target。
8. 仓内必须提供真实 Hub smoke 脚本，至少支持 `register` 与 `login` 两种认证模式。
9. smoke 脚本在 `register` 返回 `pending` / `rejected` 时必须明确失败，不得把审批中状态误判为通过。
10. smoke 脚本在 `login` 模式下必须要求或复用独立 `config_dir` 中已有 node keys，不得隐式回落到 GUI 配置目录。
11. auth 成功后，客户端必须维护最近一次成功的默认身份状态，至少包含：
   - `device_id`
   - `node_id`
   - `hub_id`
   - `role`
12. 业务工具必须允许显式传入 `source_id` / `target_id`；未传时可按默认身份状态回退。
13. `flow` 工具必须明确区分 `target_id` 与 `executor_node`：`target_id` 作为传输目标，`executor_node` 作为实际执行节点；未传 `executor_node` 时默认回退到 `target_id`。
14. `flow set/run/delete` 与 `varstore set/revoke` 在写开关关闭时必须被本地拒绝。
15. 本地配置、settings 和 node keys 不得默认写入 GUI 客户端正在使用的配置目录。
16. `session_status` 必须返回足够给 AI 自检的状态摘要，至少包含 auth、defaults、config、permissions、readiness、hints。
17. tool 错误必须返回结构化结果，至少包含 `code`、`message`、`hint`，必要时附带 `details`。
18. 结构化错误至少要能明确区分 `invalid_arguments`、`not_connected`、`missing_identity`、`write_disabled`、`upstream_error`。
19. 仓内必须提供可复用的 Codex 安装脚本，避免用户每次手工编辑 `config.toml`。

## Non-functional Requirements

- 架构:
  - 优先复用 Win 现有 `session/auth/management/flow/varpool` 服务。
  - GUI 主入口与 MCP 入口边界清晰。
- 可维护性:
  - 默认身份状态、参数回退和本地配置路径必须集中实现。
  - 不要把 GUI 偏好键直接当成 MCP 运行时真相来源。
  - 启动脚本应优先复用已构建二进制，缺失时再退回开发态启动。
  - smoke 脚本应复用正式启动脚本，而不是单独维护第二套启动逻辑。
- 安全性:
  - `stdout` 只保留给 MCP JSON-RPC。
  - 写工具默认关闭。
- 兼容性:
  - 不改变现有 GUI Win 的交互或配置语义。

## Edge Cases

- 已连接时重复连接。
- 未连接时调用非 `session/auth` 工具。
- 未认证时调用 `management/varstore`。
- 缺少可用 `executor_node`，同时默认 `target_id` 也不可用。
- `flow set` 缺少 `trigger` 或 `graph`。
- `register` 返回 `pending` 或 `rejected`。
- authority 配置存在 `authority.node_id`，但当前 auth 工具错误打到了 hub target。
- authority 工具调用方不给 `authority_id`，同时也没有可用的 hub target。
- `login` 模式给了空 `config_dir`、缺失 node keys 或缺失可用 `node_id`。
- `auth_list_roles` 因 Hub 角色权限不足而失败。
- `management_config_get` 命中不存在 key 或因角色权限不足失败。
- 默认 `target_id` 缺失。
- 本地配置目录不存在或不可写。
- `flow set/run/delete` 或 varstore 写工具在 `allow_write=false` 时被调用。
- host 配置中已存在同名 MCP server，需要覆盖而不是重复追加。

## Acceptance Criteria

1. `go build ./cmd/myflowhub-mcp` 成功。
2. MCP host 能发现并调用首版工具。
3. 在真实 Hub 上可完成 `connect -> register/login -> auth_get_perms -> auth_list_roles -> auth authority ops -> management config read -> flow list/get/status/set/run/delete -> list_nodes/node_info -> varstore list/get/set/revoke` 的基础链路。
4. MCP 客户端作为独立节点身份出现在 Hub 中。
5. MCP 客户端的 settings 与 node keys 不写入 GUI Win 默认配置目录。
6. `allow_write=false` 时，写工具被本地拒绝且错误结构可读。
7. `session_status` 可直接暴露 auth/readiness/permissions/hints。
8. `scripts/install-codex-myflowhub-mcp.ps1 -WhatIf` 可成功预演安装结果。
9. `scripts/test-myflowhub-mcp-smoke.ps1 -Help` 可输出脚本用法，且 `register pending` 会显式失败并提示保留 `config_dir`。

## Related Specs

- [mcp-client.md](../specs/mcp-client.md)
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
