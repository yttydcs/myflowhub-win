# Flow Editor Visual Form

## Background

- 当前 Win Flow 编辑器已经支持 `call/compose`、`args_template`、`inputs` 和 `Advanced JSON`，但普通用户仍需要直接理解底层 spec 结构。
- `call` 节点的常见配置路径仍然偏“协议编辑器”风格，不利于快速完成方法参数配置和前置节点输出引用。
- 当前 capability `input_schema` 中只要出现安全但未被前端接受的轻量写法，例如 nullable 包装，就会让普通模式不必要地退回 `Advanced JSON`。
- 稳定 flow 能力已经扩展到 `cron` trigger 与 `transform/branch/foreach/subflow` 节点，但 Win 编辑器此前仍会遗漏这些能力的识别、创建或安全保存路径。

## Goal

- 为 Win Flow 编辑器提供面向 `call` 节点的普通模式。
- 为 `set_var` 节点提供最小 authoring，允许用户在不手写完整 spec 的前提下设置局部变量名、模板和 bindings。
- 为 flow 项目部署提供 `cron` trigger authoring，而不是要求用户手写 trigger JSON。
- 为 `transform/branch/subflow` 提供最小但可用的普通模式 authoring，并为 `foreach` 提供外层字段的部分普通模式与 `body` 子图可视化 authoring 入口。
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
- 编辑器必须支持 `trigger.type=cron` 的读取、编辑和保存。
- `set_var` 节点必须支持在普通模式下编辑：
  - 局部变量名
  - `template`
  - `inputs`
- 编辑器必须支持创建以下新增节点：
  - `transform`
  - `branch`
  - `foreach`
  - `subflow`
- `transform` 当前必须支持普通模式 authoring，至少覆盖顶层表达式模式：
  - `literal`
  - `source`
  - `op + args`
  - `object`
  - `array`
- `branch` 当前必须支持普通模式 authoring，至少覆盖：
  - `cases[].name`
  - `cases[].match.source`
  - `cases[].match.op`
  - `cases[].match.value`
  - `default_case`
- `subflow` 当前必须支持普通模式 authoring，至少覆盖：
  - `flow_id`
  - `input_template`
  - `inputs`
  - `result_node_id`
- `foreach` 当前必须支持外层字段的普通模式 authoring，至少覆盖：
  - `source`
  - `required`
  - `body`
  - `result_node_id`
- `foreach` 节点必须提供显式的 `body` 子图可视化编辑入口。
- `foreach.body` 的可视化编辑必须至少支持：
  - 进入/退出 body 编辑会话
  - body 内节点与边的新增、选中、移动、连接、删除
  - body 内 `call/compose/transform/set_var/branch/foreach/subflow` 节点的最小 form/json spec 编辑
- `foreach.body` 内 `kind=call` 的节点必须支持与根图 `call` 节点一致的最小表单化 authoring，包括：
  - 方法选择器
  - 字段级 literal 编辑
  - 字段级 binding 对话框
- `foreach.body` 内 `compose/set_var/subflow` 节点必须支持最小 template + bindings authoring。
- `foreach.body` 内 `transform` 节点必须支持顶层表达式模式的最小普通模式 authoring。
- `foreach.body` 内 `branch` 节点必须支持 case 列表与 `default_case` 的最小普通模式 authoring。
- `foreach.body` 内 `foreach` 节点必须支持外层字段普通模式 authoring，但其嵌套 `body` 继续以内联 JSON 文本维护。
- `foreach.body` 的可视化编辑不能引入第二套持久化 graph 真相源；所有变更都必须同步回父节点的 `body` JSON。
- `branch` 的出边必须支持最小 `edge.case` 编辑，且保存时不得丢失。
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
- 本轮不为 `transform` 提供完整递归表达式树可视化编辑器，只覆盖顶层模式。
- 本轮不为 body 内嵌套 `foreach` 提供递归可视化 body 会话；其 `body` 继续以内联 JSON 文本维护。
- 本轮不修改 Flow 运行时契约或绑定规则。
- 本轮不以 `varstore/varpool` 变量读取作为主要引用路径。

## Scenarios

