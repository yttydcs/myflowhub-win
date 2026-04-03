# Plan - Win Flow Orchestrator Editor

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Merge Source Branch: `feat/win-orchestrator-editor`
- Base Branch: `main`
- Former Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Current Stage: `workflow ended`

## Stage Records

### Initialization
- `guide.md`:
  - workspace 规则已确认
  - 所有 worktree 必须位于 `D:\project\MyFlowHub3\worktrees\`
  - 子协议稳定文档以 `repo\MyFlowHub-Server\docs` 为真相源
- base/worktree confirmation:
  - `MyFlowHub-Win` 当前主线分支：`main`
  - 本轮执行 worktree：`D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
  - 主仓和原 repo 均保持控制面角色，不在主路径直接实现

### Stage 1 - Requirements Analysis
#### Goal
- 补齐 Win Flow 编辑器对当前稳定 flow 能力的 authoring 支持，优先解决 `cron` trigger 和新增节点种类在编辑器中的识别、创建、保存与安全回显问题。
- 本轮不新增 runtime/protocol 能力，只补编辑器侧支持。

#### Scope
- 必须：
  - 支持 `trigger.type=cron`
  - 支持节点种类：`transform`、`branch`、`foreach`、`subflow`
  - 保持 `call` / `compose` / `set_var` 与既有 trigger 的编辑体验不回归
  - 对复杂节点保留 `Advanced JSON` 真相源，不静默改写 spec
- 可选：
  - 为部分新增节点补最小说明和基础 authoring 文案
- 不做：
  - 不修改 Flow runtime / Proto / Server
  - 不实现 `foreach.body` 的嵌套图形编辑器
  - 不引入脚本表达式、并行 `foreach`、cross-executor `subflow`

#### Use Cases
- 用户在 Win 编辑器中创建 `cron` flow，而不是手写 trigger JSON。
- 用户在 Win 编辑器中新增 `transform` 节点并通过 Advanced JSON 配置表达式树。
- 用户读取已有含 `branch / foreach / subflow` 的 flow 时，编辑器不再把它们错误降级成 `call`。
- 用户为 `branch` 节点的出边配置 `edge.case`，保存后不丢失分支路由信息。
- 用户部署项目时可把默认 trigger 改为 `cron`。

#### Functional Requirements
- 编辑器必须识别 `trigger.type=cron` 并支持 round-trip 保存。
- 编辑器必须识别 `kind=transform|branch|foreach|subflow` 并支持 round-trip 保存。
- 编辑器必须保留并允许最小编辑 `branch` 出边的 `edge.case`。
- 对当前无法安全表单化的节点，编辑器必须直接引导到 `Advanced JSON`，不能伪装成错误表单。
- `Add Node` 与 inspector 的 kind 选择器必须覆盖上述新增 kind。
- 画布节点标签与节点摘要必须覆盖新增 kind。

#### Non-functional Requirements
- 变更面保持最小，优先复用现有 `flow.ts` store、inspector 和 JSON authoring 机制。
- 不得静默丢失复杂 spec 字段。
- 新增支持不能破坏现有 visual form / binding / deployment 流程。

#### Inputs / Outputs
- 输入：
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\requirements\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\specs\flow-editor-visual-form.md`
- 输出：
  - `MyFlowHub-Win` 编辑器代码改动
  - 本 worktree 根 `plan.md`
  - 必要的 Win 仓 requirements/specs 更新
  - 归档到 `docs/change/`

#### Edge Cases
- 已有 graph 含新增 kind，但 UI 错误映射为 `call`
- `cron` trigger 读取时被错误降为 `interval`
- `foreach` 带嵌套 `body` 时不应尝试表单化展开
- `branch` 边若丢失 `case`，保存后会破坏路由语义
- 用户从新增 kind 切回旧 kind 时，不能无声清空高级 JSON

#### Acceptance Criteria
- 编辑器可正确读取并保存 `cron` trigger。
- 编辑器可正确读取并保存 `transform / branch / foreach / subflow` 节点。
- 编辑器可正确读取、保留并编辑 `branch` 的 `edge.case`。
- 新增复杂节点默认进入安全的 JSON authoring 路径。
- 现有 `call / compose / set_var` 与 deployment trigger 逻辑不回归。

#### Risks
- `foreach` 的嵌套 `body` 若误入普通模式，极易造成 spec 丢失。
- 现有 store 对 kind/trigger 的硬编码较多，容易遗漏 import/export 分支。
- `flow-editor-visual-form` 本地 requirements/spec 需要同步扩展，否则实现会和稳定文档漂移。

#### Issue List
- 无

### Stage 2 - Architecture Design
#### Overall Solution
- 对 `cron` 做正式表单支持。
- 对 `transform / branch / foreach / subflow` 做“可识别、可创建、可保存、默认 JSON-only”的 editor 支持。
- 通过显式 helper 抽象“哪些节点支持 form mode”，把复杂节点限制在 `Advanced JSON`，避免错误表单。

#### Alternatives Considered
- 全量为新增节点补完整表单：
  - 不选；`foreach.body` 需要嵌套图编辑，超出本轮最小安全范围。
- 仅扩 kind/trigger 枚举，不处理 UI mode：
  - 不选；会继续出现错误渲染和误保存。

#### Module Responsibilities
- `frontend/src/stores/flow.ts`
  - kind/trigger 扩展、payload round-trip、spec mode 治理、保存校验
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - kind 选择、json-only 提示、cron 和新增 kind 的 authoring 入口文案
- edge inspector
  - `branch` 场景的 `edge.case` 最小编辑
- `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
  - 新增 kind 创建入口
- `frontend/src/components/flow/FlowNode.vue`
  - 节点 kind 标签显示
- `frontend/src/stores/flowProjects.ts`
  - deployment trigger draft / wire 扩展到 `cron`
- `frontend/src/pages/Flow.vue`
  - deployment trigger UI 增加 `cron`
- tests
  - store 和 UI 回归覆盖

#### Data / Call Flow
1. 读取 flow payload 时，trigger 分支新增识别 `cron`。
2. 读取 graph 节点时，kind 分支新增识别 `transform / branch / foreach / subflow`。
3. 新增复杂 kind 时，初始 `specJson` 使用对应最小脚手架，默认 `specEditorMode=json`。
4. `FlowEdge` 扩展 `case` 后，读取和保存都必须完整保留该字段。
5. 保存时，`cron` 生成正式 trigger wire；复杂节点按 `specJson` 校验后原样输出并补 `_ui`。

#### Interface Drafts
- `type FlowNodeKind = "call" | "compose" | "transform" | "set_var" | "branch" | "foreach" | "subflow"`
- `type FlowTriggerType = "interval" | "cron" | "event" | "var_changed"`
- helper:
  - `supportsFormMode(kind)`
  - `defaultSpecEditorMode(kind)`
  - `kindDefaultSpec(kind)`

#### Error Handling and Safety
- 不支持普通模式的 kind 只允许 JSON authoring。
- `cron` 为空或非法时在保存阶段报错，不做静默回退。
- 复杂节点的 advanced spec 必须保持 JSON object 约束，并仅补 `_ui`。

#### Performance and Testing Strategy
- 不新增第二套复杂节点状态模型，继续以 `specJson` 为复杂节点真相源。
- 测试覆盖：
  - kind / trigger load-save round-trip
  - add node / inspector / labels
  - 旧节点与旧 trigger 不回归

#### Extensibility Design Points
- 未来若要给 `transform` 或 `subflow` 单独补表单，只需要在 `supportsFormMode(kind)` 和对应 form 渲染处增量扩展。
- `foreach` 维持 json-only，为未来子图编辑器留清晰边界。

#### Issue List
- 无

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 `MyFlowHub-Win` 的 `flow.ts` 仍把节点种类限制为 `call|compose|set_var`，trigger 仍限制为 `interval|event|var_changed`。
- 当前结果是：稳定 flow 能力已经存在，但 Win 编辑器仍无法正确 author `cron` 和新增高阶节点。
- 本轮目标是把“错误降级/无法创建”先补成“安全 authoring + 正确 round-trip”，而不是在这一轮发明完整子图编辑器。

#### Docs Governance Routing Decision
- Requirements impact: `updated`
- Specs impact: `updated`
- stable truth：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
- workflow results：
  - `docs/change/2026-04-03_win-flow-orchestrator-editor.md`
- lessons：
  - 暂未确认；若本轮暴露“复杂节点必须强制 JSON-only”这类可复用规则，再进入 `docs/lessons/`

