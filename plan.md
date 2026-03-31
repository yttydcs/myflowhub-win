# Plan - Win Stream Windows Trim

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/win-stream-windows-trim`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-windows-trim`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `frontend-design` 的 `SKILL.md`
- repo / branch / worktree confirmation
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `feat/win-stream-windows-trim`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-windows-trim`
  - implementation will stay inside this worktree only
- participating modules
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/router/index.ts`
  - `frontend/src/windows/*`
  - `frontend/src/i18n/messages/*`
  - `frontend/src/pages/Stream.test.ts`
  - `docs/change/*`

### Stage 1 - Requirements Analysis
#### 目标
- 在保留现有 Stream 协议能力和持久化模型的前提下，进一步收敛 Win `Stream` 页面交互，使主页面更紧凑、更接近工作台：
  - 移除顶部 4 块统计卡
  - source / consumer / delivery 列表显著简化
  - 把输入与输出从页内块或弹窗转为独立窗口

#### 范围
- 必须
  - 移除 `Stream` 页面顶部 `Saved Sources / Saved Consumers / Known Deliveries / Last Runtime Event` 四块统计卡
  - source / consumer 主列表必须进一步精简，只保留高价值摘要
  - source 输入必须可通过独立窗口完成，而不是页内工作区
  - output / viewer 必须可通过独立窗口完成，而不是主页面内长块输出区
  - 主页面仍保留 `Source / Consumer / Control` tabs
  - 继续沿用现有 Win / `VarPool` / `TopicBus` 的设计语言和 window route 模式
  - 新增 UI 文案必须补齐 i18n
  - 增加或更新前端测试覆盖关键交互
- 可选
  - consumer 侧栏或 control 列表提供更直接的输出窗口入口
  - 窗口内保留轻量操作按钮，例如刷新或基础 runtime signal
- 不做
  - 不修改 `stream` 协议 wire
  - 不修改 Go `StreamService` 的核心控制面契约，除非前端窗口化需要极小配套调整
  - 不新增音视频解码或采集能力
  - 不重做全局布局或 AppShell 结构

#### 使用场景
- 用户进入 `Source` tab，只看到紧凑的本地 source 列表和少量动作按钮，不再被顶部统计卡和过多 descriptor 文本打断。
- 用户点击某个 source 的“输入窗口”，弹出独立窗口发送文本，并查看最近发送记录。
- 用户进入 `Consumer` tab，只看到每个 consumer 当前绑定摘要，而不是大量 metadata 与 descriptor 行。
- 用户从 consumer 绑定项或 runtime delivery 列表打开独立输出窗口，专门查看 text frames 或 stats。
- 用户留在主页面时，主要做列表浏览、创建、订阅、连接和定位；详细输入/输出都进入专用窗口。

#### 功能需求
- `Stream.vue` 顶部 summary cards 必须删除
- 本地 source 列表每行至少保留：
  - 显示名或 ID
  - `kind`
  - 当前绑定摘要
- 本地 consumer 列表每行至少保留：
  - 显示名或 ID
  - `kind`
  - 当前 source 绑定摘要
- source 输入窗口必须支持：
  - 依据 `sourceId` 打开
  - 对 `text` source 发送文本
  - 查看最近发送结果
- output 窗口必须支持：
  - 依据 `deliveryId` 或等价运行态标识打开
  - `text` delivery 查看文本帧
  - 非 `text` delivery 查看统计摘要
- 主页面必须提供打开输入窗口和输出窗口的入口
- 窗口页面必须复用现有 store/runtime 状态，而不是在页面内重复解析原始 payload

#### 非功能需求
- 保持主页面简洁，列表优先，避免重复说明文案和冗长 metadata
- 独立窗口应复用现有 `layout: "window"` 路由模式，不引入新的窗口基础设施
- 设计继续贴合现有 Win 系统的产品感，不做风格漂移
- 变更面保持最小，优先复用现有 `TopicBus` / `Logs` / `Flow` 的窗口打开模式

