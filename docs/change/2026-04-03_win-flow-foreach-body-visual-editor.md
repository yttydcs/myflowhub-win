# 2026-04-03_win-flow-foreach-body-visual-editor

## 变更背景 / 目标

- 上一轮 Win Flow 编辑器已支持 `foreach` 的外层字段普通模式，但 `body` 仍只能通过 JSON 文本区维护。
- 本轮目标是补齐 `foreach.body` 的显式可视化子图 authoring，同时保持 root graph 的保存、dirty 和 recovery baseline 不被破坏。

## 具体变更内容

- `frontend/src/stores/flow.ts`
  - 新增 body 会话需要的 graph helper：
    - `createGraphEditorStateFromDraft(...)`
    - `exportLooseGraphDraftFromEditorState(...)`
  - helper 只负责 graph draft 与 editor draft 的宽松转换，不改现有 root editor 状态机。
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 为 `foreach` 普通模式新增 `Open Visual Body Editor` 入口。
  - 更新说明文案，明确可视化 body editor 仍以父节点 `body` JSON 为真相源。
- `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - 新增 body 内节点的最小 JSON-first inspector。
  - 当前支持：
    - node id
    - kind
    - allow_fail
    - retry
    - timeout_ms
    - `Advanced JSON`
- `frontend/src/windows/FlowEditorWindow.vue`
  - 新增显式 body 编辑会话和顶部 breadcrumb/banner。
  - 画布可在 root graph 和 `foreach.body` 子图之间切换。
  - body 会话支持：
    - 新增节点
    - 选中节点/边
    - 移动节点
    - 连线
    - 删除节点/边
    - `edge.case` 最小编辑
  - 每次有效提交都会同步回父节点 `foreachBodyJson`，并进入 root history。
  - 保存项目、运行 flow、撤销重做前都会先尝试同步 body 会话，避免根 graph 与 body 会话漂移。
  - recovery draft 现在会一并记录 body 会话快照，避免本地 body 编辑上下文丢失。
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

- `WIN-ORCH-DOC-4`
  - 更新本地 requirements/spec，明确 body 编辑会话边界
- `WIN-ORCH-RT-8`
  - `frontend/src/stores/flow.ts`
- `WIN-ORCH-RT-9`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
- `WIN-ORCH-TEST-4`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`

## 经验 / 教训摘要

- 嵌套 graph editor 不一定要把全局 store 切换到子图；只要父节点 JSON 仍是唯一持久化真相源，就可以先用 window 级显式会话把保存/恢复风险压到最小。
- `foreach.body` 这类局部子图的最小可行路径是“图结构可视化 + 节点 JSON-first inspector”，不必一次性复刻顶层 `call` visual form 的全部能力。

## 可复用排查线索

- 看到 body editor 打不开时，优先检查：
  - `foreachBodyJson` 是否为合法 JSON
  - `body.nodes` / `body.edges` 是否为数组
  - 父 `foreach` 节点是否仍处于 `form` mode
- 看到 body 会话保存失败时，优先检查：
  - `syncBodySessionToParent(...)`
  - `exportLooseGraphDraftFromEditorState(...)`
  - `bodySessionError`
- 关键词：
  - `Failed to open foreach body graph`
  - `Failed to sync foreach body graph`
  - `Fix the foreach body editor errors before saving`

## 关键设计决策与权衡

- body 会话保持在 `FlowEditorWindow.vue`，而不是切换全局 `flowStore`
  - 好处：root graph 的 dirty、save、recovery baseline 不需要重写
  - 代价：body 会话需要额外的同步逻辑
- body 内节点先走 JSON-first inspector，而不是完整复刻顶层表单
  - 好处：改动面小，能尽快把 `foreach.body` 从纯 JSON 提升到可视化子图编辑
  - 代价：body 内 `call` 节点暂时没有方法选择器和字段级 binding 对话框

## 测试与验证方式 / 结果

- 定向前端测试
  - 命令：
    - `npx vitest run src/stores/flow.test.ts src/components/flow/editor/FlowNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：
    - 通过，`39 passed`

## 潜在影响

- Win Flow 编辑器现在可以直接以画布方式维护 `foreach.body` 的内部 DAG。
- body 会话中的图结构改动会稳定回写到父节点 `foreachBodyJson`，保存项目时不需要先手工回到 JSON 文本区。
- 当前 body 节点 payload authoring 仍是 JSON-first；如果用户需要顶层 `call` visual form 同级别体验，还需要下一轮继续补。

## 回滚方案

- 回退以下文件即可撤销本轮能力：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`

## 子Agent执行轨迹

- 本轮未使用子Agent。
