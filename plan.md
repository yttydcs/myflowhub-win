# Plan - Win Approval Permit UI Refine

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `refactor/win-approval-permit-ui`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui`
- Current Stage: `4 archive`

## Stage Records

### Initialization
- `guide.md`:
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `frontend-design` 技能说明，用于在既有产品语言内继续收敛 authority 页面层级
- base/worktree confirmation:
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `refactor/win-approval-permit-ui`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 继续收敛 `Registration Approvals` 与 `Permit Issuance` 的页面密度和操作路径，使它们与当前 `Access Policy` 一样具备“紧凑列表 / 摘要 + 聚焦编辑”的交互节奏，而不改变现有 authority 编排能力。

#### Scope
- 必须:
  - 保留 `Registration Approvals` 与 `Permit Issuance` 的现有路由、authority 解析和 Wails 接口契约
  - `Registration Approvals` 页面必须从“每条请求展开两块 approve/reject 表单”收敛为紧凑请求列表
  - 审批单条请求时，详细输入应集中到单一聚焦编辑面，而不是在列表内同时展示 approve / reject 两组输入区
  - 请求列表应优先保持单行或低高度摘要节奏，至少展示：
    - `deviceId`
    - `requestId`
    - `requestedRole`
    - `displayName`
    - 时间摘要
    - 主操作入口
  - `Permit Issuance` 页面必须从持续展开的 `Issue / Revoke` 大表单收敛为更轻的摘要与聚焦操作结构
  - permit 签发与撤销的详细输入应集中到单一操作面，而不是长时间占据整页首屏
  - 最近一次签发结果仍需可见，并继续支持复制与撤销
  - 页面需要继续明确“无历史 permit 列表”的协议边界
  - 保持中英文 i18n 一致
  - 前端测试需要覆盖新的结构和关键交互
- 可选:
  - 为审批列表补充更好扫读的状态摘要或剩余有效期信息
  - 为 permit 页面补充更轻量的动作摘要文案，减少重复协议说明
- 不做:
  - 不修改 `MyFlowHub-Server` auth 协议
  - 不新增 permit history list / query 能力
  - 不改动 Access Policy 页面本轮已经稳定的交互
  - 不引入新的后端持久化或本地缓存

#### Use Cases
- 管理员进入注册审批页后，先浏览一个紧凑的待处理请求列表，再点开单条请求做 approve 或 reject 决策，而不是被整页重复表单淹没
- 管理员查看某条 pending request 时，可以在一个聚焦对话面里同时看到摘要、可选 role override 和 rejection reason
- 管理员进入 permit 页面后，先看到当前 authority 上下文、协议边界和最近一次签发结果，再按需进入 issue 或 revoke 动作
- 管理员签发 permit 时，只在需要时打开输入面填写 `deviceId / role / expiresAt`
- 管理员想撤销 permit 时，可以从最近一次签发结果直接执行，也可以通过单独的 revoke 输入面提交任意 token

#### Functional Requirements
- `Registration Approvals` 页面必须继续支持：
  - authority 自动解析
  - pending list 读取与刷新
  - 可选 `deviceId` 过滤
  - approve 单条 request
  - reject 单条 request
  - 最近一次决策结果反馈
- `Registration Approvals` 的列表主视图不得继续为每条 request 展开 approve / reject 双卡片表单
- `Registration Approvals` 的单条 request 详细编辑必须在一个聚焦编辑面中完成，允许 role 为空、reason 为空
- `Permit Issuance` 页面必须继续支持：
  - authority 自动解析
  - issue permit
  - revoke permit
  - 最近一次签发结果展示
  - 复制最新 permit
- `Permit Issuance` 的主视图不得继续长期展开大块 issue / revoke 表单
- 最近一次签发结果必须保留：
  - `permit`
  - `deviceId`
  - `role`
  - `expiresAt`
  - `issuedAt`
  - `revoked` 状态

#### Non-functional Requirements
- 优先做前端层级收敛，不扩大到新协议或新 service
- 列表与摘要区应延续 `Access Policy` 已稳定的轻量视觉节奏，避免重新出现厚重双栏或多块同层卡片
- 聚焦编辑面应保证键盘关闭、遮罩关闭和焦点恢复稳定
- 不增加额外 authority 请求；审批后仍只回刷 pending list，permit 仍只更新本页相关状态
- 测试需要优先覆盖列表化后仍能正确执行动作的关键路径

#### Inputs / Outputs
- 输入:
  - 当前 session 身份 `sourceId / hubId`
  - resolved `authorityId`
  - pending request 列表
  - approve 可选 `role`
  - reject 可选 `reason`
  - permit issue 输入 `deviceId / role / expiresAt`
  - permit revoke 输入 `permit`
- 输出:
  - 紧凑 pending request 列表
  - 单条 request 的聚焦审批 / 拒绝编辑面
  - permit 摘要页
  - issue / revoke 聚焦操作面
  - 最新决策反馈和最新 permit 结果

#### Edge Cases
- authority 未解析
- pending 列表为空
- `requestedRole` / `displayName` 为空
- approve 的 role 为空字符串
- reject 的 reason 为空字符串
- permit issue 时缺少 `deviceId` 或 `role`
- permit revoke 时 token 为空
- 最新 permit 已被撤销
- 身份切换后页面旧数据未清空

#### Acceptance Criteria
- 注册审批页不再为每条 request 展开 approve / reject 双表单，列表高度明显下降
- 注册审批页可以从紧凑列表进入聚焦审批面，并完成 approve / reject 动作
- permit 页首屏不再长期展开两块大表单，页面更接近“摘要 + 动作入口 + 最近结果”
- permit issue / revoke 仍可完整执行，最近签发结果继续支持复制与撤销
- 不改变现有 authority service 契约
- 前端测试与构建通过

#### Risks
- 若审批聚焦面设计过度，可能导致批量处理效率下降，需要在“紧凑”与“少点一步”之间平衡
- 若 permit 动作入口收得过深，可能让常用 issue/revoke 变得不够直接
- 新增 overlay 或对话面后，需要确认 Esc / backdrop / toast 行为不互相干扰

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 保持 `registrationApprovals.ts`、`permitIssuance.ts` 和 authority store 契约不变，主要重构 `RegistrationApprovals.vue` 与 `PermitIssuance.vue` 的页面结构：
  - `Registration Approvals`:
    - 顶部保留 identity / authority 摘要与刷新动作
    - pending queue 改为紧凑列表
    - 单条 request 点击 `Review` 后打开聚焦编辑 dialog，在一个面里完成 approve / reject
  - `Permit Issuance`:
    - 顶部保留 identity / authority / protocol boundary 摘要
    - 主视图改为动作摘要 + 最新 permit 结果
    - issue permit / revoke permit 改为各自聚焦 dialog 或单一聚焦操作面
- 稳定文档同步补充 `approval / permit` 页面的紧凑列表和聚焦编辑约束

#### Alternatives Considered
- 方案 A（采用）：紧凑列表 + 聚焦 dialog / 操作面
  - 优点：最贴近用户已经确认过的 `Access Policy` 收敛方向，首屏密度最低
  - 代价：需要新增局部 dialog 状态和页面测试
- 方案 B：只保留当前页面结构，缩小卡片和输入区尺寸
  - 优点：改动最小
  - 代价：本质上还是大块表单，不能解决“页面臃肿”
- 方案 C：把审批和 permit 完全改成右侧侧栏编辑
  - 优点：可减少遮罩层
  - 代价：当前项目没有稳定复用的侧栏编辑模式，且容易再次占满页面纵向空间

#### Module Responsibilities
- `frontend/src/pages/RegistrationApprovals.vue`
  - 负责 pending 列表、审批聚焦面、最近决策反馈和 authority 上下文展示
- `frontend/src/pages/PermitIssuance.vue`
  - 负责 permit 摘要、issue/revoke 聚焦面、最新 permit 结果展示和操作
- `frontend/src/stores/registrationApprovals.ts`
  - 保持 pending list / approve / reject 状态与回刷逻辑
- `frontend/src/stores/permitIssuance.ts`
  - 保持 latest permit / revoke 状态与回写逻辑
- `frontend/src/i18n/messages/operations.ts`
  - 补充新的紧凑列表、dialog 和动作反馈文案
- `docs/requirements/authority-admin-console.md`
  - 记录审批页和 permit 页的交互收敛目标
- `docs/specs/authority-admin-console.md`
  - 记录审批页和 permit 页的稳定 UI 约束与状态模型

#### Data / Call Flow
1. authority 页面继续基于共享 authority store 自动解析身份
2. `Registration Approvals` 初次进入后加载 pending list
3. 用户从列表点开单条 request，页面将摘要和草稿状态映射到审批 dialog
4. dialog 内执行 approve 或 reject 后，继续调用 store，并在成功后回刷 pending list
5. `Permit Issuance` 页面继续只维护本地 `issueForm / revokeForm / lastIssued / lastRevoke`
6. 用户从摘要页点开 issue 或 revoke 操作面，提交后调用既有 store
7. issue 成功后更新 latest permit；revoke 成功后回写 revoked 状态

#### Interface Drafts
- `Registration Approvals`
  - 顶部摘要卡 / badge 保留
  - `Pending Queue` 改为紧凑列表：
    - `deviceId`
    - `requestId`
    - `requestedRole / displayName / time summary`
    - `Review` 按钮
  - 新增 `requestReviewDialog`
    - `open`
    - `requestId`
    - `role`
    - `reason`
- `Permit Issuance`
  - 顶部协议边界说明保持轻量摘要
  - 动作区只保留 `Issue Permit` / `Revoke Permit` 入口
  - 新增：
    - `issuePermitDialog`
    - `revokePermitDialog`
  - `Latest Permit` 保持结果摘要卡，但进一步压缩成更清晰的信息行

#### Error Handling and Safety
- 继续沿用 `ensureReady` 的本地身份校验
- approve 时继续允许 role 为空
- reject 时继续允许 reason 为空
- issue / revoke dialog 提交前继续校验必填项
- dialog 关闭不得隐式提交
- 身份变化时，审批和 permit 的临时 dialog 状态必须随 store reset 清空

#### Performance and Testing Strategy
- 所有新交互均为前端局部状态，不增加额外 authority 请求
- 验证重点:
  - `frontend/src/pages/RegistrationApprovals.test.ts`（新增）
  - `frontend/src/pages/PermitIssuance.test.ts`（新增）
  - `frontend/src/stores/authority_admin.test.ts`
  - `npm test -- RegistrationApprovals PermitIssuance authority_admin`
  - `npm run build`
- 若本地 dev host 可用，补一次 `chrome-devtools` smoke check 验证页面密度和 dialog 流程

#### Extensibility Design Points
- 若后续 pending request 需要更多动作，`Registration Approvals` 的 review dialog 可以继续承载而不必扩回列表内表单
- 若未来 permit 支持 history list，摘要页可在不破坏 issue/revoke dialog 的前提下追加独立列表区
- 页面列表行和摘要节奏后续可提炼为共享 authority row pattern，本轮先保持最小写集

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 将 `Registration Approvals` 与 `Permit Issuance` 收敛为与 `Access Policy` 一致的轻量、聚焦、低密度 authority 页面
- Current State:
  - 注册审批页对每条 request 同时展开 approve / reject 双卡片，列表高度过高
  - permit 页将 issue / revoke 两块表单长期铺在首屏，占用过多注意力
  - 两个页面目前都还没有形成“先看摘要，再进入单次操作”的路径

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Requirements impact: `clarify`
- Specs impact: `clarify`
- Stable docs destination:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Change archive destination:
  - `docs/change/2026-03-27_win-approval-permit-ui.md`
- Lessons impact:
  - none（当前属于 authority 页面交互收敛，不是排障型复盘）

#### Related Requirements / Specs / Lessons
- Requirements:
  - `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui\docs\requirements\authority-admin-console.md`
- Specs:
  - `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui\docs\specs\authority-admin-console.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- Lessons:
  - `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui\docs\lessons\README.md`

