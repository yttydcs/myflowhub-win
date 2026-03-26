# Flow Editor Run Detail Spec

## Scope

- 本规范限定 Win Flow 编辑器中“单节点结果详情”查看链路。
- 本规范消费 `flow.detail` 与已有 capability `output_schema`，但不扩展运行时协议。

## Interfaces / Contracts

### 1. 查看入口

- 结果详情入口挂在当前选中节点的右侧 inspector 中。
- 仅当存在选中节点时显示该入口。
- 面板至少包含：
  - `run_id` 输入框
  - `path` 输入框
  - `Load Detail` 按钮
  - 查询结果展示区

### 2. 请求契约

- Win `FlowService` 必须新增：
  - `Detail(ctx, sourceID, targetID uint32, req flow.DetailReq) (flow.DetailResp, error)`
  - `DetailSimple(sourceID, targetID uint32, req flow.DetailReq) (flow.DetailResp, error)`
- 前端 store 必须通过 `window.go.flow.FlowService.DetailSimple(...)` 发起请求。
- 请求字段映射：
  - `flow_id` <- 当前编辑中的 flow ID
  - `run_id` <- 用户输入，允许为空
  - `node_id` <- 当前选中节点 ID
  - `path` <- 用户输入，允许为空

### 3. 返回展示契约

- 成功响应后，前端保存并展示：
  - `run_id`
  - `path`
  - `node.id`
  - `node.status`
  - `node.code`
  - `node.msg`
  - `result`
- `result` 第一版使用格式化 JSON 文本展示；非 JSON object/array 也必须稳定显示。

### 4. `output_schema` 结构化消费

- 当选中节点是 `call` 时，前端按当前 capability route 读取 `output_schema`。
- 前端必须始终保留：
  - `output_schema` 的格式化 JSON 文本
  - `detail.result` 的格式化 JSON 文本
- 当同时满足以下条件时，面板额外提供只读结构化结果视图：
  - `output_schema` 可被前端受限 schema 解析器接受
  - 当前查询的是根结果，即 `path=""`
  - detail 返回值可按根 schema 读取字段
- 结构化结果视图要求：
  - 仅用于展示，不写回 spec
  - 使用 schema 字段顺序、标签和 pointer
  - 对缺失字段显示“无值”占位，不伪造默认值
- `compose` / `set_var` 不额外推导 schema。

## Data Model or Protocol

推荐 store 模型：

```ts
type FlowNodeDetail = {
  loading: boolean
  error: string
  requestedNodeId: string
  requestedRunId: string
  requestedPath: string
  runId: string
  path: string
  node: {
    id: string
    status: string
    code: number
    msg: string
  } | null
  resultValue: unknown
  resultText: string
}
```

约束：

- store 只维护“当前详情面板”的单份 detail 状态。
- 请求成功后，`runId/path/node/resultValue/resultText` 用响应值覆盖。
- 请求失败时，`error` 更新；`requestedRunId/requestedPath` 保留用户输入。
- 结构化结果 view model 可在 store helper 或 inspector computed 中按需生成，不单独持久化。

## Error Handling

- 未选中节点：
  - 不发请求
  - UI 显示空态
- `flow_id` 为空：
  - 立即报错，不发请求
- `path` 非法：
  - 可直接透传后端 `400`，并在 UI 显示错误
- `404`：
  - 明确提示 run / node / path 未命中
- transport / timeout：
  - 复用现有 `FlowService` await 错误包装
- `output_schema` 不受支持、查询路径非根、或结果与 schema 不兼容：
  - 不视为 detail 请求失败
  - 只禁用结构化结果视图
  - 保留原始 schema / result 文本展示

## Security / Safety

- `result` 和 `output_schema` 只按文本 / JSON 展示，不做动态执行。
- 结构化结果视图只消费受限 schema 子集，不做宽松推断型渲染。

## Performance Constraints

- 详情查询按按钮触发。
- 节点切换不自动为所有节点预加载结果。
- 只对当前选中节点读取并解析 schema 展示，不全图预解析。
- 结构化结果视图只在当前 inspector 渲染链路上按需生成。

## Testing Strategy

- store tests：
  - detail 成功后保留 `resultValue` 与 `resultText`
  - output schema 结构化 helper 在根结果和回退场景下行为稳定
- inspector tests：
  - 支持 schema 时展示结构化字段和值
  - path 非根或 schema 不支持时回退到纯文本视图
- regression：
  - 现有 detail 错误展示、raw JSON 结果展示不退化

## Related Requirements

- [flow-editor-run-detail.md](../requirements/flow-editor-run-detail.md)
- [flow-editor-visual-form.md](../requirements/flow-editor-visual-form.md)
