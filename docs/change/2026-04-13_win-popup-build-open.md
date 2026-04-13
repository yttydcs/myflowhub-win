# 2026-04-13_win-popup-build-open

## 变更背景 / 目标

- `MyFlowHub-Win` 里已有多处辅助窗口入口直接调用 `window.open(...)`。
- 在浏览器或 `wails dev` 中，这些入口能正常打开新窗；但 packaged Wails build 中点击后经常“没有任何反应”，导致用户无法进入对应的 `layout: "window"` 页面。
- 本轮目标不是引入原生多窗口，而是在保持开发态体验不变的前提下，让 packaged runtime 至少能可靠进入目标页面。

## 具体变更内容

### 新增

- `frontend/src/lib/auxWindow.ts`
  - 新增共享 helper：`openAuxWindow({ routePath, name, size })`。
  - 通过 Wails runtime `Environment().buildType` 判断是否为 packaged runtime。
  - packaged runtime 下改为当前窗口 hash 导航，dev/browser 继续使用 `window.open(...)`。
  - 对非法 route path 显式抛错，并把结果收口为 `opened` / `navigated` / `blocked`。
- `frontend/src/lib/auxWindow.test.ts`
  - 覆盖 packaged 导航、dev 新窗、popup blocked、非法路由四类关键分支。
- `docs/lessons/wails-packaged-aux-window-fallback.md`
  - 沉淀“Wails packaged runtime 不应把 `window.open(...)` 当成唯一辅助窗口策略”的复用排查经验。

### 修改

- `frontend/src/pages/Stream.vue`
  - Source / Delivery 窗口入口改为调用共享 helper。
  - 保持原有 blocked toast 行为，仅在返回 `blocked` 时提示。
- `frontend/src/pages/TopicBus.vue`
  - `Open Window` 改为调用共享 helper。
- `frontend/src/pages/ShowcaseCenter.vue`
  - Viewer / Editor 窗口入口改为调用共享 helper。
- `frontend/src/pages/Showcase.vue`
  - Viewer 窗口入口改为调用共享 helper。
- `frontend/src/pages/Logs.vue`
  - 日志窗口入口改为调用共享 helper，并补齐 blocked toast。
- `frontend/src/stores/file.ts`
  - `openTasksWindow()` 改为异步调用共享 helper，并返回 `Promise<boolean>`。
- `frontend/src/stores/flowProjects.ts`
  - `openEditorWindow()` 改为异步调用共享 helper，并返回 `Promise<boolean>`。
- `frontend/src/pages/File.vue`
  - `openTasks()` 改为 `await fileStore.openTasksWindow()`，保证 fallback 判定真实生效。
- `frontend/src/pages/Flow.vue`
  - `openEditor()` 改为 `await flowProjects.openEditorWindow(projectId)`，避免 Promise 被当成同步真值。
- `frontend/src/pages/Stream.test.ts`
  - 更新为断言共享 helper 的入参，而不是直接断言 `window.open(...)`。
- `frontend/src/i18n/messages/stores.ts`
  - 补充 Logs 页面 popup blocked 文案。
- `plan.md`
  - 回填 Stage 3.2 / 3.3 / 4 的执行、review 与归档状态。

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `updated`

## Related requirements

- 无专门稳定 requirement 需要变更。
- 背景参考：
  - `docs/requirements/stream.md`

## Related specs

- 无专门稳定 spec 需要变更。
- 背景参考：
  - `docs/specs/stream.md`

## Related lessons

- `docs/lessons/wails-packaged-aux-window-fallback.md`

## 对应 plan.md 任务映射

- `PWB-1`
  - 新增共享 helper，收口 packaged / dev 的辅助窗口打开策略。
- `PWB-2`
  - 迁移 Stream、TopicBus、Showcase、Logs、Flow、File 的现有入口到 helper。
- `PWB-3`
  - 增加 helper 定向测试，并跑前端 / Wails 构建验证。
- `PWB-4`
  - 完成 Stage 3.3 review，确认需求覆盖、回归面和残余风险。

## 经验 / 教训摘要

