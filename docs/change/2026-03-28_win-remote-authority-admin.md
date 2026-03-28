# 2026-03-28 Win 远程 Authority 审批与准入许可恢复

## 变更背景 / 目标

- 前一轮 Win 为了收敛 `auth list_register_permits: request timed out`，引入了 permit authority-local guard 和 remote blocked UI。
- 随后跨仓排查确认，稳定方向不是长期保留 guard，而是补齐 auth runtime 的 remote authority admin 链路，并让 Win 回到真实 authority 请求路径。
- 本轮目标是：
  - 删除 Win 侧对 remote authority permit 的本地快速失败
  - 恢复 permit 页在 remote authority 场景下的自动加载、签发、撤销
  - 复核注册审批页，使 Win requirements/specs、service、页面和 lessons 对齐到同一基线

## 具体变更内容

### Win service

- 更新 `internal/services/permission/service.go`
  - 删除 `ListRegisterPermits`
  - `IssueRegisterPermit`
  - `RevokeRegisterPermit`
  - 中的 authority-local 快速失败逻辑
  - 保留原有参数校验、`auth service not initialized` 检查和 authority 错误透传
- 更新 `internal/services/permission/service_test.go`
  - 把原“remote authority 必须报 `requires authority-local session`”回归改为“remote authority 允许进入 auth 调用前置校验”

### Permit 页面

- 更新 `frontend/src/pages/PermitIssuance.vue`
  - 删除 `remoteAuthorityBlocked`
  - 删除 `ensurePermitAuthorityLocal()`
  - 删除 authority-local blocked notice
  - 恢复 remote authority 场景下的首次自动加载
  - `Refresh / New Permit` 不再因 remote authority 被禁用
- 更新 `frontend/src/pages/PermitIssuance.test.ts`
  - 回归基线改为“remote authority 仍可加载 permit 列表，动作入口保持可用”
- 更新 `frontend/src/i18n/messages/operations.ts`
  - 删除已失效的 authority-local permit 阻断文案

### 稳定文档与 lessons

- 已对齐并保留本轮相关稳定文档改动：
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- 更新 `docs/lessons/authority-local-admin-actions.md`
  - 把 lesson 从“authority-local guard 是当前基线”改为“remote authority admin 才是当前基线，失败时优先查版本和路由归属”

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `updated`

## Related requirements

- `docs/requirements/authority-admin-console.md`

## Related specs

- `docs/specs/authority-admin-console.md`
- `D:\project\MyFlowHub3\worktrees\feat-server-remote-authority-admin\docs\specs\auth.md`

## Related lessons

- `docs/lessons/authority-local-admin-actions.md`

## 对应 plan.md 任务映射

- `WIN-RA-1`
- `WIN-RA-2`
- `WIN-RA-3`
- `WIN-RA-4`
- `WIN-RA-5`

## 经验 / 教训摘要

- authority-local guard 只适合作为旧版本排障的临时止血，不应继续充当稳定基线。
- 当 service、页面和稳定文档不一致时，用户会同时看到 timeout、blocked notice 和相互冲突的排障建议，排查成本明显上升。
- 远程 authority admin 失败时，优先检查协议依赖版本、authority 路由归属和旧 guard 残留，而不是先把问题解释为“必须到 authority 本机操作”。

## 可复用排查线索

- 症状
  - `Registration Approvals` 或 `Permit Issuance` 在 remote authority 场景仍失败
  - 错误包含 `auth list_pending_registers: request timed out`
  - 错误包含 `auth list_register_permits: request timed out`
  - 错误包含 `requires authority-local session`
- 触发条件
  - `sourceId != authorityId`
  - 消费方未升级到包含 remote authority admin 的 auth 依赖
  - Win 页面或 orchestration 层残留 authority-local guard
- 关键词
  - `authority-local`
  - `requires authority-local session`
  - `list_register_permits`
  - `list_pending_registers`
  - `routed source`
  - `source mismatch`
- 快速检查
  - 检查当前 session `nodeId` 与解析出的 `authorityId`
  - 检查消费者实际依赖的 `github.com/yttydcs/myflowhub-subproto/auth` 版本
  - 检查 Win 是否还存在 `authority-local` 或 `data-permit-remote-authority` 残留
  - 检查 authority 侧 `SourceID -> 入站连接` 路由归属是否正确

## 关键设计决策与权衡

- 删除 Win authority-local guard，而不是继续强化受限态
  - 优点：让 Win 行为回到真实 authority 请求路径，与新 requirements/specs 一致
  - 代价：若消费者依赖仍停留在旧版本，运行时失败会重新暴露出来，但这是更真实的系统状态
- 不新增 Win 专用协议或旁路 action
  - 优点：保持协议面单一，权限校验仍由 authority 基于真实操作者完成
  - 代价：必须依赖 SubProto/Server 消费方同步升级
- 注册审批页本轮以复核为主，不重复改动
  - 优点：保持最小安全改动面
  - 代价：把代码修改集中在确实残留 guard 的 permit 路径

## 测试与验证方式 / 结果

- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./internal/services/auth/... ./internal/services/permission/... -count=1`
  - 结果：通过
- `MyFlowHub-Win/frontend`
  - `npm ci`
  - 结果：通过
  - 说明：当前 worktree 初始缺少 `node_modules`，安装依赖后再执行前端回归
- `MyFlowHub-Win/frontend`
  - `npm test -- RegistrationApprovals PermitIssuance authority_admin`
  - 结果：通过
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：仍会打印 `Not found: time.Time`，但退出码为 0
- `MyFlowHub-Win/frontend`
  - `npm run build`
  - 结果：通过

## 潜在影响与回滚方案

- 潜在影响
  - remote authority 场景下，permit 页会重新走真实 authority admin 请求链路
  - 若消费者依赖未升级，用户看到的会是 authority 的真实错误，而不再是 Win 本地伪限制
- 回滚方案
  - 回退 `internal/services/permission/service.go`
  - 回退 `internal/services/permission/service_test.go`
  - 回退 `frontend/src/pages/PermitIssuance.vue`
  - 回退 `frontend/src/pages/PermitIssuance.test.ts`
  - 回退 `frontend/src/i18n/messages/operations.ts`
  - 回退 `docs/requirements/authority-admin-console.md`
  - 回退 `docs/specs/authority-admin-console.md`
  - 回退本 change 和对应 lesson 更新

## 子Agent执行轨迹

- 未使用子Agent
