# Plan - MyFlowHub-Win MCP Exec And Management Tools

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/win-mcp-exec-management-tools`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-exec-management-tools`
- Current Stage: `4 archived / awaiting workflow end confirmation`

## Stage 1 - Requirements Analysis
### 目标
- 为 Win MCP client 增加一组轻量只读工具，让 AI 可以查询 exec 能力索引，并补齐 management 下的节点回显与子树查看能力。

### 范围
#### 必须
- 新增 `myflowhub_exec_cap_query`
- 新增 `myflowhub_management_node_echo`
- 新增 `myflowhub_management_list_subtree`
- `exec_cap_query` 支持 AI 友好的参数回退：
  - `source_id` 继续优先走当前 auth snapshot / defaults
  - `target_id` 作为能力查询请求的传输目标，默认回退到 hub/default target
  - `requester_node` 未传时默认回退到 `source_id`
  - `req_id` 未传时由 MCP 本地生成
- `management_node_echo` 支持 `message` 非空校验
- 更新 requirements/specs/README
- 补对应测试

#### 可选
- 在 tool 返回中补充少量 MCP 本地派生字段，只要不改变原始协议语义

#### 不做
- 本轮不接入 `exec.call`
- 本轮不接入 `management_config_set`
- 本轮不接入 `topicbus`
- 本轮不接入 `file`
- 本轮不接入 `varstore subscribe/unsubscribe`
- 本轮不新增本地更细粒度权限层

### 使用场景
- AI 查询某个 hub / 聚合节点上可用的 exec capabilities，并根据 `method` 或 `provider_node` 过滤候选路由。
- AI 用 `node_echo` 快速验证某节点 management 链路是否通畅。
- AI 读取某节点可见子树，辅助做节点发现、能力路由和 target 决策。

### 功能需求
1. MCP 必须显式暴露 `myflowhub_exec_cap_query`、`myflowhub_management_node_echo`、`myflowhub_management_list_subtree` 三个工具。
2. `exec_cap_query` 必须支持透传 `method`、`prefix`、`provider_node`、`limit`、`include_schema`。
3. `exec_cap_query` 必须支持显式传入 `requester_node`；未传时默认回退到 `source_id`。
4. `exec_cap_query` 的 `req_id` 未传时，MCP 必须本地生成请求 ID。
5. `management_node_echo` 必须支持透传 `message`，且空字符串在本地失败。
6. `management_list_subtree` 必须返回目标节点视角下的子树摘要列表。
7. 三个工具都必须延续结构化错误模型，至少区分 `invalid_arguments`、`not_connected`、`missing_identity`、`upstream_error`。
8. 三个工具都必须沿用现有 `source_id` / `target_id` 回退逻辑，不要求 AI 每次显式传身份。
9. 本轮不得把 `ExecCapQuery` 混入 `flow` 工具命名；它必须以 `exec` 为中心单独暴露。
10. 本轮新增工具均为只读，不受本地 `allow_write` gate 约束。

### 非功能需求
- 继续复用 Win 现有 `internal/services/flow` 与 `internal/services/management`，不引入 GUI 依赖
- 保持与现有 MCP tools 一致的命名、参数回退、错误结构和结果包装
- 不新增新的本地持久化状态
- 不改变现有 `flow`、`management`、`auth` 工具语义

### 输入输出
#### 输入
- `source_id?`
- `target_id?`
- `req_id?`
- `requester_node?`
- `method?`
- `prefix?`
- `provider_node?`
- `limit?`
- `include_schema?`
- `message`

#### 输出
- 各工具原始 `exec` / `management` 响应
- 必要时补充 MCP 本地生成或解析后的上下文字段

### 边界异常
- 未连接时调用新工具
- 缺少可用 `source_id` 或 `target_id`
- `requester_node`、`provider_node` 传入 0
- `limit` 传入负数
- `node_echo` 传入空 `message`
- 上游返回 `403/404/500`

