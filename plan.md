# Plan - Win Stream Product Tabs

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/win-stream-product-tabs`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-product-tabs`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `frontend-design` 的 `SKILL.md`
- repo / branch / worktree confirmation
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `feat/win-stream-product-tabs`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-product-tabs`
  - implementation will stay inside this worktree only
- participating modules
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/layout/AppShell.vue`
  - `frontend/src/stores/stream.ts`
  - `frontend/src/i18n/messages/*`
  - `app_stream.go`
  - `internal/services/stream/*.go`
  - `docs/change/*`

### Stage 1 - Requirements Analysis
#### 目标
- 把 Win 的 `Stream` 页面从当前“控制面 + 观察面混排”改造成更接近产品的多 tab 界面，使用户能以更少的主页面表单完成：
  - 本地 source 管理
  - 本地 consumer 管理
  - 远端控制与运行态查看
- 同时补齐本地持久化、最小 source 输入能力和新增 UI 的 i18n 覆盖。

#### 范围
- 必须
  - `Stream` 顶部改为类似 `VarPool` 的 tab 切换
    - `Source`
    - `Consumer`
    - `Control`
  - `Source` tab 以本地 source 列表为主
    - 新增 source
    - 移除 source
    - 打开独立 source 输入界面
    - 列表只展示关键信息与当前绑定关系摘要
  - `Consumer` tab 以本地 consumer 列表为主
    - 新增 consumer
    - 移除 consumer
    - 打开独立订阅弹窗
    - 列表只展示当前绑定了谁的 source
  - `Control` tab 保留现有 control-plane 能力，但减少页面拥挤度
  - 本地 source / consumer 列表必须做 profile 级持久化
  - 登录并恢复会话后，已保存的本地 source / consumer 必须自动恢复
  - 至少为 `text` source 补一个可用的最小发送能力，满足“向 source 输入东西”
  - 本轮新增的 Stream 页面 / store / App prefs 文案必须走 i18n
  - 增加或更新关键前端 / Go 测试
- 可选
  - source 输入界面显示最近发送记录或当前 delivery 摘要
  - 在 `Control` tab 中保留更精细的 signal 操作入口
- 不做
  - 不修改 `stream` 协议 wire
  - 不新增摄像头 / 麦克风 / 屏幕采集
  - 不实现音乐 / 视频的真实采集或播放器
  - 不把主页面改成大块常驻表单编辑器

#### 使用场景
- 用户进入 `Source` tab，看到自己保存的本地 source 列表，点击按钮弹窗新增一个 source。
- 用户在某个 text source 上点击“输入”，进入独立弹窗并发送文本到当前 active deliveries。
- 用户进入 `Consumer` tab，看到自己保存的本地 consumer 列表，以及每个 consumer 当前订阅了谁。
- 用户在某个 consumer 上点击“订阅”，弹窗查询当前节点或其它节点的 source，并选择一个发起订阅。
- 用户进入 `Control` tab，继续做远端 source / consumer 查询、connect / disconnect / signal 等控制面动作。
- 用户重启 Win 或切换 profile 后，已保存的本地 source / consumer 会在身份就绪后自动恢复。

#### 功能需求
- Stream store 必须区分：
  - 本地持久化的 source / consumer 列表
  - 远端查询得到的 source / consumer catalog
  - runtime deliveries / text / stats
- source / consumer 的新增与移除必须同步：
  - 调用后端 binding
  - 更新本地 store
  - 更新 profile 持久化
- source 输入弹窗必须与主列表分离
- consumer 订阅弹窗必须与主列表分离
- consumer 主列表必须能显示当前绑定关系摘要
- source 主列表必须能显示当前连接的 consumer 摘要
- 若 source 为 `text`，输入弹窗必须能把文本发送到该 source 的 active local producer deliveries
- 自动恢复必须避免重复创建冲突，并能在同一描述符场景下安全幂等

#### 非功能需求
- 继续沿用现有 Win 设计语言，尤其是 `VarPool` 的 tab 形态
- 改动面保持最小，不重写已有 stream runtime 事件契约
- 持久化必须使用现有 profile-backed `App` storage，而不是浏览器 `localStorage`
- 表单校验失败、恢复失败、无 active delivery 等异常必须显式报错，不得静默吞掉
- 页面主画布保持简洁，不在首页堆叠大量常驻输入框

#### 输入输出
- 输入
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
  - 现有 `Stream.vue` / `stream.ts`
  - `VarPool.vue` / `varpool.ts`
  - `app_varpool.go` / `app_topicbus.go`
