# Plan - MyFlowHub-Win 移除 Fyne + Windows 打包验收（Wails）

> 归档：2026-02-10。该文件为历史 workflow 计划，不作为当前 worktree 的 active plan。

## 项目目标
1) 在 `MyFlowHub-Win` 中彻底移除 Fyne（代码路径与依赖），确保仅保留 Wails（Vue3/Vite/Tailwind 前端 + Go 后端服务）作为唯一 UI 实现。  
2) 完成 Windows 单文件打包与冒烟验证文档：能启动并连到 `MyFlowHub-Server`，跑通最小功能链路。

## 当前状态（事实）
- 本 worktree 分支：`refactor/remove-fyne`（从 `main` 创建）。
- 原主 worktree（`d:\project\MyFlowHub3\repo\MyFlowHub-Win`）存在：
  - 大量未跟踪文件：Wails 入口文件、`frontend/**`、`internal/services/**`、`internal/storage/**` 等。
  - `go.mod/go.sum` 曾包含 Fyne 相关依赖，且代码里存在 `internal/ui/**`、`internal/app/**` 等 Fyne UI 路径引用（本 workflow 将其彻底移除）。
- Linux 构建验收：本轮暂忽略（仅要求 Windows）。

## 非目标 / 约束
- 不在主 worktree（`repo/MyFlowHub-Win`）直接做实现性改动；所有改动在本 worktree 完成。
- 不提交 `node_modules/`、`dist/` 等构建产物；前端锁文件保留（当前为 `package-lock.json`）。

## 任务清单（Checklist）

### W1 - 将主 worktree 的 WIP 迁移到本 worktree（并清理构建产物）
- 目标：把主 worktree 的未提交修改与未跟踪新增文件完整迁入本 worktree，排除 `node_modules/`、`dist/` 等。
- 涉及模块/文件（预期）：
  - Wails 入口：`main.go`、`wails.json`、`app*.go`
  - 前端：`frontend/**`（排除 `frontend/node_modules`、`frontend/dist`）
  - 后端服务：`internal/services/**`
  - 存储与迁移：`internal/storage/**`
- 验收条件：
  - 本 worktree 的变更集合与主 worktree 目标一致（功能不回退）。
  - `go test ./...` 通过。
- 测试点：
  - `go test ./...`
- 回滚点：
  - 删除本 worktree 目录 + 删除分支 `refactor/remove-fyne`。

### W2 - 彻底移除 Fyne 代码路径
- 目标：删除/下线所有 Fyne UI 相关包与入口，不再编译、引用或保留死代码路径。
- 涉及模块/文件：
  - 预期删除：`internal/ui/**`、`internal/app/**`（以实际为准）
  - 以及任何仅为 Fyne 服务的辅助文件（以 `rg fyne` 搜索结果为准）
- 验收条件：
  - `rg -n "fyne\\.io/fyne|fyne\\.io" -S .` 无命中（确保无 Fyne 依赖/引用）。
  - 若存在遗留数据迁移需要的“旧路径/目录名”字面量（如旧配置目录名），仅允许出现在迁移相关代码中，且不引入任何 Fyne 相关 Go 依赖。
  - `go test ./...` 通过。
- 测试点：
  - `go test ./...`
- 回滚点：
  - 单独提交可回滚；必要时 `git revert`。

### W3 - 移除 Fyne 依赖并收敛 Go 模块
- 目标：从 `go.mod/go.sum` 中移除所有 Fyne 相关依赖，保持最小依赖集合。
- 涉及模块/文件：
  - `go.mod`、`go.sum`
- 验收条件：
  - `go mod tidy` 后 `go.mod` 不再包含任何 Fyne 相关依赖。
  - `go test ./...` 通过。
- 测试点：
  - `go test ./...`
- 回滚点：
  - 回滚该提交即可恢复。

### W4 - Windows 单文件打包 + 冒烟验证步骤（T14 子集）
- 目标：确保 `wails build -platform windows/amd64` 可成功产出（默认生成单个 `.exe`），并补齐“启动并连到 server”的冒烟验证步骤。
- 涉及模块/文件：
  - `wails.json`（如需调整 build profile）
  - 新增 `README.md`（当前仓库无 README；用于 build/run/smoke 指南）
- 冒烟步骤（需落入 README）：
  - 启动 `MyFlowHub-Server`（给出命令与必要配置/端口）。
  - 启动 Win 应用。
  - 在 Home 页面执行 Connect → Login/Register（给出最小字段说明）。
  - 观察“Connected”状态与基础信息/日志无错误。
- 验收条件：
  - `wails build -platform windows/amd64` 成功产出可运行的 `.exe`。
  - README 中包含可复现的 build/run/smoke 步骤。
- 测试点：
  - `wails build ...`（Windows）
  - 手工冒烟：按 README 步骤跑通一次，并记录预期现象。
- 回滚点：
  - 回滚 README/配置提交，不影响核心功能提交。

### W5 - 提交、Code Review（阶段 3.3）与归档（阶段 4）
- 目标：把 W1-W4 形成清晰提交序列；按要求输出 Review；在本 worktree 根目录创建 `docs/change/YYYY-MM-DD_remove-fyne.md` 归档。
- 验收条件：
  - `git status` 干净；提交信息清晰。
  - Review 清单逐项结论明确；不通过则回到阶段 3.2 修正。
  - `docs/change` 文档包含：背景、具体变更、任务映射（W1-W4）、关键决策/权衡、测试结果、影响与回滚。

## 依赖关系
- W1 完成后才能进入 W2/W3。
- W2/W3 完成后才能进入 W4（避免构建被旧依赖干扰）。
- 全部完成后进入阶段 3.3 Review，再进入阶段 4 归档。

## 风险与注意事项
- 主 worktree 存在 `frontend/node_modules`、`frontend/dist`：迁移时必须排除并确保 `.gitignore` 覆盖，避免误提交大体积目录。
- “读取旧 Fyne prefs 数据”的迁移逻辑必须改为不依赖 Fyne 库（只读取旧路径/文件并解析）。
- 若 Wails 生成的 `frontend/wailsjs` 对构建是必需的，需在 W1/W4 中明确是否提交（默认：提交稳定生成物，且不包含构建产物如 dist）。
