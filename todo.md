# Plan - Win Frontend Babel Parser Build Fix

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-missing-babel-parser`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-missing-babel-parser`
- Current Stage: `4 archive`

## Stage Records

### Initialization
- `guide.md`:
  - 仓库根未发现 `guide.md`
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `$m-docs` 的 `requirement-impact.md`、`indexing-rules.md`、`lessons-rules.md`
- base/worktree confirmation:
  - control-plane repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - active execution worktree: `D:\project\MyFlowHub3\worktrees\fix-win-missing-babel-parser`
  - dedicated branch: `fix/win-missing-babel-parser`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 修复 `MyFlowHub-Win` 在 Wails 前端编译阶段因 `@vue/compiler-core` 缺少 `@babel/parser` 而中断的问题，并保持现有 Win 前端构建链可重复执行。

#### Scope
- 必须:
  - 解决 `failed to load config from frontend/vite.config.ts` 后续的 `Cannot find module '@babel/parser'`
  - 保持现有 Vite / Vue / Wails 构建入口不回退
  - 不引入新的环境专属路径或手工步骤作为唯一修复方式
  - 验证前端 build 与 Wails 绑定 / 构建链路
- 可选:
  - 补充构建链排障归档或 lesson，前提是本轮发现了可复用的新排障线索
- 不做:
  - 不改动业务页面行为
  - 不升级无关前端框架版本
  - 不把日常构建强制改成高成本的全量重装流程，除非没有更小且稳定的方案

#### Use Cases
- 开发者执行 `wails build` 时，Wails 在前端 build 阶段不应再因缺少 `@babel/parser` 直接失败。
- 开发者在 fresh worktree 或已有 `node_modules` 的仓库里重新安装依赖后，应能稳定执行 `npm run build`。
- 构建链仍需保留现有 `frontend/dist/placeholder.txt` 占位策略和 Vite 入口策略。

#### Functional Requirements
1. `frontend/package.json` / `frontend/package-lock.json` 必须能稳定提供 `@babel/parser`，使 Vue 编译链在加载 `vite.config.ts` 时不缺依赖。
2. 修复后 `npm install` 之后应能解析 `@vue/compiler-core -> @babel/parser`。
3. 修复不得破坏现有 `vite` 直接入口和 `dist/placeholder.txt` 回写逻辑。
4. 若 `frontend/wailsjs/**` 缺失，验证阶段必须按仓库既有 bootstrap 方式补齐再判断最终 build 结果。

#### Non-functional Requirements
- 采用最小安全变更，优先修复依赖确定性而不是扩大到构建流程重写。
- 不引入每次开发构建都做不必要全量 reinstall 的额外 I/O。
- 变更应与现有 2026-03-21 / 2026-03-25 构建链修复方向一致，继续强调可重复构建。

#### Inputs / Outputs
- 输入:
  - 用户提供的 Wails build 失败日志
  - `frontend/package.json`
  - `frontend/package-lock.json`
  - `frontend/vite.config.ts`
  - 构建链相关 README / change / lessons
- 输出:
  - 稳定声明 `@babel/parser` 的前端依赖配置
  - 通过的前端构建验证
  - 本轮 change 归档，以及必要时的 lesson / index 更新

#### Edge Cases
- 现有 `node_modules` 已损坏或缺包，但锁文件仍未变
- fresh worktree 缺少 `frontend/wailsjs/**`，导致 build 在后续阶段继续失败
- 只修复 parser 缺包但破坏已有 `go:embed` 占位或 Vite 入口策略

#### Acceptance Criteria
1. `frontend/package.json` 和 `frontend/package-lock.json` 明确覆盖本次 parser 缺依赖风险。
2. `npm install` 后 `frontend/node_modules/@babel/parser/package.json` 存在。
3. `$env:GOWORK='off'; wails generate module` 成功。
4. `npm run build` 成功。
5. 如环境允许，`$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage` 成功。

#### Risks
- 若把问题归因到安装方式而不是依赖声明，可能只是暂时绕过，不足以覆盖后续环境。
- 若为追求稳定而把 `frontend:install` 改成高成本全量重装，会增加日常开发 I/O 和等待时间。
- parser 修复完成后，可能暴露下一层既有问题，例如 `frontend/wailsjs/**` 缺失。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 保持现有 `npm install`、Vite 直接入口和 embed placeholder 策略不变，在 `frontend/package.json` 中显式声明 `@babel/parser` 为构建时依赖，并同步更新 `frontend/package-lock.json`。
- 验证时按仓库 README 的 fresh worktree 预检顺序执行：先 `wails generate module` 补齐 `frontend/wailsjs/**`，再跑 `npm run build`，最后视环境执行 `wails build -debug -skipembedcreate -nopackage`。

#### Alternatives Considered
- 方案 A（采用）：显式把 `@babel/parser` 提升为前端 direct build dependency
  - 优点：
    - 变更面最小
    - 直接覆盖当前缺失的关键模块
    - 不改变日常构建的 install 策略
  - 代价：
    - 从“纯传递依赖”转为“显式声明的构建依赖”