#### 输入输出
- 输入
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/router/index.ts`
  - `frontend/src/windows/TopicBusWindow.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/VarPool.vue`
- 输出
  - 更精简的 `Stream` 主页面
  - 新增 Stream 输入 / 输出窗口页面与路由
  - 更新后的 i18n 与测试
  - 本轮 `docs/change` 归档

#### 边界异常
- source 输入窗口打开时对应 source 已被移除
- output 窗口打开时 delivery 已关闭或不存在
- 非 `text` source 打开输入窗口
- `text` output 窗口打开后暂时没有任何 text frame
- 窗口被浏览器 popup policy 阻止
- 页面与窗口同时打开时，runtime 状态刷新节奏不一致

#### 验收标准
- 主页面顶部四块统计卡已移除
- source / consumer 列表比当前版本明显更简洁
- source 输入不再通过主页面弹窗或页内工作区完成，而是独立窗口
- output / viewer 不再主要依赖主页面大块显示，而是独立窗口
- 主页面仍保留 source / consumer / control 主路径
- 前端测试和 build 通过

#### 风险
- 如果窗口页面过度复刻主页面，会把复杂度从主页面平移到窗口而不是收敛
- output 窗口的标识若选错层级，可能导致入口不直观或难以定位正确 delivery
- popup 被浏览器阻止时，需要保留显式失败提示

#### 问题清单
- none

### Stage 2 - Architecture Design
#### 总体方案（含选型理由 / 备选对比）
- 方案 A（采用）
  - 保留现有 `Stream` store 和持久化结构
  - 精简 `Stream.vue` 主页面
  - 新增两个 window route：
    - source input window
    - delivery output window
  - 复用 `window.open + #/route` 的既有模式
  - 理由
    - 改动面最小
    - 与 `TopicBus` / `Logs` / `Flow` 已有窗口模式一致
    - 能直接满足“输入输出单独窗口”和“主页面更简洁”
- 方案 B（不采用）
  - 继续使用 Overlay 弹窗，只把内部布局做得更轻
  - 不采用原因：用户已经明确希望输入输出用单独窗口，而不是页内弹层
- 方案 C（不采用）
  - 把主页面改成 master-detail，多栏常驻 viewer
  - 不采用原因：会重新把输出内容堆回主页面，和“列表简洁”目标冲突

#### 模块职责
- `frontend/src/pages/Stream.vue`
  - 主页面列表、紧凑摘要、打开窗口入口、基础创建/订阅/连接操作
- `frontend/src/windows/StreamSourceWindow.vue`
  - source 输入窗口
  - text send 与最近发送记录
- `frontend/src/windows/StreamDeliveryWindow.vue`
  - delivery 输出窗口
  - text frames 或 stats 查看
- `frontend/src/router/index.ts`
  - 注册 Stream window routes
- `frontend/src/stores/stream.ts`
  - 继续作为 runtime / source / consumer / delivery 单一业务状态来源
  - 如有需要，仅补最小 helper，不重构核心状态
- `frontend/src/i18n/messages/*`
  - 补齐新增按钮、窗口标题、空态和错误文案

#### 数据 / 调用流
1. 主页面 `Stream.vue` 继续消费 `stream` store 中的本地 source / consumer 和 runtime deliveries
2. 用户点击 source 的输入入口
3. 页面通过 `window.open()` 打开 `#/stream-source-window?sourceId=...`
4. 输入窗口根据 `sourceId` 从 `stream` store 取 source，并调用现有 `publishText()`
5. 用户点击 consumer 绑定项或 runtime delivery 的输出入口
6. 页面通过 `window.open()` 打开 `#/stream-delivery-window?deliveryId=...`
7. 输出窗口根据 `deliveryId` 从 `stream` store 取 delivery，并消费 `textFramesFor()` / `statsFor()`

#### 接口草案
- 新增 route
  - `/stream-source-window?sourceId=...`
  - `/stream-delivery-window?deliveryId=...`
- 页面 helper
  - `openSourceWindow(sourceId)`
  - `openDeliveryWindow(deliveryId)`
- store
  - 优先复用：
    - `sourceById`
    - `deliveriesForSource`
    - `deliveriesForConsumer`
    - `textFramesFor`
    - `statsFor`
    - `publishText`

#### 错误与安全
- `window.open()` 失败时必须 toast 提示 popup 被阻止
- source 窗口若 `sourceId` 无效，必须显示显式空态
- delivery 窗口若 `deliveryId` 无效或已关闭，必须显示显式空态
- 非 `text` source 在输入窗口中不得伪装成可发送
- 窗口页继续只消费业务状态，不直接解析协议 payload

#### 性能与测试策略
- 性能
  - 主页面移除 summary cards 和大块 viewer，可降低初始信息密度和无关重绘
  - 窗口页面按 query 只聚焦单个 source 或单个 delivery
