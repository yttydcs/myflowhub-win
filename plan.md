# Plan - Win Foreach Body Advanced Authoring

## Workflow Information
- Repo: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps`
- Branch: `feat/win-orchestrator-form-gaps`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`:
  - `docs/README.md` 已读取，文档读取顺序保持 `requirements -> specs -> plan -> change -> lessons`
  - `$m-autoflow` 初始化规则已确认：实现只能发生在 `D:\project\MyFlowHub3\worktrees\` 下的独立 worktree
- base/worktree confirmation:
  - 主控制仓：`D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - 主控制仓当前分支：`main`
  - 主控制仓存在用户改动：`go.mod`、`myflowhub-mcp.exe`
  - 当前执行 worktree：`D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps`
  - 当前执行分支：`feat/win-orchestrator-form-gaps`
  - 本轮只改 `MyFlowHub-Win`，不把 `MyFlowHub-Server` 纳入实现 repo

### Stage 1 - Requirements Analysis
#### Goal
- 补齐 `foreach.body` 内非 `call` 节点的普通模式 authoring，使 body editor 不再只有 `call` 具备表单能力。
- 保持 `foreach.body` 继续以父节点 `foreachBodyJson` 为单一真相源，不引入第二套持久化 graph。

#### Scope
- 必须：
  - body editor 内的 `compose`、`transform`、`set_var`、`branch`、`foreach`、`subflow` 节点允许进入与根图一致的最小普通模式
  - body editor 内的上述节点继续遵守现有 form/json mode gating
  - body editor 内需要支持这些节点普通模式所需的 bindings、source 选择和 ancestor 选择
  - requirements/specs 更新为当前实现边界
- 可选：
  - 若发现轻微重复逻辑阻碍实现，可做最小 UI 抽取，但不做大规模组件重构
- 不做：
  - 不修改 Flow runtime、Server 协议或 DAG 校验规则
  - 不实现 `transform` 递归表达式树可视化编辑器
  - 不新增 `cron`、`subflow`、`foreach` 等协议能力
  - 不改变 `foreach.body` 的 JSON 真相源约束

#### Use Cases
- 用户在 `foreach.body` 里选中 `transform` 节点时，可以直接切换表达式模式并编辑 source / op / literal 等字段，而不是手写整段 spec JSON。
- 用户在 `foreach.body` 里选中 `branch` 节点时，可以直接维护 cases 和 `default_case`。
- 用户在 `foreach.body` 里选中 `set_var`、`compose`、`subflow` 节点时，可以直接维护模板和 bindings。
- 用户在 `foreach.body` 里继续编辑嵌套 `foreach` 的外层字段，但 body 仍保持 JSON 真相源。
- 用户从 body editor 切回保存项目时，内部高级节点的普通模式修改会稳定写回父 `foreachBodyJson`。

#### Functional Requirements
- body editor inspector 必须覆盖 `call / compose / transform / set_var / branch / foreach / subflow`。
- body editor 中 `compose / set_var / subflow` 必须支持 template + inputs authoring。
- body editor 中 `transform` 必须支持与根图一致的顶层表达式模式 authoring。
- body editor 中 `branch` 必须支持与根图一致的 case 列表 authoring。
- body editor 中 `foreach` 必须支持与根图一致的外层字段 authoring，且 `body` 仍通过 JSON 文本区维护。
- body editor 的 bindings 必须支持 `node_result / trigger / flow_meta / run_meta / flow_var`。
- body editor 的 `node_result` 祖先列表必须基于 body graph 祖先关系，而不是根图祖先关系。
- body editor 中 form/json mode 切换的错误处理必须保持显式失败，不能静默降级。

#### Non-functional Requirements
- 变更面尽量限制在 body editor UI 和 body editor wiring，避免无必要修改 store 协议层。
- 不得破坏现有 body `call` visual form、method picker、binding dialog 和 dirty/recovery 机制。
- 继续保持保存、恢复草稿和脏状态以根编辑器状态与父 `foreachBodyJson` 为准。