### 验收标准
1. `tools/list` 中出现 3 个新的工具：`myflowhub_exec_cap_query`、`myflowhub_management_node_echo`、`myflowhub_management_list_subtree`。
2. `go test ./internal/mcp -count=1` 通过。
3. `go test ./... -count=1` 通过。
4. `go build ./cmd/myflowhub-mcp` 通过。
5. `exec_cap_query` 的 `req_id` 自动生成、`requester_node` 默认回退、参数校验均有测试覆盖。

### 风险
- `ExecCapQuery` 当前挂在 Win 的 `FlowService` 上，但协议语义属于 `exec`；tool 命名和文档若处理不当，容易继续把它误归到 `flow`。
- `include_schema=true` 时返回体可能较大，tool 层不应再做额外复制或重编码。
- `list_subtree` 与 `list_nodes` 语义接近，若返回说明不清，AI 可能误用。

### 问题清单
- none

## Stage 2 - Architecture Design
### 总体方案
- 方案采用：在现有 MCP runtime / backend / tools 三层上新增一个 `exec` 查询入口，并补 management 的两个只读工具。
- 选型理由：
  - `ExecCapQuery` 已存在于 Win `FlowService`，实现成本低
  - `NodeEcho` / `ListSubtree` 已存在于 `ManagementService`
  - 继续保持 `internal/mcp/tools.go` 作为唯一 AI 交互契约层

### 备选对比
- 方案 A：直接暴露 `myflowhub_exec_cap_query`
  - 优点：与协议语义一致，不再混入 `flow`
  - 代价：runtime 仍需通过 `FlowService` 走到底层 `exec` 子协议
- 方案 B：把 `ExecCapQuery` 继续塞进 `myflowhub_flow_*`
  - 优点：复用现有 `flow` backend
  - 代价：语义错误，会继续放大 `flow` / `exec` 边界混乱
- 结论：采用方案 A

### 模块职责
- `internal/mcpapp/runtime.go`
  - 增加 `ExecCapQuery`、`NodeEcho`、`ListSubtree` runtime wrapper
- `internal/mcp/tools.go`
  - 增加 exec 参数模型、schema、handler、错误映射
  - 增加 management node_echo / list_subtree 工具定义与 handler
- `internal/mcp/tools_test.go`
  - 为新工具补 fake backend 和回归测试
- `docs/requirements/mcp-client.md`
  - 更新稳定能力范围与验收
- `docs/specs/mcp-client.md`
  - 更新 exec / management 工具契约与 runtime 边界
- `README.md`
  - 更新可见能力说明

### 数据 / 调用流
1. AI 调用 `myflowhub_exec_cap_query` 或 `myflowhub_management_*`
2. tool 层解析参数并完成：
   - 连接态检查
   - `source_id` / `target_id` / `requester_node` 回退
   - `req_id` 自动生成
   - 只读参数校验
3. runtime 将请求发给 `FlowService.ExecCapQuery` 或 `ManagementService.NodeEcho/ListSubtree`
4. tool 层返回原始响应和结构化错误

### 接口草案
- `myflowhub_exec_cap_query`
  - 输入：`source_id? target_id? req_id? requester_node? method? prefix? provider_node? limit? include_schema?`
- `myflowhub_management_node_echo`
  - 输入：`source_id? target_id? message`
- `myflowhub_management_list_subtree`
  - 输入：`source_id? target_id?`

### 错误与安全
- 三个工具均为只读，不纳入 `allow_write`
- `exec_cap_query` 不开放 `exec.call`
- `management_node_echo` / `list_subtree` 不改变现有 management config 写边界
- `limit`、`provider_node`、`requester_node`、`message` 做最小必要本地校验
- 不在 tool 层重写上游 `code/msg` 语义，只补本地参数和会话错误

### 性能与测试策略
- `req_id` 继续本地轻量生成
- `include_schema` 结果体直接透传，不做额外格式转换
- 测试覆盖：
  - 工具注册
  - 身份回退
  - `requester_node` 默认回退
  - `req_id` 自动生成
  - 参数校验
  - backend 参数透传

