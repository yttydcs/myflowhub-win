# Win Flow Status Wiring

## 变更背景 / 目标

- Win Flow 编辑器的 store 已经具备 `flow.run` / `flow.status` 能力，但窗口层没有稳定的运行状态入口，画布节点也一直没有消费真实 `status` 摘要。
- `FlowCanvas.vue` 和 `FlowNode.vue` 早已有 badge 展示能力，但 `FlowEditorWindow.vue` 一直把 `statusNodes` 传空数组，导致状态 UI 与真实运行完全脱节。
- 本轮目标是完成 `WIN-ST-1`：补齐最小 run/status toolbar 入口、把 `lastStatus.nodes` 接到画布、并集中修复状态跨草稿残留问题。

## 具体变更内容

- 更新稳定文档：
  - `docs/requirements/flow-editor-status-wiring.md`
  - `docs/specs/flow-editor-status-wiring.md`
  - 明确了 `Run Flow` / `Refresh Status` 最小入口、`lastStatus` reset 约束、以及画布 badge 接线边界。
- 收口 store 状态模型：
  - `frontend/src/stores/flow.ts` 新增 `createEmptyFlowStatus()` / `resetStatusState()`
  - `newDraft()`、`applyGraphDraft()`、`applyGraphEditorState()` 统一清空 `statusRunId + lastStatus`
  - `flowStatusLabelKey(...)` 新增 `cancelled -> Cancelled`
- 更新窗口与 toolbar：
  - `frontend/src/components/flow/editor/FlowEditorToolbar.vue` 新增 `Run Flow` / `Refresh Status` 按钮
  - `frontend/src/windows/FlowEditorWindow.vue` 新增 `runCurrentFlow()` / `refreshFlowStatus()`
  - toolbar 同时展示 flow 级摘要状态和当前 `run_id`
- 更新画布与节点展示：
  - `frontend/src/windows/FlowEditorWindow.vue` 现在把 `flowStore.state.lastStatus.nodes` 传给 `FlowCanvas.vue`
  - `frontend/src/components/flow/FlowNode.vue` 补齐 `cancelled` badge tone，并让 `set_var` 节点显示正确类型标签
  - `frontend/src/components/flow/FlowCanvas.vue` 对 `set_var` 节点把局部变量名作为 meta 展示
- 补充测试与文案：
  - `frontend/src/i18n/messages/automation.ts` 新增运行状态相关文案
  - `frontend/src/stores/flow.test.ts` 覆盖 status reset、`cancelled` label key、`StatusSimple` 映射
  - `frontend/src/windows/FlowEditorWindow.test.ts` 覆盖 toolbar run/refresh 路径与 `statusNodes` 下发
  - `frontend/src/components/flow/FlowNode.test.ts` 覆盖 `cancelled` badge 和 `set_var` 标签展示

## Requirements impact: updated

## Specs impact: updated

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-status-wiring.md`
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-run-detail.md`
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-visual-form.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-status-wiring.md`
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-run-detail.md`
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\worktrees\server-local-vars-docs\docs\specs\flow.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `WIN-ST-1`
  - `docs/requirements/flow-editor-status-wiring.md`
  - `docs/specs/flow-editor-status-wiring.md`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowEditorToolbar.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
  - `frontend/src/components/flow/FlowCanvas.vue`
  - `frontend/src/components/flow/FlowNode.vue`
  - `frontend/src/components/flow/FlowNode.test.ts`
  - `frontend/src/i18n/messages/automation.ts`

## 经验 / 教训摘要

- 状态接线前必须先修 `lastStatus` reset；否则一旦 badge 接上线，脏状态会立刻从“不可见”变成“持续污染 UI”。
- `status` 与 `detail` 需要共享 `run_id`，但仍应由 window 层决定“当前刷新用哪个 run_id”，不要让画布组件知道任何请求细节。
- `set_var` 节点在画布上的 label / meta 如果不一起校正，用户会在状态接通后更明显地看到类型展示错误。

## 可复用排查线索

- 症状：
  - 画布节点始终显示 `Unknown`
  - 切换项目后仍看到上一个 flow 的 badge
  - toolbar 点 `Refresh Status` 没有命中当前 `run_id`
- 触发条件：
  - `FlowEditorWindow.vue` 仍把 `statusNodes` 传空数组
  - store 没有统一 reset `lastStatus`
  - toolbar 调用 `statusFlow("")` 而不是优先用 `statusRunId`
- 关键词：
  - `lastStatus`
  - `statusRunId`
  - `resetStatusState`
  - `Refresh Status`
  - `cancelled`
- 快速检查：
  - 检查 `frontend/src/windows/FlowEditorWindow.vue` 是否把 `flowStore.state.lastStatus.nodes` 传给 `FlowCanvas`
  - 检查 `frontend/src/stores/flow.ts` 的 `newDraft/loadGraphDraft/loadGraphEditorState` 是否都调用统一 reset helper
  - 检查 `flowStatusLabelKey(...)` 是否包含 `cancelled`

## 关键设计决策与权衡

- 决策：状态刷新继续走显式按钮，不引入轮询。
  - 原因：当前目标是保证刷新路径稳定，自动轮询会放大 transport、并发和 stale-state 处理复杂度。
- 决策：`statusNodes` 直接来自 store `lastStatus.nodes`，不在画布层再做额外衍生缓存。
  - 原因：单一状态源最稳，也更方便和后续 `WIN-SC-1` / detail 共用 `run_id`。
- 决策：toolbar 新增最小 run/status 入口，而不是新开独立 runtime 面板。
  - 原因：这轮只需要让用户能触发和刷新摘要状态，不需要再造第二个运行视图。

## 测试与验证方式 / 结果

- `npm test`
  - 结果：通过（8 个文件，33 个用例）
- 子任务定向验证：
  - `npx vitest run src/stores/flow.test.ts`
  - 结果：通过（1 个文件，13 个用例）

## 潜在影响与回滚方案

- 潜在影响：
  - `Run Flow` / `Refresh Status` 现在直接出现在 editor toolbar，后续若需要更复杂的运行控制，需要继续扩 toolbar 信息架构。
  - 当前仍是手动刷新模型，没有自动追踪远端 run 状态变化。
- 回滚方案：
  - 回退 `frontend/src/stores/flow.ts` 的 status reset / cancelled label key
  - 回退 `FlowEditorToolbar.vue`、`FlowEditorWindow.vue`、`FlowCanvas.vue`、`FlowNode.vue`
  - 回退新增 i18n 和测试文件

## 子Agent执行轨迹

- `/root/win_st1_store_status`
  - 负责文件：
    - `frontend/src/stores/flow.ts`
    - `frontend/src/stores/flow.test.ts`
  - 结果：
    - 新增统一 status reset helper
    - 补齐 `cancelled` label key
    - 补齐 store 级 status reset / payload 测试
    - 子任务内执行 `npx vitest run src/stores/flow.test.ts` 通过
