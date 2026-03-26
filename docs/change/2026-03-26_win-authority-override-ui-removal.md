# 2026-03-26_win-authority-override-ui-removal

## 变更背景 / 目标

- 在半中心 authority 方案确认后，Win 访问控制控制台仍暴露旧的 `Authority Override` 输入、旧的 authority 回退文案和内部 `authorityReason` 展示。
- 这些内容会把当前运行时 authority 语义误导成“手工覆盖优先”，与现状不符。
- 本轮目标是直接移除 Win GUI 中过时的 Authority Override 暴露，并同步长期 requirements/specs 与测试。

## 具体变更内容

- 更新 `frontend/src/stores/authority.ts`
  - 移除 GUI 侧 `authorityOverride` / `authorityReason` 状态
  - `ResolveAuthority` 固定以 `override=0` 调用，保留自动 authority 解析
  - 简化身份切换与 reset 行为，只保留当前 session 所需的 authority 上下文
- 更新以下页面：
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/pages/RegistrationApprovals.vue`
  - `frontend/src/pages/PermitIssuance.vue`
  - 移除手动 `Authority Override` 输入
  - 移除旧的 resolve rule 文案和内部 reason 展示
  - 只保留当前 authority badge 与既有业务动作
- 更新 `frontend/src/i18n/messages/operations.ts`
  - 删除不再使用的 Authority Override 相关文案
- 更新 `frontend/src/stores/authority_admin.test.ts`
  - 将共享 authority store 断言改为 `ResolveAuthority(sourceId, hubId, 0)`
  - 删除 `authorityReason` 相关断言
- 更新长期文档：
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
  - 明确 authority 由当前 session 自动解析，不再是用户可编辑输入

## Requirements impact: updated

## Specs impact: updated

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-authority-override-removal\docs\requirements\authority-admin-console.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-authority-override-removal\docs\specs\authority-admin-console.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `AUTHUI-1`
  - `frontend/src/stores/authority.ts`
  - `frontend/src/stores/authority_admin.test.ts`
- `AUTHUI-2`
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/pages/RegistrationApprovals.vue`
  - `frontend/src/pages/PermitIssuance.vue`
  - `frontend/src/i18n/messages/operations.ts`
- `DOC-1`
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- `TEST-1`
  - `npm test -- authority_admin`
  - `npm run build`
- `REV-1`
  - `git diff --check`
- `ARC-1`
  - `docs/change/README.md`
  - `docs/change/2026-03-26_win-authority-override-ui-removal.md`

## 经验 / 教训摘要

- 当 authority 语义从“配置优先”转为“运行时下发优先”后，GUI 里保留旧输入框会持续制造错误心理模型，应该优先移除，而不是继续解释旧规则。
- 对 authority 这类共享上下文，最小安全做法是保留自动解析能力，删掉仅用于展示的旧 override / reason 状态，避免误删后端兼容路径。

## 可复用排查线索

- 症状：
  - 访问控制页面仍显示 `Authority 覆盖值`
  - 页面仍出现 `manual override -> authority.node_id -> hubId fallback`
  - authority badge 仍携带内部 `reason`
- 关键词：
  - `authorityOverride`
  - `authorityReason`
  - `Authority Override`
  - `ResolveAuthority`
- 快速检查：
  - `frontend/src/stores/authority.ts`
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/pages/RegistrationApprovals.vue`
  - `frontend/src/pages/PermitIssuance.vue`
  - `frontend/src/i18n/messages/operations.ts`

## 关键设计决策与权衡

- 决策：只移除 Win GUI 过时暴露，不动后端 `ResolveAuthority` 兼容路径
  - 原因：当前 approval / permit 仍是 authority-local 行为，非 root 场景仍依赖后端 authority 目标解析
- 决策：保留手动 `Resolve` 按钮
  - 原因：它仍然是显式刷新当前 authority 上下文的有用操作，不等价于旧的 override 配置

## 测试与验证方式 / 结果

- `npm test -- authority_admin`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-authority-override-removal\frontend`
  - 结果：通过，1 个测试文件 / 4 条断言通过
- `npm run build`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-authority-override-removal\frontend`
  - 结果：通过
  - 备注：worktree 默认缺少 `frontend/wailsjs` 生成文件；验证时临时创建到主 repo `frontend/wailsjs` 的 junction，构建完成后已移除
- `git diff --check`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-authority-override-removal`
  - 结果：通过

## 潜在影响与回滚方案

- 潜在影响：
  - 用户将不再能在 GUI 中手工指定 authority override
  - 页面不再展示内部 authority 解析 reason
  - authority 解析失败时只能通过现有错误提示和 badge 状态定位
- 回滚方案：
  - 回退 `frontend/src/stores/authority.ts`
  - 回退 3 个 authority 页面与 `frontend/src/i18n/messages/operations.ts`
  - 回退 `frontend/src/stores/authority_admin.test.ts`
  - 回退 requirements/specs 中关于自动 authority 解析的更新

## 子Agent执行轨迹

- 未使用子 Agent
