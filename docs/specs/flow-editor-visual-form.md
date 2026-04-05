# Flow Editor Visual Form Spec

## Scope

- 本规范限定 Win Flow 编辑器在 `call`、`transform`、`branch`、`subflow` 节点上的普通模式行为、`set_var` 节点的最小 authoring 行为，以及 `foreach` 的部分普通模式与 body 子图编辑会话边界。
- 本规范同时覆盖 `foreach.body` 会话内 `call/compose/transform/set_var/branch/foreach/subflow` 的最小 form/json authoring 契约。
- 本规范不修改 Flow 运行时协议、DAG 校验规则或 `args_template + inputs` 的执行语义。
- 本规范同时覆盖项目部署对话框中的 trigger authoring，以及 `branch` 出边 `edge.case` 的最小编辑契约。
- 本规范同时覆盖 node-level `retry_backoff_ms`、flow-level `max_active_runs`，以及 trigger `dedup_window_ms` 的 Win-side authoring 契约。

## Canonical Contract Consumption Boundary

- Win 侧 `FlowNodeKind`、`FlowBindingSourceKind`、`FlowBranchMatchOp` 的 canonical truth 必须消费由 Proto 同步进来的 `frontend/src/generated/flow_contract.ts`。
- Win 不再本地重写完整的 kind/source/op unions 或 literal option arrays；若 Proto contract 变更，应先同步 generated artifact，再更新 Win 的 draft 投影与 UI 行为。
- Win 仍然拥有以下本地职责：
  - draft state、表单字段、标签文案、普通模式映射
  - root graph 与 `foreach.body` 对 source kind 的可见性过滤，例如 root 不暴露 `loop_item` / `loop_index`
  - strict export / ordinary-mode fallback 的前端校验与错误提示
- Proto generated artifact 只提供 canonical contract 产物，不承载 Win 的 UI draft model，也不替代本规范中的 ordinary-mode 边界定义。

## Interfaces / Contracts

### 0. Trigger authoring 契约

- flow 项目部署对话框必须支持以下 trigger 类型：
  - `interval`
  - `cron`
  - `event`
  - `var_changed`
- `cron` 使用稳定 wire 结构：

```json
{
  "type": "cron",
  "cron": "0 */5 * * *"
}
```

- `cron` 当前不扩展 timezone 字段。
- 严格保存路径中，空 `cron` 表达式必须显式报错，不得静默回退到 `interval`。
- `event` / `var_changed` trigger 允许 authoring `dedup_window_ms`；`interval` / `cron` 在严格保存路径中若 `dedup_window_ms > 0` 必须显式失败。
- flow / project metadata 必须支持 `max_active_runs`；本地草稿中必须保留“未设置”和 `0` 的差异，导出 wire 时仅在非 `null` 时写入 `max_active_runs`。

### 1. 普通模式适用范围

- `call` 节点允许进入 schema-driven 普通模式。
- `compose` 节点继续沿用现有 template + bindings 编辑方式。
- `transform` 节点允许进入“顶层表达式模式 + JSON 局部编辑”的普通模式。
- `set_var` 节点使用最小 authoring 模式，而不是 schema-driven 字段表单。
- `branch` 节点允许进入 case 列表普通模式。
- `subflow` 节点允许进入 `flow_id + input_template + inputs + result_node_id` 普通模式。
- `foreach` 节点允许进入外层字段的部分普通模式，并提供显式的 body 子图编辑会话。

### 2. 节点类型 authoring 契约

- Add Node 对话框必须支持：
  - `call`
  - `compose`
  - `set_var`
  - `transform`
  - `branch`
  - `foreach`
  - `subflow`
- Inspector 的 `kind` 选择器必须支持上述节点。
- 所有根图与 body inspector 必须暴露 `retry_backoff_ms`，并把它稳定映射到 node wire 的 `retry_backoff_ms`。
- 节点切换时沿用现有最小迁移策略：
  - `compose` / `set_var` 共享 `template + inputs`
  - `call` 与 `compose` / `set_var` 切换时，尽量复用已有 JSON 模板内容
- 新建 `transform`、`branch`、`foreach`、`subflow` 节点时，默认 `specEditorMode` 必须为 `form`。
- 在 body 编辑会话中新建 `call/compose/transform/set_var/branch/foreach/subflow` 节点时，默认 `specEditorMode` 必须与根图保持一致；支持普通模式的节点默认进入 `form`。
- 读取已有 payload 时，仅当 `transform`、`branch`、`foreach`、`subflow` 的 spec 处于当前普通模式支持子集内，才允许保持或切回 `form`；否则必须回退 `json`。
- `set_var` 节点在普通模式下至少暴露：
  - `name`
  - `template`
  - `inputs`
