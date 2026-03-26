# Plan - MyFlowHub-Win MCP Flow Tools

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `main`
- Source Branch: `feat/win-mcp-flow-tools` (merged and deleted)
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-flow-tools` (removed after merge)
- Current Stage: `workflow ended / merged to main / worktree removed`

## Stage 1 - Requirements Analysis
### 目标
- 为 Win MCP client 增加 `flow` 标准工具集，让 AI 可以在无界面模式下保存、读取、列出、运行、查看状态和删除远端 DAG flow。

### 范围
#### 必须
- 新增 `myflowhub_flow_set`
- 新增 `myflowhub_flow_delete`
- 新增 `myflowhub_flow_run`
- 新增 `myflowhub_flow_status`
- 新增 `myflowhub_flow_list`
- 新增 `myflowhub_flow_get`
- 在 MCP tool 层支持 AI 友好的参数回退：
  - `source_id` 继续优先走当前 auth snapshot / defaults
  - `target_id` 继续作为 flow 请求的传输目标，默认回退到 hub/default target
  - `executor_node` 用于声明实际 flow 执行者，未传时默认回退到 `target_id`
  - `req_id` 未传时由 MCP 本地生成
- 对 `set/delete/run` 这类会改变远端状态或触发执行的动作复用现有 `allow_write` 总开关
- 更新 requirements/specs/README
- 补对应测试

#### 可选
- 在 tool 返回中补充少量 MCP 本地派生字段，只要不改变原始 flow 协议语义

#### 不做
- 本轮不接入 `ExecCapQuery`
- 本轮不接入 `topicbus`
- 本轮不接入 `file`
- 本轮不新增本地 owner/target 白名单或更细粒度本地权限层
- 本轮不改 Server `flow` / `exec` 协议

### 使用场景
- AI 将一份 DAG flow 定义直接通过 MCP 发布到目标执行节点
- AI 列出某节点已部署的 flows，并读取某个 flow 的完整定义
- AI 手动触发一次 flow 运行，并轮询最近 run 状态摘要
- AI 删除目标节点上的指定 flow

### 功能需求
1. MCP 必须显式暴露 `flow set/delete/run/status/list/get` 六个工具。
2. `flow set` 必须支持透传 `name`、`trigger`、`graph`。
3. `flow status` 必须支持可选 `run_id`，未传时查询最近一次运行。
4. `flow list` 必须返回执行节点上的 flow 摘要列表。
5. `flow get` 必须返回完整 flow 定义。
6. `flow delete` 必须支持删除指定 `flow_id`。
7. flow 工具必须允许显式传入 `executor_node`；未传时默认使用 `target_id`。
8. `req_id` 未传时，MCP 必须本地生成请求 ID，避免要求 AI 每次自行构造。
9. `flow_id`、`run_id`、`req_id` 等字符串参数必须做非空校验；明显非法参数应在本地失败。
10. `set/delete/run` 在 `allow_write=false` 时必须被本地拒绝。
11. 继续保持结构化错误模型，至少区分 `invalid_arguments`、`not_connected`、`missing_identity`、`write_disabled`、`upstream_error`。

### 非功能需求
- 继续复用 Win 现有 `internal/services/flow`，不引入 GUI 依赖
- 保持与现有 MCP tools 一致的命名、参数回退、错误结构和结果包装
- 只做最小必要本地校验，不在 MCP 侧复制整套 flow graph 语义校验
- 不引入新的长期本地状态

### 输入输出
#### 输入
- `source_id?`
- `target_id?`，作为传输目标
- `executor_node?`
- `req_id?`
- `flow_id`
- `run_id?`
- `name?`
- `trigger`
- `graph`

#### 输出
- 各工具原始 flow 响应
- 必要时补充 MCP 本地生成或解析后的上下文字段

### 边界异常
- 未连接时调用 flow 工具
- 缺少可用 `source_id` 或 `target_id`
- `flow_id` 为空
- `set` 缺少 `trigger` 或 `graph`
- `status` 传入空 `run_id`
- `allow_write=false` 时调用 `set/delete/run`
- 上游返回 `403/404/500`

### 验收标准
1. `tools/list` 中出现 6 个新的 `myflowhub_flow_*` 工具。
2. `go test ./internal/mcp -count=1` 通过。
3. `go test ./... -count=1` 通过。
4. `go build ./cmd/myflowhub-mcp` 通过。
5. `set/delete/run` 的本地写 gate、`req_id` 自动生成、身份回退和参数校验均有测试覆盖。

### 风险
- `flow` 定义中的 `graph/spec` 为自由 JSON，tool 层若做过度校验，容易与 Server 规范漂移。
- `run` 虽然不是持久化写配置，但它会触发远端执行，仍应纳入本地写 gate。
- `ExecCapQuery` 虽然在 Win `FlowService` 中可用，但它本质属于 `exec.cap.query` 语义，本轮混入会扩大边界。

### 问题清单
- none

## Stage 2 - Architecture Design
### 总体方案
- 方案采用：在现有 MCP runtime / backend / tools 三层上补 `flow` 包装与工具定义。
- 选型理由：
  - 复用 `internal/services/flow` 的已有请求/响应类型与 timeout 语义
  - 保持 `internal/mcp/tools.go` 作为唯一的 AI 交互契约层
  - 继续使用现有 `allow_write` 总开关处理远端状态变更动作

### 备选对比
- 方案 A：直接暴露 `flow set/delete/run/status/list/get`
  - 优点：与 Server `flow` 规范一一对应，范围清晰
  - 代价：AI 仍需传递较大的 `graph/spec` JSON
- 方案 B：额外暴露更高层“项目部署”抽象
  - 优点：更贴近 GUI 使用方式
  - 代价：会把前端 store 语义带进 MCP，增加不稳定抽象
- 结论：采用方案 A

### 模块职责
- `internal/mcpapp/runtime.go`
  - 增加 flow runtime wrapper
- `internal/mcp/tools.go`
  - 增加 flow 参数模型、schema、handler、错误映射、写 gate 处理
- `internal/mcp/tools_test.go`
  - 为 flow 行为补 fake backend 和回归测试
- `docs/requirements/mcp-client.md`
  - 更新稳定能力范围与验收
- `docs/specs/mcp-client.md`
  - 更新 flow 工具契约、写 gate 和 runtime 边界
- `README.md`
  - 更新可见能力说明

### 数据 / 调用流
1. AI 调用 `myflowhub_flow_*`
2. tool 层解析参数并完成：
   - 连接态检查
   - `source_id` / `target_id` / `executor_node` 回退
   - `req_id` 自动生成
   - 写 gate 判断
3. runtime 将请求发给 `internal/services/flow`
4. flow service 透传到 Hub / executor 节点
5. tool 层返回原始响应和结构化错误

### 接口草案
- `myflowhub_flow_set`
  - 输入：`source_id? target_id? executor_node? req_id? flow_id name? trigger graph`
- `myflowhub_flow_delete`
  - 输入：`source_id? target_id? executor_node? req_id? flow_id`
- `myflowhub_flow_run`
  - 输入：`source_id? target_id? executor_node? req_id? flow_id`
- `myflowhub_flow_status`
  - 输入：`source_id? target_id? executor_node? req_id? flow_id run_id?`
- `myflowhub_flow_list`
  - 输入：`source_id? target_id? executor_node? req_id?`
- `myflowhub_flow_get`
  - 输入：`source_id? target_id? executor_node? req_id? flow_id`

### 错误与安全
- `set/delete/run` 复用 `write_disabled` 错误码
- `status/list/get` 不受本地写 gate 限制
- 不在 tool 层重写上游 `code/msg` 语义，只补本地参数和会话错误
- `graph/spec` 保持透传，避免本地复制 Server 侧复杂语义校验
- `target_id` 与 `executor_node` 必须明确区分：
  - `target_id` 是传输目标
  - `executor_node` 是 flow payload 中的执行者

### 性能与测试策略
- 不新增本地持久化和额外缓存
- `req_id` 在本地轻量生成，避免要求 host 先跑额外 UUID 工具
- 测试覆盖：
  - 默认身份回退
  - `req_id` 自动生成
  - 写 gate
  - 必填参数校验
  - flow backend 参数透传

### 可扩展性设计点
- 为后续 `topicbus/file` 接入保留相同模式：runtime wrapper + tool schema + 结构化错误
- `ExecCapQuery` 后续可按 `exec` 能力查询单独接入，不污染 `flow` 工具命名

### 问题清单
- none

## Stage 3.1 - Planning
### Project Goal And Current State
- Goal:
  - 让 Win MCP client 具备无界面 flow 管理与执行能力。
- Current State:
  - 当前 MCP 只覆盖 `session/auth/management/varstore`
  - Win 后端已存在成熟 `FlowService`
  - 现有 `flow` 能力尚未通过 MCP 暴露给 AI

### Docs Governance
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: add
- Specs impact: add
- Related requirements:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\mcp-client.md`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\mcp-client.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
- Related lessons:
  - none