#### Related Requirements / Specs / Lessons
- Related requirements:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\specs\flow-editor-visual-form.md`
- Related lessons:
  - 无

#### Executable Task List
- [x] `WIN-ORCH-DOC-1` 更新本地 requirements/spec，收口 editor 对 `cron` 与新增 kind 的支持边界
- [x] `WIN-ORCH-RT-1` 扩展 `flow.ts` 的 kind/trigger 数据模型与 load-save round-trip
- [x] `WIN-ORCH-RT-2` 扩展 inspector / add-node / canvas UI，并为复杂 kind 提供 json-only 提示
- [x] `WIN-ORCH-RT-3` 扩展项目部署 trigger 到 `cron`
- [x] `WIN-ORCH-TEST-1` 更新 store / UI 测试并执行回归

#### Task Details
##### `WIN-ORCH-DOC-1` - Update Win Flow Editor Stable Docs
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - 让 Win 仓本地 requirements/spec 和本轮 editor 行为一致
- Files / Modules:
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
- Write Set:
  - 上述两份稳定文档
- Acceptance:
  - 文档明确 `cron` 和新增 kind 的 editor 支持边界
  - 明确复杂 kind 先走 JSON-only
- Test Points:
  - 文档与代码行为一致，无与 server stable spec 冲突的表述
- Rollback:
  - 回退上述文档到本轮前状态

##### `WIN-ORCH-RT-1` - Extend Store Round-trip for Cron and New Node Kinds
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - 让 store 正确识别、创建、保存 `cron` 和新增 kind
- Files / Modules:
  - `frontend/src/stores/flow.ts`
- Write Set:
  - `frontend/src/stores/flow.ts`
- Acceptance:
  - 读取和保存时不再把新增 kind 降成 `call`
  - `cron` trigger 正确 round-trip
  - 不支持 form 的 kind 默认进入 `json`
- Test Points:
  - load/save round-trip
  - spec mode gating
  - 旧 kind / 旧 trigger 不回归
- Rollback:
  - 回退 `frontend/src/stores/flow.ts`

##### `WIN-ORCH-RT-2` - Update Editor UI for New Node Kinds
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - UI 层提供新增 kind 入口、标签和 json-only 提示
- Files / Modules:
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
- `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
- `frontend/src/components/flow/FlowNode.vue`
- 必要时：`frontend/src/components/flow/editor/FlowEdgeInspector.vue`
- 如需：`frontend/src/windows/FlowEditorWindow.vue`
- Write Set:
  - 上述 UI 文件
- Acceptance:
- Add Node 和 inspector 可选择新增 kind
- 选中 branch 出边时可编辑 `edge.case`
- 复杂 kind 不会误进 compose/set_var 表单
- 画布标签正确显示
- Test Points:
  - UI 渲染和交互测试
- Rollback:
  - 回退上述 UI 文件

##### `WIN-ORCH-RT-3` - Add Cron Support to Project Deployment Trigger
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - 项目 deployment trigger 支持 `cron`
- Files / Modules:
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/pages/Flow.vue`
- Write Set:
  - 上述两个文件
- Acceptance:
  - 项目 trigger draft / deploy wire / 页面表单正确支持 `cron`
- Test Points:
  - trigger normalize / wire conversion
  - 部署表单渲染
- Rollback:
  - 回退上述两个文件

##### `WIN-ORCH-TEST-1` - Regression Tests
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - 覆盖本轮新增 round-trip 和 UI 渲染分支
- Files / Modules:
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - 必要时新增相关测试
- Write Set:
  - 上述测试文件
- Acceptance:
  - 新增测试可稳定捕获回归
- Test Points:
  - `npm`/`vitest` 前端测试
  - 如可行，Wails/Go 基础测试保持通过
- Rollback:
  - 回退新增测试与断言

#### Dependencies
- `WIN-ORCH-DOC-1` 先于 Stage 4 完成即可，但本轮代码语义必须以其为依据
- `WIN-ORCH-RT-1` 是 `WIN-ORCH-RT-2` / `WIN-ORCH-RT-3` / `WIN-ORCH-TEST-1` 的前置
- `WIN-ORCH-RT-2` 和 `WIN-ORCH-RT-3` 可顺序实现，避免同一 store 接口并发改动

#### Risks and Notes
- `foreach` 不做子图表单，这一点必须在 UI 和 docs 中写清楚
- 当前 repo 存在旧 `todo.md` / `plan.md` 遗留记录，本轮只以当前 worktree 根 `plan.md` 为控制文档
- 不派发子 Agent；当前任务写集高度重叠，且用户未授权并行代理执行

#### Parallelism Assessment
- 评估结果：本轮不并行
- 原因：
  - `flow.ts` 是核心共享写集
  - inspector / add-node / flowProjects 都依赖其类型与 helper 变更
  - host policy 也要求未获用户明确许可时不启用子 Agent

#### Issue List
- 无

阻塞：否
进入 3.2

### Stage 3.2 - Execution
#### Completed Tasks
- `WIN-ORCH-RT-1`
  - 已补齐 `flow.ts` 对 `cron` trigger 和 `transform/branch/foreach/subflow` 的识别、创建、保存与 round-trip。
  - 已把 `FlowEdge.case` 纳入 store 状态和 graph 导出，避免 `branch` 路由信息丢失。
- `WIN-ORCH-RT-2`
  - 已扩展 Add Node、FlowNode 标签、Inspector kind 选择器和 JSON-only 提示。
  - 已新增 `FlowEdgeInspector.vue`，并在 `FlowEditorWindow.vue` 中接入选中边的侧栏编辑。
  - 已修复 `FlowEditorWindow.vue` 中旧 `nodeDetailOpen` 残留引用，避免编译失败。
- `WIN-ORCH-RT-3`
  - 已扩展 `flowProjects.ts` 和 `Flow.vue` 的项目 trigger draft / wire / UI 到 `cron`。
  - 已明确策略：本地 draft 可保留 `cronExpr`，严格部署时空表达式显式报错。
- `WIN-ORCH-DOC-1`
  - 已更新本地 requirements/spec，明确 `cron`、新增 kind、JSON-only 边界与 `edge.case` 最小编辑契约。
- `WIN-ORCH-TEST-1`
  - 已补充 store / inspector / window 回归测试，覆盖 `cron`、JSON-only kinds、`edge.case` 和边侧栏渲染。

#### Validation
- 定向前端测试：
  - `npx vitest run src/stores/flow.test.ts src/stores/flowProjects.test.ts src/components/flow/editor/FlowNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：通过，`32 passed`
- 全量前端测试：
  - `npm test`
  - 结果：失败，但失败点为仓库基线问题
  - 失败文件：`src/stores/stream.test.ts`
  - 失败原因：缺失 `../../wailsjs/runtime/runtime`
- 前端构建：
  - `npm run build`
  - 结果：失败，但失败点为仓库基线问题
  - 失败原因：缺失 `../../wailsjs/go/session/SessionService`

#### Issue List
- 基线问题：
  - 当前 worktree 缺失 `wailsjs` 生成物，导致全量前端测试与整包构建无法在本轮拿到全绿。
  - 该问题不在本轮 flow editor 改动写集内，先记录到归档，不在本轮扩散修复。

### Stage 3.3 - Code Review
#### Review Checklist
- 需求覆盖：通过
  - `cron` trigger、新增节点种类、`branch edge.case` 与 JSON-only 边界均已覆盖。
- 架构合理性：通过
  - `flow.ts` 继续作为 round-trip 真相源；复杂节点未引入第二套表单状态模型。
- 安全性：通过
  - JSON-only kinds 禁止切回 form；`cron` 空表达式在严格路径显式报错。
- 回归风险：通过
  - 已补关键 store/UI 测试；旧 `nodeDetailOpen` 编译残留已修复。
- 文档一致性：通过
  - 本地 requirements/spec 已与实际行为同步。
- 子 Agent 治理：通过
  - 本轮未使用子 Agent，符合计划中的并行限制。

#### Result
- Code Review 通过，进入 Stage 4。

### Stage 4 - Archive
#### Outputs
- `docs/change/2026-04-03_win-flow-orchestrator-editor.md`
- `docs/change/README.md`

#### Lessons Decision
- `Lessons impact: none`
- 原因：
  - 本轮暴露的是当前 Win worktree 的 `wailsjs` 基线缺失问题，仓库已有多处同类记录；
  - 新增 JSON-only 节点与 `edge.case` 约束已进入稳定 requirements/spec，无需额外拆成 lessons。

#### Workflow Status
- 本轮实现、审核和归档均已完成。
- 未执行 workflow end；等待用户决定是否继续下一轮或结束当前 workflow。

## Iteration 2 - Advanced Node Form Authoring

### Stage 1 - Requirements Analysis
#### Goal
- 在当前已支持 `cron` 和新增节点 round-trip 的基础上，继续补齐高级节点的普通模式 authoring。
- 本轮优先把 `transform`、`branch`、`subflow` 从 JSON-only 提升到可安全表单编辑；`foreach` 继续维持 JSON-only。

