# Flow Editor Run Detail

## Background

- Win Flow 编辑器当前只保留 graph authoring 能力，没有面向单次 run 的节点结果查看入口。
- Server `flow.detail` 已经提供按 `flow_id + run_id? + node_id + path?` 查询单节点结果的契约，且和 `status` 摘要明确分离。
- 当前 Win store 已经保留 capability `output_schema`，但尚未在运行结果查看链路中消费。

## Goal

- 为 Win Flow 编辑器补齐节点结果详情查看入口。
- 用户在选中节点后，可以查询该节点在某次 run 中的结果详情。
- 结果详情面板应开始结构化消费 `output_schema`，在安全可解析时给出只读字段视图，并保留原始 JSON 结果作为兜底。

## Scope

### Must

- 编辑器必须支持对当前选中节点发起 `flow.detail` 查询。
- 详情查询必须至少支持：
  - `flow_id`
  - `run_id` 可选
  - `node_id`
  - `path` 可选
- 详情面板必须展示：
  - 命中的 `run_id`
  - 命中的 `path`
  - 节点状态摘要
  - 返回的 `result`
- 当当前节点是 `call` 且 capability 已有 `output_schema` 时，详情面板必须展示原始 schema 文本，并在 schema 受支持且查询根结果时提供结构化结果视图。
- 结构化结果视图不可用时，前端仍必须保留原始 `result` 文本展示，不得静默降级为空白。
- 详情查询失败时，编辑器必须明确展示错误，不得静默失败。

### Optional

- 为 `path` 提供快捷示例或未来的路径选择器。
- 为 `path` 结果提供子 schema 导航或更细粒度的结构化渲染。

### Out of Scope

- 本轮不收口画布节点 badge 与状态刷新策略。
- 本轮不提供完整 run 历史浏览器。
- 本轮不为非根路径 `detail` 推导子 schema。
- 本轮不提供完整 JSON Schema explorer，只覆盖受限安全子集。
- 本轮不展示完整 flow local vars 视图。

## Scenarios

- 用户运行或部署过某个 flow 后，在编辑器里选中节点并查看该节点的根结果。
- 用户输入 JSON Pointer，仅查看节点结果中的某个子路径。
- 用户没有显式填写 `run_id`，希望查询最近一次 run 的该节点结果。
- 用户查看 `call` 节点结果时，希望顺带看到当前 capability 的 `output_schema`。
- 用户查看 `call` 节点根结果时，希望直接看到 schema 中关键字段和值，而不是手动在 JSON 文本里搜索。
- 目标 run / node / path 不存在时，用户需要收到明确失败提示。

## Functional Requirements

1. 编辑器必须为选中节点提供结果详情查看入口。
2. 详情查询必须调用 Win `FlowService` 对应的 `detail` binding，而不是前端自行拼接 transport。
3. 当 `run_id` 为空时，前端必须允许请求最近一次 run 的详情。
4. 当 `path` 为空时，前端必须允许请求节点根结果。
5. 返回成功时，前端必须展示：
   - 节点 `status/code/msg`
   - `result`
   - `run_id`
   - `path`
6. 返回失败时，前端必须展示可读错误，并保留用户输入的查询上下文。
7. 当选中节点为 `call` 且存在 capability `output_schema` 时，面板必须展示该 schema 的原始文本视图。
8. 当 `output_schema` 落在前端支持的安全子集内，且当前查询的是节点根结果时，面板必须提供只读结构化结果视图。
9. 结构化结果视图必须优先按 schema 字段标签和 pointer 展示值，而不是推断式自由渲染。
10. 当 schema 不支持、查询路径非根、或结果值与 schema 结构不兼容时，面板必须稳定回退到原始 JSON 文本展示。

## Non-functional Requirements

- 性能：
  - 详情查询按需触发，不随节点切换自动对所有节点批量请求。
  - 详情状态只维护当前查看节点，不引入全图级结果缓存。
- 可读性：
  - UI 必须明确区分 `status` 摘要和 `detail` 结果。
  - 当 `run_id` 为空时，UI 必须说明“将查询最近一次 run”。
  - 当结构化结果可用时，UI 仍必须保留原始 JSON 入口，避免结构化视图掩盖真实 payload。
- 可维护性：
  - 详情数据链应复用现有 `flow.ts` store，不新增第二套 flow transport 封装。
  - `output_schema` 的结构化消费方式应与普通模式的受限 schema 解析边界保持一致，避免出现两套互相漂移的规则。

## Edge Cases

- 当前未选中节点。
- 当前 `flow_id` 为空。
- `run_id` 不存在。
- `node_id` 不存在。
- `path` 非法。
- `path` 合法但结果不存在。
- 节点是 `compose` / `set_var`，没有 capability `output_schema`。
- 查询的是非根路径，无法安全映射到结构化 schema 视图。
- `output_schema` 可解析，但 detail 结果缺失部分字段。
- `output_schema` 与 detail 结果类型不匹配。

## Acceptance Criteria

1. 用户选中节点后，可以在编辑器中主动查询该节点的结果详情。
2. 用户可以查看根结果，也可以按 `path` 查看子路径结果。
3. 对于支持的 `call` 节点根结果，面板会展示基于 `output_schema` 的结构化结果视图，并保留原始 JSON 文本。
4. 对于不支持结构化展示的 schema 或查询路径，面板会稳定回退到原始 JSON 文本。
5. 查询失败时，UI 会明确提示，而不是静默无响应。

## Related Specs

- [flow-editor-run-detail.md](../specs/flow-editor-run-detail.md)
- [flow-editor-visual-form.md](../specs/flow-editor-visual-form.md)
- `D:\project\MyFlowHub3\worktrees\server-local-vars-docs\docs\specs\flow.md`
