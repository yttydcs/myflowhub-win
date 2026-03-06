# Plan - MyFlowHub-Win：Showcase Throttle 文案调整

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`fix/showcase-throttle-label`
- Worktree：`d:\project\MyFlowHub3\worktrees\fix-showcase-throttle-label\MyFlowHub-Win`
- Base：`main`
- 当前阶段：`4 归档变更`（已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 将变量配置标题 `Throttle (ms)` 调整为 `Throttle`；
  - 在 tooltip 内明确单位为毫秒（ms）。
- 当前状态：
  - 界面标签显示 `Throttle (ms)`；
  - tooltip 只说明 0 的语义，未显式说明单位。

## 可执行任务清单（Checklist）

- [x] `SHC-THROTTLE-1` 调整 Throttle 标签与 tooltip
  - 目标：
    - 标题由 `Throttle (ms)` 改为 `Throttle`；
    - tooltip 文案新增“单位毫秒（ms）”说明。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 界面显示 `Throttle`；
    - 悬浮 tooltip 包含单位说明。
  - 测试点：
    - 打开 Add/Edit Var 弹窗，检查 Throttle 标签与 tooltip。
  - 回滚点：
    - 回滚该文件相关文案修改。

- [x] `SHC-THROTTLE-2` 一致性与回归验证
  - 目标：
    - 同步校验错误文案字段名一致；
    - 执行最小构建验证并记录结果。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 校验字段名与 UI 标签一致；
    - 构建命令执行并有结果记录。
  - 测试点：
    - `npm --prefix frontend run build`。
  - 执行结果：
    - `npm --prefix frontend install`：通过；
    - `npm --prefix frontend run build`：失败，报错 `Could not resolve ../../wailsjs/go/session/SessionService from src/pages/Home.vue`（项目现有环境/生成物问题，非本次文案改动引入）。
  - 回滚点：
    - 回滚本次提交。

- [x] `SHC-THROTTLE-3` Code Review 与归档
  - 目标：
    - 完成 3.3 审查输出；
    - 产出 docs/change 归档文档。
  - 涉及模块/文件：
    - `docs/change/2026-03-06_win-showcase-throttle-label.md`
  - 验收条件：
    - 审查结论完整；
    - 归档文档包含任务映射、验证结果、回滚方案。

## 依赖关系
- `SHC-THROTTLE-2` 依赖 `SHC-THROTTLE-1`。
- `SHC-THROTTLE-3` 依赖 `SHC-THROTTLE-1`、`SHC-THROTTLE-2`。

## 风险与注意事项
- 本次仅文案调整，不应改变节流逻辑与默认值。