- 输出
  - 更新后的 Stream 页面与 store
  - 新增 `App` Stream prefs 持久化接口
  - 如有必要，新增最小 text publish binding
  - 更新后的前端 / Go 测试
  - 本轮 `docs/change` 归档

#### 边界异常
- 持久化数据损坏、缺字段或重复 ID
- source / consumer 自动恢复时本地 descriptor 与已有目录冲突
- session 已连接但身份未就绪
- text source 输入时没有 active delivery
- 对非 `text` source 打开输入界面
- 订阅弹窗查询远端 source 为空或 kind 不匹配
- profile 切换后旧列表残留在当前界面

#### 验收标准
- Stream 顶部存在 `Source / Consumer / Control` 三个 tab，视觉与 `VarPool` 一致
- `Source` tab 主界面不再堆叠大块创建表单；新增和输入都通过独立弹窗完成
- `Consumer` tab 主界面不再堆叠大块订阅表单；订阅通过独立弹窗完成
- 本地新增的 source / consumer 在重开应用并登录后可以自动恢复
- consumer 列表能直接看出当前绑定 source 摘要
- text source 输入界面能够把文本发送到 active local producer deliveries
- 新增 UI 文案在 `zh-CN` 下正确翻译，并保持 `en` 回退可用
- 相关测试通过

#### 风险
- 自动恢复如果触发时机过早，可能在身份未 ready 时失败
- store 同时承载本地列表、远端 catalog 和 runtime 事件，若边界没拆清会导致 UI 状态串扰
- text source 发送若未正确维护 position，可能导致对端 ACK / 位置语义异常
- 页面与 `AppShell` 同时触发恢复时，若没有幂等保护可能产生重复 announce

#### 问题清单
- none

### Stage 2 - Architecture Design
#### 总体方案（含选型理由 / 备选对比）
- 方案 A（采用）
  - 以现有 `stream` runtime 为基础，补一层“产品化 store + profile 持久化 + tabbed page”
  - 新增 `App.StreamPrefs / SaveStreamPrefs`
  - `AppShell` 负责 profile load 与登录后的自动恢复
  - `Stream.vue` 负责 tab、列表、弹窗和轻量交互
  - 额外补一个最小 `text` source publish binding，满足 source 输入
  - 理由
    - 复用现有 Win 页面与 profile 体系
    - 不引入前端 localStorage
    - 能满足“本地持久化 + 自动恢复 + 单独输入界面”的产品诉求
- 方案 B（不采用）
  - 只在 Stream 页面打开后恢复本地 source / consumer
  - 不采用原因：用户希望更接近产品，source / consumer 应在登录后自动可用，而不是依赖打开页面
- 方案 C（不采用）
  - 用浏览器 localStorage 保存 source / consumer 列表
  - 不采用原因：不符合现有 profile-backed 持久化模式，也会绕开多 profile 语义

#### 模块职责
- `app_stream.go`
  - 定义 `StreamPrefs`
  - 提供 `StreamPrefs()` / `SaveStreamPrefs(...)`
  - 负责 prefs 归一化、去重和 profile 存取
- `app_stream_test.go`
  - 覆盖 prefs 读写、归一化、重复 ID 合并
- `frontend/src/stores/stream.ts`
  - 维护本地 source / consumer、远端 catalog、runtime deliveries 三类状态边界
  - 加载 / 保存 prefs
  - 自动恢复本地 source / consumer
  - 提供 text source 最小发送能力
- `frontend/src/layout/AppShell.vue`
  - profile 切换时加载 Stream prefs
  - 登录 / 重连成功后触发 restore
- `frontend/src/pages/Stream.vue`
  - 提供 tab UI
  - source / consumer 列表与摘要
  - 新增弹窗、订阅弹窗、source 输入弹窗
  - 保留 control tab
- `internal/services/stream/service.go`
  - 暴露最小 text publish binding
- `internal/services/stream/runtime.go`
  - 复用现有 DATA/ACK 帧格式 helper
  - 在 producer send 路径维护必要的本地位置状态
- `frontend/src/i18n/messages/stores.ts`
  - 补齐新增 Stream 文案中文翻译

#### 数据 / 调用流
1. `AppShell` 在 profile 变化时调用 `stream.loadPrefs()`
2. Stream store 载入：
  - `activeTab`
  - `targetId`
  - 保存的本地 sources
  - 保存的本地 consumers
3. 当 session connected + loggedIn + node/hub ready 时，`AppShell` 调 `stream.restoreLocalCatalogs()`
4. `restoreLocalCatalogs()` 对保存的 source / consumer 逐个调用现有 `AnnounceSimple / AnnounceConsumerSimple`
5. Stream 页面：
  - `Source` tab 消费 store 中的本地 source 列表
  - `Consumer` tab 消费 store 中的本地 consumer 列表
  - `Control` tab 使用远端 catalog 查询状态