#### Scope
- 必须：
  - `transform` 支持普通模式 authoring，并可在 form/json 间稳定 round-trip
  - `branch` 支持普通模式 authoring，包括 case 列表与 `default_case`
  - `subflow` 支持普通模式 authoring，包括 `flow_id`、`input_template`、`inputs`、`result_node_id`
  - `branch` 的 `edge.case` 继续作为独立边侧栏编辑能力保留，不并回节点表单
  - `foreach` 继续保持 JSON-only，且 UI/文档明确此边界
- 可选：
  - 为 `transform` 提供顶层表达式模式选择与基础运算白名单选择
  - 为 `branch` / `subflow` 增加更清晰的辅助文案与默认值
- 不做：
  - 不实现 `foreach.body` 嵌套图编辑器
  - 不实现完整递归式 transform expression tree 可视化编辑器
  - 不修改 Flow runtime / Server / Proto

#### Use Cases
- 用户选择 `transform` 节点时，可以在不直接编辑整段 spec 的情况下配置顶层表达式。
- 用户选择 `branch` 节点时，可以通过表单维护 case 名称、匹配来源、比较运算和默认分支。
- 用户选择 `subflow` 节点时，可以直接填写目标 `flow_id`、输入模板和结果节点 ID，并复用现有 bindings 编辑器。
- 用户把高级节点从 form 切到 JSON，再切回 form 时，已支持的字段可稳定还原。
- 用户打开 `foreach` 节点时，仍然收到明确的 JSON-only 提示，不会误以为当前版本支持 body 子图表单。

#### Functional Requirements
- 编辑器必须允许 `transform`、`branch`、`subflow` 进入普通模式。
- `transform` 普通模式至少必须支持以下顶层表达式变体：
  - `literal`
  - `source`
  - `op + args`
  - `object`
  - `array`
- `transform.source` 必须复用现有 flow binding 来源约束，不允许非法 JSON Pointer 或非法 `flow_var` 名称。
- `branch` 普通模式必须支持：
  - case 名称
  - `match.source`
  - `match.op`
  - `match.value`
  - `default_case`
- `branch` 普通模式必须校验 case 名称非空且在同一节点内唯一。
- `subflow` 普通模式必须支持：
  - `flow_id`
  - `input_template`
  - `inputs`
  - `result_node_id`
- `subflow.flow_id` 必须按 UUID 规则校验。
- `transform/branch/subflow` 在普通模式下保存时不得静默删除额外高级字段；超出普通模式能力的 spec 必须要求回退 JSON。
- `foreach` 不得被错误开放到普通模式。

#### Non-functional Requirements
- 继续保持最小安全改动，优先在现有 `flow.ts` / `FlowNodeInspector.vue` 上增量扩展。
- 复杂表达式与匹配规则的校验必须集中在 store，不把协议验证散落到 Vue 模板中。
- 新增 form 状态模型必须明确、可回滚，不引入和 `specJson` 相互漂移的第三套真相源。

#### Inputs / Outputs
- 输入：
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\requirements\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\specs\flow-editor-visual-form.md`
- 输出：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 相关测试与本地 requirements/spec 更新

#### Edge Cases
- `transform.expr` 不符合受支持顶层变体时，切回普通模式必须显式失败。
- `branch.default_case` 指向不存在 case 时必须报错。
- `branch.match.op=exists` 时不能强制要求 `value`。
- `subflow.flow_id` 非 UUID 时必须报错。
- 已有高级 JSON 中存在当前普通模式未覆盖的字段时，不能静默吞掉。

#### Acceptance Criteria
- `transform`、`branch`、`subflow` 可在普通模式下创建、编辑、保存并 round-trip。
- `branch` 节点表单与 `edge.case` 分工清晰，保存后路由语义不丢失。
- `foreach` 仍为 JSON-only，且不回归到错误表单。
- 本轮新增表单路径具备回归测试覆盖。

#### Risks
- `transform` 若试图做完整递归表达式编辑器，范围会迅速膨胀。
- `branch` 的 case 表单若和 `edge.case` 约束不同步，会再次导致保存后路由损坏。
- `subflow` 若不校验 `flow_id`，会把协议错误延后到运行期。

### Stage 2 - Architecture Design
#### Overall Solution
- 扩展 `FlowNodeDraft`，为 `transform`、`branch`、`subflow` 增加显式表单字段。
- `transform` 采用“顶层表达式结构化表单 + 嵌套 JSON 局部编辑”的折中方案，避免一次性做完整表达式树 UI。
- `branch` 采用 case 列表表单；`edge.case` 继续由边 inspector 维护。
- `subflow` 复用 `compose/set_var` 的模板与 bindings 编辑骨架，但换成 `input_template` / `flow_id` / `result_node_id` 语义。

#### Alternatives Considered
- 继续把 `transform` 保持 JSON-only，只做 `branch/subflow`：
  - 不选；用户明确要求继续推进高级节点表单化，`transform` 是其中核心能力。
- 为 `transform` 做完整递归式表达式可视化编辑器：
  - 不选；范围过大，和本轮“最小安全前进”目标不匹配。
- 把 `branch.edge.case` 并回节点表单：
  - 不选；与协议真实模型不一致，也会让边与节点状态脱节。

#### Module Responsibilities
- `frontend/src/stores/flow.ts`
  - 扩展高级节点草稿字段
  - form/json 互转
  - 高级节点普通模式校验与 spec 生成
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 渲染 `transform` / `branch` / `subflow` 表单
  - 复用或抽出公共 source/binding 输入块
- tests
  - 覆盖 round-trip、校验失败和 inspector 渲染
- docs
  - 更新本地 requirements/spec，移除已过时的 JSON-only 限制表述

#### Data / Call Flow
1. `parseSpecDraft(...)` 在读取 `transform/branch/subflow` 时，把 spec 映射为显式表单字段与 `specJson`。
2. 普通模式下，`buildFormSpec(...)` 根据表单字段重新生成协议 spec。
3. JSON 模式下，`buildAdvancedSpec(...)` 仍作为完整兜底；切回普通模式时，只在 spec 处于支持子集内才允许还原。
4. `branch` 节点的出边 `edge.case` 不进入节点 spec；节点 case 名称和边 case 通过现有 graph 校验协同约束。

#### Interface Drafts
- `FlowNodeDraft` 新增：
  - `transformExprMode`
  - `transformLiteralJson`
  - `transformSource`
  - `transformSourceRequired`
  - `transformOp`
  - `transformArgsJson`
  - `transformObjectJson`
  - `transformArrayJson`
  - `branchCases`
  - `branchDefaultCase`
  - `subflowId`
  - `subflowInputTemplate`
  - `subflowResultNodeId`
- 新增 draft type：
  - `FlowSourceDraft`
  - `FlowBranchCaseDraft`

#### Error Handling and Safety
- 不支持回到普通模式的高级 spec 必须显式报错。
- `transform` 的 JSON 局部字段必须分别做合法 JSON 校验。
- `branch` case 名称重复、source 非法、`default_case` 缺失映射时必须报错。
- `subflow.flow_id` 非 UUID 时必须报错。

#### Performance and Testing Strategy
- 不引入新远程 I/O；所有校验仍在本地 store 完成。
- 测试重点：
  - `transform/branch/subflow` form/json round-trip
  - inspector 渲染与控件可用性
  - `foreach` 继续 JSON-only
  - `branch` 与 `edge.case` 协同不回归

#### Extensibility Design Points
- `FlowSourceDraft` 若设计成通用结构，后续可复用到 `foreach` 和更深层 transform/source 编辑。
- `transform` 顶层模式先行，后续若需要完整递归 builder，可在现有 mode 基础上递进扩展，而不必推翻当前数据模型。

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 Win 编辑器已支持高级节点的识别、创建、保存，但 `transform/branch/foreach/subflow` 仍默认 JSON-only。
- 用户已明确要求继续推进“高级节点的表单化 authoring”。
- 本轮默认实现边界：
  - `transform`、`branch`、`subflow` 提供普通模式
  - `foreach` 继续 JSON-only

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: `updated`
- Specs impact: `updated`
- stable truth：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
- workflow results：
  - `docs/change/2026-04-03_win-flow-advanced-node-forms.md`
- lessons：
  - 暂未新增

#### Related Requirements / Specs / Lessons
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- Related lessons:
  - 无

#### Executable Task List
- [x] `WIN-ORCH-DOC-2` 更新本地 requirements/spec，明确高级节点普通模式边界
- [x] `WIN-ORCH-RT-4` 扩展 `flow.ts` 的高级节点草稿模型与 form/json 互转
- [x] `WIN-ORCH-RT-5` 扩展 inspector，实现 `transform/branch/subflow` 普通模式 authoring
- [x] `WIN-ORCH-TEST-2` 补高级节点表单化回归测试

#### Task Details
##### `WIN-ORCH-DOC-2` - Update Stable Docs for Advanced Node Forms
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - 让本地 requirements/spec 明确 `transform/branch/subflow` 普通模式与 `foreach` JSON-only 边界
- Files / Modules:
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
- Write Set:
  - 上述两份稳定文档
- Acceptance:
  - 文档与实际 UI 行为一致
- Test Points:
  - 文档表述不与 server stable spec 冲突
- Rollback:
  - 回退上述文档

##### `WIN-ORCH-RT-4` - Extend Store Draft Model for Advanced Node Forms
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - 让 `transform/branch/subflow` 可在普通模式和 JSON 模式之间稳定互转
- Files / Modules:
  - `frontend/src/stores/flow.ts`
- Write Set:
  - `frontend/src/stores/flow.ts`
- Acceptance:
  - 支持 form/json round-trip
  - 明确失败路径，不静默吞字段
  - `foreach` 仍保持 JSON-only
- Test Points:
  - round-trip
  - validation
  - legacy payload load
- Rollback:
  - 回退 `frontend/src/stores/flow.ts`

##### `WIN-ORCH-RT-5` - Implement Inspector Forms for Advanced Nodes
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - 提供 `transform/branch/subflow` 的普通模式编辑 UI
- Files / Modules:
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
- Write Set:
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
- Acceptance:
  - 表单可编辑关键字段
  - `foreach` 继续显示 JSON-only 提示
- Test Points:
  - inspector 渲染
  - disabled / visible mode gating
- Rollback:
  - 回退 inspector 文件

##### `WIN-ORCH-TEST-2` - Regression Tests for Advanced Node Forms
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - 覆盖高级节点表单化后的回归风险
- Files / Modules:
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - 必要时：`frontend/src/windows/FlowEditorWindow.test.ts`
- Write Set:
  - 上述测试文件
- Acceptance:
  - 新增测试可以稳定捕获 form/json 互转和 UI 回归
- Test Points:
  - `vitest`
- Rollback:
  - 回退新增测试

#### Dependencies
- `WIN-ORCH-RT-4` 是 `WIN-ORCH-RT-5` / `WIN-ORCH-TEST-2` 的前置
- `WIN-ORCH-DOC-2` 在 Stage 4 前必须同步完成

#### Risks and Notes
- 本轮不把 `foreach` 拉进普通模式，避免计划外膨胀
- `transform` 仅做顶层表达式表单，不承诺完整递归 builder
- 当前 worktree 仍存在上一轮未归并的代码改动，本轮必须在这些改动之上继续演进，不能回退

#### Parallelism Assessment
- 评估结果：本轮不并行
- 原因：
  - `flow.ts` 和 `FlowNodeInspector.vue` 共享高级节点字段模型，写集强耦合
  - 用户未授权使用子 Agent

#### Issue List
- 无

阻塞：否
进入 3.2

### Stage 3.2 - Execution
#### Completed Tasks
- `WIN-ORCH-RT-4`
  - 已补齐 `frontend/src/stores/flow.ts` 的高级节点草稿字段、clone/create/load path、form/json 双向互转和普通模式校验。
  - `transform` 已支持顶层 `literal/source/op/object/array` 模式。
  - `branch` 已支持 case 列表、match source/op/value、`default_case`，并在保存时校验 case 名称唯一和默认分支有效性。
  - `subflow` 已支持 `flow_id`、`input_template`、`inputs`、`result_node_id`，并对 `flow_id` 做 UUID 校验。
  - `foreach` 继续维持 JSON-only；切回 form 会显式失败。
- `WIN-ORCH-RT-5`
  - 已扩展 `frontend/src/components/flow/editor/FlowNodeInspector.vue`，为 `transform`、`branch`、`subflow` 提供普通模式表单。
  - 已复用现有 bindings 编辑骨架到 `subflow`，并把 `branch edge.case` 继续留在边侧栏，不回灌进节点表单。
- `WIN-ORCH-DOC-2`
  - 已更新本地 requirements/spec，移除 `transform/branch/subflow` 过时的 JSON-only 表述，并明确 `foreach` 仍为 JSON-only。
- `WIN-ORCH-TEST-2`
  - 已更新 `frontend/src/stores/flow.test.ts` 和 `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`。
  - 已新增高级节点 form/json round-trip、unsupported JSON 回 form 失败、`foreach` JSON-only 和 inspector 渲染分支覆盖。

#### Validation
- 定向前端测试：
  - `npx vitest run src/stores/flow.test.ts src/components/flow/editor/FlowNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：通过，`35 passed`
