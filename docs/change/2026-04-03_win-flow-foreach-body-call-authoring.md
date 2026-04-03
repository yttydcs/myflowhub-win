# Win Flow Foreach Body Call Authoring

## 变更背景 / 目标

- 前一轮已经把 `foreach.body` 从纯 JSON 提升到可视化子图编辑。
- 剩余明显缺口是：body 内 `call` 节点仍只有 JSON-first inspector，缺少方法选择器、ordinary mode 字段编辑和字段级 binding。
- 本轮目标是在不破坏 body 会话边界的前提下，把 body `call` 节点补到和根图 `call` 接近的 authoring 体验。

## 本轮改动

- `docs/requirements/flow-editor-visual-form.md`
  - 把 body 内 `call` 节点的 ordinary mode authoring 边界补入稳定 requirements。
- `docs/specs/flow-editor-visual-form.md`
  - 把 body 会话内 `call` 的方法选择、schema-driven fields 和 binding 对话框补入稳定 specs。
- `frontend/src/stores/flow.ts`
  - 新增 `ensureCapabilityRouteLoaded(...)`，允许按 `method + providerNode` 静默补齐 capability schema。
- `frontend/src/windows/FlowEditorWindow.vue`
  - 新增 body `call` visual-form 计算逻辑。
  - 新增 body `call` 的 literal / binding 提交与 spec mode 切换。
  - 新增 body 子图祖先判定，约束 `node_result` binding 只能引用 body 祖先节点。
  - 复用现有 `FlowMethodPickerDialog` 和 `FlowFieldBindingDialog` 到 body 会话。
- `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - body `call` 现在支持：
    - `Form` / `Advanced JSON`
    - 方法选择按钮
    - schema-driven literal 字段
    - 字段级 binding 操作
  - body 内非 `call` 节点继续保持 JSON-first。
- tests
  - 新增 `frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts`
  - 更新 `frontend/src/windows/FlowEditorWindow.test.ts`

## 关键设计决策与权衡

- body `call` visual-form 继续在 `FlowEditorWindow.vue` 本地计算，而不是把 body 图塞进根 `flowStore`
  - 好处：不破坏 root graph 的 dirty、save 和 recovery 契约
  - 代价：window 层需要维护一套 body graph 祖先判定和提交逻辑
- 只把 body 内 `call` 升级为 ordinary mode
  - 好处：本轮改动集中在最明显的能力缺口上
  - 代价：body 内其它节点仍然是 JSON-first

## 测试与验证方式 / 结果

- 定向前端测试
  - 命令：
    - `npx vitest run src/components/flow/editor/FlowBodyNodeInspector.test.ts src/components/flow/editor/FlowNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts src/stores/flow.test.ts`
  - 结果：
    - 通过，`42 passed`

## 潜在影响

- body 会话里的 `call` 节点现在可以直接选择 capability，并通过 ordinary mode 编辑参数和 bindings。
- `node_result` binding 在 body 会话中不再错误复用根图祖先集合，避免非法引用穿透到外层 graph。
- 全量前端构建 / 全量测试基线问题未在本轮处理，仍受既有 `wailsjs` 生成物缺失影响。

## 回滚方案

- 回退以下文件即可撤销本轮能力：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts`

## 子Agent执行轨迹

- 本轮未使用子Agent。
