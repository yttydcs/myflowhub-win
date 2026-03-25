# Plan - MyFlowHub-Win MCP AI 客户端

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/mcp-ai-client`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
- Current Stage: `4 completed / waiting workflow end confirmation`

## Stage Records

### Initialization
- `guide.md`: workspace 根 `guide.md` 已读取，已确认 worktree 必须位于 `D:\project\MyFlowHub3\worktrees\`
- base/worktree confirmation:
  - control worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-ai-client`
  - repo worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
  - implementation 只在当前 Win repo worktree 进行

### Stage 1 - Requirements Analysis
#### Goal
- 在 Win 仓内新增一个无界面 MCP 客户端，让 AI host 能直接通过 MyFlowHub 协议访问变量与节点能力。

#### Scope
- 必须:
  - `stdio` MCP server
  - `session` / `auth` / `management` / `varstore` 首版工具
  - 独立本地配置目录与独立 node keys
  - 独立节点身份
  - 写操作 gate
- 可选:
  - 后续扩展 `topicbus` / `config_get`
- 不做:
  - GUI 复用
  - `config_set`
  - 外部第三方 MCP client

#### Use Cases
- AI 通过工具完成连接、注册/登录、查看节点、读取变量、修改变量、撤销变量。
- 用户保留 GUI Win 会话，AI 使用独立后台会话。

#### Functional Requirements
- `connect` 不等于完成 auth；首版必须显式支持 `register/login`。
- auth 成功后必须维护默认身份快照。
- `source_id` / `target_id` 允许显式传入，未传时按默认状态回退。
- 日志不得污染 MCP `stdout`。

#### Non-functional Requirements
- 复用现有 Win 服务，最小改动。
- 独立本地配置目录，避免与 GUI 冲突。
- 写工具默认关闭。

#### Inputs / Outputs
- 输入: 启动参数 + MCP JSON 参数
- 输出: MCP tool 结果 + `stderr` 日志 + 独立 settings / keys

#### Edge Cases
- 未连接调用业务工具
- 未认证调用 `management/varstore`
- `register` 返回 `pending`
- `allow_write=false` 调用写工具
- 缺少默认 `target_id`

#### Acceptance Criteria
- `go build ./cmd/myflowhub-mcp` 成功
- 可完成 `connect -> register/login -> list_nodes/node_info -> varstore list/get/set/revoke`
- 独立配置目录与独立节点身份生效

#### Risks
- store base dir 当前不可配置，需要安全地扩展
- tool 默认身份回退如果不集中，会变得不可维护

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 新增 `cmd/myflowhub-mcp`
- 新增 headless runtime 组装层
- 复用 Win `session/auth/management/varpool` 服务
- 用独立 `config_dir` + `mcp.*` namespace 保存本地状态

#### Alternatives Considered
- SDK 先做 typed client: 范围过大，暂不采用
- GUI 进程内嵌 MCP: 会话与配置耦合，暂不采用

#### Module Responsibilities
- `cmd/myflowhub-mcp`: CLI 参数、stdio 生命周期
- `internal/mcp`: JSON-RPC / tools / state / adapters
- `internal/storage`: base dir override
- `internal/services/*`: 协议调用实现

#### Data / Call Flow
1. host 启动 `myflowhub-mcp`
2. runtime 初始化 store/session/auth/management/varpool
3. `session_connect`
4. `auth_register/login`
5. 记录 auth snapshot
6. `management_*` / `varstore_*`

#### Interface Drafts
- Tools:
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

#### Error Handling and Safety
- `stdout` only for MCP
- `stderr` only for logs
- write tools require `allow_write=true`
- 参数缺失、本地默认值缺失时本地立即失败

#### Performance and Testing Strategy
- 单进程长连接
- 单测覆盖参数校验、状态回退、write gate、config dir
- 构建验证 `go build ./cmd/myflowhub-mcp`

#### Extensibility Design Points
- 未来可扩展 `topicbus` / `config_get`
- 未来可把 typed client 下沉到 SDK
- 先保证 tool 层与 runtime 层边界清晰

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 在当前 Win repo worktree 内完成无界面 MCP 客户端首版
- Current State:
  - 当前 repo 只有 Wails GUI 主入口，没有 headless CLI
  - 现有 `session/auth/management/varpool` 服务可直接复用
  - 当前 store 默认写 GUI 配置目录，需新增 override

#### Docs Governance Routing Decision
- 使用 `$m-docs` 结论:
  - `requirements`: add
  - `specs`: add
  - `change`: implementation 完成后再归档
