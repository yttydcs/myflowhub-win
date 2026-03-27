# Win Permit UI Refine

## 变更背景 / 目标

- 用户希望把准入许可页继续按前几轮 authority 页面收敛方向处理，尤其是编辑框、列表表现和显式 `Resolve` 入口。
- 现状里 `Permit Issuance` 仍保留独立 `Resolve` 按钮、双列 issue 表单和偏重的 revoke `textarea`，最近一次 permit 结果也仍偏块状。
- 本轮目标是在不改动 authority / permit store 契约的前提下，把准入许可页进一步压缩成更清晰的“动作入口 + 单列弹窗 + 紧凑结果”。

## 具体变更内容

- `frontend/src/pages/PermitIssuance.vue`
  - 删除页头显式 `Resolve` 按钮和对应 `resolveAuthorityAction`。
  - 保留协议边界提示，但不再把 authority 解析作为并列主操作暴露在正文。
  - 将 `Issue Permit` dialog 调整为单列输入顺序：`Device ID`、`Role`、`Expires At`。
  - 将 `Revoke Permit` 的 token 编辑框从大 `textarea` 调整为单行 `input`。
  - 将 `Latest Permit` 结果卡内部明细收敛成紧凑行列表，并保留复制 / 进入撤销流程入口。
- `frontend/src/pages/PermitIssuance.test.ts`
  - 补充页面不再渲染 `Resolve` 文案断言。
  - 补充撤销编辑框为 `INPUT` 的结构断言。
  - 补充 latest permit 紧凑明细区存在断言。

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\fix-win-permit-ui-refine\docs\requirements\authority-admin-console.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\fix-win-permit-ui-refine\docs\specs\authority-admin-console.md`

## Related lessons

- `D:\project\MyFlowHub3\worktrees\fix-win-permit-ui-refine\docs\lessons\README.md`

## 对应 plan.md 任务映射

- `IMPL-WPR-1`
  - 收敛准入许可页动作与表单布局
- `TEST-WPR-1`
  - 更新准入许可页面测试
- `REVIEW-WPR-1`
  - 完成页面回归、构建与契约复核
- `ARCHIVE-WPR-1`
  - 归档本次变更

## 经验 / 教训摘要

- 如果主动作已经隐含 authority 自动解析链路，再把 `Resolve` 作为正文主按钮保留下来，只会增加视觉噪音。
- permit 这类低频但敏感的操作更适合单列、顺序式输入；横向双列表单和过高输入框都会放大界面臃肿感。
- `Latest Permit` 更适合作为紧凑快照列表，而不是再次展开成接近表单的块状信息区。

## 可复用排查线索

- 症状：
  - 准入许可页顶部仍出现独立 `Resolve`
  - 签发弹窗仍要求左右扫读双列输入
  - 撤销 token 输入框过高，像大段文本编辑器
  - 最新 permit 卡看起来像第二个大表单块
- 关键词：
  - `resolveAuthorityAction`
  - `data-revoke-permit-input`
  - `data-latest-permit-details`
  - `Issue Permit`
- 快速检查：
  - 查看摘要卡正文是否只剩协议边界提示
  - 查看 revoke 输入是否为 `input` 而非 `textarea`
  - 查看 latest permit 明细是否为紧凑逐行列表

## 关键设计决策与权衡

- 删除显式 `Resolve`，而不是平移到其它角落：
  - 优点：因为 `issuePermit()` / `revokePermit()` 已自动走 `requireAuthority()`，用户心智更简单
  - 代价：失去“只解析不操作”的独立按钮
- latest permit 仍保留为单张卡片，而不是再拆更细模块：
  - 优点：保持页面结构稳定，改动面最小
  - 代价：整体仍是“卡片 + 卡片”组织，只是内部更紧凑

## 测试与验证方式 / 结果

- `npm ci`
  - 通过
- `npm test -- PermitIssuance`
  - 通过
- `npm run build`
  - 首次失败，原因是 fresh worktree 缺少 `frontend/wailsjs/**`
- `$env:GOWORK='off'; wails generate module`
  - 通过
- `npm run build`
  - 通过

## 潜在影响与回滚方案

- 潜在影响：
  - 页面不再提供单独的 `Resolve` 入口，用户需经 issue / revoke 主路径触发 authority 解析
  - latest permit 卡的信息密度提高，长 token 的视觉占比会更集中
- 回滚方案：
  - 回退 `frontend/src/pages/PermitIssuance.vue` 与 `frontend/src/pages/PermitIssuance.test.ts`
  - 删除本次 change 归档并回退 `docs/change/README.md`

## 子Agent执行轨迹

- 未使用子Agent
