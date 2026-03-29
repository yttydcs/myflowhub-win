# 2026-03-29 Win Stream announce timeout

## 变更背景 / 目标

- 用户在 Stream 页面创建本地 source 时，前端提示 `创建本地 Source 失败。`
- 后端错误收敛为 `stream announce: request timed out`
- 本轮目标是在不改前端契约和 stream 协议文档的前提下，修复 Win `StreamService` 的 CTRL framing / await 链路，并把同类控制动作一起收敛到同一条正确实现路径

## 具体变更内容

### 修改：stream 控制面改为显式 CTRL framing

- `internal/services/stream/service.go`
  - 新增 `encodeStreamCtrlPayload(...)`
    - 请求 payload 从裸 JSON 改为 `KindCtrl + JSON(action,data)`
  - 新增 `decodeStreamCtrlMessage(...)`
    - 响应解码前先剥离 `KindCtrl`
  - 新增局部 stream await helper
    - 通过 `session.frame/state/error` 监听匹配 `(MsgID, SubProtoStream, action)` 的 CTRL 响应
    - 不再依赖 SDK 对 `stream` 子协议的通用 CTRL 解码
  - 继续复用现有 `code/msg` 业务错误判定与 `toUIError(...)` 映射

### 修改：补充 stream 控制面回归测试

- `internal/services/stream/service_test.go`
  - 新增 fake session 测试
    - 断言 `announce` 请求 payload 带 `KindCtrl`
    - 断言 `announce_resp` CTRL 帧能被正确匹配并返回
  - 新增 timeout 测试
    - 断言未收到响应时错误仍映射为 `stream announce: request timed out`

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `updated`

## Related requirements

- `docs/requirements/stream.md`

## Related specs

- `docs/specs/stream.md`
- `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\specs\stream.md`

## Related lessons

- `docs/lessons/stream-ctrl-await-mismatch.md`

## 对应 plan.md 任务映射

- `STREAM-BE-1`
  - 修复 stream CTRL payload 编码与局部 await
- `STREAM-TEST-1`
  - 增加 `announce` CTRL request / response 回归测试
- `STREAM-VAL-1`
  - 执行 stream package 验证
- `STREAM-REVIEW-1`
  - 完成 3.3 checklist
- `STREAM-ARCHIVE-1`
  - 写入 change / lesson 归档

## 经验 / 教训摘要

- `stream` 子协议的 CTRL wire 不是裸 JSON，而是 `KindCtrl + JSON`
- 仅修请求编码不够；如果 await 解码端没有同步剥离 `KindCtrl`，最终仍会表现为 timeout
- 当某个子协议借用了 `file` 风格的 CTRL/DATA/ACK 三分流时，Win 侧 send path 和 await path 必须一起检查，不能只看 action 常量

## 可复用排查线索

- 症状
  - Stream 页面创建本地 source / consumer 超时
  - 后端报 `stream announce: request timed out`
  - 其它 stream 控制动作也可能无回包
- 触发条件
  - Win `StreamService` 发送裸 JSON，没有 `KindCtrl`
  - SDK await 只为 `file` 子协议剥离 CTRL 前缀
- 关键词
  - `stream announce: request timed out`
  - `KindCtrl`
  - `SubProtoStream`
  - `SendCommandAndAwait`
  - `announce_resp`
- 快速检查
  - 看 `internal/services/stream/service.go` 请求 payload 是否以 `0x01` 开头
  - 看响应解码是否先去掉 `KindCtrl`
  - 对照 `file` 子协议 await 逻辑确认有没有复用同样的 framing

## 关键设计决策与权衡

- 决策：在 Win `stream` 模块内做局部 await，而不直接改 SDK
  - 原因：本轮用户问题集中在 Win stream 控制面；局部修复能以最小变更面恢复功能
- 决策：共享 helper 覆盖全部 stream 控制动作，而不是只给 `announce` 打补丁
  - 原因：所有 stream 控制动作都走同一个 CTRL wire 约束，单点修复会留下后续回归

## 测试与验证方式 / 结果

- `MyFlowHub-Win` worktree
  - `$env:GOWORK='off'; go test ./internal/services/stream -count=1`
  - 结果：通过
  - 备注：由于当前 bug worktree 的 `go.mod replace` 在嵌套 worktree 下路径不成立，验证时临时创建了一个只用于测试的 junction 指向真实 `proto-stream-subproto`，测试后已清理

## 潜在影响

- 影响范围限定在 Win `StreamService` 控制面
- `stream` runtime 的 `KindData/KindAck` 路径未变
- 其它已稳定子协议（Auth / Flow / File / TopicBus / VarPool / Management）未改

## 回滚方案

- 回退：
  - `internal/services/stream/service.go`
  - `internal/services/stream/service_test.go`
  - `docs/change/2026-03-29_win-stream-announce-timeout.md`
  - `docs/change/README.md`
  - `docs/lessons/stream-ctrl-await-mismatch.md`
  - `docs/lessons/README.md`
  - `plan.md`

## 子Agent执行轨迹

- 未使用子Agent
