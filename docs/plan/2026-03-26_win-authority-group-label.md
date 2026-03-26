# Plan - Win Authority Sidebar Group Label

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-authority-group-label`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-authority-group-label`
- Current Stage: `4 archive complete, awaiting workflow end confirmation`

## Stage Records

### Initialization
- `guide.md`:
  - workspace root `D:\project\MyFlowHub3\guide.md` read
  - repo-local `guide.md` not present
- base/worktree confirmation:
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `fix/win-authority-group-label`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-win-authority-group-label`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 将 Win 左侧导航中 Authority 组的中文显示名称从 `准入管理` 调整为更符合用户预期的 `访问控制`。

#### Scope
- 必须:
  - 中文环境下左侧导航组标题由 `准入管理` 改为 `访问控制`
  - 不改变 Authority 组内页面结构、顺序、路由和英文名称
- 可选:
  - 无
- 不做:
  - 不修改后端协议、Wails 服务或前端页面行为
  - 不改动英文环境分组标题 `Authority`
  - 不调整组内子页面中文名称

#### Use Cases
- 中文环境用户希望左侧导航中的组名更接近常见产品术语，降低理解成本。

#### Functional Requirements
- `zh-CN` 环境下，左侧导航组标题必须显示为 `访问控制`
- `en` 环境下，分组标题继续显示为 `Authority`
- Authority 组内的 `Access Policy`、`Registration Approvals`、`Permit Issuance` 入口保持不变

#### Non-functional Requirements
- 保持最小安全改动，只调整必要 i18n 文案
- 不引入新的导航 key、路由耦合或可避免的代码面扩张

#### Inputs / Outputs
- 输入:
  - 导航组 title key: `Authority`
  - 当前 locale
- 输出:
  - `zh-CN` 下显示 `访问控制`
  - `en` 下显示 `Authority`

#### Edge Cases
- 折叠侧边栏时组标题不显示，本次改动不影响现有折叠行为
- 其它 locale 后续若新增，仍复用同一组 key `Authority`

#### Acceptance Criteria
- 中文环境左侧组标题显示为 `访问控制`
- 英文环境左侧组标题仍为 `Authority`
- 不影响组内页面入口和现有导航结构

#### Risks
- 用户消息写成了 `访问控制 控制`；本次按更合理且与上下文一致的 `访问控制` 实现，如需改成其他文案，可继续一行调整

#### Issue List
- 无阻塞

### Stage 2 - Architecture Design
#### Overall Solution
- 保持 `AppShell` 中导航组 title key 为 `Authority`，仅更新 `frontend/src/i18n/messages/shell.ts` 的中文翻译值，并补一条最小测试覆盖该映射。

#### Alternatives Considered
- 方案 A：只改 `shell.ts` 中文翻译
  - 选择理由：改动面最小，不触碰导航结构或其它 locale
- 方案 B：把导航组 key 从 `Authority` 改成新的 key
  - 不选理由：会扩大 i18n 和导航配置的变更面，对本次用户目标没有额外收益

#### Module Responsibilities
- `frontend/src/layout/AppShell.vue`
  - 保持通过 `t(group.title)` 渲染组标题
- `frontend/src/i18n/messages/shell.ts`
  - 维护 `Authority` 在中文环境下的显示文案
- `frontend/src/i18n/messages/shell.test.ts`
  - 对中文映射值增加最小断言

#### Data / Call Flow
- `AppShell` 读取导航组 title `Authority`
- i18n 通过 `t("Authority")` 查到 `shellZhCN["Authority"]`
- 中文界面渲染为 `访问控制`

#### Interface Drafts
- `shellZhCN["Authority"] = "访问控制"`

#### Error Handling and Safety
- 保留既有 key，避免因 key 变更导致其它 locale 或引用方翻译缺失
- 不改路由和 store，避免引入非目标行为回归

#### Performance and Testing Strategy
- 文案替换无额外运行时成本
- 运行前端 Vitest，验证新增映射断言通过

#### Extensibility Design Points
- 保持稳定 key `Authority`，后续新增 locale 时仍可在各自词典中独立决定显示文案

#### Issue List
- 无阻塞

### Stage 3.1 - Planning
#### Project Goal And Current State
- 当前状态：
  - `frontend/src/i18n/messages/shell.ts` 中 `Authority` 的中文文案仍为 `准入管理`
  - 当前没有专门覆盖该词条的测试
- 目标：
  - 以最小改动把该组标题改为 `访问控制`，并补充最小验证

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- `plan.md`：
  - 当前 worktree 根 `plan.md` 作为本次 workflow 控制文档
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`

