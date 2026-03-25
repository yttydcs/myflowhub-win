# MCP Client

## Background

- 当前 `MyFlowHub-Win` 只提供面向用户的 Wails GUI 客户端，没有面向 AI host 的无界面入口。
- 变量读写、节点查询、认证等协议能力已经在 Win 后端服务中存在，但 AI 若要复用这些能力，不能依赖“去操作一个已经打开的 GUI 窗口”。
- 用户已确认 AI 应以独立客户端身份接入 Hub，因此该能力需要独立节点身份、独立本地配置目录，以及与 GUI 进程隔离的生命周期。

## Goal

- 为 `MyFlowHub-Win` 提供一个可被 MCP host 以 `stdio` 方式拉起的无界面客户端。
- 该客户端以独立节点身份连接 Hub，并向 AI 暴露首版 `session`、`auth`、`management`、`varstore` 工具。
- 首版默认只开放读能力与受控写能力，不与现有 GUI 本地配置互相污染。

## Scope

### Must

- 提供无界面可执行入口，不依赖 Wails GUI。
- 支持 MCP `tools/list` / `tools/call`。
- 支持以下首版能力：
  - `session connect/disconnect/status`
  - `auth register/login`
  - `management list_nodes/node_info`
  - `varstore list/get/set/revoke`
- 作为独立节点身份接入 Hub。
- 使用独立本地配置目录保存 MCP 自己的 settings 与 node keys。
- 写工具默认受显式开关保护。
- 日志只能写 `stderr`，不得污染 MCP `stdout`。

### Optional

- 允许通过启动参数设置默认 `endpoint`、`device_id`、`display_name`、`timeout`。
- 为后续接入 `topicbus`、`config_get`、`subscribe/unsubscribe` 预留扩展空间。

### Out of Scope

- 不复用现有 GUI Win 的会话、节点身份或配置目录。
- 首版不开放 `config_set`。
- 首版不通过 UI 自动化去操作现有 Win 界面。
- 首版不实现“连接第三方 MCP Server”的通用 client。

## Scenarios

- AI host 启动 `myflowhub-mcp`，连接 Hub 并执行 `register/login`。
- AI 列出当前节点，定位 Hub 或目标节点。
- AI 读取某个变量、列出变量名、创建变量、修改变量、撤销变量。
- 用户同时运行 GUI Win 客户端和 MCP 客户端，两者互不干扰。

## Functional Requirements

1. MCP 客户端必须支持标准 `stdio` 传输。
2. 客户端必须维持长连接 session，而不是每次 tool 调用重新拨号。
3. 首版必须显式支持 `auth register/login`，因为未认证连接默认只能访问 auth 子协议。
4. auth 成功后，客户端必须维护最近一次成功的默认身份状态，至少包含：
   - `device_id`
   - `node_id`
   - `hub_id`
   - `role`
5. 业务工具必须允许显式传入 `source_id` / `target_id`；未传时可按默认身份状态回退。
6. `varstore set/revoke` 在写开关关闭时必须被本地拒绝。
7. 本地配置、settings 和 node keys 不得默认写入 GUI 客户端正在使用的配置目录。
8. 错误必须能明确区分未连接、未认证、参数非法、权限不足、目标未找到和超时。

## Non-functional Requirements

- 架构:
  - 优先复用 Win 现有 `session/auth/management/varpool` 服务。
  - GUI 主入口与 MCP 入口边界清晰。
- 可维护性:
  - 默认身份状态、参数回退和本地配置路径必须集中实现。
  - 不要把 GUI 偏好键直接当成 MCP 运行时真相来源。
- 安全性:
  - `stdout` 只保留给 MCP JSON-RPC。
  - 写工具默认关闭。
- 兼容性:
  - 不改变现有 GUI Win 的交互或配置语义。

## Edge Cases

- 已连接时重复连接。
- 未连接时调用非 `session/auth` 工具。
- 未认证时调用 `management/varstore`。
- `register` 返回 `pending` 或 `rejected`。
- 默认 `target_id` 缺失。
- 本地配置目录不存在或不可写。
- 写工具在 `allow_write=false` 时被调用。

## Acceptance Criteria

1. `go build ./cmd/myflowhub-mcp` 成功。
2. MCP host 能发现并调用首版工具。
3. 在真实 Hub 上可完成 `connect -> register/login -> list_nodes/node_info -> varstore list/get/set/revoke` 的基础链路。
4. MCP 客户端作为独立节点身份出现在 Hub 中。
5. MCP 客户端的 settings 与 node keys 不写入 GUI Win 默认配置目录。
6. `allow_write=false` 时，写工具被本地拒绝且错误可读。

## Related Specs

- [mcp-client.md](../specs/mcp-client.md)
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
