# Win MCP Flow Tools

## 变更背景 / 目标

- 当前 Win MCP client 已具备 `session`、`auth`、`management`、`varstore` 能力，但 AI 仍无法无界面管理远端 DAG flow。
- 本轮目标是在不接入 `localhub`、不新增本地细粒度权限层、不中途混入 `ExecCapQuery` 的前提下：
  - 暴露 `flow list/get/set/run/status/delete` 六个标准工具
  - 复用现有 `allow_write` 保护 `set/run/delete`
  - 明确 `target_id` 与 `executor_node` 的不同语义，避免 transport 路由和执行节点混淆

## 具体变更内容

- 更新 `internal/mcpapp/runtime.go`
  - 组装 `internal/services/flow`
  - 新增 `FlowSet`、`FlowDelete`、`FlowRun`、`FlowStatus`、`FlowList`、`FlowGet`
- 更新 `internal/mcp/tools.go`
  - 新增工具：
    - `myflowhub_flow_list`
    - `myflowhub_flow_get`
    - `myflowhub_flow_set`
    - `myflowhub_flow_run`
    - `myflowhub_flow_status`
    - `myflowhub_flow_delete`
  - 新增 flow 参数模型、schema、handler、请求规范化和路由解析
  - `req_id` 未传时本地自动生成
  - `target_id` 继续作为 transport target，`executor_node` 写入 flow payload 且未传时回退到 `target_id`
  - `set/run/delete` 复用现有 `allow_write` gate
- 更新 `internal/mcp/tools_test.go`
  - 扩展 fake backend 的 flow 方法
  - 新增 flow 工具注册、默认回退、显式 `executor_node`、参数校验和写 gate 测试
- 更新稳定文档
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`

## Requirements impact: updated

## Specs impact: updated

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-mcp-flow-tools\docs\requirements\mcp-client.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-mcp-flow-tools\docs\specs\mcp-client.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `DOCS-3`
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
- `IMPL-6`
  - `internal/mcpapp/runtime.go`
- `IMPL-7`
  - `internal/mcp/tools.go`
  - `internal/mcp/tools_test.go`
- `TEST-3`
  - `internal/mcp/tools_test.go`
  - 全量 `go test` / `go build`
- `REVIEW-3`
  - `todo.md`
- `ARCHIVE-3`
  - `docs/change/README.md`
  - `docs/change/2026-03-26_win-mcp-flow-tools.md`

## 经验 / 教训摘要

- flow 工具最容易出错的不是 schema，而是 transport target 和 executor node 的双重语义。
- `run` 虽然不是持久化配置写入，但它会触发远端执行，仍然必须纳入本地写 gate。
- `ExecCapQuery` 虽然现有 Win service 已有相关能力，但把它混进 flow 工具会扩大边界；应单独作为 `exec` 轮次处理。

## 可复用排查线索

- 症状：
  - AI 能连接和登录，但无法列出、读取或运行 flow
  - flow 请求被发到了错误节点，或返回的执行节点与预期不一致
  - `flow set/run/delete` 明明参数正确却在本地直接被拒绝
- 触发条件：
  - `target_id` 与 `executor_node` 被当成同一个概念使用
  - MCP 进程未使用 `--allow-write`
  - host 没有传 `req_id`，却错误假定上游会替它补齐
- 关键词：
  - `myflowhub_flow_list`
  - `myflowhub_flow_set`
  - `executor_node`
  - `target_id`
  - `allow_write`
  - `ExecCapQuery`
- 快速检查：
  - `tools/list` 中确认 6 个 `myflowhub_flow_*` 工具存在
  - `myflowhub_session_status` 确认 `defaults`、`readiness` 与写 gate 状态
  - 调 `myflowhub_flow_list` 时先观察返回中的 `source_id`、`target_id`、`executor_node`
  - 触发 `set/run/delete` 前确认 MCP 进程已带 `--allow-write`

## 关键设计决策与权衡

- 决策：复用已有 `allow_write` 总开关
  - 原因：Hub 角色权限模型才是真正授权边界，本轮不引入新的本地 RBAC / 白名单层。
- 决策：显式区分 `target_id` 与 `executor_node`
  - 原因：Win 前端与 service 已证明两者含义不同；若 MCP 折叠它们，会把路由和执行语义混淆。
- 决策：`req_id` 在 MCP 本地生成
  - 原因：减少 host 侧样板代码，让 AI 不必先构造请求 ID 才能调用 flow 工具。
- 决策：`ExecCapQuery` 保持出界
  - 原因：它属于 `exec` 能力发现，不应在本轮以 flow 名义扩 scope。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-flow-tools`
  - 结果：通过
- `$env:GOWORK='off'; go test ./... -count=1`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-flow-tools`
  - 结果：通过
- `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-flow-tools`
  - 结果：通过
- `git diff --check`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-flow-tools`
  - 结果：通过

## 潜在影响与回滚方案

- 潜在影响：
  - MCP host 会看到 6 个新增 flow 工具
  - 写 gate 现在同时覆盖 flow 的 `set/run/delete`
  - host 若误把 `target_id` 当成执行节点，返回结果会更明确暴露这一差异
- 回滚方案：
  - 回退 `internal/mcpapp/runtime.go`、`internal/mcp/tools.go`、`internal/mcp/tools_test.go`
  - 回退 `docs/requirements/mcp-client.md`、`docs/specs/mcp-client.md`、`README.md`
  - 删除 `docs/change/2026-03-26_win-mcp-flow-tools.md`
  - 从 `docs/change/README.md` 移除对应索引项

## 子Agent执行轨迹

- none
