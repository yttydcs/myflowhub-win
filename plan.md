# Plan - MyFlowHub-Win Legacy Dialog Height And Scroll Convergence

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-varpool-node-vars-dialog-scroll`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`:
  - workspace root `D:\project\MyFlowHub3\guide.md` 已阅读
  - 遵守 `AGENTS.md` 与 `$m-autoflow` 约束：实现只在 `worktrees/` 中进行，计划确认前不进入编码
- base/worktree confirmation:
  - 控制面主仓：`D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - 主仓当前存在大量未提交改动，不能作为本轮实现目录
  - 本轮实现 worktree：`D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll`
  - 本轮只修改 `MyFlowHub-Win` 前端

### Stage 1 - Requirements Analysis
#### Goal
- 将 `MyFlowHub-Win` 中仍采用旧卡片式 overlay 弹窗的页面统一收敛到“视口内限高 + 内部滚动”的结构。
- 解决 `VarPool Node Variables` 弹窗高度失控问题的同时，把同类旧弹窗一并做一致化优化。

#### Scope
- Must:
  - 盘点 `frontend/src` 中基于 `Overlay` 的卡片式弹窗
  - 保持已经符合“限高 + flex + overflow-hidden + 内部 scroll”模式的弹窗不变
  - 将仍缺少外层高度约束的旧弹窗收敛到统一结构
  - 保持各弹窗现有业务交互、字段、按钮、关闭行为不变
  - 修复滚动区可能裁切 focus ring 的问题
- Optional:
  - 为关键代表性弹窗补结构测试，锁定本轮布局契约
- Not in scope:
  - 不修改共享 `Overlay.vue` 的行为语义
  - 不变更后端、store、Wails binding、分页大小或页面业务流程
  - 不重做页面视觉风格，不把所有弹窗抽成全新共享组件

#### Use Cases
- 用户在 `VarPool`、`Devices`、`File Console`、`Flow`、`Showcase`、`ShowcaseCenter`、`Stream`、`Access Policy` 等页面打开任意旧弹窗
- 表单字段较多、列表较长、窗口高度较小或缩放较大时，弹窗仍留在视口内，主体内容可滚动
- 操作区按钮、关闭按钮和关键上下文信息不被列表或长表单挤出屏幕

#### Functional Requirements
1. 所有纳入本轮的旧式 overlay 卡片弹窗必须具备明确的最大高度约束。
2. 弹窗主体内容超过可用视口时，必须通过内部滚动访问，而不是让整张卡片超出屏幕。
3. 已经采用成熟模式的弹窗保持不动，避免无意义回归。
4. 修复后不得改变各弹窗原有字段、按钮、文案、默认值和 close 行为。
5. 对于带表单 focus ring 的滚动区，内部必须保留足够 padding，避免外圈被 overflow 裁切。
6. 旧弹窗的优化范围必须覆盖所有当前仍明显缺少外层限高的 overlay 卡片，而不是只修个别页面。

#### Non-functional Requirements
- 最小安全改动：
  - 以“局部模板/类名调整”为主，不改共享 overlay 逻辑
- 一致性：
  - 尽量采用仓内已验证的结构模式，例如 `AccessPolicy` / `PermitIssuance` / `RegistrationApprovals` / `FlowMethodPickerDialog`
- 可维护性：
  - 明确区分“已合规弹窗”和“待收敛弹窗”，避免下次重复排查
- 稳定性：
  - 小屏、缩放和长列表/长表单场景下仍保留操作可达性
- 可测试性：
  - 至少补覆盖代表性旧弹窗的结构测试，必要时对复杂弹窗做页面级 smoke

#### Inputs / Outputs
- Inputs:
  - `Overlay` 打开的各类页面/组件弹窗
  - 各弹窗现有的表单、列表、错误态与操作区
- Outputs:
  - 旧弹窗统一具备视口内限高与内部滚动
  - 用户在长内容场景下不再遇到整张卡片超出视口的问题

#### Edge Cases
- 单页内已有局部列表滚动，但整张卡片本体仍可能超出视口
- 弹窗主体是长表单而不是列表，操作按钮仍需固定在底部可达
- 滚动容器内输入框获得焦点时 focus ring 被裁切
- 短弹窗也需兼容统一结构，不能引入多余空白或布局塌陷
- 嵌套 overlay 或复杂 picker 不能因统一结构导致内部列表不可滚

#### Acceptance Criteria
1. 已有成熟结构的弹窗保持不动，只优化当前缺少外层限高的旧弹窗。
2. 纳入本轮的旧弹窗全部具备外层 `max-h + flex-col + overflow-hidden` 或等价结构。
3. 这些弹窗的主体内容区具备内部滚动，不再把整张卡片顶出视口。
4. 至少有代表性测试锁定本轮结构契约，并完成定向前端验证。
5. 手工冒烟时，关键页面的长表单/长列表弹窗在较小视口下仍可操作。

#### Risks
- 若把滚动容器放错位置，可能导致 header / footer 跟着滚动，降低操作效率。
- 批量改多个弹窗时，若结构不一致处理不当，容易出现局部 focus ring 裁切或多余空白。
- 若试图走共享组件重构，改动面会迅速扩大；本轮应避免抽象过度。

#### Issue List
- 无

### Stage 2 - Architecture Design
#### Overall Solution
- 不改共享 `Overlay.vue`，而是在各旧弹窗内部收敛到一套已被仓内验证的壳层模式：
  - 外层卡片：`flex max-h-[85vh] w-full max-w-* flex-col overflow-hidden rounded-2xl border bg-card/95 p-6 text-card-foreground shadow-xl`
  - 主体滚动区：`mt-5 min-h-0 flex-1 overflow-y-auto`
  - 必要时在滚动区内部再加 `px-1 py-1 pr-2`，避免 focus ring 被裁切
- 对已经合规的弹窗不做修改，只把旧弹窗向这套模式靠齐。
- 通过少量代表性测试锁住关键结构，不为每个弹窗都单独补一整套测试。

#### Alternatives Considered
- 修改共享 `Overlay.vue`，让所有弹窗自动限高:
  - 放弃。不同弹窗宽度、主体结构和滚动位置不一致，改共享层风险更高。
- 新增通用 `DialogCard` 组件并迁移所有弹窗:
  - 暂不采用。本轮目标是收敛旧结构，局部模板调整更小更安全。
- 只修 `VarPool Node Variables`:
  - 放弃。盘点后确认仓里还有一批旧弹窗同样缺少外层限高，局部修一处会继续留下同类问题。

#### Module Responsibilities
- 页面/组件中的旧弹窗：
  - 采用统一的限高与滚动布局
- 已合规弹窗：
  - 作为结构参考，不改实现
- 测试文件：
  - 对代表性旧弹窗和已存在测试的页面增加结构断言

#### Data / Call Flow
1. 页面或组件通过 `Overlay` 打开弹窗
2. 业务逻辑、表单状态、store 调用保持不变
3. 仅弹窗模板结构与样式类发生调整：
  - 外层卡片受限高
  - 中部主体承担滚动
  - 页脚操作区保持在底部

#### Interface Drafts
- 不新增新的业务 props / events / store 接口
- 仅在必要时增加少量 `data-*` 选择器，便于测试代表性旧弹窗结构

#### Error Handling and Safety
- 保持现有各弹窗的校验、toast、错误提示和按钮禁用逻辑
- 不因滚动结构调整吞掉错误态、空态或关闭动作
- 对带输入控件的弹窗，滚动区保留内边距，避免 focus ring 裁切

#### Performance and Testing Strategy
- 纯模板/样式层改动，不引入额外请求、循环或共享状态
- 测试策略：
  - 补至少 2 个代表性结构测试
  - 对已有测试页面补充 class / data 断言优先
- 定向验证：
  - `npm exec vitest run src/pages/AccessPolicy.test.ts`
  - `npm exec vitest run src/components/varpool/NodeVarsDialog.test.ts`
  - 如新增其他代表性测试，则一并执行
- 手工验证建议：
  - `VarPool` `Node Variables`
  - `File` `Settings` / `Preview`
  - `Stream` `New Source` / `Control Pair Picker`
  - `Showcase` `Edit Widget`

#### Extensibility Design Points
- 本轮收敛后，可把“旧弹窗壳层改造”作为清晰基线，后续新弹窗按同一结构编写。
- 若未来确实要抽共享 `DialogCard`，可基于本轮收敛后的稳定结构再做，而不是带着多套旧结构直接抽象。

#### Issue List
- 无

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前仓内 overlay 卡片弹窗分成两类：
  - 已合规：
    - `frontend/src/pages/PermitIssuance.vue`
    - `frontend/src/pages/RegistrationApprovals.vue`
    - `frontend/src/pages/AccessPolicy.vue` 中 `roleEditorDialog` / `rolePermissionPickerDialog`
    - `frontend/src/components/flow/editor/FlowMethodPickerDialog.vue`
  - 待收敛：
    - `frontend/src/layout/AppShell.vue` 中 incoming transfer dialog
    - `frontend/src/pages/Devices.vue` 中 `nodeInfo` / `config` / `edit config`
    - `frontend/src/pages/File.vue` 中 `settings` / `download` / `offer` / `offerNodePicker` / `addNode` / `addNodePicker` / `newFolder` / `preview`
    - `frontend/src/pages/Flow.vue` 中 `create` / `meta` / `deploy` / `nodePicker`
    - `frontend/src/pages/Showcase.vue` 中 `layout` / `widget` / `varQuickPick`
    - `frontend/src/pages/ShowcaseCenter.vue` 中 `create` / `rename`
    - `frontend/src/pages/Stream.vue` 中 `source` / `consumer` / `control` / `subscribe`
    - `frontend/src/pages/VarPool.vue` 中 `create variable` / `add watch` / `update variable`
    - `frontend/src/components/varpool/NodeVarsDialog.vue`
    - `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
    - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
    - `frontend/src/pages/AccessPolicy.vue` 中 `nodeOverrideDialog` / `defaultAccessDialog`
