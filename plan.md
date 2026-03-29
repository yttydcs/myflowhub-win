# Plan - Win Stream Local Owner Support

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-stream-local-owner`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-stream-local-owner`
- Current Stage: `4`
- External references:
  - `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
    - purpose: shared `protocol/stream` contract
  - `D:\project\MyFlowHub3\worktrees\subproto-stream-subproto`
    - purpose: upstream owner/coordinator behavior reference
  - `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design`
    - purpose: stream stable design and integration behavior reference

## Stage Records

### Initialization
- `guide.md`
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `$m-docs` 的 `SKILL.md`、`references/requirement-impact.md`、`references/lessons-rules.md`、`references/templates.md`
- repo / branch / worktree confirmation
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `fix/win-stream-local-owner`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-win-stream-local-owner`
  - implementation will stay inside this worktree only
- participating modules
  - `internal/services/stream`
  - `internal/services/session`
  - `docs/specs/stream.md`
  - `docs/change/*`
  - `docs/lessons/*`

### Stage 1 - Requirements Analysis
#### 目标
- 让 Win 节点在 `stream` 子协议下真正具备本地 owner 能力，使本地 Source / Consumer 的创建、查询和被 connect / subscribe 后的 delivery 生命周期都能闭环，而不是只做“发请求 + 观察 DATA/ACK”。

#### 范围
- 必须
  - Win `StreamService` 必须接收并处理发往本机 owner 的 `stream` 控制帧：
    - `announce`
    - `withdraw`
    - `list_sources`
    - `get_source`
    - `announce_consumer`
    - `withdraw_consumer`
    - `list_consumers`
    - `get_consumer`
  - Win `StreamService` 必须接收并处理协调节点发来的私有 delivery 动作：
    - `delivery_prepare`
    - `delivery_activate`
    - `delivery_abort`
    - `delivery_close`
  - Win 本地 consumer delivery 收到 `DATA` 后，必须按当前 delivery 状态发送 `ACK`
  - 已有 `stream.delivery` / `stream.text` / `stream.stats` 事件契约保持不变
  - 增加覆盖本地 owner 控制面和 delivery 生命周期的 Go 测试
  - 归档本次根因、修复路径和可复用排查线索
- 可选
  - 对本地 owner 收到的 `signal` 做最小支持，若不扩大范围则只保持现有主动调用链路
- 不做
  - 不修改 `stream` 协议 wire
  - 不修改 Server / SubProto 主线实现
  - 不新增本地 text producer 编辑器或媒体采集能力
  - 不改前端页面契约和 store 对 `stream.*` 事件的消费方式

#### 使用场景
- Win 本机创建 `source` 后，服务端把 owner 请求路由回该 Win 节点，Win 能返回 `announce_resp`
- Win 本机创建 `consumer endpoint` 后，控制侧或远端 producer 可以基于该 endpoint 建立 delivery
- Win 作为 producer owner 或 consumer owner 收到协调节点的 `delivery_prepare/activate/close` 时，能安装、激活和清理本地 delivery 状态
- Win 作为 consumer owner 收到远端 `DATA` 时，能记录 runtime 状态并回 `ACK`

#### 功能需求
- `StreamService` 不能只处理主动发起的 req/resp；还必须处理 inbound owner / private CTRL
- 本地 source / consumer 状态必须在 Win 内存中维护，并供 list/get/withdraw 查询或清理
- 私有 delivery 状态必须区分 producer 侧与 consumer 侧，避免只保留 viewer 摘要
- inbound delivery 生命周期变化必须同步到已有 `DeliverySnapshot()` 和 `stream.delivery` 事件
- inbound `DATA` 必须只对 active consumer delivery 回 `ACK`

#### 非功能需求
- 保持最小改动面，不改前端 API 和 UI 文案
- 避免重复网络监听，继续复用 `session.frame/state/error`
- 非法 CTRL / DATA / ACK 必须显式丢弃并记录 warn，不得 panic 或静默破坏状态
- 本地状态必须有明确 owner 边界，避免把 coordinator 路由逻辑重新实现到 Win

#### 输入输出
- 输入
  - 上游 `stream` 协议契约与 SubProto handler 行为
  - Win 当前 `StreamService` / `runtime.go`
  - 现有 `stream.delivery` / `stream.text` / `stream.stats` 前端契约
- 输出
  - 更新后的 `internal/services/stream/*.go`
  - 更新后的 `internal/services/stream/service_test.go`
  - 如需澄清长期技术边界，更新 `docs/specs/stream.md`
  - 本轮 `docs/change` / `docs/lessons` 归档

#### 边界异常
- 收到空 payload、非法 `KindCtrl`、未知 action、非法 JSON
- `delivery_prepare` 请求字段缺失、kind 不匹配、source / consumer 不存在
- 对已关闭 delivery 再次 `delivery_close`
- 非 active consumer delivery 收到 `DATA`
- 无法发送 ACK 或 CTRL resp

