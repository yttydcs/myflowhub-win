# Win Access Policy Save Loading

## 变更背景 / 目标

- 访问策略页经过前几轮收敛后，主要编辑区已经更轻量，但保存入口仍然集中在右侧操作面板。
- 用户反馈默认准入、节点覆盖和角色管理里就地保存的路径不够直接，同时在 policy 加载过程中也缺少明确的页面内反馈。
- 本轮目标是在不改动 authority/store 契约的前提下，补齐就近保存入口，并让加载状态在页面中显式可见。

## 具体变更内容

- `frontend/src/pages/AccessPolicy.vue`
  - 为默认准入卡、节点覆盖卡、角色管理页头补充 `Save Policy` 入口。
  - 所有新增保存按钮继续复用既有 `savePolicy(saveOptions)` 链路，不引入局部保存协议。
  - 为当前策略 tab 和角色管理 tab 增加页面内 loading notice。
  - 统一保存 / 重载按钮的 busy 文案与禁用条件。
- `frontend/src/i18n/messages/operations.ts`
  - 补充“正在从 Authority 加载当前策略…”和“正在保存策略…”文案。
- `frontend/src/pages/AccessPolicy.test.ts`
  - 补充当前策略 tab 与角色管理 tab 中 `保存策略` 入口数量断言。
  - 新增显式 loading notice 可见性测试。
- `docs/requirements/authority-admin-console.md`
- `docs/specs/authority-admin-console.md`
  - 将“主要编辑区就近保存入口”和“页面内显式加载提示”写入稳定要求与交互契约。

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading\docs\requirements\authority-admin-console.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading\docs\specs\authority-admin-console.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`

## Related lessons

- `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-save-loading\docs\lessons\README.md`

## 对应 plan.md 任务映射

- `DOC-APL-1`
  - 更新 requirements/specs，补充就近保存入口和显式加载提示约束
- `IMPL-APL-1`
  - 为默认准入、节点覆盖、角色管理补充就近保存入口
- `IMPL-APL-2`
  - 为 policy 加载过程增加页面内 loading notice
- `TEST-APL-1`
  - 补充访问策略保存入口和 loading notice 的页面测试
- `REVIEW-APL-1`
  - 完成页面回归、构建与契约复核

## 经验 / 教训摘要

- 同一页面里只有一个远端保存按钮时，即便功能完整，也容易被用户感知为“编辑完成后还要再找一次出口”。
- 对加载态来说，按钮禁用只是次级信号；涉及 authority / policy 这种耗时链路时，页面内的明确提示更可靠。

## 可复用排查线索

- 症状：
  - 用户认为访问策略页“只有一个保存按钮”
  - 页面重新加载策略时看起来像卡住，但没有明确状态提示
- 关键词：
  - `savePolicyButtonLabel`
  - `reloadPolicyButtonLabel`
  - `policyActionBusy`
  - `data-policy-loading-notice`
- 快速检查：
  - 查看默认准入、节点覆盖、角色管理页头附近是否都有 `Save Policy`
  - 查看 `accessPolicyStore.state.loading` 为真时页面中是否渲染 loading notice

## 关键设计决策与权衡

- 保留统一保存链路，而不是实现“卡片级局部保存”：
  - 优点是最小改动，不触碰现有 store / backend 契约
  - 代价是多个按钮的语义仍然是“保存整份 policy”，需要通过稳定文档明确这一点
- 使用现有 `loading` / `saving` 状态，而不是再引入新的页面级 busy 状态：
  - 优点是状态模型简单，不新增同步点
  - 代价是 loading notice 只能反映 `loadPolicy()`，不会细分更多请求来源

## 测试与验证方式 / 结果

- `npm test -- AccessPolicy`
  - 通过
- `npm run build`
  - 通过

## 潜在影响与回滚方案

- 潜在影响：
  - 页面上会出现更多 `Save Policy` 入口，但保存语义仍是整份 policy
  - loading notice 增强了状态感知，也会让加载中区域的视觉存在感更强
- 回滚方案：
  - 回退 `frontend/src/pages/AccessPolicy.vue`、`frontend/src/i18n/messages/operations.ts`、`frontend/src/pages/AccessPolicy.test.ts`
  - 恢复 `docs/requirements/authority-admin-console.md` 与 `docs/specs/authority-admin-console.md`
  - 删除本次 change 归档并回退 `docs/change/README.md`

## 子Agent执行轨迹

- 未使用子Agent
