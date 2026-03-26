# Flow Editor Visual Form

## Background

- 当前 Win Flow 编辑器已经支持 `call/compose`、`args_template`、`inputs` 和 `Advanced JSON`，但普通用户仍需要直接理解底层 spec 结构。
- `call` 节点的常见配置路径仍然偏“协议编辑器”风格，不利于快速完成方法参数配置和前置节点输出引用。
- 当前 capability `input_schema` 中只要出现安全但未被前端接受的轻量写法，例如 nullable 包装，就会让普通模式不必要地退回 `Advanced JSON`。

## Goal

- 为 Win Flow 编辑器提供面向 `call` 节点的普通模式。
- 为 `set_var` 节点提供最小 authoring，允许用户在不手写完整 spec 的前提下设置局部变量名、模板和 bindings。
- 普通模式以字段表单或最小结构化编辑为主，而不是直接编辑底层 spec。
- 保留 `Advanced JSON` 作为完整能力出口。
- 在不引入完整 JSON Schema 引擎的前提下，扩展普通模式可接受的安全 schema 子集，减少仅因轻量 schema 写法导致的无谓回退。

## Scope

### Must

- `call` 节点支持普通模式字段表单。
- 普通模式支持字段固定值输入。
- 普通模式支持字段级“上游数据引用”，来源至少包含：
  - 祖先节点输出
  - trigger
  - flow meta
  - run meta
  - flow local var
- 编辑器必须支持创建 `set_var` 节点。
- `set_var` 节点必须支持在普通模式下编辑：
  - 局部变量名
  - `template`
  - `inputs`
- `compose` / `set_var` 的 binding 编辑器都必须支持 `flow_var` 作为来源。
- 普通模式编辑结果必须稳定映射回底层 `args_template + inputs`。
- `set_var` 编辑结果必须稳定映射回底层 `name + template + inputs`。
- 当方法无可用普通模式 schema，或当前节点配置超出普通模式表达范围时，直接隐藏普通模式，只显示 `Advanced JSON`。
- 架构上应支持“所有 `call` 方法”接入普通模式，而不是只为少数方法硬编码页面。
- 当前端遇到 `type=[supportedType, "null"]` 或 `["null", supportedType]` 这类 nullable 包装时，只要非 `null` 类型仍落在支持子集内，就不应仅因此退回 `Advanced JSON`。

### Optional

- 为高频方法提供更细的专用控件、文案和默认值。
- 为字段来源路径提供可视化选择，而不是只允许手写 JSON Pointer。

### Out of Scope

- 本轮不为 `compose` 节点提供新的 schema-driven 普通模式。
- 本轮不为 `set_var` 提供字段级 schema-driven 表单，只提供最小 authoring。
- 本轮不修改 Flow 运行时契约或绑定规则。
- 本轮不以 `varstore/varpool` 变量读取作为主要引用路径。

## Scenarios

- 用户选择一个 `call` 节点后，希望直接看到方法参数表单。
- 用户为某个字段填写固定值。
- 用户点击字段右侧入口，为该字段选择祖先节点输出作为来源。
- 用户希望把某个字段直接绑定到当前 flow run 内的局部变量，而不是跨 run 的 `varstore`。
- 用户新增一个 `set_var` 节点，为局部变量填写名字，并从 trigger、祖先节点结果或已有 flow local var 物化变量值。
- 用户在 `compose` 或 `set_var` 节点里手动维护 bindings，并选择 flow local var 作为来源。
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
   - `flow_var`
4. `node_result` 仅允许引用祖先节点。
5. `flow_var` 必须要求用户填写变量名，并允许可选 `path`。
6. 编辑器必须支持创建和编辑 `kind=set_var` 节点。
7. `set_var` 普通模式必须支持编辑 `name`、`template` 和 `inputs`。
8. 普通模式必须把字段值和字段引用稳定写回底层 `args_template + inputs`。
9. `set_var` 普通模式必须把编辑结果稳定写回底层 `name + template + inputs`。
10. 普通模式不得静默删除或覆盖高级配置。
11. 当节点当前 spec 不适合普通模式表达时，编辑器必须隐藏普通模式，只保留 `Advanced JSON`。
12. 方法覆盖能力必须允许：
   - 本地高频方法覆盖 schema
   - 通用 schema 驱动兜底
13. 普通模式 schema 解析必须接受“单一受支持类型 + `null`”的 nullable 包装，包括：
   - `string`
   - `number`
   - `integer`
   - `boolean`
   - `object`
14. 除 nullable 包装外的多类型 union 仍不得做宽松猜测；一旦超出支持子集，编辑器必须回退到 `Advanced JSON`。

## Non-functional Requirements

- 性能：
  - 字段编辑不应触发无意义的全图重建或重复 I/O。
  - 普通模式 schema 解析应可缓存。
- 可读性：
  - UI 术语应使用“字段引用 / 上游数据引用”，避免与 `varstore` 混淆。
  - 当来源为 flow 局部变量时，UI 必须明确标注其“仅当前 run 有效”，避免和 `varstore` 误解为同类能力。
  - 当节点为 `set_var` 时，UI 必须明确该节点写入的是当前 flow run 的局部变量，而不是持久化变量。
- 可扩展性：
  - 允许未来逐步接入更多 `call` 方法，而无需为每个方法重写一套编辑器框架。
  - `compose` / `set_var` 应共享同一套 template + bindings 编辑骨架，避免为局部变量再复制一套状态模型。
- 可维护性：
  - 普通模式不新增第二套持久化模型。
  - 高级模式与普通模式的边界必须可审计、可解释。

## Edge Cases

- 方法没有可用 schema。
- schema 存在，但字段定义不完整或非法。
- schema 使用 nullable `type` 数组，但其非 `null` 类型不在支持子集内。
- 当前节点存在 schema 未覆盖的额外字段。
- 同一个目标字段存在多条 binding。
- binding 引用了非祖先节点。
- JSON Pointer 非法。
- `flow_var.name` 为空或不合法。
- 用户把节点切换到 `set_var` 后，原有 `compose` / `call` 模板如何最小迁移。
- 用户从“引用模式”切回“固定值模式”时，应保留原 literal 默认值。

## Acceptance Criteria

1. 对支持普通模式的方法，用户可以在不手写 `args_template` 的前提下完成常规配置。
2. 用户可以为可绑定字段选择祖先节点输出、trigger 或 meta 作为来源。
3. 用户可以创建 `set_var` 节点，并通过普通模式完成局部变量名、模板和 bindings 配置。
4. 保存后的 graph spec 与当前运行时完全兼容。
5. 对不支持普通模式的方法或节点配置，编辑器直接隐藏普通模式，只显示 `Advanced JSON`。
6. 普通模式不会静默丢失高级 JSON 中的额外能力。
7. 对仅使用 nullable 包装的安全 schema，普通模式不再无谓退回 `Advanced JSON`。

## Related Specs

- [flow-editor-visual-form.md](../specs/flow-editor-visual-form.md)
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`

## Related Changes

- [2026-03-22_win-flow-data-dag-editor.md](../change/2026-03-22_win-flow-data-dag-editor.md)