#### 验收标准
- 本地 `AnnounceSimple(...)` / `AnnounceConsumerSimple(...)` 不再因 owner 请求回到 Win 后超时
- `ListSourcesSimple(...)` / `GetSourceSimple(...)` / `ListConsumersSimple(...)` / `GetConsumerSimple(...)` 对本地 owner 数据可返回正确 descriptor
- 模拟 `delivery_prepare -> delivery_activate -> DATA -> delivery_close` 时：
  - 本地 delivery snapshot 正确更新
  - consumer 侧会发送 ACK
  - close 后 delivery 被关闭或移除
- 相关 Go 测试通过

#### 风险
- 如果 Win 私有 delivery 状态机与上游 handler 字段不一致，`connect/subscribe` 仍会在 prepare/activate 阶段失败
- `session.frame` 同时承担主动 await 与被动 owner 处理，若匹配逻辑过宽，可能误消费非本模块帧
- 本轮不实现 producer 发 DATA，因此“本地 source 可被连接”只保证控制面与生命周期闭环，不承诺有真实 payload 输出

#### 问题清单
- none

### Stage 2 - Architecture Design
#### 总体方案（含选型理由 / 备选对比）
- 方案 A（采用）
  - 在 Win `internal/services/stream` 内补一个最小 local-owner 层：
    - 继续保留当前主动请求 `sendAndAwaitJSON(...)`
    - 新增 inbound CTRL 分发，处理本地 owner catalog 与私有 delivery 生命周期
    - 在现有 runtime 上叠加 consumer ACK 发送和 owner delivery 状态复用
  - 理由
    - 直接补足当前产品缺口
    - 不需要修改前端契约
    - 不把 coordinator / hub 路由逻辑搬进 Win
- 方案 B（不采用）
  - 仅修改 Server 路由，避免请求打回 Win
  - 不采用原因：用户需要的是本地 Source / Consumer 真正可用；没有 Win owner 处理能力时，`delivery_prepare/activate` 依然无法完成
- 方案 C（不采用）
  - 在 Win 内完整重实现 coordinator / route table
  - 不采用原因：范围过大，违背“Win 不重新实现 server”边界

#### 模块职责
- `internal/services/stream/service.go`
  - 主动控制请求、CTRL 编码/解码、公共 helper、错误收敛
- `internal/services/stream/runtime.go`
  - 订阅 `session.frame/state/error`
  - 分发 inbound `MajorCmd` owner/private CTRL
  - 处理 `MajorMsg` DATA / ACK
  - 维护 delivery runtime 并发出业务事件
- `internal/services/stream` 新增本地 owner 状态文件
  - 维护 sources / consumers / producerDeliveries / consumerDeliveries
  - 提供 prepare/activate/abort/close、本地 list/get/withdraw helper
- `internal/services/stream/service_test.go`
  - 覆盖主动请求、inbound owner/private CTRL、ACK 和状态收敛

#### 数据 / 调用流
1. Win 页面主动发 `AnnounceSimple(...)`
2. `StreamService` 发送 `KindCtrl + announce`
3. 服务端 owner 路由命中 leaf owner 后，把原请求转发回 Win
4. Win inbound CTRL handler 处理 `announce`，更新本地 source catalog，并用原 `MsgID` 回 `announce_resp`
5. 远端协调节点在 connect / subscribe 时再向 Win 发送 `delivery_prepare` / `delivery_activate`
6. Win 安装本地 producer / consumer delivery，并继续用 `stream.delivery` / `stream.stats` / `stream.text` 镜像状态
7. Win 作为 consumer 收到 `DATA` 后更新 snapshot，再发送 `ACK`

#### 接口草案
- 现有 Wails bindings 不变
- 新增内部 helper
  - inbound CTRL decode / dispatch
  - local source / consumer catalog CRUD
  - private delivery request handlers
  - ACK send helper
  - local ctrl response helper

#### 错误与安全
- 只处理 `SubProtoStream` + `MajorCmd` + `KindCtrl` 的 inbound owner/private 请求
- 未知 action 直接忽略，不构造错误响应污染非本机语义
- 本地 descriptor / delivery 校验失败时返回 `code/msg`，不 panic
- ACK 只在 consumer delivery active 且方向匹配时发送

#### 性能与测试策略
- 性能
  - 不新增额外 bus 订阅
  - 复用现有 `deliveryRuntime` 作为对前端的统一快照，避免重复镜像
- 测试
  - `go test ./internal/services/stream -count=1`
  - 重点覆盖本地 owner announce/list/get、private prepare/activate/close、consumer DATA 回 ACK

