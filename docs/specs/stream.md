# Stream Module Spec

## Scope

- 本文档定义 `MyFlowHub-Win` 中 `Stream` 模块的长期技术边界。
- 模块目标是把上游 `stream` 子协议落为 Win 侧的一等控制面、本地 owner runtime，以及面向文本/媒体的观察与播放出口。
- 本文档不定义：
  - 上游 `stream` wire 本身
  - 音视频转码、重封装和编解码器下载
  - 摄像头采集
  - 麦克风采集
  - 桌面采集的系统音频

## Interfaces / Contracts

### Go Service

- `internal/services/stream` 通过 Wails 暴露 `StreamService`。
- 控制面 bindings 包括：
  - `AnnounceSimple(...)`
  - `WithdrawSimple(...)`
  - `ListSourcesSimple(...)`
  - `GetSourceSimple(...)`
  - `AnnounceConsumerSimple(...)`
  - `WithdrawConsumerSimple(...)`
  - `ListConsumersSimple(...)`
  - `GetConsumerSimple(...)`
  - `SubscribeSimple(...)`
  - `UnsubscribeSimple(...)`
  - `ConnectSimple(...)`
  - `DisconnectSimple(...)`
  - `SignalSimple(...)`
  - `PublishTextSimple(...)`
  - `PublishCaptureChunkSimple(...)`
  - `ConfigureSourceInputSimple(...)`
  - `DeliverySnapshot()`
  - `MediaSnapshot()`
- 除主动 bindings 外，`StreamService` 还必须处理发往本机 leaf owner 的 inbound CTRL：
  - owner catalog
    - `announce`
    - `withdraw`
    - `list_sources`
    - `get_source`
    - `announce_consumer`
    - `withdraw_consumer`
    - `list_consumers`
    - `get_consumer`
  - private delivery lifecycle
    - `delivery_prepare`
    - `delivery_activate`
    - `delivery_abort`
    - `delivery_close`
- `ConfigureSourceInputSimple(...)` 只接受最小输入配置：
  - `source_id`
  - `input_kind`
  - `file_path`
- `input_kind` 支持：
  - `file`
    - 仅非 `text` source 使用，且要求 `file_path`
  - `desktop`
    - 仅 `video` source 使用，且 `file_path` 必须为空
- `PublishCaptureChunkSimple(...)` 只接受桌面采集推送：
  - `source_id`
  - `delivery_ids`
  - `pts_ms`
  - `session_start`
  - `final`
  - `payload`

### App Layer

- `app_stream.go` 负责持久化 `StreamPrefs`，并在 `StreamSavedSource` 中保存：
  - `inputKind`
  - `filePath`
- `StreamSavedSource` 的归一化规则：
  - 仅保留 `file | desktop`
  - `desktop` 自动清空 `filePath`
  - 非 `video` source 不得保留 `desktop`
- `main.App` 额外暴露：
  - `PickStreamMediaFile()`
    - 调起 Wails 文件选择器
    - 基于扩展名 / MIME 猜测 `kind` 与 `contentType`
- `app.go` 必须把 `stream.delivery`、`stream.text`、`stream.stats`、`stream.media` 统一桥接到 Wails runtime。

### Business Events

- `StreamService` 监听 `session.frame`，解析 `SubProtoStream` 运行帧，并发布业务事件：
  - `stream.delivery`
    - 控制面结果或 runtime 状态变化摘要
  - `stream.text`
    - 仅承载可直接显示的文本片段
  - `stream.stats`
    - `music` / `video` / `custom` 的统计摘要
  - `stream.media`
    - 面向前端播放器的轻量播放状态
    - 字段包括：
      - `deliveryId`
      - `kind`
      - `contentType`
      - `state`
      - `mediaUrl`
      - `availableBytes`
      - `complete`
      - `error`
      - `updatedAt`
- `stream.media` 只传递播放器状态和本地 HTTP URL，不传递原始媒体字节。

