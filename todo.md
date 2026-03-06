# Plan - MyFlowHub-Win：Add Remote Node 支持 Select 树选择 Node ID

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/file-add-node-select-picker`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-add-node-select-picker`
- Base：`main`
- 当前状态：已完成（阶段 1 → 4 已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 在 `Add Remote Node` 弹窗中，为 `Node ID` 输入框增加 `Select` 按钮；
  - 点击 `Select` 后打开树形节点选择；
  - 点击 `Confirm` 后将选中 nodeId 回填到 `Node ID` 输入框。
- 当前状态：
  - `Add Remote Node` 仅支持手动输入 nodeId，无树选择入口。

## 可执行任务清单（Checklist）

- [x] `ADD-NODE-SELECT-1` Add Remote Node 输入区增加 Select 按钮
  - 目标：`Node ID` 输入框右侧增加 `Select`。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - `Add Remote Node` 中出现 `Select` 按钮且布局不破坏。
  - 测试点：
    - 打开 Add 对话框后可见按钮。
  - 回滚点：
    - 回滚 Add 对话框输入区改动。

- [x] `ADD-NODE-SELECT-2` 树选择弹窗与 Confirm 回填
  - 目标：点击 `Select` 打开树弹窗，选择节点后 `Confirm` 回填输入框。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 可打开/关闭弹窗；
    - 树中可选节点；
    - Confirm 后回填 `newNodeId`。
  - 测试点：
    - 仅选中不 Confirm 时不回填；
    - Confirm 后输入框变更。
  - 回滚点：
    - 删除 Add 节点选择弹窗逻辑。

- [x] `ADD-NODE-SELECT-3` 回归验证、Code Review、归档
  - 目标：完成关键路径验证和文档归档。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
    - `docs/change/2026-03-06_win-file-add-node-select-picker.md`
  - 验收条件：
    - Add Node 原有保存逻辑不回归；
    - 归档文档完整。
  - 测试点：
    - `go test ./...`；
    - 前端 build 尝试并记录结果。
  - 回滚点：
    - 回滚本次新增/修改文件。

## 依赖关系
- `ADD-NODE-SELECT-2` 依赖 `ADD-NODE-SELECT-1`；
- `ADD-NODE-SELECT-3` 依赖前置任务完成。

## 风险与注意事项
- Add Node 选择应排除本地节点（保持“Remote Node”语义）；
- 仅新增选择入口，不改变既有手动输入能力。
