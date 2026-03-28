# Plan - Win 准入许可远程 authority 超时收敛

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-permit-list-timeout-route`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-list-timeout-route`
- Current Stage: `4 archive complete, awaiting workflow end decision`

## Stage Records

### Initialization
- `guide.md`
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `$m-docs` 的 requirement impact 和 indexing 规则入口
- base/worktree confirmation
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `fix/win-permit-list-timeout-route`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-list-timeout-route`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 收敛 `Permit Issuance` 页在远程 authority 场景下的真实超时，把“当前后端不支持远程 authority permit 管理”从模糊 timeout 改成前端可理解、可预期的限制提示。

#### Scope
- 必须
  - 识别 `sourceId != authorityId` 的远程 authority 场景
  - permit 页在该场景下不再自动触发 `list_register_permits` 并等待超时
  - 页面需要给出明确的本地 authority 限制提示，并禁用不成立的 permit 管理动作
  - Go orchestration 层对 permit 管理动作提前返回可读错误，避免继续等待 auth timeout
  - 补齐前端和 Go 回归测试
- 可选
  - 让相同限制文案可复用于其它 authority admin 页
- 不做
  - 不扩到 `MyFlowHub-Server` / `MyFlowHub-SubProto` 的远程 authority 分布式审批链路
  - 不修改 auth wire / protocol
  - 不改访问策略页

#### Use Cases
- 管理员在非 authority 节点打开“准入许可”页时，希望直接知道“当前节点不能远程管理 authority permit”，而不是只看到 `request timed out`
- 管理员在 authority 本机打开页面时，permit 列表、签发和撤销继续正常工作
- 后续排查时，可以从错误信息直接区分“真实 authority-local 限制”和“普通网络抖动”

#### Functional Requirements
- permit 页必须在 authority 已解析且 `sourceId != authorityId` 时显示显式受限提示
- permit 页在受限状态下不得自动加载 permit 列表
- permit 页在受限状态下不得允许 `Refresh` / `New Permit`
- `PermissionService` 的 permit 管理动作在远程 authority 场景下必须快速失败，并返回包含 source/authority 信息的可读错误
- authority 本地场景下 permit 页现有成功路径不得回退

#### Non-functional Requirements
- 以最小安全改动收敛，不扩展跨仓协议范围
- 错误提示应足够明确，便于用户判断需要切换到 authority 节点操作
- 页面失败态应保持稳定，不因受限场景清空或抖动现有列表

#### Inputs / Outputs
- 输入
  - 当前 session `sourceId` / `hubId`
  - authority store 的 `authorityId`
  - permit 管理动作名
- 输出
  - permit 页的受限提示与禁用状态
  - Go service 的快速失败错误

#### Edge Cases
- authority 尚未解析时不能误判为受限
- authority 本地场景下仍需正常自动加载
- permit 页身份切换后需要重新计算受限状态
- 将来后端补齐远程 authority 管理链路时，当前 guard 需要可回滚

#### Acceptance Criteria
- 在 `sourceId != authorityId` 时，permit 页不再出现 `auth list_register_permits: request timed out`
- 页面能明确提示“当前需要在 authority 节点本地操作”
- authority 本地场景下现有 permit 列表、签发、撤销路径不变
- `go test ./internal/services/permission/...` 和 `npm test -- PermitIssuance` 通过

#### Risks
- 若后端某些拓扑实际上已经可以远程执行 permit 管理，本轮 guard 会比真实能力更保守
- 仅修 permit 页的话，注册审批页后续仍可能暴露同类限制

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 方案 A（采用）
  - 在 Win `PermissionService` 中为 permit 管理动作增加 `authority-local` 前置校验
  - permit 页基于 authority store 的 `sourceId/authorityId` 计算 `remoteAuthorityBlocked`
  - 当受限时，permit 页跳过自动加载并展示稳定提示，同时禁用刷新和新建
  - 继续保留 authority 本地场景下的原有加载和动作链路
- 不采用方案
  - 直接扩到 Server/SubProto 实现远程 authority permit 管理
  - 理由：涉及跨仓 runtime 行为，明显超出本轮 Win 缺陷修复范围

#### Module Responsibilities
- `internal/services/permission/service.go`
  - 收敛 permit 管理动作的 authority-local 前置校验
- `internal/services/permission/service_test.go`
  - 覆盖远程 authority permit 管理快速失败
- `frontend/src/pages/PermitIssuance.vue`
  - 展示 remote authority 受限提示
  - 跳过受限场景自动加载
  - 禁用不成立的动作按钮
- `frontend/src/pages/PermitIssuance.test.ts`
  - 覆盖 remote authority 受限提示和不自动加载
- `docs/requirements/authority-admin-console.md`
  - 澄清当前 backend 能力下 permit/admin 页的远程 authority 限制提示要求
- `docs/specs/authority-admin-console.md`
  - 澄清 authority-local guard 是当前长期 UI/服务契约的一部分

#### Data / Call Flow
1. 页面基于 session 身份设置 authority store `sourceId/hubId`
2. authority 解析完成后，若 `authorityId != sourceId`，页面进入 `remoteAuthorityBlocked`
3. 受限状态下不触发 `loadPermits()`，只展示限制提示
4. 若前端或其它调用者仍直接触发 permit 管理动作，`PermissionService` 立即返回显式错误
5. authority 本地场景继续沿用当前 permit store 请求链路

#### Error Handling and Safety
- permit 页受限提示优先于 timeout
- authority-local guard 返回的错误应包含动作名、当前 sourceId 和 authorityId
- 不改变 authority 本地场景的业务返回码和 toast 行为

#### Performance and Testing Strategy
- 不增加新的网络请求
- 验证重点
  - `$env:GOWORK='off'; go test ./internal/services/permission/... -count=1`
  - `npm test -- PermitIssuance`
  - `npm run build`

#### Extensibility Design Points
- 若后续后端补齐远程 authority 管理链路，可删除 service guard 和页面提示，回到真实远程加载
- 同一 guard 模式可复用到 `Registration Approvals`

#### Issue List
- none

### Stage 3.1 - Planning
#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Requirements impact: `clarify`
- Specs impact: `clarify`
- Related requirements
  - `D:\project\MyFlowHub3\worktrees\fix-win-permit-list-timeout-route\docs\requirements\authority-admin-console.md`
- Related specs
  - `D:\project\MyFlowHub3\worktrees\fix-win-permit-list-timeout-route\docs\specs\authority-admin-console.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- Related lessons
  - `none`