### Frontend

- `frontend/src/stores/stream.ts`
  - 维护 source / consumer / delivery 镜像
  - 维护 `textFramesByDelivery`、`statsByDelivery`、`mediaByDelivery`
  - 提供 `loadMedia()`、`pickMediaFile()`、`updateSourceInput()`、`mediaForDelivery()`
  - 提供 `publishCaptureChunk(...)`
    - 只允许本地 `video + inputKind=desktop`
    - 归一化 `deliveryIds`
    - 转发 `sessionStart` / `final`
- `frontend/src/pages/Stream.vue`
  - `text` source 保持原有配置
  - 非 `text` source 增加输入模式选择：
    - `Local File`
    - `Desktop Capture`
  - 选择媒体文件后自动把 source 约束到：
    - `mode = bounded`
    - `unit_mode = chunk`
    - `inputKind = file`
  - 选择桌面采集后自动把 `video` source 约束到：
    - `mode = bounded`
    - `unit_mode = chunk`
    - `inputKind = desktop`
    - `filePath = ""`
- `frontend/src/windows/StreamSourceWindow.vue`
  - `text` source 使用文本发送器
  - `music` / `video + inputKind=file` 显示已配置文件和替换入口
  - `video + inputKind=desktop` 使用桌面采集 runner：
    - 仅在显式用户点击时调用 `navigator.mediaDevices.getDisplayMedia(...)`
    - 显式检测 `MediaRecorder`
    - 通过本地 `<video>` 做静音预览
    - 在采集启动时固定当前 active `deliveryIds`
    - 通过队列顺序调用 `publishCaptureChunk(...)`
    - 第一个有效 chunk 带 `sessionStart = true`
    - 结束时发送 `final = true`
    - 在 track ended、无 active delivery、窗口关闭或 source 配置变化时停止采集
- `frontend/src/windows/StreamDeliveryWindow.vue`
  - `text` delivery 显示文本帧
  - `music` / `video` delivery 在 `stream.media.mediaUrl` 可用时使用 `<audio>` / `<video>`
  - 不支持或失败时回退到错误提示 + stats

## Data Model / Runtime Model

### Upstream Contract

- 直接依赖 `github.com/yttydcs/myflowhub-proto/protocol/stream`
- 关键常量与结构：
  - `SubProtoStream = 8`
  - `ActionAnnounce` / `ActionConnect` / `ActionDisconnect` / ...
  - `StreamKindMusic` / `StreamKindVideo` / `StreamKindText` / `StreamKindCustom`
  - `SourceDescriptor`
  - `ConsumerDescriptor`
  - `ConnectResp`
  - `SubscribeResp`
  - `StreamDataHeaderV1`
  - `StreamAckHeaderV1`

### Local Runtime State

- Go 本地维护轻量 runtime，而不是把所有高频 payload 直接透传给前端。
- 最小本地状态包括：
  - `sources[sourceID]`
    - 当前 Win 节点声明的 source descriptor
  - `consumers[consumerID]`
    - 当前 Win 节点声明的 consumer descriptor
  - `producerDeliveries[deliveryID]`
    - private prepare / activate 安装的 producer owner 状态
  - `consumerDeliveries[deliveryID]`
    - private prepare / activate 安装的 consumer owner 状态
  - `deliveries[deliveryID]`
    - 面向前端的 delivery 摘要镜像
  - `sourceInputs[sourceID]`
    - `input_kind = file | desktop`
    - `file_path`
  - `fileSenders[deliveryID]`
    - 当前 active producer delivery 对应的文件 sender
  - `media[deliveryID]`
    - consumer 侧媒体播放 runtime
    - 额外维护：
      - `basePosition`
      - `sessionSeq`
  - `mediaServer`
    - 复用的本地 HTTP outlet
- 前端 store 只持有业务镜像，不重复承担二进制解码。

### Source Input Profile

