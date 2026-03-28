# Plan - Win Stream Module Integration

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/win-stream-console`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-console`
- Current Stage: `4`
- External dependencies:
  - `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
    - purpose: provide `protocol/stream` during development
  - `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design`
    - purpose: stream requirements/spec reference and local integration target

## Stage Records

### Initialization
- `guide.md`
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `$m-docs` 的 docs routing / requirement impact 规则
- repo / branch / worktree confirmation
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `feat/win-stream-console`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-console`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 在 Win 中引入一等 `Stream` 模块，接上上游 `stream` 子协议的 control plane，并提供类型感知的本地观察能力。

#### Scope
- 必须
  - 新增独立 `Stream` 模块，不复用当前 `Flow`
  - Go 侧新增 `StreamService`，覆盖 `announce/list/get/subscribe/connect/disconnect/signal` 等控制动作
  - 前端新增 `Stream` 页面与 store
  - 支持列出 sources / consumers，并显示 `kind` / `content_type` / `mode` / `unit_mode` / `tags`
  - 支持创建 / 撤销本地 consumer endpoint
  - 支持 `subscribe` / `unsubscribe`
  - 支持控制侧 `connect` / `disconnect`
  - Go 侧新增 `stream.*` 业务事件与本地 runtime
  - `text` 类型要可读；其余类型至少要显示 delivery 状态与统计
  - 显式记录当前 Proto 版本链缺口，并在开发态使用 worktree 依赖对齐
- 可选
  - 本地 `text` producer
  - 独立 stream viewer window
  - source / consumer 草稿持久化
- 不做
  - 不实现摄像头 / 麦克风 / 屏幕采集
  - 不承诺音视频真实播放
  - 不在本轮发正式 Proto / Win release tag
  - 不把原始媒体 payload 默认走 Wails 事件桥

#### Use Cases
- 用户查询远端 producer sources 并看到 `kind`
- 用户在 Win 上创建 `text` consumer endpoint 并连接到远端 `text` source
- 用户作为控制侧把远端 source 和 consumer 连起来后，直接在 Win 中看到 delivery 建立和数据活动
- 用户查看 `music/video/custom` delivery 的状态、metadata 和吞吐统计

#### Functional Requirements
- Win 必须有独立 `Stream` 路由与导航项
- `StreamService` 必须使用 `SendCommandAndAwait`
- 页面不得直接解析原始 `session.frame`
- Go runtime 必须发布 `stream.delivery` / `stream.text` / `stream.stats` 业务事件
- `text` viewer 必须展示最近文本内容
- `music/video/custom` viewer 必须展示运行统计，而不是空白页
- 当前开发流程必须解决 `myflowhub-proto v0.1.5` 缺失 `protocol/stream` 的问题

#### Non-functional Requirements
- 控制面、runtime、页面 store 分层清晰
- 高频数据不能造成前端无界渲染
- 输入校验和业务错误必须显式返回
- 设计要为后续播放器 / 捕获器扩展留接口

#### Inputs / Outputs
- 输入
  - 当前 session `node_id` / `hub_id`
  - 上游 `protocol/stream` descriptor / resp
  - `session.frame` 中的 `SubProtoStream` 帧
- 输出
  - Wails binding：`StreamService`
  - Wails 业务事件：`stream.delivery` / `stream.text` / `stream.stats`
  - 页面：`/stream`

#### Edge Cases
- 当前依赖的 Proto 主线 tag 不含 `protocol/stream`
- `source.kind != consumer.kind`
- 重复 `disconnect`
- 高频文本导致渲染抖动
- 当前 viewer 不支持直接渲染音视频

#### Acceptance Criteria
- `Stream` 页面可用，且可执行 source / consumer / delivery 控制动作
- `text` delivery 可读
- `music/video/custom` delivery 可见状态和统计
- Go 和前端都不直接依赖手工 raw frame 解码散落在页面
- 本地验证可通过 `go test`、`wails generate module`、`npm run build`

