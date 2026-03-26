# Win MCP Exec And Management Tools

## 变更背景 / 目标

- Win MCP client 之前已覆盖 `session/auth/management config/flow/varstore`，但仍缺少：
  - exec 能力发现入口
  - management 节点回显
  - management 子树查看
- 本轮目标是把这 3 个只读能力接入 MCP，并保持既有的身份回退、结构化错误和写 gate 边界。

## 具体变更内容

- 更新 `internal/mcpapp/runtime.go`
  - 新增 `ExecCapQuery`
  - 新增 `NodeEcho`
  - 新增 `ListSubtree`
- 更新 `internal/mcp/tools.go`
  - backend interface 新增 `ExecCapQuery`、`NodeEcho`、`ListSubtree`
  - 新增 `myflowhub_exec_cap_query`
  - 新增 `myflowhub_management_node_echo`
  - 新增 `myflowhub_management_list_subtree`
  - 新增 exec 请求规范化、`requester_node` 回退、boolean schema 和相关错误包装
  - `session_status` hint 补充 `exec`
- 更新 `internal/mcp/tools_test.go`
  - fake backend 新增 exec / management read 方法
  - 增加工具注册、`req_id` 自动生成、`requester_node` 回退、过滤参数透传、`message` 非空校验、subtree 路由回退测试
- 更新稳定文档
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
  - 将 `exec_cap_query`、`management_node_echo`、`management_list_subtree` 纳入稳定 MCP 能力面
- 修复验证阻塞
  - `internal/services/flow/service.go` 之前依赖未发布的 `flow.Detail*` proto 类型，导致 `GOWORK=off` 下 `go test ./...` 失败
  - 本轮改为服务内本地 `DetailReq/DetailResp`，保持既有 JSON 字段与 `detail/detail_resp` action，不改变本轮 MCP 契约

## Requirements impact: updated

## Specs impact: updated

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-mcp-exec-management-tools\docs\requirements\mcp-client.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-mcp-exec-management-tools\docs\specs\mcp-client.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `DOCS-4`
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
- `IMPL-8`
  - `internal/mcpapp/runtime.go`
  - `internal/mcp/tools.go`
- `IMPL-9`
  - `internal/mcp/tools.go`
  - `internal/mcp/tools_test.go`
- `TEST-4`
  - `internal/mcp/tools_test.go`
- `FIX-4`
  - `internal/services/flow/service.go`

## 经验 / 教训摘要

- `ExecCapQuery` 虽然当前复用 `FlowService` 封装，但 MCP 工具命名必须保持 `exec` 语义，避免继续把 routing discovery 和 flow authoring 混在一起。
- management 新增读工具可以直接复用现有 `source_id/target_id` 回退，不需要引入新的本地状态。
- 当发布版 proto 落后于仓内服务实现时，验证阶段要先识别是“本轮改动问题”还是“基线依赖断裂”，否则容易在错误方向上排查。

## 可复用排查线索

- 症状：
  - `tools/list` 看不到 `myflowhub_exec_cap_query`
  - `management_node_echo` 传空字符串直接失败
  - `go test ./...` 在 `internal/services/flow/service.go` 报 `undefined: flow.DetailReq`
- 触发条件：
  - `internal/mcp/tools.go` 未注册 exec / management 新工具
  - `requester_node` / `provider_node` 传入 0
  - 发布版 `myflowhub-proto` 尚未包含 `flow.Detail*`
- 关键词：
  - `myflowhub_exec_cap_query`
  - `myflowhub_management_node_echo`
  - `myflowhub_management_list_subtree`
  - `resolveExecQueryRoute`
  - `normalizeExecCapQueryReq`
  - `DetailReq`
- 快速检查：
  - 检查 `internal/mcp/tools.go` 是否注册 3 个新工具
  - 检查 `internal/mcp/tools_test.go` 是否覆盖 `req_id` 自动生成与 `message` 非空校验
  - 检查 `internal/services/flow/service.go` 是否仍引用 `flow.DetailReq/flow.DetailResp`

## 关键设计决策与权衡

- 决策：`exec_cap_query` 独立暴露为 `exec` 工具，而不是挂在 `flow` 下。
  - 原因：能力发现和 flow 运行控制是不同协议语义，继续混名会放大 AI 的路由误判。
- 决策：`requester_node` 未传时回退到解析后的 `source_id`。
  - 原因：与现有 MCP 身份回退一致，减少 AI 每次显式填写身份的负担。
- 决策：对未发布 `flow.Detail*` proto 采用服务内本地类型兜底。
  - 原因：这能在不扩张本轮 MCP scope 的前提下恢复 `GOWORK=off` 验证链路；待 proto 正式落地后再回收为共享类型。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - 结果：通过
- `$env:GOWORK='off'; go test ./... -count=1`
  - 结果：通过
- `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - 结果：通过
- `git diff --check`
  - 结果：通过

## 潜在影响与回滚方案

- 潜在影响：
  - `include_schema=true` 时 exec 查询返回体可能较大，本轮仍然直接透传
  - `internal/services/flow` 的本地 `DetailReq/DetailResp` 只是编译兜底，后续 proto 正式发布后应收敛回共享定义
- 回滚方案：
  - 回退 `internal/mcpapp/runtime.go`、`internal/mcp/tools.go`、`internal/mcp/tools_test.go` 的新增 exec / management read 工具
  - 回退 `docs/requirements/mcp-client.md`、`docs/specs/mcp-client.md`、`README.md` 的对应稳定文档更新
  - 当 proto 发布后，再回退 `internal/services/flow/service.go` 的本地 detail 类型

## 子Agent执行轨迹

- none
