# Plan - MyFlowHub-Win Packaged Auxiliary Window Fallback

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-popup-build-open`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-popup-build-open`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`:
  - workspace root `D:\project\MyFlowHub3\guide.md` 已阅读
  - 遵守 `AGENTS.md` 与 `$m-autoflow` 约束：实现只在 `worktrees/` 中进行，计划确认前不进入编码
- base/worktree confirmation:
  - 控制面主仓：`D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - 主仓当前存在用户自己的未提交改动：`go.mod`、`myflowhub-mcp.exe`
  - 本轮实现 worktree：`D:\project\MyFlowHub3\worktrees\fix-win-popup-build-open`
  - 本轮只修改 `MyFlowHub-Win`

### Stage 1 - Requirements Analysis
#### Goal
- 修复 `MyFlowHub-Win` 中现有 “Open Window” / 辅助窗口入口在 `wails dev` 可用、但 packaged build 中无法打开的问题。
- 保持现有浏览器 / 开发态的新窗体验，同时保证 packaged runtime 下按钮至少可达目标页面，不出现“点击无效”。

#### Scope
- Must:
  - 统一收口当前所有基于 `window.open(...)` 的辅助窗口入口
  - packaged Wails runtime 中不得继续依赖 `window.open(...)` 作为唯一行为
  - packaged runtime 下点击入口后必须进入对应 `layout: "window"` 路由页面
  - 浏览器 / 开发态原有新窗行为保持不变
  - 失败提示文案保持清晰，不吞掉失败状态
- Optional:
  - 将辅助窗口打开逻辑提炼为共享 helper，减少后续同类页面重复踩坑
  - 为 helper 增加定向单测，覆盖 dev / packaged / blocked 三类路径
- Not in scope:
  - 不实现 Go/Wails 原生多窗口
  - 不改动路由结构或窗口页面 UI
  - 不修改业务页面本身的数据加载逻辑
  - 不调整稳定 requirement/spec 的产品语义

#### Use Cases
- 用户在 Stream 页面点击 `Input Window` / `Output Window`
- 用户在 TopicBus 页面点击 `Open Window`
- 用户在 Showcase 页面点击 Viewer / Editor 窗口入口
- 用户在 Logs、Flow、File 等页面点击现有辅助窗口入口
- 用户在 packaged build 中点击这些按钮时，至少能在当前窗口继续进入目标页面

#### Functional Requirements
1. 所有辅助窗口入口必须走统一的打开策略，而不是各自直接拼接 `window.open(...)`。
2. 在浏览器或 `wails dev` 环境，入口应继续优先尝试打开新窗并聚焦。
3. 在 packaged Wails runtime，入口必须直接降级到当前窗口导航到目标 hash 路由。
4. 若运行时环境识别失败，系统必须回退到最保守且可达的路径，不可保持“点击无反应”。
5. 现有窗口页面 query 参数必须保留，不能因 helper 收口而丢失 `sourceId`、`deliveryId`、`projectId`、`screenId`、`topic`、`targetId` 等参数。
6. 现有 popup blocked 类提示必须保持合理，仅在浏览器新窗被拦截且未进入 packaged fallback 时使用。

#### Non-functional Requirements
- 最小改动面：
  - 只调整入口打开策略，不触碰窗口页面内部业务逻辑
- 可维护性：
  - 运行时判断与降级逻辑集中在一个 helper 中
- 稳定性：
  - helper 必须对非法 route path、运行时探测失败和 `window.open` 失败做显式处理
- 可测试性：
  - 关键分支必须可通过 Vitest 模拟验证

#### Inputs / Outputs
- Inputs:
  - 目标辅助窗口 hash 路由
  - 窗口 name / size
  - 当前运行时环境信息
- Outputs:
  - 浏览器 / dev: 新窗口被打开并聚焦，或显式返回失败
  - packaged runtime: 当前窗口切换到目标路由

#### Edge Cases
- Wails runtime 对 `Environment()` 调用失败
- packaged runtime 中 `window.open` 返回空值或直接无效
- route path 为空或不以 `#/` 开头
- 浏览器中被 popup policy 拦截
- 调用点位于 store 而不是 Vue component 中，无法直接使用 router 实例

