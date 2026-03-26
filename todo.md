# Plan - win-local-vars-ui

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/local-vars-win`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui`
- Current Stage: `4 Archive`

## Goal
- 为 Win Flow 编辑器补齐 `flow_var` 来源、`set_var` 节点最小 authoring，并按顺序推进 detail/status/schema 收口

## Related Requirements
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-run-detail.md`
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-status-wiring.md`
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\worktrees\server-local-vars-docs\docs\requirements\flow_data_dag.md`

## Related Specs
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-run-detail.md`
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-status-wiring.md`
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\worktrees\server-local-vars-docs\docs\specs\flow.md`

## Requirements Impact
- `add`

## Specs Impact
- `add`

## Related Lessons
- 当前无直接相关 lessons；`docs/lessons/README.md` 中仅有 Wails 构建占位问题，与本轮无关

## Stage Records

### Initialization
- `guide.md`: not present
- base/worktree confirmation:
  - 主执行仓：`D:\project\MyFlowHub3\worktrees\win-local-vars-ui`
  - 主分支：`feat/local-vars-win`
  - Base：`main`
  - 主 repo 路径仅作 control-plane，不在其中做实现改动

### Stage 1 - Requirements Analysis
#### Goal
- 扩展 Win Flow 编辑器对安全 `input/output schema` 子集的消费能力，减少普通模式无谓回退，并让 `detail` 面板能按 `output_schema` 展示结构化结果

#### Scope
- Must:
  - 输入侧接受“单一受支持类型 + `null`”的 nullable schema 包装
  - 输出侧在 `call` 节点根结果查询时，按受限 `output_schema` 展示只读结构化字段视图
  - 所有不支持场景都稳定回退到现有 raw JSON 结果/ schema 展示
- Optional:
  - 更丰富的 schema 控件映射
  - 非根路径的子 schema 导航
- Out of Scope:
  - 修改后端 flow 协议
  - 支持 `oneOf/anyOf/allOf/$ref`
  - 支持数组驱动的动态表单或完整 schema explorer
  - 为 `compose/set_var` 推导 output schema

#### Use Cases
- capability schema 只因 nullable 写法而无法进入普通模式时，用户仍可继续使用字段表单
- 用户查看 `call` 节点根结果时，希望直接看到关键字段和值，而不是只面对原始 JSON
- 用户查看非根路径或不受支持 schema 时，仍能稳定查看 raw JSON 文本

#### Functional Requirements
- resolver 必须接受安全的 nullable schema 包装
- detail 面板必须在支持时展示结构化结果，在不支持时回退 raw JSON
- 原始结果与原始 schema 文本必须继续保留，作为审计和高级排查入口

#### Non-functional Requirements
- 不新增 transport 或第二套 store
- schema 解析必须显式拒绝超出子集的特性
- 结构化展示只针对当前 inspector，不扩散成全图缓存

#### Inputs / Outputs
- Inputs:
  - capability `input_schema`
  - capability `output_schema`
  - `flow.detail.result`
- Outputs:
  - 扩展后的 visual schema 解析结果
  - root result 的结构化只读展示模型

#### Edge Cases
- 顶层 schema 或字段 schema 使用不受支持的多类型 union
- detail 查询路径非根
- output schema 与 result 值形态不一致
- schema 字段在 result 中缺失

#### Acceptance Criteria
- 至少一批仅因 nullable 包装导致回退的方法可继续使用普通模式
- 支持 schema 的 `call` 根结果可展示结构化字段和值
- 非支持场景继续稳定展示 raw JSON，不产生破坏性错误

#### Risks
- 若把 schema 子集放得过宽，会让普通模式进入不可审计状态
- 若结构化结果遮蔽原始 payload，会增加误判风险，因此必须保留 raw JSON 视图

#### Issue List
- None

### Stage 2 - Architecture Design
#### Overall Solution
- 继续采用“受限 schema 子集 + 显式回退”的前端方案，不引入完整 JSON Schema 引擎
- 复用现有 capability route 和 detail store 链路，仅扩 schema 解析与 inspector 展示 helper

#### Alternatives Considered
- 引入通用 schema 表单/渲染库：放弃，改动面过大，超出本轮
- 继续只展示 raw JSON：放弃，无法完成 `WIN-SC-1` 对 output schema 的消费目标
- 使用前端受限解析器：采用，边界清晰、可测试、可回退

