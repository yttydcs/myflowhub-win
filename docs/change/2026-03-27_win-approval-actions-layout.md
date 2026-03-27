# Win Approval Actions Layout

## 变更背景 / 目标

- 注册审批页已经收敛成紧凑队列 + 审阅弹窗，但顶部摘要卡正文里仍保留 `Resolve / Refresh` 双按钮。
- 用户反馈这两个按钮悬在页面中部比较突兀，破坏了摘要区的扫读节奏。
- 本轮目标是在不改动 authority / pending list 契约的前提下，把动作入口收进更合理的位置，并去掉冗余入口。

## 具体变更内容

- `frontend/src/pages/RegistrationApprovals.vue`
  - 删除摘要卡正文中的 `Resolve / Refresh` 按钮行。
  - 移除显式 `Resolve` UI 入口。
  - 将单一 `Refresh` 按钮移动到 `Pending Queue` header actions。
  - 保持现有 `loadPending()`、`approveRegister()`、`rejectRegister()` 与 authority 自动解析链路不变。
- `frontend/src/pages/RegistrationApprovals.test.ts`
  - 补充 `data-approval-refresh` 入口存在断言。
  - 补充页面不再渲染 `Resolve` 按钮、且仅保留一个 `Refresh` 入口的测试。

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\fix-win-approval-actions-layout\docs\requirements\authority-admin-console.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\fix-win-approval-actions-layout\docs\specs\authority-admin-console.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`

## Related lessons

- `D:\project\MyFlowHub3\worktrees\fix-win-approval-actions-layout\docs\lessons\README.md`

## 对应 plan.md 任务映射

- `IMPL-RAA-1`
  - 收敛注册审批页动作入口布局，删除冗余 `Resolve` 并重定位 `Refresh`
- `TEST-RAA-1`
  - 更新页面测试，锁定新的动作入口结构
- `REVIEW-RAA-1`
  - 完成注册审批页回归、构建与契约复核

## 经验 / 教训摘要

- 如果某个页面动作已经能被主路径自动覆盖，继续把它作为并列主按钮暴露在正文中，通常只会增加视觉噪音。
- 列表页的手动动作最好挂在列表 header 或同语义区域，而不是插在摘要内容中间。

## 可复用排查线索

- 症状：
  - 注册审批页顶部摘要区中央出现孤立的 `Resolve / Refresh`
  - 用户感觉动作位置突兀，页面视觉节奏被打断
- 关键词：
  - `data-approval-refresh`
  - `Pending Queue`
  - `resolveAuthorityAction`
- 快速检查：
  - 查看摘要卡正文是否还存在独立动作行
  - 查看 `Pending Queue` header 是否承载唯一的手动刷新入口

## 关键设计决策与权衡

- 删除显式 `Resolve`，而不是仅仅平移位置：
  - 优点：因为 `Refresh` 已可自动触发 authority 解析，用户路径更简单
  - 代价：失去“只解析不刷新”的显式页面按钮
- 将 `Refresh` 放到 `Pending Queue` header：
  - 优点：动作与数据区域语义一致，页面更干净
  - 代价：相比悬在摘要区中央，视觉存在感会更收敛

## 测试与验证方式 / 结果

- `npm test -- RegistrationApprovals`
  - 通过
- `npm run build`
  - 首次失败，原因是 fresh worktree 缺少 `frontend/wailsjs/**`
- `$env:GOWORK='off'; wails generate module`
  - 通过
- `npm run build`
  - 通过

## 潜在影响与回滚方案

- 潜在影响：
  - 页面不再提供单独的 `Resolve` 按钮，手动入口收敛为单一 `Refresh`
  - 手动动作存在感降低，但页面整体更整洁
- 回滚方案：
  - 回退 `frontend/src/pages/RegistrationApprovals.vue` 与 `frontend/src/pages/RegistrationApprovals.test.ts`
  - 删除本次 change 归档并回退 `docs/change/README.md`

## 子Agent执行轨迹

- 未使用子Agent
