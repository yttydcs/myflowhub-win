# Stream Module Spec

## Scope

- 本文档定义 `MyFlowHub-Win` 中 `Stream` 模块的长期技术边界。
- 模块目标是把上游 `stream` 子协议落为 Win 侧的一等控制面和本地观察 runtime。
- 本文档不定义：
  - 上游 `stream` wire
  - 音视频解码细节
  - 摄像头 / 麦克风 / 屏幕采集

## Interfaces / Contracts

### Go Service

- 新增 `internal/services/stream`
- 暴露 Wails binding `StreamService`
- 首版 bindings 目标：
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

### Business Events

- Go service 负责监听 `session.frame`，解析 `SubProtoStream` 的运行帧，并发布业务事件。
- 首版事件主题建议：
  - `stream.delivery`
    - 控制面结果或 runtime 状态变化摘要
  - `stream.text`
    - 仅承载可直接显示的文本片段
  - `stream.stats`
    - `music` / `video` / `custom` 的统计摘要
- `app.go` 必须把这些事件桥接到 Wails runtime。

### Frontend

- 新增：
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/stores/stream.ts`
- `frontend/src/router/index.ts` 注册 `/stream`
- `frontend/src/layout/AppShell.vue` 增加导航入口

## Data Model Or Protocol

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

### Local Runtime Model

- Go 本地维护轻量 runtime，而不是把所有高频 payload 直接透传给前端。
- 建议最小本地状态：
  - `knownDeliveries[deliveryID]`
    - `kind`
    - `producer`
    - `consumer`
    - `consumerID`
    - `sourceID`
    - `state`
    - `lastFrameAt`
    - `lastAckAt`
    - `bytesIn`
    - `framesIn`
    - `lastPosition`
    - `lastPtsMs`
    - `lastError`
  - `textBuffers[deliveryID]`
    - 仅保留有界最近文本内容
- 前端 store 持有业务镜像，不重复承担二进制解码。

### Development Dependency Strategy

- 当前 Win 主线依赖的 `myflowhub-proto v0.1.5` 不包含 `protocol/stream`。
- 在新 Proto patch/tag 正式发布前，本 workflow 允许在 Win worktree 中使用开发态 `replace` 指向：
  - `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
- 该 `replace` 只用于本 workflow 开发与本地验证，后续 release workflow 必须移除并改为正式 semver。

## Error Handling

- 所有 `StreamService` 请求必须：
  - 校验必填参数
  - 用 `SendCommandAndAwait` 等待 `*_resp`
  - `code != 1` 时返回业务错误
- 非法输入示例：
  - 空 `source_id`
  - 空 `consumer_id`
  - 空 `delivery_id`
  - 非法 `kind`
  - 未登录时缺失本地 `node_id`
- runtime 解析异常不得导致前端崩溃：
  - 非法 DATA/ACK 头直接丢弃并记录日志
  - 文本解析失败时降级为 stats 或 hex 摘要，而不是 panic

## Security / Safety

- 页面层不直接发送任意二进制 payload 到远端 delivery。
- `text` 事件桥只发送有界文本内容；不得默认透传完整媒体 payload。
- runtime 必须对文本缓存和统计缓存做有界控制，避免无限增长。
- 前端只能通过明确的 bindings 调用控制面动作，不能散落构造原始 stream payload。

## Performance Constraints

- 高频 DATA/ACK 不应逐条全量穿过 Wails 事件桥。
- `text` 类型应支持批量合并或节流刷新。
- `music` / `video` / `custom` 默认只桥接摘要统计：
  - bytes
  - frames
  - last position
  - last pts
  - flags
- 本地 runtime 应优先复用已有 `session.frame` 订阅模型，不新增重复网络监听。

## Related Requirements

- [../requirements/stream.md](../requirements/stream.md)
- [../../../server-stream-subproto-design/docs/requirements/stream.md](../../../server-stream-subproto-design/docs/requirements/stream.md)

## Related Changes

- 待本次 workflow 完成后补充。
