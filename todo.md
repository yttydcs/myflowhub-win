# Plan - MyFlowHub-Win：Showcase 工具栏 Tooltip 组件化与变量弹窗优化

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/showcase-var-dialog-polish`
- Worktree：`d:\project\MyFlowHub3\worktrees\feat-showcase-var-dialog-polish\MyFlowHub-Win`
- Base：`main`
- 当前阶段：`4 归档变更`（已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  1. 顶部按钮改为“组件 tooltip”，并在正下方显示；
  2. `Add Event/Add Var` 与 screen 级操作分组并增加分割线；
  3. 新增变量默认 target=hub nodeId，且不暴露 target 输入；
  4. `Throttle (ms)` 详细说明放到标题 tooltip；
  5. `On Value/Off Value` 仅在 `mode=switch` 显示，并通过分割线与通用设置区分。
- 当前状态：
  - 顶部按钮使用 `title` 原生 tooltip；
  - 变量弹窗中 target 可编辑；
  - Throttle 说明在常驻文本；
  - On/Off 常驻显示。

## 可执行任务清单（Checklist）

- [x] `SHC-UX-1` 新增 Tooltip 组件并应用到顶部按钮
  - 目标：
    - 引入可复用 Tooltip 组件，tooltip 固定在触发元素下方；
    - 顶部按钮从 `title` 切到组件 tooltip；
    - screen 操作组与 widget 新增组之间加分割线。
  - 涉及模块/文件：
    - `frontend/src/components/ui/tooltip/*`
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 顶部 6 个按钮悬浮显示 tooltip（底部）；
    - 中间分割线可见；
    - 按钮点击行为与禁用态不回归。
  - 测试点：
    - 手工悬浮与点击 `Refresh/New/Rename/Delete/Add Event/Add Var`。
  - 回滚点：
    - 回滚 Tooltip 组件与按钮区改动。

- [x] `SHC-UX-2` 变量 target 固定为 hub 且隐藏输入
  - 目标：
    - 变量新增时 target 固定 `hubId`；
    - 变量表单不再展示 `Target ID` 输入。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 新增变量不需要用户输入 target；
    - hubId 不可用时阻断保存并提示；
    - 主题事件（topic_button）Target 输入保持可编辑。
  - 测试点：
    - 新增 var 成功；hubId 为空时保存报错。
  - 回滚点：
    - 回滚 `submitWidgetDialog` 目标ID处理与表单字段显示逻辑。

- [x] `SHC-UX-3` 模式专属设置分区优化
  - 目标：
    - `Throttle (ms)` 标题支持 tooltip 详细说明；
    - `On/Off` 仅 `switch` 模式显示，并增加分割线与标题。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - Throttle 说明不再常驻文本显示；
    - 非 switch 模式不显示 On/Off；
    - switch 模式显示独立分区（含分割线）。
  - 测试点：
    - 切换 mode 验证字段动态显示。
  - 回滚点：
    - 回滚变量配置区模板改动。

- [x] `SHC-UX-4` 验证、Code Review、归档
  - 目标：
    - 完成差异审查、可执行构建验证（环境允许范围）；
    - 完成阶段 3.3 审查与阶段 4 归档文档。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
    - `frontend/src/components/ui/tooltip/*`
    - `docs/change/2026-03-06_win-showcase-var-dialog-polish.md`
  - 验收条件：
    - 变更与任务映射完整；
    - 归档文档包含验证结果与回滚方案。
  - 测试点：
    - `npm --prefix frontend run build`（若依赖齐全）。
  - 执行结果：
    - `npm --prefix frontend install`：通过；
    - `npm --prefix frontend run build`：失败，报错 `Could not resolve ../../wailsjs/go/auth/AuthService from src/pages/Home.vue`（项目现有环境/生成物问题，非本次变更引入）。
  - 回滚点：
    - 回滚本分支改动。

## 依赖关系
- `SHC-UX-2` 依赖 `SHC-UX-1`（tooltip 组件引入后再统一改模板更稳妥）。
- `SHC-UX-3` 可与 `SHC-UX-2` 并行修改同文件，但提交前统一联调。
- `SHC-UX-4` 依赖全部前置任务。

## 风险与注意事项
- Tooltip 触发器使用 `as-child` 时必须保证单一可交互根节点，避免事件丢失。
- 变量 target 隐藏后需保证编辑旧数据不被意外覆盖。
