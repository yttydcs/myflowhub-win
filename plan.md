# Plan - MyFlowHub-Win Detached Window Session Snapshot Sync

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-showcase-window-session-sync`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-showcase-window-session-sync`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`:
  - workspace root `D:\project\MyFlowHub3\guide.md` 已阅读
  - 遵守 `AGENTS.md` 与 `$m-autoflow` 约束：实现只在 `worktrees/` 中进行，计划确认前不进入编码
- repo / base / worktree confirmation:
  - 控制面主仓：`D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - 主仓当前分支：`main`
  - 本轮实现 worktree：`D:\project\MyFlowHub3\worktrees\fix-win-showcase-window-session-sync`
  - 本轮只修改 `repo/MyFlowHub-Win` 前端
- current baseline:
  - 新 worktree 继承了一份与本任务无关的旧 `plan.md`
  - 已在本 worktree 内用本次会话补水修复计划替换，避免后续执行误跟旧任务

### Stage 1 - Requirements Analysis
#### Goal
- 修复 `Showcase` 编辑窗口和预览窗口在 hub 已连接时仍显示 `Disconnected` 的错误状态。
- 消除 detached window 新前端上下文里“身份已恢复，但 session 连接快照未恢复”的 disconnect 假象。

#### Scope
- Must:
  - 修复 `Showcase` 编辑页窗口和预览窗口的连接状态初始化
  - 保证这些窗口的顶部连接状态 badge 与实际 session 状态一致
  - 保证这些窗口里依赖 `sessionStore.connected` 的交互禁用逻辑与真实连接状态一致
  - 复用共享会话补水逻辑，避免在多个 detached window 中复制同一段 `IsConnected` / `LastAddr` 代码
  - 同步修复已确认存在同根因的 `TopicBusWindow`
- Optional:
  - 让 `Home` 页复用同一套共享 helper，消除重复实现
  - 补 detached window 代表性测试，锁定 badge / 交互 gating 的初始化行为
- Not in scope:
  - 不修改 hub 连接协议、后端 session 服务、Wails runtime event 协议
  - 不重做 `Home` 页连接 / 登录流程
  - 不改 packaged runtime 的辅助窗口打开策略
  - 不扩展到所有 detached window，除非验证发现它们也直接依赖 `sessionStore.connected`

#### Use Cases
- 用户已经在主界面连上 hub，然后从 `Showcase` 打开编辑窗口
- 用户已经在主界面连上 hub，然后从 `Showcase` 打开预览窗口
- 用户已经在主界面连上 hub，然后从 `TopicBus` 打开独立窗口
- 在这些窗口中，顶部 badge、按钮禁用态和 ready-check 应与真实 session 状态一致，而不是默认回落到 `Disconnected`

#### Functional Requirements
1. detached window 启动时必须显式恢复当前 session 的连接快照，而不只恢复 `nodeId` / `hubId`
2. `Showcase` 编辑窗口必须在初始化完成后正确显示当前 `Connected` / `Disconnected` 状态
3. `Showcase` 预览窗口必须在初始化完成后正确显示当前 `Connected` / `Disconnected` 状态
4. `ShowcaseWidgetCardContent` 里的交互禁用逻辑必须基于真实连接状态，而不是 store 默认值
5. 已确认同根因的 `TopicBusWindow` 必须一起接入共享补水逻辑，避免留下同类 bug
6. 若 runtime snapshot 获取失败，窗口必须优雅降级，不得阻断原有 `LoadHomeState` 身份恢复流程

#### Non-functional Requirements
- 最小安全改动:
  - 优先修共享前端 helper 与受影响窗口，不引入无关页面重构
- 一致性:
  - detached window 不再各自实现一份连接快照恢复逻辑
- 可维护性:
  - 会话补水逻辑放在明确的共享位置，便于后续窗口复用
- 稳定性:
  - 失败时保留现有事件监听与 fallback identity 流程
- 可测试性:
  - 至少用一条共享 helper 测试和一条窗口级测试覆盖本轮行为

#### Inputs / Outputs
- Inputs:
  - `SessionService.IsConnected`
  - `SessionService.LastAddr`
  - `App.HomeState`
  - detached window 启动路径中的 `loadHomeDefaults` 或同类初始化流程
- Outputs:
  - detached window 中的 `sessionStore.connected`
  - detached window 中的 `sessionStore.addr`
  - 正确的 UI badge 和交互 gating

