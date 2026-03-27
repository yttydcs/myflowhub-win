# Plan - Win Approval Filter Button Height

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-approval-filter-button-height`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-approval-filter-button-height`
- Current Stage: `4 archived (waiting for workflow-end confirmation)`

## Stage Records

### Initialization
- `guide.md`:
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`
  - 已读取 `frontend-design` 技能说明；本轮只做注册审批页筛选区的最小视觉修正，不改变现有 authority 页面语言
- base/worktree confirmation:
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `fix/win-approval-filter-button-height`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-win-approval-filter-button-height`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 修正 `Registration Approvals` 页中“设备筛选”输入框右侧按钮高度偏矮的问题，让筛选区在视觉上更和谐。

#### Scope
- 必须:
  - 保持注册审批页的筛选、刷新、approve / reject 行为不变
  - 调整筛选区按钮高度，使其与左侧输入框在视觉上对齐
  - 改动应局限在注册审批页筛选区和必要的页面测试
- 可选:
  - 为按钮补一个更稳定的测试定位属性
- 不做:
  - 不修改注册审批 store 或 authority store 契约
  - 不调整其他按钮尺寸体系
  - 不改动访问策略、准入许可页面

#### Use Cases
- 管理员在注册审批页使用设备 ID 筛选时，希望输入框和右侧动作按钮看起来像同一组控件，而不是右边明显矮一截

#### Functional Requirements
- 设备筛选输入框右侧的 `Apply Filter` 按钮必须保持可用
- `Apply Filter` 点击后仍需调用现有 `loadPending(false)` 链路
- 筛选按钮高度应与筛选输入框保持一致

#### Non-functional Requirements
- 优先做最小安全改动，不新增状态或额外请求
- 保持当前 authority 页面紧凑、克制的视觉语言
- 测试至少覆盖筛选按钮入口仍存在且具备新的高度类

#### Inputs / Outputs
- 输入:
  - `approvalsStore.state.filterDeviceId`
  - 现有 `inputClass`
  - 现有 `Apply Filter` 按钮配置
- 输出:
  - 对齐后的筛选按钮尺寸
  - 不变的筛选刷新行为

#### Edge Cases
- 按钮在 `loading` 或 `busyRequestId` 非空时仍需保持既有禁用行为
- 小屏下筛选区换行后，按钮高度仍需与输入控件一致

#### Acceptance Criteria
- 注册审批页中设备筛选按钮不再明显矮于输入框
- `Apply Filter` 仍可正常触发 pending list 刷新
- `RegistrationApprovals` 测试和前端构建通过

#### Risks
- 若直接复用全局默认按钮尺寸，仍可能与输入框高度不完全一致
- 若只改样式但不加测试，后续容易被 `size="sm"` 回退

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 继续沿用注册审批页当前筛选区结构，仅把 `Apply Filter` 按钮从 `size="sm"` 调整为显式 `h-10` 高度，并补一个测试定位属性。
- 不修改共享 `Button` 组件的全局 size 规则，避免影响全站其它 `sm` 按钮。

#### Alternatives Considered
- 方案 A（采用）：只在当前 `Apply Filter` 按钮上增加局部高度类
  - 优点：写集最小，不会波及其它页面
  - 代价：高度对齐规则保留在页面局部
- 方案 B：把全局 `Button` 的 `sm` 尺寸改成 `h-10`
  - 优点：统一
  - 代价：会影响大量现有小按钮，风险过大
- 方案 C：把输入框改矮去适配 `sm` 按钮
  - 优点：理论上也能对齐
  - 代价：会破坏现有输入控件基线，不如只调按钮安全

#### Module Responsibilities
- `frontend/src/pages/RegistrationApprovals.vue`
  - 调整筛选按钮局部尺寸和测试定位属性
- `frontend/src/pages/RegistrationApprovals.test.ts`
  - 校验筛选按钮存在且带有对齐高度类

#### Data / Call Flow
1. 用户输入 `filterDeviceId`
2. 点击 `Apply Filter`
3. 页面继续调用现有 `loadPending(false)`
4. `approvalsStore.loadPending()` 按既有逻辑刷新列表

#### Interface Drafts
- `Device Filter` 行：
  - 左侧输入框保持 `h-10`
  - 右侧 `Apply Filter` 按钮改为同高 `h-10`

#### Error Handling and Safety
- 不更改 `loadPending()` 的错误处理
- 不更改筛选按钮的禁用条件

#### Performance and Testing Strategy
- 不增加任何新的 authority 请求
- 验证重点:
  - `frontend/src/pages/RegistrationApprovals.test.ts`
  - `npm test -- RegistrationApprovals`
  - `npm run build`
- fresh worktree 如缺 `node_modules` 或 `wailsjs`，按既有预热流程补齐后再验证

