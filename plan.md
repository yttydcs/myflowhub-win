# Plan - Win Approval Actions Layout

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-approval-actions-layout`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-approval-actions-layout`
- Current Stage: `4 archived (waiting for workflow-end confirmation)`

## Stage Records

### Initialization
- `guide.md`:
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`
  - 已读取 `frontend-design` 技能说明；本轮只做注册审批页的最小视觉收敛，不重做 authority 页面语言
- base/worktree confirmation:
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `fix/win-approval-actions-layout`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-win-approval-actions-layout`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 收敛 `Registration Approvals` 页头部动作区，消除摘要卡中部突兀的 `Resolve / Refresh` 双按钮，同时保留待审批列表的刷新可达性。

#### Scope
- 必须:
  - 保持 `Registration Approvals` 的 authority 解析、pending list 读取、approve / reject 行为不变
  - 调整 `Resolve / Refresh` 的可见布局，不再让两个按钮悬在摘要卡正文中间
  - 若某个动作在现有交互中已属冗余，应允许收敛掉冗余入口，但不能破坏 authority 自动解析链路
  - 页面测试需要覆盖新的动作布局或入口存在性
- 可选:
  - 同步统一刷新按钮的禁用条件或文案
  - 在更合适的 header 区域承载刷新动作
- 不做:
  - 不修改后端 `ResolveAuthority` / `ListPendingRegisters` / `ApproveRegister` / `RejectRegister` 契约
  - 不重构注册审批列表、review dialog 或 store 数据结构
  - 不改动访问策略、准入许可页面

#### Use Cases
- 管理员进入注册审批页后，希望先看到身份和 pending 摘要，而不是被居中的控制按钮打断视线
- 管理员需要手动回刷待审批列表时，希望动作入口自然附着在列表或页头上
- 管理员不需要单独理解“先解析再刷新”的两步心智，页面应优先呈现单一可用动作

#### Functional Requirements
- `Registration Approvals` 页面必须继续支持：
  - authority 自动解析
  - pending list 读取与刷新
  - approve 单条 request
  - reject 单条 request
- 手动刷新入口必须保留
- 若移除单独 `Resolve` 入口，`Refresh` 仍必须通过既有 authority store 在需要时自动完成 authority 解析
- 页面主体摘要区不应继续出现孤立的双按钮动作行

#### Non-functional Requirements
- 优先做最小安全改动，不新增状态字段或重复请求
- 保持当前 authority 页面紧凑、轻量的视觉语言
- 动作层级必须更清晰，避免让次要控制抢占摘要区视觉中心
- 测试至少要覆盖新布局下的主动作可达性

#### Inputs / Outputs
- 输入:
  - 当前 session 身份 `sourceId / hubId`
  - authority store 当前 `authorityId`
  - pending request 列表
- 输出:
  - 更自然的刷新入口布局
  - 保持不变的 authority 解析与 pending 刷新行为

#### Edge Cases
- authority 尚未解析，但用户点击刷新
- 页面首次自动加载尚未完成时手动刷新
- 有 request 正在 approve / reject 时，刷新按钮禁用逻辑仍需稳定
- 身份切换后 authorityId 清空，刷新入口仍应走既有重解析路径

#### Acceptance Criteria
- 注册审批页不再在摘要卡正文中部显示 `Resolve / Refresh` 双按钮
- 页面仍然存在明确的手动刷新入口
- `Refresh` 在 authority 未解析时仍能正常工作
- `RegistrationApprovals` 测试和前端构建通过

#### Risks
- 若直接删除 `Resolve` 但没有清楚保留 `Refresh`，用户可能误以为失去手动控制
- 若把刷新入口放得过深，可能降低操作可发现性

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 采用最小前端收敛方案：
  - 从摘要卡正文移除独立的 `Resolve / Refresh` 按钮行
  - 删除冗余的显式 `Resolve` 操作
  - 将单一 `Refresh` 收敛到 `Pending Queue` 的 header actions 中
- 不触碰 `registrationApprovals` store，也不修改 authority store 的自动解析链路；`loadPending()` 继续通过 `requireAuthority()` 在需要时自动解析 authority。

