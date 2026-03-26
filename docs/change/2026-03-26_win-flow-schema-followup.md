# Win Flow Schema Follow-up

## 变更背景 / 目标

- `WIN-LV-1`、`WIN-RD-1`、`WIN-ST-1` 完成后，Win Flow 编辑器已经具备 local vars、detail 和 status 的基础链路，但 schema 消费仍停在第一版边界。
- 输入侧只要 capability `input_schema` 使用了安全但未被接受的 nullable 包装，普通模式就会无谓退回 `Advanced JSON`。
- 输出侧 detail 面板虽然已经接通 `output_schema`，但仍只展示原始 JSON 文本，无法帮助用户快速读取根结果关键字段。
- 本轮目标是完成 `WIN-SC-1`：在不扩后端协议、不引入完整 JSON Schema 引擎的前提下，扩展受限 schema 子集消费，并把 `output_schema` 用到 root result 的只读结构化展示中。

## 具体变更内容

- 更新稳定文档：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `docs/requirements/flow-editor-run-detail.md`
  - `docs/specs/flow-editor-run-detail.md`
  - 明确本轮新增两条稳定约束：
    - 普通模式接受“单一受支持类型 + null”的 nullable schema 包装
    - `call` 节点根结果可按受限 `output_schema` 提供只读结构化视图，并保留 raw JSON
- 完成 `WIN-SC-1A`：
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow_schema_resolver.test.ts`
  - 新增 `normalizeSchemaType(...)`，让 `T`、`[T,"null"]`、`["null",T]` 共用同一条解析路径
  - 保持原回退边界不变：数组、`oneOf/anyOf/allOf/$ref`、以及任意其他多类型 union 继续返回 `null`
- 完成 `WIN-SC-1B`：
  - `frontend/src/stores/flow.ts`
    - `FlowNodeDetailState` 新增 `resultValue`
    - 新增 `buildDetailStructuredFields(...)`
    - detail 成功后同时保留原始 `resultValue` 和格式化 `resultText`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
    - 在 root result 且 schema 受支持时展示 `Structured Result`
    - 继续保留原始 `Result` 文本和 `Output schema` 文本
    - 非根路径、schema 不支持、结果不兼容时自动回退到纯文本视图
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/i18n/messages/automation.ts`
  - 补齐结构化结果渲染与回退场景测试

## Requirements impact: updated

## Specs impact: updated

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\requirements\flow-editor-run-detail.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\worktrees\win-local-vars-ui\docs\specs\flow-editor-run-detail.md`
- `D:\project\MyFlowHub3\worktrees\server-local-vars-docs\docs\specs\flow.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `WIN-SC-1`
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `docs/requirements/flow-editor-run-detail.md`
  - `docs/specs/flow-editor-run-detail.md`
- `WIN-SC-1A`
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow_schema_resolver.test.ts`
- `WIN-SC-1B`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/i18n/messages/automation.ts`

## 经验 / 教训摘要

- nullable 支持必须收敛成“单一受支持类型 + null”的兼容层，不能被扩展成任意 union 推断，否则普通模式边界会很快失控。
- 结构化结果视图只能是 detail 的辅助阅读层，raw JSON 必须始终可见，否则一旦 schema 与真实 payload 漂移，用户会被误导。
- root-only 的结构化结果展示是本轮合理边界；非根路径如果没有子 schema 推导，就应该直接回退文本，而不是做不可靠猜测。

## 可复用排查线索

- 症状：
  - 某些方法明明只有 nullable 字段，普通模式却仍然直接隐藏
  - `call` 节点 detail 已有 `output_schema`，但 inspector 里只有 raw JSON
  - 查询根结果时没有出现 `Structured Result`
- 触发条件：
  - resolver 没有识别 `type=[T,"null"]` / `["null",T]`
  - `FlowNodeDetailState` 只保留 `resultText`，没有原始 `resultValue`
  - 当前 detail 查询的是非根路径，结构化视图按设计会禁用
- 关键词：
  - `normalizeSchemaType`
  - `buildDetailStructuredFields`
  - `resultValue`
  - `Structured Result`
  - `output_schema`
- 快速检查：
  - 检查 `frontend/src/stores/flow_schema_resolver.ts` 是否仍把 nullable type array 直接判成 unsupported
  - 检查 `frontend/src/stores/flow.ts` 的 detail state 是否包含 `resultValue`
  - 检查当前 detail `path` 是否为空；非根路径不会启用结构化视图
  - 检查 capability route 是否真的带回了 object-shaped `output_schema`

## 关键设计决策与权衡

- 决策：输入侧只支持 nullable 包装，不支持其他 union。
  - 原因：这是最小安全扩展，能覆盖一批真实 schema，同时不把普通模式变成宽松猜测器。
- 决策：输出侧结构化展示只覆盖 root result。
  - 原因：当前没有稳定的子 schema 推导链路，限制在根结果能保持 schema/result 映射诚实。
- 决策：detail store 显式保存 `resultValue`，而不是每次从 `resultText` 反解析。
  - 原因：结构化展示需要保留原始 JSON 值形态，直接存原值更稳，也更利于后续扩展。

## 测试与验证方式 / 结果

- `npm test -- src/stores/flow_schema_resolver.test.ts src/stores/flow.test.ts src/components/flow/editor/FlowNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：通过（4 个文件，28 个用例）
- 子任务定向验证：
  - `npm test -- src/stores/flow_schema_resolver.test.ts`
  - 结果：通过（1 个文件，5 个用例）

## 潜在影响与回滚方案

- 潜在影响：
  - 当前 structured result 只覆盖 root result；非根路径仍是 raw JSON-only
  - 数组、组合 schema、复杂 union 仍然不会进入普通模式或结构化结果视图
- 回滚方案：
  - 回退 `frontend/src/stores/flow_schema_resolver.ts` 与对应测试
  - 回退 `frontend/src/stores/flow.ts` 的 `resultValue` 和 `buildDetailStructuredFields(...)`
  - 回退 `FlowNodeInspector.vue`、`FlowNodeInspector.test.ts`、`flow.test.ts` 和相关文案
  - 如需完全回退稳定约束，再回退上述 requirements/specs 更新

## 子Agent执行轨迹

- `/root/win_sc_1a_resolver`
  - 负责文件：
    - `frontend/src/stores/flow_schema_resolver.ts`
    - `frontend/src/stores/flow_schema_resolver.test.ts`
  - 结果：
    - 完成 nullable schema resolver follow-up
    - 维持 unsupported schema fallback 边界
    - 子任务内执行 `npm test -- src/stores/flow_schema_resolver.test.ts` 通过
