# MCP Client Spec

## Scope

- 本规范限定 `MyFlowHub-Win` 中无界面 MCP 客户端首版的模块边界、运行时约束和工具契约。
- 本规范不修改 `auth`、`management`、`varstore` 协议本身。
- 本规范不涉及 GUI 页面、Wails bindings 或第三方 MCP client 能力。
- 本规范默认 Hub 角色权限模型是真实授权边界；本地仅保留显式写 gate 与安装/运行时保护。

## Interfaces / Contracts

### 1. 入口与进程模型

- 新入口为 `cmd/myflowhub-mcp`。
- 该命令作为独立进程运行，通过 `stdin/stdout` 提供 MCP JSON-RPC。
- `stdout` 只允许输出 MCP 消息；普通日志必须写 `stderr`。

### 2. 本地配置隔离

- MCP 客户端必须支持显式 `config_dir`。
- MCP 默认不得复用 GUI 默认配置目录。
- MCP 本地状态应使用独立 key namespace，至少包含：
  - `mcp.endpoint`
  - `mcp.device_id`
  - `mcp.display_name`
  - `mcp.node_id`
  - `mcp.hub_id`
  - `mcp.role`
  - `mcp.default_target`
  - `mcp.allow_write`

### 3. 默认身份状态

- runtime 维护一个进程内 auth snapshot，至少包含：
  - `device_id`
  - `node_id`
  - `hub_id`
  - `role`
  - `logged_in`
- `auth register/login` 成功后必须刷新该 snapshot。
- 业务工具的 `source_id` / `target_id` 解析顺序：
  1. tool 显式参数
  2. 当前 auth snapshot
  3. 启动默认值
- 若最终无法解析出必需的 `source_id` 或 `target_id`，必须本地失败。

### 4. 首版工具集合

首版固定提供：

- `myflowhub_session_status`
- `myflowhub_session_connect`
- `myflowhub_session_disconnect`
- `myflowhub_auth_register`
- `myflowhub_auth_login`
- `myflowhub_management_list_nodes`
- `myflowhub_management_node_info`
- `myflowhub_varstore_list`
- `myflowhub_varstore_get`
- `myflowhub_varstore_set`
- `myflowhub_varstore_revoke`

### 5. 会话工具契约

- `myflowhub_session_connect`
  - 输入:
    - `endpoint?`
  - 行为:
    - 建立到 Hub 的长连接
    - 已连接时允许幂等返回当前状态
- `myflowhub_session_disconnect`
  - 行为:
    - 关闭当前连接
    - 清理连接态
- `myflowhub_session_status`
  - 输出:
    - `connected`
    - `endpoint`
    - `auth`
    - `defaults`
    - `config`
    - `permissions`
    - `readiness`
    - `hints`

### 6. 认证工具契约

- `myflowhub_auth_register`
  - 输入:
    - `device_id?`
    - `source_id?`
    - `target_id?`
  - 输出:
    - 原始 auth register 结果
    - 若 `code=1`，同步刷新 auth snapshot
- `myflowhub_auth_login`
  - 输入:
    - `device_id?`
    - `node_id?`
    - `source_id?`
    - `target_id?`
  - 输出:
    - 原始 auth login 结果
    - 若 `code=1`，同步刷新 auth snapshot

### 7. 管理与变量工具契约

- `myflowhub_management_list_nodes`
  - 输入:
    - `source_id?`
    - `target_id?`
- `myflowhub_management_node_info`
  - 输入:
    - `source_id?`
    - `target_id?`
- `myflowhub_varstore_list`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `owner?`
- `myflowhub_varstore_get`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `name`
    - `owner?`
- `myflowhub_varstore_set`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `name`
    - `value`
    - `owner?`
    - `visibility?`
    - `type?`
- `myflowhub_varstore_revoke`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `name`
    - `owner?`

### 8. 写操作 gate

- `myflowhub_varstore_set` 与 `myflowhub_varstore_revoke` 属于写工具。
- 当 runtime `allow_write=false` 时，写工具必须在本地返回明确错误。
- 该 gate 发生在协议发送前。

### 9. 启动与安装链路

- `scripts/start-myflowhub-mcp.ps1` 必须优先尝试已构建的 `myflowhub-mcp.exe`，找不到时再 fallback 到 `go run ./cmd/myflowhub-mcp`。
- 启动脚本至少应检查以下候选路径：
  - `MYFLOWHUB_MCP_EXE`
  - `build/bin/myflowhub-mcp.exe`
  - repo root 下的 `myflowhub-mcp.exe`
  - repo root 下的 `bin/myflowhub-mcp.exe`
- `scripts/install-codex-myflowhub-mcp.ps1` 必须能够以幂等方式更新 Codex `config.toml` 中对应的 `mcp_servers.<name>` 配置块。
- 安装脚本必须支持 `-WhatIf` 预演。

## Data Model or Protocol

### 1. Runtime 组装边界

- MCP 入口必须复用 Win 现有服务：
  - `internal/services/session`
  - `internal/services/auth`
  - `internal/services/management`
  - `internal/services/varpool`
- MCP 入口不得依赖 Wails `runtime.EventsEmit` 或 GUI 页面状态。

### 2. 状态模型

建议 runtime 统一维护：

```go
type AuthSnapshot struct {
    DeviceID string
    NodeID   uint32
    HubID    uint32
    Role     string
    LoggedIn bool
}

type SessionSnapshot struct {
    Connected bool
    Endpoint  string
    Auth      AuthSnapshot
}
```

`session_status` 额外返回的运行摘要建议满足：

```go
type StatusPermissions struct {
    AuthorizationModel string
    LocalWriteGate     bool
}

type StatusReadiness struct {
    Authenticated bool
    HasIdentity   bool
    HasTarget     bool
    CanRegister   bool
    CanLogin      bool
    CanManage     bool
    CanVarRead    bool
    CanVarWrite   bool
}
```

### 3. Store 构造能力

- `internal/storage` 必须提供可显式指定 base dir 的构造入口。
- node keys 路径也必须随该 base dir 变化。
- 该能力既服务当前 MCP 客户端，也不得破坏 GUI 现有默认路径行为。

## Error Handling

- 未连接:
  - 非 `session/auth` 工具调用时本地失败
- 缺省身份:
  - 当业务工具无法得到可用默认 `source_id` / `hub_id` 时本地失败
- 参数非法:
  - 在 tool 层优先校验，避免把明显非法请求发往 Hub
- 写操作被禁:
  - 使用独立错误码或明确消息区分于普通协议失败
- 协议失败:
  - 保留原有服务返回的 `code/msg`

MCP tool 结构化错误至少包含：

```json
{
  "code": "invalid_arguments | not_connected | missing_identity | write_disabled | upstream_error",
  "message": "human readable summary",
  "hint": "next action for the AI or user",
  "details": {}
}
```

## Security / Safety

- `stdout` 只承载协议，`stderr` 只承载日志。
- MCP 本地配置目录默认独立于 GUI。
- 写操作默认关闭。
- 首版不开放 `config_set`，避免 AI 与用户手动配置互相覆盖。
- Hub 角色权限仍是真实授权边界，本地不额外实现 owner/target 白名单。

## Performance Constraints

- 单进程内复用长连接，不为每次 tool 调用重新连接。
- 避免重复读取本地配置；runtime 启动后复用同一份 services/store。
- 不增加与 GUI 无关的额外前端依赖或构建步骤。

## Related Requirements

- [mcp-client.md](../requirements/mcp-client.md)

## Related Specs

- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
