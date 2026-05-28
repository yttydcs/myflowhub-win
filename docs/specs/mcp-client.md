# MCP Client Spec

## Scope

- 本规范限定 `MyFlowHub-Win` 中无界面 MCP 客户端和本机共享 MCP Server 的模块边界、运行时约束和工具契约。
- 本规范不修改 `auth`、`management`、`exec`、`varstore` 协议本身。
- 本规范新增 `exec_cap_query` 只读工具，但不新增 `exec.call` 等写/执行能力。
- 本规范包含受 `allow_write` 保护的 authority permission snapshot 与 management config 写工具。
- 本规范不涉及 GUI 页面、Wails bindings 或第三方 MCP client 能力。
- 本规范新增本机 HTTP MCP Server 入口，用于多个 MCP client 共享一个 `myflowhub-mcp` 进程内 runtime。
- 本规范默认 Hub 角色权限模型是真实授权边界；本地仅保留显式写 gate 与安装/运行时保护。

## Interfaces / Contracts

### 1. 入口、传输与进程模型

- 新入口为 `cmd/myflowhub-mcp`。
- 该命令作为独立进程运行，并支持两种 transport：
  - `stdio`：通过 `stdin/stdout` 提供 MCP JSON-RPC，适合单会话 MCP host 直接拉起子进程。
  - `http`：通过本机 HTTP MCP endpoint 提供 JSON-RPC，适合多个 Codex 会话连接同一个常驻进程。
- `stdio` 模式下，`stdout` 只允许输出 MCP 消息；普通日志必须写 `stderr`。
- `http` 模式下，进程启动时只创建一个 `mcpapp.Runtime`，所有 HTTP request 共享该 runtime。
- `http` 模式默认监听 `127.0.0.1:17688`，默认 MCP path 为 `/mcp`。
- `http` 模式默认不得监听 `0.0.0.0`、公网 IP 或非 loopback 地址。
- `http` 模式收到 `Origin` header 时，必须校验该来源为 loopback host；不符合时返回 HTTP 403。
- `http` 模式至少支持 HTTP `POST`：
  - JSON-RPC request 返回 `Content-Type: application/json` 与单个 JSON-RPC response。
  - JSON-RPC notification 或 response 被接受时返回 HTTP `202 Accepted` 且不返回 body。
  - 非 JSON-RPC 或无法解析的 body 返回 HTTP 400 或 JSON-RPC parse error。
- `http` 模式可暂不支持 SSE；HTTP `GET` 在未实现 SSE 时返回 HTTP 405。

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
- `myflowhub_auth_get_perms`
- `myflowhub_auth_list_roles`
- `myflowhub_auth_push_perms_snapshot`
- `myflowhub_auth_list_pending_registers`
- `myflowhub_auth_approve_register`
- `myflowhub_auth_reject_register`
- `myflowhub_auth_issue_register_permit`
- `myflowhub_auth_revoke_register_permit`
- `myflowhub_management_list_nodes`
- `myflowhub_management_node_info`
- `myflowhub_management_node_echo`
- `myflowhub_management_list_subtree`
- `myflowhub_management_config_get`
- `myflowhub_management_config_set`
- `myflowhub_management_config_list`
- `myflowhub_exec_cap_query`
- `myflowhub_flow_list`
- `myflowhub_flow_get`
- `myflowhub_flow_set`
- `myflowhub_flow_run`
- `myflowhub_flow_status`
- `myflowhub_flow_delete`
- `myflowhub_topicbus_publish`
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
- `myflowhub_auth_get_perms`
  - 输入:
    - `authority_id?`
    - `node_id?`
    - `source_id?`
    - `target_id?`
  - 行为:
    - `node_id` 未传时，优先回退到当前 auth snapshot，再回退到启动默认值
    - `authority_id` 未传时，先按 management 路由拿到 hub target，再尝试读取 `authority.node_id`
  - 输出:
    - 原始 auth get_perms 结果
- `myflowhub_auth_list_roles`
  - 输入:
    - `authority_id?`
    - `offset?`
    - `limit?`
    - `role?`
    - `node_ids?`
    - `source_id?`
    - `target_id?`
  - 行为:
    - 透传 Hub `list_roles` 过滤能力
    - 本地校验 `offset/limit >= 0`、`node_ids > 0`
    - `authority_id` 未传时，先按 management 路由拿到 hub target，再尝试读取 `authority.node_id`
  - 输出:
    - 原始 auth list_roles 结果