#### Alternatives Considered
- 方案 A（采用）：删除显式 `Resolve`，只保留 `Refresh`，并把它移到 `Pending Queue` header
  - 优点：心智最简单，布局也最自然
  - 代价：失去单独“只解析不刷新”的显式按钮
- 方案 B：保留 `Resolve` 和 `Refresh`，但整体移到 header actions
  - 优点：不改功能面
  - 代价：仍然保留冗余动作，只是位置更好
- 方案 C：保留主体按钮行，只改样式弱化
  - 优点：改动更小
  - 代价：仍然会打断摘要区节奏，不能真正解决突兀感

#### Module Responsibilities
- `frontend/src/pages/RegistrationApprovals.vue`
  - 收敛动作区布局
  - 移除冗余的 `Resolve` UI 入口
  - 保留并重定位 `Refresh`
- `frontend/src/stores/registrationApprovals.ts`
  - 保持现有 pending list 加载与 approve / reject 逻辑不变
- `frontend/src/stores/authority.ts`
  - 保持 `requireAuthority()` 自动解析行为不变
- `frontend/src/pages/RegistrationApprovals.test.ts`
  - 覆盖新动作入口和页面仍可加载的关键断言

#### Data / Call Flow
1. 页面 ready 后仍自动调用 `loadPending(true)`
2. `loadPending()` 内部继续走 `approvalsStore.loadPending()`
3. `approvalsStore.loadPending()` 继续通过 `authority.requireAuthority()` 自动确保 authority 可用
4. 用户需要手动更新时，从 `Pending Queue` header 触发 `Refresh`
5. approve / reject 成功后仍由 store 内部回刷 pending list

#### Interface Drafts
- 顶部摘要卡:
  - 保留连接状态、登录状态、identity、authority badges
  - 不再在正文中部放置独立按钮行
- `Pending Queue` header actions:
  - `Refresh`

#### Error Handling and Safety
- `Refresh` 继续复用现有 `loadPending(false)`，沿用既有 toast / error handling
- `busyRequestId` 非空或 `loading` 时，刷新按钮继续禁用
- 不新增任何可能绕开 `ensureReady()` 的路径

#### Performance and Testing Strategy
- 不增加任何新的 authority 请求
- 验证重点:
  - `frontend/src/pages/RegistrationApprovals.test.ts`
  - `npm test -- RegistrationApprovals`
  - `npm run build`
- 如本地 dev host 可用，可追加一次 `chrome-devtools` 页面冒烟，确认按钮位置更自然

#### Extensibility Design Points
- 若后续注册审批页也需要显式 loading notice，可复用当前 header actions 的承载方式
- 若将来真的需要“只解析 authority 不拉列表”的诊断入口，更适合放到开发/诊断语境，而不是页面主体

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 让 `Registration Approvals` 的手动动作入口更自然，不再打断摘要区
- Current State:
  - 顶部摘要卡正文中间存在独立 `Resolve / Refresh` 按钮行
  - `Refresh` 实际已经能在需要时自动解析 authority，`Resolve` 在页面主路径中较冗余

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Requirements impact: `none`
- Specs impact: `none`
- Stable docs destination:
  - none
- Change archive destination:
  - `docs/change/2026-03-27_win-approval-actions-layout.md`
- Lessons impact:
  - none

#### Related Requirements / Specs / Lessons
- Requirements:
  - `D:\project\MyFlowHub3\worktrees\fix-win-approval-actions-layout\docs\requirements\authority-admin-console.md`
- Specs:
  - `D:\project\MyFlowHub3\worktrees\fix-win-approval-actions-layout\docs\specs\authority-admin-console.md`
- Lessons:
  - `D:\project\MyFlowHub3\worktrees\fix-win-approval-actions-layout\docs\lessons\README.md`

#### Executable Task List
- [x] IMPL-RAA-1 收敛注册审批页动作入口布局
- [x] TEST-RAA-1 更新注册审批页面测试
- [x] REVIEW-RAA-1 完成 3.3 代码复核
- [x] ARCHIVE-RAA-1 归档到 `docs/change`

#### Task Details
##### IMPL-RAA-1 - 收敛注册审批页动作入口布局
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-approval-actions-layout`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-approval-actions-layout\plan.md`
- Goal:
  - 去掉摘要卡中间突兀的双按钮，只保留自然的刷新入口