- `StreamSavedSource` 与 `ConfigureSourceInputSimple(...)` 共享同一最小输入模型：
  - `inputKind`
  - `filePath`
- 约束：
  - `text` source 不允许使用任何 source input
  - `file`
    - `file_path` 必须存在、可读、非目录、非空文件
  - `desktop`
    - 仅 `video` source 可配置
    - `file_path` 必须为空
  - source 从 `file` 切换输入文件时，必须取消该 source 现有 sender，并对新的 active delivery 重新启动 sender
  - source 从 `desktop` 切回其他模式时，必须使当前 capture 会话停止

## Media Delivery Design

### Producer Path

- 当 producer delivery 进入 `active`，且对应 source 已配置 `input_kind=file` 时，`StreamService` 启动独立 `fileDeliverySender`。
- `fileDeliverySender` 行为：
  - 从 `file_path` 顺序读取文件
  - 每块大小固定为 `64 KiB`
  - 根据 consumer ACK 窗口等待发送，不可无限前冲
  - 调用 `buildDataPayloadWithFlags(...)` 发送 DATA
  - 复用 `flags` bit0 表达 EOF
- 当 source 配置为 `input_kind=desktop` 时，不启动 `fileDeliverySender`。
- 桌面采集改由 `PublishCaptureChunkSimple(...)` 直接送入 producer deliveries：
  - 只允许 `video + chunk + inputKind=desktop`
  - 只允许发送到显式传入且当前已 active 的 `delivery_ids`
  - `session_start=true` 时复用 DATA `flags` bit1
  - `final=true` 时复用 DATA `flags` bit0
- 当前桌面采集发送语义：
  - 采集目标集合固定为开始采集时的 active deliveries
  - 每个 delivery 继续沿用现有 `position` / `ACK` 推进
  - 不引入额外媒体子协议，不做 late-join 补关键帧

### Consumer Path

- 当 consumer delivery 进入 `active`，且 kind 为非 `text` 时，`prepareConsumerMediaRuntime(...)` 尝试创建媒体播放 runtime。
- 播放 runtime 结构：
  - `progressiveMediaSink`
    - 使用临时文件承接顺序写入的媒体字节
    - 维护 `availableBytes`、`complete`、`errText`
  - `mediaHTTPServer`
    - 在 `127.0.0.1` 上按需监听随机端口
    - 通过 `/stream/{deliveryID}` 暴露渐进读取 URL
- `handleData(...)` 对非 `text` DATA 调用 `writeMediaChunk(...)`：
  - 校验 `position` 必须与当前会话的 `basePosition + availableBytes` 对齐
  - 收到 `session_start` 后执行 `resetMediaRuntimeSession(...)`
    - 创建新的 sink
    - 增加 `sessionSeq`
    - 重置 `basePosition`
    - 生成新的 `mediaUrl?session=<n>`
  - 收到 EOF flag 后标记 `complete`
  - 推进媒体状态：
    - `buffering`
    - `ready`
    - `complete`
    - `error`
    - `closed`
- `closeMediaRuntime(...)` 必须在以下场景关闭 sink、删除临时文件并发出 `closed`：
  - delivery 被移除
  - source / consumer withdraw
  - session 断线或 service `Close()`
- 这套设计允许：
  - 桌面采集重新开始时重置播放会话
  - 接收端不需要把 wire position 回退到 0

### Playback Support Boundary

- `mediaPlaybackSupportError(...)` 是当前播放边界的真相：
  - `unit_mode` 必须为 `chunk`
  - `music` 的 `contentType` 必须以 `audio/` 开头
  - `video` 的 `contentType` 必须以 `video/` 开头
  - 仅 `music` / `video` 可进入 runtime 播放路径
- 对不满足上述约束的 delivery：
  - 不创建媒体 sink
  - 通过 `stream.media` 发出 `error`
  - 前端回退到错误提示 + stats
