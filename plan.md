# Plan - Win Permit UI Refine

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-permit-ui-refine`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-ui-refine`
- Current Stage: `4 archived, awaiting workflow end decision`

## Stage Records

### Initialization
- `guide.md`:
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`
  - 已读取 `frontend-design` 技能说明；本轮只在既有 authority 页面语言内收敛准入许可页，不重做整体视觉体系
- base/worktree confirmation:
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `fix/win-permit-ui-refine`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-ui-refine`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 继续收敛 `Permit Issuance` 页面，修正与注册审批页类似的冗余解析入口、编辑框形态和结果区臃肿感，让页面更像“动作入口 + 紧凑结果”。

#### Scope
- 必须:
  - 保持准入许可页的 authority 解析、issue permit、revoke permit 和 latest permit 结果链路不变
  - 去掉页面中冗余的显式 `Resolve` 入口，避免与主动作并列抢占注意力
  - 优化 permit 编辑框形态，使其更符合输入内容本身
  - 收敛 latest permit 结果区，让信息展示更接近紧凑列表而不是块状大卡片
  - 页面测试需要覆盖新的动作入口与关键表单结构
- 可选:
  - 统一 issue / revoke dialog 的单列阅读节奏
  - 为最新 permit 结果补稳定测试定位点
- 不做:
  - 不修改 `IssueRegisterPermit` / `RevokeRegisterPermit` 后端契约
  - 不新增 permit history list/query 能力
  - 不改动注册审批或访问策略页面

#### Use Cases
- 管理员进入准入许可页后，希望先看到简洁的协议边界和动作入口，而不是先被一个单独的 `Resolve` 按钮打断
- 管理员签发 permit 时，希望在单列表单里顺序填写设备、角色和过期时间，而不是横向来回扫读
- 管理员撤销 permit 时，希望直接粘贴单行 token，而不是在一个过高的 textarea 里操作
- 管理员查看最近一次 permit 时，希望结果区是紧凑明细行，便于快速扫读和触发复制 / 撤销

#### Functional Requirements
- `Permit Issuance` 页面必须继续支持：
  - authority 自动解析
  - issue permit
  - revoke permit
  - latest permit 结果展示
- 页面不得再保留单独的显式 `Resolve` 按钮作为主操作
- issue / revoke 仍必须通过 focused dialogs 完成
- revoke dialog 的 token 输入必须继续允许粘贴完整 permit
- latest permit 区仍必须支持复制和直接送入 revoke 流程

#### Non-functional Requirements
- 优先做最小安全改动，不新增状态字段或请求
- 保持当前 authority 页面紧凑、克制的视觉语言
- 页面主视图应继续遵守“动作入口 + 最新结果”的既有设计方向
- 测试需覆盖新布局下的关键入口仍可用

#### Inputs / Outputs
- 输入:
  - 当前 session 身份 `sourceId / hubId`
  - authority store 当前 `authorityId`
  - permit issue 输入 `deviceId / role / expiresAt`
  - permit revoke 输入 `permit`
- 输出:
  - 更收敛的 permit 动作区和结果区
  - 保持不变的签发 / 撤销行为

#### Edge Cases
- authority 未解析但用户直接 issue / revoke
- latest permit 已被 revoked
- revoke dialog 打开时没有 latest permit，只能手动粘贴 token
- expiresAt 为空时仍需保持 authority default TTL 语义

#### Acceptance Criteria
- 准入许可页不再显示显式 `Resolve` 按钮
- issue dialog 改为更聚焦的单列输入节奏
- revoke token 输入不再使用大 textarea
- latest permit 结果区更紧凑，且复制 / 撤销路径不变
- `PermitIssuance` 测试和前端构建通过

#### Risks
- 若移除 `Resolve` 但没有保留清晰的 issue/revoke 主路径，用户可能误以为 authority 解析能力消失
- 若 latest permit 区压得过紧，长 token 的可读性可能下降

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 采用最小前端收敛方案：
  - 删除 `Permit Issuance` 顶部的 `resolveAuthorityAction` 和对应按钮
  - 保持 `permitStore.issuePermit()` / `revokePermit()` 现有自动 authority 解析链路不变
  - 将 issue dialog 改成单列输入布局
  - 将 revoke dialog 的 token 从 textarea 改成单行 input
  - 将 `Latest Permit` 卡内部明细改成紧凑明细行结构

#### Alternatives Considered
- 方案 A（采用）：页面局部收敛，移除冗余 `Resolve`，压缩结果区与表单结构
  - 优点：最贴近用户当前反馈，且不触碰 store / protocol
  - 代价：最新 permit 结果仍是卡片语境，只是内部更紧凑