#### Executable Task List
- [x] DOC-API-1 更新 authority admin console 的 requirements/specs
- [x] IMPL-API-1 收敛注册审批页为紧凑列表 + 聚焦审批面
- [x] IMPL-API-2 收敛准入许可页为摘要 + 聚焦动作面
- [x] TEST-API-1 补页面测试并更新 authority admin store 回归验证
- [x] REVIEW-API-1 完成 3.3 代码复核
- [x] ARCHIVE-API-1 归档到 `docs/change`

#### Task Details
##### DOC-API-1 - 稳定文档更新
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui\plan.md`
- Goal:
  - 把审批页和 permit 页的“紧凑列表 / 摘要 + 聚焦编辑”约束写入 requirements/specs
- Files / Modules:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Write Set:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Acceptance:
  - 稳定文档不再把审批和 permit 页面默认描述为大块持久展开表单
- Test Points:
  - 文档自检
- Rollback:
  - 回退 requirements/specs 修改

##### IMPL-API-1 - 注册审批页收敛
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui\plan.md`
- Goal:
  - 把 pending queue 改成紧凑列表，并通过单一聚焦面完成 approve / reject
- Files / Modules:
  - `frontend/src/pages/RegistrationApprovals.vue`
  - `frontend/src/i18n/messages/operations.ts`