- 分阶段测试：
  - `npx vitest run src/stores/flow.test.ts`
  - `npx vitest run src/components/flow/editor/FlowNodeInspector.test.ts`
  - 结果：均通过

#### Issue List
- 基线问题仍未解除：
  - 全量 `npm test` 和 `npm run build` 仍受既有 `wailsjs` 缺失影响，本轮不扩散修复

### Stage 3.3 - Code Review
#### Review Checklist
- 需求覆盖：通过
  - `transform`、`branch`、`subflow` 的普通模式 authoring 与 `foreach` JSON-only 边界均已落地。
- 架构合理性：通过
  - `flow.ts` 继续作为 round-trip 真相源；未引入新的持久化模型，也未把 `edge.case` 从边错误挪回节点。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅本地状态扩展，无新增远程 I/O；输入绑定与高级 spec 校验仍集中在保存路径。
- 可读性与一致性：通过
  - 高级节点字段命名、表单分支和错误提示与已有 `call/compose/set_var` 结构保持一致。
- 可扩展性与配置化：通过
  - `FlowSourceDraft` 与复用的 bindings 骨架为后续更深层 source/editor 扩展留出接口。
- 稳定性与安全：通过
  - 普通模式只接受受支持 spec 子集；超出覆盖范围时显式失败，不静默吞字段。
- 测试覆盖情况：通过
  - store / inspector / window 级定向回归均已通过。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未使用子 Agent。

#### Result
- Code Review 通过，进入 Stage 4。

### Stage 4 - Archive
#### Outputs
- `docs/change/2026-04-03_win-flow-advanced-node-forms.md`
- `docs/change/README.md`

#### Lessons Decision
- `Lessons impact: none`
- 原因：
  - 本轮主要是既有 flow editor 的能力补齐，相关边界已进入 requirements/spec；
  - 未暴露需要单独沉淀为长期排查入口的新型故障模式。

#### Workflow Status
- 本轮高级节点表单化实现、审核和归档均已完成。
- 未执行 workflow end；等待用户决定是否继续下一轮或结束当前 workflow。

## Iteration 3 - Foreach Form Authoring

### Stage 1 - Requirements Analysis
#### Goal
- 在当前已具备高级节点普通模式 authoring 的基础上，继续补齐 `foreach` 的最小安全普通模式。
- 本轮只补 `foreach` 的外层 authoring，不实现 `body` 子图可视化编辑器。

#### Scope
- 必须：
  - `foreach` 支持进入普通模式
  - 普通模式至少覆盖：
    - `source`
    - `required`
    - `body`
    - `result_node_id`
  - `body` 继续以 JSON 文本区形式编辑，不做嵌套子图 UI
  - `foreach` 在 form/json 间可以稳定 round-trip
  - `foreach` 相关 UI、测试和本地 requirements/spec 与行为同步
- 可选：
  - 为 `foreach` 表单补充更清晰的辅助文案和空白默认值
- 不做：
  - 不实现 `foreach.body` 图形编辑器
  - 不引入 `loop_item` / `loop_index` 的专门可视化绑定控件
  - 不修改 Flow runtime / Server / Proto

#### Use Cases
- 用户新增 `foreach` 节点后，可以在普通模式下配置数组来源、required 开关、body JSON 和结果节点 ID。
- 用户读取已有结构简单的 `foreach` 节点时，可以不切到整段高级 JSON 就维护关键外层字段。
- 用户在普通模式和 `Advanced JSON` 之间切换时，`body` 子图内容不丢失。
- 用户遇到超出当前普通模式子集的 `foreach` spec 时，编辑器会显式回退到 `Advanced JSON`。

#### Functional Requirements
- 编辑器必须允许 `foreach` 进入普通模式。
- `foreach.source` 必须复用现有 source draft 契约，支持：
  - `node_result`
  - `trigger`
  - `flow_meta`
  - `run_meta`
  - `flow_var`
- `foreach.source` 仍须遵守现有校验：
  - `node_result.node_id` 必须是祖先
  - `path` 必须是合法 JSON Pointer
  - `flow_var.name` 必须合法
- `foreach.required` 必须映射到正式 spec 的布尔字段。
- `foreach.body` 普通模式必须以 JSON 文本区维护，并要求其解析为 object。
- `foreach.body` 在普通模式保存前至少必须校验：
  - `body.nodes` 是数组
  - `body.edges` 是数组
