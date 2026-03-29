# stream-ctrl-await-mismatch

## Summary

- Win 侧某个子协议如果沿用了 `file` 风格的 `CTRL / DATA / ACK` 分流，发送端和 await 解码端都必须同步理解 `KindCtrl`
- 在 Stream 模块里，`StreamService` 初版把控制请求发成了裸 JSON，而服务端和协议文档要求的是 `KindCtrl + JSON`
- 即使只修正了请求发送，如果等待响应的解码逻辑没有在 await 前剥离 `KindCtrl`，最终依然会表现成 `request timed out`

## Lookup Hints

- `stream announce: request timed out`
- `创建本地 Source 失败`
- `KindCtrl`
- `SubProtoStream`
- `announce_resp`
- `SendCommandAndAwait`
- `payload[0]`
- `KindData`
- `KindAck`

## Symptoms

- Stream 页面创建本地 source / consumer 时超时
- 控制动作没有同步结果，但底层连接本身仍然正常
- 日志里能看到 stream 请求发出，却没有业务 resp 返回到调用方

## Impact

- Stream 控制面无法可靠创建 / 查询 / 订阅 / 连接资源
- 用户只能看到统一超时错误，难以分辨是权限、路由还是 wire 失配
- 同一 helper 覆盖的其它 stream 控制动作也会潜伏同类问题

## Trigger Conditions

- 子协议协议实现要求 CTRL payload 以 `KindCtrl` 开头
- Win 侧仍复用裸 `transport.EncodeMessage(...)` 发送请求
- await 解码逻辑只对 `file` 子协议做了 `KindCtrl` 剥离，没有覆盖当前子协议

## Root Cause

- Stream 协议的控制面 wire 约定和 Win `StreamService` 的实际实现不一致：
  - 服务端 handler 只接受 `payload[0] == KindCtrl`
  - Win 初版却发送裸 JSON，所以服务端不会进入 CTRL handler
- 同时，Win 底层依赖的 SDK await 只对 `file` 子协议在响应解码时剥离 `KindCtrl`
- 结果是：
  - 请求可能根本不被服务端接收
  - 即使服务端响应了 `KindCtrl + JSON(action=*_resp)`，SDK await 也可能解不出来，最后暴露为 timeout

## Investigation Trail

- 先确认前端 `announceSource()` 调的是 `AnnounceSimple(...)`
- 再看 `internal/services/stream/service.go`，发现它直接用了裸 `transport.EncodeMessage(...)`
- 对照 `docs/specs/stream.md` 和 `subproto-stream-subproto/stream/handler.go`
  - 发现 stream CTRL handler 只接受 `KindCtrl`
- 再对照 `internal/services/file/service.go` 与 SDK `await/client.go`
  - 发现 `file` 请求会手动补 `KindCtrl`
  - SDK await 也只为 `file` 的响应去掉 `KindCtrl`
- 最终确认是“子协议 CTRL framing / await 解码不一致”导致的系统性 timeout

## Resolution

- 在 Win `StreamService` 内部显式把请求编码为 `KindCtrl + JSON(action,data)`
- 不再依赖 SDK 的通用 `SendCommandAndAwait` 直接匹配 stream CTRL 响应
- 改为基于 `session.frame/state/error` 在 Stream 模块内做局部 await
  - 匹配 `(MsgID, SubProtoStream, action)`
  - 解码前先剥离 `KindCtrl`
- 增加单元测试，固定住 request prefix 和 response decode 两个回归点

## Prevention / Guardrails

- 新增采用 `file` 风格 CTRL framing 的子协议时，必须同时检查：
  - 请求发送是否补 `KindCtrl`
  - await 解码是否去掉 `KindCtrl`
  - 非 CTRL 的 runtime 帧是否会被排除
- 不要只靠 `Action*` 常量一致就假设请求链路正确；先确认 payload framing
- 调试 `request timed out` 时，优先检查：
  - 服务端 handler 的 `payload[0]` 判定
  - Win 侧请求实际发出的首字节
  - await 解码是否在正确的子协议上处理了 CTRL 前缀

## Related Docs

- [2026-03-29_win-stream-announce-timeout.md](../change/2026-03-29_win-stream-announce-timeout.md)
- [stream.md](../requirements/stream.md)
- [stream.md](../specs/stream.md)
- [2026-02-18_win-file-await.md](../change/2026-02-18_win-file-await.md)
