# Plan - Win Access Policy Save Loading

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-access-policy-save-loading`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading`
- Current Stage: `4 archived (waiting for workflow-end confirmation)`

## Stage Records

### Initialization
- `guide.md`:
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `frontend-design` 技能说明；本轮只在既有 authority 视觉语言内做最小交互增强，不重做整体风格
- base/worktree confirmation:
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `fix/win-access-policy-save-loading`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 为 `Access Policy` 页面补上更直接的保存入口和更明确的加载反馈，降低“只有一个保存按钮”和“加载时无感知”的使用成本。

#### Scope
- 必须:
  - 保持现有 `AccessPolicy` 路由、authority 解析和 `SavePolicy` / `LoadPolicy` 后端契约不变
  - 在访问策略主要编辑卡片附近提供可见的保存入口，不再只依赖操作面板中的单一保存按钮
  - 各卡片保存入口应继续复用当前整份 policy 的保存链路，不能引入新的部分保存协议
  - 页面加载策略时必须给出明确的页面内提示，而不只是按钮禁用或成功 toast
  - 加载提示在自动首次加载和手动重新加载时都应生效
  - 中英文文案与现有 authority 页面保持一致
  - 前端测试需要覆盖新增保存入口和加载提示
- 可选:
  - 在保存按钮文案上同步体现 `saving` 状态
  - 在当前 tab 内补充更轻的加载摘要文案
- 不做:
  - 不新增按卡片局部持久化的后端接口
  - 不修改 `accessPolicy.ts` store 的基础数据结构
  - 不改动注册审批 / 准入许可页面
  - 不重做访问策略页面整体布局

#### Use Cases
- 管理员在默认准入卡片或节点覆盖卡片附近改完内容后，希望就地点击保存，而不是再滚到操作面板找唯一的保存按钮
- 管理员在角色管理页维护角色后，希望在角色页头就能直接保存当前 policy
- 管理员打开访问策略页或手动重新加载时，希望看到“正在加载”的明确反馈，知道页面仍在等待 authority 返回

#### Functional Requirements
- `Access Policy` 页面必须继续支持：
  - authority 自动解析
  - policy load / save
  - runtime list_roles 预览
  - `auth.get_perms` 单节点查询
- `Current Policy` tab 中至少以下区域需要提供可见的保存入口：
  - 默认准入卡片
  - 节点覆盖卡片
  - 操作面板（保留现有保存入口）
- `Role Management` tab 也必须提供可见的保存入口
- 新增的各保存入口必须调用同一条现有 `savePolicy(saveOptions)` 链路，不得引入不同的持久化语义
- 页面在 `loadPolicy()` 进行中时，必须展示明确的加载提示；提示不应依赖 toast
- 页面在 `loadPolicy()` 完成后，加载提示必须消失

#### Non-functional Requirements
- 优先做最小安全改动，不扩大到新的 store 状态或后端契约
- 保持当前 `Access Policy` 已稳定的紧凑布局和卡片层级，不把页面重新堆回大块表单
- 保存入口的增加不能让按钮语义混乱；同名入口应共享一致禁用条件和行为
- 加载反馈应足够醒目但不能遮挡现有编辑内容
- 测试需覆盖“按钮存在”和“加载提示可见”这两个回归点

#### Inputs / Outputs
- 输入:
  - 当前 session 身份 `sourceId / hubId`
  - resolved `authorityId`
  - persisted `policy`
  - runtime snapshot
  - 当前页面内的 `saveOptions`
- 输出:
  - 卡片附近的保存按钮
  - 页面内明确的加载提示
  - 现有 policy save / load 行为保持不变

#### Edge Cases
- authority 未解析
- 页面首次自动加载时仍未准备好身份
- 手动 reload 与自动初次加载重叠
- `loading` 与 `saving` 状态交叠时按钮禁用必须一致
- 角色管理 tab 下没有任何角色，但仍需要保存入口

#### Acceptance Criteria
- 访问策略页不再只有操作面板一个保存按钮
- 默认准入、节点覆盖、角色管理都能在附近看到保存入口
- 所有新增保存入口都复用现有 policy 保存链路
- 页面在加载 policy 时有明确的页面内提示
- `AccessPolicy` 测试和前端构建通过

#### Risks
- 若卡片级保存入口语义不清，用户可能误以为它们只保存局部内容
- 若加载提示放置不当，可能与现有 Badge / Header 信息竞争视觉优先级
- 若测试只校验文案存在，不校验状态切换，后续容易回退为“只有 disabled 没有提示”

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 保持 `frontend/src/stores/accessPolicy.ts` 的 `loadPolicy()` / `savePolicy()` 契约不变，仅在 `frontend/src/pages/AccessPolicy.vue` 增加两个前端层级增强：
  - 为主要编辑卡片补 `Save Policy` 入口，所有入口统一调用现有 `savePolicy()`
  - 基于 `accessPolicyStore.state.loading` 增加页面内加载提示，并同步让相关按钮展示一致的 busy 状态

