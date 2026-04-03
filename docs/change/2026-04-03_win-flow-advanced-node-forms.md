# 2026-04-03_win-flow-advanced-node-forms

## 变更背景 / 目标

- 上一轮 Win Flow 编辑器已支持 `cron` 和高阶节点的识别、创建、保存，但 `transform/branch/foreach/subflow` 仍默认只能走 `Advanced JSON`。
- 本轮目标是把 `transform`、`branch`、`subflow` 提升到可安全表单 authoring，同时明确保留 `foreach` 的 JSON-only 边界。

## 具体变更内容

- `frontend/src/stores/flow.ts`
  - 扩展 `FlowNodeDraft`，新增 `transform`、`branch`、`subflow` 的显式表单字段。
  - 新增高级节点 spec 解析、宽松序列化、严格保存校验和 form/json mode gating。
  - `transform` 支持顶层 `literal/source/op/object/array` 五种模式。
  - `branch` 支持 case 列表、`match.source/op/value`、`default_case`，并校验 case 名称唯一。
  - `subflow` 支持 `flow_id`、`input_template`、`inputs`、`result_node_id`，并校验 UUID。
  - `foreach` 继续保持 JSON-only；回切 form 会显式报错。
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 新增 `transform` 普通模式表单。
  - 新增 `branch` 普通模式表单。
  - 新增 `subflow` 普通模式表单，并复用既有 bindings 编辑器。
  - 保留 `foreach` 的 Advanced JSON only 提示。
- 测试
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - 相关 window 级回归
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

- `WIN-ORCH-DOC-2`
  - 更新本地 requirements/spec，明确 `transform/branch/subflow` 普通模式与 `foreach` JSON-only 边界
- `WIN-ORCH-RT-4`
  - `frontend/src/stores/flow.ts`
- `WIN-ORCH-RT-5`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
- `WIN-ORCH-TEST-2`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`

## 经验 / 教训摘要

- 高阶节点表单化可以先做“顶层结构化 + 局部 JSON 文本区”，不必一次性做完整递归可视化编辑器。
- `branch` 的节点表单与 `edge.case` 必须保持职责分离；一旦把边路由语义错误地塞回节点，会再次破坏 graph round-trip。

## 可复用排查线索

- 看到高级节点从 JSON 切回 form 失败时，优先检查：
  - `parseSpecDraft(...)`
  - `parseTransformDraft(...)`
  - `parseBranchDraft(...)`
  - `parseSubflowDraft(...)`
- 看到 `branch` 保存时报 `default_case` 或 case 重复错误时，优先检查：
  - `branchCases`
  - `branchDefaultCase`
  - graph 出边的 `edge.case`
- 看到 `subflow` 保存时报 UUID 错误时，优先检查：
  - `subflowId`
  - server stable spec 中 `flow_id` 的 UUID 约束

## 关键设计决策与权衡

- `transform` 只做顶层模式表单
  - 好处：范围可控，仍能覆盖常见 literal/source/op 场景
  - 代价：深层表达式仍需通过 JSON 文本输入
- `subflow` 复用 template + bindings 骨架
  - 好处：最小改动、减少重复 UI 和状态模型
  - 代价：和 `compose` 的视觉结构相近，需要靠文案强调这是子 flow 输入
- `foreach` 继续 JSON-only
  - 好处：避免在本轮引入 body 子图编辑器，边界清晰
  - 代价：`foreach` 仍是当前高阶编排中唯一必须直接写 JSON 的节点

## 测试与验证方式 / 结果

- 定向前端测试
  - 命令：
    - `npx vitest run src/stores/flow.test.ts src/components/flow/editor/FlowNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：
    - 通过，`3 passed / 35 passed`
- 分阶段测试
  - 命令：
    - `npx vitest run src/stores/flow.test.ts`
    - `npx vitest run src/components/flow/editor/FlowNodeInspector.test.ts`
  - 结果：
    - 通过

## 潜在影响

- Win Flow 编辑器现在可直接表单编辑 `transform`、`branch`、`subflow` 的关键字段。
- `foreach` 仍不会误入普通模式，避免把 `body` 子图错误压扁。
- 高级节点从 JSON 切回 form 时会更严格；超出当前普通模式覆盖范围的 spec 现在会显式报错，而不是构造伪表单。

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