#### Related Requirements / Specs / Lessons
- Related requirements:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\authority-admin-console.md`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\authority-admin-console.md`
- Related lessons:
  - none

#### Executable Task List
- [x] IMPL-1 更新 Authority 组中文词条
- [x] TEST-1 补最小词条测试并执行验证
- [x] REVIEW-1 完成 3.3 自检
- [x] ARCHIVE-1 归档到 `docs/change`

#### Task Details
##### IMPL-1 - 更新 Authority 组中文词条
- Owner: 主 Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-authority-group-label`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-authority-group-label\plan.md`
- Goal:
  - 将 `Authority` 的中文显示从 `准入管理` 改为 `访问控制`
- Files / Modules:
  - `frontend/src/i18n/messages/shell.ts`
- Write Set:
  - `frontend/src/i18n/messages/shell.ts`
- Acceptance:
  - 中文环境下组标题显示为 `访问控制`
- Test Points:
  - 词条映射值断言
- Rollback:
  - 回退该词条改动

##### TEST-1 - 补最小词条测试并执行验证
- Owner: 主 Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-authority-group-label`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-authority-group-label\plan.md`
- Goal:
  - 为该词条增加最小回归验证
- Files / Modules:
  - `frontend/src/i18n/messages/shell.test.ts`
- Write Set:
  - `frontend/src/i18n/messages/shell.test.ts`
- Acceptance:
  - Vitest 能验证 `Authority` 中文词条为 `访问控制`
- Test Points:
  - `npm test`
- Rollback:
  - 删除新增测试文件

#### Dependencies
- 无外部依赖

#### Risks And Notes
- 由于用户输入存在轻微歧义，本次以 `访问控制` 作为实现假设
- 若后续确认目标文案不同，仅需调整同一词条和测试

#### Parallelism Assessment
- 改动仅涉及单一 i18n 词条与对应测试，写集很小且互相关联，不派发子 Agent
- Owner: 主 Agent

#### Issue List
- 无阻塞

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Change Summary
- `IMPL-1`
  - 将 `frontend/src/i18n/messages/shell.ts` 中 `Authority` 的中文映射由 `准入管理` 改为 `访问控制`
- `TEST-1`
  - 新增 `frontend/src/i18n/messages/shell.test.ts`，对该中文词条增加最小断言

#### Validation Notes
- 目标 worktree 默认没有 `frontend/node_modules`
- 验证时临时创建了一个指向主仓依赖目录的 junction，仅用于本地跑测，测试结束后已移除
- 运行命令：
  - `npm test -- src/i18n/messages/shell.test.ts`

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 仅修改中文环境下的 Authority 组标题，英文环境和组内入口不变
- 架构合理性：通过
  - 保持稳定 key `Authority`，只调整 zh-CN 词条，不扩大导航耦合
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 纯静态文案替换，无新增运行时开销
- 可读性与一致性：通过
  - 词条命名沿用现有 i18n 结构，测试文件直接表达变更目标
- 可扩展性与配置化：通过
  - 继续以 locale 词典承载显示名，便于后续追加其它语言
- 稳定性与安全：通过
  - 不涉及后端协议、权限动作或破坏性状态变更
- 测试覆盖情况：通过
  - 新增最小回归测试，Vitest 已通过
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未派发子 Agent

### Stage 4 - Change Archive
- 使用 `$m-docs` 复核归档路由与 impacts
- `docs/change/2026-03-26_win-authority-group-label.md` 已创建
- `docs/change/README.md` 已更新索引
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`
