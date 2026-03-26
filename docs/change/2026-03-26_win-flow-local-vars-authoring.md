# Win Flow Local Vars Authoring

## 变更背景 / 目标

- Server `flow` 协议已经把 `set_var` 和 `flow_var` 定义为正式契约，但 Win Flow 编辑器仍只支持 `call/compose` 与 `node_result/trigger/flow_meta/run_meta`。
- 用户已明确要求把 flow local var 与 `varstore` 分离，并先完成“局部变量定义 + 最小 authoring”，再继续 detail/status/schema 后续轮次。
- 本轮目标是完成 `WIN-LV-1`：在 Win Flow 编辑器中补齐 `set_var` 节点与 `flow_var` 绑定来源的最小可用 authoring。

## 具体变更内容

- 更新稳定文档：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - 明确 `set_var` 为最小 authoring，不与 `varstore` 混淆。
- 扩展前端 store / draft / spec 构造：
  - `FlowNodeKind` 新增 `set_var`
  - `FlowBindingSourceKind` 新增 `flow_var`
  - `FlowInputBindingDraft` 新增 `name`
  - `FlowNodeDraft` 新增 `setVarName`
  - `parse/build/form/json` 全链路支持 `set_var` 与 `flow_var`
- 扩展 visual form binding 类型：
  - `VisualBindingSource` 新增 `flow_var`
  - call 字段绑定支持写回 `source.kind=flow_var`
- 更新 Win 编辑器 UI：
  - Add Node 对话框新增 `Set Var`
  - Field Binding Dialog 新增 `Flow Local Var` 来源、变量名与可选路径输入
  - Inspector 支持 `set_var` 节点最小 authoring：变量名、模板、bindings
  - `compose/set_var` 的手工 binding 编辑器新增 `flow_var`
- 补充测试：
  - store round-trip 覆盖 `set_var` / `flow_var`
  - dialog / inspector 覆盖最小 UI 路径

## Requirements impact: updated

## Specs impact: updated

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\worktrees\server-local-vars-docs\docs\requirements\flow_data_dag.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\worktrees\server-local-vars-docs\docs\specs\flow.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `WIN-LV-1`
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flow_visual_form.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/stores/flow_visual_form.test.ts`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`

## 经验 / 教训摘要

- `set_var` 不需要再造一套独立编辑状态，直接复用 `compose` 的 `template + inputs` 骨架，再补一个 `name` 即可。
- `flow_var` 不能只在 call 字段弹窗里加，visual-form 类型、store draft、手工 binding 编辑器必须一起扩，否则 UI 和 graph spec 会脱节。
- flow local var 的文案必须反复强调“仅当前 run 有效”，否则很容易被误认为是 `varstore`。

## 可复用排查线索

- 症状：
  - Add Node 对话框里没有 `Set Var`
  - binding dialog 里没有 `Flow Local Var`
  - 保存 graph 后缺少 `source.name` 或 `spec.name`
- 触发条件：
  - 只改了 UI 文案，没有同步改 `flow.ts` 的 parse/build 逻辑
  - 只改了 call visual binding，没有同步改 `compose/set_var` 手工 bindings
- 关键词：
  - `set_var`
  - `flow_var`
  - `setVarName`
  - `FlowInputBindingDraft.name`
- 快速检查：
  - 检查 `frontend/src/stores/flow.ts` 是否同时改了 `normalize* / parse* / build* / setNodeKind`
  - 检查 `frontend/src/stores/flow_visual_form.ts` 是否支持 `flow_var`
  - 检查 `FlowFieldBindingDialog.vue` 与 `FlowNodeInspector.vue` 是否都暴露了 `flow_var`

## 关键设计决策与权衡

- 决策：`set_var` 采用最小 authoring，而不是 schema-driven 表单。
  - 原因：它的核心契约只有 `name + template + inputs`，继续复用现有 template/bindings 编辑骨架，改动最小也最稳。
- 决策：`compose` 和 `set_var` 共用 `composeTemplate` 前端草稿字段。
  - 原因：避免新增第三套模板状态；真正写回 spec 时再区分 `compose.template` 与 `set_var.template`。
- 决策：前端只校验 `flow_var.name` 格式，不尝试在 Win 侧推导唯一写入者。
  - 原因：祖先歧义判定属于运行时 / set 阶段契约，本轮先保证 authoring 输出正确。

## 测试与验证方式 / 结果

- `npm test`
  - 结果：通过（7 个文件，23 个用例）
- `npm run build`
  - 结果：失败，但原因为现有环境问题，与本轮改动无关
  - 现象：`Could not resolve "../../wailsjs/runtime/runtime" from "src/windows/TopicBusWindow.vue"`

## 潜在影响与回滚方案

- 潜在影响：
  - 节点类型切换时，`call` 与 `compose/set_var` 之间会做最小模板迁移；非 object 模板切回 `call/compose` 时会回落到安全默认值。
  - `flow_var` 现在进入 call visual binding 和手工 binding 两条路径，后续 detail/status 轮次需要继续沿用同一数据模型。
- 回滚方案：
  - 回退 `frontend/src/stores/flow.ts` 与 `frontend/src/stores/flow_visual_form.ts` 中的 `set_var/flow_var` 扩展
  - 回退 `FlowAddNodeDialog.vue`、`FlowFieldBindingDialog.vue`、`FlowNodeInspector.vue`、`FlowEditorWindow.vue`
  - 回退新增测试与 requirements/spec 更新

## 子Agent执行轨迹

- none