- Write Set:
  - `frontend/src/pages/RegistrationApprovals.vue`
  - `frontend/src/i18n/messages/operations.ts`
- Acceptance:
  - 列表高度显著下降
  - 单条 request 不再内联展开双表单
  - approve / reject 仍完整可用
- Test Points:
  - `npm test -- RegistrationApprovals`
  - `npm run build`
- Rollback:
  - 回退审批页列表结构和 dialog 状态

##### IMPL-API-2 - 准入许可页收敛
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui\plan.md`
- Goal:
  - 把 permit 页改成摘要 + 聚焦操作路径，保留 latest permit 的高频动作
- Files / Modules:
  - `frontend/src/pages/PermitIssuance.vue`
  - `frontend/src/i18n/messages/operations.ts`
- Write Set:
  - `frontend/src/pages/PermitIssuance.vue`
  - `frontend/src/i18n/messages/operations.ts`
- Acceptance:
  - issue / revoke 不再长期占据首屏大块表单
  - latest permit 继续可见并支持 copy / revoke
- Test Points:
  - `npm test -- PermitIssuance`
  - `npm run build`
- Rollback:
  - 回退 permit 页结构和 dialog 状态

##### TEST-API-1 - 前端回归验证
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui`
- Plan Path: `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui\plan.md`
- Goal:
  - 锁定审批页和 permit 页的新结构与关键动作路径
