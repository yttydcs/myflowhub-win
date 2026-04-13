# detached-window-session-snapshot-hydration

## Summary

- `MyFlowHub-Win` 的 detached window / 独立 route 会在新的前端上下文里启动，`sessionStore` 会回到默认空态。
- `LoadHomeState()` 只能恢复 `nodeId` / `hubId` 这类 identity fallback，不会自动恢复当前 runtime 的 session 连接快照。
- 如果窗口直接用 `sessionStore.connected` 控 badge、按钮禁用态或 ready-check，但初始化阶段没有显式调用 `IsConnected()` / `LastAddr()`，就会出现“hub 实际已连接，但窗口仍显示 `Disconnected`”的假象。
- 稳定修复模式是在共享 store 中提供显式补水 helper，并在 detached window 启动时优先调用。

## Lookup Hints

- `Disconnected`
- `sessionStore.connected`
- `LoadHomeState`
- `IsConnected`
- `LastAddr`
- `ShowcaseWindow`
- `TopicBusWindow`
- `detached window`
- `fresh frontend context`
- `loadHomeDefaults`

## Symptoms

- 主窗口已经连上 hub，但新开的 `Showcase` 编辑窗口、预览窗口或 `TopicBusWindow` 顶部仍显示 `Disconnected`。
- detached window 中依赖 `connected` 的控件被错误禁用。
- 同一会话里主窗口状态正确，只有新开的窗口错误。
- 手动刷新事件或后续 runtime `session.state` 事件到达后，窗口状态可能又恢复正常，表现出“偶发”特征。

## Impact

- 用户会把 UI 假断开误判成 hub 真断开。
- 窗口内的 ready-check 和交互 gating 错误，导致本应可用的操作被禁止。
- 相邻窗口如果复用了同样的启动模式，会批量继承同一个问题。

## Trigger Conditions

- 页面运行在 detached window、辅助窗口或独立 route 的新前端上下文。
- 初始化阶段只调用 `LoadHomeState()` 恢复 identity。
- 页面在首次渲染前就依赖 `sessionStore.connected`。

## Root Cause

- `sessionStore` 默认 `connected=false`。
- detached window 冷启动时没有像 `Home.vue` 那样主动查询当前 runtime session snapshot。
- `LoadHomeState()` 不负责恢复 session 连接状态，因此只补 identity 不补 session 会直接落回 `Disconnected`。

## Investigation Trail

- 先确认 hub 实际仍连接，排除后端 session 真断开。
- 对比 `Home.vue` 与目标 detached window 的启动流程，查看谁会调用 `IsConnected()` / `LastAddr()`。
- 核对目标窗口是否直接用 `sessionStore.connected` 渲染 badge 或控制交互禁用态。
- 若主窗口正确、独立窗口错误，优先怀疑“新前端上下文只恢复 HomeState，没有恢复 session snapshot”。

## Resolution

- 在 `frontend/src/stores/session.ts` 新增 `hydrateSessionConnectionSnapshot()`。
- helper 内部显式调用 `IsConnected()`，并在已连接时补 `LastAddr()`，统一更新 `sessionStore`。
- 在 `Showcase.vue`、`ShowcaseWindow.vue`、`TopicBusWindow.vue` 的 `loadHomeDefaults()` 中接入该 helper。
- 让 `Home.vue` 也复用这套 helper，消除重复实现。
- 用 `frontend/src/stores/session.test.ts` 和 `frontend/src/windows/ShowcaseWindow.test.ts` 锁定关键路径。

## Prevention / Guardrails

- 新增 detached window 时，如果页面会读取 `sessionStore.connected`，必须显式补水 runtime session snapshot，或统一走共享 bootstrap helper。
- 把 `LoadHomeState()` 视为 identity fallback，而不是完整的会话恢复入口。
- 至少保留一条代表性 detached window 测试，避免后续窗口改动再次漏掉启动补水。

## Related Requirements / Specs / Changes

- Requirements:
  - `docs/requirements/showcase-display-widgets.md`
- Specs:
  - `docs/specs/showcase-display-widgets.md`
- Changes:
  - `docs/change/2026-04-14_win-showcase-window-session-sync.md`