- `foreach.result_node_id` 必须支持文本编辑并在保存时要求非空。
- 当已有 `foreach` spec 含当前普通模式无法表达的额外顶层字段时，切回普通模式必须显式失败。

#### Non-functional Requirements
- 继续保持最小安全改动，优先复用现有 `FlowSourceDraft`、source helper 和普通模式/高级模式切换机制。
- `body` 子图继续以 JSON 真相源保存，不引入第二套嵌套图状态模型。
- 校验继续集中在 store，不把协议判断散落到模板层。

#### Inputs / Outputs
- 输入：
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\requirements\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\specs\flow-editor-visual-form.md`
- 输出：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - 本地 requirements/spec 更新

#### Edge Cases
- `foreach.source.kind=node_result` 但引用的节点不是祖先。
- `foreach.body` 不是 object，或缺失 `nodes/edges` 数组。
- `foreach.result_node_id` 为空。
- 已有 `foreach` spec 含未覆盖的顶层字段，错误切回普通模式后被静默吞掉。
- `flow_var` 来源未填写合法变量名。

#### Acceptance Criteria
- 编辑器可在普通模式下创建、读取、编辑并保存 `foreach` 节点的外层字段。
- `foreach.body` 子图内容在 form/json round-trip 中保持不丢失。
- `foreach` 普通模式失败时会显式回退到 `Advanced JSON`，不静默构造伪表单。
- 本轮新增路径具备 store 和 inspector 回归测试覆盖。

#### Risks
- 若错误放宽 `body` 结构校验，可能在保存时破坏嵌套 graph。
- `foreach` 从 JSON-only 变为普通模式后，form/json 切换逻辑很容易遗漏旧测试断言。
- 本地 requirements/spec 必须同步改写，否则会与实现边界漂移。

### Stage 2 - Architecture Design
#### Overall Solution
- 扩展 `FlowNodeDraft`，为 `foreach` 增加独立普通模式字段。
- 复用 `buildSourceSpec(...)` / `parseSourceDraft(...)` 做 `source` 互转和校验。
- `body` 继续保留为 JSON 文本区，由 store 负责解析与最小结构校验。
- `foreach` 的 form/json mode gating 改为“支持受限普通模式子集”，而不是始终 JSON-only。

#### Alternatives Considered
- 继续让 `foreach` 保持 JSON-only：
  - 不选；这是当前 flow 编辑器剩余最明显的高级节点空洞。
- 直接实现嵌套 `body` 图编辑器：
  - 不选；范围过大，也会引入新的嵌套状态模型。
- 在普通模式下把 `body` 拆成大量字段：
  - 不选；协议真实形态是子图，拆散后更容易丢语义。

#### Module Responsibilities
- `frontend/src/stores/flow.ts`
  - 新增 `foreach` 草稿字段
  - `foreach` form/json 互转与严格保存校验
  - `supportsFormMode(kind)` / `setNodeSpecEditorMode(...)` / `buildFormSpec(...)` 扩展
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 新增 `foreach` 普通模式表单
  - 复用 source-kind/source-path 的输入块
- tests
  - 覆盖 `foreach` round-trip、失败路径和 inspector 渲染
- docs
  - 更新 requirements/spec，改为 `foreach` 支持“部分普通模式”

#### Data / Call Flow
1. `parseSpecDraft(...)` 读取 `foreach` spec 并映射到 `foreachSource`、`foreachRequired`、`foreachBodyJson`、`foreachResultNodeId`。
2. 处于普通模式时，`buildFormSpec(...)` 根据上述字段构造正式 `foreach` spec。
3. 切到 `Advanced JSON` 时，`buildAdvancedSpec(...)` 继续保存完整对象，并保留 `_ui`。
4. 从 JSON 切回普通模式时，仅当 `foreach` spec 落在受支持顶层子集内才允许恢复。

#### Interface Drafts
- `FlowNodeDraft` 新增：
  - `foreachSource`
  - `foreachRequired`
  - `foreachBodyJson`
  - `foreachResultNodeId`
- 新增 helper：
  - `parseForeachDraft(...)`
  - `parseForeachBodyObject(...)`

#### Error Handling and Safety
- `foreach.body` JSON 非法、不是 object、或缺少 `nodes/edges` 数组时，保存必须显式报错。
- `foreach.result_node_id` 为空时，保存必须显式报错。
- `foreach.source` 继续复用现有 source 校验，不新增宽松路径。
- 超出普通模式覆盖范围的 `foreach` spec 必须显式停留在 JSON 模式。

#### Performance and Testing Strategy
- 不新增远程 I/O；所有变更都在本地状态和序列化层完成。
- 测试重点：
  - `foreach` form/json round-trip
  - `foreach` JSON-only 旧断言替换为 form-mode 可用断言
  - `body` 结构校验失败路径
  - 既有高级节点不回归

#### Extensibility Design Points
- `foreachSource` 沿用现有 `FlowSourceDraft`，后续若要支持 `loop_item` / `loop_index` 可在 source kind 扩展点继续前进。
- `body` 先维持 JSON 文本区，为未来嵌套图编辑器保留单一真实来源。

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 Win 编辑器已经支持 `transform`、`branch`、`subflow` 的普通模式 authoring，但 `foreach` 仍保留 JSON-only。
- 用户要求继续把 flow 编辑器补到更接近完备；本轮优先补 `foreach` 的最小安全普通模式。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: `updated`
- Specs impact: `updated`
- stable truth：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
- workflow results：
  - `docs/change/2026-04-03_win-flow-foreach-form-authoring.md`
- lessons：
  - 暂未新增

#### Related Requirements / Specs / Lessons
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- Related lessons:
  - 无

#### Executable Task List
- [x] `WIN-ORCH-DOC-3` 更新本地 requirements/spec，明确 `foreach` 的部分普通模式边界
- [x] `WIN-ORCH-RT-6` 扩展 `flow.ts` 的 `foreach` 草稿模型、校验和 form/json 互转
- [x] `WIN-ORCH-RT-7` 扩展 inspector，实现 `foreach` 普通模式 authoring
- [x] `WIN-ORCH-TEST-3` 补 `foreach` 表单化回归测试

#### Task Details
##### `WIN-ORCH-DOC-3` - Update Stable Docs for Foreach Form Authoring
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - 让本地 requirements/spec 与 `foreach` 普通模式边界保持一致
- Files / Modules:
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
- Write Set:
  - 上述两份稳定文档
- Acceptance:
  - 文档明确 `foreach` 支持 `source/required/body/result_node_id` 的部分普通模式
  - 文档明确 `body` 仍是 JSON 文本区，不是子图可视化编辑器
- Test Points:
  - 文档表述不与 server stable spec 冲突
- Rollback:
  - 回退上述文档

##### `WIN-ORCH-RT-6` - Extend Store Draft Model for Foreach Form Authoring
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - 让 `foreach` 在普通模式和 JSON 模式之间稳定互转，并保留 `body` 子图
- Files / Modules:
  - `frontend/src/stores/flow.ts`
- Write Set:
  - `frontend/src/stores/flow.ts`
- Acceptance:
  - `foreach` 可进入普通模式
  - `source/required/body/result_node_id` 支持 form/json round-trip
  - 不支持的 `foreach` JSON 仍显式回退 `Advanced JSON`
- Test Points:
  - round-trip
  - validation
  - unsupported JSON fallback
- Rollback:
  - 回退 `frontend/src/stores/flow.ts`

##### `WIN-ORCH-RT-7` - Implement Foreach Inspector Form
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - 提供 `foreach` 的普通模式编辑 UI
- Files / Modules:
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
- Write Set:
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
- Acceptance:
  - 表单可编辑 `source`、`required`、`body` JSON、`result_node_id`
  - UI 明确 `body` 仍需以 JSON 维护
- Test Points:
  - inspector 渲染
  - form mode 可用性
- Rollback:
  - 回退 inspector 文件

##### `WIN-ORCH-TEST-3` - Regression Tests for Foreach Form Authoring
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\plan.md`
- Goal:
  - 覆盖 `foreach` 表单化带来的回归风险
- Files / Modules:
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - 必要时：`frontend/src/windows/FlowEditorWindow.test.ts`
- Write Set:
  - 上述测试文件
- Acceptance:
  - 新增测试可稳定捕获 `foreach` form/json 互转和 UI 回归
- Test Points:
  - `vitest`
- Rollback:
  - 回退新增测试

#### Dependencies
- `WIN-ORCH-RT-6` 是 `WIN-ORCH-RT-7` / `WIN-ORCH-TEST-3` 的前置
- `WIN-ORCH-DOC-3` 在 Stage 4 前必须同步完成

#### Risks and Notes
- `body` 继续作为 JSON 文本维护，避免计划外扩展到嵌套图编辑器
- 当前 worktree 已有未归并改动，本轮必须在现有状态上继续前进，不能回退无关文件