### Parallelism Assessment
- `internal/mcp/tools.go`、`internal/mcpapp/runtime.go`、`internal/mcp/tools_test.go` 写集高度重叠，不派发子 Agent。
- Owner: 主 Agent

### Executable Checklist
- [ ] DOCS-3 更新 MCP requirements/specs/README，纳入 flow 工具契约
- [ ] IMPL-6 为 runtime/backend 增加 flow wrapper
- [ ] IMPL-7 新增 flow MCP tools、参数解析、写 gate 和错误包装
- [ ] TEST-3 补 flow MCP 回归测试并完成构建验证
- [ ] REVIEW-3 完成 3.3 代码复核
- [ ] ARCHIVE-3 归档到 `docs/change`

### Task Details
#### DOCS-3 - 稳定文档更新
- Goal:
  - 将 flow MCP 能力写入 requirements/specs/README
- Files:
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
- Acceptance:
  - 稳定文档准确描述 6 个 flow 工具、参数回退、写 gate 和 `ExecCapQuery` 不在本轮范围
- Tests:
  - 文档自检
- Rollback:
  - 回退文档新增内容

#### IMPL-6 - Flow Runtime Wrapper
- Goal:
  - 让 MCP runtime 能调用 Win `FlowService`
- Files:
  - `internal/mcpapp/runtime.go`
  - `internal/mcp/tools.go`
