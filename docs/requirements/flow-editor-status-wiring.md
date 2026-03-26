# Flow Editor Status Wiring

## Background

- Win Flow 编辑器已经具备 `flow.run` / `flow.status` 的 store 能力，但当前窗口层没有稳定的运行状态刷新入口。
- 画布节点组件已经支持展示 `status/code/msg` badge，但窗口层仍把 `statusNodes` 固定传空数组，导致节点状态始终不反映真实运行摘要。
- 当前 store 在 `newDraft` / `loadGraphEditorState` / `loadGraphDraft` 等入口没有统一清空 `lastStatus`，一旦接上线真实 badge，旧 run 状态会泄漏到新草稿。

## Goal

- 为 Win Flow 编辑器补齐稳定的状态刷新路径。
- 让画布节点 badge 直接消费真实 `flow.status` 摘要。
- 保证 flow / run 状态在草稿切换、新建和重新加载时不会残留旧数据。

## Scope

### Must

- 编辑器必须提供明确的手动状态刷新入口。
- 运行 flow 后，编辑器必须能把命中的 `run_id` 和节点摘要状态接到画布。
- 画布节点 badge 必须展示真实的 `status/code/msg`，而不是占位值。
- 状态刷新必须复用现有 `flow.status` 能力，不得新增第二套 transport。
- `newDraft`、`loadGraphDraft`、`loadGraphEditorState` 和切换项目时，必须清理旧的 flow 运行状态，避免跨草稿泄漏。
- 当存在已命中的 `run_id` 时，手动刷新必须优先刷新该 `run_id`；否则查询最近一次 run。
- `cancelled` / `unknown` / 无状态等边界情况必须有稳定展示。

### Optional

- 在 toolbar 中展示 flow 级摘要状态和当前命中的 `run_id`。
- 为“尚未运行”提供更明确的空态文案。

### Out of Scope

- 本轮不引入自动轮询。
- 本轮不提供完整 run 历史浏览。
- 本轮不把 `status` 与 `detail` 面板做更深双向联动。
- 本轮不扩展 `output_schema` 或结果结构化渲染。

## Scenarios

- 用户点击运行后，希望在画布节点上立即看到 `queued/running/succeeded/failed/cancelled` 的摘要状态。
- 用户在运行一段时间后点击手动刷新，希望继续查看同一个 `run_id` 的节点状态。
- 用户未显式运行但已有历史执行，希望刷新最近一次 run 的状态。
- 用户切换到另一个项目、重新加载草稿或新建空白图后，不希望看到旧 flow 的 badge 残留。
- 某些节点没有状态、状态未知或被取消时，用户仍能看到一致且可理解的 UI。

## Functional Requirements

1. 编辑器必须提供对当前 flow 执行 `status` 查询的入口。
2. 编辑器运行 flow 成功后，必须继续查询该次 `run_id` 的状态摘要。
3. 当用户手动刷新状态时：
   - 若当前已有命中的 `run_id`，优先查询该 `run_id`
   - 否则查询最近一次 run
4. 画布层必须消费 `status.nodes[]`，并把对应节点的 `status/code/msg` 传入节点 badge。
5. 节点组件必须对协议中的有效状态至少支持：
   - `queued`
   - `running`
   - `succeeded`
   - `failed`
   - `cancelled`
   - 以及未知/空状态兜底
6. 当状态刷新失败时，编辑器必须给出明确错误，不得静默失败。
7. 当切换项目、新建草稿或重载 graph 时，编辑器必须清空旧的 flow 级状态和节点 badge。

## Non-functional Requirements

- 性能：
  - 状态查询必须按显式用户动作触发，不引入后台轮询。
  - 画布 badge 映射必须基于内存态摘要，不做额外 I/O。
- 可读性：
  - flow 级摘要状态和节点级摘要状态必须语义清晰，不与 detail 结果混淆。
  - toolbar / badge 文案必须对未运行、未知和已取消状态保持一致。
- 可维护性：
  - 状态刷新入口、当前 `run_id` 和节点摘要状态应继续由 `flow.ts` 单一 store 承担。
  - 状态 reset 逻辑必须集中，不允许散落在多个窗口层分支里。
- 可扩展性：
  - 后续如新增自动刷新或更丰富的运行概览，不应要求重写画布节点组件。

## Edge Cases

- 当前 `flow_id` 为空。
- 当前没有任何历史 run。
- 当前记录的 `run_id` 已失效。
- `status` 返回的节点列表不包含图中的所有节点。
- 状态为 `cancelled`。
- 节点状态为空或未知。
- 用户在请求进行中切换项目或新建草稿。

## Acceptance Criteria

1. 用户运行或手动刷新后，画布节点 badge 会显示真实状态摘要。
2. 节点 badge 会显示对应的 `code/msg` 摘要信息。
3. 用户再次刷新时，默认延续当前命中的 `run_id`；没有 `run_id` 时回落到最近一次 run。
4. 切换项目或新建草稿后，不会残留上一份 flow 的状态 badge。

## Related Specs

- [flow-editor-status-wiring.md](../specs/flow-editor-status-wiring.md)
- [flow-editor-run-detail.md](../specs/flow-editor-run-detail.md)
- `D:\project\MyFlowHub3\worktrees\server-local-vars-docs\docs\specs\flow.md`
