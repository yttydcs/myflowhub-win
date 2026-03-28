# Plan - Win 远程 Authority 审批与准入许可收口

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/remote-authority-admin`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin`
- Current Stage: `4 archive complete, awaiting workflow end decision`

## Goal / Current State
- 目标：让 Win 端 `Registration Approvals` 与 `Permit Issuance` 在 `sourceId != authorityId` 的 remote authority 场景下继续可用，并复用已完成的 auth runtime 远程转发能力。
- 当前状态：
  - `feat-subproto-remote-authority-admin` 已完成 auth runtime 的转发、原始操作者权限校验和 targeted response。
  - Win worktree 的 requirements/specs 已更新为“remote authority 允许审批/permit 管理”。
  - Win 代码仍残留 permit authority-local guard 和 permit 页受限态 UI，因此前端行为与稳定文档不一致。

## Participating Repos / Dependencies
- Active execution worktree:
  - Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - Branch: `feat/remote-authority-admin`
  - Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin`
- Related dependency worktrees:
  - Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto`
  - Branch: `feat/remote-authority-admin`
  - Worktree: `D:\project\MyFlowHub3\worktrees\feat-subproto-remote-authority-admin`
  - Ownership boundary: auth runtime forwarding and permission enforcement
  - Key reference: `D:\project\MyFlowHub3\worktrees\feat-subproto-remote-authority-admin\auth\remote_admin_test.go`
  - Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Server`
  - Branch: `feat/remote-authority-admin`
  - Worktree: `D:\project\MyFlowHub3\worktrees\feat-server-remote-authority-admin`
  - Ownership boundary: stable auth spec and eventual dependency bump
  - Key reference: `D:\project\MyFlowHub3\worktrees\feat-server-remote-authority-admin\docs\specs\auth.md`

## Related Requirements / Specs / Lessons
- Requirements impact: `updated`
- Specs impact: `updated`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\docs\requirements\authority-admin-console.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\docs\specs\authority-admin-console.md`
  - `D:\project\MyFlowHub3\worktrees\feat-server-remote-authority-admin\docs\specs\auth.md`
- Related lessons:
  - `D:\project\MyFlowHub3\docs\lessons\authority-local-admin-actions.md`

## Parallelism Assessment
- 不派发子 Agent。
- 原因：
  - `PermissionService`、`PermitIssuance.vue`、页面测试都围绕同一行为边界，写集和验证结果彼此强耦合。
  - 当前关键路径是尽快恢复 Win 端 remote authority 行为并做本地回归，主 agent 直接完成更稳妥。

## Task Checklist
- [x] `WIN-RA-1` 删除 `PermissionService` 对 remote authority permit 的本地快速失败
- [x] `WIN-RA-2` 清理 permit 页面 remote blocked 逻辑，恢复自动加载和正常动作
- [x] `WIN-RA-3` 更新 Go / 前端回归，确认 remote authority permit 行为与文档一致
- [x] `WIN-RA-4` 审核注册审批页是否仍满足 remote authority 场景与当前规范
- [x] `WIN-RA-5` 执行 `go test`、`npm test`、`wails generate module`、`npm run build`

## Tasks

### `WIN-RA-1`
- Goal:
  - 移除 permit authority-local guard，允许 `ListRegisterPermits`、`IssueRegisterPermit`、`RevokeRegisterPermit` 继续透传给 authority。
- Files:
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\internal\services\permission\service.go`
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\internal\services\permission\service_test.go`
- Acceptance:
  - service 不再因 `sourceId != authorityId` 快速失败
  - 参数校验与 `auth service not initialized` 等本地错误保持不变
- Tests:
  - `go test ./internal/services/permission/... -count=1`
- Rollback:
  - 恢复 `ensureAuthorityLocalPermitAction(...)` 及相关单测

### `WIN-RA-2`
- Goal:
  - Permit 页面在 remote authority 场景下继续自动加载 permit 列表，并允许签发与撤销。
- Files:
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\frontend\src\pages\PermitIssuance.vue`
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\frontend\src\i18n\messages\operations.ts`
- Acceptance:
  - 不再出现 authority-local blocked notice
  - `Refresh` / `New Permit` 在 remote authority 场景不因 block 被禁用
  - 自动加载失败时仍以内联错误呈现，不弹额外 toast
