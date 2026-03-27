# Frontend Build Babel Parser Missing

## Summary
- 在 `MyFlowHub-Win` 的前端构建链里，`@babel/parser` 虽然来自 Vue 编译器的传递依赖，但一旦缺失，Vite 会在加载 `frontend/vite.config.ts` 阶段直接失败。
- 这个症状和 `frontend/wailsjs/**` 缺失、`frontend/dist` embed 占位失效属于不同层级的问题；需要先按错误文本区分再排查。

## Lookup Hints
- 症状：
  - `failed to load config from .../frontend/vite.config.ts`
  - `Error: Cannot find module '@babel/parser'`
- 关键词：
  - `@babel/parser`
  - `@vue/compiler-core`
  - `@vue/compiler-dom`
  - `Cannot find module`
  - `Compiling frontend`
- 快速检查：
  - `frontend/package.json` 是否显式声明 `@babel/parser`
  - `frontend/node_modules/@babel/parser/package.json` 是否存在
  - 是否已经执行 `$env:GOWORK='off'; wails generate module`

## Symptoms
- Wails CLI 在 `Compiling frontend` 阶段失败。
- `npm run build` 在真正进入页面源码打包前就退出。
- 错误堆栈包含：
  - `@vue/compiler-core`
  - `@vue/compiler-dom`
  - `vue`

## Impact
- Win 客户端无法完成 `wails build` 或前端 build。
- fresh worktree 的其它问题会被 parser 缺失遮住，导致排障顺序混乱。

## Trigger Conditions
- `frontend/node_modules/@vue/compiler-core` 已安装，但 `@babel/parser` 缺失。
- 项目依赖只把 parser 留作传递依赖，没有显式声明，而当前环境正好暴露出缺包状态。
- Wails 或手工执行前端 build 时，需要加载 Vue 编译器解析 SFC。

## Root Cause
- 当前 Win 前端构建链对 `@babel/parser` 的存在有硬依赖，但 `frontend/package.json` 之前没有显式声明它。
- 一旦安装结果、目录状态或环境触发 parser 缺失，Vite Vue 插件在解析 `vite.config.ts` 和 SFC 编译链时就会立刻失败。

## Investigation Trail
1. 根据用户日志确认失败点发生在 `frontend/vite.config.ts` 加载阶段，而不是业务页面模块。
2. 检查主仓 `frontend/node_modules`，确认 `@vue/compiler-core` 存在、`@babel/parser` 缺失，与日志一致。
3. 检查 `frontend/package-lock.json`，确认 Vue 编译链确实依赖 `@babel/parser@^7.28.5`。
4. 评估备选方案：
   - 改 `frontend:install` 为 `npm ci`
   - 新增自愈脚本
   - 直接显式声明 parser
5. 选择最小修复：把 `@babel/parser` 提升为 direct build dependency。
6. 用 `wails generate module -> npm run build -> wails build -debug -skipembedcreate -nopackage` 完整验证。

## Resolution
- 在 `frontend/package.json` 的 `devDependencies` 中显式加入 `@babel/parser@^7.28.5`。
- 同步更新 `frontend/package-lock.json`。
- 对 fresh worktree，先执行 `$env:GOWORK='off'; wails generate module` 补齐 `frontend/wailsjs/**`，再继续 build。

## Prevention / Guardrails
- 对“缺了就直接挡住构建”的关键编译依赖，不要完全依赖隐式传递依赖。
- 排查 Win 前端 build 时，先按错误文本区分问题层级：
  - `@babel/parser` 缺失：先看 parser / Vue 编译链
  - `../../wailsjs/...` 无法解析：先跑 `wails generate module`
  - `contains no embeddable files`：先看 `frontend/dist/placeholder.txt`
- 若后续升级 `vue` / `@vue/compiler-*`，同时检查 parser 版本范围。

## Related Docs
- `docs/change/2026-03-21_win-frontend-build-chain.md`
- `docs/change/2026-03-25_win-embed-dist-placeholder.md`
- `docs/change/2026-03-27_win-babel-parser-build.md`
- `docs/lessons/wails-embed-dist-placeholder.md`