#### Parallelism Assessment
- 评估结果：本轮不并行
- 原因：
  - `flow.ts` 与 `FlowNodeInspector.vue` 共享 `foreach` 字段模型和 mode gating，写集强耦合
  - 用户未授权使用子 Agent

#### Issue List
- 无

阻塞：否
进入 3.2

### Stage 3.2 - Execution
#### Completed Tasks
- `WIN-ORCH-RT-6`
  - 已扩展 `frontend/src/stores/flow.ts` 的 `FlowNodeDraft`，新增 `foreachSource`、`foreachRequired`、`foreachBodyJson`、`foreachResultNodeId`。
  - 已补齐 `foreach` 的 parse / build / mode gating 路径，使其可在受支持子集内 form/json round-trip。
  - 已新增 `foreach` 表单保存校验：`source`、`body`、`result_node_id` 都会在 store 中显式校验。
- `WIN-ORCH-RT-7`
  - 已扩展 `frontend/src/components/flow/editor/FlowNodeInspector.vue`，新增 `foreach` 普通模式表单。
  - UI 当前覆盖 `source`、`required`、`result_node_id` 和 `body` JSON 文本区，并明确 `body` 仍非可视化子图编辑器。
- `WIN-ORCH-DOC-3`
  - 已更新本地 requirements/spec，把 `foreach` 从 JSON-only 改为“外层字段部分普通模式 + body 继续 JSON 真相源”。
- `WIN-ORCH-TEST-3`
  - 已更新 `frontend/src/stores/flow.test.ts` 和 `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`。
  - 已把旧的 `foreach` JSON-only 断言替换为 form/json round-trip、unsupported JSON fallback 和 inspector 渲染覆盖。

#### Validation
- 定向前端测试：
  - `npx vitest run src/stores/flow.test.ts src/components/flow/editor/FlowNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：通过，`3 passed / 36 passed`

#### Issue List
- 基线问题仍未解除：
  - 本仓库全量 `npm test` / `npm run build` 仍受既有 `wailsjs` 生成物缺失影响。
  - 本轮未扩散修复该基线，只执行与 `foreach` authoring 相关的定向前端测试。

### Stage 3.3 - Code Review
#### Review Checklist
- 需求覆盖：通过
  - `foreach` 已支持外层字段普通模式 authoring，`body` 仍保留 JSON 真相源。
- 架构合理性：通过
  - 继续复用 `FlowSourceDraft` 和现有 mode gating；未引入第二套 `body` 子图状态模型。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅本地状态和序列化层扩展，无新增远程 I/O。
- 可读性与一致性：通过
  - `foreach` 字段命名、校验风格和 inspector 分支与既有高级节点实现保持一致。
- 可扩展性与配置化：通过
  - `body` 仍作为 JSON 真相源，为未来嵌套图编辑器保留清晰扩展点。
- 稳定性与安全：通过
  - 超出受支持子集的 `foreach` spec 会显式回退 `Advanced JSON`，不会静默吞字段。
- 测试覆盖情况：通过
  - store / inspector / window 级定向回归均已通过。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未使用子 Agent。

#### Result
- Code Review 通过，进入 Stage 4。

### Stage 4 - Archive
#### Outputs
- `docs/change/2026-04-03_win-flow-foreach-form-authoring.md`
- `docs/change/README.md`

#### Lessons Decision
- `Lessons impact: none`
- 原因：
  - 本轮主要是 editor authoring 边界继续补齐，相关行为已经进入 requirements/spec；
  - 未新增需要长期复用的独立故障排查规则。

#### Workflow Status
- 本轮 `foreach` 普通模式 authoring、审核和归档均已完成。
- 未执行 workflow end；等待用户决定是否继续下一轮或结束当前 workflow。

## Iteration 4 - Foreach Body Visual Editor

### Stage 1 - Requirements Analysis
#### Goal
- 在 `foreach` 已支持外层字段普通模式的基础上，补齐 `foreach.body` 的显式可视化子图 authoring。
- 本轮重点是 body 内 DAG 的可视化编辑以及与根 graph 保存/恢复语义的正确衔接。

#### Scope
- 必须：
  - `foreach` inspector 提供显式 body 编辑入口
  - body 编辑会话支持节点/边的新增、选中、移动、连接、删除
  - body 编辑会话支持节点级最小 JSON-first spec 编辑
  - body 会话中的每次提交都同步回父节点 `foreachBodyJson`
  - 根 graph 的保存、脏状态、本地恢复草稿继续正确覆盖 body 编辑中的未保存改动
- 可选：
  - body 会话增加更清晰的 breadcrumb / 返回主图文案
- 不做：
  - 不为 body 内 `call` 节点补方法选择器、字段级 binding 对话框和完整 visual form
  - 不引入第二套持久化 graph 模型
  - 不修改 Flow runtime / Server / Proto

#### Use Cases
- 用户选中 `foreach` 节点后，可进入 body 编辑会话，用画布而不是整段 JSON 维护内部 DAG。
- 用户在 body 会话中移动节点、增删边或修改内部节点 JSON 后，直接保存项目即可把变更写回父 `foreach.body`。
- 用户在 body 会话中尚未退出时触发保存、恢复草稿或撤销重做，系统仍不会丢失内层改动。

#### Functional Requirements
- 编辑器必须允许用户从 `foreach` inspector 进入和退出 body 编辑会话。
- body 编辑会话必须以 `nodes/edges` 画布形式展示 `foreach.body`。
- body 编辑会话中的每次图结构提交都必须同步回父节点 `foreachBodyJson`。
- body 编辑会话必须支持内部节点的最小 JSON-first spec 编辑，不要求完整表单 parity。
- body 编辑会话开启后，保存项目必须仍基于根 graph 导出，并包含当前 body 改动。

#### Non-functional Requirements
- 继续保持最小安全改动，不重写现有 root graph store。
- body 会话不能破坏 root graph 的 dirty、save、recovery baseline 语义。
- body 会话应尽量复用现有 `FlowCanvas`、`FlowEdgeInspector` 和 Add Node 对话框。

#### Inputs / Outputs
- 输入：
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\requirements\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- 输出：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 必要的 store helper 和测试更新
  - 本地 requirements/spec 更新

#### Edge Cases
- `foreach.body` JSON 非法，无法进入 body 编辑会话。
- body 会话中未退出就保存项目，body 改动仍必须随根 graph 一并保存。
- body 会话中撤销重做导致父节点 `foreachBodyJson` 回滚时，画布必须同步。
- body 会话期间若父 `foreach` 节点被删除或不再可用，编辑器必须显式退出该会话。

#### Acceptance Criteria
- 用户可以通过显式 body 编辑会话可视化编辑 `foreach.body` 的内部 DAG。
- body 会话中的改动会稳定回写到父节点 `foreachBodyJson`。
- 项目保存、dirty 判断和恢复草稿在 body 会话开启时仍保持正确。
- 本轮新增路径具备 window / store / inspector 级定向测试覆盖。

### Stage 2 - Architecture Design
#### Overall Solution
- 保持 root `flowStore` 作为唯一持久化真相源，不把 body 图切换进全局 store。
- 在 `FlowEditorWindow.vue` 引入显式 body 编辑会话，维护一个仅用于 UI 的本地 body graph draft。
- 每次 body 会话提交时，把本地 draft 序列化回父 `foreach` 节点的 `foreachBodyJson`，并写入 root store history。

#### Alternatives Considered
- 直接把全局 `flowStore.state.nodes/edges` 切换到 body 图：
  - 不选；会污染 root graph 的保存、恢复草稿和 baseline 语义。
- 继续只保留 `body` JSON 文本区：
  - 不选；这是当前 `foreach` authoring 最明显的剩余空洞。

#### Module Responsibilities
- `frontend/src/stores/flow.ts`
  - 提供 body 会话所需的 graph draft parse / loose serialize helper
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 提供进入 body 编辑会话的显式入口
- `frontend/src/windows/FlowEditorWindow.vue`
  - 负责 body 会话状态、画布切换、回写父节点 JSON，以及保存/撤销/恢复联动
- tests
  - 覆盖 body 会话进入、回写和保存语义

#### Data / Call Flow
1. 用户在 `foreach` inspector 中点击 body 编辑入口。
2. Window 从父节点 `foreachBodyJson` 解析出本地 body graph draft，并切换画布到 body 会话。
3. 用户对 body 图的每次提交都会更新本地 draft，并同步回父节点 `foreachBodyJson`。
4. 根 graph 的 `saveProject()`、dirty 和 recovery 继续只读取 root store，但由于父节点 JSON 已同步，因此会自然带上 body 改动。

#### Error Handling and Safety
- `foreach.body` 非法 JSON 或缺失 `nodes/edges` 数组时，进入 body 会话必须显式失败。
- body 会话同步回父节点 JSON 失败时，必须显式报错，不静默吞掉。
- 若父节点在会话期间消失或不再是 `foreach`，会话必须主动退出。

#### Performance and Testing Strategy
- 不新增远程 I/O；所有 body 会话行为都在本地状态和序列化层完成。
- 测试重点：
  - body 会话进入与退出
  - body 改动同步回父节点
  - body 会话开启时保存项目仍导出正确根 graph

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 Win 编辑器已经支持 `foreach` 的外层字段普通模式，但 `body` 仍是 JSON 文本区。
- 用户要求继续推进 flow 完备性；本轮目标是把 `foreach.body` 补到“可视化子图 authoring”，同时不破坏 root graph 的保存和恢复契约。

#### Docs Governance Routing Decision
- Requirements impact: `updated`
- Specs impact: `updated`
- stable truth：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
- workflow results：
  - `docs/change/2026-04-03_win-flow-foreach-body-visual-editor.md`
- lessons：
  - 暂未新增

#### Executable Task List
- [x] `WIN-ORCH-DOC-4` 更新本地 requirements/spec，明确 body 编辑会话边界
- [x] `WIN-ORCH-RT-8` 为 body 会话补 graph parse / serialize helper
- [x] `WIN-ORCH-RT-9` 在 window / inspector 中接入 `foreach.body` 可视化编辑会话
- [x] `WIN-ORCH-TEST-4` 补 body 会话与保存语义回归测试

#### Dependencies
- `WIN-ORCH-DOC-4` 在 Stage 4 前必须完成
- `WIN-ORCH-RT-8` 是 `WIN-ORCH-RT-9` / `WIN-ORCH-TEST-4` 的前置

#### Risks and Notes
- 当前实现必须避免把 body 图直接替换成全局 root graph editor state。
- body 会话只做 UI 投影，不能成为新的 persisted source of truth。

#### Parallelism Assessment
- 评估结果：本轮不并行
- 原因：
  - `FlowEditorWindow.vue`、`FlowNodeInspector.vue` 和 `flow.ts` 的写集强耦合
  - 用户未授权使用子 Agent

#### Issue List
- 无

阻塞：否
进入 3.2

### Stage 3.2 - Execution
#### Completed Tasks
- `WIN-ORCH-DOC-4`
  - 已更新本地 requirements/spec，把 `foreach.body` 从“仅 JSON 文本区”提升为“显式 body 编辑会话 + JSON 真相源”。
  - 已明确当前边界：body 内节点先走 JSON-first inspector，不扩到顶层 `call` visual form parity。
- `WIN-ORCH-RT-8`
  - 已扩展 `frontend/src/stores/flow.ts`，新增 body 会话需要的 graph draft parse / loose serialize helper：
    - `createGraphEditorStateFromDraft(...)`
    - `exportLooseGraphDraftFromEditorState(...)`
  - helper 保持为最小增量，不改现有 root editor store 行为。
- `WIN-ORCH-RT-9`
  - 已扩展 `frontend/src/components/flow/editor/FlowNodeInspector.vue`，新增 `Open Visual Body Editor` 入口。
  - 已新增 `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`，提供 body 内节点的最小 JSON-first inspector。
  - 已扩展 `frontend/src/windows/FlowEditorWindow.vue`，新增显式 body 编辑会话：
    - body 画布切换
    - body 节点/边新增、移动、连接、删除
    - body 改动回写父 `foreachBodyJson`
    - body 会话与 save / undo / redo / recovery draft 联动
- `WIN-ORCH-TEST-4`
  - 已更新 `frontend/src/stores/flow.test.ts`
  - 已更新 `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - 已更新 `frontend/src/windows/FlowEditorWindow.test.ts`
  - 新增覆盖：
    - body 会话入口渲染
    - body 会话保存回写根 graph
    - body graph parse / serialize helper

