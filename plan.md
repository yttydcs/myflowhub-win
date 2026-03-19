# Plan - Win：Flow Local 能力选择器

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`feat/win-capability-local-picker`
- Worktree：`d:\project\MyFlowHub3\repo\MyFlowHub-Win\worktrees\feat-win-capability-local-picker`
- Base：`main`

## 项目目标与当前状态
- 目标：
  - 在 Flow 页的 `local` 节点编辑中支持能力下拉选择；
  - 本地能力选择后自动回填 `spec.method`；
  - 保留手工 method 覆盖，兼容历史 flow。
- 当前状态：
  - `local` 节点仅支持手填 method；
  - 能力选择器仅覆盖 `exec` 节点；
  - 用户无法直接可视化选择“当前执行节点（executor）上的能力”。

## 范围
- 必须：
  - `kind=local` 时新增能力查询与能力下拉；
  - 按当前 Flow executor（顶栏 Executor）过滤本地可选能力；
  - 选择能力后自动回填 `selectedNode.method`；
  - 保留手工 method 输入（manual override）；
  - 不影响 `exec` 节点现有能力选择逻辑。
- 可选：
  - local 能力为空时，给出明确提示与刷新引导。
- 不做：
  - 不改后端协议；
  - 不改 DAG 执行语义；
  - 不做自动生成 args schema 填充。

## 可执行任务清单（Checklist）

### WIN-LOCAL-CAP-1 - local 能力联动建模
- 目标：
  - 在页面层增加 local 能力列表计算与选中同步逻辑。
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
- 验收条件：
  - local 选中节点切换后，能力选中态与 method 保持同步；
  - executor 变化时，local 候选能力随之变化。
- 测试点：
  - 切换节点与切换 executor，候选能力列表正确更新。
- 回滚点：
  - 回退 local 相关 computed/watcher。

### WIN-LOCAL-CAP-2 - local 节点详情 UI 改造
- 目标：
  - 将 local 节点从“纯手填”改为“能力选择优先 + 手工覆盖兜底”。
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
- 验收条件：
  - local 能力可刷新、可选择、可应用；
  - 选择后 method 自动写回并进入历史。
- 测试点：
  - 创建 local 节点，选择能力后 Save + Run 无回归。
- 回滚点：
  - 回退 local UI 区块。

### WIN-LOCAL-CAP-3 - 错误反馈与兼容
- 目标：
  - 收敛 local 能力查询/应用异常路径，保留未知 method 兼容。
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
- 验收条件：
  - 查询失败有 toast；
  - method 不在缓存能力中不强制清空。
- 测试点：
  - 403/timeout 场景提示；
  - 历史 flow 的 method 正常显示。
- 回滚点：
  - 回退新增提示逻辑。

### WIN-LOCAL-CAP-4 - 构建与关键路径验证
- 目标：
  - 验证 local 能力选择器不破坏现有流程。
- 涉及模块 / 文件：
  - `frontend/`
- 验收条件：
  - `npm run build` 完成至现有可达阶段；
  - local/exec 的保存与运行路径行为一致。
- 测试点：
  - local 选能力 + Save + Run；
  - exec 现有能力选择行为回归。
- 回滚点：
  - 清理构建产物并回退改动。

### WIN-LOCAL-CAP-5 - Code Review（强制）
- 目标：
  - 对需求、架构、性能、稳定性、测试覆盖逐项审查。
- 涉及模块 / 文件：
  - 本 workflow 全量改动文件
- 验收条件：
  - Review 清单给出通过/不通过结论。

### WIN-LOCAL-CAP-6 - 归档变更（强制）
- 目标：
  - 输出可交接、可审计的变更文档。
- 涉及模块 / 文件：
  - `docs/change/2026-03-19_win-local-capability-picker.md`
- 验收条件：
  - 包含目标、变更、任务映射、权衡、测试与回滚。

## 依赖关系
- `WIN-LOCAL-CAP-1` → `WIN-LOCAL-CAP-2` → `WIN-LOCAL-CAP-3` → `WIN-LOCAL-CAP-4` → `WIN-LOCAL-CAP-5` → `WIN-LOCAL-CAP-6`

## 风险与注意事项
- local 能力依赖远端缓存，可能出现能力缓存与 method 不一致；
- 需避免 exec/local 两套能力选中状态相互污染；
- executor 输入为空时需使用现有默认解析逻辑，避免过滤误判。