### 可扩展性设计点
- 为后续 `exec.call`、`topicbus`、`file` 接入保留同样模式：runtime wrapper + tool schema + 结构化错误
- management 新增读工具后，`config_set` 仍单独留在后续高风险轮次，不混入当前回合

### 问题清单
- `MyFlowHub-Server` 当前没有单独的 `management.md` 稳定 spec，本轮以 management proto types 与现有 Win service 作为技术契约来源。

## Stage 3.1 - Planning
### Project Goal And Current State
- Goal:
  - 让 Win MCP client 具备 exec 能力查询和更完整的 management 只读观测能力。
- Current State:
  - 当前 MCP 已覆盖 `session/auth/management(list_nodes/node_info/config_get/config_list)/flow/varstore`
  - `FlowService` 已存在 `ExecCapQuery`
  - `ManagementService` 已存在 `NodeEcho` 与 `ListSubtree`
  - 当前 MCP spec 明确把 `ExecCapQuery` 排除在外

### Docs Governance
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: add
- Specs impact: add
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-exec-management-tools\docs\requirements\mcp-client.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-exec-management-tools\docs\specs\mcp-client.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
  - `C:\Users\HelloWorld\go\pkg\mod\github.com\yttydcs\myflowhub-proto@v0.1.2-0.20260318063708-7eef50dcc471\protocol\management\types.go`
- Related lessons:
  - none

### Parallelism Assessment
- `internal/mcp/tools.go`、`internal/mcpapp/runtime.go`、`internal/mcp/tools_test.go` 写集重叠，不派发子 Agent。
- Owner: 主 Agent

### Executable Checklist
- [x] DOCS-4 更新 MCP requirements/specs/README，纳入 exec 查询与 management 新工具契约
- [x] IMPL-8 为 runtime/backend 增加 exec query / management wrapper
- [x] IMPL-9 新增 exec / management MCP tools、参数解析和错误包装
- [x] TEST-4 补回归测试并完成构建验证
- [x] FIX-4 修复 `flow.Detail*` 对未发布 proto 契约的编译依赖，恢复 `GOWORK=off` 验证
- [x] REVIEW-4 完成 3.3 代码复核
- [x] ARCHIVE-4 归档到 `docs/change`

### Task Details
#### DOCS-4 - 稳定文档更新
- Goal:
  - 将 exec 查询和 management 新读工具写入 requirements/specs/README
- Files:
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
- Acceptance:
  - 稳定文档准确描述 3 个新工具、参数回退和 `config_set` / `exec.call` 不在本轮范围
- Tests:
  - 文档自检
- Rollback:
  - 回退文档新增内容

#### IMPL-8 - Runtime Wrapper
- Goal:
  - 让 MCP runtime 能调用 `ExecCapQuery`、`NodeEcho`、`ListSubtree`
- Files:
  - `internal/mcpapp/runtime.go`
  - `internal/mcp/tools.go`
- Acceptance:
  - backend interface 暴露 `ExecCapQuery`、`NodeEcho`、`ListSubtree`
  - 继续沿用统一 timeout 和 defaults/auth snapshot 回退
- Tests:
  - `go test ./internal/mcp -count=1`
- Rollback:
  - 回退新增 wrapper

#### IMPL-9 - Exec And Management Tools
- Goal:
  - 向 AI 暴露 exec 能力查询和 management 新读工具
- Files:
  - `internal/mcp/tools.go`
  - `internal/mcp/tools_test.go`
- Acceptance:
  - `tools/list` 出现 `myflowhub_exec_cap_query`、`myflowhub_management_node_echo`、`myflowhub_management_list_subtree`
  - `req_id` 支持自动生成
  - `requester_node` 默认回退到 `source_id`
  - `node_echo` / `list_subtree` 为只读
- Tests:
  - `go test ./internal/mcp -count=1`
- Rollback:
  - 回退新增 MCP tools

#### TEST-4 - 回归验证
- Goal:
  - 确保 exec / management 新工具不破坏现有 MCP 工具
- Files:
  - `internal/mcp/tools_test.go`
- Acceptance:
  - 新工具注册、参数校验、身份回退、结果透传测试通过
