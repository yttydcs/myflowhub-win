# Plan - MyFlowHub-Win：修复 Showcase 页面空白（Tooltip Provider 注入）

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`fix/showcase-page-blank`
- Worktree：`d:\project\MyFlowHub3\worktrees\fix-showcase-page-blank\MyFlowHub-Win`
- Base：`main`
- 当前阶段：`4 归档变更`（已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 修复 `Showcase` 页面进入后主内容区空白的问题，恢复页面渲染与交互可用性。
- 当前状态（已定位）：
  - 新增 `Tooltip` 封装缺少 `TooltipProvider`；
  - `radix-vue` 在 `TooltipRoot` 初始化时强依赖 Provider 注入，缺失即抛错并中断页面渲染。

## 可执行任务清单（Checklist）

- [x] `SHC-BLANK-1` 修复 Tooltip 注入链路
  - 目标：在 Tooltip 组件内补齐 `TooltipProvider`，保证 `TooltipRoot` 上下文完整。
  - 涉及模块/文件：
    - `frontend/src/components/ui/tooltip/Tooltip.vue`
  - 验收条件：
    - Tooltip 使用点不改即可正常渲染；
    - 不再出现 Provider 注入缺失导致的渲染中断。
  - 测试点：
    - 静态审查：`Tooltip.vue` 引入并包裹 `TooltipProvider`。
  - 回滚点：
    - 回滚该文件改动。

- [x] `SHC-BLANK-2` 最小回归验证
  - 目标：确认修复未引入额外编译错误。
  - 验收条件：
    - 执行前端构建命令并记录结果；
    - 若受项目环境影响失败，明确失败与本次改动的关系。
  - 测试点：
    - `npm --prefix frontend run build`
  - 执行结果：
    - 构建命令执行到 Vite 打包阶段；
    - 失败于项目现有 `wailsjs` 生成物缺失（`Home.vue` 引用 `../../wailsjs/go/session/SessionService` 无法解析），非本次 tooltip 修复引入。
  - 回滚点：
    - 回滚本次修复提交。

- [x] `SHC-BLANK-3` Code Review 与归档
  - 目标：完成 3.3 审查与 4 阶段归档文档。
  - 涉及模块/文件：
    - `docs/change/2026-03-06_win-showcase-page-blank-fix.md`
  - 验收条件：
    - 审查结论覆盖需求/架构/性能/可读性/稳定性/测试。
  - 回滚点：
    - 回滚本 workflow 改动。

## 依赖关系
- `SHC-BLANK-2` 依赖 `SHC-BLANK-1`。
- `SHC-BLANK-3` 依赖 `SHC-BLANK-1`、`SHC-BLANK-2`。

## 风险与注意事项
- TooltipProvider 若放置层级错误（不包裹 TooltipRoot）仍会报注入错误。
- 本修复不触碰 Showcase 业务数据逻辑，避免引入功能性副作用。