- 用户选择一个 `call` 节点后，希望直接看到方法参数表单。
- 用户为某个字段填写固定值。
- 用户点击字段右侧入口，为该字段选择祖先节点输出作为来源。
- 用户希望把某个字段直接绑定到当前 flow run 内的局部变量，而不是跨 run 的 `varstore`。
- 用户新增一个 `set_var` 节点，为局部变量填写名字，并从 trigger、祖先节点结果或已有 flow local var 物化变量值。
- 用户在 `compose` 或 `set_var` 节点里手动维护 bindings，并选择 flow local var 作为来源。
- 用户在项目部署对话框中把 trigger 改为 `cron`，填写表达式后直接部署到目标节点。
- 用户读取已有 `transform/branch/subflow` flow 时，可以在普通模式下直接维护关键字段，而不是被迫手写整段 spec。
- 用户读取已有 `foreach` flow 时，可以在普通模式下维护外层字段，并进入显式 body 编辑会话可视化维护内部 DAG。
- 用户在 `foreach.body` 里选中 `call` 节点后，可以直接选择 capability 方法，并为字段填写 literal 或绑定上游来源。
- 用户在 `foreach.body` 里选中 `transform/branch/set_var/subflow/compose` 节点后，可以直接在普通模式下维护关键字段，而不是回到整段父节点 JSON。
- 用户在 `foreach.body` 里选中嵌套 `foreach` 节点后，可以继续维护外层字段，但其内层 `body` 仍保持 JSON 文本边界。
- 用户为 `branch` 节点的出边填写 `edge.case`，保存后路由语义保持不变。
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
15. 项目部署 trigger 编辑器必须支持 `interval`、`cron`、`event`、`var_changed`。
16. 当 trigger 为 `cron` 时，编辑器必须写回 `{ type: "cron", cron }`，且空表达式在严格保存路径中必须显式报错。
17. Add Node 对话框、节点 inspector 和画布节点标签必须覆盖 `transform/branch/foreach/subflow`。
18. `transform/branch/subflow` 在当前版本必须允许进入普通模式；当已有 spec 超出普通模式覆盖范围时，必须显式保留在 `Advanced JSON`。
19. `branch` 的边编辑器必须允许最小编辑 `edge.case`，并在 graph round-trip 中稳定保留。
20. `foreach` 在当前版本必须允许进入部分普通模式；当已有 spec 超出当前普通模式覆盖范围时，必须显式保留在 `Advanced JSON`。
21. `foreach` 普通模式必须保留 `body` JSON 真相源，并提供显式的 body 子图可视化编辑会话。
22. body 编辑会话中的每次图结构变更都必须同步回父 `foreach` 节点的 `body` JSON，使保存、恢复草稿和脏状态判断继续以根 graph 为准。
23. body 编辑会话必须允许 `call/compose/transform/set_var/branch/foreach/subflow` 节点使用最小 form/json inspector 编辑 spec，不能要求用户直接回到整段父节点 JSON 才能改内部节点。
24. body 编辑会话中的 `compose/set_var/subflow` 普通模式必须稳定写回各自的 template/input bindings 草稿。
25. body 编辑会话中的 `transform` 普通模式必须稳定写回顶层表达式模式与 source/op/object/array 对应草稿。
26. body 编辑会话中的 `branch` 普通模式必须稳定写回 case 列表与 `default_case` 草稿。
27. body 编辑会话中的 `foreach` 普通模式必须稳定写回外层字段，同时保留其内层 `body` JSON 文本边界。
28. `transform/branch/subflow/foreach` 在普通模式下保存时不得静默删除额外高级字段；若当前 spec 超出普通模式覆盖范围，必须要求回退 `Advanced JSON`。
29. body 编辑会话中的 `call` 节点必须允许打开方法选择器，并把选中的 `method + target` 稳定写回 body 节点草稿。
30. body 编辑会话中的 `call` 字段编辑必须复用与根图相同的 schema-driven ordinary mode 规则，稳定写回 `args_template + inputs`。
31. body 编辑会话中的 bindings/source 校验必须继续遵守现有来源契约，其中 `node_result` 仅允许引用当前 body 子图内的祖先节点。

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
  - `subflow` 应尽量复用 `compose` / `set_var` 的 template + bindings 编辑骨架，避免再复制第三套输入物化模型。
- `foreach` 继续把 `body` 保留为 JSON 真相源，为未来 body 子图编辑器保留边界。
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
- `cron` 触发表达式为空。
- JSON Pointer 非法。
- `flow_var.name` 为空或不合法。
- `branch` 出边未保留 `edge.case` 导致路由语义丢失。
- `branch.default_case` 指向不存在 case。
- `transform.expr` 若超出当前顶层模式支持范围，切回普通模式必须显式失败。
- `subflow.flow_id` 非 UUID。
- `foreach.body` 若被错误表单化展开，会造成嵌套 graph 丢失。
- `foreach.body` 若在可视化编辑期间脱离父节点 JSON 真相源，保存、恢复草稿或撤销重做会丢失内层改动。
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
8. 项目部署可读取并保存 `cron` trigger。
9. 编辑器可在普通模式下读取并保存 `transform/branch/subflow` 节点。
10. 编辑器可在普通模式下读取并保存 `foreach` 的外层字段，并保持 `body` 子图不丢失。
11. 编辑器可通过显式 body 编辑会话可视化维护 `foreach.body` 的内部 DAG，并在保存时写回父节点 `body`。
12. `branch` 的 `edge.case` 在读取、编辑、保存后保持不丢失。
13. 编辑器可在 `foreach.body` 中以普通模式读取并保存 `compose/transform/set_var/branch/foreach/subflow` 的最小支持字段。

## Related Specs

- [flow-editor-visual-form.md](../specs/flow-editor-visual-form.md)
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`

## Related Changes

- [2026-03-22_win-flow-data-dag-editor.md](../change/2026-03-22_win-flow-data-dag-editor.md)
