# 2026-04-03_win-flow-orchestrator-editor

## 变更背景 / 目标

- 稳定 flow 已支持 `cron` trigger 和 `transform/branch/foreach/subflow` 节点，但 Win 编辑器此前仍只覆盖旧节点集合和旧 trigger 集合。
- 本轮目标是先把编辑器补到“安全可 authoring”状态：
  - `cron` 可在项目部署和 flow 编辑链路中正确 round-trip；
  - 新节点种类可识别、可创建、可保存；
  - 复杂节点先走 JSON-only，不伪装成错误表单；
  - `branch` 出边 `edge.case` 可编辑且不会在保存时丢失。

## 具体变更内容

- `frontend/src/stores/flow.ts`
  - 扩展 `FlowNodeKind` 到 `call|compose|transform|set_var|branch|foreach|subflow`
  - 扩展 `FlowTriggerType` 到 `interval|cron|event|var_changed`
  - 为复杂节点增加 `supportsFormMode/defaultSpecEditorMode/kindDefaultSpec`
  - 保存和读取 graph 时保留 `edge.case`
  - `cron` trigger round-trip 使用 `{ type: "cron", cron }`
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - kind 选择器覆盖新增 kind
  - 为 JSON-only 节点提供明确提示，禁止误入表单路径
- `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
  - 新增 `transform/branch/foreach/subflow` 入口
- `frontend/src/components/flow/FlowNode.vue`
  - 画布节点标签覆盖新增 kind
- `frontend/src/components/flow/editor/FlowEdgeInspector.vue`
  - 新增边侧栏，最小支持 `branch` 出边 `edge.case` 编辑
- `frontend/src/windows/FlowEditorWindow.vue`
  - 接入边详情侧栏
  - 修复旧 `nodeDetailOpen` 残留引用，统一到 `detailPanelOpen`
- `frontend/src/stores/flowProjects.ts`
  - 项目 trigger draft/wire 扩展到 `cron`
  - trigger 标签格式化增加 `Cron · <expr>`
- `frontend/src/pages/Flow.vue`
  - 部署对话框支持 `cron` 选项与表达式输入
- 测试
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/stores/flowProjects.test.ts`
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
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-editor\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`

## Related lessons

- 无

## 对应 plan.md 任务映射

- `WIN-ORCH-DOC-1`
  - 更新 Win 本地 requirements/spec，明确 `cron`、新增 kind、JSON-only 边界和 `edge.case`
- `WIN-ORCH-RT-1`
  - `frontend/src/stores/flow.ts`
- `WIN-ORCH-RT-2`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
  - `frontend/src/components/flow/FlowNode.vue`
  - `frontend/src/components/flow/editor/FlowEdgeInspector.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
- `WIN-ORCH-RT-3`
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/pages/Flow.vue`
- `WIN-ORCH-TEST-1`
  - 相关前端回归测试

## 经验 / 教训摘要

- 复杂节点补 editor 支持时，先做“可识别 + 可保存 + JSON-only”，比匆忙发明半成品表单更安全。
- `branch` 不只是节点 kind 扩展，`edge.case` 也是稳定协议的一部分；如果边模型不保留它，保存会破坏路由语义。

## 可复用排查线索

- 看到新增 flow node 在 Win 编辑器里被显示成 `call` 时，优先检查：
  - `normalizeNodeKind(...)`
  - `supportsFormMode(...)`
  - `kindDefaultSpec(...)`
  - `mapNode(...)`
- 看到 `branch` 保存后分支路由失效时，优先检查：
  - `FlowEdge.case`
  - `mapEdge(...)`
  - `buildGraph()`
  - 边侧栏是否真正写回 `setSelectedEdgeCase(...)`
- 看到项目部署对话框无法保存 `cron` 时，优先检查：
  - `FlowTriggerDraft`
  - `normalizeTriggerDraft(...)`
  - `toTriggerWire(...)`
  - `Flow.vue` 的 trigger UI 分支

## 关键设计决策与权衡

- 对 `transform/branch/foreach/subflow` 先采用 JSON-only
  - 好处：不丢 spec，不假装有安全表单
  - 代价：普通用户仍需在这些节点上编辑 JSON
- `branch` 路由信息放在边上，不放回节点表单
  - 好处：与稳定协议一致，保存路径最短
  - 代价：需要新增独立 edge inspector
- `cron` 在项目部署链路中单独使用 `cronExpr`
  - 好处：前端 draft 命名与其他 camelCase 字段一致
  - 代价：需要在 wire 转换时做一次显式映射

## 测试与验证方式 / 结果

- 定向前端测试
  - 命令：
    - `npx vitest run src/stores/flow.test.ts src/stores/flowProjects.test.ts src/components/flow/editor/FlowNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：
    - 通过，`4 passed / 32 passed`
- 全量前端测试
  - 命令：
    - `npm test`
  - 结果：
    - 失败，但失败点为既有基线问题
  - 失败信息：
    - `src/stores/stream.test.ts`
    - 缺失 `../../wailsjs/runtime/runtime`
- 前端构建
  - 命令：
    - `npm run build`
  - 结果：
    - 失败，但失败点为既有基线问题
  - 失败信息：
    - `src/pages/Home.vue`
    - 缺失 `../../wailsjs/go/session/SessionService`

## 潜在影响

- Win flow 编辑器现在会把 `transform/branch/foreach/subflow` 明确当作 stable kind 处理，不再继续按 `call` 降级。
- `branch` 的出边现在有编辑入口，保存出来的 graph 会携带 `case`。
- 项目部署 trigger 现在允许 `cron`；若用户显式选择 `cron` 但表达式为空，部署阶段会收到明确错误。

## 回滚方案

- 回退以下文件即可撤销本轮能力：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
  - `frontend/src/components/flow/FlowNode.vue`
  - `frontend/src/components/flow/editor/FlowEdgeInspector.vue`
  - 相关测试与本地 requirements/spec 文档

## 子Agent执行轨迹

- 本轮未使用子Agent。
- 原因：
  - `flow.ts`、`FlowEditorWindow.vue`、`flowProjects.ts` 写集耦合明显；
  - 计划阶段已判定不适合并行拆分；
  - 当前会话未获得用户对子 Agent 的显式授权。