#### Executable Task List
- [x] `PERMIT-GUARD-1` 在 Go orchestration 层增加 permit authority-local guard
- [x] `PERMIT-GUARD-2` 在 permit 页增加 remote authority 受限提示与按钮禁用
- [x] `DOC-CLARIFY-1` 更新 requirements/specs，明确当前远程 authority 限制提示要求
- [x] `VALIDATE-1` 补测试并执行 Go / 前端验证
- [x] `REVIEW-1` 完成 3.3 checklist
- [x] `ARCHIVE-1` 归档到 `docs/change`

#### Task Details
##### `PERMIT-GUARD-1` - Go authority-local guard
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-list-timeout-route`
- Goal
  - permit 管理动作在远程 authority 场景快速失败为可读错误
- Files
  - `internal/services/permission/service.go`
  - `internal/services/permission/service_test.go`
- Acceptance
  - `ListRegisterPermits` 至少不再等到 auth timeout
- Tests
  - `$env:GOWORK='off'; go test ./internal/services/permission/... -count=1`
- Rollback
  - 回退 authority-local guard 和对应测试

##### `PERMIT-GUARD-2` - Permit page remote authority UX
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-list-timeout-route`
- Goal
  - permit 页在 remote authority 下给出稳定限制提示，而不是继续自动加载
- Files
  - `frontend/src/pages/PermitIssuance.vue`
  - `frontend/src/pages/PermitIssuance.test.ts`
- Acceptance
  - 受限状态不自动调用 `loadPermits`
  - 页面展示可理解提示
  - 刷新和新建按钮禁用
