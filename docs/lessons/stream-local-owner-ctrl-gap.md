# stream-local-owner-ctrl-gap

## Summary

- Stream 的 `request timed out` 并不只会来自 `KindCtrl` / await 解码错位。
- 当服务端把 owner 请求或 private delivery 动作路由回 leaf Win 节点时，如果 Win `StreamService` 只实现了主动 req/resp，没有实现 inbound `MajorCmd` owner 控制面，同样会超时。
- 对 local consumer 来说，即使 control-plane 建链成功，如果 Win 不回 `ACK`，也仍然不是完整的协议参与者。

## Lookup Hints

- `stream announce: request timed out`
- `stream announce_consumer: request timed out`
- `创建本地 Source 失败`
- `delivery_prepare`
- `routeOwnerRequest`
- `MajorCmd`
- `ACK`
- `runtime.go`

## Symptoms

- 本地 Source / Consumer 在 UI 中创建失败，错误仍然是统一 timeout
- `stream` 页面能看到基础 UI 和 runtime 观察能力，但本地 owner 资源无法真正建立
- 打开服务端实现会发现 owner 请求被转回 leaf 连接，Win 却没有任何对应响应

## Impact

- 本地 Source / Consumer 不能用
- 远端协调节点对 Win owner 发起的 `delivery_prepare/activate/close` 无法完成
- local consumer 不能正确回 ACK，delivery 只能停留在“观察状态”

## Trigger Conditions

- 当前节点是 `stream` 的 leaf owner
- 服务端 / SubProto 的 owner 路由已经启用
- Win `internal/services/stream/runtime.go` 只处理 `MajorMsg`
- Win `StreamService` 没有本地 source / consumer / private delivery 状态

## Root Cause

- Win `StreamService` 初版是按“桌面端客户端”视角实现的：
  - 主动发控制请求
  - 监听 DATA / ACK 做 viewer
- 但上游 `stream` 协议实际要求 leaf owner 还要承担：
  - owner catalog 的 inbound CTRL
  - private `delivery_prepare/activate/abort/close`
  - consumer ACK
- 结果是服务端已经把请求路由到正确的 Win 连接，但 Win 自身没有处理和回包能力。

## Investigation Trail

- 先确认 `run-dev.ps1` 启动的仓库路径和主线提交都已经是最新
- 再核对 `stream` 子协议 server / subproto 代码，发现 owner 请求会被转回请求者连接
- 对照 Win `internal/services/stream/runtime.go`
  - 发现它只处理 `MajorMsg`
- 对照 Win `internal/services/stream/service.go`
  - 发现只有主动请求 helper，没有 inbound owner/private CTRL 状态机
- 最终确认真正缺的是 Win 本地 owner 控制面，而不是脚本或旧 build

## Resolution

- 在 Win `internal/services/stream` 内补本地 `sources` / `consumers` / `producerDeliveries` / `consumerDeliveries`
- 处理 inbound `MajorCmd + KindCtrl`：
  - owner catalog 动作
  - private `delivery_prepare/activate/abort/close`
- 复用原 `MsgID` 回 `MajorOKResp`
- local consumer 收到 `DATA` 后按 active delivery 计算下一位置并发送 `ACK`
- 增加回归测试固定住 owner request loopback 和 private lifecycle

## Prevention / Guardrails

- 新增桌面端子协议模块时，不要只问“能不能主动发请求”；还要问：
  - 服务端会不会把 owner 请求路由回这个客户端？
  - 是否存在 private lifecycle 动作？
  - 客户端是否需要在 DATA/ACK 面承担协议职责？
- 排查 `request timed out` 时，顺序建议：
  1. 查 CTRL framing / await 是否正确
  2. 查服务端是否把请求路由回 leaf owner
  3. 查 Win 是否处理 inbound `MajorCmd`
  4. 查 consumer 是否真正回 ACK

## Related Docs

- [2026-03-29_win-stream-local-owner.md](../change/2026-03-29_win-stream-local-owner.md)
- [2026-03-29_win-stream-announce-timeout.md](../change/2026-03-29_win-stream-announce-timeout.md)
- [stream-ctrl-await-mismatch.md](stream-ctrl-await-mismatch.md)
- [stream.md](../requirements/stream.md)
- [stream.md](../specs/stream.md)