6. 用户在 source 输入弹窗提交文本时：
  - 前端调用新 binding
  - Go 根据本地 `producerDeliveries` 找到该 source 的 active deliveries
  - 按现有 `KindData` 帧格式发往各个 consumer
7. runtime 的 `stream.delivery / stream.text / stream.stats` 事件继续驱动控制台与绑定摘要刷新

#### 接口草案
- App prefs
  - `StreamPrefs() -> { activeTab, targetId, sources, consumers }`
  - `SaveStreamPrefs(prefs) -> normalized prefs`
- Stream store
  - `loadPrefs()`
  - `savePrefs()`
  - `restoreLocalCatalogs(options?)`
  - `publishText(sourceId, text)`
- Stream service
  - `PublishTextSimple(sourceID uint32, req PublishTextReq) -> PublishTextResp`
- 页面状态
  - `activeTab`
  - `sourceDialogOpen`
  - `consumerDialogOpen`
  - `sourceStudioOpen`
  - `subscribeDialogOpen`

#### 错误与安全
- prefs 载入失败时必须显式报错，不使用损坏数据继续恢复
- `publishText` 只允许：
  - 已登录本机 node
  - 已存在的本地 source
  - `kind=text`
  - 至少一个 active local producer delivery
- 非 `text` source 输入界面只展示说明，不伪装成可发送
- 自动恢复必须以相同 descriptor 幂等，不因重复恢复产生冲突
- 页面不直接拼原始 DATA payload；统一通过 Go binding 发送

#### 性能与测试策略
- 性能
  - prefs 为 profile 小对象 JSON，不新增高频 I/O
  - 自动恢复只在 profile / 身份变更后触发，并带幂等保护
  - 复用现有 runtime 事件，不新增重复监听
- 测试
  - `go test ./... -count=1 -p 1`
  - `npm test -- Stream`
  - 重点覆盖
    - prefs 归一化与持久化
    - store 恢复与保存
    - text source publish
    - Stream 页面 tab / 弹窗主交互

#### 可扩展性设计点
- 本地持久化结构保持 source / consumer 分离，后续可扩展更多本地草稿或 source studio 配置
- source 输入 binding 独立成最小 `text` publisher，后续可扩成其它 kind 的 producer editor
- 页面把“本地列表”和“远端控制 catalog”拆开，后续增加更强的浏览器或 viewer 时不需要重写主列表

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
  - `docs/change/2026-03-31_win-stream-product-tabs.md`
- reusable troubleshooting knowledge
  - 目前无新增 lesson 需求，先沿用已有 `stream` lessons 作为背景参考
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements
  - `D:\project\MyFlowHub3\worktrees\feat-win-stream-product-tabs\docs\requirements\stream.md`
- Related specs
  - `D:\project\MyFlowHub3\worktrees\feat-win-stream-product-tabs\docs\specs\stream.md`
- Related lessons
  - `D:\project\MyFlowHub3\worktrees\feat-win-stream-product-tabs\docs\lessons\stream-ctrl-await-mismatch.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-stream-product-tabs\docs\lessons\stream-local-owner-ctrl-gap.md`

#### Executable Checklist
- [x] `STRUX-1` 增加 Stream profile-backed prefs 与测试
- [x] `STRUX-2` 重构 Stream store，拆分本地列表 / 远端 catalog / 自动恢复逻辑
- [x] `STRUX-3` 实现 text source 最小 publish binding 与测试
- [x] `STRUX-4` 重构 Stream 页面为 `Source / Consumer / Control` tabs，并补齐弹窗交互和 i18n
- [x] `STRUX-5` 在 `AppShell` 接入 Stream prefs 加载与自动恢复
- [x] `STRUX-6` 完成 3.3 review checklist 与 4 阶段归档

#### Task Details
##### `STRUX-1` - Stream prefs
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-product-tabs`
- Goal
  - 为 Stream 的本地 source / consumer 列表、tab 和 target 提供 profile-backed 持久化
- Files
  - `app_stream.go`
  - `app_stream_test.go`
- Acceptance
  - 可读取 / 保存 / 去重 Stream prefs
- Tests
  - `go test ./... -count=1 -p 1`
- Rollback
  - 回退 `app_stream.go`
  - 回退 `app_stream_test.go`

##### `STRUX-2` - Stream store
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-product-tabs`
- Goal
  - 把本地列表、远端 catalog、恢复逻辑和 runtime 状态拆清楚