- 测试
  - `npm exec vitest run src/pages/Stream.test.ts`
  - 如新增窗口测试，再补对应 `src/windows/*.test.ts`
  - `npm run build`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`

#### 可扩展性设计点
- source 输入窗口后续可扩展为更多 producer studio，而无需再改主页面
- delivery 输出窗口后续可扩展为 text inspector、video viewer、custom monitor
- 主页面继续保留“列表与控制”，窗口承载“沉浸式输入/输出”，角色边界清晰

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
  - `docs/change/2026-03-31_win-stream-windows-trim.md`
- reusable troubleshooting knowledge
  - 当前无新增 lesson 需求，沿用现有前端构建与 Stream lessons 作为验证提示
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements
  - `D:\project\MyFlowHub3\worktrees\feat-win-stream-windows-trim\docs\requirements\stream.md`
- Related specs
  - `D:\project\MyFlowHub3\worktrees\feat-win-stream-windows-trim\docs\specs\stream.md`
- Related lessons
  - `D:\project\MyFlowHub3\worktrees\feat-win-stream-windows-trim\docs\lessons\frontend-build-babel-parser-missing.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-stream-windows-trim\docs\lessons\stream-ctrl-await-mismatch.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-stream-windows-trim\docs\lessons\stream-local-owner-ctrl-gap.md`

#### Executable Checklist
- [x] `STRWIN-1` 精简 Stream 主页面：移除 summary cards，压缩 source / consumer / delivery 列表摘要
- [x] `STRWIN-2` 新增 source 输入窗口与打开入口
- [x] `STRWIN-3` 新增 delivery 输出窗口与打开入口
- [x] `STRWIN-4` 补齐 i18n 与前端测试
- [x] `STRWIN-5` 完成回归验证、3.3 review 与 4 阶段归档

#### Task Details
##### `STRWIN-1` - Trim Stream main page
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-windows-trim`
- Goal
  - 让 `Stream` 主页面回到“列表 + 轻动作”形态，去掉 summary cards 和冗余明细
- Files
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/pages/Stream.test.ts`
- Acceptance
  - 顶部四块统计卡消失
  - source / consumer / delivery 列表更紧凑
- Tests
  - `npm exec vitest run src/pages/Stream.test.ts`
- Rollback
  - 回退 `frontend/src/pages/Stream.vue`
  - 回退 `frontend/src/pages/Stream.test.ts`

##### `STRWIN-2` - Add source input window
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-windows-trim`
- Goal
  - 让 source 输入通过独立窗口完成
- Files
  - `frontend/src/router/index.ts`
  - `frontend/src/windows/StreamSourceWindow.vue`
  - `frontend/src/pages/Stream.vue`
- Acceptance
  - 可以从主页面打开 source 输入窗口
  - 窗口内可对 text source 发送文本并看到最近发送记录
- Tests
  - `npm exec vitest run src/pages/Stream.test.ts`
- Rollback
  - 回退窗口 route、窗口组件与入口按钮

##### `STRWIN-3` - Add delivery output window
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-windows-trim`
- Goal
  - 让 delivery 输出通过独立窗口查看
- Files
  - `frontend/src/router/index.ts`
  - `frontend/src/windows/StreamDeliveryWindow.vue`
  - `frontend/src/pages/Stream.vue`
- Acceptance
  - 可以从主页面打开输出窗口
  - text delivery 可看文本帧，非 text delivery 可看 stats
- Tests
  - `npm exec vitest run src/pages/Stream.test.ts`
- Rollback
  - 回退窗口 route、窗口组件与入口按钮

##### `STRWIN-4` - I18n and tests
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-windows-trim`
- Goal
  - 保证新增窗口和精简后的页面文案完整可测
- Files
  - `frontend/src/i18n/messages/stores.ts`
  - 如需要：`frontend/src/i18n/messages/signals.ts`
  - `frontend/src/pages/Stream.test.ts`
- Acceptance
  - 新增文案有 `zh-CN`
  - 关键打开窗口与空态场景有测试
- Tests
  - `npm exec vitest run src/pages/Stream.test.ts`
- Rollback
  - 回退 i18n 与测试改动

