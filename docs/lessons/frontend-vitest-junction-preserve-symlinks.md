# frontend-vitest-junction-preserve-symlinks

## Summary

- 在 Windows worktree 中，如果 `frontend/node_modules` 通过 junction 指向主仓依赖目录，Vitest 可能在 worktree 下报 `TypeError: Cannot read properties of undefined (reading 'config')`，并且所有测试都显示 `0 test`。
- 同一条测试命令在主仓 `frontend` 可通过，但在 worktree `frontend` 失败时，优先怀疑 junction / realpath 解析导致的 Vitest 运行态错配，而不是产品代码本身回归。
- 该环境下的可靠执行方式是显式使用：
  - `node --preserve-symlinks --preserve-symlinks-main ./node_modules/vitest/vitest.mjs run ...`

## Lookup Hints

- `TypeError: Cannot read properties of undefined (reading 'config')`
- `describe`
- `vitest`
- `junction node_modules`
- `preserve-symlinks`
- `worktree`
- `0 test`

## Symptoms

- Vitest 报 suite failed，但测试文件里显示 `0 test`。
- 报错位置通常指向 `describe(...)` 行，而不是某条具体断言。
- 同一测试在主仓目录能通过，在 worktree 目录失败。

## Impact

- 会把验证阻塞误判成产品代码问题。
- 新增测试和既有测试都会一起失败，难以快速区分回归和环境故障。

## Trigger Conditions

- Windows worktree。
- `frontend/node_modules` 不是本地真实目录，而是 junction 到主仓。
- 使用默认 `node ./node_modules/vitest/vitest.mjs run ...` 或 `npm exec vitest run ...` 执行测试。

## Root Cause

- junction / realpath 解析会让 Vitest runner 与测试文件导入的 `vitest` 包在内部 worker state 上发生错配。
- 结果是 `describe()` 取不到当前测试上下文，进而在读取内部 `config` 时抛错。

## Investigation Trail

- 先在主仓 `frontend` 下运行同一条测试命令，确认是否只在 worktree 复现。
- 用 `npm ls vitest` / `require.resolve('vitest')` 确认 worktree 并没有第二份不同版本的 `vitest`。
- 若 worktree 的 `node_modules` 是 junction，再试一次 `--preserve-symlinks --preserve-symlinks-main`。
- 如果加 flags 后恢复通过，就把问题归类为 worktree 运行环境，而不是产品代码回归。

## Resolution

- 在该类 worktree 中，运行 Vitest 时统一改为：
  - `node --preserve-symlinks --preserve-symlinks-main ./node_modules/vitest/vitest.mjs run <tests...>`
- 如果后续该 worktree 要进行大量前端验证，也可以改为安装一份本地真实 `node_modules`，避免每次都带 flags。

## Prevention / Guardrails

- 用 junction 复用 `node_modules` 时，优先把这条 Vitest 命令记入 plan / change，避免后续重复踩坑。
- 当 worktree 下所有测试同时在 `describe(...)` 处报 `reading 'config'`，先检查依赖目录是否是 junction，再去怀疑代码回归。
- 对“主仓通过、worktree失败”的前端测试，先做路径/依赖解析对比，再决定是否需要重装依赖。

## Related Requirements / Specs / Changes

- Requirements:
  - 无直接产品 requirement 变更
- Specs:
  - 无直接产品 spec 变更
- Changes:
  - `docs/change/2026-04-14_win-showcase-window-session-sync.md`
