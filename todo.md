# Plan - MyFlowHub-Win：Showcase 顶部按钮图标化与 Tooltip

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/showcase-toolbar-icons`
- Worktree：`d:\project\MyFlowHub3\worktrees\feat-showcase-toolbar-icons\MyFlowHub-Win`
- Base：`main`
- 当前阶段：`4 归档变更`（已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 将 `Showcase` 页面顶部操作区按钮由文字按钮改为图标按钮；
  - 为每个按钮增加 tooltip（显示原按钮文案）。
- 当前状态：
  - 顶部操作区包含 6 个文字按钮：`Refresh Vars / New Screen / Rename Screen / Delete Screen / Add Event / Add Var`。

## 可执行任务清单（Checklist）

- [x] `SHC-TOPBAR-1` 图标化顶部按钮并保留原行为
  - 目标：
    - 将 6 个按钮切换为 `size="icon"` 图标按钮；
    - 保持点击事件、禁用状态与视觉分组行为不变。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 6 个按钮均为图标按钮；
    - 原有事件处理函数（如 `refreshVars` 等）不变且可触发。
  - 测试点：
    - 进入 `#/showcase` 页面，逐个点击按钮验证行为。
  - 回滚点：
    - 回滚 `Showcase.vue` 顶部按钮模板区块。

- [x] `SHC-TOPBAR-2` 增加 tooltip 与可访问性文本
  - 目标：
    - 每个图标按钮新增 tooltip，文案与原按钮文字一致；
    - 增加 `sr-only` 文本，保障可访问性。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 鼠标悬浮可显示对应文案；
    - 页面无可访问性文本丢失问题。
  - 测试点：
    - 手动悬浮验证：`Refresh Vars / New Screen / Rename Screen / Delete Screen / Add Event / Add Var`。
  - 回滚点：
    - 回滚按钮 `title` 与 `sr-only` 相关改动。

- [x] `SHC-TOPBAR-3` 回归验证与归档准备
  - 目标：
    - 完成最小回归验证；
    - 为阶段 3.3 与 4 产出输入。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 代码可通过前端构建（若环境允许）；
    - 无多余计划外改动。
  - 测试点：
    - `npm --prefix frontend run build`（若脚本存在）；
    - `git diff -- frontend/src/pages/Showcase.vue` 人工审查。
  - 执行结果：
    - `git diff` 人工审查完成；
    - `npm --prefix frontend run build` 失败：环境缺少 `vite` 可执行文件（依赖未安装）。
  - 回滚点：
    - 回滚本次分支全部修改。

## 依赖关系
- `SHC-TOPBAR-2` 依赖 `SHC-TOPBAR-1`。
- `SHC-TOPBAR-3` 依赖 `SHC-TOPBAR-1`、`SHC-TOPBAR-2`。

## 风险与注意事项
- 图标选择需避免语义混淆，尽量使用已有页面常用图标风格。
- 仅修改 `Showcase` 顶部操作区，不影响 widget 区域与其他页面。
- tooltip 采用原生 `title`，无需引入新组件，降低依赖和回归风险。