- `set_var` 的 `template` 可为任意合法 JSON 值，不限制为 object。
- `transform` 节点在普通模式下至少暴露：
  - `expr` 顶层模式选择
  - `literal`
  - `source`
  - `op + args`
  - `object`
  - `array`
- `branch` 节点在普通模式下至少暴露：
  - `cases[].name`
  - `cases[].match.source`
  - `cases[].match.op`
  - `cases[].match.value`
  - `default_case`
- `subflow` 节点在普通模式下至少暴露：
  - `flow_id`
  - `input_template`
  - `inputs`
  - `result_node_id`
- `foreach` 节点在普通模式下至少暴露：
  - `source`
  - `required`
  - `body`（JSON 文本区）
  - `result_node_id`
- `foreach` 节点在普通模式下必须提供 “Open Visual Body Editor” 之类的显式入口。
- `branch` 节点的分支路由不写在节点表单里，而是由出边 `edge.case` 承载；编辑器必须提供最小 edge inspector 来编辑该字段。

### 3. 方法 schema 解析优先级

- 编辑器按以下顺序解析普通模式 schema：
  1. 本地 override schema
  2. capability query 返回的 `input_schema`
  3. 无 schema
- 若最终无法解析出合法视觉 schema，则隐藏普通模式，只显示 `Advanced JSON`。

### 4. 能力查询要求

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

### 5. 视觉 schema 契约

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

### 6. 字段与底层 spec 的映射

- 每个普通模式字段对应一个固定的 JSON Pointer。
- 字段固定值写入 `args_template`。
- 字段引用写入 `inputs`，其 `to` 必须等于该字段 pointer。
- 同一个字段 pointer 在普通模式下最多允许一条 binding。
- 字段从“引用模式”切回“固定值模式”时，只删除对应 binding，不清除 `args_template` 中的 literal 值。

### 6.1 高级节点普通模式映射契约

- `transform` 普通模式只覆盖顶层表达式变体：
  - `literal`
  - `source`
  - `op + args`
  - `object`
  - `array`
- `transform` 不在本轮实现完整递归表达式树编辑器；嵌套表达式继续通过 JSON 文本区输入。
- `branch` 普通模式只维护节点内的 case 列表和 `default_case`；`edge.case` 继续由边 inspector 维护。
- `branch` 必须校验：
  - case 名称非空
  - case 名称在同一节点内唯一
  - `default_case` 指向已有 case
- `foreach` 普通模式必须校验：
  - `source` 复用既有 source 契约
  - `body` 是 JSON object
  - `body.nodes` / `body.edges` 是数组
  - `result_node_id` 非空
  - 严格保存时 `body` 必须递归通过 DAG 校验，且 `result_node_id` 必须存在于 body graph
- `subflow` 普通模式必须校验：
  - `flow_id` 是 UUID
  - `input_template` 是 JSON object
  - 若当前 flow_id 已知，则 `flow_id` 不得等于当前 flow_id
- 当 `transform`、`branch`、`foreach`、`subflow` 的高级 JSON 含有当前普通模式无法表达的额外字段时，切回普通模式必须显式失败。

### 6.2 Foreach body 保留契约

- `foreach.body` 以 JSON 文本区为真相源。
- 读取已有 payload 时，编辑器必须按原 kind 建模，不得降级成 `call`。
- 保存时，前端必须保留 `body` object，并仅补 `_ui` 坐标信息。
- `foreach.body` 必须作为嵌套 graph 原样保留，不做结构化拆散。
- 编辑器允许进入显式的 body 子图编辑会话，但该会话只是 `foreachBodyJson` 的 UI 投影，不是新的持久化 graph 状态。
- body 编辑会话必须复用现有 graph draft 结构：
  - `nodes`
  - `edges`
- body 编辑会话中的新增、移动、连线、删除和最小节点 JSON 编辑，必须在每次提交后立即同步回父节点 `foreachBodyJson`。
- 根 graph 的保存、脏状态和本地恢复草稿，继续以外层 editor state 为准；body 会话不能绕开父节点 JSON 直接单独持久化。
- body 编辑会话对以下节点提供最小 ordinary mode：
  - `call`
  - `compose`
  - `transform`
  - `set_var`
  - `branch`
  - `foreach`
  - `subflow`