#### Risks
- 当前 `myflowhub-proto v0.1.5` 不含 `stream`，正式 release 前必须补新的 semver 收口
- 若把高频 payload 直接桥接到前端，会造成性能问题
- 音视频渲染若在本轮硬上，会显著扩大实现面

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 方案 A（采用）
  - 新增独立 `internal/services/stream` + `frontend/src/stores/stream.ts` + `frontend/src/pages/Stream.vue`
  - 控制面全部走 `SendCommandAndAwait`
  - `session.frame` 的 `SubProtoStream` 帧由 Go runtime 消费，转换成 `stream.*` 业务事件
  - `text` 类型桥接有界文本片段，其他类型桥接统计摘要
- 不采用方案
  - 把 `stream` 混入 `Flow` 或 `Debug`
    - 理由：语义和用户心智都错位
  - 让页面直接解析原始 `session.frame`
    - 理由：协议细节会扩散到 UI，且不利于高频流量治理
  - 默认把原始媒体 payload 全量发到前端
    - 理由：Wails 事件桥不适合承载高频 bulk media

#### Module Responsibilities
- `go.mod`
  - 在开发态对齐 `protocol/stream` 依赖
- `internal/services/stream`
  - 控制面请求编排
  - DATA/ACK runtime 解码
  - 业务事件发布
- `app.go`
  - 绑定 `StreamService`
  - 桥接 `stream.*` 业务事件
- `frontend/src/stores/stream.ts`
  - 维护 UI state、表单、delivery 视图状态
- `frontend/src/pages/Stream.vue`
  - sources / consumers / deliveries / viewers 组合页
- `frontend/src/router/index.ts`
  - 注册 `/stream`
- `frontend/src/layout/AppShell.vue`
  - 增加导航入口

#### Data / Call Flow
1. 页面调用 `StreamService.ListSourcesSimple` / `ListConsumersSimple`
2. Go 侧编码 `protocol/stream` payload 并 `SendCommandAndAwait`
3. `*_resp` 解包后返回结构化结果给前端
4. 已建立 delivery 后，`session.frame` 收到 `SubProtoStream`
5. Go runtime 解析：
   - `KindCtrl`：必要时更新 delivery 状态
   - `KindData`：
     - `kind=text` -> 发布 `stream.text`
     - `kind!=text` -> 只更新本地统计并发布 `stream.stats`
   - `KindAck` -> 更新本地 delivery stats / 状态
6. `app.go` 把 `stream.*` 桥接到前端
7. 前端 store 更新 viewers 与 delivery 列表

#### Error Handling and Safety
- 所有 control-plane 请求必须做输入校验和 `code` 判定
- 非法 stream payload 或未知 delivery 只记录日志，不允许传播 panic
- 文本桥接有界，避免内存无界增长
- `replace` 仅用于开发态；release 前必须移除

#### Performance and Testing Strategy
- Go:
  - `go test ./internal/services/stream/... -count=1`
  - `go test ./... -count=1 -p 1`
- Frontend:
  - `npm test -- Stream`
  - `npm run build`
- Bindings:
  - `wails generate module`

#### Extensibility Design Points
- `viewer` 与 `runtime` 解耦，后续可以给 `music/video/custom` 换成真实播放器
- `text` / `stats` 事件接口稳定后，可扩到独立 window 或 MCP 工具
- 若 Proto 正式 release 收口，只需移除开发态 `replace`，不应重写 Stream 模块

#### Issue List
- none

### Stage 3.1 - Planning
#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Requirements impact: `add`
- Specs impact: `add`
- Related requirements
  - `D:\project\MyFlowHub3\worktrees\feat-win-stream-console\docs\requirements\stream.md`
- Related specs
  - `D:\project\MyFlowHub3\worktrees\feat-win-stream-console\docs\specs\stream.md`
  - `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\specs\stream.md`
- Related lessons
  - `D:\project\MyFlowHub3\worktrees\feat-win-stream-console\docs\lessons\wails-binding-proto-drift.md`

#### Executable Task List
- [x] `WIN-DOC-1` 新增 Win `stream` requirements/specs 并更新索引
- [x] `WIN-DEP-1` 解决开发态 `protocol/stream` 依赖
- [x] `WIN-BE-1` 新增 `internal/services/stream` 控制面 service
- [x] `WIN-BE-2` 新增 stream runtime 和业务事件桥
- [x] `WIN-FE-1` 新增 `/stream` 路由、导航、页面和 store
- [x] `WIN-FE-2` 补 text viewer 和 generic stats viewer
- [x] `WIN-VAL-1` 执行 Go / Wails / 前端验证
- [x] `WIN-REVIEW-1` 完成 3.3 checklist
- [x] `WIN-ARCHIVE-1` 归档到 `docs/change`