#### Edge Cases
- 主窗口尚未进入 `Home` 页，但 detached window 直接首次启动
- `LoadHomeState` 成功，但 runtime session snapshot 查询失败
- session 已连接但没有 `lastAddr`
- detached window 在 profile 切换后重新加载
- runtime event 还没推到当前窗口时，初始化补水仍应先给出正确静态快照

#### Acceptance Criteria
1. 在 hub 已连接的前提下，`Showcase` 编辑窗口打开后显示 `Connected`
2. 在 hub 已连接的前提下，`Showcase` 预览窗口打开后显示 `Connected`
3. 这两个窗口中依赖 `sessionStore.connected` 的控件不再被错误禁用
4. `TopicBusWindow` 在同样场景下不再显示错误的 `Disconnected`
5. 共享 helper 与代表性窗口测试通过
6. 不引入新的 build / type / test 回归

#### Risks
- 若把 helper 放在过于底层的位置并自动执行，可能影响已有依赖 `sessionStore` 初始空态的测试
- 若窗口初始化顺序处理不当，可能出现 `LoadHomeState` 与连接快照写入先后覆盖问题
- 若扩大到所有窗口，改动面会不必要地变大；本轮只修已确认依赖 `sessionStore.connected` 的窗口

#### Issue List
- 无

### Stage 2 - Architecture Design
#### Overall Solution
- 推荐方案：在前端共享层新增 detached window 会话补水 helper，由它统一调用 `SessionService.IsConnected` / `LastAddr` 更新 `sessionStore`，然后在 `Showcase.vue`、`ShowcaseWindow.vue`、`TopicBusWindow.vue` 的启动流程中接入
- 同时让 `Home.vue` 复用该 helper，消除同一逻辑的双份实现

#### Alternatives Considered
- 方案 A: 只在 `Showcase.vue` 和 `ShowcaseWindow.vue` 本地补 `IsConnected` / `LastAddr`
  - 优点：改动最小
  - 缺点：保留 `TopicBusWindow` 同类 bug，并复制一套初始化逻辑
  - 结论：不选
- 方案 B: 在 `session` store 暴露共享连接快照补水 helper，并接入已确认受影响的 detached windows
  - 优点：最小共享抽象，解决 `Showcase` 和 `TopicBus` 同根因问题，也可让 `Home` 页去重
  - 缺点：需要补测试，避免影响依赖默认 store 初值的用例
  - 结论：选用
- 方案 C: 自动让 `useSessionStore()` 在首次调用时直接访问后端补水
  - 优点：调用方最少
  - 缺点：store getter 引入隐式异步副作用，测试和调用时序更难控
  - 结论：不选

#### Module Responsibilities
- `frontend/src/stores/session.ts`
  - 继续持有 `session.state` 事件监听
  - 新增显式的 runtime snapshot 补水方法，负责刷新 `connected` / `addr`
- `frontend/src/pages/Home.vue`
  - 复用共享 helper，而不是保留本地 `refreshConnectionSnapshot`
- `frontend/src/pages/Showcase.vue`
  - 在编辑窗口初始化时补水 session 快照，再继续现有 `LoadHomeState` / `showcase.load` 流程
- `frontend/src/windows/ShowcaseWindow.vue`
  - 在预览窗口初始化时补水 session 快照，再继续现有 `LoadHomeState` / `showcase.load` 流程
- `frontend/src/windows/TopicBusWindow.vue`
  - 同样接入共享 helper，修复同根因 disconnect 假象
- tests
  - 验证共享 helper 的行为
  - 验证代表性 detached window 能在启动时显示正确连接状态

#### Data / Call Flow
1. detached window mounted
2. 页面先调用 `LoadHomeState()` 恢复 `nodeId` / `hubId` fallback
3. 页面调用共享 session snapshot helper
4. helper 调用 `IsConnected()`；若为 true，再补 `LastAddr()`
5. helper 更新 `sessionStore.connected` / `sessionStore.addr`
6. 页面继续原有 store identity 和业务加载流程
7. 后续 runtime `session.state` 事件仍照常接管实时更新

#### Interface Draft
- `frontend/src/stores/session.ts`
  - 新增显式 helper，例如 `hydrateSessionConnectionSnapshot(): Promise<void>`
- 该 helper:
  - 不抛出致命错误给调用方
  - 不清空现有 `auth` 快照
  - 只负责更新 `connected` / `addr` / `lastStateAt`

#### Error Handling and Safety
- `IsConnected` / `LastAddr` 失败时:
  - 记录 `console.warn`
  - 保留现有 session store 状态
  - 不中断 `LoadHomeState`、`showcase.load`、`topicbus.loadPrefs` 等后续流程