- Acceptance:
  - backend interface 暴露 `Set/Delete/Run/Status/List/Get`
  - 继续沿用统一 timeout 和 defaults/auth snapshot 回退
- Tests:
  - `go test ./internal/mcp -count=1`
- Rollback:
  - 回退 flow wrapper

#### IMPL-7 - Flow Tools
- Goal:
  - 向 AI 暴露标准 flow 管理与执行工具
- Files:
  - `internal/mcp/tools.go`
  - `internal/mcp/tools_test.go`
- Acceptance:
  - `tools/list` 出现 6 个 `myflowhub_flow_*`
  - `req_id` 支持自动生成
  - `set/delete/run` 受 `allow_write` 保护
  - `list/get/status` 为只读
- Tests:
  - `go test ./internal/mcp -count=1`
- Rollback:
  - 回退新增 flow tools

#### TEST-3 - 回归验证
- Goal:
  - 确保 flow 接入不破坏现有 MCP 工具
- Files:
  - `internal/mcp/tools_test.go`
- Acceptance:
  - flow 参数校验、身份回退、写 gate、结果透传测试通过
- Tests:
  - `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - `$env:GOWORK='off'; go test ./... -count=1`
  - `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - `git diff --check`
- Rollback:
  - 回退 flow 测试与实现

### Dependencies
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
- `github.com/yttydcs/myflowhub-proto/protocol/flow`

### Risks And Notes
- `graph.nodes[].spec` 是自由 JSON，tool 层只做最低限度结构检查
- `target_id` 与 `executor_node` 若混淆，会让请求被错误发往 hub 或错误执行者
- 若后续决定开放 `ExecCapQuery`，应在下一轮以 `exec` 为中心重新规划，不在本轮扩 scope

阻塞：否
进入 3.2

## Stage 3.2 - Implementation Record
### Task Execution
- `DOCS-3`
  - 更新 `docs/requirements/mcp-client.md`，补充 `flow` 工具范围、`target_id` / `executor_node` 语义和 `ExecCapQuery` 出界说明
  - 更新 `docs/specs/mcp-client.md`，补充 6 个 `myflowhub_flow_*` 工具契约、写 gate 和 runtime 组装边界
  - 更新 `README.md`，补充 flow 工具能力说明、路由语义和写 gate 提示
