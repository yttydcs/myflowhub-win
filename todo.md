# Workflow Todo - Win MCP Full Chain Smoke

## Workflow Information
- Repo: `MyFlowHub-Win`
- Branch: `feat/win-mcp-full-chain-smoke`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
- Current Stage: `4`

## Stage Records

### Initialization
- guide.md:
  - 已读取 workspace `guide.md`，确认 worktree 必须位于 `D:\project\MyFlowHub3\worktrees\`，实现不得在主 repo 目录直接进行。
- base/worktree confirmation:
  - 执行 repo: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
  - 当前分支: `feat/win-mcp-full-chain-smoke`
  - 当前 repo 基线: `repo/MyFlowHub-Win` `main`

### Stage 1 - Requirements Analysis
#### Goal
- 在不扩张 MCP 协议能力面的前提下，把现有 `myflowhub-mcp` 工具串成更完整的真实 Hub smoke 验证链路。
- 补齐 authority / management / exec / flow / varstore 的真实环境验收入口，同时保持默认行为安全、可回滚、可重复执行。

#### Scope
- 必须:
  - 扩展 `scripts/test-myflowhub-mcp-smoke.ps1`，使其能覆盖现有已实现工具的更完整验证链路。
  - 默认保持非破坏性或最小破坏性行为，写操作和 authority 操作必须显式 opt-in。
  - 为需要真实写入的阶段提供明确参数、命名策略、清理策略和失败提示。
  - 对齐 `docs/requirements/mcp-client.md`、`docs/specs/mcp-client.md`、`README.md` 的 smoke 契约描述。
  - 保持 `start-myflowhub-mcp.ps1` / `install-codex-myflowhub-mcp.ps1` 现有兼容行为。
- 可选:
  - 若当前脚本结构阻碍维护，可抽取少量本地 helper，但仅限脚本内最小重构。
  - 为真实 Hub 验证输出更清晰的阶段摘要和资源清理结果。
- 不做:
  - 不新增 `topicbus`、`subscribe/unsubscribe`、`config_set`、`exec.call` 等新 MCP 能力。
  - 不修改 `internal/mcp` 工具契约，除非为 smoke 暴露已存在能力所必需。
  - 不引入 GUI 自动化或第三方 MCP client。

#### Use Cases
- 维护者需要在真实 Hub 上验证当前 MCP client 不只是“能连上”，而是能完成 auth、authority、management、exec、flow、varstore 的关键路径。
- 维护者需要在默认安全模式下先跑只读链路，再按需开启 authority / 写操作阶段。
- 维护者需要在失败时拿到明确的阶段、上下文参数、保留资源和清理建议，而不是只看到某个 RPC 调用失败。

#### Functional Requirements
- 脚本必须继续复用 `scripts/start-myflowhub-mcp.ps1` 拉起 MCP 进程。
- 脚本必须先验证完整工具集合已暴露，再进入对应阶段调用。
- 脚本必须支持分阶段执行：
  - 基础链路：`initialize`、`tools/list`、`session_connect`、`register/login`、`auth_get_perms`、`auth_list_roles`、`management_list_nodes`
  - 扩展只读链路：`management_node_info`、`management_node_echo`、`management_list_subtree`、`management_config_get/list`、`exec_cap_query`
  - flow / varstore 验证链路
  - authority 验证链路
- authority 和写操作默认不得自动执行；只有显式参数满足时才进入。
- 任何写操作 smoke 都必须使用可追踪、可清理的临时资源命名。
- 写操作完成后必须尽量执行清理，并在清理失败时明确提示残留资源。
- 缺少 authority 权限、缺少 write gate、缺少 flow method、缺少 pending request 等前提时，脚本必须本地显式失败。

#### Non-functional Requirements
- 安全性:
  - 默认运行不得触发 flow 写入、varstore 写入、permit 签发或审批动作。
  - opt-in 写阶段必须明确依赖 `--allow-write` 和显式参数。
- 可维护性:
  - 阶段控制、工具调用、资源命名和结果摘要应集中在脚本中，避免分散魔法值。
  - 文档中的运行方式、参数说明和脚本实际行为保持一致。
- 兼容性:
  - 不破坏现有 `register/login` 基础 smoke 用法。

#### Inputs / Outputs
- 输入:
  - 真实 Hub 地址
  - 登录或注册模式
  - 专用 `ConfigDir`
  - 可选 authority / write / flow / varstore / permit 参数
- 输出:
  - 分阶段日志
  - 成功时的 NodeID / HubID / 角色 / 关键计数和资源信息
  - 失败时的阶段、错误摘要、`ConfigDir`、`stderr` tail、需要保留或回收的资源提示

#### Edge Cases
- 只读工具存在，但某些 Hub/角色缺少 authority 或 management config 权限。
- `flow_list` 为空，导致 `flow_get/status` 无法依赖现有 flow。
- 写阶段已开启，但 `start-myflowhub-mcp.ps1` 未携带 `--allow-write`。
- authority 测试缺少 `authority_id`、`permit device` 或 `pending request id`。
- cleanup 失败导致残留 var / flow / permit。

#### Acceptance Criteria
- `todo.md` 中定义的 smoke 阶段可映射到现有 MCP 工具面，且默认路径安全。
- `scripts/test-myflowhub-mcp-smoke.ps1 -Help` 明确说明各阶段、前提和写入风险。
- 本地验证至少覆盖脚本帮助、主要参数失败路径、`go test ./internal/mcp -count=1`、`go build ./cmd/myflowhub-mcp`。
- 真实 Hub 验证可按阶段执行并覆盖 requirements 中的全链路 acceptance，至少文档和脚本已经为此提供明确入口。

#### Risks
- 真实 Hub smoke 的 authority / 写阶段天然带状态副作用，若参数设计不清晰，容易污染线上或共享环境。
- flow 写入需要合法 graph / method，若默认值不稳，会导致脚本看似失败但问题实际在环境约束。
- 过度扩展单脚本会降低可维护性，因此本轮必须坚持最小改动和参数化分阶段策略。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 采用“在现有 smoke 脚本上做分阶段扩展”的方案：
  - 保留当前基础链路和启动方式。
  - 新增阶段开关和必要参数，把只读验证与写/authority 验证分层。
  - 对写阶段使用临时资源命名，并在阶段结束后回收。
- 选型理由:
  - 最小改动，直接复用现有 JSON-RPC 驱动逻辑。
  - 风险最可控，默认仍可做只读健康检查。
  - 文档和 README 只需围绕同一个脚本更新。

#### Alternatives Considered
- 方案 A: 新建第二个“full smoke”脚本
  - 优点: 可把高级用法和基础用法完全隔离。
  - 不采用原因: 会复制启动、RPC、错误处理和帮助文本，后续维护成本更高。
- 方案 B: 直接把所有 authority / write 动作放入默认 smoke
  - 优点: 一次命令覆盖最多能力。
  - 不采用原因: 对真实 Hub 风险过高，违背最小安全改动原则。

#### Module Responsibilities
- `scripts/test-myflowhub-mcp-smoke.ps1`
  - 负责参数解析、阶段编排、MCP JSON-RPC 调用、资源命名和清理。
- `README.md`
  - 负责公开脚本用法、阶段前提、推荐命令和风险提示。
- `docs/requirements/mcp-client.md`
  - 负责澄清 smoke 验收路径从“基础链路”扩展为“分阶段全链路验证”。
- `docs/specs/mcp-client.md`
  - 负责澄清 smoke 脚本阶段模型、必要输入、写阶段保护和清理约束。

#### Data / Call Flow
1. 脚本拉起 `start-myflowhub-mcp.ps1`。
2. 通过 stdio 完成 `initialize` / `tools/list` / 基础 auth。
3. 进入扩展只读阶段，校验 management / exec / flow-read。
4. 仅当显式启用且参数满足时进入 authority / write 阶段。
5. 对临时资源执行 cleanup。
6. 输出阶段摘要和残留提示。

#### Interface Drafts
- 脚本计划新增或调整的参数类别：
  - 阶段控制:
    - 只读扩展阶段开关
    - authority 阶段开关
    - 写阶段开关
  - authority 输入:
    - `authority_id`
    - `permit device / role`
    - `pending request id`
    - `approve | reject` 选择
  - flow 输入:
    - `executor_node`
    - `flow_id` 或临时 flow 前缀
    - `flow method`
  - varstore 输入:
    - `var name/value`
    - `owner`
- 具体命名以最小兼容改动为准，优先复用现有参数命名风格。

#### Error Handling and Safety
- 基础链路失败时立即停止，不继续后续阶段。
- authority / 写阶段前先校验参数和本地前提。
- cleanup 失败单独报出，不吞掉原始业务阶段错误。
- 写阶段默认关闭；只有显式参数开启并满足前置条件才执行。

#### Performance and Testing Strategy
- 不增加额外进程；继续复用单 MCP 进程和单条 session。
- 本地验证:
  - `go test ./internal/mcp -count=1`
  - `go build ./cmd/myflowhub-mcp`
  - `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Help`
  - 关键失败路径命令
- 真实 Hub 验证:
  - 至少给出 read-only 基础命令
  - 再给出 authority / write 的 staged 命令模板

#### Extensibility Design Points
- 让阶段控制和工具调用映射表可扩展，为后续 `topicbus` 或 subscription smoke 预留入口。
- 资源命名与 cleanup helper 保持独立，避免后续新增写阶段时复制逻辑。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- 目标: 补齐 `MyFlowHub-Win` MCP 在真实 Hub 上的分阶段全链路 smoke 验证能力。
- 当前状态:
  - `myflowhub-mcp` 工具面已覆盖 `session/auth/management/exec/flow/varstore`。
  - 现有 `scripts/test-myflowhub-mcp-smoke.ps1` 只覆盖基础链路到 `management_list_nodes`。
  - authority / exec / flow / varstore 的真实环境验收缺少统一脚本入口。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 change 索引入口。
- 结论:
  - 稳定真相仍在 `docs/requirements/mcp-client.md` 与 `docs/specs/mcp-client.md`
  - 本轮计划文档保留在 worktree 根 `todo.md`
  - 完成后结果归档进入 `docs/change/YYYY-MM-DD_win-mcp-full-chain-smoke.md`
- Requirements impact: clarify
- Specs impact: clarify

#### Related Requirements / Specs / Lessons
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\docs\requirements\mcp-client.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\docs\specs\mcp-client.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- Related lessons:
  - none known yet

#### Executable Task List
- [x] DOCS-1 澄清 requirements/spec/README 中的 smoke 分阶段契约
- [x] SMOKE-1 扩展脚本参数、工具发现和阶段编排
- [x] SMOKE-2 接入扩展只读链路验证
- [x] SMOKE-3 接入 authority / write 阶段与 cleanup
- [x] TEST-1 执行本地验证并整理真实 Hub 运行说明
- [x] REVIEW-1 完成代码评审和归档准备

#### Task Details
##### DOCS-1 - Smoke 契约澄清
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\todo.md`
- Goal: 让 requirements/spec/README 与新的 staged smoke 行为一致
- Files / Modules:
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\docs\requirements\mcp-client.md`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\docs\specs\mcp-client.md`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\README.md`
- Acceptance:
  - 文档明确区分默认基础 smoke 与 opt-in authority / write 阶段
  - 参数前提、清理策略和风险提示可被 README 直接消费
- Test Points:
  - 文档与脚本参数一致性人工校对
- Rollback:
  - 回退文档澄清内容

##### SMOKE-1 - 参数与阶段编排
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\todo.md`
- Goal: 为现有 smoke 脚本增加 staged full-chain 执行骨架
- Files / Modules:
  - `scripts/test-myflowhub-mcp-smoke.ps1`
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\scripts\test-myflowhub-mcp-smoke.ps1`
- Acceptance:
  - 现有基础 smoke 兼容
  - 新增阶段参数后可显式控制 read / authority / write 阶段
  - 完整工具集合校验与阶段摘要可读
- Test Points:
  - `-Help`
  - 缺参失败路径
- Rollback:
  - 回退新增参数和阶段编排逻辑

##### SMOKE-2 - 扩展只读链路
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\todo.md`
- Goal: 将 management / exec / flow-read 的真实 Hub 验证接入 smoke 脚本
- Files / Modules:
  - `scripts/test-myflowhub-mcp-smoke.ps1`
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\scripts\test-myflowhub-mcp-smoke.ps1`
- Acceptance:
  - 可验证 `management_node_info` / `node_echo` / `list_subtree` / `config_get/list`
  - 可验证 `exec_cap_query`
  - 可验证 flow read 路径，必要时根据现有 flow 或参数决定 `get/status`
- Test Points:
  - 本地帮助和参数校验
  - 真实 Hub 命令模板
- Rollback:
  - 回退只读阶段扩展

##### SMOKE-3 - Authority 与写阶段
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\todo.md`
- Goal: 为 authority / flow-write / varstore-write 提供显式 opt-in smoke 路径
- Files / Modules:
  - `scripts/test-myflowhub-mcp-smoke.ps1`
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\scripts\test-myflowhub-mcp-smoke.ps1`
- Acceptance:
  - authority 操作只有显式启用且前提满足时才执行
  - flow / varstore 写操作使用临时资源并在结束后清理
  - cleanup 失败可见
- Test Points:
  - 缺 authority 输入失败路径
  - 缺 write gate 失败路径
- Rollback:
  - 回退 authority / write 阶段代码

##### TEST-1 - 本地验证与真实 Hub 指引
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\todo.md`
- Goal: 在当前环境完成可执行的本地验证，并给出真实 Hub 运行步骤
- Files / Modules:
  - `scripts/test-myflowhub-mcp-smoke.ps1`
  - `README.md`
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\scripts\test-myflowhub-mcp-smoke.ps1`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\README.md`
- Acceptance:
  - 本地验证命令可执行
  - README 给出 read-only / authority / write 三类示例