- 若 `IsConnected()` 返回 false:
  - 只更新 `connected=false`
  - 不强行清空本地持久化身份字段
- runtime 事件优先级:
  - helper 只做启动补水；连接建立 / 断开后的实时变更仍由 `session.state` 事件驱动

#### Performance and Testing Strategy
- 性能:
  - 每个 detached window 启动最多多做一次 `IsConnected` 和一次 `LastAddr` 查询
  - 不新增轮询，不引入额外订阅
- Tests:
  - 新增 `frontend/src/stores/session.test.ts` 覆盖 helper 对 `connected` / `addr` 的刷新
  - 新增 `frontend/src/windows/ShowcaseWindow.test.ts` 或等价代表性窗口测试，覆盖已连接场景下的 badge / 交互 gating
  - 如 `TopicBusWindow` 改动面需要额外回归，再补 `frontend/src/windows/TopicBusWindow.test.ts`
- Validation:
  - `npm exec vitest run src/stores/session.test.ts`
  - `npm exec vitest run src/windows/ShowcaseWindow.test.ts`
  - 如新增 TopicBusWindow 测试，则一并执行

#### Extensibility Design Points
- 后续若有更多 detached window 需要真实 session 快照，可复用同一 helper
- 不把 helper 做成隐式自动执行，避免 future tests / startup 顺序失控

#### Issue List
- 无

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 `Showcase` 编辑窗口、预览窗口和 `TopicBusWindow` 都会直接读取 `sessionStore.connected` 渲染 badge 或 ready-check
- 这些窗口的启动路径只调用 `LoadHomeState()` 恢复 `nodeId` / `hubId`，没有像 `Home.vue` 那样显式调用 `IsConnected()` / `LastAddr()`
- `session` store 默认 `connected=false`，所以 detached window 在新前端上下文里会先落到错误的 `Disconnected`
- 这是 detached window 冷启动补水缺失，不是 hub 实际断开

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Canonical destination:
  - 稳定需求 / 规格：不改
  - 执行控制面：worktree 根 `plan.md`
  - 完成结果：`docs/change/2026-04-14_win-showcase-window-session-sync.md`
  - lessons：stage 4 再判断是否需要新增 detached window 会话补水 lesson
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `docs/requirements/showcase-display-widgets.md`
- Related specs:
  - `docs/specs/showcase-display-widgets.md`
- Related lessons:
  - `docs/lessons/wails-packaged-aux-window-fallback.md`
  - `docs/lessons/detached-window-session-snapshot-hydration.md`
  - `docs/lessons/frontend-vitest-junction-preserve-symlinks.md`

#### Executable Task List
- [x] `SWS-1` 提取共享 session runtime snapshot helper，并让 `Home.vue` 复用
- [x] `SWS-2` 接入 `Showcase` 编辑页、`ShowcaseWindow`、`TopicBusWindow`
- [x] `SWS-3` 补测试并完成定向验证

#### Task Details
##### `SWS-1` - Shared Session Snapshot Hydration
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-showcase-window-session-sync`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-showcase-window-session-sync\plan.md`
- Goal:
  - 把 `IsConnected` / `LastAddr` 连接快照恢复逻辑集中到共享 helper
- Files / Modules:
  - `frontend/src/stores/session.ts`
  - `frontend/src/pages/Home.vue`
- Write Set:
  - 上述 2 个文件
- Acceptance:
  - 共享 helper 能更新 `sessionStore.connected` / `sessionStore.addr`
  - `Home.vue` 继续保留现有行为，但不再自己维护重复逻辑
- Tests:
  - `frontend/src/stores/session.test.ts`
- Rollback:
  - 回退 helper 与 `Home.vue` 对其的引用

##### `SWS-2` - Detached Window Consumers
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-showcase-window-session-sync`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-showcase-window-session-sync\plan.md`
- Goal:
  - 修复已确认受影响的 detached window disconnect 假象