- Files
  - `frontend/src/stores/stream.ts`
  - `frontend/src/stores/stream.test.ts`
- Acceptance
  - 本地 source / consumer 可保存、恢复，并与远端 catalog 查询互不串扰
- Tests
  - `npm test -- Stream`
- Rollback
  - 回退 `frontend/src/stores/stream.ts`
  - 回退 `frontend/src/stores/stream.test.ts`

##### `STRUX-3` - Text source publish
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-product-tabs`
- Goal
  - 为 text source 增加最小可用的发送文本能力
- Files
  - `internal/services/stream/service.go`
  - `internal/services/stream/runtime.go`
  - `internal/services/stream/service_test.go`
- Acceptance
  - 对 active local producer deliveries 可发送 text DATA
  - 无 active delivery 或非 text source 时返回显式错误
- Tests
  - `go test ./... -count=1 -p 1`
- Rollback
  - 回退 `internal/services/stream/service.go`
  - 回退 `internal/services/stream/runtime.go`
  - 回退 `internal/services/stream/service_test.go`

##### `STRUX-4` - Stream page tabs and dialogs
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-product-tabs`
- Goal
  - 用 tab + 列表 + 弹窗重做 Stream 页面主交互
- Files
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/pages/Stream.test.ts`
  - `frontend/src/i18n/messages/stores.ts`
- Acceptance
  - 页面主画布简洁，source / consumer / control 分离明确
  - source 输入与 consumer 订阅通过独立弹窗完成
- Tests
  - `npm test -- Stream`
- Rollback
  - 回退 `frontend/src/pages/Stream.vue`
  - 回退 `frontend/src/pages/Stream.test.ts`
  - 回退 `frontend/src/i18n/messages/stores.ts`

##### `STRUX-5` - AppShell restore integration
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-product-tabs`
- Goal
  - 在 profile / session 生命周期内自动加载 Stream prefs 并恢复本地 catalogs
- Files
  - `frontend/src/layout/AppShell.vue`
- Acceptance
  - 登录后无需打开 Stream 页面，也会恢复本地 source / consumer
- Tests
  - `npm test -- Stream`
- Rollback
  - 回退 `frontend/src/layout/AppShell.vue`

##### `STRUX-6` - Review and archive
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-product-tabs`
- Goal
  - 按 checklist 复核并归档本轮变更
- Files
  - `plan.md`
  - `docs/change/2026-03-31_win-stream-product-tabs.md`
  - `docs/change/README.md`
- Acceptance
  - 3.3 每项给出结论
  - change 归档完整
- Tests
  - review checklist
- Rollback
  - 回退本轮 docs 变更

#### Dependencies
- `internal/services/stream`
  - 现有 local owner / runtime 作为 source 恢复和 text publish 的基础
- `app_varpool.go`
  - 提供 profile-backed prefs 实现参考
- `frontend/src/pages/VarPool.vue`
  - 提供 tab 风格参考

#### Risks and Notes
- `plan.md` 原先残留的是上一轮 `fix/win-stream-local-owner` 内容，本轮已重置为当前 workflow
- 本轮优先实现最小 text source publisher，不扩展到 `music/video/custom`
- 若实现中发现 text publish 需要修改稳定协议或外部 proto，则必须回到 `3.1`

#### Parallelism Assessment
- 不派发子Agent
- 原因
  - 当前会话没有用户显式授权子Agent
  - 变更集中在同一组 Stream 前后端文件，串行实现更安全

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Task Mapping
- `STRUX-1`
  - `app_stream.go`
  - `app_stream_test.go`
- `STRUX-2`
  - `frontend/src/stores/stream.ts`
  - `frontend/src/stores/stream.test.ts`
- `STRUX-3`
  - `internal/services/stream/publish.go`
  - `internal/services/stream/service_test.go`
- `STRUX-4`
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/pages/Stream.test.ts`
  - `frontend/src/i18n/messages/stores.ts`
- `STRUX-5`
  - `frontend/src/layout/AppShell.vue`
- `STRUX-6`
  - `plan.md`
  - `docs/change/2026-03-31_win-stream-product-tabs.md`
  - `docs/change/README.md`

#### File-level Change Summary
- `app_stream.go`
  - 新增 Stream profile-backed prefs 读写接口，统一归一化 `activeTab / targetId / sources / consumers`
- `app_stream_test.go`
  - 补 prefs 默认值、去重和非法值归一化测试
- `frontend/src/stores/stream.ts`
  - 将本地 source / consumer、远端 catalog、runtime deliveries 拆分建模
  - 新增 prefs 载入保存、自动恢复、text source publish、按 scope 查询 helper