#### Validation
- 定向前端测试：
  - `npx vitest run src/stores/flow.test.ts src/components/flow/editor/FlowNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：通过，`39 passed`

#### Issue List
- 基线问题仍未解除：
  - 全量 `npm test` / `npm run build` 仍受仓库既有 `wailsjs` 生成物缺失影响。
  - 本轮未扩散修复该基线，仅执行与 flow editor 改动直接相关的定向前端测试。

### Stage 3.3 - Code Review
#### Review Checklist
- 需求覆盖：通过
  - `foreach.body` 已具备显式可视化子图 authoring 入口和最小 JSON-first 节点编辑。
- 架构合理性：通过
  - body 会话保持为 window 层 UI 投影，root `flowStore` 仍是唯一持久化真相源。
- 保存 / 恢复安全性：通过
  - body 会话会把有效改动同步回父 `foreachBodyJson`，保存前和撤销重做前都会执行显式同步。
- 可维护性：通过
  - store 只新增 parse / serialize helper，未重写现有 root graph editor 状态机。
- 测试覆盖情况：通过
  - store / inspector / window 定向回归已覆盖 body 会话主路径。
- 子Agent治理与审计：通过
  - 本轮未使用子 Agent。

#### Result
- Code Review 通过，进入 Stage 4。

### Stage 4 - Archive
#### Outputs
- `docs/change/2026-04-03_win-flow-foreach-body-visual-editor.md`
- `docs/change/README.md`

#### Lessons Decision
- `Lessons impact: none`
- 原因：
  - 本轮核心边界已经写入 requirements/spec；
  - 暂未出现需要单独沉淀到 `docs/lessons` 的长期排查模式。

#### Workflow Status
- 本轮 `foreach.body` 可视化子图 authoring、审核和归档均已完成。
- 未执行 workflow end；等待用户决定是否继续下一轮或结束当前 workflow。

## Iteration 5 - Foreach Body Call Authoring

### Stage 1 - Requirements Analysis
#### Goal
- 在 `foreach.body` 已具备可视化子图编辑的基础上，补齐 body 内 `call` 节点的普通模式 authoring。
- 本轮重点是把 body 会话里的 `call` 节点补到“可选方法、可编辑 literal、可做字段 binding”，但不扩展到其它 body 节点。

#### Scope
- 必须：
  - body 会话中的 `call` 节点支持方法选择器
  - body 会话中的 `call` 节点支持 schema-driven visual form
  - body 会话中的 `call` 字段支持 binding 对话框和 `flow_var`
  - body 会话中的 `node_result` binding 仅允许引用当前 body 子图祖先节点
  - body 会话内 `call` 变更继续同步回父节点 `foreachBodyJson`
- 可选：
  - 复用现有 root dialog 与 field-draft 状态，而不是再造一套 dialog 组件
- 不做：
  - 不把 body 内 `compose/transform/set_var/branch/foreach/subflow` 升级为完整普通模式
  - 不修改 Flow runtime / Server / Proto

#### Use Cases
- 用户在 body 会话里选中 `call` 节点后，可以直接打开 capability 方法选择器。
- 用户在 body 会话里为 `call` 字段填写 literal，保存项目后结果写回父 `foreach.body`。
- 用户在 body 会话里把字段绑定到 body 子图中的祖先节点结果、trigger、meta 或 flow local var。

#### Functional Requirements
- body 会话中的 `call` 节点必须支持 `Form` / `Advanced JSON` 双模式切换。
- body 会话中的 `call` visual form 必须复用现有 schema 解析与 compatibility 判定规则。
- body 会话中的 `call` binding 对话框必须继续支持：
  - `node_result`
  - `trigger`
  - `flow_meta`
  - `run_meta`
  - `flow_var`
- body 会话中的 `node_result` binding 祖先判定必须仅基于当前 body graph，而不是根 graph。
- body 会话中非 `call` 节点继续维持 JSON-first inspector。

#### Non-functional Requirements
- 保持最小安全增量，避免把 body 图接入根 `flowStore.state.nodes`。
- 优先复用既有 `FlowMethodPickerDialog`、`FlowFieldBindingDialog`、schema resolver 和 visual-form helper。
- 不得破坏 body 会话与 root graph 的保存、恢复草稿和 dirty 语义。

#### Inputs / Outputs
- 输入：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
- 输出：
  - body 会话内 `call` authoring 代码改动
  - 本地 requirements/spec 更新
  - 本 worktree 根 `plan.md`

#### Edge Cases
- body `call` 节点方法已存在，但 capability 尚未加载。
- body `call` 节点切到新方法后，旧 `args_template` / `inputs` 里存在新 schema 不支持的字段。
- body `call` 节点 binding 指向根图祖先节点时，必须显式拒绝。

#### Acceptance Criteria
- body 会话中的 `call` 节点可打开方法选择器并写回 `method + target`。
- body 会话中的 `call` 节点可在 ordinary mode 编辑 literal 字段和 binding。
- body 会话中的 `node_result` binding 只允许选择 body 祖先节点。
- 定向前端测试覆盖 body `call` authoring 主路径。

### Stage 2 - Architecture Design
#### Overall Solution
- 继续让 `FlowEditorWindow.vue` 维护 body 会话的本地 graph draft，不改 root store 真相源。
- body `call` visual-form 直接在 window 层根据 body 节点草稿和已加载 capability 计算，不把 body 节点塞回 root store。
- `FlowBodyNodeInspector.vue` 只对 `kind=call` 增量复用 root call authoring 交互；其余 kind 保持 JSON-first。

#### Alternatives Considered
- 直接把 body 节点挂到 root `flowStore` 做复用：
  - 不选；会破坏 body 会话与根图的边界和 ancestor 判定。
- 为 body 再造一套方法 / binding dialog：
  - 不选；现有 dialog 已可复用，没必要扩写第二套。

#### Module Responsibilities
- `frontend/src/windows/FlowEditorWindow.vue`
  - 计算 body `call` visual form
  - 处理 body 节点方法应用、literal 提交、binding 校验与提交
  - 维护 body 祖先节点选项
- `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - 在 `kind=call` 时渲染 ordinary mode / JSON mode 和字段 authoring UI
- tests
  - 覆盖 body `call` inspector 渲染和 window 级主路径