- Stable docs:
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`

#### Related Requirements / Specs / Lessons
- Requirements:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\requirements\mcp-client.md`
- Specs:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\specs\mcp-client.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
  - `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
- Lessons:
  - none

#### Executable Task List
- [x] DOCS-1 补稳定 requirements/specs 和索引
- [x] IMPL-1 增加 headless runtime 与 store base dir override
- [x] IMPL-2 增加 MCP `stdio` 运行时与工具注册
- [x] IMPL-3 接入 auth/session 状态与 management/varstore adapters
- [x] IMPL-4 单测、构建与 README 说明

#### Task Details
##### DOCS-1 - 稳定文档与索引
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\todo.md`
- Goal: 建立 MCP 客户端稳定真相入口
- Files / Modules:
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `docs/requirements/README.md`
  - `docs/specs/README.md`
- Write Set:
  - 同上
- Acceptance:
  - 文档与索引可导航
- Test Points:
  - 文档自检
- Rollback:
  - 删除新增文档并恢复索引

##### IMPL-1 - Headless Runtime 与独立配置目录
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\todo.md`
- Goal: 不改 GUI 主入口前提下，新增无界面 runtime 组装能力
- Files / Modules:
  - `internal/storage/*`
  - `internal/mcpapp/*` 或等价 headless runtime 模块
  - `cmd/myflowhub-mcp/*`
- Write Set:
  - 新增 runtime / CLI 文件
  - 修改 store 构造入口
- Acceptance:
  - runtime 可独立构建
  - `config_dir` 生效
- Test Points:
  - base dir / keys 路径单测
- Rollback:
  - 回退新增 runtime / storage 入口

##### IMPL-2 - MCP 运行时与工具注册
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\todo.md`
- Goal: 打通 `stdio` MCP 基础链路
- Files / Modules:
  - `cmd/myflowhub-mcp/*`
  - `internal/mcp/*`
- Write Set:
  - 新增 MCP runtime / JSON-RPC / tools registry
- Acceptance:
  - `tools/list` / `tools/call` 正常
- Test Points:
  - MCP 单测
- Rollback:
  - 删除新增 MCP runtime

##### IMPL-3 - Auth / Management / VarStore 适配
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\todo.md`
- Goal: 复用现有服务实现首版工具并维护默认身份状态
- Files / Modules:
  - `internal/mcp/*`
  - 必要的 `internal/services/*`
- Write Set:
  - auth / management / varpool tool adapters
  - session/auth snapshot 状态模块
- Acceptance:
  - register/login 后默认 `node_id/hub_id` 可回退
  - varstore 写工具受 gate 保护
- Test Points:
  - snapshot / fallback / gate 单测
- Rollback:
  - 回退 adapters 与状态模块

##### IMPL-4 - 验证与说明
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\todo.md`
- Goal: 验证首版可构建、可测试、可使用
- Files / Modules:
  - `README.md`
  - `*_test.go`
- Write Set:
  - README 使用说明
  - 测试文件
- Acceptance:
  - `go test ./... -count=1`
  - `go build ./cmd/myflowhub-mcp`
- Test Points:
  - 见 Acceptance
- Rollback:
  - 回退 README 与测试补充

#### Dependencies
- `github.com/yttydcs/myflowhub-sdk`
- `MyFlowHub-Server` 真实 auth / varstore / management 协议行为

#### Risks and Notes
- 当前 repo 没有通用 headless 组装层，需要避免把 Wails 依赖带进新命令。
- MCP 本地状态与 GUI `home.*` / `app.*` 键应隔离，避免未来迁移成本。
- 如果实现中发现必须修改 SDK 或 Server 契约，应返回 `3.1` 更新 plan。

#### Parallelism Assessment
- 不使用子 Agent。
- 理由:
  - `cmd`、runtime、storage、tool adapters 写集重叠明显
  - auth 状态与默认回退是主风险点，拆分后反而增加集成成本

#### Issue List
- none

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Task Completion
- `IMPL-1`
  - `internal/storage/store.go` 新增 `NewStoreWithBaseDir()`，保留 GUI `NewStore()` 默认路径行为。
  - `internal/storage/store_test.go` 覆盖 override base dir 与默认 fallback。
  - 新增 `internal/mcpapp/runtime.go`，组装 headless runtime、独立 `mcp.*` 配置、`stderr` 日志桥接与 auth snapshot 持久化。
- `IMPL-2`
  - 新增 `internal/mcp/server.go`，实现 line-delimited MCP `stdio` server，支持 `initialize`、`tools/list`、`tools/call`、`ping`、`shutdown`、`exit`。
  - 新增 `cmd/myflowhub-mcp/main.go`，提供独立 CLI 入口和参数解析。
- `IMPL-3`
  - 新增 `internal/mcp/tools.go`，接入 `session/auth/management/varstore` 工具，集中处理参数校验、route fallback、write gate。
  - `internal/services/auth/service.go` / `internal/services/management/service.go` 优先读取 `mcp.display_name`。
- `IMPL-4`
  - 新增 `internal/mcp/server_test.go` / `internal/mcp/tools_test.go`。
  - 扩展 `internal/services/auth/service_test.go` / `internal/services/management/service_test.go`。
  - `README.md` 增加 MCP CLI build / run / host config 说明。

#### Validation
- `$env:GOWORK='off'; go test ./... -count=1`
  - 结果：通过
- `$env:GOWORK='off'; go build -o (Join-Path $env:TEMP 'myflowhub-mcp.exe') ./cmd/myflowhub-mcp`
  - 结果：通过
- 进程级 smoke：
  - `initialize` + `tools/list` 直连 `myflowhub-mcp` 返回正常 JSON-RPC 响应

### Stage 3.3 - Code Review
#### Review Checklist
- 需求覆盖：通过
  - 首版工具、独立配置目录、独立节点身份、写 gate、`stderr/stdout` 分流均已实现。
- 架构合理性：通过
  - `cmd/myflowhub-mcp`、`internal/mcpapp`、`internal/mcp` 分层清晰，GUI 入口未被耦合进 headless runtime。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - runtime 复用单实例 store / services / session；tool 层不重复构造连接；日志桥只订阅一次总线。
- 可读性与一致性：通过
  - route fallback、write gate、tool schema 与结果包装集中在 `internal/mcp/tools.go`。
- 可扩展性与配置化：通过
  - 启动参数和 `mcp.*` namespace 已集中，后续可继续扩展其他 tool。
- 稳定性与安全：通过
  - `stdout` 仅输出 JSON-RPC；写操作默认关闭；连接/参数缺失优先本地失败。
- 测试覆盖情况：通过
  - 覆盖了 store base dir、display name 优先级、MCP initialize/list、tool fallback、write gate。
- 子Agent治理与审计：通过
  - 未使用子 Agent；原因与写集冲突已在计划中记录。

### Stage 4 - Change Archive
#### `$m-docs` Impact Check
- Requirements impact: updated
- Specs impact: updated
- Lessons impact: none
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\requirements\mcp-client.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\specs\mcp-client.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
  - `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
- Related lessons:
  - none

#### Archive Output
- `docs/change/2026-03-25_win-mcp-ai-client.md`

阻塞：否
Stage 4 已完成
等待用户确认是否结束 workflow

## Round 2 - MCP 启动脚本

### Stage 1 - Requirements Analysis
#### Goal
- 在仓内新增一个 `scripts/` 下可直接启动 `myflowhub-mcp` 的脚本，便于 MCP host 或本地用户复用。

#### Scope
- 必须:
  - 新增独立脚本文件
  - 复用现有 `cmd/myflowhub-mcp`
  - 支持把额外参数继续透传给 MCP CLI
- 不做:
  - 改动 MCP 协议或工具集合
  - 改动 GUI 启动流程

#### Acceptance Criteria
- 脚本可从仓根启动 `cmd/myflowhub-mcp`
- 可透传 `--version` 与常见启动参数

### Stage 2 - Architecture Design
#### Overall Solution
- 新增 `scripts/start-myflowhub-mcp.ps1`
- 脚本负责：
  - 定位 repo root
  - 设置 `GOWORK=off`
  - 调用 `go run ./cmd/myflowhub-mcp`
  - 透传显式参数和剩余参数

### Stage 3.1 - Planning
#### `$m-docs` Routing
- Requirements impact: none
- Specs impact: none
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\requirements\mcp-client.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\specs\mcp-client.md`
- Related lessons:
  - none

#### Executable Task List
- [x] SCRIPT-1 新增 `scripts/start-myflowhub-mcp.ps1`
- [x] SCRIPT-2 更新 `README.md`
- [x] SCRIPT-3 验证与 `docs/change` 归档

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
- `SCRIPT-1`
  - 新增 `scripts/start-myflowhub-mcp.ps1`
  - 固定在 repo root 下执行 `go run ./cmd/myflowhub-mcp`
  - 透传剩余 CLI 参数并临时设置 `GOWORK=off`
- `SCRIPT-2`
  - `README.md` 新增脚本启动方式与 MCP host 配置示例

### Stage 3.3 - Code Review
- 需求覆盖：通过
- 架构合理性：通过
- 可读性与一致性：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
  - 脚本级 smoke 覆盖 `--version` 和 `initialize/tools/list`
- 子Agent治理与审计：通过
  - none

### Stage 4 - Change Archive
- Requirements impact: none
- Specs impact: none
- Lessons impact: none
- Archive Output:
  - `docs/change/2026-03-25_win-mcp-start-script.md`

阻塞：否
Round 2 Stage 4 已完成
可执行 workflow 结束收口