#### Module Responsibilities
- `frontend/src/stores/flow_schema_resolver.ts`：扩展 input schema 解析规则
- `frontend/src/stores/flow.ts`：保留 detail 原始值并提供 output schema 结构化展示 helper
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`：消费结构化结果模型并保留 raw JSON 展示

#### Data / Call Flow
- capability query 继续用 `include_schema=true` 获取 `input_schema/output_schema`
- resolver 解析 `input_schema` 生成普通模式字段
- detail 成功后 store 保留原始 `result`
- inspector 结合 `output_schema + result` 生成只读结构化视图
- 任何不支持情况都回退到现有 raw JSON 视图

#### Interface Drafts
- `FlowNodeDetailState` 增加 `resultValue: unknown`
- 新增 output schema 结构化 helper，输入为 `output_schema + result + requestedPath`
- `requestedPath` 非空时直接走回退路径，不尝试子 schema 推导

#### Error Handling and Safety
- 不支持 schema 仅禁用结构化视图，不视为 detail 请求失败
- 结构化展示为只读视图，不执行 schema
- 结果字段缺失时展示空值占位，不伪造默认值

#### Performance and Testing Strategy
- 只解析当前选中节点的 output schema
- 新增 resolver 单测、detail helper 单测、inspector 渲染单测
- 保持现有 raw JSON detail 展示回归验证

#### Extensibility Design Points
- 输入和输出 schema 解析风格保持一致，但职责分离
- 后续若要支持数组或 path 子 schema，可在 helper 层扩展，不重写 store 主链路

#### Issue List
- None

### Stage 3.1 - Planning
#### Project Goal and Current State
- 已完成：
  - `WIN-LV-1`
  - `WIN-RD-1`
  - `WIN-ST-1`
- 当前 round：
  - 进入 `WIN-SC-1`
  - 已完成 Stage `1`、Stage `2`
  - 当前补齐 requirements/specs 影响和可执行计划，准备进入 `3.2`

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验后确认：本轮 stable truth 继续落在既有 leaf docs
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `docs/requirements/flow-editor-run-detail.md`
  - `docs/specs/flow-editor-run-detail.md`
- 不新增 `docs/change` 或 `docs/lessons` 于本阶段
- 本轮未新增 docs 拓扑，因此不需要更新 `docs/README.md` 或 category README

#### Related Requirements / Specs / Lessons
- Requirements:
  - `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-run-detail.md`
- Specs:
  - `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-run-detail.md`
  - `D:\project\MyFlowHub3\worktrees\server-local-vars-docs\docs\specs\flow.md`
- Lessons:
  - none currently relevant

#### Executable Task List
- [x] 完成 `WIN-SC-1` Stage `1`
- [x] 完成 `WIN-SC-1` Stage `2`
- [x] `WIN-SC-1A` 扩展 input schema resolver 和测试
- [x] `WIN-SC-1B` 扩展 output schema 结果展示、store helper 和测试
- [x] 运行相关前端测试
- [x] 执行 Stage `3.3` review
- [x] 进入 Stage `4` 归档

#### Dependencies
- capability query 必须继续返回 `include_schema=true`
- detail 结果仍依赖 `flow.detail` 现有后端契约
- output schema 结构化消费必须与 visual form 的受限 schema 子集保持一致

#### Risks and Notes
- `flow.ts` 是本轮主要耦合点，需避免与并行修改冲突
- 结构化结果视图只能是辅助展示，不能取代 raw JSON
- 用户已明确：`flow local var` 与 `varstore` 语义必须保持清晰分离

#### Parallelism Assessment
- 可拆分子任务：
  - `WIN-SC-1A`：write set 限于 resolver 与测试，适合委派
  - `WIN-SC-1B`：涉及 `flow.ts` 与 inspector 联动，保留主 agent 集成更安全
- 进入 `3.2` 后，如宿主策略和上下文包满足条件，可对 `WIN-SC-1A` 使用子 agent

#### Issue List
- None

## Tasks
##### WIN-LV-1 - local vars authoring
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\todo.md`
- Goal: 支持 `set_var` 节点和 `flow_var` 来源
- Files / Modules:
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/editor/*`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/**.test.ts`
- Write Set:
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/editor/**`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - 对应测试
- Acceptance:
  - 可创建 `set_var`
  - binding dialog 可选 `flow_var`
  - graph 存取与 SubProto 契约一致
- Test Points:
  - store round-trip
  - dialog / inspector tests
- Rollback:
  - 回退 `set_var` / `flow_var` UI 相关改动

##### WIN-RD-1 - result detail panel
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\todo.md`
- Goal: 接入 `run detail`，展示节点结果详情
- Files / Modules:
  - `internal/services/flow/service.go`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/**`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/**.test.ts`
- Write Set:
  - `internal/services/flow/service.go`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/**`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - 对应测试
- Acceptance:
  - 用户可在选中节点上查询并查看结果详情
  - 对 `output_schema` 有基础消费
- Test Points:
  - flow service detail binding
  - detail panel store/UI
- Rollback:
  - 回退 detail panel 相关改动

##### WIN-ST-1 - status wiring
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\todo.md`
- Goal: 画布节点状态接线和刷新策略收口
- Files / Modules:
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/FlowCanvas.vue`
  - `frontend/src/components/flow/FlowNode.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write Set:
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/**`
  - `frontend/src/windows/FlowEditorWindow.vue`
- Acceptance:
  - 节点 badge 使用真实状态
  - refresh 路径稳定
- Test Points:
  - status payload render
  - refresh behavior
- Rollback:
  - 回退 status wiring

##### WIN-SC-1 - schema follow-up
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\todo.md`
- Goal: 扩更多 input/output schema 的普通模式和结果展示消费
- Files / Modules:
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/**`
- Write Set:
  - `frontend/src/stores/flow_schema_resolver*`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/**`
- Acceptance:
  - 更多方法不再退回 `Advanced JSON`
  - output schema 可参与结果展示
- Test Points:
  - resolver tests
  - result render tests
- Rollback:
  - 回退 schema consumption 相关改动

##### WIN-SC-1A - nullable schema resolver follow-up
- Owner: delegable worker or main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\todo.md`
- Goal: 让受限 nullable input schema 继续进入普通模式，同时保持 unsupported schema 的显式回退边界
- Files / Modules:
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow_schema_resolver.test.ts`
- Write Set:
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow_schema_resolver.test.ts`
- Key Context References:
  - `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\worktrees\server-local-vars-docs\docs\specs\flow.md`
- Acceptance:
  - `type=[T,"null"]` 的安全 schema 可生成普通模式字段
  - 其他多类型 union、数组、组合 schema 继续回退
- Test Points:
  - resolver unit tests
  - unsupported fallback tests
- Rollback:
  - 回退 resolver follow-up 改动

##### WIN-SC-1B - output schema result rendering
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\todo.md`
- Goal: 在 detail 面板中消费受限 `output_schema`，为根结果提供结构化只读展示并保留 raw JSON 回退
- Files / Modules:
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/i18n/messages/automation.ts`
- Write Set:
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/i18n/messages/automation.ts`
- Key Context References:
  - `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-run-detail.md`
  - `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-run-detail.md`
  - `D:\project\MyFlowHub3\worktrees\server-local-vars-docs\docs\specs\flow.md`
- Acceptance:
  - 支持 schema 的 `call` 根结果展示结构化字段和值
  - 非根路径或不支持 schema 时稳定回退 raw JSON
  - 原始 schema / result 文本继续保留
- Test Points:
  - detail helper/store tests
  - inspector render tests
- Rollback:
  - 回退 output schema 结果展示改动

## Notes
- 本仓严格按顺序执行：`WIN-LV-1` -> `WIN-RD-1` -> `WIN-ST-1` -> `WIN-SC-1`
- 已完成：`WIN-LV-1`、`WIN-RD-1`、`WIN-ST-1`
- 下一步：`WIN-SC-1`
- 归档完成后，如继续 workflow，需从 `WIN-SC-1` 重新进入 Stage `1`
- 当前 `WIN-SC-1` 已完成 Stage `1`、`2`、`3.1`、`3.2`、`3.3`、`4`

阻塞：否
完成 4，等待是否结束 workflow
