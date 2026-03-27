# 2026-03-27 Win：修复前端构建缺少 `@babel/parser`

## 变更背景 / 目标
- 用户在 `repo\MyFlowHub-Win` 执行 Wails build 时，前端阶段报错：
  - `failed to load config from frontend/vite.config.ts`
  - `Error: Cannot find module '@babel/parser'`
- 错误链路来自 `@vue/compiler-core` / `@vue/compiler-dom` / `vue` 在加载 Vite Vue 插件时缺少 parser。
- 本次目标是用最小变更恢复 Win 前端构建链的确定性，避免继续把关键构建依赖完全押在隐式传递依赖上。

## 具体变更内容
- `frontend/package.json`
  - 将 `@babel/parser@^7.28.5` 显式加入 `devDependencies`，与当前 `@vue/compiler-core@3.5.26` / `@vue/compiler-sfc@3.5.26` 的要求对齐。
- `frontend/package-lock.json`
  - 同步记录 direct dependency，保证 Wails 自动执行前端安装时锁定同一解析结果。
- `todo.md`
  - 记录本轮 `$m-autoflow` 的 requirements / architecture / plan / validation 结果。
- `docs/change/2026-03-27_win-babel-parser-build.md`
  - 归档本轮构建链修复。
- `docs/lessons/frontend-build-babel-parser-missing.md`
  - 固化可复用排障线索和快速检查步骤。

## Requirements impact
`none`

## Specs impact
`none`

## Lessons impact
`updated`

## Related requirements
- `none`

## Related specs
- `none`

## Related lessons
- `docs/lessons/frontend-build-babel-parser-missing.md`
- `docs/lessons/wails-embed-dist-placeholder.md`

## 对应 plan.md 任务映射
- `BUILD-DEP-1`
  - 显式声明 `@babel/parser` 并同步锁文件。
- `BUILD-DEP-2`
  - 执行 `wails generate module`、`npm run build`、`wails build -debug -skipembedcreate -nopackage` 验证。
- `REVIEW-DEP-1`
  - 完成 3.3 复核，确认方案最小且未扩大构建流程写集。
- `ARCHIVE-DEP-1`
  - 写入本归档并更新 lessons / indexes。

## 经验 / 教训摘要
- `@babel/parser` 虽然是 Vue 编译链的传递依赖，但在当前 Win 构建链里，它已经是“缺了就直接挡住 Vite config 加载”的关键构建依赖。
- 对 Wails 项目而言，修复 parser 缺失后，fresh worktree 仍可能继续暴露 `frontend/wailsjs/**` 缺失；两者需要按顺序排查，不能混为一个问题。
- 若已有 `npm install` / Vite 入口 / embed placeholder 策略都稳定，优先做 direct dependency 加固，比改成每次全量 `npm ci` 更小、更符合当前仓的 I/O 约束。

## 可复用排查线索
- 症状
  - `failed to load config from .../frontend/vite.config.ts`
  - `Error: Cannot find module '@babel/parser'`
  - require stack 包含 `@vue/compiler-core`、`@vue/compiler-dom`、`vue`
- 触发条件
  - `frontend/node_modules/@vue/compiler-core` 已安装，但 `frontend/node_modules/@babel/parser` 缺失
  - Wails 在 `Compiling frontend` 阶段执行 `npm run build`
- 关键词
  - `@babel/parser`
  - `@vue/compiler-core`
  - `failed to load config from vite.config.ts`
  - `Compiling frontend`
  - `Cannot find module`
- 快速检查
  - 检查 `frontend/package.json` 是否显式声明 `@babel/parser`
  - 检查 `frontend/node_modules/@babel/parser/package.json` 是否存在
  - 若是 fresh worktree，再检查是否已经执行 `$env:GOWORK='off'; wails generate module`

## 关键设计决策与权衡
- 采用“显式 direct build dependency”而不是改 `wails.json` 为 `npm ci`
  - 原因：当前目标是修复单个关键缺包；改成 `npm ci` 会把每次构建成本提高到全量 reinstall，扩大 I/O。
- 不新增自定义安装自愈脚本
  - 原因：当前通过一个显式依赖即可覆盖症状，额外脚本会提高维护复杂度。
- 保持 `frontend/package.json` 里的 Vite 直接入口和 `dist/placeholder.txt` 回写逻辑不变
  - 原因：这两条策略已经由前序构建链修复验证过稳定，不应与本轮 parser 问题耦合修改。

## 测试与验证方式 / 结果
- `npm install`
  - 结果：通过
- `npm ls @babel/parser`
  - 结果：通过
  - 说明：root `devDependency` 与 `vue -> @vue/compiler-core` 的传递依赖都能解析到 `@babel/parser@7.28.5`
- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 说明：fresh worktree 所需的 `frontend/wailsjs/**` 已补齐
- `npm run build`
  - 结果：通过
- `$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`
  - 结果：通过
  - 说明：用户原始的前端编译阻塞点已消失，生成 `build/bin/myflowhub-win.exe`

## 潜在影响
- `frontend` 新增一个显式构建依赖，后续如升级 Vue 编译链，需一起审视 `@babel/parser` 版本范围是否仍匹配。
- 本轮不会自动修复任何与 `frontend/wailsjs/**`、`go:embed all:frontend/dist` 无关的其它环境问题；这些仍按既有 README / lessons 排查。

## 回滚方案
- 回退以下文件即可回到本轮修复前状态：
  - `frontend/package.json`
  - `frontend/package-lock.json`
  - `todo.md`
  - `docs/change/2026-03-27_win-babel-parser-build.md`
  - `docs/lessons/frontend-build-babel-parser-missing.md`
  - `docs/change/README.md`
  - `docs/lessons/README.md`

## 子Agent执行轨迹
- 本轮未使用子 Agent。
