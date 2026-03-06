# Plan - MyFlowHub-Win：移除左侧 Nodes 顶部 Select 按钮

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/file-remove-nodes-select`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-remove-nodes-select`
- Base：`main`
- 当前状态：已完成（阶段 1 → 4 已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 移除左侧 `Nodes` 顶部 `Select` 按钮；
  - 同步删除该按钮关联的节点选择状态、函数与弹窗。
- 当前状态：
  - `Nodes` 顶部仍存在 `Select` 按钮，并关联 `browserNodePicker` 全套逻辑。

## 可执行任务清单（Checklist）

- [x] `REMOVE-NODES-SELECT-1` 删除 Nodes 顶部 Select 按钮
  - 目标：仅保留 Add 图标按钮。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 左侧 `Nodes` 顶部不再显示 Select。
  - 回滚点：
    - 回滚按钮区改动。

- [x] `REMOVE-NODES-SELECT-2` 删除 browserNodePicker 相关逻辑
  - 目标：移除状态、函数与弹窗节点。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 无 `browserNodePickerOpen/browserPickerTargetId/openBrowserNodePicker/onBrowserNodePicked/confirmBrowserNodePicker` 引用。
  - 回滚点：
    - 回滚逻辑删除改动。

- [x] `REMOVE-NODES-SELECT-3` 回归验证、Code Review、归档
  - 目标：验证无回归，完成归档。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
    - `docs/change/2026-03-06_win-file-remove-nodes-select.md`
  - 验收条件：
    - 文件页面可正常编译运行（环境允许范围内）；
    - docs/change 完整。
  - 测试点：
    - `go test ./...`；
    - 前端 build 尝试并记录结果。
  - 回滚点：
    - 回滚本次新增/修改文件。

## 依赖关系
- `REMOVE-NODES-SELECT-2` 依赖 `REMOVE-NODES-SELECT-1`；
- `REMOVE-NODES-SELECT-3` 依赖前置任务完成。

## 风险与注意事项
- 删除逻辑时避免误删 `Add Remote Node` 里的 `Select` 功能。