- Files / Modules:
  - `frontend/src/pages/RegistrationApprovals.vue`
- Write Set:
  - `frontend/src/pages/RegistrationApprovals.vue`
- Acceptance:
  - 页面主体不再出现 `Resolve / Refresh` 双按钮行
  - `Refresh` 仍可用且布局更自然
- Test Points:
  - `npm test -- RegistrationApprovals`
  - `npm run build`
- Rollback:
  - 回退注册审批页按钮布局与 handler 引用

##### TEST-RAA-1 - 注册审批页回归验证
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-approval-actions-layout`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-approval-actions-layout\plan.md`
- Goal:
  - 锁定页面新动作布局和手动刷新入口存在性
- Files / Modules:
  - `frontend/src/pages/RegistrationApprovals.test.ts`
- Write Set:
  - `frontend/src/pages/RegistrationApprovals.test.ts`
- Acceptance:
  - 测试覆盖主体不再渲染 `Resolve`
  - 测试覆盖 `Refresh` 仍可见
- Test Points:
  - `npm test -- RegistrationApprovals`
  - `npm run build`
- Rollback:
  - 回退新增断言

#### Dependencies
- `RegistrationApprovals.vue` 与 `registrationApprovals.ts`、`authority.ts`、toast 高度耦合
- `loadPending()` 当前通过 `requireAuthority()` 隐式依赖 authority 自动解析链路

#### Risks and Notes
- 本轮是现有交互的收敛，不涉及服务接口或数据结构变化
- `Resolve` 若仍被某些诊断路径依赖，后续应放到更合适的位置，而不是恢复到摘要区正文

#### Parallelism Assessment
- 不派发子Agent
- 原因:
  - 写集集中在单个页面和对应测试，规模小且高度耦合
  - 当前会话未获得显式子Agent委派授权
- Owner:
  - 主Agent

#### Issue List
- none

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Completed Work
- `IMPL-RAA-1`
  - 删除注册审批页摘要卡正文中的独立 `Resolve / Refresh` 按钮行
  - 移除显式 `Resolve` UI 入口，保留 `Refresh` 并收敛到 `Pending Queue` header
  - 保持 `loadPending()` 与 authority 自动解析链路不变
- `TEST-RAA-1`
  - 更新 `frontend/src/pages/RegistrationApprovals.test.ts`
  - 新增“无 `Resolve`、保留单一 `Refresh` 入口”的断言

#### Files Updated
- `frontend/src/pages/RegistrationApprovals.vue`
- `frontend/src/pages/RegistrationApprovals.test.ts`

#### Implementation Notes
- 未修改 `frontend/src/stores/registrationApprovals.ts`
- 未修改 `frontend/src/stores/authority.ts`
- `Refresh` 继续通过 `requireAuthority()` 隐式确保 authority 已解析

### Stage 3.3 - Review
#### Review Checklist
- 需求覆盖：
  - 通过，页面仍保留手动刷新，且去除了突兀的中部双按钮
- 架构合理性：
  - 通过，仅收敛页面动作布局，不触碰 store / service 契约
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：
  - 通过，没有新增请求或重复解析
- 可读性与一致性：
  - 通过，动作入口更贴近列表语境
- 可扩展性与配置化：
  - 通过，未来若有更多列表动作可继续挂在 `Pending Queue` header
- 稳定性与安全：
  - 通过，仍复用既有 `ensureReady()` 与 `requireAuthority()`
- 测试覆盖情况：
  - 通过，页面测试与前端构建通过
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：
  - 通过，未使用子Agent

#### Validation Results
- `npm test -- RegistrationApprovals`
  - passed
- `npm run build`
  - 首次失败，原因是 fresh worktree 缺少 `frontend/wailsjs/**`
- `$env:GOWORK='off'; wails generate module`
  - passed
- `npm run build`
  - passed

#### Review Conclusion
- 通过，可进入归档

### Stage 4 - Archive
#### Archive Outputs
- `docs/change/2026-03-27_win-approval-actions-layout.md`
- `docs/change/README.md`

#### Lessons Decision
- `none`
- 原因：本轮属于局部界面收敛，没有新增可复用的排障模式

#### Workflow Status
- 已完成本轮实现、验证和归档
- 等待用户决定是否结束当前 workflow
