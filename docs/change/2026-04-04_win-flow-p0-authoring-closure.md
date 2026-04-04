# 2026-04-04_win-flow-p0-authoring-closure

## 变更背景 / 目标

- Win Flow 编辑器此前已分阶段补齐高阶节点与 `foreach.body` visual session，但 P0 仍缺少几类会直接影响保存正确性的 authoring 能力：
  - `loop_item` / `loop_index` 的 Win-side source authoring 闭环
  - node / flow / trigger 的运行控制字段 authoring
  - `subflow` 自调用与本地可判定递归链的前端拦截
- 本轮目标是把这些剩余 P0 能力补齐，并用定向测试把根图 / body 图、strict save 和本地 project 保存路径固定下来。

## 具体变更内容

- `frontend/src/stores/flow.ts`
  - 收敛 `loop_item` / `loop_index` 的 parse、draft、strict save 和 export 规则。
  - 增加 `retry_backoff_ms`、`max_active_runs`、`dedup_window_ms` 的 load / draft / strict export。
  - 对 `interval` / `cron` + `dedup_window_ms > 0` 做本地显式失败。
  - 对 `subflow.flow_id == current flow_id` 做 strict save 拦截。
  - 收紧 source 解析：超出普通模式支持子集的 source shape 不再被静默吸收，避免根图或 body 图误回 form。
- `frontend/src/windows/FlowEditorWindow.vue`
  - `foreach.body` session 继续以 `foreachBodyJson` 为真相源，但现在在保存 project 前会基于本地 project 图做 subflow recursion guard。
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 根图 inspector 继续不暴露 loop sources，并提供 `retry_backoff_ms` authoring。
- `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - body inspector 暴露 `loop_item` / `loop_index`，并提供 `retry_backoff_ms` authoring。
- `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - 仅在 body 会话显式开启 `allowLoopSources` 时暴露 loop source 选项。
- `frontend/src/pages/Flow.vue`
  - flow / deploy authoring 暴露 `max_active_runs` 与 `dedup_window_ms`。
- `frontend/src/stores/flowProjects.ts`
  - 本地 project trigger / metadata 归一化、持久化与 deploy wire 支持 `max_active_runs`、`dedup_window_ms`，并对不支持的 dedup 组合显式失败。
- 测试
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/stores/flowProjects.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
- 文档
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps\docs\requirements\flow-editor-visual-form.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`

## Related lessons

- `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps\docs\lessons\flow-body-spec-mode-normalization.md`

## 对应 plan.md 任务映射

- `WIN-P0-1`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
- `WIN-P0-2`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/pages/Flow.vue`
  - 相关 tests
- `WIN-P0-3`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
- `WIN-P0-4`
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - 定向 Vitest 回归

## 经验 / 教训摘要

- `foreach.body` 可以继续保持 JSON 真相源，同时把 source-scope 与 strict save 递归校验补齐，不需要引入第二套持久化模型。
- `max_active_runs` 这类 flow-level 可选数字字段，前端必须显式保留 `null` 与 `0` 的差异；否则 UI 看似可编辑，wire 语义却会漂移。
- source draft 的 parse 阶段不能只“归一化”已知字段；若 shape 超出普通模式支持子集，必须及时退回 `Advanced JSON`，否则会在 form/json 切换时静默吃掉字段。

## 可复用排查线索

- 症状
  - 根图意外出现 `loop_item` / `loop_index`
  - `foreach.body` 中 `loop_index` 带 path 后仍被当成 form
  - `interval` / `cron` trigger 配了 dedup window 仍能保存
  - 本地 project 存在明显 subflow 环时仍能保存
- 触发条件
  - form/json mode 切换
  - `exportPayload()` / `deployProject()`
  - `FlowEditorWindow.vue` 的 project save
- 关键词 / 错误文本
  - `loop_item`
  - `loop_index`
  - `retry_backoff_ms`
  - `max_active_runs`
  - `dedup_window_ms`
  - `Subflow recursion detected across local projects`
  - `must not call the current flow itself`
- 快速检查
  - `parseSourceDraft(...)`
  - `buildTrigger(...)`
  - `exportPayload()`
  - `deployProject(...)`
  - `buildLocalSubflowDependencyMap(...)`
  - `findRecursiveSubflowChain(...)`

## 关键设计决策与权衡

- 继续把 `foreachBodyJson` 作为唯一真相源
  - 好处：不破坏 dirty、recovery、root graph save 契约
  - 代价：body 会话需要在窗口层做额外同步与递归校验
- 对不支持的 source shape 选择回退 `Advanced JSON`
  - 好处：不静默删除 `path`、额外字段或错误 scope
  - 代价：旧 payload 若带着无效 shape，用户会更早看到显式失败
- 本地 recursion guard 只基于当前 project + 已知本地 project 图
  - 好处：不引入远端查询或额外 I/O
  - 代价：只能拒绝“当前可判定”的递归链，不能替代服务端的最终校验

## 测试与验证方式 / 结果

- 定向前端测试
  - 命令：
    - `node ./node_modules/vitest/vitest.mjs run src/stores/flow.test.ts src/stores/flowProjects.test.ts src/components/flow/editor/FlowNodeInspector.test.ts src/components/flow/editor/FlowBodyNodeInspector.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：
    - 通过，`5 passed / 60 passed`
- 只读审查
  - `git diff --check`
  - 结果：
    - 通过
- 全量前端构建
  - 命令：
    - `npm run build`
  - 结果：
    - 未作为本轮有效验收关闭
    - 原因：仓库既有基线缺失 `../../wailsjs/go/main/App`，错误位于 `src/windows/StreamDeliveryWindow.vue`

## 潜在影响

- Win Flow 编辑器现在可在 root/body scope 上明确区分 loop sources 的可见范围。
- flow / deploy authoring 现在会更早拒绝无效运行控制组合，部分用户会从“服务端报错”变成“前端本地报错”。
- 本地 project save 对 subflow 递归会更严格；若用户本地 project 图本身存在环，保存会被立即阻止。

## 回滚方案

- 回退以下文件即可撤销本轮能力：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/stores/flowProjects.test.ts`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/components/flow/editor/FlowBodyNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`

## 子Agent执行轨迹

- `Stage 3.3` → `Hegel` → `D:\project\MyFlowHub3\worktrees\feat-win-p0-authoring-gaps`
  - 任务：只读审查 `WIN-P0-1/2/3` 的实现与测试覆盖
  - 修改文件：无
  - 结果：`no findings`
  - 额外记录：提示全量前端构建仍受既有 `src/windows/StreamDeliveryWindow.vue` 的 Wails binding 缺失阻塞