- Tests:
  - `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - `$env:GOWORK='off'; go test ./... -count=1`
  - `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - `git diff --check`
- Rollback:
  - 回退测试与实现

#### FIX-4 - Flow Detail Compile Unblock
- Goal:
  - 在不改变 MCP 本轮接口范围的前提下，修复 `internal/services/flow/service.go` 对未发布 `flow.Detail*` proto 契约的编译依赖，确保 `GOWORK=off` 回归可执行
- Files:
  - `internal/services/flow/service.go`
- Acceptance:
  - `go test ./... -count=1` 与 `go build ./cmd/myflowhub-mcp` 均可在当前依赖版本下通过
  - `Detail` / `DetailSimple` 继续保留既有 JSON 字段契约
- Tests:
  - `$env:GOWORK='off'; go test ./... -count=1`
  - `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
- Rollback:
  - 待 proto 正式发布后回退到共享 proto 类型

### Dependencies
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
- `C:\Users\HelloWorld\go\pkg\mod\github.com\yttydcs\myflowhub-proto@v0.1.2-0.20260318063708-7eef50dcc471\protocol\exec\types.go`
- `C:\Users\HelloWorld\go\pkg\mod\github.com\yttydcs\myflowhub-proto@v0.1.2-0.20260318063708-7eef50dcc471\protocol\management\types.go`

### Risks And Notes
- `ExecCapQuery` 挂在 `FlowService` 上只是 Win 当前封装位置，不代表它应被命名到 `flow` 工具族
- `include_schema` 打开后返回体可能变大，但本轮不引入额外截断逻辑
- `config_set` 仍是高风险写接口，本轮禁止顺手开放
- 当前发布版 `myflowhub-proto` 尚未提供 `flow.Detail*`；本轮以 `internal/services/flow` 本地类型维持既有 JSON 契约，后续应在 proto 落地后收敛回共享定义

## Stage 3.2 - Execution Record
### File-Level Change Summary
- DOCS-4
  - 对齐 `docs/requirements/mcp-client.md`、`docs/specs/mcp-client.md`、`README.md`
  - 将 `exec_cap_query`、`management_node_echo`、`management_list_subtree` 纳入稳定契约，并保留 `config_set` / `exec.call` out-of-scope
- IMPL-8
  - `internal/mcpapp/runtime.go` 新增 `ExecCapQuery`、`NodeEcho`、`ListSubtree` wrapper
  - `internal/mcp/tools.go` backend interface 同步扩展
- IMPL-9
  - `internal/mcp/tools.go` 新增 3 个 MCP tool、exec 路由解析、请求规范化、boolean schema、session hint 更新
  - `exec_cap_query` 默认回退 `requester_node -> source_id`，`req_id` 未传时本地生成
  - `management_node_echo` 本地校验非空 `message`
- TEST-4
  - `internal/mcp/tools_test.go` 扩展 fake backend
  - 新增工具注册、默认回退、过滤参数透传和参数校验测试
- FIX-4
  - `internal/services/flow/service.go` 将未发布 proto 的 `DetailReq/DetailResp` 收敛为服务内本地类型
  - 保持 `detail/detail_resp` action 与 JSON 字段契约不变，恢复当前依赖版本下的编译能力

### Validation Results
- `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - 结果：通过
- `$env:GOWORK='off'; go test ./... -count=1`
  - 结果：通过
- `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - 结果：通过
- `git diff --check`
  - 结果：通过

## Stage 3.3 - Code Review
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过

## Stage 4 - Change Archive
- 使用 `$m-docs` 完成归档与索引维护
- Archive file:
  - `docs/change/2026-03-26_win-mcp-exec-management-tools.md`
- Requirements impact: updated
- Specs impact: updated
- Lessons impact: none
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-exec-management-tools\docs\requirements\mcp-client.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-exec-management-tools\docs\specs\mcp-client.md`
- Related lessons:
  - none
- Index updates:
  - `docs/change/README.md`

阻塞：否
等待用户确认是否结束 workflow
