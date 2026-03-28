# 2026-03-29 Win run-dev Proto Replace Path

## 变更背景 / 目标

- 用户从 workspace 根执行 `.\scripts\run-dev.ps1` 启动 Win 时，Wails CLI v2.11.0 在前置 `go mod tidy` 阶段失败。
- 根因不是 `proto-stream-subproto` 缺失，而是 `go.mod` 中的开发态 `replace github.com/yttydcs/myflowhub-proto => ../proto-stream-subproto` 只对某个 Win worktree 路径成立；当启动入口是 `D:\project\MyFlowHub3\repo\MyFlowHub-Win` 时，它会被解析成不存在的 `D:\project\MyFlowHub3\repo\proto-stream-subproto`。
- 本次目标是恢复 root `run-dev.ps1` 对 Win 的开发启动链路，同时保留当前 Stream 模块对开发态 proto worktree 的依赖能力。

## 具体变更内容

### 修改
- `go.mod`
  - 将 `replace github.com/yttydcs/myflowhub-proto` 从 `../proto-stream-subproto` 改为 `../../worktrees/proto-stream-subproto`
  - 让主仓库路径 `repo/MyFlowHub-Win` 与根级 worktree `worktrees/*` 都解析到同一个 `proto-stream-subproto` 目录
- `go.sum`
  - 重新执行 `GOWORK=off go mod tidy` 后，移除了 `github.com/yttydcs/myflowhub-proto v0.1.5` 的远端校验和记录
  - 原因：当前开发态依赖由本地 `replace` 接管
- `docs/lessons/wails-binding-proto-drift.md`
  - 增补“相对 `replace` 只对单一目录层级成立”这一变体
  - 加入错误关键词、快速检查与预防规则
- `docs/lessons/README.md`
  - 更新 lesson 索引，使该错误文本可从目录页直接检索
- `docs/change/README.md`
  - 挂载本次 change 入口
- `plan.md`
  - 记录本轮 workflow 的计划、验证、review 和归档结论

### 不变
- 不修改 `scripts/run-dev.ps1`
- 不修改 `docs/requirements/stream.md` / `docs/specs/stream.md`
- 不移除当前 Stream 模块对开发态 proto worktree 的依赖

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `updated`

## Related requirements

- `docs/requirements/stream.md`

## Related specs

- `docs/specs/stream.md`

## Related lessons

- `docs/lessons/wails-binding-proto-drift.md`

## 对应 plan.md 任务映射

- `RUNDEV-1`
  - 修复 `go.mod` 中的 proto 开发态 `replace` 路径
- `RUNDEV-2`
  - 执行 `go mod tidy` / `wails generate module` 验证
- `RUNDEV-3`
  - 完成 3.3 checklist
- `RUNDEV-4`
  - 更新 `docs/change` / `docs/lessons` 归档

## 经验 / 教训摘要

- 对开发态共享 repo 使用 `replace` 时，路径写法本身就是技术契约；如果仓库同时支持主路径启动和根级 worktree 启动，`replace` 必须在两种模块根下都成立。
- 根脚本暴露出来的是“启动入口目录”和“模块根目录”耦合问题，不应误判为目标 worktree 缺失。
- 这类 Wails / Go 依赖问题仍应优先用 `GOWORK=off` 直接跑 `go mod tidy` 复现，不要只看 `wails dev` 外层包装错误。

## 可复用排查线索

- 症状
  - `wails dev` / `wails generate module` / `go mod tidy` 报 `replacement directory ../proto-stream-subproto does not exist`
  - 错误里出现 `reading ..\proto-stream-subproto\go.mod`
  - 路径落到 `D:\project\MyFlowHub3\repo\proto-stream-subproto`
- 触发条件
  - `go.mod` 中提交了只对某个 worktree 层级成立的相对 `replace`
  - 实际启动入口来自主仓库路径 `repo/MyFlowHub-Win`
  - 当前功能仍依赖开发态 proto worktree，而不是正式 semver
- 关键词
  - `replacement directory ../proto-stream-subproto does not exist`
  - `reading ..\proto-stream-subproto\go.mod`
  - `replace github.com/yttydcs/myflowhub-proto`
  - `run-dev.ps1`
  - `GOWORK=off`
- 快速检查
  - 从实际模块根执行 `Resolve-Path ../../worktrees/proto-stream-subproto`
  - 在 Win 仓库执行 `$env:GOWORK='off'; go mod tidy`
  - 检查 `go.mod` 的 `replace` 是否仍是 `../proto-stream-subproto`

## 关键设计决策与权衡

- 决策：修复 `go.mod`，不改 `scripts/run-dev.ps1`
  - 原因：根因在模块依赖声明；脚本只是稳定暴露了主仓库路径启动这一场景
- 决策：使用相对路径 `../../worktrees/proto-stream-subproto`
  - 原因：同时兼容主仓库路径与根级 worktree，且不引入环境专用绝对路径
- 决策：保留开发态 `replace`
  - 原因：当前 `myflowhub-proto v0.1.5` 仍不包含 `protocol/stream`，直接移除会让 Stream 模块失去编译基线

## 测试与验证方式 / 结果

- 路径解析验证
  - 执行：分别从 `repo/MyFlowHub-Win` 与 `worktrees/fix-win-run-dev-proto-path` 解析 `../../worktrees/proto-stream-subproto`
  - 结果：两者都解析到 `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
- Go 依赖收敛
  - 执行：`$env:GOWORK='off'; go mod tidy`
  - 结果：通过
- Wails bindings
  - 执行：`$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：仍打印既有 `Not found: time.Time` 提示，但退出码为 0；本轮无新增阻塞

## 潜在影响与回滚方案

- 潜在影响
  - `go.sum` 在本地 `replace` 模式下不再保留 `myflowhub-proto v0.1.5` 的远端校验记录
  - 主仓库路径与 worktree 路径现在共享同一个 proto 开发态入口，后续若移动 workspace 布局需同步检查该相对路径
- 回滚方案
  - 回退 `go.mod` 与 `go.sum`
  - 回退 `docs/lessons/wails-binding-proto-drift.md`
  - 回退 `docs/lessons/README.md`
  - 回退 `docs/change/README.md`
  - 删除或回退本归档文件

## 子Agent执行轨迹

- 未使用子Agent