#### Inputs / Outputs
- 输入：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow.ts`
- 输出：
  - body editor 高级节点表单 authoring 代码
  - 对应测试更新
  - worktree 根 `plan.md`
  - `docs/change/` 归档

#### Edge Cases
- body 节点 advanced spec 含当前普通模式无法表达的额外字段时，切回 form 必须继续显式失败。
- body editor 内 `flow_var.name` 为空、source path 非法或 branch case 重名时，保存阶段必须沿用现有显式错误。
- body 内 `foreach` 仍不能把内部 `body` 错误结构化拆散。
- body 会话期间切换 kind 后，已有模板 / bindings / advanced 字段必须遵守现有最小迁移策略。
- body 节点 bindings 的 ancestor node 列表只能来自 body graph 拓扑。

#### Acceptance Criteria
- body editor 可在普通模式下读取并保存 `compose / transform / set_var / branch / foreach / subflow`。
- body editor 的普通模式交互与根图同类节点保持语义一致。
- body editor 中 bindings / ancestor 选择可用，且不会引用根图无关节点。
- `foreachBodyJson` 仍是唯一持久化真相源，保存项目后 body graph 不丢失。
- 对超出普通模式支持范围的 body 节点，编辑器继续只显示 `Advanced JSON` 或切回 form 显式失败。

#### Risks
- 如果 body editor 误复用根图祖先列表，会导致 `node_result` 引用非法节点。
- 如果 body editor 的 bindings 交互只做 UI 复制、不接入 body 会话提交逻辑，会造成看起来可编辑但不落盘。
- 若过度抽象共享组件，反而会扩大回归面，影响根图 inspector。

#### Issue List
- 无

### Stage 2 - Architecture Design
#### Overall Solution
- 采用“body inspector 增量补齐根图已有高级节点表单”的方案。
- 优先复用现有 store 中的 form/json mode、spec 解析和保存逻辑，只在 `FlowBodyNodeInspector.vue` 暴露缺失字段与事件，并在 `FlowEditorWindow.vue` 把 body 场景接到已有 binding/source 处理函数。

#### Alternatives Considered
- 把 `FlowNodeInspector.vue` 抽成根图/body 全共享大组件：
  - 不选；改动面过大，容易影响根图现有交互和测试。
- 继续保持 body 非 `call` JSON-first：
  - 不选；这正是当前主要完备性缺口。
- 在 body editor 中复制一套独立 store：
  - 不选；违背 `foreachBodyJson` 单一真相源约束。

#### Module Responsibilities
- `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - 增加 body 内高级节点普通模式 UI
  - 增加 body 场景需要的 bindings/source/ancestor 交互
- `frontend/src/windows/FlowEditorWindow.vue`
  - 向 body inspector 传入 ancestor 选项
  - 让 add/remove binding 与 source-kind 切换在 body 会话里也可工作
  - 继续把 body 改动提交回父 `foreachBodyJson`
- `frontend/src/stores/flow.ts`
  - 仅在发现 body 场景确实缺 helper 时做最小补充；默认不改协议与数据模型
- tests
  - `FlowBodyNodeInspector.test.ts`
  - `FlowEditorWindow.test.ts`
  - 如有必要：`flow.ts` 相关测试

#### Data / Call Flow
1. 用户进入 `foreach.body` 可视化编辑会话。
2. 选中 body 节点后，`FlowBodyNodeInspector` 根据 `selectedNode.kind` 渲染对应 form/json UI。
3. 用户编辑 template / bindings / source / branch cases / transform mode。
4. `FlowEditorWindow` 将这些编辑写回 body session snapshot，并通过既有提交路径同步到父节点 `foreachBodyJson`。
5. 项目保存时，body graph 仍通过父 `foreach` 的 `body` JSON 一次性输出。

#### Interface Drafts
- `FlowBodyNodeInspector` 新增 props：
  - `ancestorNodeOptions: string[]`
- `FlowBodyNodeInspector` 新增 emits：
  - `add-binding`
  - `remove-binding`
  - `binding-source-kind-change`
- `FlowEditorWindow` 现有函数扩展为 body-aware：
  - `addBinding()`
  - `removeBinding(index)`
  - `onBindingSourceKindChange(binding, sourceKind)`

#### Error Handling and Safety
- body editor 切回 form 继续复用 store 的 `setNodeSpecEditorMode` 校验语义。
- bindings/source/path/form 校验继续依赖保存阶段和现有 store helper，不在 body 侧引入新规则。
- body 场景出错时继续通过 toast 和 `bodySessionError` 暴露，不静默吞错。

