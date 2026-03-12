# Plan - MyFlowHub-Win：修复 FlowCanvas 事件监听导致 Node Detail 不可选

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`fix/win-canvas-events`
- Worktree：`d:\project\MyFlowHub3\worktrees\fix-win-canvas-events`
- Base：`main`

## 项目目标与当前状态
- 目标：Win → Flow 页面中，选择某个 Flow 后，点击画布中的节点能够正确选中并在右侧 `Node Detail` 展示其详情；同时边/空白点击与拖拽结束事件能正常触发（用于编辑体验与位置保存）。
- 当前状态（已复现）：右侧长期显示 `Select a node to edit its details.`。根因是 `FlowCanvas.vue` 的事件回调签名与 `@vue-flow/core` 不匹配：`nodeClick/edgeClick/nodeDragStop` 实际传入的是单个对象参数（例如 `NodeMouseEvent` / `EdgeMouseEvent` / `NodeDragEvent`，形如 `{ event, node }`），但当前代码按 `(event, node)` 两个参数解析，导致 `node`/`edge` 恒为 `undefined`，从而不会触发选中与位置写回。

## 范围（必须 / 可选 / 不做）
- 必须：
  - 修复 `FlowCanvas.vue` 的事件监听写法，确保 node/edge/pane/drag-stop 相关回调在 Win 中可用。
  - 行为保持最小变更：不改节点数据结构、不改协议、不改权限、不改现有 store 逻辑。
- 可选：
  - 无（本轮先聚焦修复事件监听，避免范围外扩）。
- 不做：
  - 不增加 `debug::echo` 的“执行输出/result”展示（当前仅用 `Status` 节点状态验证执行是否成功）。

## 可执行任务清单（Checklist）

### CANVAS-1 修复 FlowCanvas 事件监听（kebab-case）
- 目标：点击节点/边/空白、拖拽停止时，对应事件能触发并更新 store 选择与坐标。
- 涉及模块/文件：`frontend/src/components/flow/FlowCanvas.vue`
- 变更要点：
  - 将监听改为 kebab-case（例如 `@node-click`），并将回调参数按 `@vue-flow/core` 的事件对象结构解析（`{ event, node }` / `{ event, edge }` / `{ event, node, nodes }`）。
- 验收条件：
  - 点击节点：右侧 `Node Detail` 出现并可编辑。
  - 点击空白：清空选择，右侧回到提示文案。
  - 点击边：边被选中（视觉高亮）；同时节点详情隐藏（保持现状：选边时清空节点选择）。
  - 拖拽节点并松开：节点位置写回 store（保存后再次 `Get` 该 flow，节点位置仍保持）。
- 测试点：
  - 手工：启动 Server + 本 worktree 的 Win 后在 UI 冒烟（见 CANVAS-2）。
- 回滚点：
  - revert 本任务提交。

### CANVAS-2 手工冒烟（可交接步骤）
- 目标：从用户视角确认“选择节点”功能恢复。
- 步骤：
  1) 启动 Server（workspace 根目录执行）：`pwsh -File d:\project\MyFlowHub3\scripts\run-dev.ps1 -WaitServer -SkipWin -SkipMetricsNode`
  2) 启动 Win（本 worktree 根目录执行）：`wails dev`
  3) Win → Home：Connect + Register/Login
  4) Win → Flow：点左侧某个 flow（触发 Get），再点击画布节点，观察右侧 Node Detail。
  5) 拖拽节点 → Save → 再点左侧该 flow（Get）确认位置保持。
- 验收条件：上述步骤通过。
- 回滚点：revert 本任务提交。

### CANVAS-3 归档变更
- 目标：沉淀本次修复背景、根因、改动与验证方式。
- 涉及模块/文件：`docs/change/2026-03-12_win-canvas-events.md`
- 验收条件：文档包含任务映射、关键决策、验证步骤、回滚方案。
- 回滚点：文档改动可独立回退。
