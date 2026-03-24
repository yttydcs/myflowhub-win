# Flow Editor Accessibility Spec

## Scope

- 本规范限定 Win Flow 编辑器的表单可访问性、弹层焦点管理和基础键盘交互约束。
- 本规范不改变 Flow 协议、graph 数据模型或 Flow 画布本体的交互语义。

## Interfaces / Contracts

### 1. 表单标签契约

- editor 关键输入控件必须至少满足以下其一：
  - 使用 `label[for] -> input/select/textarea[id]`
  - 使用 `aria-labelledby`
  - 使用明确的 `aria-label`
- 对需要额外解释文字的控件，允许补充 `aria-describedby` 指向辅助说明。

### 2. Icon-only 操作契约

- icon-only 按钮必须具备稳定 accessible name。
- 如视觉上仅显示图标，可通过 `sr-only` 或 `aria-label` 暴露名称。

### 3. Overlay 焦点管理契约

- 支持焦点管理的 overlay 在打开时必须：
  - 记录当前激活元素
  - 将焦点移动到指定初始目标，或第一个可聚焦元素
- 关闭时必须尝试恢复到先前激活元素。
- 若当前 overlay 开启焦点约束，则 Tab / Shift+Tab 只能在当前 overlay 内循环。

### 4. Editor 弹层约束

- `FlowAddNodeDialog`
- `FlowMethodPickerDialog`
- `FlowFieldBindingDialog`

以上 editor 弹层必须：

- 使用 dialog 语义
- 提供 `aria-labelledby`
- 在必要时提供 `aria-describedby`
- 启用 overlay 焦点管理

### 5. Inspector 键盘约束

- `FlowNodeInspector` 必须提供可识别的区域标题语义。
- 在未打开弹层、且当前焦点不在可编辑输入中时，Escape 应允许关闭 inspector。

## Data Model or Protocol

- 本轮仅新增前端可访问性语义和焦点管理状态，不新增持久化字段。
- Overlay 焦点管理所需状态只存在于前端组件生命周期内，不写回 store。

## Error Handling

- 若 overlay 内没有可聚焦控件，允许回退到 overlay 容器自身接收焦点。
- 若恢复焦点时原始元素已卸载或不可聚焦，应静默跳过恢复，不抛出破坏性异常。

## Security / Safety

- 焦点约束只能作用于当前打开的 overlay，避免影响页面其他区域。
- 不允许为达成焦点管理而绕过现有 overlay 栈的 Escape 关闭规则。

## Performance Constraints

- 焦点管理只在 overlay 打开、关闭和 Tab 导航时工作，不应引入持续轮询。
- 标签和描述增强只能是局部 DOM 属性更新，不增加后端 I/O。

## Related Requirements

- [flow-editor-accessibility.md](../requirements/flow-editor-accessibility.md)
- [flow-editor-visual-form.md](../requirements/flow-editor-visual-form.md)

## Related Changes

- [2026-03-24_win-flow-editor-visual-form-ux.md](../change/2026-03-24_win-flow-editor-visual-form-ux.md)