#### Task Details
##### `WIN-DOC-1` - Stream requirements/specs
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-console`
- Goal
  - 把 Win stream 模块的长期需求和技术边界写入 docs 真源
- Files
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
  - `docs/requirements/README.md`
  - `docs/specs/README.md`
- Acceptance
  - Win repo docs tree 可明确回答“Stream 是什么、放在哪、怎么扩展”
- Tests
  - manual doc review
- Rollback
  - 回退新增 docs

##### `WIN-DEP-1` - Development dependency alignment
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-console`
- Goal
  - 让 Win 在当前 workflow 中可编译 `protocol/stream`
- Files
  - `go.mod`
  - `go.sum`
- Acceptance
  - `go test ./internal/services/stream/... -count=1` 可解析 `protocol/stream`
- Tests
  - `go test ./... -count=1 -p 1`
- Rollback
  - 回退开发态 `replace` 或依赖改动

##### `WIN-BE-1` - Stream control plane service
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-console`
- Goal
  - 提供 source / consumer / delivery 的 control-plane bindings
- Files
  - `internal/services/stream/*.go`
- Acceptance
  - `announce/list/get/connect/disconnect/...` 可调用且统一错误处理
- Tests
  - `go test ./internal/services/stream/... -count=1`
- Rollback
  - 回退 `internal/services/stream`

##### `WIN-BE-2` - Stream runtime and event bridge
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-console`
- Goal
  - 在 Go 侧解析 stream 运行帧并发布业务事件
- Files
  - `internal/services/stream/*.go`
  - `app.go`
- Acceptance
  - `stream.delivery` / `stream.text` / `stream.stats` 事件可发到前端
- Tests
  - `go test ./internal/services/stream/... -count=1`
  - `wails generate module`
- Rollback
  - 回退 runtime 和 bridge

##### `WIN-FE-1` - Stream page and store
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-console`
- Goal
  - 提供独立 Stream 页面和状态管理
- Files
  - `frontend/src/router/index.ts`
  - `frontend/src/layout/AppShell.vue`
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/stores/stream.ts`
- Acceptance
  - 页面可查询、连接、断开，并显示 sources / consumers / deliveries
- Tests
  - `npm run build`
  - `npm test -- Stream`
- Rollback
  - 回退页面和 store

##### `WIN-FE-2` - Type-aware viewers
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-console`
- Goal
  - 给 `text`、`music`、`video`、`custom` 提供可区分的 viewer
- Files
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/stores/stream.ts`
  - `frontend/src/stores/stream.test.ts`
- Acceptance
  - `text` 可读；其余类型有非空统计视图
- Tests
  - `npm run build`
  - `npm test -- Stream`
- Rollback
  - 回退 viewer 逻辑

#### Dependencies
- 本地开发态依赖 `proto-stream-subproto` worktree 提供 `protocol/stream`
- 本地集成目标依赖 `server-stream-subproto-design` worktree 或后续正式 release

#### Risks and Notes
- 当前主线 `myflowhub-proto v0.1.5` 不含 `stream`，这是 release blockers，不是实现 blockers
- 若本轮把音视频播放做满，会让实现面显著失控，因此先锁定“控制面完整 + text 真正可用 + 非 text 有状态观察”

#### Parallelism Assessment
- 不派发子Agent
- 原因
  - 当前会话未获得显式子Agent授权
  - 依赖对齐、Go runtime、Wails bindings、前端页面和 docs 强耦合，主 Agent 集成成本更低

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Task Mapping
- `WIN-DOC-1`
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
  - `docs/requirements/README.md`
  - `docs/specs/README.md`
- `WIN-DEP-1`
  - `go.mod`
- `WIN-BE-1`
  - `internal/services/stream/service.go`
- `WIN-BE-2`
  - `internal/services/stream/runtime.go`
  - `internal/services/stream/events.go`
  - `internal/services/stream/service_test.go`
  - `app.go`
- `WIN-FE-1`
  - `frontend/src/router/index.ts`
  - `frontend/src/layout/AppShell.vue`
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/stores/stream.ts`
- `WIN-FE-2`
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/stores/stream.ts`
  - `frontend/src/stores/stream.test.ts`

#### File-level Change Summary
- `docs/requirements/stream.md` / `docs/specs/stream.md`
  - 新增 Win Stream 的长期需求、边界和架构约束
- `docs/requirements/README.md` / `docs/specs/README.md`
  - 挂载 Stream 文档入口
- `go.mod`
  - 开发态 `replace` 指向 `proto-stream-subproto`，补齐 `protocol/stream`
- `internal/services/stream/service.go`
  - 新增 source / consumer / delivery 全控制面 binding 与统一错误处理
- `internal/services/stream/runtime.go`
  - 新增 DATA / ACK runtime 解析、本地 delivery 状态跟踪和有界事件发布
- `internal/services/stream/events.go`
  - 定义 `stream.delivery` / `stream.text` / `stream.stats` 事件载荷
- `internal/services/stream/service_test.go`
  - 覆盖 text DATA 与 ACK stats 事件路径
- `app.go`
  - 注册 `StreamService` 并桥接 `stream.*` 业务事件到 Wails runtime
- `frontend/src/router/index.ts` / `frontend/src/layout/AppShell.vue`
  - 增加 `/stream` 路由与导航入口
- `frontend/src/stores/stream.ts`
  - 新增 Stream store，负责控制面调用、delivery/viewer 状态和 runtime 事件消费
- `frontend/src/pages/Stream.vue`
  - 新增 sources / consumers / deliveries / viewers 组合页
- `frontend/src/stores/stream.test.ts`
  - 覆盖控制面结果归一化、connect/disconnect 状态流转和 runtime 事件镜像

#### Design Notes
- 保持 `Stream` 为独立模块，不复用现有 `Flow`/`TopicBus` 语义
- 原始 `SubProtoStream` 帧只在 Go runtime 解析，Vue 页面只消费业务事件和结构化状态
- `text` 提供真实可读 viewer，`music` / `video` / `custom` 首版只桥接摘要统计，避免把高频媒体 payload 直接穿过 Wails 事件桥
- 当前 `replace github.com/yttydcs/myflowhub-proto => ../proto-stream-subproto` 仅为开发态收口，正式 release 前必须切回 semver 依赖

#### Validation
- `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 结果：通过
- `npm test -- src/stores/stream.test.ts`
  - 结果：通过（1 个文件，2 个用例）
- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：仍会打印 `Not found: time.Time`，但退出码为 0
- `npm run build`
  - 结果：通过
  - 备注：Vite 继续提示主 bundle 超过 `500 kB`，本轮未处理 code splitting

#### Blockers
- none

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 已覆盖 sources / consumers / deliveries 控制面、kind 感知 viewer、runtime 业务事件和依赖链缺口记录
- 架构合理性：通过
  - 控制面、runtime 和页面 store 分层明确，未把协议解码扩散到 Vue 页面
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 高频 runtime 帧在 Go 侧节流为有界 text/stats 事件，未把 bulk media 直接桥到前端
- 可读性与一致性：通过
  - 命名和错误处理沿用现有 service/store 模式，Stream 与 Flow/TopicBus 语义边界清晰
- 可扩展性与配置化：通过
  - `kind`、viewer 和 runtime 事件模型可继续扩到真实播放器或 producer/editor
- 稳定性与安全：通过
  - 输入校验、无效 payload 丢弃、session 断链统一收敛到 delivery 状态关闭
- 测试覆盖情况：通过
  - Go 单测覆盖 runtime 关键路径；前端新增 store 单测覆盖控制面与事件镜像；全量 Go/Wails/Build 验证通过
- 子Agent治理与审计：通过
  - 未使用子Agent

### Stage 4 - Change Archive
#### $m-docs Check
- 使用 `$m-docs` 校验 plan/change/lessons 路由
- Requirements impact: `updated`
- Specs impact: `updated`
- Lessons impact: `updated`
- 新增：
  - `docs/change/2026-03-28_win-stream-module.md`
- 更新：
  - `plan.md`
  - `docs/change/README.md`
  - `docs/lessons/wails-binding-proto-drift.md`
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`

#### Archive Status
- 已完成 repo-local 归档
- 等待用户确认是否结束 workflow
