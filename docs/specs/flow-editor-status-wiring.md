# Flow Editor Status Wiring Spec

## Scope

- 本规范限定 Win Flow 编辑器中“flow status 摘要 -> 画布节点 badge”的接线方案。
- 本规范只消费 `flow.run` / `flow.status` 的既有契约，不扩展运行时协议。

## Interfaces / Contracts

### 1. Toolbar / Window 刷新入口

- `FlowEditorWindow.vue` 必须提供最小运行状态入口：
  - `Run Flow`
  - `Refresh Status`
- `Run Flow` 成功后，继续使用响应中的 `run_id` 拉取该次运行的状态摘要。
- `Refresh Status` 行为：
  - 若 store 已保存 `statusRunId`，优先查询该 `run_id`
  - 否则查询最近一次 run

### 2. Store 状态模型

- `flow.ts` 继续保留：
  - `statusRunId`
  - `lastStatus`
- `lastStatus` 至少包含：

```ts
type FlowStatus = {
  status: string
  runId: string
  executorNode: number
  nodes: Array<{
    id: string
    status: string
    code: number
    msg: string
  }>
}
```

- store 必须新增统一 reset helper，至少清理：
  - `statusRunId`
  - `lastStatus`
- 以下入口必须调用统一 reset：
  - `newDraft()`
  - `applyGraphDraft()`
  - `applyGraphEditorState()`

### 3. 画布接线

- `FlowEditorWindow.vue` 必须把 `flowStore.state.lastStatus.nodes` 直接传给 `FlowCanvas.vue` 的 `statusNodes`。
- `FlowCanvas.vue` 使用 `statusNodes` 生成 `Map<nodeId, status>`，并透传给对应 `FlowNode.vue`。
- 未命中状态的节点不构造占位状态对象，由节点组件走空态展示。

### 4. 节点 badge 展示

- `FlowNode.vue` 必须支持以下 badge 状态映射：
  - `queued`
  - `running`
  - `succeeded`
  - `failed`
  - `cancelled`
  - unknown / empty
- badge 需展示：
  - 状态标签
  - `Code {code}`
  - 可选 `msg`
- `kind` 标签必须与现有节点类型兼容，不因接 status 破坏 `call/compose/set_var` 显示。

## Data / Call Flow

1. 用户点击 `Run Flow`
2. `flow.ts.runFlow()` 发送 `flow.run`
3. run 成功后，store 保存 `statusRunId`
4. store 继续调用 `statusFlow(runId)`
5. `handleStatusResp(...)` 更新 `lastStatus`
6. `FlowEditorWindow.vue` 把 `lastStatus.nodes` 传给 `FlowCanvas.vue`
7. `FlowCanvas.vue` 把对应节点摘要透传给 `FlowNode.vue`
8. `FlowNode.vue` 渲染 badge

手动刷新路径：

1. 用户点击 `Refresh Status`
2. `FlowEditorWindow.vue` 读取 `flowStore.state.statusRunId`
3. 若为空，调用 `statusFlow("")` 查询最近一次 run
4. 若非空，调用 `statusFlow(statusRunId)`
5. 后续更新链路同上

## Error Handling

- `flow_id` 为空：
  - 本地立即失败，不发请求
- `status` 请求失败：
  - 复用现有消息通道显示错误
  - 不覆盖最近一次成功的 `lastStatus`
- 项目切换 / 新建草稿：
  - 立即 reset status state，避免旧状态残留
- `status.nodes` 缺项：
  - 仅已命中的节点显示 badge，其他节点显示空态

## Performance / Safety

- 所有 status 查询必须显式触发，不引入定时器。
- `lastStatus.nodes` 只保存摘要，不缓存完整 detail 结果。
- 画布映射只在前端内存中完成，不做额外网络请求。

## Testing Strategy

- store tests：
  - `handleStatusResp(...)` / `statusFlow(...)` 的状态映射
  - status reset helper 在草稿加载 / 新建时生效
- window tests：
  - `Run Flow` / `Refresh Status` 调用路径
  - `statusNodes` 正确传给 `FlowCanvas`
- component tests：
  - `FlowNode.vue` 对 `cancelled` / unknown 的 badge 展示

## Related Requirements

- [flow-editor-status-wiring.md](../requirements/flow-editor-status-wiring.md)
- [flow-editor-run-detail.md](../requirements/flow-editor-run-detail.md)
