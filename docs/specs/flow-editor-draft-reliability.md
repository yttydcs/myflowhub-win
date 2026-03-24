# Flow Editor Draft Reliability Spec

## Scope

- 本规范限定 Win Flow 编辑器壳层在保存状态、退出保护、本地恢复和组件边界上的稳定技术约束。
- 本规范不定义 Flow 运行时协议，也不替代 `flow-editor-visual-form.md` 中的 visual form 契约。

## Interfaces / Contracts

### 1. Editor Shell Boundary

- `frontend/src/windows/FlowEditorWindow.vue` 负责：
  - 生命周期
  - 项目加载 / 保存
  - 快捷键
  - 顶层 toast / message 消费
  - 本地恢复协调
- 以下 UI 分区必须为独立组件：
  - `FlowEditorToolbar.vue`
  - `FlowNodeInspector.vue`
  - `FlowMethodPickerDialog.vue`
  - `FlowFieldBindingDialog.vue`
  - `FlowAddNodeDialog.vue`
- 子组件只负责：
  - UI 渲染
  - 表单输入
  - 事件上抛
- 子组件不得直接承担项目保存、本地恢复或窗口级生命周期职责。

### 2. Dirty State Contract

- dirty signature 必须只基于 graph 编辑内容：
  - `nodes`
  - `edges`
- 以下状态不得参与 dirty signature：
  - history index / history length
  - 当前 capability 查询结果
  - loading / saving UI 状态
- 允许保留 `selectedNodeIndex` / `selectedEdgeIndex` 于恢复 snapshot 中，但 dirty 判定不得仅因选择变化而变脏。

### 3. Recovery Record Contract

本地恢复记录采用按 `projectId` 隔离的 key，并至少包含：

```ts
type FlowEditorRecoveryRecord = {
  version: 1
  projectId: string
  baseSignature: string
  savedAt: string
  snapshot: FlowGraphEditorState
}
```

其中：

- `baseSignature` 表示生成该恢复草稿时对应的已保存 graph 基线。
- `savedAt` 用于恢复提示文案。
- `snapshot` 必须来源于原始 editor state，而不是验证后的持久化 graph。

### 4. Flow Store Recovery API

- `flow.ts` 必须提供：
  - `exportGraphEditorState()`
  - `loadGraphEditorState(...)`
  - `graphEditorSignature()`
- `exportGraphEditorState()` / `loadGraphEditorState(...)` 使用原始 draft nodes / edges 模型。
- `exportGraphDraft()` 继续保留为“可保存 graph”的导出接口，并要求完整校验通过后才能持久化。

### 5. Load / Restore Order

- editor window 加载 project 时，顺序必须为：
  1. 读取已保存 project graph
  2. 更新 saved baseline
  3. 检查本地恢复记录
  4. 仅在 `baseSignature` 与当前 saved baseline 一致时提示恢复
- 若恢复记录等价于当前 saved graph，直接清理，不再提示。
- 若恢复记录损坏、缺字段或无法解析，直接清理并忽略。

### 6. Save / Unload Behavior

- 手动保存使用 `exportGraphDraft()`，只保存通过校验的 graph。
- 保存成功后必须：
  - 更新 saved baseline
  - 清理当前 project 的本地恢复记录
- 页面卸载且存在 dirty graph 时，必须：
  - best-effort 写入本地恢复记录
  - 触发浏览器标准 `beforeunload` 提示

## Data Model or Protocol

### 1. Saved Baseline

editor shell 必须维护：

- `lastSavedSignature`
- `lastSavedAt`

其中：

- `lastSavedSignature` 来源于当前已保存 graph 的 content signature
- `lastSavedAt` 来源于 project store 的 `updatedAt`，若缺失则回退到当前时间

### 2. Recovery Write Policy

- 本地恢复写入必须为去抖后的 best-effort 行为。
- dirty 为 `false` 时，本地恢复记录必须被清理。
- 不允许把每次字段输入都同步为后端保存。

## Error Handling

- `localStorage` 不可用、写入失败或读取失败时：
  - 仅记录 warning
  - 不阻塞 editor 正常使用
- 恢复失败时：
  - 清理损坏记录
  - 给出明确错误提示
- 恢复草稿与当前 saved baseline 不一致时：
  - 直接丢弃该恢复记录
  - 不做模糊合并

## Security / Safety

- 本地恢复只保存在当前运行环境的本地存储，不发送到远端。
- 恢复逻辑不得绕过现有 graph save 校验；恢复后若 graph 非法，仍必须等用户修正后再手动保存。
- 退出保护必须依赖浏览器标准机制，不自定义不可预测的关闭拦截行为。

## Performance Constraints

- dirty signature 计算应复用轻量内容签名，不重复构建完整 graph spec。
- 本地恢复写入必须去抖，避免高频同步写入 local storage。
- 结构收敛后的 editor shell 不应重新把 inspector / dialog 逻辑拉回画布组件层。

## Related Requirements

- [flow-editor-draft-reliability.md](../requirements/flow-editor-draft-reliability.md)
- [flow-editor-visual-form.md](../requirements/flow-editor-visual-form.md)

## Related Changes

- [2026-03-24_win-flow-editor-shell-reliability.md](../change/2026-03-24_win-flow-editor-shell-reliability.md)