- `myflowhub_auth_push_perms_snapshot`
  - 输入:
    - `authority_id?`
    - `snapshot`
    - `source_id?`
    - `target_id?`
  - 行为:
    - 写工具，受 `allow_write` gate 保护。
    - `snapshot` 至少必须包含 `default_role`、`default_perms`、`node_roles` 或 `role_perms` 中的一项。
    - `authority_id` 未传时，先按 management 路由拿到 hub target，再尝试读取 `authority.node_id`。
  - 输出:
    - 解析后的 `source_id`
    - 解析后的 `target_id`
    - `hub_target_id`
    - `target_resolution`
    - 原始 `snapshot`
- `myflowhub_auth_list_pending_registers`
  - 输入:
    - `authority_id?`
    - `offset?`
    - `limit?`
    - `device_id?`
    - `source_id?`
    - `target_id?`
  - 输出:
    - 原始 auth list_pending_registers 结果
- `myflowhub_auth_approve_register`
  - 输入:
    - `authority_id?`
    - `request_id`
    - `role?`
    - `source_id?`
    - `target_id?`
  - 输出:
    - 原始 auth approve_register 结果
- `myflowhub_auth_reject_register`
  - 输入:
    - `authority_id?`
    - `request_id`
    - `reason?`
    - `source_id?`
    - `target_id?`
  - 输出:
    - 原始 auth reject_register 结果
- `myflowhub_auth_issue_register_permit`
  - 输入:
    - `authority_id?`
    - `device_id`
    - `role`
    - `expires_at?`
    - `source_id?`
    - `target_id?`
  - 输出:
    - 原始 auth issue_register_permit 结果
- `myflowhub_auth_revoke_register_permit`
  - 输入:
    - `authority_id?`
    - `permit`
    - `source_id?`
    - `target_id?`
  - 输出:
    - 原始 auth revoke_register_permit 结果

### 7. Exec 工具契约

- `myflowhub_exec_cap_query`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `requester_node?`
    - `req_id?`
    - `method?`
    - `prefix?`
    - `provider_node?`
    - `limit?`
    - `include_schema?`
  - 行为:
    - `target_id` 是传输目标，用于把 exec query 路由到 hub 或目标 routing 节点
    - `requester_node` 写入 exec payload，未传时默认回退到解析后的 `source_id`
    - `req_id` 未传时，由 MCP 本地生成 UUID
    - 本地校验 `requester_node > 0`、`provider_node > 0`、`limit >= 0`
  - 输出:
    - 解析后的 `source_id`
    - 解析后的 `target_id`
    - 解析后的 `requester_node`
    - 规范化后的 `request`
    - 原始 exec cap query `response`

### 8. Flow 工具契约

- 通用路由约束:
  - `target_id` 是传输目标，用于把 flow 请求路由到 hub 或目标 transport 节点。
  - `executor_node` 是 flow payload 中的实际执行节点。
  - `executor_node` 未传时，默认回退到 `target_id`。
  - `req_id` 未传时，由 MCP 本地生成 UUID。
- `myflowhub_flow_list`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `executor_node?`
    - `req_id?`
  - 输出:
    - 解析后的 `source_id`
    - 解析后的 `target_id`
    - 解析后的 `executor_node`
    - 规范化后的 `request`
    - 原始 flow list `response`
- `myflowhub_flow_get`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `executor_node?`
    - `req_id?`
    - `flow_id`
  - 行为:
    - `flow_id` 必须为非空字符串
- `myflowhub_flow_set`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `executor_node?`
    - `req_id?`
    - `flow_id`
    - `name?`
    - `trigger`
    - `graph`
  - 行为:
    - `flow_id` 必须为非空字符串
    - `trigger.type` 必须存在
    - `graph` 必须存在
- `myflowhub_flow_run`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `executor_node?`
    - `req_id?`
    - `flow_id`
  - 行为:
    - `flow_id` 必须为非空字符串
- `myflowhub_flow_status`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `executor_node?`
    - `req_id?`
    - `flow_id`
    - `run_id?`
  - 行为:
    - `flow_id` 必须为非空字符串
    - `run_id` 仅在非空时允许透传
    - 未传 `run_id` 时，读取最近一次运行状态
- `myflowhub_flow_delete`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `executor_node?`
    - `req_id?`
    - `flow_id`
  - 行为:
    - `flow_id` 必须为非空字符串

### 9. TopicBus 工具契约

