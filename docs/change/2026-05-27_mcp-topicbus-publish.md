# 2026-05-27_mcp-topicbus-publish

## 变更背景 / 目标

MetricsNode NotifyNode 已经能够订阅 TopicBus topic 并在 Windows / Android 上弹出系统通知，但 `MyFlowHub-Win` 的 headless MCP 客户端尚未暴露 TopicBus publish 能力。Codex 或其他 MCP host 因此缺少一个稳定发布端，无法在任务完成时直接向 NotifyNode 订阅的 topic 发出通知事件。

本次目标是在 `myflowhub-mcp` 中新增最小发布能力：`myflowhub_topicbus_publish`。在真实云服务器接入和独立 superadmin MCP 设备配置过程中，本轮也保留两个受 `allow_write` gate 保护的操作工具：`myflowhub_management_config_set` 与 `myflowhub_auth_push_perms_snapshot`。

## 具体变更内容

- 更新 `internal/mcpapp/runtime.go`
  - 将 `TopicBusService` 接入 headless MCP runtime。
  - 在关闭 runtime 时清理 TopicBus service 订阅。
  - 新增 `TopicBusPublish(...)` timeout-wrapped wrapper。
  - 新增 `ConfigSet(...)` 与 `PushPermsSnapshot(...)` timeout-wrapped wrapper。
- 更新 `internal/mcp/tools.go`
  - 修复该文件中已存在的重复/游离代码残留，恢复 `internal/mcp` 包可编译状态。
  - 扩展 MCP `Backend` 接口，新增 `TopicBusPublish`、`ConfigSet`、`PushPermsSnapshot`。
  - 新增工具 `myflowhub_topicbus_publish`。
  - 支持 `topic/name/title/body/level/source/url/payload/meta/source_id/target_id`。
  - `topic` 必填并 trim；`name` 未传时默认 `mcp.topicbus.publish`，显式传空时报 `invalid_arguments`。
  - `payload` / `meta` 仅接受 JSON object。
  - `title/body/level/source/url/meta` 合并到 outgoing payload。
  - 发布前复用 management source/target fallback，并要求 `allow_write=true`。
  - 新增工具 `myflowhub_management_config_set`。
    - `key` 必填并 trim，空 key 报 `invalid_arguments`。
    - 复用 management source/target fallback，并要求 `allow_write=true`。
  - 新增工具 `myflowhub_auth_push_perms_snapshot`。
    - `snapshot` 至少包含 `default_role/default_perms/node_roles/role_perms` 中的一项。
    - 复用 authority route resolution：显式 `authority_id` -> `authority.node_id` -> hub target fallback。
    - 要求 `allow_write=true`。
  - 更新 `session_status` write-gate hint，使其列出本轮新增的所有写工具。
- 更新 `internal/mcp/tools_test.go`
  - 扩展 fake backend。
  - 覆盖工具注册、write gate、空 topic、payload merge、默认 name、upstream error。
  - 覆盖 `management_config_set` 的 write gate、路由、key/value 透传。
  - 覆盖 `auth_push_perms_snapshot` 的 authority 解析与 snapshot 后端调用。
