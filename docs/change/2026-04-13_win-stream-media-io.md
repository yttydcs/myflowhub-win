# 2026-04-13_win-stream-media-io

## 变更背景 / 目标

- 把 `MyFlowHub-Win` 的 `Stream` 从“非 text 只看 stats”提升到“本地文件输入 + 接收端严格边接收边播放”的可用基线。
- 本轮输入范围只包含本地媒体文件，不做桌面采集、摄像头或麦克风。
- 接收端必须在支持的音视频格式上于文件未完整接收前开始播放；不支持的格式必须显式降级。

## 具体变更内容

### 修改

- `docs/requirements/stream.md`
  - 将长期 requirements 更新为文件型 `music` / `video` source、`bounded + chunk` 约束、严格边接收边播放，以及失败时的显式降级。
- `docs/specs/stream.md`
  - 更新长期技术契约，补充 `ConfigureSourceInputSimple(...)`、`MediaSnapshot()`、`stream.media`、progressive sink、本地 HTTP outlet、ACK window 发送约束与播放边界。
- `app_stream.go`
  - 扩展 `StreamSavedSource`，持久化 `inputKind` / `filePath`。
  - 增加 `PickStreamMediaFile()`，通过 Wails 文件对话框选择媒体文件并推导 `kind` / `contentType`。
- `app.go`
  - 新增 `stream.media` 事件桥接。
- `app_stream_test.go`
  - 增加 source 输入配置归一化与持久化回归。
- `internal/services/stream/events.go`
  - 新增 `EventStreamMedia` 与 `StreamMediaEvent`。
- `internal/services/stream/service.go`
  - 为 `StreamService` 增加 `sourceInputs`、`fileSenders`、`media`、`mediaServer` 运行态。
  - `Close()` 清理媒体 sender、sink 与 HTTP outlet。
- `internal/services/stream/publish.go`
  - 为 DATA payload builder 增加 flags 版 helper，复用 flags bit0 表达 EOF。
- `internal/services/stream/runtime.go`
  - 非 `text` DATA 改为写入媒体 sink。
  - 在 delivery removal / disconnect / session close 时清理 media runtime。
- `internal/services/stream/local_owner.go`
  - 初始化新状态 map。
  - producer activate 自动启动文件 sender。
  - consumer activate 自动准备 media runtime。
  - withdraw / close / abort 路径补齐媒体清理。
- `internal/services/stream/service_test.go`
  - 增加文件 sender、EOF、consumer media runtime 和 cleanup 回归。
- `frontend/src/stores/stream.ts`
  - 扩展 source 模型、`mediaByDelivery` 状态、`loadMedia()`、`pickMediaFile()`、`updateSourceInput()`、`mediaForDelivery()`。
  - 监听并归一化 `stream.media`。
- `frontend/src/pages/Stream.vue`
  - 非 `text` source 对话框支持文件选择。
  - 选择媒体文件后自动设置 `kind`、`contentType`、`mode=bounded`、`unitMode=chunk`。
- `frontend/src/windows/StreamSourceWindow.vue`
  - 非 `text` source 改为显示已配置媒体文件与替换入口。
- `frontend/src/windows/StreamDeliveryWindow.vue`
  - 支持 `<video>` / `<audio>` 的渐进播放。
  - 在 buffering / ready / error / closed 场景下显式显示状态或降级到 stats。
- `frontend/src/i18n/messages/stores.ts`
  - 补充媒体文件选择、缓冲、渐进播放、错误降级相关文案。
- `frontend/src/pages/Stream.test.ts`
  - 更新 Stream 页面 mock 与 source 新字段覆盖。
- `frontend/src/stores/stream.test.ts`
  - 更新 store mock 与 media binding 覆盖。
- `plan.md`
  - 回填 Stage 3.2 / 3.3 / 4 的执行、review 与归档状态。

### 新增

- `internal/services/stream/media.go`
  - 新增 source 输入配置、媒体文件检测、per-delivery 文件 sender、progressive sink、本地 HTTP outlet 与 `MediaSnapshot()` 实现。
- `docs/lessons/stream-media-progressive-playback-http-outlet.md`
  - 沉淀“严格边接收边播放必须走 Go 数据面 + 本地 HTTP outlet，不能走 Wails 媒体字节桥”的复用经验。

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `updated`

## Related requirements

- `docs/requirements/stream.md`

## Related specs

- `docs/specs/stream.md`
- `D:\project\MyFlowHub3\server-stream-subproto-design\docs\specs\stream.md`

## Related lessons

- `docs/lessons/stream-media-progressive-playback-http-outlet.md`
- `docs/lessons/stream-ctrl-await-mismatch.md`
- `docs/lessons/stream-local-owner-ctrl-gap.md`
- `docs/lessons/wails-binding-proto-drift.md`

## 对应 plan.md 任务映射

- `WIN-DOC-1`
  - 更新 Stream requirements/specs，纳入文件输入、渐进播放和显式降级的长期真相。
- `WIN-BE-1`
  - 扩展 App 层 source 偏好持久化与文件选择入口。
- `WIN-BE-2`
  - 增加 per-delivery 文件 sender、progressive sink、本地 HTTP 播放出口和 media runtime。