#### Acceptance Criteria
1. Stream、TopicBus、Showcase、Logs、Flow、File 的现有辅助窗口入口都改为共享策略。
2. helper 能区分 packaged Wails runtime 与浏览器 / dev 路径。
3. packaged runtime 下不再依赖 `window.open` 才能访问目标页面。
4. 定向 Vitest 与前端 build 通过。
5. `GOWORK=off wails build -debug -skipembedcreate -nopackage` 通过，确保 Wails 打包链仍正常。

#### Risks
- packaged runtime 判定如果过度依赖 URL 特征，后续 Wails 版本升级容易再次失效。
- 若 helper 在 blocked 场景直接改为同窗导航，可能改变浏览器里用户对“新窗失败提示”的预期。
- 某些页面如果隐式依赖“新窗打开后保留原页上下文”，单窗口降级会改变交互路径，但这是比“完全打不开”更安全的退化。

#### Issue List
- 无

### Stage 2 - Architecture Design
#### Overall Solution
- 新增一个前端共享 helper，集中负责：
  - 读取 Wails runtime `Environment()`
  - 判断当前是否为 packaged runtime
  - packaged runtime 下执行当前窗口 hash 导航
  - 浏览器 / dev 下继续走 `window.open(...)`
  - 在新窗被拦截时把失败显式返回给调用者
- 所有现有 `window.open(...)` 调用点迁移到该 helper。

#### Alternatives Considered
- 继续在各页面分别修补 `window.open(...)`:
  - 放弃。重复逻辑太多，后续新页面仍会继续踩坑。
- 在 Go/Wails 侧重做原生多窗口:
  - 放弃。本轮目标是恢复 packaged 可达性，不扩成跨层重构。
- 只在 `window.open(...)` 返回空值时才 fallback:
  - 不作为主方案。packaged runtime 可能不是标准 popup blocked 语义，显式按 runtime 分流更稳。

#### Module Responsibilities
- `frontend/src/lib/auxWindow.ts`:
  - 统一辅助窗口打开策略
  - 运行时探测与 fallback
- 各调用点:
  - 仅负责构造 route path、name、size 和失败提示
- `frontend/src/lib/auxWindow.test.ts`:
  - 覆盖 packaged/dev/blocked 分支
- `frontend/src/pages/Stream.test.ts`:
  - 调整现有对 `window.open` 的直接断言，改为匹配共享 helper 行为

#### Data / Call Flow
1. 页面或 store 构造目标 `#/...` 路由和窗口参数。
2. 调用共享 helper。
3. helper 读取 Wails runtime `Environment()`：
   - 若为 packaged runtime，直接切换当前窗口 hash
   - 否则尝试 `window.open(...)`
4. helper 返回结果给调用者：
   - `opened`
   - `navigated`
   - `blocked`
5. 调用者仅在 `blocked` 时提示 toast。

#### Interface Drafts
- helper 草案:
  - `openAuxWindow({ routePath, name, size }): Promise<"opened" | "navigated" | "blocked">`
- 运行时探测草案:
  - 优先使用 `frontend/wailsjs/runtime` 暴露的 `Environment()`
  - 判定依据：`buildType !== "dev"` 视为 packaged Wails runtime
  - 若 runtime 不存在或探测失败，回退到浏览器 / dev 逻辑

#### Error Handling and Safety
- 非法 route path:
  - helper 直接抛出显式错误
- `Environment()` 失败:
  - 记录为浏览器 / dev 路径，不吞错到页面层
- `window.open(...)` 失败:
  - 返回 `blocked`，由调用者决定 toast
- 当前窗口导航:
  - 仅接受 `#/` 路由，避免误跳外链

#### Performance and Testing Strategy
- helper 内缓存一次 environment 判断结果，避免每次点击重复调用 runtime
- 定向测试:
  - `npm exec vitest run src/lib/auxWindow.test.ts src/pages/Stream.test.ts`
- 回归验证:
  - `npm run build`
  - `$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`

