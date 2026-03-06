# Plan - MyFlowHub-Win：Offer 目标节点输入框 + 选择按钮弹窗树

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/file-offer-target-picker-dialog`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-offer-target-picker-dialog`
- Base：`main`
- 当前状态：已完成（阶段 1 → 4 已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 将 Offer 弹窗中的目标节点选择改为“输入框 + 选择按钮”；
  - 点击按钮弹出树形节点选择框；
  - 点击节点后回填 NodeID 到输入框。
- 当前状态：
  - 现有 `Target Node ID` 使用内嵌树组件，不符合“输入框+按钮”交互预期。

## 可执行任务清单（Checklist）

- [x] `OFFER-PICKER-UI-1` 需求落地：输入框 + 选择按钮
  - 目标：Offer 表单提供可手输 NodeID 的输入框，右侧提供选择按钮。
  - 涉及模块 / 文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 输入框风格与 `Remote Dir (relative)` 输入一致；
    - 右侧有可点击 `Select` 按钮。
  - 测试点：
    - 手输合法/非法 nodeId 时发送校验正确。
  - 回滚点：
    - 回滚 `File.vue` 中 Offer 目标输入区域。

- [x] `OFFER-PICKER-UI-2` 弹窗树形选择并回填
  - 目标：点击选择按钮打开树形弹窗，点击节点后把 nodeId 回填输入框。
  - 涉及模块 / 文件：
    - `frontend/src/pages/File.vue`
    - `frontend/src/components/file/OfferNodeTreePicker.vue`（复用）
  - 验收条件：
    - 可打开/关闭选择弹窗；
    - 点击可选节点后回填输入框；
    - 本地节点仍不可作为远端目标。
  - 测试点：
    - 选择节点后发送成功；
    - 取消选择不改变原值。
  - 回滚点：
    - 删除选择弹窗接入逻辑，恢复原布局。

- [x] `OFFER-PICKER-UI-3` 回归验证、Code Review、归档
  - 目标：验证关键路径，输出评审结论并归档。
  - 涉及模块 / 文件：
    - `frontend/src/pages/File.vue`
    - `docs/change/2026-03-06_win-file-offer-target-picker-dialog.md`
  - 验收条件：
    - Offer 发送路径不回归；
    - 审查结论完整；
    - docs/change 文档完整。
  - 测试点：
    - `go test ./...`（如受环境限制，至少保证后端回归）；
    - 前端构建尝试并记录结果。
  - 回滚点：
    - 回滚本次新增/修改文件。

## 依赖关系
- `OFFER-PICKER-UI-2` 依赖 `OFFER-PICKER-UI-1`；
- `OFFER-PICKER-UI-3` 依赖前置任务完成。

## 风险与注意事项
- 弹窗嵌套时需确保不会误关闭父 Offer 弹窗；
- 选择后应只回填 nodeId，不应改变 `Remote Dir` 和 `wantHash`。
