# 2026-04-13_win-stream-desktop-capture

## 变更背景 / 目标

- 在上一轮“本地文件输入 + 接收端严格边接收边播放”基础上，为 `MyFlowHub-Win` 的 `Stream` 增加桌面采集输入。
- 继续保持接收端严格边接收边播放，而不是先接收完整录制再播放。
- 本轮范围收敛为：
  - `video` source 的 `desktop` 输入
  - 浏览器 / WebView `getDisplayMedia() + MediaRecorder`
  - 不含系统音频
  - 不保证采集中途新增 delivery 无缝加入

## 具体变更内容

### 修改

- `docs/requirements/stream.md`
  - 将长期 requirements 从“仅文件输入”扩展为“文件输入 + 桌面采集”。
  - 明确桌面采集只允许 `video` source、必须由用户手势触发、严格边接收边播放、固定 capture start 时的 active deliveries。
- `docs/specs/stream.md`
  - 更新长期技术契约：
    - `ConfigureSourceInputSimple(...)` 支持 `input_kind=file | desktop`
    - 新增 `PublishCaptureChunkSimple(...)`
    - 记录 `session_start` / `final` DATA flags 语义
    - 记录 consumer 侧 `resetMediaRuntimeSession(...)` 与 `mediaUrl?session=<n>` 旋转语义
- `app_stream.go`
  - source 偏好归一化允许 `inputKind=desktop`
  - 对 `desktop` 自动清空 `filePath`
  - 对非 `video` source 自动剔除非法 `desktop`
- `app_stream_test.go`
  - 增加 `desktop` 输入持久化 / 归一化回归
- `internal/services/stream/media.go`
  - `ConfigureSourceInputSimple(...)` 接受 `desktop`
  - 新增 `streamDataFlagSessionStart`
  - consumer media runtime 支持基于 `session_start` 重置 sink / URL / basePosition
- `internal/services/stream/publish.go`
  - 新增 `PublishCaptureChunkReq` / `PublishCaptureChunkResp`
  - 新增 `PublishCaptureChunk(...)` / `PublishCaptureChunkSimple(...)`
  - 只允许 `video + inputKind=desktop + chunk` 的 active deliveries
- `internal/services/stream/service_test.go`
  - 增加 `desktop` input 校验
  - 增加桌面采集 chunk publish 回归
  - 增加 `session_start` 驱动 consumer runtime reset 回归
- `frontend/src/stores/stream.ts`
  - `announceSource(...)` 允许 `video + inputKind=desktop` 不带文件路径
  - 新增 `publishCaptureChunk(...)`
  - 归一化 `deliveryIds`
- `frontend/src/stores/stream.test.ts`
  - 增加桌面采集 binding 覆盖
  - 增加桌面采集 source announce 覆盖
- `frontend/src/pages/Stream.vue`
  - source 对话框增加 `Local File` / `Desktop Capture` 输入模式
  - `desktop` 模式隐藏文件配置，显示本轮边界说明
- `frontend/src/pages/Stream.test.ts`
  - 增加桌面采集配置 UI 覆盖
- `frontend/src/windows/StreamSourceWindow.vue`
  - 新增桌面采集分支：
    - 检测 `getDisplayMedia()` 与 `MediaRecorder`
    - 用户点击后开始 / 停止采集
    - 本地 `<video>` 预览
    - 固定 capture start 时的 active `deliveryIds`
    - 第一个有效 chunk 带 `sessionStart`
    - 停止时发送 `final`
    - 在 track ended、无 active delivery、窗口关闭、source 配置变化时自动停机
- `frontend/src/windows/StreamSourceWindow.test.ts`
  - 覆盖“首个 chunk 带 sessionStart，停止时发送 final”的窗口级回归
- `frontend/src/i18n/messages/stores.ts`
  - 补充桌面采集状态、错误和说明文案
- `plan.md`
  - 回填 Iteration 2 的 Stage 3.2 / 3.3 / 4 状态与审查结果

### 新增

- `docs/lessons/stream-desktop-capture-session-reset.md`
  - 沉淀“固定 delivery 集合 + `session_start` + 接收端 session reset”的复用规则。

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

## Related lessons

- `docs/lessons/stream-desktop-capture-session-reset.md`
- `docs/lessons/stream-media-progressive-playback-http-outlet.md`
- `docs/lessons/wails-binding-proto-drift.md`

## 对应 plan.md 任务映射

- `WIN-DC-DOC-1`
  - 更新长期 requirements/specs，把桌面采集纳入稳定真相。
