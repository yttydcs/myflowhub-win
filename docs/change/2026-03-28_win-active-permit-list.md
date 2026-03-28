# Win Active Permit List

## 变更背景 / 目标

- authority 管理端的“准入许可”页此前仍以“最近一次签发结果”为中心，和用户希望的“顶部操作 + 当前活动许可列表”不一致。
- 本轮目标是把页面改成以真实活动 permit 列表为单一事实来源，并补齐 Win 侧到 auth action 的 typed 调用链。

## 具体变更内容

- `internal/services/auth/authority.go`
  - 新增 `ListRegisterPermitsReq/Resp`
  - 新增 `RegisterPermitInfo`
  - 新增 `ListRegisterPermits` / `ListRegisterPermitsSimple`
- `internal/services/permission/service.go`
  - 新增 `ListRegisterPermitsRequest`
  - 新增 `RegisterPermit`
  - 新增 `ListRegisterPermitsResult`
  - 新增 `ListRegisterPermits(...)`
  - 新增 `toRegisterPermits(...)`
- `frontend/src/stores/permitIssuance.ts`
  - 从 `lastIssued/lastRevoke` 状态切到列表状态
  - 新增 `loading`、`busyPermit`、`total`、`items`
  - 新增 `loadPermits()`
  - `issuePermit()` / `revokePermit()` 只返回动作结果，成功后由页面统一刷新列表
- `frontend/src/pages/PermitIssuance.vue`
  - 页面顶部改为 `Refresh` + `New Permit`
  - 主体改为活动 permit 列表、空态和 loading 态
  - 撤销改为行内 `Revoke`
  - 删除 latest-only 结果卡与手工 revoke dialog
- `frontend/src/pages/PermitIssuance.test.ts`
  - 改为列表流测试
  - 修正 `reset()` mock，只清前端状态，不误删模拟中的后端 permit 数据
- `frontend/src/stores/authority_admin.test.ts`
  - 补 `ListRegisterPermits` mock 和 list -> issue -> list -> revoke -> list 回归
- `frontend/src/i18n/messages/operations.ts`
  - 新增 permit list 相关文案
- `docs/requirements/authority-admin-console.md`
  - 明确 Win authority console 需要暴露 `ListRegisterPermits`
- `docs/specs/authority-admin-console.md`
  - 明确 permit issuance 页改为活动 permit 列表视图

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `none`

## Related requirements

- `docs/requirements/authority-admin-console.md`

## Related specs

- `docs/specs/authority-admin-console.md`

## Related lessons

- `docs/lessons/frontend-build-babel-parser-missing.md`

## 对应 plan.md 任务映射

- `WIN-PERMIT-1`
- `WIN-PERMIT-2`
- `VALIDATE-PERMIT-1`
- `REVIEW-PERMIT-1`
- `ARCHIVE-PERMIT-1`

## 经验 / 教训摘要

- permit UI 一旦已经有真实后端列表，就不应该继续在前端维护“latest permit”分支状态；否则列表和动作结果会天然分叉。
- 页面初始化时的 `reset()` 只能重置本地视图状态，不能顺带抹掉测试里的“服务端现状”。
- fresh worktree 的前端构建仍依赖先生成 `frontend/wailsjs/**`。

## 可复用排查线索

- 症状
  - 页面只显示最近一次 permit，而不是当前全部活动 permit
  - issue 或 revoke 成功后，页面状态和后端真实 permit 集不一致
  - fresh worktree 下 `npm run build` 报 `Could not resolve "../../wailsjs/go/main/App"`
- 触发条件
  - 仍沿用 latest-only store 设计
  - 页面刷新链路没有回到 `ListRegisterPermits`
  - 尚未执行 `wails generate module`
- 关键词
  - `ListRegisterPermits`
  - `permitStore.loadPermits`
  - `data-permit-list`
  - `frontend/wailsjs`
- 快速检查
  - 查看 `frontend/src/stores/permitIssuance.ts` 是否以 `items/total/loading` 为中心
  - 查看 `frontend/src/pages/PermitIssuance.vue` 是否在 issue / revoke 后重新加载列表
  - 查看 `frontend/wailsjs` 是否已经生成

## 关键设计决策与权衡

- 不新增 `auth.permit.list` 权限，列表沿用 `auth.permit.issue` 或 `auth.permit.revoke`
  - 优点：改动面最小，和现有权限模型兼容
  - 代价：列表权限粒度没有再细分
- 以“活动 permit 列表”为单一事实来源，而不是保留 latest card
  - 优点：状态简单，和真实 auth runtime 语义一致
  - 代价：不再提供脱离列表的快捷复制/撤销卡片

## 测试与验证方式 / 结果

- 临时 workspace 下执行 `go test ./internal/services/... -count=1 -p 1`
  - 环境：临时 `go.work` 指向本地 `MyFlowHub-Core`、Proto worktree 和 Win worktree
  - 结果：通过
- `npm test -- PermitIssuance authority_admin`
  - 结果：通过
- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：过程中有 `Not found: time.Time` 提示，但命令退出码为 0，bindings 已生成
- `npm run build`
  - 结果：通过

## 潜在影响与回滚方案

- 潜在影响
  - 准入许可页不再保留 latest-only 卡片，操作路径全部收敛到活动列表
  - `PermitIssuance` store 的外部读取方式从 latest 状态改为列表状态
- 回滚方案
  - 回退 `internal/services/auth/authority.go`
  - 回退 `internal/services/permission/service.go`
  - 回退 `frontend/src/stores/permitIssuance.ts`
  - 回退 `frontend/src/pages/PermitIssuance.vue`
  - 回退相关测试与 i18n 文案

## 子Agent执行轨迹

- 未使用子Agent