#### Extensibility Design Points
- 若后续 authority 页面也有“输入框 + 动作按钮”组合，可考虑抽离统一的 filter action pattern；本轮先不扩 scope

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 让注册审批页的设备筛选区在视觉上更对齐
- Current State:
  - 输入框使用 `h-10`
  - 右侧 `Apply Filter` 按钮使用 `size="sm"`，只有 `h-8`

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Requirements impact: `none`
- Specs impact: `none`
- Stable docs destination:
  - none
- Change archive destination:
  - `docs/change/2026-03-27_win-approval-filter-button-height.md`
- Lessons impact:
  - none

#### Related Requirements / Specs / Lessons
- Requirements:
  - `D:\project\MyFlowHub3\worktrees\fix-win-approval-filter-button-height\docs\requirements\authority-admin-console.md`
- Specs:
  - `D:\project\MyFlowHub3\worktrees\fix-win-approval-filter-button-height\docs\specs\authority-admin-console.md`
- Lessons:
  - `D:\project\MyFlowHub3\worktrees\fix-win-approval-filter-button-height\docs\lessons\README.md`

#### Executable Task List
- [x] IMPL-RAF-1 调整注册审批页筛选按钮高度
- [x] TEST-RAF-1 更新注册审批页面测试
- [x] REVIEW-RAF-1 完成 3.3 代码复核
- [x] ARCHIVE-RAF-1 归档到 `docs/change`

#### Task Details
##### IMPL-RAF-1 - 调整注册审批页筛选按钮高度
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-approval-filter-button-height`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-approval-filter-button-height\plan.md`
- Goal:
  - 让设备筛选按钮与左侧输入框同高
- Files / Modules:
  - `frontend/src/pages/RegistrationApprovals.vue`
- Write Set:
  - `frontend/src/pages/RegistrationApprovals.vue`
- Acceptance:
  - 按钮不再明显矮于输入框
  - 点击行为保持不变
- Test Points:
  - `npm test -- RegistrationApprovals`
  - `npm run build`
- Rollback:
  - 回退注册审批页筛选按钮的局部类名调整

##### TEST-RAF-1 - 注册审批页筛选区回归验证
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-approval-filter-button-height`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-approval-filter-button-height\plan.md`
- Goal:
  - 锁定筛选按钮入口和对齐高度类
- Files / Modules:
  - `frontend/src/pages/RegistrationApprovals.test.ts`
- Write Set:
  - `frontend/src/pages/RegistrationApprovals.test.ts`
- Acceptance:
  - 测试覆盖筛选按钮存在
  - 测试覆盖筛选按钮包含 `h-10`
- Test Points:
  - `npm test -- RegistrationApprovals`
  - `npm run build`
- Rollback:
  - 回退新增断言

#### Dependencies
- `RegistrationApprovals.vue` 与 `registrationApprovals.ts` 的筛选刷新链路耦合
- 共享 `Button` 组件的 `sm` size 当前固定为 `h-8`

#### Risks and Notes
- 本轮只做页面局部样式修正，不应回退为全局按钮体系调整

#### Parallelism Assessment
- 不派发子Agent
- 原因:
  - 写集集中在单个页面和其测试，规模小且高度耦合
  - 当前会话未获得显式子Agent委派授权
- Owner:
  - 主Agent

#### Issue List
- none

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Completed Work
- `IMPL-RAF-1`
  - 为注册审批页 `Apply Filter` 按钮增加局部 `h-10` 高度类
  - 补充 `data-approval-filter-apply` 定位属性，便于后续测试和排查
- `TEST-RAF-1`
  - 为注册审批页新增筛选按钮高度对齐测试
  - 保持原有刷新与审阅路径测试不变

#### Files Updated
- `frontend/src/pages/RegistrationApprovals.vue`
- `frontend/src/pages/RegistrationApprovals.test.ts`

#### Implementation Notes
- 未修改共享 `Button` 组件
- 未修改注册审批 store、authority store 或任何后端协议

### Stage 3.3 - Review
#### Review Checklist
- 需求覆盖：
  - 通过，设备筛选按钮已与输入框同高
- 架构合理性：
  - 通过，仅调整页面局部样式，不影响共享按钮体系
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：
  - 通过，无新增请求或重复计算
- 可读性与一致性：
  - 通过，筛选控件组更一致，且保留稳定测试定位点
- 可扩展性与配置化：
  - 通过，本轮避免修改全局尺寸体系，影响面最小
- 稳定性与安全：
  - 通过，按钮禁用和点击行为未改
- 测试覆盖情况：
  - 通过，定向测试与前端构建通过
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
- `docs/change/2026-03-27_win-approval-filter-button-height.md`
- `docs/change/README.md`

#### Lessons Decision
- `none`
- 原因：本轮是局部视觉修正，没有形成新的通用排障经验

#### Workflow Status
- 已完成本轮实现、验证和归档
- 等待用户决定是否结束当前 workflow
