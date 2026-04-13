# Plan - MyFlowHub-Win Stream Media IO

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/win-stream-media-io`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\win-stream-media-io`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`:
  - `repo/MyFlowHub-Win/guide.md` 不存在
  - 已按 `AGENTS.md` 与 `$m-autoflow` 约束执行：实现仅在 `worktrees/` 中进行，当前先停在分析与计划阶段
- base/worktree confirmation:
  - `repo/MyFlowHub-Win` 默认分支为 `main`
  - 控制面仓库 `D:\project\MyFlowHub3` 仅用于 worktree 管理和后续归档
  - `repo/MyFlowHub-Win` 主仓当前存在未提交改动：`go.mod`、`myflowhub-mcp.exe`
  - 本轮唯一实现 worktree：`D:\project\MyFlowHub3\worktrees\win-stream-media-io`
  - worktree 已切到 `feat/win-stream-media-io`

### Stage 1 - Requirements Analysis
#### Goal
- 把 `MyFlowHub-Win` 的 `Stream` 从“控制面 + 统计观测”提升到“至少一种真实媒体输入 + 接收端可播放输出”的可用基线。
- 保持现有 `text` source / consumer / delivery 语义不回退。
- 在不改 `stream` 协议 wire 的前提下，为后续更强的实时媒体和桌面采集能力预留可扩展结构。

#### Scope
- Must:
  - `Stream` 必须支持至少一种非 `text` 的本地真实输入方式，而不是只声明 descriptor
  - 接收端 `delivery` 必须在支持的 `audio/video` 内容类型下提供实际可播放的输出，而不是只显示 stats
  - 新能力必须继续沿用现有 `source` / `consumer` / `delivery` 控制面，不新造并行模块
  - 现有 `text` 发送、`music/video/custom` stats、连接/订阅/断开路径不得回退
  - 非法输入、空路径、不支持的媒体类型、播放失败和临时文件清理失败都必须显式处理
- Optional:
  - 在播放窗口显示进度、缓冲状态、文件名和内容类型等辅助信息
- Not in scope:
  - 不改上游 `stream` 子协议 wire
  - 本轮不做桌面采集
  - 不做摄像头、麦克风采集
  - 不做转码器、编解码器下载器或跨格式转封装平台
  - 不把 `Stream` 做成完整 NLE/播放器产品

#### Use Cases
1. 用户在 `Stream` 中创建本地 `video` source，选择一个本地视频文件作为输入，连接到本地或远端 consumer 后，接收端可以打开窗口实际播放。
2. 用户在 `Stream` 中创建本地 `music` source，选择一个本地音频文件作为输入，连接后接收端可以实际播放。
3. 用户在 `Stream` 中断开 delivery、关闭窗口或切换 source 时，媒体缓存、渐进播放状态和运行态会被正确清理或降级。

#### Functional Requirements
1. `Stream` source 创建或编辑路径必须能声明“真实媒体输入配置”，而不是只有抽象 descriptor 字段。
2. 若选择文件输入，Win 必须允许用户挑选本地媒体文件，并保存最小必要配置。
3. 文件 source 建立 delivery 后，Win 必须能够把文件内容送入现有 `stream` delivery。
4. 接收端 delivery 窗口在 `audio/video` 内容类型受支持时必须在文件尚未完整接收时开始播放，而不是等接收完成后再播放。
5. 对不受支持或尚未就绪的媒体格式，页面必须明确显示“不支持播放/仍在缓冲/尚未完成”，不能伪装为成功。
6. 现有 `Stream` 控制页、source 窗口、delivery 窗口、store 和 Go runtime 必须保持职责清晰，不把大体积媒体 payload 默认全量塞过 Wails 高频事件桥。
7. 本轮默认只承诺浏览器原生可渐进播放的文件格式；对不满足条件的格式必须显式拒绝或降级。

#### Non-functional Requirements
- 架构边界:
  - 继续保持控制面、媒体生产/消费 runtime、前端播放器/窗口三层边界
- 性能:
  - 高频媒体数据不得走当前 `stream.text` / `stream.stats` 这类摘要事件桥
  - 需要有界缓存、临时文件或等价缓冲策略，避免无限内存增长
- 可维护性:
  - `text` 与媒体 source 的差异应收敛在明确的 source profile 或 runner 层，而不是散落在页面判断
- 稳定性:
  - delivery 关闭、会话断开、窗口关闭时必须有清理策略
- 安全性:
  - 本地文件路径、内容类型和目标 delivery 状态必须校验，不得静默失败

#### Inputs / Outputs
- Inputs:
  - 现有 `Stream` source / consumer / delivery 控制面
  - 本地媒体文件路径与文件元信息
- 接收端 runtime delivery 数据流
- Outputs:
  - 持久化后的 source 配置与最小运行偏好
  - 可播放的 `audio/video` delivery 输出窗口
  - 保留现有 stats / text 观测能力

#### Edge Cases
- 用户选择不存在、不可读或格式不受支持的本地文件
- 单个 source 对多个 delivery 同时发送
- delivery 中途断开、重连或重复打开多个播放窗口
- 会话断线后残留缓冲、临时文件或播放句柄
- 视频/音频内容类型与实际文件不匹配
- 大文件导致长时间发送，播放窗口需要明确状态而不是卡死

#### Acceptance Criteria
1. `Stream` 中至少一种真实媒体输入路径可用，不再只是“声明 video/music source”。
2. 支持的媒体文件在接收端窗口可实际播放。
3. 不支持播放时有明确降级提示，仍保留 stats 观测。
4. `text` source / viewer、连接控制和现有测试基线不回退。
5. `go test ./...`、定向前端测试、`wails generate module`、`npm run build` 能通过或明确区分环境性阻塞。

#### Risks
- 当前稳定 requirements/specs 明确把“屏幕采集”和“音视频最终播放渲染”排除在范围外，本轮必须先修正文档，否则代码会与稳定文档冲突。
- 严格边接收边播放要求消费者侧提供渐进可播放资源，而不是简单落盘后再播，实现复杂度显著高于完整接收后播放。
- 浏览器原生播放器对渐进播放的容器格式有要求；若文件不是可渐进播放格式，就必须显式降级。

#### Issue List
- 已确认：本轮只做文件输入，不做桌面采集。
- 已确认：本轮必须严格边接收边播放。

### Stage 2 - Architecture Design
#### Overall Solution
- 推荐方案：先落“文件型媒体 source + 播放型 delivery window”的最小可用闭环，再按同一 source runner 架构扩展桌面采集。
- 方案拆层：
  - 控制层:
    - 保持 `StreamService` 的 `announce/connect/disconnect/...` 契约
    - 为 source 增加明确的本地输入 profile，而不是只靠 `kind + metadataRaw` 猜测
  - 生产层:
    - 在 Go 侧为本地媒体 source 增加 runner，负责读取媒体文件、切块发送、跟踪进度和关闭
    - `text` 继续走现有 `PublishTextSimple`
  - 消费层:
    - 在 Go runtime 为媒体 delivery 增加渐进 sink、本地 HTTP 播放出口和资源管理
    - 用新的轻量业务事件把“播放 URL/缓冲状态/错误”而不是原始媒体 payload 发送给前端
  - 表现层:
    - `StreamSourceWindow` 用于选择文件、显示发送状态
    - `StreamDeliveryWindow` 在支持时渲染 `<video>` / `<audio>` 指向渐进播放 URL，不支持时降级到 stats

#### Alternatives Considered
- 只改前端窗口，直接让 Vue 读取本地文件并推流:
  - 放弃。会把大体积媒体直接压到 Wails 前端侧，破坏当前 Go runtime 分层。
- 把非 `text` 继续保持 stats，不做真实播放:
  - 放弃。不满足当前需求。
- 本轮同时做文件 source、桌面采集、严格边接收边播:
  - 放弃。用户已确认本轮必须严格边接收边播，桌面采集会把变更面扩大过多，按当前范围延后。
- 新造独立“Media”模块，不复用 Stream:
  - 放弃。会与现有 `source/consumer/delivery` 重复建模。

#### Module Responsibilities
- `docs/requirements/stream.md`
  - 把“非 text 只看统计”的旧口径升级为“支持选定的真实媒体输入与播放输出”
- `docs/specs/stream.md`
  - 增加 source profile、媒体 runner、渐进播放出口和缓存/清理约束
- `app_stream.go`
  - 扩展 `StreamPrefs` 与 source 持久化模型，保存新的本地输入 profile
- `app.go`
  - 继续桥接现有 `stream.*`，并桥接新的媒体状态事件
- `internal/services/stream/service.go`
  - 保持控制面 binding，新增文件选择后需要的启动/停止控制入口
- `internal/services/stream/publish.go`
  - 保留 `text` 直发逻辑；媒体文件 source 不应复用这条只适合 text 的路径
- `internal/services/stream/runtime.go`
  - 继续维护 delivery 摘要；对媒体 delivery 增加渐进 sink、播放 URL 和缓冲状态更新
- `internal/services/stream/*`
  - 新增本地媒体 source runner / sink / cleanup 相关实现与测试
- `frontend/src/stores/stream.ts`
  - 扩展 source 类型、保存/恢复 profile、消费媒体状态事件、为窗口提供 player 状态
- `frontend/src/pages/Stream.vue`
  - source 弹窗增加输入模式与受控配置入口
- `frontend/src/windows/StreamSourceWindow.vue`
  - 从仅 text 输入，扩展为 text 与媒体文件 source 的专用输入窗口
- `frontend/src/windows/StreamDeliveryWindow.vue`
  - 从 text/stats 窗口扩展为 text/stats/player 三态
- `frontend/src/router/index.ts`
  - 维持现有窗口路由，必要时补独立媒体窗口参数
- `frontend/src/pages/Stream.test.ts`
  - 回归 source 窗口与 delivery 窗口行为
- `internal/services/stream/service_test.go`
  - 回归 text/stats，新增媒体 source/sink/cleanup 测试

#### Data / Call Flow
1. 用户在 `Stream` 创建 `music` 或 `video` source，并选择本地文件输入 profile。
2. source 配置通过 `app_stream.go` 保存最小必要偏好；descriptor 仍通过 `StreamService` 向当前协议宣告。
3. delivery 进入 active 后，Go 侧媒体 source runner 开始读取本地文件并向活跃 producer deliveries 发送 `DATA`。
4. consumer 侧 runtime 按 delivery 汇聚媒体字节，维护渐进 sink 和本地 HTTP 播放出口，并通过业务事件把“可播放 URL/状态/错误/统计”同步给前端。
5. `StreamDeliveryWindow` 在收到初始可播放状态后立即用 `<video>` 或 `<audio>` 连接渐进播放 URL；后续边接收边播，不满足条件则显示 stats 或错误态。
6. delivery 关闭、窗口退出或会话断线时，Go runtime 清理对应缓冲和临时资源。

#### Interface Drafts
- `StreamSavedSource` 扩展草案：
  - `inputProfile.kind`
  - `inputProfile.filePath`
  - `inputProfile.autostart`
- Go App/Wails 草案：
  - `PickStreamMediaFile()`
  - `ClearStreamMediaSource(sourceID)`
- Stream store 草案：
  - `mediaStateByDelivery`
  - `sourceInputProfile`
- 业务事件草案：
  - `stream.media`
    - `deliveryId`
    - `contentType`
    - `playbackState`
    - `mediaURL`
    - `availableBytes`
    - `buffering`
    - `error`

#### Error Handling and Safety
- 空 source、空路径、空 delivery、文件不存在、内容类型不支持时必须返回明确错误
- 媒体资源尚未达到可播放门槛时，前端显示“缓冲中”，但一旦满足条件必须尽快启动播放
- 会话断线或 delivery close 时，必须终止对应 runner/sink 并清理临时状态
- 若播放器无法渲染该内容类型，回退到 stats，并保留错误信息

#### Performance and Testing Strategy
- 媒体 payload 不走现有高频前端事件桥；前端只拿资源状态、URL 和摘要
- 需要有界缓存、渐进可读资源或等价策略，避免把完整媒体永远放在内存里
- 测试重点：
  - source profile 归一化与持久化
  - 文件 source 启停与 delivery 关闭清理
  - text 路径不回退
  - delivery window 的 player / buffering / stats / error 四态
  - `wails generate module` 绑定更新

#### Extensibility Design Points
- source runner 按 profile 类型分发，为后续桌面采集保留独立 runner 插槽
- delivery 播放状态与原始字节流分离，为后续更强的 progressive playback 或 capture 编码留余地
- 若后续接入桌面采集，可作为 `video` source 的第二类 profile，而不是重做整页模型

#### Issue List
- 本轮按默认假设只保证浏览器原生可渐进播放的容器/编码组合；不引入转码/重封装依赖。

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 `Stream` 已有独立控制面、text 输入窗口和 delivery stats/text 窗口，但非 `text` 只有统计观察，没有真实媒体输入与播放输出。
- 本轮目标是在不破坏现有 `Stream` 架构的前提下，引入真实媒体 I/O。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验后确认：
  - 稳定真相文档：`docs/requirements/stream.md`、`docs/specs/stream.md`
  - 执行控制文档：当前 worktree 根 `plan.md`
  - 完成归档：`docs/change/2026-04-13_win-stream-media-io.md`
  - Lessons 先关联既有条目；若本轮产生新的媒体缓存/播放器排障模式，再补 `docs/lessons`
- Requirements impact: `add`
- Specs impact: `add`
- Related requirements:
  - `docs/requirements/stream.md`
- Related specs:
  - `docs/specs/stream.md`
- Related lessons:
  - `docs/lessons/stream-ctrl-await-mismatch.md`
  - `docs/lessons/stream-local-owner-ctrl-gap.md`
  - `docs/lessons/wails-binding-proto-drift.md`

#### Executable Task List
- `WIN-DOC-1` 更新 Stream requirements/specs，纳入真实媒体 I/O 的稳定口径
- `WIN-BE-1` 扩展 source 持久化模型与文件选择/输入 profile 入口
- `WIN-BE-2` 增加本地媒体 source runner、delivery 渐进 sink 与本地播放出口
- `WIN-FE-1` 扩展 Stream store / source dialog / source window
- `WIN-FE-2` 扩展 delivery window，支持 audio/video 播放与错误降级
- `WIN-VAL-1` 补测试与验证
#### Task Details
##### WIN-DOC-1 - Update Stream Requirements And Specs
- Owner: 主代理
- Worktree: `D:\project\MyFlowHub3\worktrees\win-stream-media-io`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-stream-media-io\plan.md`
- Goal: 让稳定 requirements/specs 与目标实现一致，不再停留在“非 text 只看统计”
- Files / Modules:
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
  - 必要时 `docs/requirements/README.md`
  - 必要时 `docs/specs/README.md`
- Write Set:
  - 仅上述 docs 文件
- Acceptance:
  - 文档明确记录本轮批准的媒体输入模式和播放语义
- Test Points:
  - 文档链路清楚，无冲突源
- Rollback:
  - 回退本轮 docs 变更

##### WIN-BE-1 - Persist Source Input Profile
- Owner: 主代理
- Worktree: `D:\project\MyFlowHub3\worktrees\win-stream-media-io`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-stream-media-io\plan.md`
- Goal: 扩展 source 偏好与选择入口，使媒体 source 不再只是抽象 descriptor
- Files / Modules:
  - `app_stream.go`
  - 新增或更新 `app_stream_test.go`
  - 必要时 `app.go`
- Write Set:
  - 仅 app 层持久化与绑定相关文件
- Acceptance:
  - source profile 可保存、恢复、校验
- Test Points:
  - prefs normalize
  - 空路径/非法 profile 拒绝
- Rollback:
  - 回退新增字段与 app 绑定

##### WIN-BE-2 - Add Media Source Runner, Progressive Sink, And Playback Outlet
- Owner: 主代理
- Worktree: `D:\project\MyFlowHub3\worktrees\win-stream-media-io`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-stream-media-io\plan.md`
- Goal: 让文件型媒体 source 真正产生 stream DATA，并让 consumer 侧生成可渐进播放的本地资源出口
- Files / Modules:
  - `internal/services/stream/service.go`
  - `internal/services/stream/runtime.go`
  - `internal/services/stream/publish.go`
  - 新增 `internal/services/stream/*`
  - `internal/services/stream/service_test.go`
- Write Set:
  - 仅 `internal/services/stream/`
- Acceptance:
  - 文件 source 能向 active delivery 发送媒体数据
  - consumer 侧能在文件未接收完成前给出可播放资源状态
  - text 路径不回退
- Test Points:
  - source 启停
  - 渐进播放 sink 状态推进
  - delivery close cleanup
  - unsupported content fallback
- Rollback:
  - 回退 `internal/services/stream/` 变更

##### WIN-FE-1 - Extend Stream Source UI And Store
- Owner: 主代理
- Worktree: `D:\project\MyFlowHub3\worktrees\win-stream-media-io`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-stream-media-io\plan.md`
- Goal: 让用户可以配置媒体 source，并看到运行状态
- Files / Modules:
  - `frontend/src/stores/stream.ts`
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/windows/StreamSourceWindow.vue`
  - 必要时 `frontend/src/i18n/messages/stores.ts`
  - `frontend/src/pages/Stream.test.ts`
- Write Set:
  - 仅上述前端 Stream source 相关文件
- Acceptance:
  - 可配置文件型 source
  - text source 仍可发送文本
- Test Points:
  - source dialog
  - source window
  - store 恢复逻辑
- Rollback:
  - 回退 Stream 前端 source 相关变更

##### WIN-FE-2 - Add Delivery Player Window
- Owner: 主代理
- Worktree: `D:\project\MyFlowHub3\worktrees\win-stream-media-io`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-stream-media-io\plan.md`
- Goal: 让 delivery 窗口支持 audio/video 播放，并在失败时降级到 stats
- Files / Modules:
  - `frontend/src/windows/StreamDeliveryWindow.vue`
  - `frontend/src/stores/stream.ts`
  - 必要时 `frontend/src/router/index.ts`
  - 必要时 `frontend/src/i18n/messages/stores.ts`
  - `frontend/src/pages/Stream.test.ts`
- Write Set:
  - 仅 delivery 播放窗口及其直接依赖
- Acceptance:
  - 支持内容类型时可播放
  - 支持内容类型时必须在未完整接收前启动播放
  - 不支持时清楚降级
- Test Points:
  - player / buffering / stats / error 四态
- Rollback:
  - 回退 delivery player 相关变更

##### WIN-VAL-1 - Validation
- Owner: 主代理
- Worktree: `D:\project\MyFlowHub3\worktrees\win-stream-media-io`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-stream-media-io\plan.md`
- Goal: 验证 Go、Wails、前端和关键回归
- Files / Modules:
  - `go test ./...`
  - `wails generate module`
  - `frontend` 定向测试
  - `frontend` build
- Write Set:
  - 无
- Acceptance:
  - 验证结果可复述，可区分代码回归与环境阻塞
- Test Points:
  - Stream tests
  - app prefs tests
  - Wails bindings
  - 前端窗口测试
- Rollback:
  - 无

#### Dependencies
- 当前 `go.mod` 仍带开发态 proto `replace`；在此基础上实现，但正式发布前仍需走 release 对齐
- 若采用文件选择绑定，需要依赖 Wails runtime 文件对话框能力
- 若采用本地 HTTP 渐进播放 URL，需确认当前 Wails/WebView 环境可访问本机监听端口

#### Risks and Notes
- 当前仓库内未发现现成的本地文件选择或媒体播放器实现，意味着本轮要补首个基线实现
- 当前 delivery 窗口是 text/stats 二态，需要谨慎避免把媒体 payload 重新塞回现有事件桥
- `plan.md` 在本 worktree 中已从旧 Showcase 流程重置为本轮 Stream 计划

#### Parallelism Assessment
- 当前已完成 `3.1`，但本平台要求只有在用户显式授权时才可派发子 Agent；本轮未获该授权
- 不派发子 Agent，原因：
  - 主后端 runtime 与前端 player 写集高度耦合
  - 当前平台授权边界不满足

#### Issue List
- 无

阻塞：否
进入 3.2
禁止派发子Agent

### Stage 3.2 - Implementation
#### Task Execution Summary
- `WIN-DOC-1`
  - 已更新 `docs/requirements/stream.md` 与 `docs/specs/stream.md`，把本轮范围收敛为“文件输入 + 严格边接收边播放 + 显式降级”。
  - 已新增 `docs/lessons/stream-media-progressive-playback-http-outlet.md` 并更新 `docs/lessons/README.md`。
- `WIN-BE-1`
  - `app_stream.go` 已扩展 `StreamSavedSource.inputKind/filePath`，并新增 `PickStreamMediaFile()`。
  - `app.go` 已桥接 `stream.media`。
  - `app_stream_test.go` 已补持久化 / 归一化回归。
- `WIN-BE-2`
  - 已新增 `internal/services/stream/media.go`，提供 source 输入配置、媒体文件探测、per-delivery 文件 sender、progressive sink、本地 HTTP outlet 与 `MediaSnapshot()`。
  - `internal/services/stream/service.go`、`runtime.go`、`local_owner.go`、`publish.go` 已接入 `sourceInputs`、`fileSenders`、`media` runtime 与 EOF flags。
  - `internal/services/stream/service_test.go` 已补文件 sender、media runtime 与 cleanup 回归。
- `WIN-FE-1`
  - `frontend/src/stores/stream.ts` 已扩展 source 输入与 `mediaByDelivery`。
  - `frontend/src/pages/Stream.vue` 已支持文件型 source 创建，并自动约束到 `bounded/chunk`。
  - `frontend/src/windows/StreamSourceWindow.vue` 已支持媒体文件查看与替换。
- `WIN-FE-2`
  - `frontend/src/windows/StreamDeliveryWindow.vue` 已支持 `<audio>` / `<video>` 渐进播放、缓冲态提示与错误降级。
  - `frontend/src/i18n/messages/stores.ts`、`frontend/src/pages/Stream.test.ts`、`frontend/src/stores/stream.test.ts` 已同步更新。

### Stage 3.3 - Code Review
#### Checklist
- 需求覆盖：通过
  - 已覆盖本轮确认范围：本地文件输入、接收端严格边接收边播放、桌面采集仍不在本轮范围内。
- 架构合理性：通过
  - 控制面、Go 数据面、前端播放器状态面保持分层；媒体字节不经过 Wails 事件桥。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - sender 按 ACK window 推进，媒体摘要事件节流，播放通过本地 HTTP outlet；已知代价是多 consumer 时会重复读取同一文件，属于本轮接受范围。
- 可读性与一致性：通过
  - source 输入模型在 prefs、bindings、store、窗口之间保持一致，命名沿用 `inputKind/filePath/mediaByDelivery`。
- 可扩展性与配置化：通过
  - `inputKind=file` 与独立 `media.go` 已为后续 capture runner 预留清晰扩展点。
- 稳定性与安全：通过
  - 非法路径、空文件、`text` source 文件输入、非 `chunk` 播放路径、kind/contentType 不匹配与 disconnect/withdraw cleanup 均有显式处理。
- 测试覆盖情况：通过
  - `$env:GOWORK='off'; go test ./... -count=1`：通过
  - `npx vitest run src/stores/stream.test.ts src/pages/Stream.test.ts`：通过
  - `$env:GOWORK='off'; wails generate module`：通过，保留既有 `Not found: time.Time` 噪声
  - `npm run build`：通过，保留既有大 chunk warning
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子Agent。

#### Residual Risks
- 当前未做真实 Wails 会话下的端到端人工播放冒烟，仍需后续在真实 delivery 环境中确认浏览器/WebView 对目标媒体格式的原生渐进播放表现。

### Stage 4 - Change Archive
#### Archive Result
- 已新增 `docs/change/2026-04-13_win-stream-media-io.md`。
- 已更新：
  - `docs/change/README.md`
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
  - `docs/lessons/README.md`
  - `docs/lessons/stream-media-progressive-playback-http-outlet.md`
- Requirements impact: `updated`
- Specs impact: `updated`
- Lessons impact: `updated`
- Workflow end confirmation: 待用户明确确认；在确认前不合并、不清理 worktree。

## Iteration 2 - Desktop Capture

### Stage 1 - Requirements Analysis
#### Goal
- 在上一轮“文件输入 + 严格边接收边播放”基础上，为 `Stream` 增加第二种真实输入：桌面采集。
- 保持接收端仍然是边接收边播放，而不是先完整录制再播。
- 继续复用现有 `source` / `consumer` / `delivery` 控制面，不新增独立媒体模块。

#### Scope
- Must:
  - 本地 `video` source 必须支持 `desktop` 类型输入，而不只是 `file`
  - Source Window 必须允许用户通过显式点击启动和停止桌面采集
  - 采集开始后，当前 active delivery 必须收到持续到达的媒体 chunk，接收端继续走可播放输出
  - 采集 API 不可用、用户拒绝授权、无 active delivery、录制器初始化失败、采集过程中 track 中止都必须显式报错或停机
  - 现有 `text` source、文件型 `music/video` source、接收端播放窗口不得回退
- Optional:
  - Source Window 显示本地预览
  - 展示采集状态、已绑定 delivery 数和错误信息
- Not in scope:
  - 本轮不做系统音频采集
  - 本轮不做摄像头、麦克风采集
  - 本轮不持久化用户上一次选择的具体屏幕/窗口
  - 本轮不做原生 Go / OS 级桌面采集服务
  - 本轮不保证“采集中途新加入的 delivery 无缝加入当前录制会话”

#### Use Cases
1. 用户创建本地 `video` source，输入模式选择 `desktop capture`，在 Source Window 点击开始采集并选择一个桌面/窗口后，当前 active delivery 开始接收并在接收端播放。
2. 用户停止采集或系统/浏览器中止共享后，source 端停止发送，接收端播放会自然结束或关闭。
3. 用户在不支持 `getDisplayMedia()` 的环境中打开桌面采集 source，会收到明确错误，而不是静默失败。

#### Functional Requirements
1. `Stream` source 必须允许除 `file` 外再声明一种 `desktop` 输入模式。
2. `desktop` 输入模式仅允许 `video` source 使用。
3. Source Window 必须通过用户手势调用桌面采集 API，不能自动绕过权限弹窗。
4. 桌面采集开始后，系统必须将编码后的媒体 chunk 送入现有 `stream` delivery，而不是先落成本地完整文件。
5. 接收端对桌面采集 delivery 必须继续满足“边接收边播放”。
6. 若当前没有 active delivery，系统必须明确提示，而不是假装采集已成功推流。
7. 采集停止时必须发送终止信号或 EOF，使接收端 runtime 可以收口。

#### Non-functional Requirements
- 架构边界:
  - 桌面采集的数据面允许从前端进入 Go，但不得通过 `stream.media` / `stream.stats` 这类事件桥回传原始字节
- 兼容性:
  - 必须显式检测 `navigator.mediaDevices.getDisplayMedia` 和 `MediaRecorder` 支持情况
- 稳定性:
  - 采集 track ended、窗口关闭、source 切换或 delivery 全部断开时必须正确停止录制
- 可维护性:
  - `file` 与 `desktop` 输入差异应收敛在明确的 input kind 和 source-window runner，而不是散在多处条件分支

#### Inputs / Outputs
- Inputs:
  - 本地 `video` source
  - 浏览器/系统桌面采集授权
  - 当前 source 对应的 active deliveries
- Outputs:
  - 采集中的本地预览和状态
  - 持续推送到 active deliveries 的媒体 chunk
  - 接收端可播放输出

#### Edge Cases
- 浏览器 / WebView 不支持 `getDisplayMedia()` 或 `MediaRecorder`
- 用户授权后立即取消共享或共享中途被系统终止
- 采集启动时没有 active delivery
- 桌面采集 source 在录制中被 withdraw
- 新 delivery 在录制中途加入，缺少当前录制会话的初始化片段

#### Acceptance Criteria
1. 本地 `video` source 可配置为 `desktop` 输入模式。
2. 用户可从 Source Window 启动桌面采集，并看到明确运行状态。
3. 当前 active delivery 能在采集进行中接收并播放桌面内容。
4. 采集停止或异常中止后，source 和 delivery 都能显式收口。
5. 现有文件输入、文本输入和播放基线不回退。

#### Risks
- `getDisplayMedia()` 需要瞬时用户激活，且权限不能持久化；不能按文件输入那样自动恢复。
- `MediaRecorder` 的 `timeslice` 不是精确时钟，chunk 大小和到达间隔会抖动。
- 当前最小方案默认只覆盖“采集开始时已 active 的 deliveries”；中途新增 delivery 若需无缝加入，可能要独立 recorder 或重启采集。

#### Issue List
- 本轮按最小安全范围先做“桌面视频，不带系统音频”。
- 本轮默认把“中途新加入 delivery 需要重启采集才能加入当前会话”视为接受约束。

### Stage 2 - Architecture Design
#### Overall Solution
- 推荐方案：在 `StreamSourceWindow.vue` 中使用浏览器 `getDisplayMedia()` 获取 `MediaStream`，再用 `MediaRecorder` 以固定 timeslice 生成 `video/webm` chunk，通过新的 Go binding 直接发布到当前 active deliveries。
- 选型理由：
  - 当前仓库没有现成原生桌面采集基础，浏览器侧采集是最小改动面
  - consumer 侧刚完成的媒体 runtime、progressive sink 和播放窗口可以继续复用
  - Wails bindings 已支持 `[]byte -> Array<number>`，可以承接桌面采集 chunk 入站
- 备选对比：
  - 原生 Go / OS 级桌面采集
    - 放弃。当前仓库没有基础，跨平台与依赖面过大
  - 先录成本地文件，再复用 file sender
    - 放弃。不满足“边接收边播放”
  - 让前端直接把采集字节走事件桥到接收端
    - 放弃。破坏现有 Go 数据面分层

#### Module Responsibilities
- `internal/services/stream/media.go`
  - 扩展 `input_kind=desktop`
  - 新增桌面采集 chunk 发布入口和 active delivery 发送逻辑
- `internal/services/stream/service.go`
  - 暴露新的桌面采集发布 binding
- `frontend/src/stores/stream.ts`
  - 扩展 source 输入模型，支持 `desktop`
  - 暴露桌面采集 chunk 发布 helper
- `frontend/src/pages/Stream.vue`
  - source dialog 支持在 `video` source 下选择 `desktop capture`
- `frontend/src/windows/StreamSourceWindow.vue`
  - 实现 start/stop capture、本地预览、chunk 上报和异常处理
- `docs/requirements/stream.md`
  - 把 `desktop capture` 纳入长期输入模式
- `docs/specs/stream.md`
  - 记录前端采集 -> Go 数据面的长期契约和约束

#### Data / Call Flow
1. 用户创建或打开本地 `video` source，并将 `inputKind` 设为 `desktop`。
2. Source Window 点击开始采集，调用 `navigator.mediaDevices.getDisplayMedia(...)`。
3. 前端基于共享得到的 `MediaStream` 创建 `MediaRecorder`，以固定 timeslice 输出 blob chunk。
4. 每个 chunk 转为 `Array<number>` 后调用新的 Go binding，把字节发送到“采集开始时的 active deliveries”。
5. Go 侧按现有 stream DATA/ACK 机制把 chunk 发给对应 producer deliveries。
6. consumer 侧继续写入 media sink，通过本地 HTTP outlet 暴露播放 URL，Delivery Window 继续边接收边播放。
7. 用户停止采集或 track ended 时，前端发送 final chunk / EOF 并停止本地采集状态。

#### Interface Drafts
- `SourceInputConfigReq`
  - `input_kind = "file" | "desktop"`
- 新 binding 草案：
  - `PublishCaptureChunkSimple(sourceID uint32, req PublishCaptureChunkReq)`
  - `PublishCaptureChunkReq`
    - `source_id`
    - `delivery_ids`
    - `pts_ms`
    - `final`
    - `payload []byte`

#### Error Handling and Safety
- `getDisplayMedia` 不可用时立即报错
- 用户拒绝授权时不落入“已开始采集”状态
- `desktop` 输入若用于非 `video` source，后端和前端都必须拒绝
- capture track ended 时必须自动 stop 并尝试发送 final
- 如果 active delivery 为空，开始采集前就拒绝

#### Performance and Testing Strategy
- 使用 `MediaRecorder.start(timeslice)` 控制 chunk 粒度，避免逐帧上报
- 桌面采集原始字节只走前端 -> Go binding，不走 Wails 事件广播
- 测试重点：
  - `desktop` input kind 持久化与校验
  - binary chunk 发布到 active deliveries
  - stop / final / EOF 路径
  - Source Window 的支持检测与状态切换

#### Extensibility Design Points
- 未来若要支持系统音频，可在 `desktop` input config 上增加 `captureAudio`
- 未来若要支持“中途新加入 delivery 无缝加入”，可演进为 per-delivery recorder 或 capture restart 策略
- 未来若要支持原生桌面采集，可保留同一 `inputKind=desktop`，只替换 source runner 实现

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 worktree 已完成文件输入与接收端严格边接收边播放。
- 新一轮目标是在不破坏上一轮基线的前提下，为 `video` source 增加桌面采集输入。

#### Docs Governance Routing Decision
- 稳定真相文档：
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
- 执行控制文档：
  - 当前 worktree 根 `plan.md`
- 完成归档：
  - `docs/change/2026-04-13_win-stream-desktop-capture.md`
- Lessons:
  - `docs/lessons/stream-desktop-capture-session-reset.md`
- Requirements impact: `updated`
- Specs impact: `updated`
- Related requirements:
  - `docs/requirements/stream.md`
- Related specs:
  - `docs/specs/stream.md`
- Related lessons:
  - `docs/lessons/stream-media-progressive-playback-http-outlet.md`
  - `docs/lessons/stream-desktop-capture-session-reset.md`
  - `docs/lessons/wails-binding-proto-drift.md`

#### Executable Task List
- `WIN-DC-DOC-1` 更新 Stream requirements/specs，纳入桌面采集输入
- `WIN-DC-BE-1` 扩展 Go stream service，支持 `desktop` input kind 和 capture chunk 发布
- `WIN-DC-FE-1` 扩展 Stream source dialog / source window / store，支持桌面采集
- `WIN-DC-VAL-1` 补测试与验证

#### Task Details
##### WIN-DC-DOC-1 - Update Stream Requirements And Specs For Desktop Capture
- Owner: 主代理
- Worktree: `D:\project\MyFlowHub3\worktrees\win-stream-media-io`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-stream-media-io\plan.md`
- Goal: 把桌面采集输入的长期边界写入 requirements/specs
- Files / Modules:
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
  - 必要时 `docs/lessons/*`
- Acceptance:
  - 文档明确桌面采集范围、约束与降级语义
- Test Points:
  - 文档链路无冲突
- Rollback:
  - 回退本轮 docs 改动

##### WIN-DC-BE-1 - Add Desktop Capture Chunk Publishing
- Owner: 主代理
- Worktree: `D:\project\MyFlowHub3\worktrees\win-stream-media-io`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-stream-media-io\plan.md`
- Goal: 让前端桌面采集 chunk 可以进入 Go stream 数据面并发布到 active deliveries
- Files / Modules:
  - `internal/services/stream/media.go`
  - `internal/services/stream/service.go`
  - `internal/services/stream/service_test.go`
  - 必要时 `internal/services/stream/publish.go`
- Acceptance:
  - `desktop` input kind 合法
  - capture chunk 能发布到指定 active deliveries
  - stop/final 可收口
- Test Points:
  - input kind 校验
  - binary chunk publish
  - EOF / final
- Rollback:
  - 回退 stream backend 相关变更

##### WIN-DC-FE-1 - Add Desktop Capture Source UI
- Owner: 主代理
- Worktree: `D:\project\MyFlowHub3\worktrees\win-stream-media-io`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-stream-media-io\plan.md`
- Goal: 让用户可以创建桌面采集 source 并在 Source Window 启动/停止采集
- Files / Modules:
  - `frontend/src/stores/stream.ts`
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/windows/StreamSourceWindow.vue`
  - `frontend/src/i18n/messages/stores.ts`
  - 必要时 `frontend/src/pages/Stream.test.ts`
  - 必要时 `frontend/src/stores/stream.test.ts`
- Acceptance:
  - `video` source 可选择 `desktop capture`
  - Source Window 可 start/stop capture，并处理不支持/拒绝授权
- Test Points:
  - input kind 切换
  - Source Window 状态切换
- Rollback:
  - 回退 Stream 前端桌面采集相关变更

##### WIN-DC-VAL-1 - Validation
- Owner: 主代理
- Worktree: `D:\project\MyFlowHub3\worktrees\win-stream-media-io`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-stream-media-io\plan.md`
- Goal: 验证 Go、前端、Wails 和关键桌面采集回归
- Files / Modules:
  - `go test ./...`
  - `npx vitest run ...`
  - `wails generate module`
  - `npm run build`
- Acceptance:
  - 验证结果可复述
- Rollback:
  - 无

#### Dependencies
- 桌面采集依赖浏览器 / WebView 对 `getDisplayMedia()` 与 `MediaRecorder` 的支持
- 桌面采集必须由 Source Window 内的用户点击触发
- 依赖上一轮已完成的 consumer media runtime 与播放窗口

#### Risks and Notes
- `getDisplayMedia()` 权限不可持久化，无法像文件输入那样自动恢复
- 当前最小实现默认不支持“录制中途新增 delivery 无缝加入”
- 若 `MediaRecorder` 输出格式与浏览器渐进播放能力不匹配，仍可能降级到错误提示

#### Parallelism Assessment
- 本轮仍不派发子Agent，原因：
  - 桌面采集方案需要同时改 source window、store 与 backend publish 路径，耦合度高
  - 当前平台未获显式子Agent授权

#### Issue List
- 无

阻塞：否
进入 3.2
禁止派发子Agent

### Stage 3.2 - Implementation
#### Task Execution Summary
- `WIN-DC-DOC-1`
  - 已更新 `docs/requirements/stream.md` 与 `docs/specs/stream.md`，把桌面采集输入纳入长期真相。
  - 已新增 `docs/lessons/stream-desktop-capture-session-reset.md` 并更新 `docs/lessons/README.md`。
- `WIN-DC-BE-1`
  - `app_stream.go` 已支持 `inputKind=desktop` 持久化归一化。
  - `internal/services/stream/media.go` 已支持 `desktop` 输入配置、`streamDataFlagSessionStart` 与 consumer session reset。
  - `internal/services/stream/publish.go` 已新增 `PublishCaptureChunk(...)` / `PublishCaptureChunkSimple(...)`。
  - `internal/services/stream/service_test.go` 已补 `desktop` input、capture publish、session reset 回归。
- `WIN-DC-FE-1`
  - `frontend/src/stores/stream.ts` 已新增 `publishCaptureChunk(...)`，并允许 `desktop` video source 无文件路径 announce。
  - `frontend/src/pages/Stream.vue` 已新增 `Local File` / `Desktop Capture` 输入模式切换。
  - `frontend/src/windows/StreamSourceWindow.vue` 已支持桌面采集 start/stop、预览、固定 delivery 集合和 final 收口。
  - `frontend/src/windows/StreamSourceWindow.test.ts` 已覆盖首个 chunk 带 `sessionStart`、停止发送 `final`。
- `WIN-DC-VAL-1`
  - 已执行 Go / Vitest / Wails / build 验证，结果见下方 review 与 archive。

### Stage 3.3 - Code Review
#### Checklist
- 需求覆盖：通过
  - 已覆盖本轮确认范围：桌面视频采集、严格边接收边播放、无系统音频、固定 capture start 时的 delivery 集合。
- 架构合理性：通过
  - 采集只走前端 -> Go binding -> 现有 stream DATA 数据面，consumer 继续复用现有 media runtime 与本地 HTTP outlet。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 采集通过 `MediaRecorder` 分块推送，媒体字节不经过事件桥；当前接受的代价是 late-join 不做额外补片段。
- 可读性与一致性：通过
  - `desktop` 输入模型在 prefs、backend、store、source dialog、source window 之间命名一致，`session_start` 语义在前后端保持一致。
- 可扩展性与配置化：通过
  - `inputKind=file | desktop` 与独立 `PublishCaptureChunkSimple(...)` 已为后续系统音频或原生采集替换保留清晰边界。
- 稳定性与安全：通过
  - 不支持运行时、无 active delivery、非 `video` source、非 `chunk` delivery、系统终止共享等路径都有显式拒绝或停机。
- 测试覆盖情况：通过
  - `$env:GOWORK='off'; go test ./... -count=1`：通过
  - `npx vitest run src/stores/stream.test.ts src/pages/Stream.test.ts src/windows/StreamSourceWindow.test.ts`：通过
  - `$env:GOWORK='off'; wails generate module`：通过，保留既有 `Not found: time.Time` 噪声
  - `npm run build`：通过
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子Agent，符合本计划显式限制。

#### Residual Risks
- 当前尚未在真实 Wails 会话下做端到端桌面共享播放冒烟。
- 采集中途新增 delivery 仍需重启 capture 才能纳入当前会话。

### Stage 4 - Change Archive
#### Archive Result
- 已新增：
  - `docs/change/2026-04-13_win-stream-desktop-capture.md`
  - `docs/lessons/stream-desktop-capture-session-reset.md`
- 已更新：
  - `docs/change/README.md`
  - `docs/lessons/README.md`
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
  - `plan.md`
- Requirements impact: `updated`
- Specs impact: `updated`
- Lessons impact: `updated`
- Workflow end confirmation: 待用户明确确认；在确认前不合并、不清理 worktree。