#### 可扩展性设计点
- 本地 owner 状态与 viewer runtime 分层，后续若补本地 text producer 或真实播放器，不必重写 owner 目录和 delivery 生命周期
- 私有 `delivery_*` 常量与结构在 Win 内部封装，不污染前端或公开 binding

#### 问题清单
- none

### Stage 3.1 - Planning
#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- docs tree 无需 bootstrap 或 repair
- stable truth
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
- workflow result
  - `docs/change/2026-03-29_win-stream-local-owner.md`
- reusable troubleshooting knowledge
  - `docs/lessons/stream-local-owner-ctrl-gap.md`
- Requirements impact: `none`
- Specs impact: `clarify`
- Related requirements
  - `D:\project\MyFlowHub3\worktrees\fix-win-stream-local-owner\docs\requirements\stream.md`
- Related specs
  - `D:\project\MyFlowHub3\worktrees\fix-win-stream-local-owner\docs\specs\stream.md`
  - `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\specs\stream.md`
- Related lessons
  - `D:\project\MyFlowHub3\worktrees\fix-win-stream-local-owner\docs\lessons\stream-ctrl-await-mismatch.md`
  - `D:\project\MyFlowHub3\worktrees\fix-win-stream-local-owner\docs\lessons\stream-local-owner-ctrl-gap.md`

#### Executable Checklist
- [x] `STOWN-1` 澄清 stream spec 中的 Win local-owner 技术边界
- [x] `STOWN-2` 实现 Win 本地 source / consumer catalog 与 inbound owner CTRL 处理
- [x] `STOWN-3` 实现私有 delivery 生命周期和 consumer ACK 路径
- [x] `STOWN-4` 增补 Go 测试覆盖本地 owner / private delivery / ACK
- [x] `STOWN-5` 完成 3.3 review checklist
- [x] `STOWN-6` 归档 change / lesson / index

#### Task Details
##### `STOWN-1` - Clarify stable stream spec
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-stream-local-owner`
- Goal
  - 把 Win `StreamService` 需要处理 inbound owner/private CTRL 的边界补进长期 spec
- Files
  - `docs/specs/stream.md`
- Acceptance
  - spec 明确本地 owner control handling 与 consumer ACK 责任
- Tests
  - manual doc review
- Rollback
  - 回退 `docs/specs/stream.md`

##### `STOWN-2` - Add local owner catalog and inbound CTRL handling
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-stream-local-owner`
- Goal
  - 让 Win 能处理 routed `announce/list/get/withdraw` 与 consumer 对应动作
- Files
  - `internal/services/stream/service.go`
  - `internal/services/stream/runtime.go`
  - `internal/services/stream/*.go`（如需新增 owner 状态文件）
- Acceptance
  - 本地 owner 控制请求不再超时，list/get 返回本地目录状态
- Tests
  - `go test ./internal/services/stream -count=1`
- Rollback
  - 回退 `internal/services/stream/*`