- Tests
  - `npm test -- PermitIssuance`
- Rollback
  - 回退 permit 页面受限态逻辑

##### `DOC-CLARIFY-1` - Requirements/Specs clarify
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-list-timeout-route`
- Goal
  - 让 Win 稳定文档与当前 backend 能力保持一致
- Files
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Acceptance
  - 文档明确 permit/admin 页在 remote authority 下需给出 authority-local 限制提示
- Tests
  - doc consistency by manual review
- Rollback
  - 回退文档澄清

#### Dependencies
- permit 页依赖 authority store 的 `sourceId/authorityId`
- Win 文档澄清依赖 Server auth spec 中“审批列表/permit 仍建议从 authority 节点操作”的现有事实

#### Risks and Notes
- 本轮不承诺“remote authority permit 管理真实可用”，只收敛为显式限制提示

#### Parallelism Assessment
- 不派发子Agent
- 原因
  - 写集小，Go service、Vue 页面、测试和 docs 高度耦合
  - 当前会话未获得显式子Agent授权

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Task Mapping
- `PERMIT-GUARD-1`
  - `internal/services/permission/service.go`
  - `internal/services/permission/service_test.go`
- `PERMIT-GUARD-2`
  - `frontend/src/pages/PermitIssuance.vue`
  - `frontend/src/pages/PermitIssuance.test.ts`
- `DOC-CLARIFY-1`
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`

#### File-level Change Summary
- `internal/services/permission/service.go`
  - 为 permit 管理动作增加 authority-local 前置校验
- `internal/services/permission/service_test.go`
  - 覆盖 remote authority 快速失败
- `frontend/src/pages/PermitIssuance.vue`
  - remote authority 下跳过自动加载
  - 展示 authority-local 受限提示
  - 禁用 `Refresh / New Permit`
- `frontend/src/pages/PermitIssuance.test.ts`
  - 补 remote authority 受限态回归
- `frontend/src/i18n/messages/operations.ts`
  - 新增 permit remote authority 提示文案
- `docs/requirements/authority-admin-console.md`
  - 澄清 permit 页 remote authority 限制提示要求
- `docs/specs/authority-admin-console.md`
  - 澄清 authority-local 受限态与快速失败契约

#### Design Notes
- 不扩到跨仓 remote authority 管理链路，只把当前 backend 边界显式化
- Go orchestration guard 和前端受限态同时存在，避免页面外调用者继续等待 timeout

#### Validation
- `$env:GOWORK='off'; go test ./internal/services/permission/... -count=1`
  - 结果：通过
- `npm test -- PermitIssuance`
  - 结果：通过
- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：仍会打印 `Not found: time.Time`，但退出码为 0
- `npm run build`
  - 结果：通过

#### Blockers
- none

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 已覆盖 remote authority permit timeout 收敛、显式限制提示、authority-local guard 和文档澄清
- 架构合理性：通过
  - 保持改动面在 Win repo 内，不误扩到跨仓 runtime
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 只减少了无效 timeout 等待，没有新增额外请求
- 可读性与一致性：通过
  - 页面与 Go service 的错误边界语义保持一致
- 可扩展性与配置化：通过
  - 将来若补齐 remote 链路，可集中回退 guard 和页面受限态
- 稳定性与安全：通过
  - authority 本地场景不变；remote authority 不再伪装成普通失败
- 测试覆盖情况：通过
  - Go / permit 页面单测通过，前端构建通过
- 子Agent治理与审计：通过
  - 未使用子Agent

### Stage 4 - Change Archive
#### $m-docs Check
- 使用 `$m-docs` 校验 plan/change/lessons 路由
- Requirements impact: `updated`
- Specs impact: `updated`
- Lessons impact: `updated`
- 新增：
  - `docs/change/2026-03-28_win-permit-remote-authority-guard.md`
  - `docs/lessons/authority-local-admin-actions.md`
- 更新：
  - `docs/change/README.md`
  - `docs/lessons/README.md`
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`

#### Archive Status
- 已完成 repo-local 归档
- 等待用户确认是否结束 workflow
