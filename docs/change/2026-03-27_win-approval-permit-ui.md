# Win Approval Permit UI Refine

## 变更背景 / 目标

- `Registration Approvals` 之前会在列表中为每条请求常驻展开 approve / reject 两块输入区，页面高度和注意力成本都偏高。
- `Permit Issuance` 之前把 issue / revoke 两块大表单长期铺在首屏，遮住了“最近结果”和协议边界这类更高频的信息。
- 本轮目标是在不改动 authority store、Wails 接口和 auth 协议的前提下，把两个页面都收敛成与 `Access Policy` 一致的“紧凑列表 / 摘要 + 聚焦弹窗”交互节奏。

## 具体变更内容

- `frontend/src/pages/RegistrationApprovals.vue`
  - pending queue 改成紧凑单行摘要列表，只保留设备、申请角色、显示名、时间摘要和 `Review` 入口。
  - 新增单一 review dialog，在同一个聚焦面中完成 approve / reject，并保留 request 级 role / reason draft。
  - 身份变化或请求消失时，自动清理 review dialog 与本地草稿，避免跨 authority 脏状态。
- `frontend/src/pages/PermitIssuance.vue`
  - 首屏改成 `Permit Actions` 紧凑动作列表 + `Latest Permit` 结果卡，不再常驻展开 issue / revoke 表单。
  - issue / revoke 改成各自独立 overlay dialog；最近一次 permit 结果继续支持复制，并可直接送入 revoke 流程。
  - 身份变化或 session 不可用时，清空 dialog 状态和本地 form，避免沿用旧 token / 旧设备草稿。
- `frontend/src/i18n/messages/operations.ts`
  - 补充审批 review dialog、permit action rows 和 focused dialog 所需文案。
- `frontend/src/pages/RegistrationApprovals.test.ts`
  - 新增紧凑队列、review dialog、approve / reject 路径测试。
- `frontend/src/pages/PermitIssuance.test.ts`
  - 新增动作行、issue dialog、copy latest permit、revoke dialog 路径测试。
- `docs/requirements/authority-admin-console.md`
- `docs/specs/authority-admin-console.md`
  - 补充注册审批页和准入许可页的长期 UI 约束，明确禁止回退为首屏大表单。

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui\docs\requirements\authority-admin-console.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\refactor-win-approval-permit-ui\docs\specs\authority-admin-console.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`

## Related lessons

- `none`

## 对应 plan.md 任务映射

- `DOC-API-1`
  - 更新 requirements/specs，明确 approvals / permit 页面不得回退为大块持久展开表单。
- `IMPL-API-1`
  - 收敛注册审批页为紧凑队列 + 聚焦 review dialog。
- `IMPL-API-2`
  - 收敛准入许可页为动作列表 + focused issue/revoke dialogs。
- `TEST-API-1`
  - 新增 approvals / permit 页面测试，覆盖关键交互路径。
- `REVIEW-API-1`
  - 通过 diff、自检和定向回归确认没有发现新增问题。
- `ARCHIVE-API-1`
  - 归档本次 workflow 结果，并补 requirements/specs / change 索引。

## 经验 / 教训摘要

- 对 authority 管理页来说，减少“同层同时可编辑”的块数，比单纯压缩间距更能控制页面臃肿感。
- 审批和 permit 这类低频但高风险操作，放进 focused dialog 后更容易保持主视图的扫读节奏，也更利于后续补测试。
- fresh worktree 的前端验证要先补 `frontend/wailsjs/**`，否则很容易把缺少生成物误判成页面改坏了。

## 可复用排查线索

- 症状：
  - 注册审批页又出现每条 request 同时展开 approve / reject 输入区
  - 准入许可页首屏再次回到长表单
  - `npm run build` 报 `Could not resolve "../../wailsjs/go/main/App"`
- 关键词：
  - `data-approval-review-dialog`
  - `data-open-issue-dialog`
  - `data-permit-revoke-dialog`
  - `wails generate module`
- 快速检查：
  - 查看 approvals 列表行中是否只保留 `Review` 主操作
  - 查看 permit 页首屏是否只剩动作入口和 `Latest Permit`
  - 若 fresh worktree 缺少 `frontend/wailsjs/**`，先执行 `$env:GOWORK='off'; wails generate module`

## 关键设计决策与权衡

- 审批页选择“紧凑队列 + 单 review dialog”，而不是保留行内 approve/reject：
  - 优点是显著降低首屏高度，并把决策信息集中到一个地方。
  - 代价是批量处理时多一次打开 dialog 的动作。
- permit 页选择“动作入口 + 最新结果”而不是把 issue / revoke 压缩成更小表单：
  - 优点是页面结构更稳定，最近 permit 的复制 / 回收路径更清楚。
  - 代价是 issue / revoke 都变成二段式操作。
- 最近 permit 卡保留直接进入 revoke flow 的入口，但不直接在卡片里执行 revoke：
  - 这样既保留高频路径，也不破坏“危险输入集中在 focused dialog”这条约束。

## 测试与验证方式 / 结果

- `npm test -- RegistrationApprovals PermitIssuance authority_admin`
  - 通过
- `$env:GOWORK='off'; wails generate module`
  - 通过
- `npm run build`
  - 通过

## 潜在影响与回滚方案

- 潜在影响：
  - approve / reject、issue / revoke 都改成先进入 dialog，操作节奏比旧版多一步。
  - permit 页把 revoke 从首屏输入改成 focused dialog 后，用户需要适应新的入口位置。
- 回滚方案：
  - 回退 `frontend/src/pages/RegistrationApprovals.vue`
  - 回退 `frontend/src/pages/PermitIssuance.vue`
  - 回退 `frontend/src/pages/RegistrationApprovals.test.ts`
  - 回退 `frontend/src/pages/PermitIssuance.test.ts`
  - 回退 `frontend/src/i18n/messages/operations.ts`
  - 回退 `docs/requirements/authority-admin-console.md`
  - 回退 `docs/specs/authority-admin-console.md`
  - 删除本次 `docs/change/2026-03-27_win-approval-permit-ui.md` 归档并恢复 `docs/change/README.md`

## 子Agent执行轨迹

- 未使用子Agent