##### `STOWN-3` - Add private delivery lifecycle and ACK path
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-stream-local-owner`
- Goal
  - 支撑 coordinator 发来的 `delivery_prepare/activate/abort/close`，并在 consumer DATA 路径回 ACK
- Files
  - `internal/services/stream/service.go`
  - `internal/services/stream/runtime.go`
  - `internal/services/stream/*.go`（如需新增内部类型文件）
- Acceptance
  - prepare/activate/close 后 delivery snapshot 正确变化
  - consumer 收到 DATA 后发 ACK
- Tests
  - `go test ./internal/services/stream -count=1`
- Rollback
  - 回退 `internal/services/stream/*`

##### `STOWN-4` - Test coverage
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-stream-local-owner`
- Goal
  - 固定本地 owner 超时回归和 ACK / private lifecycle 行为
- Files
  - `internal/services/stream/service_test.go`
- Acceptance
  - 覆盖 inbound public owner actions、private actions、DATA->ACK
- Tests
  - `go test ./internal/services/stream -count=1`
- Rollback
  - 回退 `internal/services/stream/service_test.go`

##### `STOWN-5` - Review
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-stream-local-owner`
- Goal
  - 按 3.3 checklist 审核需求覆盖、架构边界和测试充分性
- Files
  - `plan.md`
- Acceptance
  - 3.3 各项明确给出 `通过` / `不通过`
- Tests
  - review checklist
- Rollback
  - 更新 `plan.md`

##### `STOWN-6` - Archive
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-stream-local-owner`
- Goal
  - 把本轮根因、实现和排查方法沉淀到 change / lessons / index
- Files
  - `docs/change/2026-03-29_win-stream-local-owner.md`
  - `docs/change/README.md`
  - `docs/lessons/stream-local-owner-ctrl-gap.md`
  - `docs/lessons/README.md`
  - `docs/specs/stream.md`（若 Stage 1/2 判定需澄清）
- Acceptance
  - change / lesson 有明确症状、关键词、快速检查、回滚
- Tests
  - manual doc review
- Rollback
  - 回退本轮 docs 变更

#### Dependencies
- `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
  - 提供 stream proto types
- `D:\project\MyFlowHub3\worktrees\subproto-stream-subproto`
  - 提供 private delivery action 字段和 owner 行为参考

#### Risks and Notes
- 当前 `repo\MyFlowHub-Win\go.mod` 在主仓库有用户未提交变更，禁止直接在主仓库路径实现
- Win 只补 leaf owner 最小能力，不承担 coordinator 路由表职责
- 本轮若发现稳定 spec 与上游实现冲突，再回到 3.1 更新计划

#### Parallelism Assessment
- 不派发子Agent
- 原因
  - 当前会话没有用户显式授权子Agent
  - 变更集中在单模块同一写集合，串行实现更安全

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Task Mapping
- `STOWN-1`
  - `docs/specs/stream.md`
- `STOWN-2`
  - `internal/services/stream/service.go`
  - `internal/services/stream/local_owner.go`
  - `internal/services/stream/runtime.go`
- `STOWN-3`
  - `internal/services/stream/local_owner.go`
  - `internal/services/stream/runtime.go`
- `STOWN-4`
  - `internal/services/stream/service_test.go`
- `STOWN-6`
  - `docs/change/2026-03-29_win-stream-local-owner.md`
  - `docs/change/README.md`
  - `docs/lessons/stream-local-owner-ctrl-gap.md`
  - `docs/lessons/README.md`

#### File-level Change Summary
- `internal/services/stream/service.go`
  - 扩充 `StreamService` 本地 owner 状态
- `internal/services/stream/local_owner.go`
  - 新增 inbound owner/private CTRL 分发
  - 新增本地 source / consumer catalog
  - 新增 private delivery prepare / activate / abort / close
  - 新增 runtime snapshot 与 owner state 协同更新
- `internal/services/stream/runtime.go`
  - `session.frame` 同时处理 `MajorCmd` 与 `MajorMsg`
  - local consumer DATA -> ACK
  - local producer ACK -> `AckedPosition`
  - 断线清理本地 catalog 与 delivery
- `internal/services/stream/service_test.go`
  - 新增 owner loopback、private lifecycle、ACK 回归测试
- `docs/specs/stream.md`
  - 澄清 Win local-owner 与 ACK 责任
- `docs/change/2026-03-29_win-stream-local-owner.md`
  - 归档本轮实现与验证
- `docs/lessons/stream-local-owner-ctrl-gap.md`
  - 固化“leaf owner 控制面缺失”排查规则
- `docs/change/README.md`
  - 更新归档入口
- `docs/lessons/README.md`
  - 更新 lesson 入口

#### Design Notes
- 复用现有 `stream.delivery/text/stats` 前端契约，不改页面 / store API
- Win 只补 leaf owner 最小控制面，不引入 coordinator 路由表或权限裁决
- local owner 状态与 viewer runtime 分层，避免后续本地 producer / player 能力接入时重写当前边界

#### Validation
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./internal/services/stream -count=1`
  - 结果：通过
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 结果：通过

#### Blockers
- none

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 本地 Source / Consumer 创建、查询、本地 owner private lifecycle 与 consumer ACK 都已补齐
- 架构合理性：通过
  - 变更限制在 Win `stream` 模块内部，没有把 server coordinator 职责搬进客户端
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 继续复用单一 `session.frame` 订阅；新增状态是内存 map，没有额外网络监听
- 可读性与一致性：通过
  - owner/private CTRL 逻辑集中到 `local_owner.go`，避免在 `service.go` / `runtime.go` 分散拼装
- 可扩展性与配置化：通过
  - local owner 状态和 viewer runtime 解耦，后续可以独立扩展 producer / player
- 稳定性与安全：通过
  - inbound CTRL 只处理 `SubProtoStream + MajorCmd + KindCtrl`
  - 非法 delivery / role / descriptor 返回显式 `code/msg`
- 测试覆盖情况：通过
  - 覆盖 owner request loopback、private prepare/activate/close、consumer DATA -> ACK、producer ACK 更新
- 子Agent治理与审计：通过
  - 未使用子Agent

### Stage 4 - Change Archive
#### $m-docs Check
- 使用 `$m-docs` 校验 plan/change/lessons 路由
- Requirements impact: `none`
- Specs impact: `updated`
- Lessons impact: `updated`
- 新增：
  - `docs/change/2026-03-29_win-stream-local-owner.md`
  - `docs/lessons/stream-local-owner-ctrl-gap.md`
- 更新：
  - `plan.md`
  - `docs/specs/stream.md`
  - `docs/change/README.md`
  - `docs/lessons/README.md`

#### Archive Status
- 已完成 repo-local 归档
- 等待用户确认是否结束 workflow
