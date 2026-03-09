# Plan - MyFlowHub-Win：Showcase 变量快捷选择（订阅 / Mine）

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/showcase-var-quickpick`
- Worktree：`d:\project\MyFlowHub3\worktrees\feat-showcase-var-quickpick\MyFlowHub-Win`
- Base：`main`
- 当前阶段：`4 归档变更`（已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 在变量弹窗 `Variable Name` 右侧增加选择按钮；
  - 点击后弹出变量候选，来源为“当前订阅 + mine”；
  - 选中后自动回填 `Owner NodeID` 与 `Variable Name`。
- 当前状态：
  - 变量名仅可手输，无快捷选择入口。

## 可执行任务清单（Checklist）

- [x] `SHC-QUICKPICK-1` 增加变量快捷选择弹窗与按钮
  - 目标：
    - 在 `Variable Name` 右侧增加选择按钮；
    - 新增弹窗展示候选变量并支持点击回填。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 按钮可见可点；
    - 弹窗可打开/关闭；
    - 点击候选后回填成功。
  - 测试点：
    - 打开 Add/Edit Var，选择候选变量并观察字段回填。
  - 回滚点：
    - 回滚 `Showcase.vue` 相关 UI 代码。

- [x] `SHC-QUICKPICK-2` 复用 VarPool 数据并分组展示
  - 目标：
    - 候选数据来自 `VarPool` 当前缓存；
    - 以 `Subscribed` 与 `Mine` 分组展示，避免口径不一致。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 仅展示有 owner 的有效变量；
    - 订阅与 mine 分组正确。
  - 测试点：
    - 有订阅/有mine/无数据三种场景显示正常。
  - 回滚点：
    - 回滚数据计算逻辑。

- [x] `SHC-QUICKPICK-3` 回归验证、Code Review、归档
  - 目标：
    - 完成最小构建验证并记录结果；
    - 完成阶段 3.3 与阶段 4 文档归档。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
    - `docs/change/2026-03-09_win-showcase-var-quickpick.md`
  - 验收条件：
    - Review 结论完整；
    - 归档文档包含任务映射与回滚方案。
  - 测试点：
    - `npm --prefix frontend run build`（环境允许）。
  - 执行结果：
    - `npm --prefix frontend install`：通过；
    - `npm --prefix frontend run build`：失败，报错 `Could not resolve ../../wailsjs/go/session/SessionService from src/pages/Home.vue`（项目现有环境/生成物问题，非本次改动引入）。
  - 回滚点：
    - 回滚本 workflow 改动。

## 依赖关系
- `SHC-QUICKPICK-2` 依赖 `SHC-QUICKPICK-1`。
- `SHC-QUICKPICK-3` 依赖前置任务完成。

## 风险与注意事项
- 候选来自缓存数据，若缓存为空需提供明确空状态提示。
- 该功能应仅影响 Var Widget 表单，不影响 Topic Widget。