- 这些旧弹窗普遍只有 `max-w-*`，缺少外层 `max-h` 约束；部分虽然内部某个列表已经会滚，但整张卡片本体仍可能超出视口。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Canonical destination:
  - 稳定需求 / 规格：本轮不改
  - 执行控制面：worktree 根 `plan.md`
  - 完成结果：`docs/change/2026-04-14_win-dialog-height-scroll-convergence.md`
  - lessons：暂不新增，stage 4 再判断是否值得沉淀为“Win 弹窗结构基线”
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - 未找到与 Win 弹窗统一高度约束直接对应的稳定 requirements 文档
- Related specs:
  - `docs/specs/flow-editor-accessibility.md`（仅作为 editor 弹层可访问性与 overlay 行为参考）
- Related lessons:
  - `docs/lessons/README.md`（已检查，无直接对应的弹窗高度 lesson）
- Related change context:
  - `docs/change/2026-02-24_win-overlay-mask.md`
  - `docs/change/2026-03-03_varpool-dialog-subprefs.md`
  - `docs/change/2026-03-27_win-access-policy-role-dialog-refine.md`

#### Executable Task List
- [x] `DHC-1` 定义旧弹窗统一壳层模式并收敛 `VarPool` / `Devices` / `AccessPolicy` / `AppShell`
- [x] `DHC-2` 收敛 `File` / `Flow` / `Showcase` / `ShowcaseCenter` / `Stream`
- [x] `DHC-3` 收敛 flow editor 旧弹窗组件
- [x] `DHC-4` 补代表性测试并完成定向验证

