# 2026-03-31 Win Stream Product Tabs

## 变更背景 / 目标

- 当前 Win `Stream` 页面虽然已经具备基础控制面能力，但主页面仍是“控制 + 创建 + 订阅 + 观察”混排，使用成本高，也不够像一个可持续使用的产品界面。
- 用户希望 `Stream` 更接近 `VarPool` 的使用方式：
  - 顶部 tab 切换
  - `Source / Consumer / Control` 职责分离
  - 主界面以列表为主，不堆大块常驻表单
  - 新增、订阅、输入通过弹窗或独立工作区完成
  - 本地 source / consumer 需要持久化，并在登录后自动恢复
- 本轮目标是在不修改 `stream` 协议 wire 的前提下，把 Win `Stream` 改造成更接近产品形态的模块，并补最小可用的 text source 输入能力。

## 具体变更内容

### 新增：Stream profile-backed prefs

- `app_stream.go`
  - 新增 `StreamPrefs()` / `SaveStreamPrefs(...)`
  - 持久化：
    - `activeTab`
    - `targetId`
    - 本地 `sources`
    - 本地 `consumers`
  - 统一做 trim、去重、非法值回退
- `app_stream_test.go`
  - 补默认值、去重、非法输入归一化测试

### 修改：Stream store 分层为本地列表 / 远端目录 / runtime 状态

- `frontend/src/stores/stream.ts`
  - 新增 `localSources` / `localConsumers`
  - 保留远端目录 `sources` / `consumers`
  - 保留 `deliveries` / `textFramesByDelivery` / `statsByDelivery`
  - 新增：
    - `loadPrefs()`
    - `savePrefs()`
    - `restoreLocalCatalogs()`
    - `publishText()`
    - `setActiveTab()`
    - `sourceById()` / `consumerById()`
    - `deliveriesForSource()` / `deliveriesForConsumer()`
- `frontend/src/stores/stream.test.ts`
  - 补 store 恢复、本地/远端目录隔离、text publish 回归测试

### 新增：text source 最小发送能力

- `internal/services/stream/publish.go`
  - 新增 `PublishText()` / `PublishTextSimple()`
  - 只允许：
    - 本地已存在 source
    - `kind=text`
    - 存在 active local producer deliveries
  - 复用现有 DATA 帧格式发送文本到活跃 delivery
  - 发送后同步更新 producer 侧 runtime snapshot
- `internal/services/stream/service_test.go`
  - 补 publish 成功与无 active delivery 失败测试

### 修改：Stream 页面重构为产品化 tabs + dialogs

- `frontend/src/pages/Stream.vue`
  - 顶部改为 `Source / Consumer / Control` tabs
  - `Source`
    - 以本地 source 列表为主
    - 新增 source 改为弹窗
    - 输入改为独立 `Source Input Studio`
    - 列表与侧栏展示当前 binding 摘要
  - `Consumer`
    - 以本地 consumer 列表为主
    - 新增 consumer 改为弹窗
    - 订阅改为独立弹窗
    - 列表与侧栏展示当前绑定到的 source 摘要
  - `Control`
    - 保留远端查询、connect / subscribe、runtime delivery 检查能力
    - 收敛布局，避免和本地新增表单混杂
- `frontend/src/pages/Stream.test.ts`
  - 补 tab、source 新增、source studio、consumer 订阅弹窗测试

### 修改：AppShell 自动恢复 Stream 本地 catalogs

- `frontend/src/layout/AppShell.vue`
  - profile 切换时加载 Stream prefs
  - session connected + loggedIn + node/hub ready 时恢复本地 source / consumer
  - 自动恢复部分失败时给出 toast 警告

### 修改：新增 Stream 文案 i18n

- `frontend/src/i18n/messages/stores.ts`
  - 补齐本轮新增的 Stream 页面、store、AppShell 相关 `zh-CN` 文案

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `none`

## Related requirements

- `docs/requirements/stream.md`

## Related specs

- `docs/specs/stream.md`

## Related lessons

- `docs/lessons/stream-ctrl-await-mismatch.md`
- `docs/lessons/stream-local-owner-ctrl-gap.md`
- `docs/lessons/frontend-build-babel-parser-missing.md`

## 对应 plan.md 任务映射

- `STRUX-1`
  - Stream prefs 持久化接口与测试
- `STRUX-2`
  - Stream store 分层、prefs、自动恢复、publishText
- `STRUX-3`
  - text source 最小 publish binding 与测试
- `STRUX-4`
  - Stream 页面 tabs / dialogs / i18n / 页面测试