- `IMPL-6`
  - `internal/mcpapp/runtime.go` 组装 `internal/services/flow`
  - 暴露 `FlowSet`、`FlowDelete`、`FlowRun`、`FlowStatus`、`FlowList`、`FlowGet`
- `IMPL-7`
  - `internal/mcp/tools.go` 新增 6 个 `myflowhub_flow_*` 工具定义、参数模型、路由解析、请求规范化、错误包装和写 gate
  - 明确 `target_id` 为 transport target，`executor_node` 为 flow payload 中的实际执行节点
  - `req_id` 未传时本地自动生成
- `TEST-3`
  - `internal/mcp/tools_test.go` 扩展 fake backend 与 flow 回归测试
  - 覆盖工具注册、身份回退、`req_id` 自动生成、显式 `executor_node`、参数校验和写 gate

### Validation Results
- `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - 结果：通过
- `$env:GOWORK='off'; go test ./... -count=1`
  - 结果：通过
- `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - 结果：通过
- `git diff --check`
  - 结果：通过

### Checklist Update
- [x] DOCS-3 更新 MCP requirements/specs/README，纳入 flow 工具契约
- [x] IMPL-6 为 runtime/backend 增加 flow wrapper
- [x] IMPL-7 新增 flow MCP tools、参数解析、写 gate 和错误包装
- [x] TEST-3 补 flow MCP 回归测试并完成构建验证
- [ ] REVIEW-3 完成 3.3 代码复核
- [ ] ARCHIVE-3 归档到 `docs/change`

## Stage 3.3 - Code Review
### Review Checklist
- 需求覆盖：通过
  - 6 个 `myflowhub_flow_*` 工具、`req_id` 自动生成、`allow_write` gate、`target_id` / `executor_node` 语义区分均已落地
- 架构合理性：通过
  - 继续沿用 runtime wrapper + MCP tool contract 的既有分层，没有把 GUI 或前端 store 语义引入 MCP
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅新增轻量参数归一化和单次 service 调用，没有引入额外持久化或重复查询
- 可读性与一致性：通过
  - 命名、错误码、schema、结果包装与现有 `auth/management/varstore` 工具保持一致
- 可扩展性与配置化：通过
  - flow 路由解析、请求规范化和 runtime wrapper 独立成块，后续接 `exec/topicbus/file` 时可复用同一模式
- 稳定性与安全：通过
  - 明确本地只保留 coarse `allow_write` gate，不新增与 Hub RBAC 冲突的本地权限层
- 测试覆盖情况：通过
  - flow 工具注册、默认回退、`executor_node` 覆盖、参数校验、写 gate 和全量回归验证均已完成
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子 Agent，任务映射与执行记录完整

### Review Conclusion
- `REVIEW-3` 通过
- 进入 Stage 4

## Stage 4 - Change Archive
### Docs Governance
- 使用 `$m-docs` 校验 requirements/specs/change 路由与索引更新义务
- Requirements impact: updated
- Specs impact: updated
- Lessons impact: none
- Related requirements:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\mcp-client.md`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\mcp-client.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
- Related lessons:
  - none

### Archive Outputs
- `docs/change/2026-03-26_win-mcp-flow-tools.md`
- `docs/change/README.md`

### Checklist Update
- [x] REVIEW-3 完成 3.3 代码复核
- [x] ARCHIVE-3 归档到 `docs/change`

### Workflow Status
- Stage 4 已完成归档
- 用户已确认结束当前 workflow

## Workflow End Record
- User confirmation:
  - `2026-03-26` confirmed ending the workflow
- Merge result:
  - source branch merged into `main` at commit `e63f09d`
- Cleanup:
  - removed worktree `D:\project\MyFlowHub3\worktrees\win-mcp-flow-tools`
  - ran `git worktree prune`
  - deleted local branch `feat/win-mcp-flow-tools`

阻塞：否
Workflow 已结束并已合并到 `main`
worktree 已移除并完成清理