#### Data / Call Flow
1. 用户在 body 会话中选中一个 `call` 节点。
2. Window 根据 body 节点 `method + target + execCapabilities` 解析 visual schema，并构造 visual form。
3. 用户通过方法对话框或字段交互修改 body `call` 节点。
4. 每次提交都写回 body 节点 draft，并同步回父 `foreachBodyJson`。

#### Error Handling and Safety
- body `call` 字段 binding 若引用非祖先 body 节点，必须显式失败。
- body `call` 切回 ordinary mode 时，若当前 spec 超出 visual form 表达范围，必须保持在 `Advanced JSON`。
- capability 缺失时，body `call` 只能显示兼容性降级提示，不能伪造表单。

#### Performance and Testing Strategy
- 不新增远程 I/O；body 能力加载继续复用现有 capability query/hydration 机制。
- 测试重点：
  - body `call` inspector 表单渲染
  - body 会话里打开方法选择器
  - body 子图保存仍通过父 `foreach.body` 回写

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 Win 编辑器已经支持 `foreach.body` 的可视化子图 authoring，但 body 内节点仍以 JSON-first 为主。
- 下一步的最小缺口是 body 内 `call` 节点还没有 root graph 同级的 capability 选择与字段 authoring。

#### Docs Governance Routing Decision
- Requirements impact: `updated`
- Specs impact: `updated`
- stable truth：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
- workflow results：
  - `docs/change/2026-04-03_win-flow-foreach-body-call-authoring.md`
- lessons：
  - 暂未新增

#### Related Requirements / Specs / Lessons
- Related requirements:
  - `docs/requirements/flow-editor-visual-form.md`
- Related specs:
  - `docs/specs/flow-editor-visual-form.md`
- Related lessons:
  - 无

#### Executable Task List
- [x] `WIN-ORCH-DOC-5` 更新本地 requirements/spec，明确 body 内 `call` authoring 边界
- [x] `WIN-ORCH-RT-10` 在 window 层补 body `call` 的 visual-form / method / binding 逻辑
- [x] `WIN-ORCH-RT-11` 扩展 `FlowBodyNodeInspector.vue`，为 body `call` 渲染普通模式 UI
- [x] `WIN-ORCH-TEST-5` 补 body `call` inspector / window 回归测试

#### Task Details
##### `WIN-ORCH-DOC-5` - Update Stable Docs for Body Call Authoring
- Owner: main agent
- Goal:
  - 让 requirements/spec 与 body `call` authoring 目标一致

##### `WIN-ORCH-RT-10` - Implement Body Call Authoring Logic in Window
- Owner: main agent
- Goal:
  - 让 body 会话中的 `call` 节点具备方法选择、visual form 和 binding 提交
- Files / Modules:
  - `frontend/src/windows/FlowEditorWindow.vue`

##### `WIN-ORCH-RT-11` - Add Body Call Form UI
- Owner: main agent
- Goal:
  - 在 body inspector 中仅为 `call` 节点补普通模式 authoring UI
- Files / Modules:
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`

##### `WIN-ORCH-TEST-5` - Regression Tests for Body Call Authoring
- Owner: main agent
- Goal:
  - 覆盖 body `call` authoring 主路径回归风险
- Files / Modules:
  - `frontend/src/windows/FlowEditorWindow.test.ts`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts`

#### Dependencies
- `WIN-ORCH-DOC-5` 必须先完成
- `WIN-ORCH-RT-10` 是 `WIN-ORCH-RT-11` / `WIN-ORCH-TEST-5` 的前置

#### Risks and Notes
- body `call` 祖先判定若误复用根图，会放开非法 binding。
- body 会话当前是显式 UI 投影；本轮不能把它重构成第二套持久化 graph store。

#### Parallelism Assessment
- 评估结果：本轮不并行
- 原因：
  - `FlowEditorWindow.vue` 与 `FlowBodyNodeInspector.vue` 的写集强耦合
  - 用户未授权使用子 Agent

#### Issue List
- 无

阻塞：否
进入 3.2

### Stage 3.2 - Execution
#### Completed Tasks
- `WIN-ORCH-RT-10`
  - 已扩展 `frontend/src/windows/FlowEditorWindow.vue`，新增 body `call` visual-form 计算、spec mode 切换、字段 literal 提交、binding 校验和 capability 应用逻辑。
  - 已新增 body 子图祖先判定，确保 `node_result` 只允许引用当前 body graph 的祖先节点。
  - 已补一个通用的 `ensureCapabilityRouteLoaded(...)` store 入口，供 body 会话静默补全 capability schema。
- `WIN-ORCH-RT-11`
  - 已扩展 `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`。
  - body `call` 现在支持：
    - `Form` / `Advanced JSON` 切换
    - 方法选择入口
    - schema-driven literal 字段
    - 字段级 binding 对话框入口
  - body 内非 `call` 节点继续保持 JSON-first inspector。
- `WIN-ORCH-DOC-5`
  - 已更新本地 requirements/spec，把 body `call` authoring 边界纳入稳定文档。
- `WIN-ORCH-TEST-5`
  - 已新增 `frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts`
  - 已更新 `frontend/src/windows/FlowEditorWindow.test.ts`
  - 覆盖 body `call` ordinary mode 渲染、body inspector 事件、body 会话里打开方法选择器等主路径。

#### Validation
- 定向前端测试：
  - `npx vitest run src/components/flow/editor/FlowBodyNodeInspector.test.ts src/components/flow/editor/FlowNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts src/stores/flow.test.ts`
  - 结果：通过，`42 passed`

#### Issue List
- 基线问题状态未变：
  - 全量 `npm test` / `npm run build` 仍受仓库既有 `wailsjs` 生成物缺失影响。
  - 本轮未扩散处理该基线，继续以 flow editor 相关定向测试为准。

### Stage 3.3 - Code Review
#### Review Checklist
- 需求覆盖：通过
  - body `call` 已具备方法选择、ordinary mode 字段编辑和 binding 对话框。
- 架构合理性：通过
  - body 会话仍是 window 层本地 UI 投影，未把 body 图并入根 store 持久化真相源。
- 安全性：通过
  - body `node_result` binding 已限制到 body 子图祖先节点。
- 可维护性：通过
  - 现有 `FlowMethodPickerDialog`、`FlowFieldBindingDialog` 和 visual-form helper 均被复用，没有再造第二套 dialog。
- 测试覆盖情况：通过
  - body inspector 与 window 集成主路径已补定向回归测试。
- 子Agent治理与审计：通过
  - 本轮未使用子 Agent。

#### Result
- Code Review 通过，进入 Stage 4。

### Stage 4 - Archive
#### Outputs
- `docs/change/2026-04-03_win-flow-foreach-body-call-authoring.md`
- `docs/change/README.md`

#### Lessons Decision
- `Lessons impact: none`
- 原因：
  - 本轮主要是把既有 body editor 能力补到 body `call` ordinary mode parity；
  - 相关边界已经进入 requirements/spec，无新增需要独立沉淀的长期排查规则。

#### Workflow Status
- 本轮 body `call` authoring、审核和归档均已完成。
- 未执行 workflow end；等待用户决定是否继续下一轮或结束当前 workflow。

## Workflow End
- 用户已确认结束 workflow。
- 主仓控制面已执行：
  - `main <- feat/win-orchestrator-editor` fast-forward 合并
  - worktree 中的 `plan.md`、`docs/change/*` 和稳定文档改动已进入主仓
- 当前主仓仍保留用户自己的未提交状态：
  - `go.mod`
  - `myflowhub-mcp.exe`
- 上述用户本地改动未被本次 workflow 清理、回退或纳入提交。
