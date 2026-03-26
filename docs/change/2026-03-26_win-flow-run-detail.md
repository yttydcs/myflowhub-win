# Win Flow Run Detail

## 变更背景 / 目标

- Win Flow 编辑器已经具备 graph authoring 和本地 `set_var/flow_var` 最小能力，但仍缺少对单次 run 的节点结果查看入口。
- Server `flow.detail` 已提供按 `flow_id + run_id? + node_id + path?` 查询节点结果的契约，本轮目标是完成 `WIN-RD-1` 的最小闭环接入。
- 本轮明确采用“选中节点后手动查询”的方案，不与后续 `WIN-ST-1` 的状态刷新和画布 badge 提前耦合。

## 具体变更内容

- 更新稳定文档：
  - `docs/requirements/flow-editor-run-detail.md`
  - `docs/specs/flow-editor-run-detail.md`
  - 明确 `flow.detail` 的 Win 编辑器入口、请求契约、展示字段和 `output_schema` 第一版消费范围。
- 扩展 Win FlowService：
  - `internal/services/flow/service.go` 新增 `Detail` / `DetailSimple`
  - `extractCodeMsg(...)` 新增 `*flow.DetailResp` 分支，保持现有错误包装链路一致。
- 扩展前端 store：
  - `frontend/src/stores/flow.ts` 新增 `FlowNodeDetailState`
  - 新增 `loadNodeDetail(...)`、`handleDetailResp(...)`、`resetNodeDetail(...)`
  - 新增 `getNodeOutputSchemaText(...)`，把当前 capability route 的 `output_schema` 格式化为文本
  - 节点切换、草稿加载、状态回写时同步重置或补齐当前 detail 面板状态。
- 更新 Win 编辑器 UI：
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue` 新增 `Result Detail` 区域
  - 展示可选 `run_id`、可选 `path`、`Load Detail` 按钮、错误区、resolved `run_id/path`、节点状态摘要、结果文本
  - `call` 节点额外展示当前 capability `output_schema` 的格式化 JSON 文本
  - `frontend/src/windows/FlowEditorWindow.vue` 负责把 inspector 输入、加载动作和 store `nodeDetail` 状态接线。
- 补充文案与测试：
  - `frontend/src/i18n/messages/automation.ts` 新增 result detail 文案
  - `frontend/src/stores/flow.test.ts` 覆盖 `DetailSimple` 请求映射、响应落库、`statusRunId` 更新、`getNodeOutputSchemaText(...)`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts` 覆盖 result detail 最小渲染。

## Requirements impact: updated

## Specs impact: updated

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-run-detail.md`
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-visual-form.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-run-detail.md`
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\worktrees\server-local-vars-docs\docs\specs\flow.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `WIN-RD-1`
  - `docs/requirements/flow-editor-run-detail.md`
  - `docs/specs/flow-editor-run-detail.md`
  - `internal/services/flow/service.go`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/i18n/messages/automation.ts`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`

## 经验 / 教训摘要

- 结果详情第一版必须保持“单节点、手动触发、单份状态”，否则会和后续状态刷新策略交叉污染。
- `output_schema` 第一版只作为文本展示是合理分层，避免在 `WIN-RD-1` 提前把 schema 跟结果渲染做过深耦合。
- `status` 与 `detail` 虽然共享 `run_id`，但 UI 语义必须分开处理；本轮只允许 `status` 为 detail 默认回填 `run_id`，不反过来驱动画布状态。

## 可复用排查线索

- 症状：
  - inspector 里点 `Load Detail` 无响应
  - `call` 节点始终显示 `No schema`
  - 节点切换后仍残留上一个节点的 detail 结果
- 触发条件：
  - `FlowEditorWindow.vue` 未把 `nodeDetail` 状态和 `load-node-detail` 事件接到 store
  - `frontend/src/stores/flow.ts` 没有在节点切换 / 草稿加载时重置 `nodeDetail`
  - capability route 未加载，导致找不到 `output_schema`
- 关键词：
  - `DetailSimple`
  - `FlowNodeDetailState`
  - `loadNodeDetail`
  - `getNodeOutputSchemaText`
  - `output_schema`
- 快速检查：
  - 检查 `internal/services/flow/service.go` 是否暴露 `DetailSimple`
  - 检查 `frontend/src/stores/flow.ts` 是否存在 `handleDetailResp(...)` 和 `resetNodeDetail(...)`
  - 检查 `FlowEditorWindow.vue` 的节点切换 watcher 是否调用 `flowStore.resetNodeDetail(...)`
  - 检查当前节点对应 capability route 是否已 hydration 到 `state.execCapabilities`

## 关键设计决策与权衡

- 决策：结果详情入口放在现有右侧 inspector，而不是新增独立运行结果窗口。
  - 原因：用户查询目标始终是“当前选中节点”，放在 inspector 内最直接，改动也最小。
- 决策：查询采用按钮触发，不随节点切换自动请求。
  - 原因：避免对未知 run 自动打请求，也避免在 `WIN-ST-1` 完成前制造刷新耦合。
- 决策：`output_schema` 只做格式化 JSON 文本展示。
  - 原因：本轮的目标是契约接通和结果可见性，不提前扩成结构化 schema/result viewer。

## 测试与验证方式 / 结果

- `npm test`
  - 结果：通过（7 个文件，26 个用例）
- `go test ./internal/services/flow`
  - 环境：`GOWORK=D:\project\MyFlowHub3\worktrees\win-local-vars-ui\verify.go.work`，并挂载 `../proto-local-vars-observability`
  - 结果：通过（目标包无测试文件，但编译与契约解析通过）

## 潜在影响与回滚方案

- 潜在影响：
  - 当前 detail 面板只覆盖“选中节点的单次查询”，尚未处理画布节点状态 badge 或自动刷新。
  - `compose` / `set_var` 目前不展示 schema，这一层仍留给后续 `WIN-SC-1`。
- 回滚方案：
  - 回退 `internal/services/flow/service.go` 的 `Detail` / `DetailSimple`
  - 回退 `frontend/src/stores/flow.ts` 的 `nodeDetail` 状态与 detail 查询逻辑
  - 回退 `FlowEditorWindow.vue`、`FlowNodeInspector.vue`、i18n 与对应测试

## 子Agent执行轨迹

- `/root/win_rd1_tests_i18n`
  - 负责文件：
    - `frontend/src/i18n/messages/automation.ts`
    - `frontend/src/stores/flow.test.ts`
    - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - 结果：
    - 补齐 result detail 文案
    - 补齐 store / inspector 测试
    - 子任务内执行 `npx vitest run src/stores/flow.test.ts src/components/flow/editor/FlowNodeInspector.test.ts` 通过
