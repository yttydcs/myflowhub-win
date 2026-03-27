# Win Approval Filter Button Height

## 变更背景 / 目标

- 注册审批页已经把 `Resolve` 收敛掉，但设备筛选输入框右侧的 `Apply Filter` 按钮仍然明显矮于输入框本体。
- 这不是交互问题，而是筛选区控件组在视觉上的高度基线不一致。
- 本轮目标是在不改动筛选行为和全局按钮体系的前提下，把这个局部按钮拉齐到和输入框同高。

## 具体变更内容

- `frontend/src/pages/RegistrationApprovals.vue`
  - 为 `Apply Filter` 按钮增加局部 `h-10` 高度类。
  - 补充 `data-approval-filter-apply` 定位属性，便于后续测试。
- `frontend/src/pages/RegistrationApprovals.test.ts`
  - 新增筛选按钮高度类断言。
  - 保持点击后仍会触发 `loadPending()` 的行为验证。

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\fix-win-approval-filter-button-height\docs\requirements\authority-admin-console.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\fix-win-approval-filter-button-height\docs\specs\authority-admin-console.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`

## Related lessons

- `D:\project\MyFlowHub3\worktrees\fix-win-approval-filter-button-height\docs\lessons\README.md`

## 对应 plan.md 任务映射

- `IMPL-RAF-1`
  - 调整注册审批页筛选按钮高度并补充稳定定位属性
- `TEST-RAF-1`
  - 更新页面测试，锁定 `h-10` 高度类和点击行为
- `REVIEW-RAF-1`
  - 完成页面回归和构建复核

## 经验 / 教训摘要

- 输入框和动作按钮成组出现时，哪怕只差一个尺寸档位，也会明显破坏视觉和谐。
- 这种局部对齐问题更适合页面级修正，而不是贸然改动全局按钮 size 基线。

## 可复用排查线索

- 症状：
  - `Device Filter` 右侧按钮明显矮于输入框
- 关键词：
  - `data-approval-filter-apply`
  - `h-10`
  - `size="sm"`
- 快速检查：
  - 查看筛选按钮是否仍在使用 `size="sm"`
  - 查看按钮是否显式带有 `h-10`

## 关键设计决策与权衡

- 采用页面局部 `h-10`，而不是修改全局 `Button` 组件：
  - 优点是影响面小，不会牵动其它使用 `sm` 的按钮
  - 代价是这个对齐规则留在页面局部，不是全局尺寸系统的一部分

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
  - 筛选按钮高度变高，但按钮语义和禁用逻辑不变
- 回滚方案：
  - 回退 `frontend/src/pages/RegistrationApprovals.vue` 与 `frontend/src/pages/RegistrationApprovals.test.ts`
  - 删除本次 change 归档并回退 `docs/change/README.md`

## 子Agent执行轨迹

- 未使用子Agent
