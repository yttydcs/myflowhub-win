# Flow Editor Visual Form

## Background

- 当前 Win Flow 编辑器已经支持 `call/compose`、`args_template`、`inputs` 和 `Advanced JSON`，但普通用户仍需要直接理解底层 spec 结构。
- `call` 节点的常见配置路径仍然偏“协议编辑器”风格，不利于快速完成方法参数配置和前置节点输出引用。

## Goal

- 为 Win Flow 编辑器提供面向 `call` 节点的普通模式。
- 普通模式以字段表单为主，而不是直接编辑 `args_template` 和 `inputs`。
- 保留 `Advanced JSON` 作为完整能力出口。

## Scope

### Must

- `call` 节点支持普通模式字段表单。
- 普通模式支持字段固定值输入。
- 普通模式支持字段级“上游数据引用”，来源至少包含：
  - 祖先节点输出
  - trigger
  - flow meta
  - run meta
- 普通模式编辑结果必须稳定映射回底层 `args_template + inputs`。
- 当方法无可用普通模式 schema，或当前节点配置超出普通模式表达范围时，直接隐藏普通模式，只显示 `Advanced JSON`。
- 架构上应支持“所有 `call` 方法”接入普通模式，而不是只为少数方法硬编码页面。

### Optional

- 为高频方法提供更细的专用控件、文案和默认值。
- 为字段来源路径提供可视化选择，而不是只允许手写 JSON Pointer。

### Out of Scope

- 本轮不为 `compose` 节点提供新的普通模式。
- 本轮不修改 Flow 运行时契约或绑定规则。
- 本轮不以 `varstore/varpool` 变量读取作为主要引用路径。

## Scenarios

- 用户选择一个 `call` 节点后，希望直接看到方法参数表单。
- 用户为某个字段填写固定值。
- 用户点击字段右侧入口，为该字段选择祖先节点输出作为来源。
- 用户遇到复杂方法或高级配置时，退回 `Advanced JSON`。
- 用户切换方法后，编辑器按该方法重新生成可编辑字段。

## Functional Requirements

1. 编辑器必须支持根据当前 `call.method` 解析普通模式字段定义。
2. 每个字段必须支持以下两类值来源：
   - 固定值
   - 引用来源
3. 引用来源必须支持以下 `source.kind`：
   - `node_result`
   - `trigger`
   - `flow_meta`
   - `run_meta`
4. `node_result` 仅允许引用祖先节点。
5. 普通模式必须把字段值和字段引用稳定写回底层 `args_template + inputs`。
6. 普通模式不得静默删除或覆盖高级配置。
7. 当节点当前 spec 不适合普通模式表达时，编辑器必须隐藏普通模式，只保留 `Advanced JSON`。
8. 方法覆盖能力必须允许：
   - 本地高频方法覆盖 schema
   - 通用 schema 驱动兜底

## Non-functional Requirements

- 性能：
  - 字段编辑不应触发无意义的全图重建或重复 I/O。
  - 普通模式 schema 解析应可缓存。
- 可读性：
  - UI 术语应使用“字段引用 / 上游数据引用”，避免与 `varstore` 混淆。
- 可扩展性：
  - 允许未来逐步接入更多 `call` 方法，而无需为每个方法重写一套编辑器框架。
- 可维护性：
  - 普通模式不新增第二套持久化模型。
  - 高级模式与普通模式的边界必须可审计、可解释。

## Edge Cases

- 方法没有可用 schema。
- schema 存在，但字段定义不完整或非法。
- 当前节点存在 schema 未覆盖的额外字段。
- 同一个目标字段存在多条 binding。
- binding 引用了非祖先节点。
- JSON Pointer 非法。
- 用户从“引用模式”切回“固定值模式”时，应保留原 literal 默认值。

## Acceptance Criteria

1. 对支持普通模式的方法，用户可以在不手写 `args_template` 的前提下完成常规配置。
2. 用户可以为可绑定字段选择祖先节点输出、trigger 或 meta 作为来源。
3. 保存后的 graph spec 与当前运行时完全兼容。
4. 对不支持普通模式的方法或节点配置，编辑器直接隐藏普通模式，只显示 `Advanced JSON`。
5. 普通模式不会静默丢失高级 JSON 中的额外能力。

## Related Specs

- [flow-editor-visual-form.md](../specs/flow-editor-visual-form.md)
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`

## Related Changes

- [2026-03-22_win-flow-data-dag-editor.md](../change/2026-03-22_win-flow-data-dag-editor.md)
