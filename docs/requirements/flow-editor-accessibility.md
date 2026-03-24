# Flow Editor Accessibility

## Background

- Win Flow 编辑器已经有基本的 editor shell、ordinary mode 和草稿可靠性，但关键表单、弹层和 inspector 仍缺少稳定的可访问性约束。
- 当前问题主要不是功能缺失，而是标签关联、焦点落点和键盘路径不够稳定，导致屏幕阅读器和纯键盘使用成本偏高。

## Goal

- 为 Win Flow 编辑器建立可维护的无障碍和键盘交互基线。
- 让普通模式表单、编辑器弹层和节点 inspector 在不改变 Flow 协议的前提下更容易被辅助技术和键盘用户使用。

## Scope

### Must

- editor 关键表单控件必须具备可编程可识别的标签或等效 accessible name。
- icon-only 操作必须具备明确 accessible name。
- editor 弹层打开后必须把焦点带入弹层，关闭后恢复到触发控件或先前焦点。
- editor 弹层在打开时必须支持稳定的键盘 Tab / Shift+Tab 路径，不让焦点意外逃出当前弹层。
- inspector 至少要支持明确的区域语义，并提供基础键盘关闭路径。

### Optional

- 对选择态按钮、卡片或切换控件补充额外状态语义。
- 对普通模式字段的说明文字补充 `aria-describedby` 级联信息。

### Out of Scope

- 本轮不做完整 WCAG 审计。
- 本轮不引入专门的无障碍组件库替换现有 UI。
- 本轮不要求为 Flow 画布交互本身建立完整键盘编辑模型。

## Scenarios

- 用户只用键盘打开 “Add Node” / “Select Capability” / “Bind Field” 弹层并完成输入。
- 用户使用屏幕阅读器浏览 inspector 表单时，能够知道输入框、开关和下拉框对应哪个字段。
- 用户在 inspector 打开时，希望用键盘快速退出当前节点详情。

## Functional Requirements

1. 关键表单输入必须能被辅助技术识别到对应标签。
2. 弹层打开时必须聚焦到首个合理输入或可操作控件。
3. 弹层关闭时必须恢复到打开前的焦点上下文。
4. 打开的弹层必须截住 Tab / Shift+Tab，保持焦点循环在当前弹层内部。
5. inspector 打开后必须具备明确的区域标题和可用的键盘关闭路径。

## Non-functional Requirements

- 改动应尽量复用现有 `Overlay` 和已拆分的 editor 组件，不重建第二套弹层体系。
- 可访问性增强不得改变 Flow graph 的保存、协议生成或 visual form 兼容性判定。
- 焦点管理和键盘处理必须避免影响未开启的页面区域。

## Edge Cases

- 弹层中没有显式输入框，只存在按钮或只读内容。
- 弹层由 backdrop click、Escape、应用按钮或业务状态切换关闭。
- inspector 中存在动态字段、binding 列表或 form/json 切换，标签和描述仍需保持可识别。

## Acceptance Criteria

1. editor 关键表单控件具备明确标签或 accessible name。
2. `Add Node`、`Select Capability`、`Bind Field` 弹层打开后焦点落点稳定，关闭后恢复焦点。
3. 弹层打开期间 Tab / Shift+Tab 不会把焦点移出当前弹层。
4. inspector 至少支持 Escape 关闭和区域标题语义。

## Related Specs

- [flow-editor-accessibility.md](../specs/flow-editor-accessibility.md)
- [flow-editor-visual-form.md](../specs/flow-editor-visual-form.md)

## Related Changes

- [2026-03-24_win-flow-editor-visual-form-ux.md](../change/2026-03-24_win-flow-editor-visual-form-ux.md)
