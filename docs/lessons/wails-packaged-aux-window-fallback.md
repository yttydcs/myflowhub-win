# wails-packaged-aux-window-fallback

## Summary

- `MyFlowHub-Win` 这类 Wails 单窗口应用里，辅助窗口入口如果直接依赖 `window.open(...)`，很容易出现“`wails dev` 正常、packaged build 点击无反应”的问题。
- 根因通常不是打包失败，而是 packaged runtime 的窗口语义与浏览器不同，且仓内并没有对应的 Go/Wails 原生多窗口实现。
- 稳定修复模式是把辅助窗口入口收口到共享 helper：dev/browser 继续新窗，packaged runtime 则回退到当前窗口 hash 导航。

## Lookup Hints

- `window.open`
- `Environment().buildType`
- `auxWindow`
- `layout: "window"`
- `packaged runtime`
- `wails dev works but build does not`
- `TopicBus Window`
- `Stream Source Window`
- `Stream Delivery Window`
- `Showcase Viewer`

## Symptoms

- 浏览器或 `wails dev` 中点击 `Open Window` / `Input Window` / `Output Window` 可以正常打开新窗。
- packaged build 中点击同一入口没有任何反应，既不打开新窗，也不导航到目标页面。
- 代码和构建日志没有明显错误，看起来像“功能只在 build 后失效”。

## Impact

- 用户无法进入 TopicBus、Stream、Showcase、Logs、Flow、File 等模块的辅助窗口页面。
- 单独窗口承载的重要功能会直接变成不可用，而不是单纯体验退化。

## Trigger Conditions

- 前端把辅助窗口实现为散落的 `window.open(...)`。
- 当前 Wails 应用仍只有单个 `options.App`，没有 Go 侧原生多窗口支持。
- 开发者用浏览器/`wails dev` 的 popup 行为去推断 packaged runtime。

## Root Cause

- `window.open(...)` 是浏览器语义，不是 Wails packaged runtime 的稳定契约。
- 当前仓库的 Win 应用没有原生多窗口实现，前端却默认 packaged runtime 也能像浏览器那样开辅助窗口。
- 因为 packaged runtime 不一定表现为标准的 popup blocked 语义，所以单纯依赖 `window.open(...)` 返回值也不稳。

## Investigation Trail

- 先对前端搜索 `window.open(...)`，确认所有辅助窗口入口的真实分布。
- 再检查 Go/Wails 入口，确认当前应用只有单个 `options.App`，没有多窗口实现。
- 通过 `frontend/wailsjs/runtime/runtime.d.ts` 确认可读取 `Environment()` 与 `buildType`。
- 跑通 `npm run build` 和 `GOWORK=off wails build -debug -skipembedcreate -nopackage`，确认问题不是构建链失败。
- 最后把问题收敛为：需要按 runtime 分流，而不是继续把 packaged runtime 当作普通浏览器 popup。

## Resolution

- 新增共享 helper 统一辅助窗口策略。
- 仅允许 `#/...` hash 路由作为辅助窗口入口。
- 在 dev/browser 中继续尝试 `window.open(...)`，并保留 blocked 反馈。
- 在 packaged runtime 中直接设置 `window.location.hash` 导航到目标页面。
- 将所有现有辅助窗口调用点迁移到该 helper，避免后续再出现散落实现。

## Prevention / Guardrails

- 新增任何辅助窗口入口时，默认先复用共享 helper，不要直接写 `window.open(...)`。
- 对“dev 正常、build 异常”的 Wails 窗口问题，先判断 runtime 能力边界，再判断代码 bug。
- 若未来要引入真正的原生多窗口，把切换逻辑放在 helper 内部，而不是重新修改所有页面。
- 构建通过只能证明编译链没坏；如果问题与 packaged GUI 交互有关，最终仍应补一次点击级人工冒烟。

## Related Docs

- [2026-04-13_win-popup-build-open.md](../change/2026-04-13_win-popup-build-open.md)
- [2026-03-31_win-stream-windows-trim.md](../change/2026-03-31_win-stream-windows-trim.md)