- `myflowhub_topicbus_publish`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `topic`
    - `name?`
    - `title?`
    - `body?`
    - `level?`
    - `source?`
    - `url?`
    - `payload?`
    - `meta?`
  - 行为:
    - `topic` 必须为非空字符串，按精确 topic 发布。
    - `name` 未传时使用 MCP 本地默认事件名；显式传入空字符串时本地失败。
    - `payload` 与 `meta` 只接受 JSON object。
    - `title/body/level/source/url/meta` 会合并进发送 payload；同名字段以显式通知字段为准。
    - `target_id` 是传输目标，用于把 TopicBus publish 发往 hub 或目标 routing 节点。
    - TopicBus publish 是实时 fire-and-forget 事件；MCP 成功只表示发送成功，不表示任何订阅者已展示通知。
  - 输出:
    - 解析后的 `source_id`
    - 解析后的 `target_id`
    - 规范化后的 `topic`
    - 规范化后的 `name`
    - 发送 payload 的 JSON 预览

### 10. 管理与变量工具契约

- `myflowhub_management_list_nodes`
  - 输入:
    - `source_id?`
    - `target_id?`
- `myflowhub_management_node_info`
  - 输入:
    - `source_id?`
    - `target_id?`
- `myflowhub_management_node_echo`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `message`
  - 行为:
    - `message` 必须为非空字符串
- `myflowhub_management_list_subtree`
  - 输入:
    - `source_id?`
    - `target_id?`
- `myflowhub_management_config_get`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `key`
- `myflowhub_management_config_set`
  - 输入:
    - `source_id?`
    - `target_id?`
    - `key`
    - `value`
  - 行为:
    - 写工具，受 `allow_write` gate 保护。
    - `key` 必须为非空字符串。
- `myflowhub_management_config_list`
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

### 11. 写操作 gate

- `myflowhub_auth_push_perms_snapshot` 属于写工具。
- `myflowhub_management_config_set` 属于写工具。
- `myflowhub_flow_set`、`myflowhub_flow_run`、`myflowhub_flow_delete` 属于写工具。
- `myflowhub_topicbus_publish` 属于写工具。
- `myflowhub_varstore_set` 与 `myflowhub_varstore_revoke` 属于写工具。
- `myflowhub_exec_cap_query`、`myflowhub_management_node_echo`、`myflowhub_management_list_subtree` 属于只读工具，不受本地 `allow_write` gate 约束。
- 当 runtime `allow_write=false` 时，写工具必须在本地返回明确错误。
- 该 gate 发生在协议发送前。

### 12. 启动与安装链路

- `scripts/start-myflowhub-mcp.ps1` 必须优先尝试已构建的 `myflowhub-mcp.exe`，找不到时再 fallback 到 `go run ./cmd/myflowhub-mcp`。
- 启动脚本至少应检查以下候选路径：
  - `MYFLOWHUB_MCP_EXE`
  - `build/bin/myflowhub-mcp.exe`
  - repo root 下的 `myflowhub-mcp.exe`
  - repo root 下的 `bin/myflowhub-mcp.exe`
- `scripts/install-codex-myflowhub-mcp.ps1` 必须能够以幂等方式更新 Codex `config.toml` 中对应的 `mcp_servers.<name>` 配置块。
- 安装脚本必须支持 `stdio` 和 `http` 两种 Codex 配置：
  - `stdio` 输出 `command = "powershell.exe"` 与启动脚本参数。
  - `http` 输出 `type = "http"` 与 `url = "http://127.0.0.1:<port>/mcp"`。
- `http` 安装模式只写 Codex 连接配置，不负责长期托管 daemon；用户仍需单独启动或注册系统服务。
- 安装脚本必须支持 `-WhatIf` 预演。
- `scripts/test-myflowhub-mcp-smoke.ps1` 必须复用 `scripts/start-myflowhub-mcp.ps1` 拉起 MCP 进程，并通过 stdio 逐条发送 JSON-RPC。
- smoke 脚本必须采用 staged 模型：
  - 默认基础阶段固定覆盖：
    - `initialize`
    - `tools/list`
    - `myflowhub_session_connect`
    - `myflowhub_auth_register` 或 `myflowhub_auth_login`
    - `myflowhub_auth_get_perms`
    - `myflowhub_auth_list_roles`
    - `myflowhub_management_list_nodes`
  - `-EnableExtendedRead` 额外覆盖：
    - `myflowhub_management_node_info`
    - `myflowhub_management_node_echo`
    - `myflowhub_management_list_subtree`
    - `myflowhub_management_config_list`
    - `myflowhub_management_config_get`
    - `myflowhub_exec_cap_query`
    - `myflowhub_flow_list`
    - `myflowhub_flow_get`
    - `myflowhub_flow_status`
  - `-EnableAuthoritySmoke` 额外覆盖：
    - `myflowhub_auth_list_pending_registers`
    - 可选 `myflowhub_auth_issue_register_permit` / `myflowhub_auth_revoke_register_permit`
    - 可选 `myflowhub_auth_approve_register` / `myflowhub_auth_reject_register`
  - `-EnableWriteSmoke` 额外覆盖：
    - `myflowhub_varstore_list`
    - `myflowhub_varstore_set`
    - `myflowhub_varstore_get`
    - `myflowhub_varstore_revoke`
    - `myflowhub_flow_set`
    - `myflowhub_flow_list`
    - `myflowhub_flow_get`
    - `myflowhub_flow_run`
    - `myflowhub_flow_status`
    - `myflowhub_flow_delete`