- Wails packaged runtime 不能等同于普通浏览器语义，`window.open(...)` 在 packaged 下不应被当作唯一可达路径。
- 对这类“开发态正常、打包后失效”的窗口问题，优先判断运行时差异和单窗口/多窗口能力边界，而不是先怀疑打包链本身。
- 将运行时分流放到单个 helper 中，比在每个页面分别修补更稳，也更适合未来替换成真正的原生多窗口实现。

## 可复用排查线索

- 症状
  - `wails dev` 或浏览器下能打开辅助窗口，但 packaged build 点击后无反应
  - 同一页面在开发态有新窗，在 build 后既不报错也不导航
- 触发条件
  - 前端直接依赖 `window.open(...)`
  - Wails 应用仍是单个 `options.App`，没有原生多窗口实现
  - packaged runtime 与 dev/browser 的 WebView 行为差异没有被显式分流
- 关键词
  - `window.open`
  - `Environment().buildType`
  - `layout: "window"`
  - `aux window`
  - `packaged runtime`
  - `TopicBus Window`
  - `Stream Delivery Window`
- 快速检查
  - 先搜索当前仓内是否仍有散落的 `window.open(...)` 调用点
  - 检查 Go/Wails 入口是否真的实现了原生多窗口；没有的话，不要假设 packaged 能复现浏览器 popup 行为
  - 用 `Environment().buildType` 或等价运行时信号区分 dev 与 packaged，再决定新窗还是同窗 fallback
  - 构建链通过只能证明编译没坏，不能证明 packaged GUI 的窗口交互已经恢复

## 关键设计决策与权衡

- 选择“共享 helper + packaged 同窗 fallback”，而不是直接做 Go/Wails 原生多窗口。
  - 优点：改动面最小，能立即恢复 packaged 可达性。
  - 代价：packaged 下从“新窗”退化为“当前窗口跳转”，交互路径与开发态不同。
- 选择显式按 runtime 分流，而不是仅在 `window.open(...)` 返回空值时 fallback。
  - 优点：避免把 packaged runtime 错误地当成普通 popup blocked 语义。
  - 代价：helper 对 `Environment().buildType` 语义存在一定耦合。
- 选择让 store 返回 `Promise<boolean>`，而不是在调用点继续假设同步布尔值。
  - 优点：保证 File / Flow 这类 store 入口能够正确等待 helper 结果。
  - 代价：需要同步调整少量页面调用方式。

## 测试与验证方式 / 结果

- 前端定向测试：
  - `npm exec vitest run src/lib/auxWindow.test.ts src/pages/Stream.test.ts`
  - 结果：通过。
- 前端构建：
  - `npm run build`
  - 结果：通过。
- Wails 打包链：
  - `$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`
  - 结果：通过。
- 生成链副作用检查：
  - `go.mod`、`frontend/dist/placeholder.txt`
  - 结果：验证产生的副作用已恢复，不纳入最终改动。
- 手工 / UI：
  - 本轮未在真实 packaged GUI 中执行点击级人工冒烟。
  - 结论：当前结论建立在 helper 分支测试、前端 build 与 Wails build 都通过的基础上；packaged 点击行为仍建议你本地再点一次确认。

## 潜在影响与回滚方案

### 潜在影响

- packaged runtime 下辅助窗口行为会退化为当前窗口导航，用户不再同时保留原页面上下文。
- helper 当前依赖 `Environment().buildType !== "dev"` 识别 packaged；若后续 Wails 调整该语义，需要同步改 helper。
- 新增辅助窗口入口如果绕过 helper 直接写 `window.open(...)`，同类问题会再次出现。

### 回滚方案

- 回退以下文件即可恢复原始行为：
  - `frontend/src/lib/auxWindow.ts`
  - `frontend/src/lib/auxWindow.test.ts`
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/pages/Logs.vue`
  - `frontend/src/stores/file.ts`
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/pages/File.vue`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/pages/Stream.test.ts`
  - `frontend/src/i18n/messages/stores.ts`
- 若同时不保留归档与经验文档，再一并回退：
  - `docs/change/2026-04-13_win-popup-build-open.md`
  - `docs/change/README.md`
  - `docs/lessons/wails-packaged-aux-window-fallback.md`
  - `docs/lessons/README.md`
  - `plan.md`

## 子Agent执行轨迹

- 未派发子Agent。
- 原因：本轮 helper 设计、调用点迁移与归档写集都很小且强耦合，由主Agent串行完成更稳。