- Test Points:
  - `go test ./internal/mcp -count=1`
  - `go build ./cmd/myflowhub-mcp`
  - `powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Help`
- Rollback:
  - 回退 README 和脚本说明调整

##### REVIEW-1 - 评审与归档准备
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\todo.md`
- Goal: 复核需求覆盖、风险、测试和归档输入
- Files / Modules:
  - `todo.md`
  - 本轮改动文件
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\todo.md`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\docs\change\*.md`
- Acceptance:
  - Stage 3.3 checklist 全部通过
  - Stage 4 输入完整
- Test Points:
  - review checklist
- Rollback:
  - 返回对应 Task 继续修正

#### Dependencies
- 真实 Hub 环境和对应角色权限决定 authority / write 阶段是否能完全执行。
- flow 写阶段依赖有效的 method / executor 环境。

#### Risks and Notes
- 默认安全优先于“单命令全覆盖”；若真实 Hub 环境复杂，允许按阶段多次执行脚本。
- 当前主线 `repo/MyFlowHub-Win` 存在未提交的 `go.mod` 变更，但本 worktree 独立，不回退主线状态。

#### Parallelism Assessment
- 结论: 不拆子 Agent。
- 原因:
  - 写集集中在单个脚本和同一组文档，耦合度高。
  - 本轮关键路径是脚本阶段设计与统一验收，不存在安全的并行写集拆分。

#### Issue List
- none

### Stage 3.2 - Execution
#### Completed Work
- `DOCS-1`
  - 已更新 `docs/requirements/mcp-client.md`、`docs/specs/mcp-client.md`、`README.md`
  - 明确 staged smoke 合同、opt-in authority/write 风险和示例命令
- `SMOKE-1`
  - 已重建 `scripts/test-myflowhub-mcp-smoke.ps1`
  - 保持 base smoke 兼容
  - 新增 staged 参数、阶段摘要、工具集合预检和统一错误输出
- `SMOKE-2`
  - 已接入 `management_node_info/node_echo/list_subtree/config_get/list`
  - 已接入 `exec_cap_query`
  - 已接入 `flow_list/get/status`，并对无可读 flow 的情况显式记录 skipped
- `SMOKE-3`
  - authority 阶段默认关闭，已接入 pending list + 可选 permit / approve / reject
  - write 阶段默认关闭，且只有显式启用时才自动追加 `--allow-write`
  - write 阶段使用临时 flow / var 资源并显式 cleanup，cleanup 失败可见

#### Local Validation
- `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - 结果：通过
- `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - 结果：通过
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Help`
  - 结果：通过
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login`
  - 结果：按预期失败，提示缺少 `-ConfigDir`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir C:\temp\missing-smoke -EnableWriteSmoke -FlowMethod demo::run`
  - 结果：按预期失败，提示缺少 `-ExecutorNode`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir C:\temp\missing-smoke -EnableAuthoritySmoke -PendingAction approve`
  - 结果：按预期失败，提示缺少 `-PendingRequestID`

#### Residual Risk
- 当前环境未实际连接真实 Hub 执行 staged smoke，因此 authority / write 的真实环境副作用仍需用户在目标环境下分阶段确认。

#### Issue List
- none

### Stage 3.3 - Review
#### Checklist
- [x] 需求影响已写回 `docs/requirements/mcp-client.md`
- [x] 技术契约影响已写回 `docs/specs/mcp-client.md`
- [x] README 示例命令与脚本参数一致
- [x] 脚本帮助、关键失败路径、`go test`、`go build` 已执行
- [x] authority / write 默认安全关闭
- [x] cleanup 失败可见，不吞错误
- [x] 无需回退到 3.2 的评审问题

#### Review Result
- 通过，进入 Stage 4 归档

### Stage 4 - Archive
#### Archive Output
- 已新增 `docs/change/2026-04-01_win-mcp-full-chain-smoke.md`
- 已更新 `docs/change/README.md`
- lessons 结论：本轮暂无需要沉淀到 `docs/lessons` 的新稳定经验

#### Workflow State
- Stage 4 已完成
- 等待用户决定是否结束当前 workflow，或继续下一轮迭代

阻塞：否
进入 4
