# 2026-04-03_win-flow-foreach-body-advanced-authoring

## 变更背景 / 目标

- 前一轮已经把 `foreach.body` 做成可视化子图编辑，并补齐了 body `call` 的 ordinary mode。
- 剩余明显缺口是：body 内 `compose / transform / set_var / branch / foreach / subflow` 仍停留在 JSON-first。
- 本轮目标是在不引入第二套持久化 graph 的前提下，为这些 body 节点补齐最小普通模式 authoring，并打通 body 场景下的 bindings / ancestor wiring。

## 具体变更内容

- `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - 为 body inspector 补齐 `compose / transform / set_var / branch / foreach / subflow` 的最小普通模式 UI。
  - 新增 body 场景需要的：
    - `ancestorNodeOptions`
    - `add-binding`
    - `remove-binding`
    - `binding-source-kind-change`
  - `transform` 复用顶层表达式模式 authoring。
  - `branch` 复用 case 列表和 `default_case` authoring。
  - `compose / set_var / subflow` 复用 template + bindings authoring。
  - `foreach` 复用外层字段 authoring，同时继续把嵌套 `body` 保持为 JSON 文本区。
- `frontend/src/windows/FlowEditorWindow.vue`
  - body 模式下的 add/remove binding 与 source-kind 切换已接到 body session，并在每次提交后同步回父 `foreachBodyJson`。
  - `setBodySelectedNodeSpecMode()` 不再只允许 body `call` 回切普通模式。
  - body 会话初始化与新建节点时，不再把非 `call` 节点强制回落为 `json`。
  - body `node_result` 祖先选项继续基于 body 子图拓扑，而不是根图祖先集合。
- 测试
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
  - 新增 body `set_var` 普通模式入口和 body binding 落盘验证。
- 文档
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - 稳定文档已更新为“body 非 `call` 节点支持最小普通模式，但仍保留单一 JSON 真相源与嵌套 `foreach.body` 非递归可视化边界”。
- lessons
  - 新增 `docs/lessons/flow-body-spec-mode-normalization.md`
  - 记录 body session 归一化错误覆盖 `specEditorMode` 时的典型症状和排查路径。

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `updated`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps\docs\requirements\flow-editor-visual-form.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`

## Related lessons

- `D:\project\MyFlowHub3\worktrees\feat-win-orchestrator-form-gaps\docs\lessons\flow-body-spec-mode-normalization.md`

## 对应 plan.md 任务映射

- `BODY-AUTH-DOC-1`
  - 更新 requirements/spec，明确 body 非 `call` 最小普通模式边界和嵌套 `foreach` 约束。
- `BODY-AUTH-UI-1`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
- `BODY-AUTH-WIN-1`
  - `frontend/src/windows/FlowEditorWindow.vue`
- `BODY-AUTH-TEST-1`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`

## 经验 / 教训摘要

- body editor 的 authoring 能力不只取决于 inspector 模板，还取决于 body session 初始化和新建节点默认值是否保留了正确的 `specEditorMode`。
- 在 body 场景里继续复用根图已有的最小 form contract，比抽象一个大而全的 shared inspector 更稳。

## 可复用排查线索

- 症状：
  - `foreach.body` 内高级节点明明实现了表单，但窗口里仍只显示 `Advanced JSON`
  - body window 级测试里能看到节点标题，却找不到 `Add Binding`
- 触发条件：
  - `normalizeBodySessionSnapshot(...)` 把非 `call` 节点强制设为 `json`
  - `createBodyNodeDraft(...)` 为支持普通模式的 body 节点写死 `json`
- 关键词：
  - `normalizeBodySessionSnapshot`
  - `createBodyNodeDraft`
  - `specEditorMode`
  - `foreach.body`
  - `Advanced JSON`
- 快速检查：
  - 检查 body inspector 是否真的渲染到了 `selectedNode.specEditorMode === 'form'`
  - 检查 body 会话初始化和新建节点路径是否覆盖了 `specEditorMode`

## 关键设计决策与权衡

- 继续保持 `foreachBodyJson` 作为唯一持久化真相源
  - 好处：不破坏现有保存、恢复草稿和脏状态判断
  - 代价：window 层仍需维护 body session -> parent JSON 的同步链路
- 不为 body 内嵌套 `foreach` 提供递归可视化 body 会话
  - 好处：边界清晰，避免第二套嵌套 session 状态机
  - 代价：更深层 body 仍需要以内联 JSON 维护
- 通过最小测试证明 body binding 真正落盘，而不是只验证 inspector 渲染
  - 好处：能覆盖这轮最关键的 window 级 wiring 风险
  - 代价：仍未扩展到更大范围的全量前端回归

## 测试与验证方式 / 结果

- 依赖安装
  - 命令：
    - `npm ci`
  - 原因：
    - worktree 初始缺少 `frontend/node_modules`
    - 当前环境无 `pnpm`
- 定向前端测试
  - 命令：
    - `npm test -- src/components/flow/editor/FlowBodyNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：
    - 通过，`2 passed / 11 passed`

## 潜在影响

- `foreach.body` 内的 `compose / transform / set_var / branch / foreach / subflow` 现在可进入最小普通模式，不再只能手写整段高级 JSON。
- body 场景下的 bindings/source 交互现在会真实写回父 `foreachBodyJson`。
- 嵌套 `foreach.body` 仍保持 JSON 文本边界，没有引入递归 body session。

## 回滚方案

- 回退以下文件即可撤销本轮能力：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `docs/lessons/README.md`
  - `docs/lessons/flow-body-spec-mode-normalization.md`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`

## 子Agent执行轨迹

- 本轮未使用子Agent。