- 方案 B：保留 `Resolve`，只平移到 header actions
  - 优点：改动更小
  - 代价：仍保留冗余动作，没有真正收敛页面心智
- 方案 C：进一步把 permit 页面拆 tab 或拆更多子卡片
  - 优点：层级更细
  - 代价：超过当前需求，风险和写集都偏大

#### Module Responsibilities
- `frontend/src/pages/PermitIssuance.vue`
  - 收敛页头动作区
  - 调整 issue / revoke dialog 表单结构
  - 调整 latest permit 结果区布局
- `frontend/src/stores/permitIssuance.ts`
  - 保持现有签发 / 撤销与 latest permit 状态语义不变
- `frontend/src/stores/authority.ts`
  - 保持 `requireAuthority()` 自动解析行为不变
- `frontend/src/pages/PermitIssuance.test.ts`
  - 覆盖移除 `Resolve`、revoke input 结构和关键动作路径

#### Data / Call Flow
1. 页面 ready 后继续只建立本地 authority 上下文，不主动发 permit 请求
2. 用户从 `Permit Actions` 打开 issue / revoke dialog
3. `issuePermit()` / `revokePermit()` 内部继续通过 store 调用 `requireAuthority()`
4. latest permit 结果继续由 store 的 `lastIssued` / `lastRevoke` 驱动

#### Interface Drafts
- 顶部摘要卡:
  - 保留连接状态、登录状态、identity、authority badges
  - 保留协议边界说明
  - 移除单独 `Resolve`
- `Permit Actions`:
  - 保留 `Issue Permit` / `Revoke Permit` 两行紧凑动作
- `Issue Permit` dialog:
  - `Device ID`
  - `Role`
  - `Expires At (optional)`
  - 单列顺序布局
- `Revoke Permit` dialog:
  - 单行 token input
- `Latest Permit`:
  - token 展示
  - device / role / expires / issued 明细行
  - copy / revoke actions

#### Error Handling and Safety
- 不改变 issue / revoke 的既有本地校验和 toast 行为
- 不改变 `latestIssued.revoked` 与 `lastRevoke` 的回写逻辑
- revoke 仍需显式按钮触发

#### Performance and Testing Strategy
- 不增加任何新的 authority 请求
- 验证重点:
  - `frontend/src/pages/PermitIssuance.test.ts`
  - `npm test -- PermitIssuance`
  - `npm run build`
- fresh worktree 如缺 `node_modules` 或 `wailsjs`，按既有预热流程补齐后再验证

#### Extensibility Design Points
- 若后续 permit 支持历史记录，当前紧凑结果区可以继续作为“latest snapshot”，旁边再挂独立列表
- 若其它 authority 页面也要消除冗余 `Resolve`，本轮结构可作为直接参考

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 让准入许可页继续向“动作入口 + 聚焦表单 + 紧凑结果”收敛
- Current State:
  - 顶部协议边界区仍带独立 `Resolve` 按钮
  - issue dialog 仍使用双列表单
  - revoke dialog 仍使用较重的 textarea
  - latest permit 结果区仍偏块状

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Requirements impact: `none`
- Specs impact: `none`
- Stable docs destination:
  - none
- Change archive destination:
  - `docs/change/2026-03-27_win-permit-ui-refine.md`
- Lessons impact:
  - none

#### Related Requirements / Specs / Lessons
- Requirements:
  - `D:\project\MyFlowHub3\worktrees\fix-win-permit-ui-refine\docs\requirements\authority-admin-console.md`
- Specs:
  - `D:\project\MyFlowHub3\worktrees\fix-win-permit-ui-refine\docs\specs\authority-admin-console.md`
- Lessons:
  - `D:\project\MyFlowHub3\worktrees\fix-win-permit-ui-refine\docs\lessons\README.md`

#### Executable Task List
- [x] IMPL-WPR-1 收敛准入许可页动作与表单布局
- [x] TEST-WPR-1 更新准入许可页面测试
- [x] REVIEW-WPR-1 完成 3.3 代码复核
- [x] ARCHIVE-WPR-1 归档到 `docs/change`

#### Task Details
##### IMPL-WPR-1 - 收敛准入许可页动作与表单布局
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-ui-refine`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-permit-ui-refine\plan.md`
- Goal:
  - 移除冗余 `Resolve`，优化 issue/revoke 编辑框形态，并压缩 latest permit 结果区