- Files / Modules:
  - `frontend/src/pages/RegistrationApprovals.test.ts`
  - `frontend/src/pages/PermitIssuance.test.ts`
  - `frontend/src/stores/authority_admin.test.ts`
- Write Set:
  - `frontend/src/pages/RegistrationApprovals.test.ts`
  - `frontend/src/pages/PermitIssuance.test.ts`
  - `frontend/src/stores/authority_admin.test.ts`
- Acceptance:
  - 测试覆盖紧凑列表、dialog 打开、approve/reject、issue/revoke、latest permit 动作
- Test Points:
  - `npm test -- RegistrationApprovals PermitIssuance authority_admin`
  - `npm run build`
- Rollback:
  - 回退新增页面测试和相关断言

#### Dependencies
- `RegistrationApprovals.vue` 与 `registrationApprovals.ts`、toast、authority store 高度耦合
- `PermitIssuance.vue` 与 `permitIssuance.ts`、clipboard 行为和 toast 文案高度耦合
- dialog 交互依赖现有 `Overlay` 组件的焦点管理与关闭行为
- 列表节奏会参考 `AccessPolicy.vue` / `Flow.vue` 已经稳定的紧凑行样式

#### Risks and Notes
- 审批页若把过多信息折叠到 dialog，可能影响批量扫读效率
- permit 页若过度隐藏 revoke 入口，可能降低操作可发现性
- 当前 worktree 基于 `main@99cad7d` 创建，已包含 Access Policy 角色列表样式基线的 spec 固化提交

#### Parallelism Assessment
- 不派发子Agent
- 原因:
  - 两个页面都会同时改动页面结构、i18n 文案和测试，写集重叠明显
  - 当前会话未获得显式子Agent委派授权
- Owner:
  - 主Agent

#### Issue List
- none

### Stage 3.2 - Implementation
#### Execution Summary
- `IMPL-API-1`
  - `RegistrationApprovals.vue` 已改为紧凑待办队列；approve / reject 从列表内联表单收敛到单一 review dialog。
- `IMPL-API-2`
  - `PermitIssuance.vue` 已改为 `Permit Actions` 动作列表 + `Latest Permit` 结果卡；issue / revoke 改为 focused dialogs。
- `DOC-API-1`
  - `operations.ts` 已补齐新文案；requirements/specs 已同步稳定约束。
- `TEST-API-1`
  - 已新增 `RegistrationApprovals.test.ts`、`PermitIssuance.test.ts`。

#### Validation
- `npm test -- RegistrationApprovals PermitIssuance authority_admin`
  - 通过
- 首次 `npm run build`
  - 失败，原因是 fresh worktree 缺少 `frontend/wailsjs/**`
- `$env:GOWORK='off'; wails generate module`
  - 通过
- 再次 `npm run build`
  - 通过

#### Issue List
- none

### Stage 3.3 - Review
#### Review Checklist
- 需求 / 规格是否同步
  - 是，`docs/requirements/authority-admin-console.md` 与 `docs/specs/authority-admin-console.md` 已更新
- 页面是否回退到首屏大表单
  - 否，approvals 与 permit 均已改为紧凑列表 / 摘要 + focused dialogs
- 关键交互是否有测试
  - 是，新增 approvals / permit 页面测试并复用 authority store 回归测试
- 是否发现新增问题
  - 未发现

#### Findings
- none

#### Issue List
- none

### Stage 4 - Archive
#### Archive Outputs
- `docs/change/2026-03-27_win-approval-permit-ui.md`
  - 已创建
- `docs/change/README.md`
  - 已更新索引

#### Lessons Decision
- `Lessons impact: none`
- 原因：
  - 本轮主要是 authority 页面交互收敛，没有形成新的可复用排障模型；fresh worktree 的 `wails generate module` 预检已在既有 README / change 中覆盖

#### Ready For Workflow End
- 是
- 后续如用户确认结束 workflow，可执行合并 / 清理 worktree

阻塞：否
进入 3.2