- Tests:
  - `npm test -- PermitIssuance`
- Rollback:
  - 恢复 `remoteAuthorityBlocked`、`ensurePermitAuthorityLocal()` 和相关文案

### `WIN-RA-3`
- Goal:
  - 把测试基线从“remote authority 被阻断”改为“remote authority 仍可用”。
- Files:
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\frontend\src\pages\PermitIssuance.test.ts`
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\internal\services\permission\service_test.go`
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\frontend\src\stores\authority_admin.test.ts`
- Acceptance:
  - permit 页测试覆盖 remote authority 自动加载与动作入口可用
  - service 测试不再断言 authority-local 错误
- Tests:
  - `go test ./internal/services/permission/... -count=1`
  - `npm test -- PermitIssuance authority_admin`
- Rollback:
  - 恢复 guard 基线相关断言

### `WIN-RA-4`
- Goal:
  - 复核 `Registration Approvals` 是否已经满足 remote authority 规范，避免遗漏旧 guard。
- Files:
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\frontend\src\pages\RegistrationApprovals.vue`
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\frontend\src\pages\RegistrationApprovals.test.ts`
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\frontend\src\stores\registrationApprovals.ts`
- Acceptance:
  - 审批页不存在 authority-local 阻断或与规范冲突的旧行为
- Tests:
  - `npm test -- RegistrationApprovals`
- Rollback:
  - 无代码回滚要求，仅记录复核结论

### `WIN-RA-5`
- Goal:
  - 执行 Win worktree 所需回归，确认 bindings、前端构建和目标测试可通过。
- Files:
  - 无新增写集；执行验证命令
- Acceptance:
  - 目标测试和构建命令均返回成功；若依赖版本未完成导致跨仓阻塞，显式记录
- Tests:
  - `$env:GOWORK='off'; go test ./internal/services/auth/... ./internal/services/permission/... -count=1`
  - `npm test -- RegistrationApprovals PermitIssuance authority_admin`
  - `$env:GOWORK='off'; wails generate module`
  - `npm run build`
- Rollback:
  - 无

## Risks / Notes
- Win 行为恢复后，若消费方实际链接到旧版 `myflowhub-subproto/auth`，remote authority admin 仍可能在运行时失败；这属于依赖发布/升级阻塞，不是当前 Win guard 问题。
- 主仓存在用户自己的未提交更改，本轮只在 worktree 内实施，不回退主仓内容。

## Validation Results
- `$env:GOWORK='off'; go test ./internal/services/auth/... ./internal/services/permission/... -count=1`
  - 结果：通过
- `npm ci`
  - 结果：通过
  - 说明：该 worktree 初始缺少 `frontend/node_modules`，补齐依赖后再执行前端回归
- `npm test -- RegistrationApprovals PermitIssuance authority_admin`
  - 结果：通过
- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：仍会打印 `Not found: time.Time`，但退出码为 0
- `npm run build`
  - 结果：通过

## Stage 3.3 Review
- 需求覆盖：通过
  - permit 页恢复 remote authority 自动加载、签发、撤销；审批页已复核，无 authority-local 残留 guard
- 架构合理性：通过
  - 复用既有 `PermissionService` 和 auth stable action，不引入 Win 专用协议分支
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 只删除错误的本地阻断，不增加额外轮询或重复加载
- 可读性与一致性：通过
  - Win UI、service 和 requirements/specs 重新对齐到同一行为基线
- 可扩展性与配置化：通过
  - 若后续依赖版本未升级，问题会落在真实 authority 响应或版本漂移，而不是隐藏在 Win guard 中
- 稳定性与安全：通过
  - 仍保留本地参数校验和 authority/auth 错误透传，不吞错
- 测试覆盖情况：通过
  - Go、Vitest、Wails binding 生成和前端构建均已验证
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未使用子Agent

## Stage 4 Archive Status
- 已使用 `$m-docs` 完成 change / lessons 路由校验
- 已新增：
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\docs\change\2026-03-28_win-remote-authority-admin.md`
- 已更新：
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\docs\lessons\authority-local-admin-actions.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\docs\lessons\README.md`

阻塞：否
等待用户确认是否结束 workflow
