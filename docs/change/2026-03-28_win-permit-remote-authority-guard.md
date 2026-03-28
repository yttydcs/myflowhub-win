# 2026-03-28 Win 准入许可页 remote authority 超时收敛

## 变更背景 / 目标

- 在 `2026-03-28` 已完成的 Win permit load feedback 修复后，用户仍在 `Permit Issuance` 页看到：
  - `加载准入许可失败。`
  - `auth list_register_permits: request timed out`
- 这次排查确认，问题不再是 Win 侧响应解析缺口，而是当前 backend 对审批/permit 管理动作仍以 authority 本地操作为前提。
- 本轮目标是把这个事实收敛成可理解、可预期的 authority-local 限制提示，而不是继续让 permit 页表现成“普通加载失败”。

## 具体变更内容

### Go orchestration

- 更新 `internal/services/permission/service.go`
  - 为 `ListRegisterPermits`
  - `IssueRegisterPermit`
  - `RevokeRegisterPermit`
  - 增加 `ensureAuthorityLocalPermitAction(...)`
  - 当 `source_id != authority_id` 时，直接返回 `requires authority-local session` 错误
  - 不再继续等待 auth 层 8 秒超时
- 更新 `internal/services/permission/service_test.go`
  - 新增 remote authority permit 管理 guard 回归

### 前端 permit 页

- 更新 `frontend/src/pages/PermitIssuance.vue`
  - 新增 remote authority 受限态判断
  - 当 `authorityId != current nodeId` 时：
    - 不再自动请求 permit 列表
    - 显示 authority-local 提示
    - 禁用 `Refresh / New Permit`
  - authority 本机场景下保持现有列表、签发、撤销路径不变
- 更新 `frontend/src/pages/PermitIssuance.test.ts`
  - 补 remote authority 下“不自动加载 + 显式提示 + 按钮禁用”回归
- 更新 `frontend/src/i18n/messages/operations.ts`
  - 新增 remote authority permit 限制提示中文文案

### 稳定文档澄清

- 更新 `docs/requirements/authority-admin-console.md`
  - 澄清 permit 页在 remote authority 场景下必须显式提示并停止继续等待 timeout
- 更新 `docs/specs/authority-admin-console.md`
  - 澄清 permit 页 authority-local 受限态与 orchestration 快速失败约束

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
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`

## Related lessons

- `docs/lessons/authority-local-admin-actions.md`

## 对应 plan.md 任务映射

- `PERMIT-GUARD-1`
- `PERMIT-GUARD-2`
- `DOC-CLARIFY-1`
- `VALIDATE-1`
- `REVIEW-1`
- `ARCHIVE-1`

## 经验 / 教训摘要

- `list_register_permits` 超时不一定是 Win 页面 bug；也可能是当前 backend 根本没有把该管理动作实现成通用 remote authority 链路。
- 对这种“后端能力边界”场景，页面应优先展示显式限制提示，而不是继续表现成普通失败态。
- 把 authority-local guard 放到 Win orchestration 层后，页面外的其它调用者也不会再无意义地等满 auth timeout。

## 可复用排查线索

- 症状
  - permit 页打开就提示 `加载准入许可失败。`
  - 详细错误是 `auth list_register_permits: request timed out`
  - 当前登录节点并不是 authority 节点
- 触发条件
  - `sourceId != authorityId`
  - backend 仍要求审批/permit 管理从 authority 本机发起
- 关键词
  - `requires authority-local session`
  - `list_register_permits`
  - `authorityId != sourceId`
  - `data-permit-remote-authority`
- 快速检查
  - 查看当前 session `nodeId` 是否等于 `authorityId`
  - 查看 `docs/specs/auth.md` 是否仍注明审批/permit 管理建议从 authority 节点操作
  - 查看 `PermitIssuance.vue` 是否在 remote authority 下跳过自动加载并展示提示

## 关键设计决策与权衡

- 不扩到跨仓 remote authority permit 管理链路，只在 Win 侧做 authority-local guard
  - 优点：改动面小，行为和当前 backend 事实一致
  - 代价：这轮不提供真实 remote authority permit 管理能力
- 受限态使用页面内提示，而不是继续等 timeout
  - 优点：用户能直接理解限制边界
  - 代价：若后端将来补齐 remote 链路，需要回退 guard

## 测试与验证方式 / 结果

- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./internal/services/permission/... -count=1`
  - 结果：通过
- `MyFlowHub-Win/frontend`
  - `npm test -- PermitIssuance`
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
  - remote authority 场景下 permit 页不再尝试自动加载，也不再把 timeout 暴露成普通失败
  - authority 本机场景保持原有 permit 列表、签发、撤销路径
- 回滚方案
  - 回退 `internal/services/permission/service.go`
  - 回退 `internal/services/permission/service_test.go`
  - 回退 `frontend/src/pages/PermitIssuance.vue`
  - 回退 `frontend/src/pages/PermitIssuance.test.ts`
  - 回退 `frontend/src/i18n/messages/operations.ts`
  - 回退 requirements/specs 澄清和本归档

## 子Agent执行轨迹

- 未使用子Agent