- body 会话中的 `call` ordinary mode 继续提供：
  - 方法选择器
  - schema-driven literal fields
  - 字段级 binding 对话框
- body 会话中的 `compose/set_var/subflow` ordinary mode 继续复用 template + bindings 编辑骨架。
- body 会话中的 `transform` ordinary mode 继续只覆盖顶层表达式模式，不扩展到递归表达式树可视化。
- body 会话中的 `branch` ordinary mode 继续复用 case 列表与 `default_case` 编辑契约。
- body 会话中的 `foreach` ordinary mode 只覆盖外层字段；其嵌套 `body` 继续以内联 JSON 文本维护，不提供递归可视化 body 会话。
- body 会话中的 `node_result` source/binding 祖先集合必须基于当前 body 子图拓扑计算，不能引用根图无关节点。
- body 会话中的 source / binding 编辑必须额外暴露：
  - `loop_item`
  - `loop_index`
- `loop_item` / `loop_index` 只允许出现在 `foreach.body`；根图 editor 和 field-binding dialog 不得暴露它们。
- `loop_index` 不接受 `path`；若高级 JSON 携带非空 `path`，必须回退到 `Advanced JSON` 或在严格保存时显式失败。

### 7. 引用来源契约

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
- `flow_var`
  - 必须填写 `name`
  - `path` 可选，默认根变量值
  - UI 需明确提示其语义为“当前 flow run 的局部变量”，不是 `varstore`
- `loop_item`
  - 仅允许在 `foreach.body` 内出现
  - `path` 可选，默认当前 item 根值
- `loop_index`
  - 仅允许在 `foreach.body` 内出现
  - 不接受 `path`

`flow_var` 不仅用于 `call` 字段级绑定，也必须用于 `compose` / `set_var` 的手工 binding 编辑器。

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

### 1.1 `set_var` authoring 判定

`set_var` 节点不依赖 capability schema，也不复用 `call` 普通模式兼容性判定。其普通模式始终基于最小 authoring 结构：

- `name` 文本输入
- `template` JSON 文本区
- `inputs` 列表

若用户需要超出该结构的能力，仍可退回 `Advanced JSON`。

### 1.2 `foreach` 普通模式判定

`foreach` 节点不复用 `call` 的 schema-driven 普通模式，而是采用固定字段的部分普通模式：

- `source`
- `required`
- `body` JSON 文本区
- `result_node_id`

仅当 `foreach` spec 满足以下条件时，才允许显示普通模式：

- 顶层字段只包含 `source`、`required`、`body`、`result_node_id`、`_ui`
- `source` 是合法 object，且可映射到现有 source draft
- `body` 是 JSON object，且至少包含 `nodes` / `edges` 数组

否则：

- 回退到 `Advanced JSON`
- 不得把 `foreach` 误复用到 `compose` / `set_var` / `subflow` 表单分支

### 2. JSON Schema 子集约束

对于 capability 返回的 `input_schema`，前端仅接受受限 JSON Schema 子集：

- 顶层必须是 `type=object`，或 `type=["object","null"]` / `["null","object"]`
- 支持：
  - `title`
  - `description`
  - `default`
  - `required`
  - `properties`
  - 基础类型：`string` / `number` / `integer` / `boolean` / `object`
  - nullable 包装：
    - `type=[T, "null"]`
    - `type=["null", T]`
    - 其中 `T` 仅允许上述受支持基础类型
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
  - 除“单一受支持类型 + `null`”之外的多类型 union

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
  | { kind: "loop_item"; path: string; required: boolean }
  | { kind: "loop_index"; required: boolean }
  | { kind: "flow_var"; name: string; path: string; required: boolean }

type FieldVisualState = {
  mode: "literal" | "binding"
  literalValue: unknown
  binding: VisualBindingSource | null
}

type FlowInputBindingDraft = {
  to: string
  sourceKind: "" | "node_result" | "trigger" | "flow_meta" | "run_meta" | "loop_item" | "loop_index" | "flow_var"
  nodeId: string
  path: string
  field: string
  name: string
  required: boolean
}