#### Performance and Testing Strategy
- 不新增 body 专属 store，不做多余 JSON 往返，只沿用当前 body session -> parent JSON 同步链路。
- 测试覆盖：
  - body inspector 渲染高级节点普通模式
  - body inspector 触发 bindings 相关 emits
  - body editor 场景下 `add/remove binding` 真正落到 body 节点
  - 现有 body call authoring 不回归

#### Extensibility Design Points
- 先允许 body 非 `call` 具备与根图一致的最小表单能力，后续若要进一步抽共享组件，可在行为稳定后再重构。
- 继续保留 `transform` 顶层模式限制和 `foreach.body` JSON 真相源边界，为后续更强 authoring 保留清晰扩展点。

#### Issue List
- 无

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前状态：
  - 根图已支持 `compose / transform / set_var / branch / foreach / subflow` 的最小普通模式。
  - `foreach.body` 已有显式可视化编辑会话。
  - `foreach.body` 的 `call` 已具备 ordinary mode parity。
  - `foreach.body` 非 `call` 节点仍停留在 JSON-first。
- 本轮目标：
  - 把 body editor 补到“高阶节点也可最小表单 authoring”，使 body 子图与根图之间的作者体验差距进一步收敛。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: `updated`
- Specs impact: `updated`
- Lessons impact: `none`（当前未发现需要单独沉淀到 `docs/lessons` 的新排障规则）
- stable truth：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
- workflow results：
  - `docs/change/2026-04-03_win-flow-foreach-body-advanced-authoring.md`

#### Related Requirements / Specs / Lessons
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- Related lessons:
  - 无

#### Executable Task List
- [x] `BODY-AUTH-DOC-1` 更新本地 requirements/spec，放开 body 非 `call` 节点的最小普通模式边界
- [x] `BODY-AUTH-UI-1` 扩展 `FlowBodyNodeInspector.vue`，补齐高级节点表单和 bindings/source 交互
- [x] `BODY-AUTH-WIN-1` 扩展 `FlowEditorWindow.vue` 的 body 场景 wiring，使 bindings 与 ancestor 选项在 body 中可用
- [x] `BODY-AUTH-TEST-1` 更新 body inspector / window 测试并执行验证

#### Task Details
##### `BODY-AUTH-DOC-1` - Update Stable Docs for Body Advanced Authoring
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps\plan.md`
- Goal:
  - 让 requirements/spec 与 body 非 `call` 节点的新普通模式边界一致
- Files / Modules:
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
- Write Set:
  - 上述两份文档
- Acceptance:
  - 不再把 body 非 `call` 普通模式一律列为 out-of-scope
  - 明确保留 `foreach.body` JSON 真相源和现有 form/json gating
- Test Points:
  - 文档与实现边界一致
- Rollback:
  - 回退这两份文档

##### `BODY-AUTH-UI-1` - Extend Body Inspector Form Coverage
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps\plan.md`
- Goal:
  - 在 body inspector 中暴露根图已有的高级节点最小表单
- Files / Modules:
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
- Write Set:
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
- Acceptance:
  - body 非 `call` 节点可显示对应 form UI
  - bindings/source/ancestor 交互入口完整
  - 不破坏现有 call authoring
- Test Points:
  - 组件测试覆盖 transform / branch / set_var 或 subflow 至少一类高级节点
  - bindings 相关 emits 被正确触发
- Rollback:
  - 回退 `FlowBodyNodeInspector.vue`

##### `BODY-AUTH-WIN-1` - Wire Body Binding and Ancestor Actions
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps\plan.md`
- Goal:
  - 让 body editor 里的表单交互真正作用于 body session snapshot
- Files / Modules:
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write Set:
  - `frontend/src/windows/FlowEditorWindow.vue`
- Acceptance:
  - body 模式下可以 add/remove binding
  - body 模式下 source-kind 切换后字段重置逻辑正确
  - ancestor 选项来自 body graph
- Test Points:
  - body editor 场景的窗口测试
- Rollback:
  - 回退 `FlowEditorWindow.vue`

##### `BODY-AUTH-TEST-1` - Verification
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps\plan.md`
- Goal:
  - 用最小测试证明 body 非 `call` authoring 已打通且未回归现有路径
