# 2026-03-26_win-authority-group-label

## 变更背景 / 目标
- 用户希望将 Win 左侧导航中 Authority 分组的中文显示名从 `准入管理` 调整为更直观的 `访问控制`。
- 本轮目标仅限中文文案替换，不改导航结构、页面路由和后端能力。

## 具体变更内容
- 更新 `frontend/src/i18n/messages/shell.ts`
  - 将 `shellZhCN["Authority"]` 从 `准入管理` 改为 `访问控制`
- 新增 `frontend/src/i18n/messages/shell.test.ts`
  - 为该词条增加最小 Vitest 断言，避免后续回归

## Requirements impact
- none

## Specs impact
- none

## Lessons impact
- none

## Related requirements
- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\authority-admin-console.md`

## Related specs
- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\authority-admin-console.md`

## Related lessons
- none

## 对应 plan.md 任务映射
- `IMPL-1`
  - 更新 Authority 组中文词条
- `TEST-1`
  - 新增词条回归测试并执行验证
- `REVIEW-1`
  - 完成 stage 3.3 自检

## 经验 / 教训摘要
- 对这类只影响展示文案的导航调整，优先保留既有导航 key 和路由结构，只改 locale 词条，变更面最小。
- 即使是纯文案改动，也可以补一条极小的词条测试，提高回归可见性。

## 可复用排查线索
- 症状：
  - 中文环境左侧仍显示 `准入管理`
- 触发条件：
  - `shellZhCN["Authority"]` 未更新或被后续提交改回
- 关键词 / 快速检查：
  - `frontend/src/i18n/messages/shell.ts`
  - `shellZhCN`
  - `Authority`
  - `访问控制`
  - `准入管理`

## 关键设计决策与权衡
- 决策：保留导航组 key `Authority`，只调整中文 locale 映射
- 原因：这是最小安全改动，不影响英文环境、路由配置和已有引用
- 代价：若未来希望统一改成新的英文术语，需要再单独调整英文文案

## 测试与验证方式 / 结果
- 执行：
  - `npm test -- src/i18n/messages/shell.test.ts`
- 结果：
  - 1 个测试文件通过，1 条断言通过
- 备注：
  - 由于 worktree 默认没有 `frontend/node_modules`，验证前临时创建了指向主仓依赖目录的 junction，测试后已移除

## 潜在影响
- 仅影响中文环境左侧导航分组标题显示
- 不影响组内页面、业务逻辑、权限动作和后端交互

## 回滚方案
- 将 `frontend/src/i18n/messages/shell.ts` 中 `Authority` 的中文文案改回 `准入管理`
- 删除 `frontend/src/i18n/messages/shell.test.ts`

## 子Agent执行轨迹
- 未使用子 Agent
