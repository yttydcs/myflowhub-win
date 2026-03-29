# 2026-03-29 Win Stream Local Owner Support

## 变更背景 / 目标

- 在修复 `stream` 的 `KindCtrl` / await 解码错位后，用户仍然反馈本地 Source 创建失败，错误继续表现为：
  - `创建本地 Source 失败。`
  - `stream announce: request timed out`
- 进一步排查发现，真正卡住的不是启动脚本或旧 build，而是 Win 侧只实现了“主动发请求 + 观察 DATA/ACK”，没有实现 leaf owner 的 inbound CTRL 处理。
- 当服务端把 owner 请求或 private delivery 动作路由回本机 Win 节点时，Win 没有回 `*_resp`，因此本地 Source / Consumer 仍然不可用。
- 本轮目标是在不修改前端契约和协议 wire 的前提下，补齐 Win 的本地 owner 控制面和最小 delivery 生命周期能力。

## 具体变更内容

### 修改：Win `StreamService` 增加本地 owner catalog

- `internal/services/stream/service.go`
  - `StreamService` 新增本地 `sources` / `consumers` / `producerDeliveries` / `consumerDeliveries` 状态
- `internal/services/stream/local_owner.go`
  - 新增 inbound `MajorCmd` 处理：
    - `announce`
    - `withdraw`
    - `list_sources`
    - `get_source`
    - `announce_consumer`
    - `withdraw_consumer`
    - `list_consumers`
    - `get_consumer`
  - 对 routed owner 请求复用原 `MsgID` 回 `MajorOKResp`
  - 本地 catalog 做 descriptor 校验、冲突检测和 list/get 查询

### 修改：补齐 private delivery lifecycle

- `internal/services/stream/local_owner.go`
  - 新增 private action 结构与处理：
    - `delivery_prepare`
    - `delivery_activate`
    - `delivery_abort`
    - `delivery_close`
  - 本地维护 producer / consumer delivery 状态，并复用 `stream.delivery` 快照对前端曝光
- `internal/services/stream/runtime.go`
  - `session.frame` 现在同时处理：
    - `MajorCmd` 的 inbound owner/private CTRL
    - `MajorMsg` 的 DATA / ACK
  - session 断开时同时清理本地 catalog 与 delivery 状态

### 修改：本地 consumer 收到 DATA 后回 ACK

- `internal/services/stream/runtime.go`
  - 当 active consumer delivery 收到方向正确的 `DATA` 时：
    - 更新 runtime snapshot
    - 计算下一期望位置
    - 回发 `ACK`
  - producer owner 收到 `ACK` 时同步更新本地 `AckedPosition`

### 修改：补充回归测试与稳定 spec

- `internal/services/stream/service_test.go`
  - 新增“owner 请求回环到本机 Win 仍能完成”的 announce / announce_consumer 测试
  - 新增 consumer private lifecycle + DATA -> ACK 测试
  - 新增 producer private lifecycle + ACK 更新测试
- `docs/specs/stream.md`
  - 澄清 Win 除主动 bindings 外，还必须承担 leaf owner inbound CTRL 与 consumer ACK 责任

## Requirements impact

- `none`

## Specs impact

- `updated`

## Lessons impact

- `updated`

## Related requirements

- `docs/requirements/stream.md`

## Related specs

- `docs/specs/stream.md`
- `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\specs\stream.md`

## Related lessons

- `docs/lessons/stream-ctrl-await-mismatch.md`
- `docs/lessons/stream-local-owner-ctrl-gap.md`

## 对应 plan.md 任务映射

- `STOWN-1`
  - 澄清 Win local-owner 技术边界
- `STOWN-2`
  - 本地 source / consumer catalog 与 inbound owner CTRL
- `STOWN-3`
  - private delivery lifecycle 与 consumer ACK
- `STOWN-4`
  - owner / private lifecycle / ACK 测试
- `STOWN-5`
  - 3.3 checklist review
- `STOWN-6`
  - change / lesson / index 归档

## 经验 / 教训摘要

- `request timed out` 在 stream 场景里不能只查 await；还要查这个子协议是否要求 Win 节点承担 leaf owner。
- 当服务端会把 owner 请求重新路由回请求者连接时，Win service 必须同时具备“主动客户端”和“被动 owner”两种角色。
- 只实现 viewer runtime 而不实现 consumer ACK，会让 local consumer delivery 停留在“能看状态，不能真正参与协议”的半成品状态。

## 可复用排查线索

- 症状
  - `创建本地 Source 失败`
  - `创建本地 Consumer 失败`
  - `stream announce: request timed out`
  - `stream announce_consumer: request timed out`
- 触发条件
  - 服务端 `stream` 已启用
  - 当前节点是 leaf owner
  - 服务端把 owner 请求或 `delivery_prepare` 路由回 Win
- 关键词
  - `routeOwnerRequest`
  - `delivery_prepare`
  - `delivery_activate`
  - `runtime.go`
  - `MajorCmd`
  - `stream local owner`
- 快速检查
  - 看 Win `internal/services/stream/runtime.go` 是否只处理 `MajorMsg`
  - 看 Win `StreamService` 是否有本地 `sources/consumers` 状态
  - 看 private `delivery_*` 是否在 Win 内有 handler
  - 看 consumer 收到 `DATA` 后是否会发 `ACK`

## 关键设计决策与权衡

- 决策：在 Win 只补 leaf owner 最小能力，不补 coordinator 路由
  - 原因：满足本地 Source / Consumer 真实可用，同时不把 server 角色搬进桌面端
- 决策：继续沿用 `stream.delivery/text/stats` 事件契约
  - 原因：不扩大前端变更面，页面和 store 无需重新设计
- 决策：consumer ACK 在 runtime 内直接发送
  - 原因：ACK 属于协议必要行为，不应依赖前端或后续页面动作

## 测试与验证方式 / 结果

- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./internal/services/stream -count=1`
  - 结果：通过
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 结果：通过

## 潜在影响与回滚方案

- 潜在影响
  - Win `stream` 现在会处理 inbound `MajorCmd` owner/private CTRL，而不只是 viewer runtime
  - session 断开时会同时清空本地 source / consumer catalog 和 delivery 状态
  - 本地 Source 仍然只保证控制面和 lifecycle 闭环，不新增 DATA 发送器
- 回滚方案
  - 回退 `internal/services/stream/service.go`
  - 回退 `internal/services/stream/runtime.go`
  - 回退 `internal/services/stream/local_owner.go`
  - 回退 `internal/services/stream/service_test.go`
  - 回退 `docs/specs/stream.md`
  - 回退本 change / lesson / index 变更

## 子Agent执行轨迹

- 未使用子Agent
