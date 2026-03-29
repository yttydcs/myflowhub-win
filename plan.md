# Plan - Win run-dev Proto Replace Path

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-run-dev-proto-path`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-run-dev-proto-path`
- Current Stage: `4`
- External dependencies:
  - `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
    - purpose: provide `github.com/yttydcs/myflowhub-proto/protocol/stream` during development

## Stage Records

### Initialization
- `guide.md`
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `$m-docs` 的 `SKILL.md`、`references/requirement-impact.md`、`references/lessons-rules.md`、`references/templates.md`
- repo / branch / worktree confirmation
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `fix/win-run-dev-proto-path`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-win-run-dev-proto-path`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 修复从 workspace 根 `scripts/run-dev.ps1` 启动 Win 时，Wails 预执行 `go mod tidy` 因 `github.com/yttydcs/myflowhub-proto` 的开发态 `replace` 路径错误而失败的问题。

#### Scope
- 必须
  - 让 Win 仓库当前开发态 `replace github.com/yttydcs/myflowhub-proto` 在以下两类目录都能正确解析：
    - 主仓库路径 `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
    - 独立 worktree 路径 `D:\project\MyFlowHub3\worktrees\...`
  - 保持 `protocol/stream` 的开发态依赖能力，不回退 Stream 模块
  - 让 `GOWORK=off go mod tidy` 在当前 worktree 可通过
  - 归档本次根因、修复方式和可复用排查线索
- 可选
  - 用 Wails CLI 再做一次 bindings 级验证
- 不做
  - 不修改 `scripts/run-dev.ps1` 的窗口编排或 repo 选择逻辑
  - 不发布新的 `myflowhub-proto` semver tag
  - 不移除 Stream 模块当前对开发态 proto worktree 的依赖

#### Use Cases
- 开发者在 workspace 根执行 `.\scripts\run-dev.ps1` 启动 Win，本地 Go 依赖能找到 `proto-stream-subproto`
- 开发者在 Win worktree 内执行 `GOWORK=off go mod tidy` 或 `wails generate module`，不再因为 `../proto-stream-subproto` 指到错误目录而失败

#### Functional Requirements
- `go.mod` 必须继续通过开发态 `replace` 提供 `protocol/stream`
- `replace` 路径必须同时兼容主仓库路径与 worktree 路径
- 错误修复后，`go mod tidy` 不得再尝试访问 `D:\project\MyFlowHub3\repo\proto-stream-subproto`
- 排查文档必须明确记录根因是“相对路径只对单一目录结构成立”，不是 `proto-stream-subproto` 缺失

#### Non-functional Requirements
- 变更面最小，只修改必要文件
- 不引入环境专用绝对路径
- 文档与代码保持一致，避免未来再次把 worktree 专用相对路径带回主线

#### Inputs / Outputs
- 输入
  - `scripts/run-dev.ps1` 固定把 Win 工作目录设为 `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - `go.mod` 当前 `replace github.com/yttydcs/myflowhub-proto => ../proto-stream-subproto`
  - 已存在的 proto worktree：`D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
- 输出
  - 更新后的 `go.mod`
  - 必要的 `docs/change` 和 `docs/lessons` 记录

#### Edge Cases
- `replace` 只在 worktree 中有效，但在主仓库路径无效
- 根 `go.work` 未包含当前 worktree，验证时必须显式使用 `GOWORK=off`
- 后续 release 若切回 semver，不能再依赖本次开发态 `replace`

#### Acceptance Criteria
- `GOWORK=off go mod tidy` 在 `D:\project\MyFlowHub3\worktrees\fix-win-run-dev-proto-path` 通过
- `go mod tidy` 不再报 `replacement directory ../proto-stream-subproto does not exist`
- 若本机 `wails` 可用，则 `GOWORK=off wails generate module` 不再因为同一路径问题失败
- `docs/change` 完整记录本次修复
- `docs/lessons` 可让后续开发者通过错误关键词快速定位本类问题

#### Risks
- 当前 `stream` 仍依赖开发态 proto worktree，这是已知 release blocker，不属于本轮修复范围
- 如果未来再把只对单一目录层级成立的 `replace` 合入主线，根脚本仍会回归

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 方案 A（采用）
  - 将 `go.mod` 的开发态 `replace` 改成相对于模块根的双场景兼容路径：`../../worktrees/proto-stream-subproto`
  - 该路径从主仓库路径和任意根级 Win worktree 都能解析到同一个 workspace 共享 proto worktree
- 不采用方案
  - 改成绝对路径
    - 理由：引入环境硬编码，违反配置化要求
  - 修改 `scripts/run-dev.ps1` 去特殊处理某个 worktree
    - 理由：根因在 Win 模块依赖声明，不在控制脚本
  - 直接移除开发态 `replace`
    - 理由：当前 `myflowhub-proto v0.1.5` 仍缺 `protocol/stream`，会让 Stream 功能直接失效

#### Module Responsibilities
- `go.mod`
  - 定义开发态 proto 依赖解析方式
- `docs/change/*`
  - 归档本次修复背景、验证方式和回滚策略
- `docs/lessons/*`
  - 沉淀“主仓库路径与 worktree 路径共存时，replace 相对路径必须双场景成立”的排查规则

#### Data / Call Flow
1. workspace 根 `scripts/run-dev.ps1` 启动 Win，工作目录进入 `repo\MyFlowHub-Win`
2. `wails dev` 先触发 `go mod tidy`
3. Go 解析 `go.mod` 中 `replace github.com/yttydcs/myflowhub-proto`
4. `replace` 应定位到 `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
5. 依赖收敛后，Wails 才能继续后续 bindings / frontend 流程

#### Interface Draft
- `go.mod`
  - from: `replace github.com/yttydcs/myflowhub-proto => ../proto-stream-subproto`
  - to: `replace github.com/yttydcs/myflowhub-proto => ../../worktrees/proto-stream-subproto`

#### Error Handling and Safety
- 保持 `replace` 为显式声明，不在脚本层静默重写路径
- 验证统一使用 `GOWORK=off`，避免父级 `go.work` 掩盖真实问题

#### Performance and Testing Strategy
- `GOWORK=off go mod tidy`
- `GOWORK=off wails generate module`
  - 仅在本机 `wails` 可用时执行

#### Extensibility Design Points
- 只要后续开发态共享 repo 仍放在 workspace 根 `worktrees/` 下，该相对路径模型可复用到主仓库路径和 worktree 路径
- 正式 release 收口后，直接移除 `replace` 即可，不影响其它模块

#### Issue List
- none

### Stage 3.1 - Planning
#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- docs tree 无需 bootstrap 或 repair
- stable truth 继续保留在：
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
- workflow result 进入：
  - `docs/change/2026-03-29_win-run-dev-proto-path.md`
- reusable troubleshooting knowledge 进入：
  - `docs/lessons/wails-binding-proto-drift.md`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements
  - `D:\project\MyFlowHub3\worktrees\fix-win-run-dev-proto-path\docs\requirements\stream.md`
- Related specs
  - `D:\project\MyFlowHub3\worktrees\fix-win-run-dev-proto-path\docs\specs\stream.md`
- Related lessons
  - `D:\project\MyFlowHub3\worktrees\fix-win-run-dev-proto-path\docs\lessons\wails-binding-proto-drift.md`

#### Executable Task List
- [x] `RUNDEV-1` 修复 Win `go.mod` 的 proto 开发态 `replace` 路径
- [x] `RUNDEV-2` 执行 `go mod tidy` / Wails 验证并确认错误消失
- [x] `RUNDEV-3` 完成 3.3 checklist
- [x] `RUNDEV-4` 归档 `docs/change` 与 `docs/lessons`

#### Task Details
##### `RUNDEV-1` - Fix proto replace path
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-run-dev-proto-path`
- Goal
  - 让 `myflowhub-proto` 的开发态 `replace` 同时兼容主仓库路径与 worktree 路径
- Files
  - `go.mod`
- Acceptance
  - `go mod tidy` 不再访问 `D:\project\MyFlowHub3\repo\proto-stream-subproto`
- Tests
  - `GOWORK=off go mod tidy`
- Rollback
  - 回退 `go.mod`

##### `RUNDEV-2` - Validate tidy and Wails path
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-run-dev-proto-path`
- Goal
  - 确认本轮修复覆盖用户报错链路，而不只是静态改字符串
- Files
  - no additional implementation files required
- Acceptance
  - `GOWORK=off go mod tidy` 通过
  - 若 `wails` 可用，`GOWORK=off wails generate module` 不再因 replace 路径失败
- Tests
  - `GOWORK=off go mod tidy`
  - `GOWORK=off wails generate module`
- Rollback
  - 仅回退验证产物或文档记录

##### `RUNDEV-3` - Review
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-run-dev-proto-path`
- Goal
  - 对照需求、架构、稳定性和测试完整性做 3.3 checklist 复核
- Files
  - `plan.md`
- Acceptance
  - 3.3 各项给出 `通过` / `不通过`
- Tests
  - review checklist
- Rollback
  - 更新 `plan.md` 复核记录

##### `RUNDEV-4` - Archive change and lesson
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-run-dev-proto-path`
- Goal
  - 让后续开发者能从 `docs/change` 和 `docs/lessons` 快速复用这次排查结果
- Files
  - `docs/change/2026-03-29_win-run-dev-proto-path.md`
  - `docs/change/README.md`
  - `docs/lessons/wails-binding-proto-drift.md`
  - `docs/lessons/README.md`
- Acceptance
  - change/lesson 包含关键词、触发条件、快速检查和回滚方案
- Tests
  - manual doc review
- Rollback
  - 回退新增 / 更新的文档

#### Dependencies
- `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
  - 提供 `protocol/stream`

#### Risks and Notes
- 根 `scripts/run-dev.ps1` 当前固定启动 `repo\MyFlowHub-Win`，所以任何只在 worktree 路径成立的相对 `replace` 都会回归
- `docs/change/2026-03-28_win-stream-module.md` 中记录的是上轮实现时的开发态写法，本轮通过新增归档说明后续修正，不把旧 change 文档改写成“当前真源”

#### Parallelism Assessment
- 不派发子Agent
- 原因
  - 当前会话未获得显式子Agent授权
  - 代码变更面很小，串行完成更快且审计成本更低

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Task Mapping
- `RUNDEV-1`
  - `go.mod`
  - `go.sum`
- `RUNDEV-2`
  - validation only
- `RUNDEV-4`
  - `docs/change/2026-03-29_win-run-dev-proto-path.md`
  - `docs/change/README.md`
  - `docs/lessons/wails-binding-proto-drift.md`
  - `docs/lessons/README.md`

#### File-level Change Summary
- `go.mod`
  - 将 proto 开发态 `replace` 改为 `../../worktrees/proto-stream-subproto`
- `go.sum`
  - 在本地 `replace` 生效后移除 `myflowhub-proto v0.1.5` 的远端校验和记录
- `docs/lessons/wails-binding-proto-drift.md`
  - 增补“相对 `replace` 只对单一路径层级成立”的变体
- `docs/lessons/README.md`
  - 更新 lesson 索引关键词
- `docs/change/2026-03-29_win-run-dev-proto-path.md`
  - 归档本次修复背景、验证和回滚
- `docs/change/README.md`
  - 挂载本次归档入口

#### Design Notes
- 选择修复 `go.mod` 而不是改 `scripts/run-dev.ps1`
  - 脚本固定使用主仓库路径启动 Win 是合理行为，模块依赖声明必须兼容该入口
- 选择相对路径 `../../worktrees/proto-stream-subproto`
  - 同时兼容主仓库路径和根级 worktree，且不引入环境专用绝对路径
- 保留开发态 `replace`
  - 当前 `myflowhub-proto v0.1.5` 仍缺 `protocol/stream`

#### Validation
- `Resolve-Path ../../worktrees/proto-stream-subproto`
  - 结果：从 `repo/MyFlowHub-Win` 与 `worktrees/fix-win-run-dev-proto-path` 两个模块根都解析到 `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
- `$env:GOWORK='off'; go mod tidy`
  - 结果：通过
- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：仍打印既有 `Not found: time.Time`，但退出码为 0

#### Blockers
- none

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 已覆盖主仓库路径启动与 worktree 启动的双场景依赖解析
- 架构合理性：通过
  - 根因收敛在模块依赖声明，未把路径修复扩散到控制脚本
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅涉及依赖路径与文档，无新增运行时性能风险
- 可读性与一致性：通过
  - 使用明确的 workspace 相对路径，且文档同步记录原因
- 可扩展性与配置化：通过
  - 未引入环境专用绝对路径；后续 release 仍可直接移除 `replace`
- 稳定性与安全：通过
  - `GOWORK=off` 验证通过，Wails 入口不再因错误目录解析失败
- 测试覆盖情况：通过
  - 已执行 `go mod tidy`、`wails generate module` 和路径解析验证
- 子Agent治理与审计：通过
  - 未使用子Agent

### Stage 4 - Change Archive
#### $m-docs Check
- 使用 `$m-docs` 校验 plan/change/lessons 路由
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `updated`
- 新增：
  - `docs/change/2026-03-29_win-run-dev-proto-path.md`
- 更新：
  - `plan.md`
  - `docs/change/README.md`
  - `docs/lessons/wails-binding-proto-drift.md`
  - `docs/lessons/README.md`

#### Archive Status
- 已完成 repo-local 归档
- 等待用户确认是否结束 workflow

---

## Imported Workflow - 2026-03-29 Stream Announce Timeout

### Workflow Information
- Source branch: `fix/stream-announce-timeout`
- Source worktree: `D:\project\MyFlowHub3\worktrees\fix-stream-announce-timeout\MyFlowHub-Win`
- Merge target: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Final status: `completed and merged`

### Goal
- 修复 Stream 页面创建本地 source 时出现的 `创建本地 Source 失败。` / `stream announce: request timed out`
- 收敛 Win `StreamService` 的 CTRL framing 与 await 行为，避免其它 stream 控制动作继续复现同类 timeout

### Key Findings
- `stream` 子协议的控制请求 / 响应都要求 `KindCtrl + JSON(action,data)` framing
- Win 初版 `internal/services/stream/service.go` 发送的是裸 JSON
- SDK 当前仅为 `file` 子协议在 await 解码时剥离 `KindCtrl`，导致 `stream` 的 `*_resp` 无法被通用 await 匹配

### Merged Changes
- `internal/services/stream/service.go`
  - 新增 stream CTRL payload 编码 helper
  - 新增 stream 局部 await helper
  - 统一 stream 控制动作的 request / response 解码路径
- `internal/services/stream/service_test.go`
  - 新增 CTRL request prefix、CTRL response match、timeout 映射回归测试
- `docs/change/2026-03-29_win-stream-announce-timeout.md`
  - 记录本次修复背景、验证和回滚
- `docs/lessons/stream-ctrl-await-mismatch.md`
  - 固化“子协议 CTRL framing 与 await 解码错位导致 timeout”的排查规则

### Validation
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./internal/services/stream -count=1`
  - 结果：通过
- 验证备注
  - 当前嵌套 worktree 的 `go.mod replace` 解析路径不适合直接 `GOWORK=off` 编译
  - 测试期间临时创建 junction 指向 `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
  - 验证完成后已清理，不留工作区残留

### Related Docs
- Change: `docs/change/2026-03-29_win-stream-announce-timeout.md`
- Lesson: `docs/lessons/stream-ctrl-await-mismatch.md`
- Requirements: `docs/requirements/stream.md`
- Specs: `docs/specs/stream.md`