#### Extensibility Design Points
- 后续若真的引入 Go/Wails 原生多窗口，只需要替换 helper 实现，不必再改所有调用点。
- helper 返回枚举结果，便于未来接入埋点、日志或不同 fallback 行为。

#### Issue List
- 无

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前仓内所有辅助窗口入口都在直接调用 `window.open(...)`：
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/pages/Logs.vue`
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/stores/file.ts`
- 当前 Wails 入口只有单个 `options.App`，没有任何原生多窗口实现。
- 已验证构建链本身正常，问题集中在 packaged runtime 的窗口打开策略。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验路由与 impact
- Canonical destination:
  - 稳定需求 / 规格：本轮不改
  - 执行控制面：worktree 根 `plan.md`
  - 完成结果：`docs/change/YYYY-MM-DD_win-popup-build-open.md`
  - 可复用排查经验：若本轮验证出稳定规律，新增 `docs/lessons/wails-packaged-aux-window-fallback.md`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `docs/requirements/stream.md`（仅作为已有独立窗口 UX 背景）
- Related specs:
  - `docs/specs/stream.md`（仅作为已有独立窗口 UX 背景）
- Related lessons:
  - `docs/lessons/README.md`（已检查，当前没有辅助窗口 packaged runtime 的现成 lesson）

#### Executable Task List
- [x] `PWB-1` 新增共享辅助窗口 helper，并实现 packaged runtime fallback
- [x] `PWB-2` 迁移所有现有 `window.open(...)` 调用点到 helper
- [x] `PWB-3` 补定向测试并跑前端 / Wails 构建验证
- [x] `PWB-4` 执行 Stage 3.3 review，确认需求覆盖与回归面

#### Task Details
##### `PWB-1` - 新增共享辅助窗口 helper
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-popup-build-open`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-popup-build-open\plan.md`
- Goal:
  - 把 packaged/dev 的窗口打开差异收口到单点实现
- Files / Modules:
  - `frontend/src/lib/auxWindow.ts`
  - `frontend/wailsjs/runtime/runtime.d.ts`（只读参考，不改）
- Write Set:
  - `frontend/src/lib/auxWindow.ts`
- Acceptance:
  - helper 能返回 `opened` / `navigated` / `blocked`
  - packaged runtime 下直接走当前窗口导航
- Test Points:
  - helper 定向单测
- Rollback:
  - 删除 helper 并恢复各调用点原始实现

##### `PWB-2` - 迁移现有调用点
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-popup-build-open`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-popup-build-open\plan.md`
- Goal:
  - 消除散落的 `window.open(...)`，统一行为
- Files / Modules:
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/pages/Logs.vue`
  - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/stores/file.ts`
- Write Set:
  - 上述 7 个文件
- Acceptance:
  - 现有 route path、query、window name、size 均保留
  - toast 行为只在真正 blocked 时触发
- Test Points:
  - 现有 Stream 页面测试更新通过
- Rollback:
  - 逐文件恢复原始 `window.open(...)` 逻辑

##### `PWB-3` - 测试与构建验证
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-popup-build-open`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-popup-build-open\plan.md`
- Goal:
  - 确认 helper 与迁移改动不破坏前端 / Wails 构建链
- Files / Modules:
  - `frontend/src/lib/auxWindow.test.ts`
  - `frontend/src/pages/Stream.test.ts`
- Write Set:
  - `frontend/src/lib/auxWindow.test.ts`
  - `frontend/src/pages/Stream.test.ts`
- Acceptance:
  - 定向 Vitest 通过
  - `npm run build` 通过
  - `GOWORK=off wails build -debug -skipembedcreate -nopackage` 通过
- Test Points:
  - `npm exec vitest run src/lib/auxWindow.test.ts src/pages/Stream.test.ts`
  - `npm run build`
  - `$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`
- Rollback:
  - 删除新增测试或回退测试断言到旧行为

#### Dependencies
- `frontend/wailsjs/runtime/runtime.d.ts` 中 `Environment()` 与 `EnvironmentInfo.buildType`
- 现有窗口路由:
  - `frontend/src/router/index.ts`

