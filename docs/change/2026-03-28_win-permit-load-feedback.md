# 2026-03-28 Win 准入许可页加载反馈与头部动作收敛

## 变更背景 / 目标

- 当前 `Permit Issuance` 页打开时会立即自动加载活动 permit 列表；一旦请求失败，页面会直接弹出 `Failed to load permits.` toast，打断首屏阅读。
- 同时，`Refresh / New Permit` 仍停留在上方介绍卡片中，而“活动许可”卡片右上角还保留 `共 X 条 / 实时` 标签，视觉上偏臃肿。
- 本轮目标是：
  - 修复 Win 侧 permit list 回包解析缺口
  - 把 permit list 失败反馈收敛为页面内错误提示
  - 把动作按钮移动到“活动许可”卡片右上角，并移除多余状态标签

## 具体变更内容

- `internal/services/auth/service.go`
  - 在 `extractAuthCodeMsg()` 中补齐 `*ListRegisterPermitsResp`
  - 确保 `list_register_permits` 成功回包不会被误判为 `code=0`
- `internal/services/auth/service_test.go`
  - 新增 `TestExtractAuthCodeMsgSupportsListRegisterPermitsResp`
- `frontend/src/pages/PermitIssuance.vue`
  - 新增本地 `loadState.error`
  - permit list 自动加载或手动刷新失败时，改为在 permit card 内展示错误提示
  - 不再对 permit list 加载失败直接触发 `toast.errorOf`
  - 将 `Refresh` / `New Permit` 移动到“活动许可”卡片右上角
  - 移除 permit 总数和“实时” badge
  - 保持已有 permit 列表数据，不因一次刷新失败被清空
- `frontend/src/pages/PermitIssuance.test.ts`
  - 调整动作按钮断言到 permit card actions
  - 新增自动加载失败时的页面内错误提示回归
- `frontend/src/i18n/messages/operations.ts`
  - 新增 `Failed to load permits.` 中文文案

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `none`

## Related requirements

- `docs/requirements/authority-admin-console.md`

## Related specs

- `docs/specs/authority-admin-console.md`
- `D:\\project\\MyFlowHub3\\repo\\MyFlowHub-Server\\docs\\specs\\auth.md`

## Related lessons

- `none`

## 对应 plan.md 任务映射

- `PERMIT-FIX-1`
- `PERMIT-FIX-2`
- `PERMIT-FIX-3`

## 经验 / 教训摘要

- auth typed action 一旦新增新响应结构，统一 `code/msg` 抽取必须同步扩展；否则成功回包也可能被 UI 误判。
- 对首屏自动加载型列表，失败反馈更适合内嵌在页面自身，而不是直接打断用户的 toast。
- permit 页面真正的主动作属于“活动许可”列表本身，把按钮挂在列表卡片头部会比挂在介绍卡里更顺手。

## 可复用排查线索

- 症状
  - 打开准入许可页直接看到 `Failed to load permits.`
  - permit list 成功回包后仍被 UI 当作失败
  - “活动许可”卡片头部仍显示 `共 X 条 / 实时`
- 触发条件
  - `extractAuthCodeMsg()` 未覆盖 `ListRegisterPermitsResp`
  - permit 页首屏自动加载失败仍走 toast 路径
- 关键词
  - `ListRegisterPermitsResp`
  - `extractAuthCodeMsg`
  - `data-permit-load-error`
  - `data-permit-card-actions`
- 快速检查
  - 查看 `internal/services/auth/service.go` 是否已处理 `*ListRegisterPermitsResp`
  - 查看 `frontend/src/pages/PermitIssuance.vue` 是否使用页面内错误提示而非直接 toast
  - 查看 `frontend/src/pages/PermitIssuance.vue` 中按钮是否位于 permit card header actions

## 关键设计决策与权衡

- 不修改 authority 解析或跨 hop permit 协议，只修 Win 本地可确定的解析和交互问题
  - 优点：改动面小，回滚清晰
  - 代价：若现场真实根因是后端完全不回 `list_register_permits_resp`，页面仍只会显示错误提示而不会伪造列表
- permit list 失败改为页面内提示，而不是继续 toast
  - 优点：首屏更稳定，用户仍可继续点 `Refresh` 或 `New Permit`
  - 代价：错误提示不再弹到全局提示层

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./internal/services/auth/... -count=1`
  - 结果：通过
- `npm test -- PermitIssuance`
  - 结果：通过
- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：过程中仍会打印 `Not found: time.Time`，但命令退出码为 0
- `npm run build`
  - 结果：通过

## 潜在影响与回滚方案

- 潜在影响
  - permit 页加载失败后会优先显示卡片内错误提示，不再自动弹 toast
  - permit 页按钮位置发生变化，动作入口更靠近列表标题
- 回滚方案
  - 回退 `internal/services/auth/service.go`
  - 回退 `internal/services/auth/service_test.go`
  - 回退 `frontend/src/pages/PermitIssuance.vue`
  - 回退 `frontend/src/pages/PermitIssuance.test.ts`
  - 回退 `frontend/src/i18n/messages/operations.ts`

## 子Agent执行轨迹

- 未使用子Agent