#### Task Details
##### `DHC-1` - 核心旧弹窗收敛
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll\plan.md`
- Goal:
  - 先覆盖最直接暴露给用户且已确认存在问题的一组核心弹窗
- Files / Modules:
  - `frontend/src/components/varpool/NodeVarsDialog.vue`
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/pages/Devices.vue`
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/layout/AppShell.vue`
- Write Set:
  - 上述 5 个文件
- Acceptance:
  - 这些旧弹窗均采用外层限高 + 主体滚动
  - `AccessPolicy` 的 legacy dialogs 不落后于其 role dialogs
- Test Points:
  - `VarPool Node Variables`
  - `AccessPolicy` 代表性结构断言
- Rollback:
  - 回退上述 5 个文件中的弹窗结构调整

##### `DHC-2` - 页面级旧弹窗批量收敛
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll\plan.md`
- Goal:
  - 把主要业务页面仍停留在旧模式的弹窗统一收敛
- Files / Modules:
  - `frontend/src/pages/File.vue`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/Stream.vue`
- Write Set:
  - 上述 5 个文件
- Acceptance:
  - 这些页面中所有旧式 overlay 卡片均不再只有裸 `max-w`
  - 长表单 / 长列表弹窗保留底部操作区可达
- Test Points:
  - 手工冒烟 `File` / `Showcase` / `Stream`
- Rollback:
  - 回退上述 5 个文件中的弹窗结构调整

##### `DHC-3` - Flow editor 旧弹窗组件收敛
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll\plan.md`
- Goal:
  - 让 editor 相关旧弹窗和现有 `FlowMethodPickerDialog` 结构对齐
- Files / Modules:
  - `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
- Write Set:
  - 上述 2 个文件
- Acceptance:
  - editor 弹窗具备视口内限高与滚动
  - 焦点管理与现有 dialog 语义保持不变
- Test Points:
  - 如有现成测试则补结构断言，否则通过定向 smoke 验证
- Rollback:
  - 回退上述 2 个文件

##### `DHC-4` - 代表性测试与验证
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll\plan.md`
- Goal:
  - 用少量测试锁定结构契约，并确认这轮批量收敛没有破坏前端测试环境