- 方案 B：把 `wails.json` 的 `frontend:install` 改为 `npm ci`
  - 优点：
    - 更强的确定性
  - 代价：
    - 每次构建都可能触发高成本全量 reinstall，额外 I/O 明显，不符合最小变更
- 方案 C：新增自定义安装自愈脚本
  - 优点：
    - 可以主动探测缺包和自愈
  - 代价：
    - 复杂度更高，需要额外脚本维护，超出本轮最小修复目标

#### Module Responsibilities
- `frontend/package.json`
  - 声明构建链直接依赖与脚本入口
- `frontend/package-lock.json`
  - 固定 direct dependency 的解析结果，保证 Wails 自动安装时一致
- `README.md`
  - 已有 fresh worktree bootstrap 说明，验证阶段按现有说明执行
- `wails.json`
  - 保持当前安装 / 构建入口，不扩大变更面

#### Data / Call Flow
1. Wails 执行 `frontend:install` 安装前端依赖。
2. Node 解析 `frontend/vite.config.ts` 时加载 `@vitejs/plugin-vue`。
3. Vue 编译链通过 `@vue/compiler-core` / `@vue/compiler-sfc` 解析 SFC。
4. 构建时从 root `node_modules` 解析到显式声明的 `@babel/parser`，避免缺包。
5. 若为 fresh worktree，先由 `wails generate module` 生成 `frontend/wailsjs/**`，再继续 Vite build。

#### Interface Drafts
- 无新增业务接口。
- 依赖接口约束：
  - `@vue/compiler-core@3.5.26` / `@vue/compiler-sfc@3.5.26` 需要 `@babel/parser@^7.28.5`

#### Error Handling and Safety
- 不吞掉安装或构建错误；验证中若出现后续失败点，明确记录是 parser 之后的新阻塞。
- 不修改已有业务代码，避免把构建链修复扩散到运行时行为。

#### Performance and Testing Strategy
- 性能:
  - 保持现有 `npm install`，避免改为每次重装的 `npm ci`
- 验证:
  - `npm install`
  - `Test-Path frontend/node_modules/@babel/parser/package.json`
  - `$env:GOWORK='off'; wails generate module`
  - `npm run build`
  - `$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`

#### Extensibility Design Points
- 若后续 Vue 编译链再出现关键 parser / compiler 缺包，可沿用“将关键构建依赖显式化”的策略，而不是把稳定性完全押在隐式传递依赖上。
- 现有 README / change / lessons 仍作为构建链和 fresh worktree 预检的查阅入口，本轮只在确认有新增复用价值时追加 lesson。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 恢复 Win 仓库在当前环境中的前端构建稳定性，消除 `@babel/parser` 缺失导致的 Wails build 阻塞。
- Current State:
  - 主仓 `frontend/node_modules/@vue/compiler-core` 已存在，但 `frontend/node_modules/@babel/parser` 缺失，和用户报错一致。
  - worktree 中 fresh `npm install` 可恢复 parser，但 fresh worktree build 随后会暴露既有 `frontend/wailsjs/**` 缺失，需要按 README 预检补齐。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: none
- Specs impact: none
- Related requirements:
  - none
- Related specs:
  - none
- Related lessons:
  - `docs/lessons/wails-embed-dist-placeholder.md`
- Related changes:
  - `docs/change/2026-03-21_win-frontend-build-chain.md`
  - `docs/change/2026-03-25_win-embed-dist-placeholder.md`
- Stable truth routing:
  - 本轮不改长期 requirements / specs
  - workflow 执行控制使用 worktree 根 `todo.md`
  - 完成结果归档到 `docs/change`
  - 仅在发现新的可复用排障模式时更新 `docs/lessons`

#### Related Requirements / Specs / Lessons
- Requirements:
  - none
- Specs:
  - none
- Lessons:
  - `docs/lessons/wails-embed-dist-placeholder.md`
- Prior change archives:
  - `docs/change/2026-03-21_win-frontend-build-chain.md`
  - `docs/change/2026-03-25_win-embed-dist-placeholder.md`

#### Executable Task List
- [x] BUILD-DEP-1 显式声明 `@babel/parser` 并同步锁文件
- [x] BUILD-DEP-2 完成 Wails / frontend 构建验证
- [x] REVIEW-DEP-1 完成 3.3 代码复核
- [x] ARCHIVE-DEP-1 归档本轮修复并按需更新索引 / lessons

#### Task Details
##### BUILD-DEP-1 - Parser Dependency Hardening
- Owner:
  - main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\fix-win-missing-babel-parser`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\fix-win-missing-babel-parser\todo.md`
- Goal:
  - 通过最小变更让 Vue 编译链稳定拿到 `@babel/parser`
- Files / Modules:
  - `frontend/package.json`
  - `frontend/package-lock.json`
- Write Set:
  - `frontend/package.json`
  - `frontend/package-lock.json`
- Acceptance:
  - direct dependency 已声明并锁定
  - 不破坏现有 Vite / embed build 脚本
- Test Points:
  - `npm install`
  - `Test-Path frontend/node_modules/@babel/parser/package.json`
- Rollback:
  - 回退 `frontend/package.json` / `frontend/package-lock.json`