type FlowNodeDraft = {
  kind: "call" | "compose" | "transform" | "set_var" | "branch" | "foreach" | "subflow"
  method: string
  target: number
  retryBackoffMs: number
  argsTemplate: string
  composeTemplate: string
  setVarName: string
  inputs: FlowInputBindingDraft[]
  transformExprMode: "literal" | "source" | "op" | "object" | "array"
  transformLiteralJson: string
  transformSource: {
    sourceKind: "" | "node_result" | "trigger" | "flow_meta" | "run_meta" | "loop_item" | "loop_index" | "flow_var"
    nodeId: string
    path: string
    field: string
    name: string
  }
  transformSourceRequired: boolean
  transformOp: string
  transformArgsJson: string
  transformObjectJson: string
  transformArrayJson: string
  branchCases: Array<{
    key: string
    name: string
    source: {
      sourceKind: "" | "node_result" | "trigger" | "flow_meta" | "run_meta" | "loop_item" | "loop_index" | "flow_var"
      nodeId: string
      path: string
      field: string
      name: string
    }
    op: "eq" | "ne" | "gt" | "gte" | "lt" | "lte" | "exists"
    valueJson: string
  }>
  branchDefaultCase: string
  foreachSource: {
    sourceKind: "" | "node_result" | "trigger" | "flow_meta" | "run_meta" | "loop_item" | "loop_index" | "flow_var"
    nodeId: string
    path: string
    field: string
    name: string
  }
  foreachRequired: boolean
  foreachBodyJson: string
  foreachResultNodeId: string
  subflowId: string
  subflowInputTemplate: string
  subflowResultNodeId: string
}

type FlowTriggerDraft = {
  type: "interval" | "cron" | "event" | "var_changed"
  everyMs: number
  cronExpr: string
  eventMode: "publish" | "received" | "any"
  eventName: string
  eventTopic: string
  dedupWindowMs: number
  varOwner: number
  varName: string
}

type FlowEdgeDraft = {
  from: string
  to: string
  case?: string
}
```

### 4. Trigger / Edge wire mapping

- flow metadata 在保存时必须支持：
  - `max_active_runs`
  - `null` 表示省略该字段
  - `0` 必须被稳定保留为 `max_active_runs: 0`
- node metadata 在保存时必须支持：
  - `retry_backoff_ms`
- `interval` 继续映射为 `{ type: "interval", every_ms }`
- `cron` 映射为 `{ type: "cron", cron }`
- `event` 继续映射为 `{ type: "event", event_mode, event_name?, event_topic?, dedup_window_ms? }`
- `var_changed` 继续映射为 `{ type: "var_changed", var_owner?, var_name?, dedup_window_ms? }`
- `interval` / `cron` 在严格保存路径中不得导出 `dedup_window_ms > 0`
- graph edge 在保存时必须保留 `case`：

```json
{
  "from": "branch1",
  "to": "sub1",
  "case": "approved"
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
- `loop_item` / `loop_index` 出现在根图，或 `loop_index` 携带 `path`：
  - 前端立即拦截或回退 `Advanced JSON`
- `flow_var.name` 为空：
  - 前端立即提示，不写回无效配置
- `retry_backoff_ms`、`max_active_runs` 或 `dedup_window_ms` 非法：
  - 前端立即提示，不写回无效配置
- `subflow.flow_id` 直接指向当前 flow，或当前本地项目图可判定递归链：
  - 前端在保存 / 部署前显式失败
- 当前节点 spec 超出普通模式表达范围：
  - 给出明确原因
  - 仅保留 `Advanced JSON`

## Security / Safety

- capability 返回的 schema 只作为前端展示元数据使用，不得驱动任意代码执行。
- 不支持的 schema 特性必须被显式拒绝，不做宽松猜测。
- 普通模式不得静默删改高级模式中未覆盖的字段。
- nullable 包装只能作为单一基础类型的只读兼容层使用，不得被扩展成任意 union 推断。

## Performance Constraints

- schema 归一化结果应按 `method + version` 缓存。
- 字段编辑只更新当前节点相关数据，不重新构建全图 state。
- 祖先集合应在图结构变更后统一重算，不在每个字段渲染时重复扫描整图。
- capability query 仍应显式触发，不在每次节点切换时自动请求。

## Related Requirements

- [flow-editor-visual-form.md](../requirements/flow-editor-visual-form.md)

## Related Changes

- [2026-03-22_win-flow-data-dag-editor.md](../change/2026-03-22_win-flow-data-dag-editor.md)
- [2026-04-04_win-flow-p0-authoring-closure.md](../change/2026-04-04_win-flow-p0-authoring-closure.md)
