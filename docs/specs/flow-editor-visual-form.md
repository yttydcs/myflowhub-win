# Flow Editor Visual Form Spec

## Scope

- 本规范限定 Win Flow 编辑器在 `call` 节点上的普通模式行为。
- 本规范不修改 Flow 运行时协议、DAG 校验规则或 `args_template + inputs` 的执行语义。

## Interfaces / Contracts

### 1. 普通模式适用范围

- 仅 `kind=call` 节点允许进入本规范定义的普通模式。
- `compose` 节点继续沿用现有编辑方式。

### 2. 方法 schema 解析优先级

- 编辑器按以下顺序解析普通模式 schema：
  1. 本地 override schema
  2. capability query 返回的 `input_schema`
  3. 无 schema
- 若最终无法解析出合法视觉 schema，则隐藏普通模式，只显示 `Advanced JSON`。

### 3. 能力查询要求

- `cap_query` 请求应设置 `include_schema=true`，以便前端获取 `input_schema/output_schema`。
- 前端 capability route 模型必须保留：
  - `provider_node`
  - `via_node`
  - `method`
  - `version`
  - `default_timeout_ms`
  - `permissions`
  - `tags`
  - `input_schema`
  - `output_schema`

### 4. 视觉 schema 契约

建议统一为以下结构：

```ts
type MethodFieldControl =
  | "text"
  | "textarea"
  | "number"
  | "select"
  | "switch"
  | "json"

type MethodFieldSchema = {
  key: string
  label: string
  pointer: string
  control: MethodFieldControl
  description?: string
  required?: boolean
  bindable?: boolean
  defaultValue?: unknown
  options?: Array<{ label: string; value: string | number | boolean }>
}

type MethodVisualSchema = {
  method: string
  title: string
  supportsVisualForm: boolean
  fields: MethodFieldSchema[]
}
```

### 5. 字段与底层 spec 的映射

- 每个普通模式字段对应一个固定的 JSON Pointer。
- 字段固定值写入 `args_template`。
- 字段引用写入 `inputs`，其 `to` 必须等于该字段 pointer。
- 同一个字段 pointer 在普通模式下最多允许一条 binding。
- 字段从“引用模式”切回“固定值模式”时，只删除对应 binding，不清除 `args_template` 中的 literal 值。

### 6. 引用来源契约

普通模式允许的字段引用来源：

- `node_result`
  - 必须填写 `node_id`
  - `node_id` 必须是当前节点祖先
  - `path` 可选，默认根结果
- `trigger`
  - `path` 可选
- `flow_meta`
  - 当前仅允许 `field=flow_id`
- `run_meta`
  - 当前仅允许 `field=run_id`

## Data Model or Protocol

### 1. 普通模式兼容性判定

节点满足以下条件时，才能显示普通模式：

- 节点存在且 `kind=call`
- 方法存在可解析的视觉 schema
- `args_template` 能解析为 JSON object
- 所有非空 binding 的 `to` 都落在视觉 schema 已知字段内
- 每个 schema pointer 至多对应一条 binding
- `args_template` 不含 schema 未覆盖的额外叶子字段

若任一条件不成立：

- 隐藏普通模式
- 仅显示 `Advanced JSON`

### 2. JSON Schema 子集约束

对于 capability 返回的 `input_schema`，前端仅接受受限 JSON Schema 子集：

- 顶层必须是 `type=object`
- 支持：
  - `title`
  - `description`
  - `default`
  - `required`
  - `properties`
  - 基础类型：`string` / `number` / `integer` / `boolean` / `object`
  - `enum`
- 支持的受限 vendor extension：
  - 属性级 `x-ui-control`
  - 当前支持值：`textarea`
  - 当前仅对 `type=string` 生效，用于覆盖默认单行文本控件
- 第一版不支持：
  - `oneOf`
  - `anyOf`
  - `allOf`
  - `$ref`
  - 数组驱动的复杂动态表单

出现不支持特性时：

- 视为该方法不可生成普通模式
- 仅显示 `Advanced JSON`
- 对未知或不兼容的 `x-ui-control` 值，前端必须忽略并回退到基础类型推断，不得抛出破坏性异常

### 3. 推荐前端模型

```ts
type VisualBindingSource =
  | { kind: "node_result"; nodeId: string; path: string; required: boolean }
  | { kind: "trigger"; path: string; required: boolean }
  | { kind: "flow_meta"; field: "flow_id"; required: boolean }
  | { kind: "run_meta"; field: "run_id"; required: boolean }

type FieldVisualState = {
  mode: "literal" | "binding"
  literalValue: unknown
  binding: VisualBindingSource | null
}
```

## Error Handling

- schema 解析失败：
  - 不抛出破坏性异常到 UI 主路径
  - 记录失败原因
  - 隐藏普通模式
- JSON Pointer 非法：
  - 前端立即提示，不写回无效配置
- `node_result` 非祖先引用：
  - 前端立即拦截
- 当前节点 spec 超出普通模式表达范围：
  - 给出明确原因
  - 仅保留 `Advanced JSON`

## Security / Safety

- capability 返回的 schema 只作为前端展示元数据使用，不得驱动任意代码执行。
- 不支持的 schema 特性必须被显式拒绝，不做宽松猜测。
- 普通模式不得静默删改高级模式中未覆盖的字段。

## Performance Constraints

- schema 归一化结果应按 `method + version` 缓存。
- 字段编辑只更新当前节点相关数据，不重新构建全图 state。
- 祖先集合应在图结构变更后统一重算，不在每个字段渲染时重复扫描整图。
- capability query 仍应显式触发，不在每次节点切换时自动请求。

## Related Requirements

- [flow-editor-visual-form.md](../requirements/flow-editor-visual-form.md)

## Related Changes

- [2026-03-22_win-flow-data-dag-editor.md](../change/2026-03-22_win-flow-data-dag-editor.md)