- `WIN-DC-BE-1`
  - 为 stream backend 增加 `desktop` 输入与 capture chunk publish 路径。
- `WIN-DC-FE-1`
  - 为 Stream source dialog / source window / store 增加桌面采集交互。
- `WIN-DC-VAL-1`
  - 执行 Go / Vitest / Wails / build 验证并完成 review。

## 经验 / 教训摘要

- 桌面采集不能简单等价为“没有文件路径的 video source”，它需要显式的会话边界。
- 最小可用方案不是“动态追踪所有新 delivery”，而是 capture start 时固定目标 delivery 集合，避免晚加入者拿不到会话起始片段。
- `session_start` 与接收端 `mediaUrl` 旋转是同一件事的两端：前者声明新会话开始，后者强制播放器切到新资源。

## 可复用排查线索

- 症状
  - 点击开始采集后没有任何 delivery 开始播放
  - 第二次开始采集后接收端停在旧画面
  - 采集中途新连入的 delivery 没有完整画面
- 触发条件
  - `inputKind` 不是 `desktop`
  - source 或 delivery 不是 `chunk`
  - 首个有效 chunk 没带 `session_start`
  - capture start 时没有 active delivery
- 关键词
  - `PublishCaptureChunkSimple`
  - `streamDataFlagSessionStart`
  - `resetMediaRuntimeSession`
  - `getDisplayMedia`
  - `MediaRecorder`
  - `delivery_ids`
- 快速检查
  - 确认 source 对话框把 `desktop` 限制在 `video`
  - 确认 Source Window 启动采集时抓取的是当前 active delivery 列表
  - 确认第一块有效 payload 传了 `session_start=true`
  - 确认 `MediaSnapshot()` 中 URL 在新会话时切到了新的 `?session=<n>`

## 关键设计决策与权衡

- 选择浏览器 / WebView `getDisplayMedia() + MediaRecorder`，而不是原生 Go / OS 级桌面采集。
  - 优点：最小改动面，复用现有媒体 runtime。
  - 代价：依赖当前运行时支持，且权限必须由用户手势触发。
- 选择 capture start 时固定 delivery 集合，而不是让新 delivery 中途自动加入。
  - 优点：不需要额外设计补关键帧或独立 recorder。
  - 代价：中途新增 delivery 需要重启采集才能进入当前会话。
- 选择 `session_start` 重置 consumer sink，而不是把新采集会话拼接到旧 sink。
  - 优点：接收端可以稳定重启播放。
  - 代价：需要维护 `basePosition` 和 `sessionSeq`。

## 测试与验证方式 / 结果

- Go：
  - `$env:GOWORK='off'; go test ./... -count=1`
  - 结果：通过。
- 前端定向测试：
  - `npx vitest run src/stores/stream.test.ts src/pages/Stream.test.ts src/windows/StreamSourceWindow.test.ts`
  - 结果：通过。
- Wails bindings：
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过；控制台仍有既有 `Not found: time.Time` 噪声，但生成未失败。
- 前端构建：
  - `npm run build`
  - 结果：通过。
- 手工 / UI：
  - 未在真实 Wails 会话中做端到端桌面采集播放冒烟。
  - 原因：本轮已完成代码级、绑定级与构建级验证，但未额外拉起真实桌面共享与远端 delivery 环境。

## 潜在影响与回滚方案

### 潜在影响

- 当前桌面采集只覆盖桌面视频，不带系统音频。
- capture start 时固定目标 delivery 集合，晚加入 delivery 不会自动进入当前会话。
- 当前仍依赖浏览器原生对 `MediaRecorder` 输出格式的播放支持；若目标运行时不支持对应容器 / 编码，会进入错误降级。

### 回滚方案

- 回退以下文件即可撤销本轮桌面采集能力：
  - `app_stream.go`
  - `app_stream_test.go`
  - `internal/services/stream/media.go`
  - `internal/services/stream/publish.go`
  - `internal/services/stream/service_test.go`
  - `frontend/src/stores/stream.ts`
  - `frontend/src/stores/stream.test.ts`
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/pages/Stream.test.ts`
  - `frontend/src/windows/StreamSourceWindow.vue`
  - `frontend/src/windows/StreamSourceWindow.test.ts`
  - `frontend/src/i18n/messages/stores.ts`
- 若同时不保留长期文档，再回退：
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
  - `docs/lessons/README.md`
  - `docs/lessons/stream-desktop-capture-session-reset.md`
  - `docs/change/2026-04-13_win-stream-desktop-capture.md`
  - `docs/change/README.md`

## 子Agent执行轨迹

- 未派发子Agent。