- `frontend/src/stores/stream.test.ts`
  - 补 store 恢复、本地/远端边界、text publish 测试
- `internal/services/stream/publish.go`
  - 新增 text source 最小发送 binding，只向 active local producer deliveries 发 DATA
- `internal/services/stream/service_test.go`
  - 补本地 text publish 成功与无 active delivery 失败测试
- `frontend/src/layout/AppShell.vue`
  - 在 profile / session 生命周期内加载 Stream prefs 并自动恢复本地 catalogs
- `frontend/src/pages/Stream.vue`
  - 重做为 `Source / Consumer / Control` tabs
  - `Source` / `Consumer` 改为列表主界面 + 独立弹窗
  - `text` source 增加独立输入工作区
  - consumer 订阅改为独立弹窗
- `frontend/src/pages/Stream.test.ts`
  - 补 tab、创建弹窗、source studio、consumer subscribe 弹窗测试
- `frontend/src/i18n/messages/stores.ts`
  - 补齐新增 Stream 文案的 `zh-CN` 翻译

#### Design Notes
- 持久化复用现有 profile-backed `App` storage，不引入浏览器 `localStorage`
- 自动恢复放在 `AppShell`，确保登录后无需打开 `Stream` 页面也能恢复本地 source / consumer
- `text` source 的输入能力只补最小可用 binding，不改 `stream` 协议 wire，不伪造其它 kind 的 producer editor
- 页面沿用现有 Win / `VarPool` 视觉模式，主界面保持列表优先，把新增、订阅、输入收敛到弹窗或独立工作区

#### Validation
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./internal/services/stream -count=1`
  - 结果：通过
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test . -count=1`
  - 结果：通过
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：有 `Not found: time.Time` 提示，但退出码为 `0`，bindings 已刷新
- `MyFlowHub-Win/frontend`
  - `npm ci`
  - 结果：通过
- `MyFlowHub-Win/frontend`
  - `npm exec vitest run src/stores/stream.test.ts src/pages/Stream.test.ts`
  - 结果：通过（`2` files, `6` tests）
- `MyFlowHub-Win/frontend`
  - `npm run build`
  - 结果：通过
  - 备注：保留既有大 chunk warning，非本轮回归

#### Blockers
- none

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - `Source / Consumer / Control` tabs 已落地
  - source / consumer 主界面已改为列表优先，新增与订阅通过弹窗完成
  - `text` source 有独立输入工作区，并能实际发送文本
  - 本地 source / consumer 支持 profile 级持久化与登录后自动恢复
- 架构合理性：通过
  - prefs、store、AppShell 生命周期和 stream service publish 各自边界清晰
  - 未改稳定协议，也没有把页面逻辑反向塞进 runtime
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - prefs 为小对象按需保存
  - 自动恢复带幂等保护，未新增重复网络监听
  - text publish 只遍历匹配 source 的 active producer deliveries
- 可读性与一致性：通过
  - Stream store 以 `local` / `catalog` / `runtime` 分层命名
  - 页面交互按 tab 和弹窗拆分，和现有 `VarPool` 风格一致
- 可扩展性与配置化：通过
  - 本地 source / consumer 持久化结构分离
  - source studio 与 publish binding 后续可继续扩展到更多 producer editor
- 稳定性与安全：通过
  - 输入、target、metadata、text content、source kind、active delivery 都有显式校验
  - 恢复失败与自动恢复不完整会显式反馈，不静默吞错
- 测试覆盖情况：通过
  - Go：prefs、publish、stream 全量回归通过
  - Frontend：store 与页面交互测试通过，生产 build 通过
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未使用子Agent

### Stage 4 - Change Archive
#### $m-docs Check
- 使用 `$m-docs` 校验 plan/change/lessons 路由
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`
- 相关 requirements
  - `docs/requirements/stream.md`
- 相关 specs
  - `docs/specs/stream.md`
- 相关 lessons
  - `docs/lessons/stream-ctrl-await-mismatch.md`
  - `docs/lessons/stream-local-owner-ctrl-gap.md`
  - `docs/lessons/frontend-build-babel-parser-missing.md`
- 本轮无需更新 `docs/requirements`、`docs/specs`、`docs/lessons` 或 `docs/lessons/README.md`
- 已更新
  - `plan.md`
  - `docs/change/2026-03-31_win-stream-product-tabs.md`
  - `docs/change/README.md`

#### Archive Status
- 已完成 repo-local 归档
- 等待用户确认是否结束当前 workflow
