# Plan - MyFlowHub-Win：升级 Core 到 v0.4.2 并准备 v0.0.4

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`chore/win-core-v0.4.2-bump`
- Worktree：`d:\project\MyFlowHub3\worktrees\win-core-v0.4.2-bump`
- Base：`main`
- 关联仓库：`MyFlowHub-Core`

## 项目目标与当前状态
- 目标：
  - 将 `MyFlowHub-Win` 的 `myflowhub-core` 依赖从 `v0.4.1` 升级到 `v0.4.2`；
  - 验证桌面端在新 Core 版本下可构建、可测试；
  - 形成 `v0.0.4` 发布准备，供 B 电脑通过 release 包获取最新 Windows RFCOMM dial 修复。
- 当前状态：
  - Win 仓库当前依赖 `github.com/yttydcs/myflowhub-core v0.4.1`；
  - Win release 由 `.github/workflows/release.yml` 在推送 `v*.*.*` tag 后自动构建；
  - B 电脑当前 release 版仍使用旧 Core，无法获得最新 Windows RFCOMM dial 修复。

## 可执行任务清单（Checklist）

### WIN-REL-1 - 对齐依赖基线
- 目标：
  - 升级 `go.mod` / `go.sum` 中的 `myflowhub-core` 到 `v0.4.2`；
  - 保持 `sdk` 与现有业务代码不做额外实现性改动。
- 涉及模块 / 文件：
  - `go.mod`
  - `go.sum`
- 验收条件：
  - 依赖图解析到 `github.com/yttydcs/myflowhub-core v0.4.2`；
  - 不引入计划外业务改动。
- 测试点：
  - `go list -m github.com/yttydcs/myflowhub-core`
  - `go mod tidy`
- 回滚点：
  - 回退 `go.mod` / `go.sum` 提交。

### WIN-REL-2 - 验证桌面端构建与测试
- 目标：
  - 证明升级后仓库仍可通过最小可执行验证；
  - 为 release workflow 提供前置信心。
- 涉及模块 / 文件：
  - `frontend/dist`（仅测试占位，不入库）
  - 全仓 `go test`
- 验收条件：
  - `GOWORK=off go test ./... -count=1` 通过；
  - 必要时补充 `frontend/dist` 占位以满足 `go:embed`。
- 测试点：
  - `GOWORK=off go test ./... -count=1`
- 回滚点：
  - 删除测试占位目录，回退依赖升级提交。

### WIN-REL-3 - 维护发布归档
- 目标：
  - 形成独立 `plan.md` 与 `docs/change`，记录本次 Win 发布准备。
- 涉及模块 / 文件：
  - `plan.md`
  - `docs/change/2026-03-15_win-core-v0.4.2-bump.md`
- 验收条件：
  - 文档覆盖背景、依赖升级、验证、影响与回滚；
  - 明确 `v0.0.4` 目的是携带最新 Windows RFCOMM dial 修复。
- 测试点：
  - 文档内容可供他人接手执行发布。
- 回滚点：
  - 回退文档提交。

### WIN-REL-4 - 准备 v0.0.4 发布
- 目标：
  - 在 Review 通过后基于 `main` 推送 `v0.0.4` tag，触发 GitHub Release workflow。
- 涉及模块 / 文件：
  - Git tag / remote refs
  - `.github/workflows/release.yml`
- 验收条件：
  - `origin/main` 包含本次依赖升级；
  - `v0.0.4` tag 已推送并满足 workflow 校验。
- 测试点：
  - `git show v0.0.4`
  - GitHub Actions 触发记录
- 回滚点：
  - `git tag -d v0.0.4`
  - `git push origin :refs/tags/v0.0.4`

## 依赖关系
- `WIN-REL-1` 完成后进入 `WIN-REL-2`
- `WIN-REL-2` 完成后进入 `WIN-REL-3`
- Review 通过且 `origin/main` 更新后才能执行 `WIN-REL-4`

## 风险与注意事项
- Win 发布 workflow 要求 tag commit 已包含在 `origin/main`，不能直接对临时分支打 release tag；
- 若 Core `v0.4.2` 尚未推送，Win 仓库无法完成依赖升级；
- 本轮只做依赖升级与发布准备，不处理新的 UI/业务逻辑问题。