- Files / Modules:
  - `frontend/src/components/varpool/NodeVarsDialog.test.ts`
  - `frontend/src/pages/AccessPolicy.test.ts`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.test.ts`
  - 如需要，再补 1 个代表性页面或组件测试
- Write Set:
  - 以上测试文件
- Acceptance:
  - 代表性结构测试通过
  - 定向 Vitest 命令通过
- Test Points:
  - `npm exec vitest run src/components/varpool/NodeVarsDialog.test.ts`
  - `npm exec vitest run src/pages/AccessPolicy.test.ts`
  - `npm exec vitest run src/components/flow/editor/FlowFieldBindingDialog.test.ts`
- Rollback:
  - 删除新增测试并回退测试断言

#### Dependencies
- 现有 `Overlay` 可承载自定义高度和内部滚动结构
- 已合规弹窗提供可复用的模板参考
- 前端测试环境可运行 Vue 组件测试

#### Risks and Notes
- 本轮是范围扩大后的统一收敛，必须控制在结构层，不扩展到共享 overlay 行为重构。
- `AccessPolicy` 里存在“部分弹窗已合规、部分仍旧式”的混合状态，本轮需要一并对齐。
- 某些短表单弹窗虽然当前内容不长，也会纳入统一壳层，以免后续字段增长再次复发。
- 若实施中发现某一类弹窗更适合独立策略，将返回 `3.1` 更新计划后再继续。

#### Parallelism Assessment
- 不派发子 Agent。
- 原因：虽然文件较多，但都围绕同一类模板结构；串行修改更便于保持模式一致和统一回归。

#### Issue List
- 无

阻塞：否
进入 3.2
禁止派发子Agent

### Stage 3.2 - Implementation
#### Execution Summary
- `DHC-1`
  - `frontend/src/components/varpool/NodeVarsDialog.vue`
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/pages/Devices.vue`
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/layout/AppShell.vue`
  - 将核心旧弹窗统一收敛为 `flex + max-h-[85vh] + overflow-hidden` 外层卡片，并把长内容移入独立滚动区。
- `DHC-2`
  - `frontend/src/pages/File.vue`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/Stream.vue`
  - 对页面级旧式 overlay 卡片做同样壳层收敛，保留现有业务字段、按钮和关闭行为不变。
- `DHC-3`
  - `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - 让 flow editor 旧弹窗对齐已验证的 dialog 结构，同时保留焦点管理和 aria 语义。
- `DHC-4`
  - `frontend/src/components/varpool/NodeVarsDialog.test.ts`
  - `frontend/src/pages/AccessPolicy.test.ts`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.test.ts`
  - 新增/补充代表性结构断言，锁定限高与内部滚动契约。

#### Validation Results
- `npm exec vitest run src/pages/AccessPolicy.test.ts`
  - 通过，`5` 个测试全部通过。
- `npm exec vitest run src/components/flow/editor/FlowFieldBindingDialog.test.ts`
  - 通过，`1` 个测试通过。
- `npm exec vitest run src/components/varpool/NodeVarsDialog.test.ts`
  - 通过，`1` 个测试通过。
- `rg -n -U -P '<Overlay[\\s\\S]{0,900}?<div[^>]*class="(?:(?!max-h-)[^"])*max-w-[^"]*"' frontend/src/pages frontend/src/components`
  - 无命中，说明本轮扫描范围内未残留“overlay 卡片仍只有裸 `max-w`”的旧结构。
- `git diff --check`
  - 通过；仅出现仓库当前的 LF/CRLF 提示，无 patch 级格式错误。

#### Issue List
- 无

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 用户提出的 `VarPool Node Variables` 限高/滚动问题已覆盖，并扩展到同类旧弹窗统一收敛。
- 架构合理性：通过
  - 未修改共享 `Overlay.vue`，仅在各弹窗局部模板中收敛结构，符合最小安全改动。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 本轮仅模板/样式和测试调整，不新增运行时请求或额外计算路径。
- 可读性与一致性：通过
  - 旧弹窗统一到同一壳层模式，代表性弹窗增加稳定 `data-*` 选择器便于后续测试和排查。
- 可扩展性与配置化：通过
  - 形成可复用的局部 dialog shell 基线，后续新弹窗可直接沿用，无需提前抽共享组件。
- 稳定性与安全：通过
  - 保持原有按钮、字段、关闭行为和 editor 焦点管理不变；滚动区补内边距避免 focus ring 裁切。
- 测试覆盖情况：通过
  - 代表性组件和页面测试已补齐并全部通过，覆盖 `NodeVarsDialog`、`AccessPolicy`、`FlowFieldBindingDialog`。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未使用子Agent；所有改动均在单 worktree 内完成并已记录到任务映射。

#### Issue List
- 无

### Stage 4 - Change Archive
#### Archive Outputs
- `docs/change/2026-04-14_win-dialog-height-scroll-convergence.md`
- `docs/lessons/win-legacy-overlay-dialog-scroll-shell.md`
- 更新索引：
  - `docs/change/README.md`
  - `docs/lessons/README.md`

#### Impact Record
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `updated`

#### Ready State
- 当前 worktree 已完成实现、验证、Code Review 和 change/lessons 归档，可以进入“是否结束 workflow”确认。
