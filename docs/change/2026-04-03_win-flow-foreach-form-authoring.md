# 2026-04-03_win-flow-foreach-form-authoring

## 变更背景 / 目标

- 上一轮 Win Flow 编辑器已支持 `transform`、`branch`、`subflow` 的普通模式 authoring，但 `foreach` 仍只能走 `Advanced JSON`。
- 本轮目标是把 `foreach` 提升到“外层字段可表单编辑、`body` 继续保持 JSON 真相源”的最小安全普通模式。

## 具体变更内容

- `frontend/src/stores/flow.ts`
  - 扩展 `FlowNodeDraft`，新增 `foreachSource`、`foreachRequired`、`foreachBodyJson`、`foreachResultNodeId`。
  - 新增 `foreach` spec 解析、宽松序列化、严格保存校验和 form/json mode gating。
  - `foreach` 当前支持表单化的字段为：
    - `source`
    - `required`
    - `body`（JSON 文本区）
    - `result_node_id`
  - 对超出当前普通模式覆盖范围的 `foreach` 顶层字段，切回 form 会显式失败。
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 新增 `foreach` 普通模式表单。
  - 复用既有 source-kind/source-path 交互来编辑 `foreach.source`。
  - 明确 `body` 仍需通过 JSON 文本区维护，不伪装成嵌套子图编辑器。
- 测试
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
- 文档
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\requirements\flow-editor-visual-form.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`

## Related lessons

- 无

## 对应 plan.md 任务映射

- `WIN-ORCH-DOC-3`
  - 更新本地 requirements/spec，明确 `foreach` 的部分普通模式边界
- `WIN-ORCH-RT-6`
  - `frontend/src/stores/flow.ts`
- `WIN-ORCH-RT-7`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
- `WIN-ORCH-TEST-3`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`

## 经验 / 教训摘要

- 高阶编排节点不必非要等到完整子图编辑器就绪后才开放普通模式；只要边界清晰，先做“外层结构化 + 内层 JSON 真相源”也能安全前进。
- `foreach.body` 这类嵌套 graph 一旦不再作为单一文本真相源，就很容易在 mode 切换时丢字段或破坏语义。

## 可复用排查线索

- 看到 `foreach` 从 JSON 切回 form 失败时，优先检查：
  - `parseForeachDraft(...)`
  - `supportsFormMode(...)`
  - `setNodeSpecEditorMode(...)`
- 看到 `foreach` 保存时报错时，优先检查：
  - `foreachSource`
  - `foreachBodyJson`
  - `foreachResultNodeId`
- 关键词：
  - `foreach body must be valid JSON`
  - `foreach body must include a nodes array`
  - `foreach result node ID is required`
- 快速检查：
  - `body` 是否为 object
  - `body.nodes` / `body.edges` 是否为数组
  - 顶层是否多出当前普通模式未覆盖字段

## 关键设计决策与权衡

- `foreach` 采用部分普通模式而不是继续 JSON-only
  - 好处：补齐了当前 flow editor 最大的高级节点空洞，用户能直接维护外层关键字段
  - 代价：`body` 仍需通过 JSON 文本维护，暂时没有可视化嵌套图编辑体验
- `body` 继续保持 JSON 真相源
  - 好处：不引入新的嵌套状态模型，form/json round-trip 风险更低
  - 代价：普通模式体验只覆盖外层字段，不覆盖子图内部 authoring

## 测试与验证方式 / 结果

- 定向前端测试
  - 命令：
    - `npx vitest run src/stores/flow.test.ts src/components/flow/editor/FlowNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：
    - 通过，`3 passed / 36 passed`

## 潜在影响

- Win Flow 编辑器现在可以在普通模式下创建、读取和维护 `foreach` 的外层字段。
- `foreach.body` 仍不会被错误拆成伪表单状态，减少嵌套 graph 丢失风险。
- `foreach` 相关校验会更早暴露到前端保存路径，例如空 `result_node_id` 或错误的 `body` 结构。

## 回滚方案

- 回退以下文件即可撤销本轮能力：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`

## 子Agent执行轨迹

- 本轮未使用子Agent。