- 更新稳定文档
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`

## Requirements impact

updated

`docs/requirements/mcp-client.md` 已将 `topicbus publish`、`management config_set` 与 `auth push_perms_snapshot` 纳入 MCP 稳定能力面，并记录 exact-topic、无离线重放、无投递确认、写 gate、空 key / 空 snapshot 等边界。

## Specs impact

updated

`docs/specs/mcp-client.md` 已新增 `myflowhub_topicbus_publish`、`myflowhub_management_config_set`、`myflowhub_auth_push_perms_snapshot` 工具契约、运行时服务边界、写 gate 分类和参数错误行为。

## Lessons impact

none

本轮没有新增需要长期复用的事故 lesson。前置发现的 `internal/mcp/tools.go` 重复/游离代码残留已在本归档记录。

## Related requirements

- `docs/requirements/mcp-client.md`

## Related specs

- `docs/specs/mcp-client.md`

## Related lessons

- none

## 对应 plan.md 任务映射

- `BASE-1`：清理 `internal/mcp/tools.go` 中已存在的重复/游离代码残留。
- `DOC-1`：更新 MCP requirement/spec。
- `RT-1`：将 TopicBus service 与操作写工具 wrapper 接入 MCP runtime。
- `TOOL-1`：新增 `myflowhub_topicbus_publish` 工具。
- `TOOL-2`：新增 `myflowhub_management_config_set` 与 `myflowhub_auth_push_perms_snapshot`。
- `TEST-1`：补充 MCP 工具单测。
- `VERIFY-1`：完成 targeted test / build 和 Stage 3.3 review。
- `ARCHIVE-1`：创建本变更归档。

## 经验 / 教训摘要

- MCP 作为 Codex 发布端即可满足当前通知链路，不需要改 NotifyNode 或 TopicBus 协议。
- TopicBus publish 是 fire-and-forget，MCP success 只能表示发送成功，不能表示订阅节点已弹通知。
- 事件发布、management config 写入、auth permission snapshot 推送都属于可见副作用，应纳入 `allow_write` gate。

## 可复用排查线索

- 症状：
  - `tools/list` 看不到 `myflowhub_topicbus_publish`。
  - Codex 调用 publish 成功，但目标机器没有弹通知。
  - MCP 能连接云服务器，但 authority 路由或权限快照不符合当前调试目标。
  - `go test ./internal/mcp` 报 `non-declaration statement outside function body`。
- 触发条件：
  - MCP 运行的二进制还未更新。
  - NotifyNode 未启动、未登录、未订阅同名 topic，或 topic 字符串不精确一致。
  - `authority.node_id` 未设置或运行期 permission snapshot 未同步。
  - `internal/mcp/tools.go` 中存在重复/游离 handler 残留。
- 关键词：
  - `myflowhub_topicbus_publish`
  - `myflowhub_management_config_set`
  - `myflowhub_auth_push_perms_snapshot`
  - `TopicBusPublish`
  - `ConfigSet`
  - `PushPermsSnapshot`
  - `defaultTopicBusPublishName`
  - `codex/task/done`
  - `authority.node_id`
  - `notify.topics_json`
  - `non-declaration statement outside function body`
- 快速检查：
  - 调 `tools/list` 确认 `myflowhub_topicbus_publish` 已注册。
  - 调 `myflowhub_session_status` 确认已连接、已登录并开启 `allow_write`。
  - 如需配置 authority 路由，先用 `myflowhub_management_config_set` 写入并用 `myflowhub_management_config_get` 复核。
  - 如需同步权限快照，确认 snapshot 非空且当前角色可向 authority 写入。
  - 确认 MetricsNode NotifyNode 订阅 topic 与 publish topic 完全一致。
  - 确认调用参数中 `payload` / `meta` 是 JSON object。

## 关键设计决策与权衡

- 只新增 publish，不新增 subscribe / unsubscribe / list_subs：
  - 原因：当前 Codex 通知用例只需要发布端，订阅端已经由 MetricsNode NotifyNode 提供。
- 复用 `TopicBusService.Publish`：
  - 原因：避免重复 TopicBus envelope 编码和发送逻辑。
- 使用 write gate：
  - 原因：publish 会向 Hub 发出实时事件，config set / permission snapshot 会改变运行期配置或权限，均属于有外部可见影响的写操作。
- 不保证通知投递：
  - 原因：当前 TopicBus 没有 ack / replay 语义，保持协议真实边界。
- 不把云服务器 IP、节点 ID 或角色写死进工具：
  - 原因：MCP 工具应保持通用能力，实际环境参数通过调用参数或 MCP 本地配置传入。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - 结果：通过
- `$env:GOWORK='off'; go test ./internal/mcp ./internal/mcpapp ./internal/services/topicbus -count=1`
  - 结果：通过
- `$env:GOWORK='off'; go build -o .\build\bin\myflowhub-mcp.exe .\cmd\myflowhub-mcp`
  - 结果：通过

## 潜在影响

- `myflowhub-mcp` 的工具列表新增三个写工具；未开启 `--allow-write` 时会返回 `write_disabled`。
- `internal/mcp/tools.go` 清理了已存在的重复/游离代码残留，恢复测试编译。
- `myflowhub_management_config_set` 与 `myflowhub_auth_push_perms_snapshot` 可改变远端运行期配置 / 权限快照，必须只在明确 setup 或修复场景中使用。
- 真实端到端弹窗仍依赖 MetricsNode 在线、登录、Notify 已启动、topic 精确匹配和平台通知权限。

## 回滚方案

- 回退 `internal/mcpapp/runtime.go` 中 TopicBus service wiring、`TopicBusPublish`、`ConfigSet`、`PushPermsSnapshot` wrapper。
- 回退 `internal/mcp/tools.go` 中 `myflowhub_topicbus_publish`、`myflowhub_management_config_set`、`myflowhub_auth_push_perms_snapshot` tool、Backend 方法、payload helper、snapshot helper 和 status hint。
- 回退 `internal/mcp/tools_test.go` 中 fake backend 与新测试。
- 回退 `docs/requirements/mcp-client.md` 与 `docs/specs/mcp-client.md` 中 TopicBus publish / operational write tool 契约。
- 删除本 change 归档并移除索引项。

## 子Agent执行轨迹

- 未派发子Agent。