#### Risks and Notes
- helper 若错误识别 dev 为 packaged，会把本应新窗的行为收敛成同窗导航。
- 当前 `plan.md` 所在 worktree 已因验证命令触发生成物状态变化：
  - `frontend/dist/placeholder.txt`
  - `go.mod`
  - 后续实现完成后需再次检查是否仅为生成链副作用，不应把无关生成物混入最终改动。

#### Parallelism Assessment
- 不使用子Agent。
- 原因：
  - 写集很小且强耦合，helper 设计与调用点迁移需要一次性统一落地。
  - 当前平台规则要求只有在用户显式允许委派时才使用子Agent。

#### Issue List
- 无

### Stage 3.2 - Implementation
#### Execution Result
- `PWB-1`
  - 已新增 `frontend/src/lib/auxWindow.ts`。
  - helper 统一校验 `#/` 路由、缓存运行时探测结果，并通过 `Environment().buildType` 区分 packaged 与 dev/browser。
  - packaged runtime 下改为当前窗口 hash 导航；dev/browser 继续走 `window.open(...)`，并显式返回 `opened` / `navigated` / `blocked`。
- `PWB-2`
  - 已迁移以下调用点到共享 helper：
    - `frontend/src/pages/Stream.vue`
    - `frontend/src/pages/TopicBus.vue`
    - `frontend/src/pages/ShowcaseCenter.vue`
    - `frontend/src/pages/Showcase.vue`
    - `frontend/src/pages/Logs.vue`
    - `frontend/src/stores/file.ts`
    - `frontend/src/stores/flowProjects.ts`
  - `frontend/src/pages/File.vue` 与 `frontend/src/pages/Flow.vue` 已改为 `await` store 返回值，避免把 `Promise<boolean>` 当成同步布尔值。
  - `frontend/src/i18n/messages/stores.ts` 补充 Logs blocked toast 文案。
- `PWB-3`
  - 已新增 `frontend/src/lib/auxWindow.test.ts`，覆盖 packaged / dev / blocked / invalid route 分支。
  - 已更新 `frontend/src/pages/Stream.test.ts`，从直接断言 `window.open(...)` 改为断言共享 helper 调用参数。
- 生成链副作用处理：
  - 验证过程中曾脏化 `go.mod` 与 `frontend/dist/placeholder.txt`。
  - 已确认只是生成链副作用，并恢复 worktree 内容，不纳入最终改动。

### Stage 3.3 - Code Review
#### Review Conclusion
- 需求覆盖：通过
  - 所有已知辅助窗口入口都已收口到同一策略。
  - packaged runtime 下不再依赖 `window.open(...)` 才能到达目标 `layout: "window"` 路由。
- 架构与边界：通过
  - 本轮只改前端入口打开策略，不引入 Go/Wails 原生多窗口，不改窗口页面业务逻辑。
  - 运行时探测集中在 helper，后续若引入原生多窗口，可在单点替换实现。
- 回归面：通过
  - 页面与 store 仍保留原有 route、query、window name 与 size。
  - blocked toast 只在浏览器 popup 被拦截时触发，packaged fallback 不误报。
- 验证结果：通过
  - `npm exec vitest run src/lib/auxWindow.test.ts src/pages/Stream.test.ts`
  - `npm run build`
  - `$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`
- 残余风险：
  - 本轮未在真实 packaged GUI 中做点击级人工冒烟。
  - 当前 packaged 探测依赖 `Environment().buildType !== "dev"`；若未来 Wails 改变该语义，需要同步调整 helper。
- 子Agent治理与审计：通过
  - 本轮未派发子Agent。

### Stage 4 - Change Archive
#### Archive Result
- 使用 `$m-docs` 完成归档路由与 impact 校验。
- 已新增：
  - `docs/change/2026-04-13_win-popup-build-open.md`
  - `docs/lessons/wails-packaged-aux-window-fallback.md`
- 已更新：
  - `docs/change/README.md`
  - `docs/lessons/README.md`
  - `plan.md`
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `updated`
