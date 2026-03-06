# Plan - MyFlowHub-Win：File Nodes 选择器交互优化（图标按钮 + 确定回填）

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/file-node-picker-confirm`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-node-picker-confirm`
- Base：`main`
- 当前状态：已完成（阶段 1 → 4 已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 左侧 Nodes 顶部 Add 改图标；
  - 新增左侧 Select 按钮，用树形选择节点；
  - Select Target Node 弹窗改为“选中 + 确定回填”；
  - 移除弹窗冗余文案与 selected target 展示；
  - 右上角关闭改无边框图标。
- 当前状态：
  - Offer 节点选择是“点击即回填”；
  - Nodes 顶部仅有文本 Add 按钮；
  - 选择弹窗仍有多余描述与 selected target 文本。

## 可执行任务清单（Checklist）

- [x] `NODE-PICKER-1` 左侧 Nodes 顶部操作区改造
  - 目标：Add 改图标，并新增 Select 按钮。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - Add 为 icon 按钮；
    - Select 可打开节点树选择弹窗。
  - 测试点：
    - 点击 Add 仍可打开原 Add Node 流程；
    - 点击 Select 可进入选择流程。
  - 回滚点：
    - 回滚 Nodes 顶部按钮区改动。

- [x] `NODE-PICKER-2` Offer 选择弹窗改为“确定回填”
  - 目标：点击树节点仅选中，Confirm 才回填 `Target Node ID`。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 点击节点不立即回填；
    - 点击 Confirm 后回填并关闭弹窗；
    - 本地节点限制仍生效。
  - 测试点：
    - 展开节点时不误触回填；
    - Confirm 后发送正常。
  - 回滚点：
    - 回滚 Offer 选择弹窗交互改动。

- [x] `NODE-PICKER-3` 选择弹窗 UI 清理
  - 目标：移除提示描述、移除 selected target 展示、右上角 Close 改图标无边框。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
    - `frontend/src/components/file/OfferNodeTreePicker.vue`
  - 验收条件：
    - 不再显示 `Click a node in the tree to apply it.`；
    - 不再显示 `Selected target`；
    - 关闭按钮为无边框图标。
  - 测试点：
    - 弹窗视觉与交互符合要求。
  - 回滚点：
    - 回滚上述文件 UI 变更。

- [x] `NODE-PICKER-4` 左侧 Select 弹窗确认选择并切换节点
  - 目标：左侧 Select 使用同一树选择器，Confirm 后切换浏览节点。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 选择并 Confirm 后切换到对应 Node；
    - 远端节点可自动加入 saved nodes（若尚未保存）。
  - 测试点：
    - 选择本地节点/远端节点均可切换；
    - 远端节点首次选择后出现在列表。
  - 回滚点：
    - 回滚左侧 Select 接入。

- [x] `NODE-PICKER-5` 回归验证、Code Review、归档
  - 目标：完成验证、评审与归档。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
    - `frontend/src/components/file/OfferNodeTreePicker.vue`
    - `docs/change/2026-03-06_win-file-node-picker-confirm.md`
  - 验收条件：
    - 关键路径验证通过；
    - 评审结论完整；
    - docs/change 文档完整。
  - 测试点：
    - `go test ./...`；
    - 前端 build 尝试并记录结果。
  - 回滚点：
    - 回滚本次新增/修改文件。

## 依赖关系
- `NODE-PICKER-2/3/4` 依赖 `NODE-PICKER-1`；
- `NODE-PICKER-5` 依赖前置任务完成。

## 风险与注意事项
- 弹窗内展开节点和选中节点均为点击行为，需保留最小误触风险；
- Select 选择远端节点时需避免重复写入 saved nodes。