- smoke 脚本必须在进入每个已启用阶段前校验该阶段所需工具已经出现在 `tools/list` 中。
- `login` 模式必须要求显式 `config_dir`，且该目录必须已存在，避免误用 GUI 默认目录或空目录。
- `register` 模式在未显式传 `config_dir` 时，可创建独立临时目录，但必须把目录路径打印给用户。
- 当 register 返回 `code=202` 或 `status=pending` 时，脚本必须显式失败并提示“保留当前 config_dir，待审批完成后再 login”。
- authority 阶段必须始终先执行 pending list；permit 签发要求 `device_id + role` 成对输入，approve/reject 要求显式 `request_id`。
- extended read 阶段的 `config_get` 必须优先使用 `-ConfigKey`；若该 key 不可用则回退到 `config_list` 返回的首个 key；若没有任何可读 key，必须显式记录 skipped。
- write 阶段必须自动让 launcher 追加 `--allow-write`，并在本地要求 `executor_node` 与 `flow_method`。
- write 阶段必须使用临时资源名，完成后显式执行 `flow_delete` 与 `varstore_revoke`；cleanup 失败不得吞掉。
- 若 extended read 阶段未拿到现成 `flow_id` 且用户也未传 `-FlowID`，`flow_get/status` 可显式记录为 skipped，但 `flow_list` 仍必须执行。

## Data Model or Protocol

### 1. Runtime 组装边界

- MCP 入口必须复用 Win 现有服务：
  - `internal/services/session`
  - `internal/services/auth`
  - `internal/services/management`
  - `internal/services/flow`
  - `internal/services/topicbus`
  - `internal/services/varpool`
- `exec_cap_query` 在 MCP 命名上归属 `exec`，但 runtime 当前通过 `internal/services/flow` 复用已有 `ExecCapQuery` 封装，不改变底层协议边界。
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

type ServerRuntimeInfo struct {
    Transport  string
    ListenAddr string
    Path       string
    URL        string
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
- flow 路由缺失:
  - 当 flow 工具既拿不到显式 `executor_node`，也拿不到可回退的 `target_id` 时本地失败
- topicbus publish 参数非法:
  - `topic` 为空、`name` 显式传入但为空、`payload/meta` 不是 JSON object 时本地失败
- auth permission snapshot 参数非法:
  - `snapshot` 为空时本地失败
- management config 写入参数非法:
  - `key` 为空时本地失败
- 参数非法:
  - 在 tool 层优先校验，避免把明显非法请求发往 Hub
  - `flow_id`、`run_id` 等字符串参数必须在本地校验非空
  - `exec_cap_query` 的 `requester_node` / `provider_node` 必须为正整数，`limit` 必须大于等于 0
  - `management_node_echo.message` 必须为非空字符串
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
- `management_config_set` 仅在显式开启 `allow_write` 后可用，避免 AI 与用户手动配置互相覆盖。
- exec 工具必须明确区分 `target_id` 与 `requester_node`，避免把 transport 路由和请求身份语义混为一谈。
- flow 工具必须明确区分 `target_id` 与 `executor_node`，避免把 transport 路由和执行节点语义混为一谈。
- topicbus publish 的 `target_id` 仅表示 transport target，不代表通知已经投递到某个订阅节点。
- Hub 角色权限仍是真实授权边界，本地不额外实现 owner/target 白名单。
- `auth_get_perms` / `auth_list_roles` 不增加本地权限层，只复用现有 session、路由回退和结构化错误模型。
- authority 类 auth 工具的目标解析顺序为：
  - 显式 `authority_id`
  - `config_get("authority.node_id")`
  - hub target 回退

## Performance Constraints

- 单进程内复用长连接，不为每次 tool 调用重新连接。
- HTTP MCP Server 模式下，多个 client 和多个 request 共享同一个 runtime；不得按 request 重建 store、session 或 auth service。
- 避免重复读取本地配置；runtime 启动后复用同一份 services/store。
- 不增加与 GUI 无关的额外前端依赖或构建步骤。

## Related Requirements

- [mcp-client.md](../requirements/mcp-client.md)

## Related Specs

- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