- `STRUX-5`
  - `AppShell` profile / session 生命周期接入 Stream 自动恢复
- `STRUX-6`
  - 3.3 review 与本 change 归档

## 经验 / 教训摘要

- `Stream` 这类产品化页面如果继续把“创建、订阅、连接、观察”堆在一个平面里，功能虽然齐了，但很难稳定使用；列表优先 + 弹窗动作更符合 Win 现有模块风格。
- 本地 source / consumer 这类用户资产应该放进 profile-backed prefs，而不是浏览器侧临时存储；否则登录恢复和多 profile 语义会断裂。
- fresh worktree 的前端验证仍然要先补齐 Wails bindings；否则很容易把 `../../wailsjs/...` 缺失误判成页面代码回归。

## 可复用排查线索

- 症状
  - `Could not resolve "../../wailsjs/..."`
  - `Stream` 页面测试或 build 无法解析 Wails import
  - 本地 source / consumer 重启后消失
  - `Only text sources support direct input.`
  - `no active deliveries`
- 触发条件
  - fresh worktree 未生成 `frontend/wailsjs/**`
  - profile 未保存或未恢复 Stream prefs
  - 对非 `text` source 打开输入工作区
  - 当前 source 没有 active local producer delivery
- 关键词
  - `StreamPrefs`
  - `restoreLocalCatalogs`
  - `PublishTextSimple`
  - `Source Input Studio`
  - `Subscribe Consumer`
  - `wails generate module`
- 快速检查
  - 看 `frontend/wailsjs/**` 是否存在；若缺失先执行 `$env:GOWORK='off'; wails generate module`
  - 看 `AppShell.vue` 是否在 profile / session 生命周期中调用 `stream.loadPrefs()` 和 `stream.restoreLocalCatalogs()`
  - 看 `stream.ts` 中本地列表是否保存在 `localSources` / `localConsumers` 而不是远端 catalog
  - 看 `internal/services/stream/publish.go` 是否找到 active producer deliveries

## 关键设计决策与权衡

- 决策：持久化使用 `App` profile storage，而不是 `localStorage`
  - 原因：要和现有多 profile 模型对齐，并保证登录恢复语义稳定
- 决策：自动恢复放在 `AppShell`，不是只在打开 `Stream` 页面时恢复
  - 原因：用户期望恢复后本地 source / consumer 立即可用，而不是依赖页面访问
- 决策：只为 `text` source 补最小输入 binding
  - 原因：满足“向 source 输入东西”的核心诉求，同时不扩展到音视频采集或其它 producer editor
- 决策：页面采用 tab + 列表 + 独立弹窗
  - 原因：保持主画布简洁，并与 `VarPool` 的产品形态对齐

## 测试与验证方式 / 结果

- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./internal/services/stream -count=1`
  - 结果：通过
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test . -count=1`
  - 结果：通过
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：有 `Not found: time.Time` 提示，但退出码为 `0`
- `MyFlowHub-Win/frontend`
  - `npm ci`
  - 结果：通过
- `MyFlowHub-Win/frontend`
  - `npm exec vitest run src/stores/stream.test.ts src/pages/Stream.test.ts`
  - 结果：通过（`2` files, `6` tests）
- `MyFlowHub-Win/frontend`
  - `npm run build`
  - 结果：通过
  - 备注：保留既有大 chunk warning，非本轮新增问题

## 潜在影响与回滚方案

- 潜在影响
  - `Stream` 现在会按 profile 保存本地 source / consumer、tab 和 target
  - 登录后会自动尝试恢复已保存的本地 source / consumer
  - `text` source 可以从 Win 直接发文本到 active local producer deliveries
  - `Stream` 页面主交互已明显收敛为 tab + 列表 + 弹窗，不再是大块内联表单
- 回滚方案
  - 回退以下文件即可恢复到改造前状态：
    - `app_stream.go`
    - `app_stream_test.go`
    - `frontend/src/stores/stream.ts`
    - `frontend/src/stores/stream.test.ts`
    - `internal/services/stream/publish.go`
    - `internal/services/stream/service_test.go`
    - `frontend/src/pages/Stream.vue`
    - `frontend/src/pages/Stream.test.ts`
    - `frontend/src/layout/AppShell.vue`
    - `frontend/src/i18n/messages/stores.ts`
    - `plan.md`
    - `docs/change/2026-03-31_win-stream-product-tabs.md`
    - `docs/change/README.md`

## 子Agent执行轨迹

- 未使用子Agent