- Files / Modules:
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`
  - `frontend/src/windows/TopicBusWindow.vue`
- Write Set:
  - 上述 3 个文件
- Acceptance:
  - 这些窗口初始化后使用真实 session 快照
  - UI badge 和依赖 `sessionStore.connected` 的 gating 与真实状态一致
- Tests:
  - `frontend/src/windows/ShowcaseWindow.test.ts`
  - 如有必要，`frontend/src/windows/TopicBusWindow.test.ts`
- Rollback:
  - 回退窗口初始化中的 helper 调用

##### `SWS-3` - Verification
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-showcase-window-session-sync`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-showcase-window-session-sync\plan.md`
- Goal:
  - 用测试和定向检查锁定本轮修复
- Files / Modules:
  - `frontend/src/stores/session.test.ts`
  - `frontend/src/windows/ShowcaseWindow.test.ts`
  - 如需要，再加 `frontend/src/windows/TopicBusWindow.test.ts`
- Write Set:
  - 上述测试文件
- Acceptance:
  - 共享 helper 测试通过
  - detached window 代表性测试通过
  - 定向 `vitest` 通过
- Tests:
  - `node --preserve-symlinks --preserve-symlinks-main ./node_modules/vitest/vitest.mjs run src/lib/auxWindow.test.ts src/stores/session.test.ts src/windows/ShowcaseWindow.test.ts`
- Rollback:
  - 删除新增测试并回退相关实现

#### Dependencies
- `SessionService` 已提供 `IsConnected` / `LastAddr`
- `LoadHomeState` 继续作为 identity fallback 来源
- 现有 detached window 测试基建可用

#### Risks and Notes
- 只补“启动快照”，不改变 runtime event 订阅路径
- `TopicBusWindow` 已确认与 `Showcase` 共用同一根因；一起修可以避免再次返工
- `StreamSourceWindow` / `StreamDeliveryWindow` / `FlowEditorWindow` 虽也调用 `LoadHomeState`，但目前未发现它们直接以 `sessionStore.connected` 决定 badge 或 gating；本轮先不扩大范围

#### Parallelism Assessment
- 不派发子 Agent
- 原因：改动集中在同一条会话补水链路和几个相关窗口，串行实现更容易保证时序一致和统一验证

#### Issue List
- 无

阻塞：否
进入 3.2

### Stage 3.2 - Implementation Record
#### File-level Change Summary
- `SWS-1`
  - `frontend/src/stores/session.ts`
    - 新增 `hydrateSessionConnectionSnapshot()`，集中补水 `IsConnected()` / `LastAddr()` 到 `sessionStore`
  - `frontend/src/pages/Home.vue`
    - 删除本地重复的连接快照刷新逻辑，改为复用共享 helper
- `SWS-2`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`
  - `frontend/src/windows/TopicBusWindow.vue`
    - 在 `loadHomeDefaults()` 中先恢复 `HomeState`，再显式补水 session snapshot，最后再设置 identity
- `SWS-3`
  - `frontend/src/stores/session.test.ts`
    - 覆盖共享 helper 的 connected / disconnected 两条主路径
  - `frontend/src/windows/ShowcaseWindow.test.ts`
    - 覆盖 detached window 已连接启动时的 badge / widget gating 初始化

#### Validation Notes
- worktree 的 `frontend/node_modules` 通过 Windows junction 指向主仓依赖目录时，直接运行 Vitest 会报 `TypeError: Cannot read properties of undefined (reading 'config')`
- 本轮验证命令改为显式使用：
  - `node --preserve-symlinks --preserve-symlinks-main ./node_modules/vitest/vitest.mjs run ...`
- 该现象已纳入 lesson，避免后续把它误判为产品代码回归

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 已覆盖 Showcase 编辑页、Showcase 预览窗口、TopicBusWindow 的 session 快照补水
- 架构合理性：通过
  - 共享 helper 收口到 `session` store，避免在多个 detached window 重复拼接 Wails 调用
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 每次窗口冷启动只增加一次 `IsConnected()` 和最多一次 `LastAddr()`，无新增轮询
- 可读性与一致性：通过
  - `Home.vue` 与 detached windows 复用同一 helper，命名和职责边界清晰
- 可扩展性与配置化：通过
  - 后续其他 detached window 若依赖 `sessionStore.connected`，可直接复用同一补水入口
- 稳定性与安全：通过
  - Wails 查询失败时只 `console.warn`，不阻断原有 `LoadHomeState()` fallback
- 测试覆盖情况：通过
  - 共享 helper 测试与代表性窗口测试均已通过
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子Agent

### Stage 4 - Archive Prep
- 使用 `$m-docs` 完成 requirement/spec/lesson 影响复核
- `Requirements impact: none`
- `Specs impact: none`
- `Lessons impact: updated`
- 归档目标：
  - `docs/change/2026-04-14_win-showcase-window-session-sync.md`
  - `docs/lessons/detached-window-session-snapshot-hydration.md`
  - `docs/lessons/frontend-vitest-junction-preserve-symlinks.md`