- Files / Modules:
  - `frontend/src/pages/PermitIssuance.vue`
- Write Set:
  - `frontend/src/pages/PermitIssuance.vue`
- Acceptance:
  - 页面不再显示 `Resolve`
  - issue/revoke dialog 更聚焦
  - latest permit 更紧凑
- Test Points:
  - `npm test -- PermitIssuance`
  - `npm run build`
- Rollback:
  - 回退 permit 页面布局和 dialog 结构修改

##### TEST-WPR-1 - 准入许可页回归验证
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-ui-refine`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-permit-ui-refine\plan.md`
- Goal:
  - 锁定新 permit 页面结构和关键动作路径
- Files / Modules:
  - `frontend/src/pages/PermitIssuance.test.ts`
- Write Set:
  - `frontend/src/pages/PermitIssuance.test.ts`
- Acceptance:
  - 测试覆盖不再渲染 `Resolve`
  - 测试覆盖 revoke token 使用单行 input
  - issue / revoke 关键路径继续通过
- Test Points:
  - `npm test -- PermitIssuance`
  - `npm run build`
- Rollback:
  - 回退新增断言

#### Dependencies
- `PermitIssuance.vue` 与 `permitIssuance.ts`、`authority.ts`、toast 高度耦合
- issue / revoke 主路径依赖 `requireAuthority()` 自动解析链路

#### Risks and Notes
- 本轮只做 permit 页面局部收敛，不修改任何 authority 服务契约

#### Parallelism Assessment
- 不派发子Agent
- 原因:
  - 写集集中在单个页面和其页面测试，规模小且高度耦合
  - 当前会话未获得显式子Agent委派授权
- Owner:
  - 主Agent

#### Issue List
- none

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Task Mapping
- `IMPL-WPR-1`
  - `frontend/src/pages/PermitIssuance.vue`
- `TEST-WPR-1`
  - `frontend/src/pages/PermitIssuance.test.ts`

#### File-level Change Summary
- `frontend/src/pages/PermitIssuance.vue`
  - 删除显式 `resolveAuthorityAction` 和页头 `Resolve` 按钮，保留协议边界说明。
  - 将 `Issue Permit` dialog 从双列输入改为单列顺序输入。
  - 将 `Revoke Permit` 的 token 编辑框从 `textarea` 改为单行 `input`。
  - 将 `Latest Permit` 明细压缩为紧凑行列表，并补 `data-latest-permit-details` 供测试定位。
- `frontend/src/pages/PermitIssuance.test.ts`
  - 补充不再渲染 `Resolve` 文案断言。
  - 补充撤销输入框为 `INPUT` 的结构断言。
  - 补充 latest permit 紧凑明细区存在断言。

#### Design Notes
- `issuePermit()` / `revokePermit()` 继续经 `permitStore` 调用 `authority.requireAuthority()` 自动解析 authority，不新增额外请求。
- 继续保持“动作入口 + focused dialog + latest snapshot”结构，只收敛视觉密度，不改 store / backend 契约。

#### Validation
- `npm ci`
  - 通过
- `npm test -- PermitIssuance`
  - 通过
- `npm run build`
  - 首次失败，原因是 fresh worktree 缺少 `frontend/wailsjs/**`
- `$env:GOWORK='off'; wails generate module`
  - 通过
- `npm run build`
  - 通过

#### Blockers
- none

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 已覆盖去除冗余 `Resolve`、单列 issue 表单、单行 revoke 输入和更紧凑的 latest permit 明细。
- 架构合理性：通过
  - 仅修改页面层和页面测试，不触碰 authority / permit store 契约。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅新增本地 `computed` 明细数组；未增加网络请求。
- 可读性与一致性：通过
  - 延续 authority 页面现有 `CardHeader + compact rows + overlay dialog` 结构。
- 可扩展性与配置化：通过
  - 明细行抽成 `latestPermitDetails`，后续若扩展字段可继续顺序追加。
- 稳定性与安全：通过
  - 保持既有输入校验、toast 和危险操作触发路径不变。
- 测试覆盖情况：通过
  - 页面结构变化已由 `PermitIssuance.test.ts` 锁定，并完成前端构建验证。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未使用子Agent。

### Stage 4 - Change Archive
#### $m-docs Check
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`
- `docs/README.md` 无需更新。
- 需更新：
  - `docs/change/2026-03-27_win-permit-ui-refine.md`
  - `docs/change/README.md`

#### Archive Status
- 已归档到 `docs/change/2026-03-27_win-permit-ui-refine.md`
- 等待用户确认是否结束 workflow