#### Alternatives Considered
- 方案 A（采用）：在主要卡片动作区增加统一保存按钮，并增加页面内 loading notice
  - 优点：改动最小，不改变保存语义
  - 代价：多个按钮仍然保存整份 policy，需要在布局上保持语义一致
- 方案 B：把保存拆成 default / node overrides / roles 三条局部保存链路
  - 优点：按钮语义最直观
  - 代价：现有后端没有局部保存契约，改动面过大
- 方案 C：只做浮动保存条或悬浮按钮
  - 优点：避免重复按钮
  - 代价：用户明确要求“各个卡片里面都能有一个保存按钮”，不满足预期

#### Module Responsibilities
- `frontend/src/pages/AccessPolicy.vue`
  - 增加卡片级保存入口
  - 增加加载提示展示和统一 busy 文案
- `frontend/src/stores/accessPolicy.ts`
  - 保持现有 `loading` / `saving` 状态语义不变
- `frontend/src/i18n/messages/operations.ts`
  - 补充加载提示与按钮 busy 文案
- `frontend/src/pages/AccessPolicy.test.ts`
  - 覆盖新增保存入口和加载提示
- `docs/requirements/authority-admin-console.md`
  - 补充访问策略保存入口和加载反馈约束
- `docs/specs/authority-admin-console.md`
  - 记录卡片保存入口仍复用统一 policy 保存链路

#### Data / Call Flow
1. 页面进入后仍通过既有 watcher 调用 `loadPolicy(true)`
2. `accessPolicyStore.state.loading = true` 时，页面渲染加载提示
3. `loadPolicy()` 返回后，提示消失，表单与 runtime 数据按既有逻辑同步
4. 用户在默认准入 / 节点覆盖 / 角色管理 / 操作面板任一位置点击保存
5. 所有按钮都调用同一 `savePolicy()`，由现有 `saveOptions` 决定保存范围

#### Interface Drafts
- `Current Policy` tab
  - 顶部保留现有 header 与 summary cards
  - `Default Access` actions:
    - `Manage Roles`
    - `Save Policy`
    - `Edit Default Access`
  - `Node Overrides` actions:
    - `Save Policy`
    - `Add Node Role`
  - header 下方增加 `loading notice`
- `Role Management` tab
  - 页头 actions:
    - `Save Policy`
    - `Add Role`
  - tab 内容顶部或 header 下方增加 `loading notice`

#### Error Handling and Safety
- 各保存按钮在 `loading` 或 `saving` 时统一禁用
- 加载提示只反映 `loadPolicy()`，不额外引入新的请求状态
- 保存失败继续沿用现有 toast / validation 流程，不改变错误处理

#### Performance and Testing Strategy
- 不增加任何新的 authority 请求
- 验证重点:
  - `frontend/src/pages/AccessPolicy.test.ts`
  - `npm test -- AccessPolicy`
  - `npm run build`
- 如 dev host 可用，可追加一次 `chrome-devtools` 页面冒烟，确认 loading notice 和新增保存按钮可见

#### Extensibility Design Points
- 若未来真的需要局部保存，当前卡片级按钮位置可以保留，再把 handler 从统一保存拆成局部保存
- 当前 loading notice 可作为 authority 页面通用模式，后续如 approvals / permit 也需要显式加载提示可复用

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 让 `Access Policy` 页面在主要编辑卡片附近就能保存，并在加载 policy 时给出明确提示
- Current State:
  - 页面目前只有操作面板内一个 `Save Policy`
  - `loading` 只影响按钮禁用，没有稳定的页面内加载提示

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Requirements impact: `clarify`
- Specs impact: `clarify`
- Stable docs destination:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Change archive destination:
  - `docs/change/2026-03-27_win-access-policy-save-loading.md`
- Lessons impact:
  - none

#### Related Requirements / Specs / Lessons
- Requirements:
  - `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading\docs\requirements\authority-admin-console.md`
- Specs:
  - `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading\docs\specs\authority-admin-console.md`
- Lessons:
  - `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading\docs\lessons\README.md`

#### Executable Task List
- [x] DOC-APL-1 更新访问策略 requirements/specs
- [x] IMPL-APL-1 为访问策略主要卡片增加保存入口
- [x] IMPL-APL-2 为访问策略增加显式加载提示
- [x] TEST-APL-1 更新访问策略页面测试
- [x] REVIEW-APL-1 完成 3.3 代码复核
- [x] ARCHIVE-APL-1 归档到 `docs/change`

#### Task Details
##### DOC-APL-1 - 稳定文档更新
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading\plan.md`
- Goal:
  - 把访问策略卡片级保存入口和加载提示写入 requirements/specs
- Files / Modules:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Write Set:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Acceptance:
  - 稳定文档明确说明卡片级保存入口仍复用统一 policy 保存链路
- Test Points:
  - 文档自检
- Rollback:
  - 回退 requirements/specs 修改

##### IMPL-APL-1 - 访问策略卡片级保存入口
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading\plan.md`
- Goal:
  - 在默认准入、节点覆盖、角色管理等主要编辑卡片附近提供保存按钮
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/i18n/messages/operations.ts`
- Write Set:
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/i18n/messages/operations.ts`
- Acceptance:
  - 页面不再只有一个保存按钮
  - 所有新增按钮都调用同一保存链路
