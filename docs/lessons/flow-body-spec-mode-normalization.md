# Flow Body Spec Mode Normalization

## Summary
- `foreach.body` 会话里，支持普通模式的节点不能被 window 层二次归一化成固定 `json`，否则 UI 会看起来“已支持表单”，实际却永远只能显示 `Advanced JSON`。
- 排查 body authoring 问题时，除了看 inspector 模板本身，还必须检查 body session 的初始化和新建节点默认值是否改写了 `specEditorMode`。

## Lookup Hints
- 关键词：
  - `normalizeBodySessionSnapshot`
  - `createBodyNodeDraft`
  - `specEditorMode`
  - `foreach.body`
  - `Advanced JSON`
  - `Body node authoring`
- 快速检查：
  - 现有 body 节点从 JSON 打开后是否仍保持 `form`
  - 新建 body 节点时是否沿用了根图默认 `specEditorMode`
  - body 普通模式按钮可点击，但节点内容是否仍只渲染 JSON 区

## Symptoms
- `foreach.body` 内的 `set_var`、`transform`、`branch`、`subflow`、`compose`、`foreach` 明明已实现普通模式 UI，但窗口里仍只显示 `Advanced JSON`。
- body window 级测试里能看到节点标题，却找不到 `Add Binding` 或其它普通模式控件。
- 从 `Advanced JSON` 切回 `Form` 后仍立即回到 JSON-only 体验。

## Impact
- body editor 的高级节点 authoring 实际不可用。
- 容易误判为 inspector 模板没渲染，导致在错误层面继续排查。

## Trigger Conditions
- `FlowEditorWindow.vue` 在 body 会话初始化时覆盖 `specEditorMode`。
- `FlowEditorWindow.vue` 在 body 新建节点时把支持普通模式的节点默认写成 `json`。
- body inspector 已补表单，但 body session 仍沿用旧的 JSON-first 兼容逻辑。

## Root Cause
- body editor 的普通模式可用性不只由 `FlowBodyNodeInspector.vue` 决定，还受 body session snapshot 的默认值和归一化过程控制。
- 只要 `normalizeBodySessionSnapshot(...)` 或 `createBodyNodeDraft(...)` 里把非 `call` 节点强制设为 `json`，后续即使 inspector 支持表单，窗口层也不会进入对应分支。

## Investigation Trail
1. 先确认 `FlowBodyNodeInspector.vue` 是否已经存在对应 kind 的普通模式模板。
2. 再看 `wrapper.text()` 或窗口级 UI，确认节点是否仍停在 `Advanced JSON`。
3. 检查 `FlowEditorWindow.vue` 中：
   - `createBodyNodeDraft(...)`
   - `normalizeBodySessionSnapshot(...)`
   - `setBodySelectedNodeSpecMode(...)`
4. 如果 body binding 按钮缺失，但节点标题已正常，优先怀疑 `specEditorMode` 被 window 层重写。

## Resolution
- 删除 body 会话中“非 `call` 一律强制 `json`”的兼容逻辑。
- 让 body 新建节点和已有节点都沿用根图一致的 `specEditorMode` 默认规则。
- 用窗口级测试验证 body 普通模式控件真正可见且能落盘到父 `foreachBodyJson`。

## Prevention / Guardrails
- 每次为 body editor 新增普通模式节点时，都同时检查：
  - inspector 模板覆盖
  - body session 初始化
  - body 新建节点默认值
  - body form/json 回切
- body editor 的兼容性回退逻辑只能基于当前 spec 是否可表示，不能按 kind 粗暴强制回 `json`。

## Related Docs
- [2026-04-03_win-flow-foreach-body-advanced-authoring.md](../change/2026-04-03_win-flow-foreach-body-advanced-authoring.md)
- [2026-04-03_win-flow-foreach-body-call-authoring.md](../change/2026-04-03_win-flow-foreach-body-call-authoring.md)