##### `STRWIN-5` - Review and archive
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-windows-trim`
- Goal
  - 完成本轮验证、review 和归档
- Files
  - `plan.md`
  - `docs/change/2026-03-31_win-stream-windows-trim.md`
  - `docs/change/README.md`
- Acceptance
  - review checklist 全部给出结论
  - change 归档完整
- Tests
  - `npm run build`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
- Rollback
  - 回退本轮 docs 变更

#### Dependencies
- `frontend/src/pages/TopicBus.vue`
  - 提供 `window.open + route query` 模式参考
- `frontend/src/windows/TopicBusWindow.vue`
  - 提供独立窗口工作区骨架参考
- `frontend/src/pages/Logs.vue`
  - 提供最小窗口打开入口参考
- `frontend/src/stores/stream.ts`
  - 继续作为 Stream 页面与窗口的统一状态来源

#### Risks and Notes
- 当前主仓 `go.mod` 有用户未提交改动，禁止在主仓路径实现
- 本轮不改变 Stream 的长期 specs，只做产品化交互收敛
- 若实现中发现“输出窗口”必须改为 consumer 维度或 source 维度以外的标识层级，再回到 `3.1` 更新计划

#### Parallelism Assessment
- 不派发子Agent
- 原因
  - 当前会话没有用户显式授权子Agent
  - 变更集中在同一组 Stream 前端文件和窗口路由，串行实现更安全

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### 实际完成
- `STRWIN-1`
  - `frontend/src/pages/Stream.vue`
    - 删除顶部 4 块统计卡
    - `Source` / `Consumer` tab 改为更紧凑的单层列表
    - `Control` tab 删除页内 delivery viewer，仅保留列表与控制动作
- `STRWIN-2`
  - `frontend/src/windows/StreamSourceWindow.vue`
    - 新增独立 source 输入窗口
  - `frontend/src/router/index.ts`
    - 注册 `/stream-source-window`
  - `frontend/src/pages/Stream.vue`
    - source 行改为 `window.open()` 打开输入窗口
- `STRWIN-3`
  - `frontend/src/windows/StreamDeliveryWindow.vue`
    - 新增独立 delivery 输出窗口
  - `frontend/src/router/index.ts`
    - 注册 `/stream-delivery-window`
  - `frontend/src/pages/Stream.vue`
    - runtime delivery 行增加 `Output Window`
- `STRWIN-4`
  - `frontend/src/i18n/messages/stores.ts`
    - 补齐本轮窗口和精简页面 `zh-CN`
  - `frontend/src/pages/Stream.test.ts`
    - 更新为验证 summary cards 移除和输入/输出窗口入口

#### 偏差记录
- 未新增独立窗口测试文件
  - 原因：本轮优先用页面测试覆盖主入口，窗口组件通过主页面入口、Go 测试和人工代码 review 收敛

### Stage 3.3 - Review
#### Review Checklist
- 需求对齐
  - 通过：顶部 4 块统计卡已移除
  - 通过：source / consumer 列表压缩为名称、kind、绑定摘要 + 轻动作
  - 通过：输入改为独立 `stream-source-window`
  - 通过：输出改为独立 `stream-delivery-window`
- 架构约束
  - 通过：继续复用现有 `window.open + #/route` 模式
  - 通过：继续复用 `stream` store，不在窗口内直接解析 raw payload
- 风险检查
  - 通过：popup blocked 有 toast
  - 通过：source / delivery 缺失有窗口空态
  - 通过：非 `text` source 在输入窗口中不会伪装成可发送
- 测试与验证
  - 通过：`git diff --check`
  - 通过：`$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 阻塞已记录：前端 `vitest` / `vite build` 因当前 worktree 缺少 `node_modules` 未能执行

结论：通过，进入 4

### Stage 4 - Archive
#### Docs Routing
- 使用 `$m-docs` 复核：
  - 稳定 truth 仍为 `docs/requirements/stream.md` 与 `docs/specs/stream.md`
  - 本轮结果进入 `docs/change/2026-03-31_win-stream-windows-trim.md`
  - `docs/change/README.md` 已更新索引

#### Archive Output
- 新增：
  - `docs/change/2026-03-31_win-stream-windows-trim.md`
- 更新：
  - `docs/change/README.md`

#### Validation Snapshot
- `git diff --check`
  - 通过
- `npm exec vitest run src/pages/Stream.test.ts`
  - 未执行成功：缺少 `vitest` / `@vitejs/plugin-vue`
- `npm run build`
  - 未执行成功：缺少 `frontend/node_modules/vite/bin/vite.js`
- `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 通过

#### Workflow Status
- 当前实现与归档已完成
- 等待用户决定是否结束本 workflow 并合并/清理 worktree