- `WIN-FE-1`
  - 扩展 Stream store、source dialog 与 source window，使本地文件输入可配置。
- `WIN-FE-2`
  - 扩展 delivery window，支持 progressive `<audio>` / `<video>` 播放与错误降级。
- `WIN-VAL-1`
  - 执行 Go / 前端 / Wails 验证并完成 Stage 3.3 review。

## 经验 / 教训摘要

- 严格边接收边播放时，最稳妥的分层是“Go 数据面 + 前端状态面”：媒体 DATA 留在 Go runtime，前端只消费 `stream.media` 和本地 URL。
- 本轮只承诺 `bounded + chunk` 的文件型媒体传输，能明确收敛实现复杂度，并为后续桌面采集保留独立 input runner 插槽。
- per-delivery sender 比单一全局 source runner 更适合当前控制面，因为晚加入的 delivery 仍然需要从字节 0 开始发送。

## 可复用排查线索

- 症状
  - delivery 持续收包，但窗口只显示 stats，不开始播放
  - 错误提示 `media playback requires chunk unit mode`
  - 选择媒体文件后 source 创建成功，但 consumer 无法进入播放器态
- 触发条件
  - source 或 delivery 不是 `chunk` 模式
  - `kind` / `contentType` 与文件实际媒体类型不匹配
  - 尝试把原始媒体 payload 直接走 Wails 高频事件桥
  - 浏览器不支持该容器/编码的渐进播放
- 关键词
  - `stream.media`
  - `ConfigureSourceInputSimple`
  - `MediaSnapshot`
  - `progressiveMediaSink`
  - `media playback requires chunk unit mode`
  - `PickStreamMediaFile`
- 快速检查
  - 确认 source dialog 已把非 `text` source 约束到 `mode=bounded` 和 `unitMode=chunk`
  - 确认 consumer delivery 进入 `active` 后已生成 `stream.media.mediaUrl`
  - 确认 `contentType` 与 `kind` 前缀分别为 `audio/` 或 `video/`
  - 若前端播放器报错，先看是否为浏览器原生不支持格式，再判断是否需要未来引入转码/重封装能力

## 关键设计决策与权衡

- 选择复用现有 `stream` wire 与 DATA flags，而不是引入新的媒体子协议。
  - 优点：最小改动面，保留现有 owner / delivery / ACK 机制。
- 选择“progressive sink + 本地 HTTP outlet”，而不是把原始媒体字节穿过 Wails 事件桥。
  - 优点：满足严格边接收边播放，同时控制内存和桥接开销。
  - 代价：当前只适合浏览器原生支持的渐进播放格式。
- 选择 per-delivery sender，而不是单一全局 source runner。
  - 优点：每个 delivery 都能独立从 byte 0 开始、按自身 ACK window 推进。
  - 代价：同一 source 对多个 consumer 时会产生重复文件读取。

## 测试与验证方式 / 结果

- Go：
  - `$env:GOWORK='off'; go test ./... -count=1`
  - 结果：通过。
- 前端定向测试：
  - `npx vitest run src/stores/stream.test.ts src/pages/Stream.test.ts`
  - 结果：通过。
- Wails bindings：
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过；控制台仍有既有 `Not found: time.Time` 噪声，但生成未失败。
- 前端构建：
  - `npm run build`
  - 结果：通过；保留既有的大 chunk warning，本轮未新增构建失败。
- 手工 / UI：
  - 未在真实 Wails 会话中做端到端人工播放冒烟。
  - 原因：本 turn 内完成了代码级、绑定级和构建级验证，但未拉起真实本地/远端 Stream delivery 环境。

## 潜在影响与回滚方案

### 潜在影响

- 本轮只支持文件型媒体输入，不包含桌面采集；若后续要接桌面内容，需新增独立 input kind / runner。
- 媒体播放现在依赖本机 `127.0.0.1` 临时 HTTP outlet 和临时文件 sink；若运行环境限制本地端口访问，delivery 会降级到错误态 + stats。
- 同一 source 同时面向多个 consumer 时会各自读取一次本地文件，当前没有去重发送优化。

### 回滚方案

- 回退以下文件即可撤销本轮媒体 I/O 基线：
  - `app.go`
  - `app_stream.go`
  - `app_stream_test.go`
  - `internal/services/stream/events.go`
  - `internal/services/stream/local_owner.go`
  - `internal/services/stream/media.go`
  - `internal/services/stream/publish.go`
  - `internal/services/stream/runtime.go`
  - `internal/services/stream/service.go`
  - `internal/services/stream/service_test.go`
  - `frontend/src/stores/stream.ts`
  - `frontend/src/stores/stream.test.ts`
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/pages/Stream.test.ts`
  - `frontend/src/windows/StreamSourceWindow.vue`
  - `frontend/src/windows/StreamDeliveryWindow.vue`
  - `frontend/src/i18n/messages/stores.ts`
- 如果确认不保留长期文档，同时回退：
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
  - `docs/lessons/README.md`
  - `docs/lessons/stream-media-progressive-playback-http-outlet.md`
  - `docs/change/2026-04-13_win-stream-media-io.md`
  - `docs/change/README.md`

## 子Agent执行轨迹

- 未派发子Agent。