- 本轮只承诺浏览器原生可渐进播放的音视频文件格式；若容器 / 编码不满足浏览器能力，UI 必须显式提示播放失败。

## Development Dependency Strategy

- 当前 Win 主线依赖的 `myflowhub-proto v0.1.5` 不包含 `protocol/stream`。
- 在新 Proto patch/tag 正式发布前，本 workflow 允许在 Win worktree 中使用开发态 `replace` 指向：
  - `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
- 该 `replace` 只用于本 workflow 开发与本地验证，后续 release workflow 必须移除并改为正式 semver。

## Error Handling

- 所有 `StreamService` 请求必须：
  - 校验必填参数
  - 只在 `code == 1` 时视为成功
  - 把业务错误明确返回给前端
- inbound owner/private CTRL 必须：
  - 只处理 `SubProtoStream + MajorCmd + KindCtrl`
  - 复用原请求 `MsgID` 返回 `MajorOKResp`
  - 用 `code/msg` 显式表达 descriptor 缺失、kind 不匹配、delivery 冲突等错误
- 文件型 source 输入必须显式拒绝：
  - 空 `source_id`
  - 空 `file_path`
  - 文件不存在 / 不可读 / 为空目录
  - `text` source 使用文件输入
  - 非 `chunk` 媒体 delivery 请求播放
- 桌面采集必须显式拒绝：
  - 非 `video` source 使用 `desktop`
  - `source` 或 active delivery 不是 `chunk`
  - source 未配置 `input_kind=desktop`
  - 空 `delivery_ids`
  - 空 payload 且 `final=false`
  - `getDisplayMedia()` / `MediaRecorder` 不可用
  - 启动时没有 active deliveries
- runtime 解析异常不得导致前端崩溃：
  - 非法 DATA/ACK 头直接丢弃并记录日志
  - 文本解析失败时降级为 stats 或 hex 摘要，而不是 panic
  - 媒体写入位置错位时将对应 delivery 标记为 `error`

## Security / Safety

- 页面层不直接发送任意二进制 payload 到远端 delivery。
- `text` 事件桥只发送有界文本内容；不得默认透传完整媒体 payload。
- `stream.media` 只发送状态和 URL；不得把原始字节通过 Wails 高频事件桥转发。
- 桌面采集原始字节只允许走前端 -> Go binding -> stream DATA 路径，不允许反向通过事件桥广播。
- runtime 必须对文本缓存、统计缓存和媒体临时文件做有界控制与及时清理。
- Win 只补 leaf owner 最小能力，不承担 coordinator 路由表或权限裁决职责。

## Performance Constraints

- 高频 DATA/ACK 不应逐条全量穿过 Wails 事件桥。
- `stream.stats` 与 `stream.media` 使用节流发射，默认以 `streamStatsEvery = 250ms` 为最小摘要间隔。
- 文件 sender 必须尊重 ACK 窗口，避免在 consumer 未确认时无限发送。
- 桌面采集必须以 `MediaRecorder.start(timeslice)` 的 chunk 粒度推送，而不是逐帧事件。
- 本地 runtime 应优先复用已有 `session.frame` 订阅模型，不新增重复网络监听。

## Related Requirements

- [../requirements/stream.md](../requirements/stream.md)
- [../../../server-stream-subproto-design/docs/requirements/stream.md](../../../server-stream-subproto-design/docs/requirements/stream.md)

## Related Changes

- [../change/2026-04-13_win-stream-desktop-capture.md](../change/2026-04-13_win-stream-desktop-capture.md)
- [../change/2026-04-13_win-stream-media-io.md](../change/2026-04-13_win-stream-media-io.md)
- [../change/2026-03-29_win-stream-local-owner.md](../change/2026-03-29_win-stream-local-owner.md)
- [../change/2026-03-29_win-stream-announce-timeout.md](../change/2026-03-29_win-stream-announce-timeout.md)