##### BUILD-DEP-2 - Build Chain Validation
- Owner:
  - main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\fix-win-missing-babel-parser`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\fix-win-missing-babel-parser\todo.md`
- Goal:
  - 确认 parser 缺依赖修复后，fresh worktree 在补齐 Wails bindings 后能够通过构建
- Files / Modules:
  - generated `frontend/wailsjs/**` only if Wails updates them
- Write Set:
  - generated files only if tool updates them
- Acceptance:
  - `wails generate module` 通过
  - `npm run build` 通过
  - `wails build -debug -skipembedcreate -nopackage` 尽可能通过
- Test Points:
  - `$env:GOWORK='off'; wails generate module`
  - `npm run build`
  - `$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`
- Rollback:
  - 若生成物与本轮修复无关且不稳定，则回退生成物改动

##### ARCHIVE-DEP-1 - Change Archive
- Owner:
  - main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\fix-win-missing-babel-parser`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\fix-win-missing-babel-parser\todo.md`
- Goal:
  - 归档 parser 构建链修复，并决定是否追加 lesson
- Files / Modules:
  - `docs/change/YYYY-MM-DD_*.md`
  - related indexes if needed
  - related lesson only if the investigation reveals a reusable pattern
- Write Set:
  - archive and indexes
  - optional lesson docs
- Acceptance:
  - archive 记录背景、验证、回滚和 lesson 决策
  - 若新增 lesson，则 `docs/lessons/README.md` 同步可检索线索
- Test Points:
  - 手工文档一致性检查
- Rollback:
  - 回退新增归档 / 索引 / lesson 文件

#### Dependencies
- `BUILD-DEP-1` before `BUILD-DEP-2`
- `BUILD-DEP-2` before `REVIEW-DEP-1` and `ARCHIVE-DEP-1`

#### Risks and Notes
- 主仓当前 `node_modules` 已经处于“compiler-core 在、parser 不在”的不一致状态；最终修复必须以锁文件和依赖声明为准，而不是依赖当前目录偶然状态。
- fresh worktree 的 `frontend/wailsjs/**` 缺失是现有已知基线问题，需要按 README 预检补齐，不能误判成 parser 修复失败。
- 本轮不派发子 Agent，写集集中在前端依赖和单条验证链路，且当前会话未获得显式子 Agent 授权。

#### Parallelism Assessment
- No sub-agent dispatch.
- Reason:
  - 写集只涉及一个前端依赖面和同一条验证流水线
  - 子 Agent 无法并行缩短关键路径，反而会增加状态同步成本

#### Issue List
- none

### Stage 3.2 - Implementation
#### Execution Summary
- `BUILD-DEP-1`
  - `frontend/package.json` 已显式加入 `@babel/parser@^7.28.5`
  - `frontend/package-lock.json` 已同步 direct dependency 记录
- `BUILD-DEP-2`
  - 已按 fresh worktree 预检顺序执行 `wails generate module`
  - `npm run build` 已通过
  - `wails build -debug -skipembedcreate -nopackage` 已通过

#### Validation
- `npm install`
  - 通过
- `npm ls @babel/parser`
  - 通过
  - 说明：root 依赖与 Vue 编译链都解析到 `@babel/parser@7.28.5`
- `$env:GOWORK='off'; wails generate module`
  - 通过
  - 说明：`frontend/wailsjs/**` 已补齐
- `npm run build`
  - 通过
- `$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`
  - 通过
  - 说明：前端编译阶段不再报 `Cannot find module '@babel/parser'`

#### Issue List
- none

### Stage 3.3 - Review
#### Review Checklist
- 需求覆盖：通过
  - 直接消除了用户日志里的 `@babel/parser` 缺依赖阻塞点
- 架构合理性：通过
  - 采用 direct dependency 加固，没有扩大到构建流程改写
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 未引入每次构建都做全量 reinstall 的额外 I/O
- 可读性与一致性：通过
  - 改动集中在依赖声明、计划文档和归档文档
- 可扩展性与配置化：通过
  - 后续升级 Vue 编译链时可按显式依赖策略继续维护
- 稳定性与安全：通过
  - 保持现有 Vite 入口、Wails install 入口和 embed placeholder 逻辑不变
- 测试覆盖情况：通过
  - 已完成 `npm install`、`npm ls @babel/parser`、`wails generate module`、`npm run build`、`wails build`
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子 Agent

#### Findings
- none

#### Issue List
- none

### Stage 4 - Archive
#### Archive Outputs
- `docs/change/2026-03-27_win-babel-parser-build.md`
  - 已创建
- `docs/lessons/frontend-build-babel-parser-missing.md`
  - 已创建
- `docs/change/README.md`
  - 已更新索引
- `docs/lessons/README.md`
  - 已更新检索线索

#### Lessons Decision
- `Lessons impact: updated`
- 原因：
  - `failed to load config from vite.config.ts` + `Cannot find module '@babel/parser'` 是可复用、可搜索、容易再次出现的构建链症状

#### Ready For Workflow End
- 是
- 后续如用户确认结束 workflow，可执行合并 / 清理 worktree

阻塞：否
已完成 Stage 4，等待 workflow end confirmation