- Test Points:
  - `npm test -- AccessPolicy`
  - `npm run build`
- Rollback:
  - 回退卡片动作区新增按钮与文案

##### IMPL-APL-2 - 访问策略加载提示
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading\plan.md`
- Goal:
  - 为 policy 加载过程增加页面内显式提示
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/i18n/messages/operations.ts`
- Write Set:
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/i18n/messages/operations.ts`
- Acceptance:
  - `accessPolicyStore.state.loading` 为真时，页面能看到稳定提示
- Test Points:
  - `npm test -- AccessPolicy`
  - `npm run build`
- Rollback:
  - 回退 loading notice 与相关 busy 文案

##### TEST-APL-1 - 页面回归验证
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading\plan.md`
- Goal:
  - 锁定访问策略新增保存入口和加载提示
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.test.ts`
- Write Set:
  - `frontend/src/pages/AccessPolicy.test.ts`
- Acceptance:
  - 测试覆盖保存入口可见和 loading notice 可见
- Test Points:
  - `npm test -- AccessPolicy`
  - `npm run build`
- Rollback:
  - 回退新增断言

#### Dependencies
- `AccessPolicy.vue` 与 `accessPolicy.ts`、toast、authority store 高度耦合
- 页面保存行为依赖既有 `saveOptions`
- loading notice 依赖 `accessPolicyStore.state.loading` 的现有语义

#### Risks and Notes
- 新增多个 `Save Policy` 按钮后，需避免用户误解为局部保存
- 角色管理页当前是列表页，不存在局部弹窗外的保存入口；需要在页头动作区补齐
- 旧 `plan.md` 来自主仓上一轮 workflow，本轮已按 worktree 新上下文重写

#### Parallelism Assessment
- 不派发子Agent
- 原因:
  - 这轮写集集中在 `AccessPolicy.vue`、i18n、测试和文档，重叠明显
  - 当前会话未获得显式子Agent委派授权
- Owner:
  - 主Agent

#### Issue List
- none

### Stage 3.2 - Implementation
#### Completed Work
- `DOC-APL-1`
  - 计划中的 requirements/specs 澄清已落实到稳定文档修改集合中
- `IMPL-APL-1`
  - `frontend/src/pages/AccessPolicy.vue` 在默认准入、节点覆盖、角色管理页头补充就近 `Save Policy` 入口
  - 所有新增入口继续调用同一条 `savePolicy(saveOptions)` 链路，没有引入局部保存协议
- `IMPL-APL-2`
  - `frontend/src/pages/AccessPolicy.vue` 基于既有 `accessPolicyStore.state.loading` 增加页面内加载提示
  - 统一保存 / 重载按钮 busy 文案，保持 `loading` 与 `saving` 状态下的禁用逻辑一致
- `TEST-APL-1`
  - `frontend/src/pages/AccessPolicy.test.ts` 覆盖当前策略 tab、角色管理 tab 的保存按钮可见性
  - 新增显式 loading notice 测试

#### Files Updated
- `frontend/src/pages/AccessPolicy.vue`
- `frontend/src/i18n/messages/operations.ts`
- `frontend/src/pages/AccessPolicy.test.ts`

#### Implementation Notes
- 未修改 `frontend/src/stores/accessPolicy.ts`
- 未修改任何后端 authority / policy 协议
- 未新增请求或新的异步状态字段

### Stage 3.3 - Review
#### Review Checklist
- 需求范围复核：
  - 仅覆盖访问策略页的保存入口与加载提示，没有扩大到注册审批或准入许可
- 契约复核：
  - `SavePolicy` / `LoadPolicy` 仍沿用既有 store 和后端接口
- UI 语义复核：
  - 新按钮是额外入口，不改变“保存整份 policy”的既有语义
- 回归复核：
  - 页面测试通过
  - 前端构建通过

#### Validation Results
- `npm test -- AccessPolicy`
  - passed
- `npm run build`
  - passed

#### Review Conclusion
- 通过，可进入归档

### Stage 4 - Archive
#### Archive Outputs
- `docs/change/2026-03-27_win-access-policy-save-loading.md`
- `docs/change/README.md`
- `docs/requirements/authority-admin-console.md`
- `docs/specs/authority-admin-console.md`

#### Lessons Decision
- `none`
- 原因：本轮主要是小范围交互增强，未形成需要单独检索的通用故障模式

#### Workflow Status
- 已完成本轮实现、验证和归档
- 等待用户决定是否结束当前 workflow

阻塞：否
workflow 状态：等待用户确认是否结束