- Files / Modules:
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
- Write Set:
  - 上述测试文件
- Acceptance:
  - 相关 vitest 测试通过
  - 若存在环境限制，需明确记录未覆盖项
- Test Points:
  - `pnpm vitest run frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts frontend/src/windows/FlowEditorWindow.test.ts`
- Rollback:
  - 回退测试改动

#### Dependencies
- `BODY-AUTH-DOC-1` 与实现任务可并行起草，但最终必须与实现一致
- `BODY-AUTH-WIN-1` 依赖 `BODY-AUTH-UI-1` 的 props/emits 设计定稿
- `BODY-AUTH-TEST-1` 依赖前两项完成

#### Risks and Notes
- 优先做最小复制，不主动进行根图/body inspector 合并重构
- 如果实现中发现必须修改 `flow.ts` 才能让 body 场景落盘，需要先回到 `3.1` 更新计划
- 主仓的用户改动不触碰、不回退

#### Parallelism Assessment
- 当前不派发子 agent
- 原因：
  - 这轮主要是同一条 UI/交互链路上的小范围耦合改动
  - `FlowBodyNodeInspector.vue` 与 `FlowEditorWindow.vue` 写集相互依赖，强行并行收益低且容易冲突

#### Issue List
- 无

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Task Execution Record
- `BODY-AUTH-UI-1`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - 为 body inspector 补齐 `compose / transform / set_var / branch / foreach / subflow` 的最小普通模式 UI。
  - 新增 body 场景需要的 `ancestorNodeOptions`、`add-binding`、`remove-binding`、`binding-source-kind-change`。
- `BODY-AUTH-WIN-1`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - body 会话下的 add/remove binding 与 source-kind 切换已接入父 `foreachBodyJson` 同步链路。
  - `setBodySelectedNodeSpecMode()` 已允许支持普通模式的 body 节点回切 `form`。
  - body 会话初始化与新建节点时，不再把非 `call` 节点强制回落为 `json`。
- `BODY-AUTH-TEST-1`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
  - 增加 body `set_var` 普通模式与 body binding 落盘验证。
- `BODY-AUTH-DOC-1`
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - 稳定文档已更新为“body 非 call 节点支持最小普通模式，但仍保留单一 JSON 真相源和嵌套 foreach 非递归 body 会话”。

#### Validation
- 依赖安装：
  - `npm ci`（原因：worktree 初始缺少 `frontend/node_modules`，且环境无 `pnpm`）
- 定向测试：
  - `npm test -- src/components/flow/editor/FlowBodyNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：`2 passed / 11 passed`

#### Notes
- 实现保持在计划内文件集合，没有引入 `flow.ts` 或运行时协议层改动。
- `foreach.body` 继续只以父节点 `foreachBodyJson` 为持久化真相源。

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 计划中的 body 非 `call` 最小普通模式、bindings wiring、ancestor 约束和文档同步已覆盖。
- 架构合理性：通过
  - 只扩展 body inspector 和 body window wiring；未引入第二套 body 持久化模型。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅复用现有 body session -> parent JSON 同步链路，未新增额外 I/O 或全图重复构建。
- 可读性与一致性：通过
  - body UI 文案、form/json gating 和根图同类节点保持一致；测试断言同步更新。
- 可扩展性与配置化：通过
  - 保持 root/body 组件边界，未做高耦合大重构；后续仍可继续抽共享层。
- 稳定性与安全：通过
  - body `node_result` 祖先仍按 body 子图拓扑限制；超出普通模式覆盖范围时继续显式失败。
- 测试覆盖情况：通过
  - 已覆盖 body inspector 的 `set_var` 普通模式入口与 window 级 body binding 落盘路径。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未使用子Agent。

### Stage 4 - Change Archive
#### Archive Target
- `docs/change/2026-04-03_win-flow-foreach-body-advanced-authoring.md`

#### Lessons Decision
- `Lessons impact: updated`
- 新增：
  - `docs/lessons/flow-body-spec-mode-normalization.md`
- 原因：
  - body 高级节点 authoring 是否真正可用，不只取决于 inspector，还取决于 body session 是否错误覆盖 `specEditorMode`；该排障线索隐蔽且可能复发。
