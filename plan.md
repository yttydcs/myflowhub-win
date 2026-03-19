# Plan - Win：Flow Exec 能力选择器（按节点+能力联动）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`feat/win-flow-capability-picker`
- Worktree：`d:\project\MyFlowHub3\repo\MyFlowHub-Win\repo\MyFlowHub-Win\worktrees\feat-win-flow-capability-picker`
- Base：`main`

## 项目目标与当前状态
- 目标：
  - 在 Flow 页的 exec 节点编辑中，支持“先选目标节点，再选能力”的可视化配置；
  - 选择能力后自动回填 `spec.target`、`spec.method`（必要时可回填示例 `args`）；
  - 保持现有保存/运行协议不变，兼容已有手填 method 的历史 flow。
- 当前状态：
  - exec 节点能力查询依赖“当前 method 前缀”手动输入触发；
  - capability 列表为单层列表，缺少按 provider node 的联动筛选；
  - 用户需手动维护 target 与 method 的一致性，易出错。

## 范围
- 必须：
  - `kind=exec` 时提供目标节点选择与能力选择联动；
  - 能力选择后自动写入 `selectedNode.target` + `selectedNode.method`；
  - 目标节点切换后，能力列表按该节点过滤；
  - 对未知/历史能力配置保持兼容，不强制清空已有 method；
  - 保存、运行、状态查询链路不回归。
- 可选：
  - 若后端返回可用 schema，默认填充 args 模板（当前先保留接口位）。
- 不做：
  - 不改后端协议字段；
  - 不引入自动选路（cap_query 决策）；
  - 不改变 local 节点语义。

## 可执行任务清单（Checklist）

### WIN-CAP-1 - 能力选择交互建模
- 目标：
  - 在前端建立 provider 节点选项与能力选项的联动计算逻辑。
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
- 验收条件：
  - 可从能力缓存中提取 provider node 列表；
  - 目标节点变化时，能力下拉即时收敛到该节点。
- 测试点：
  - 切换目标节点后 capability 选项变化正确；
  - 无能力时提示文案明确。
- 回滚点：
  - 回退新增计算属性与 watcher。

### WIN-CAP-2 - exec 节点详情改造（节点+能力联动）
- 目标：
  - 以“目标节点 + 能力”为主路径替代手填 method 前缀操作。
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/stores/flow.ts`（如需补充 helper）
- 验收条件：
  - 选择能力后 `target/method` 被写回节点；
  - 已有未知 method 不会被强制覆盖。
- 测试点：
  - 新建 exec 节点，加载能力并选择后可保存；
  - 历史 flow 中 method 不在列表时，仍可显示并运行。
- 回滚点：
  - 回退 exec 节点详情区改动。

### WIN-CAP-3 - 输入校验与错误反馈收敛
- 目标：
  - 对能力查询/应用失败给出清晰反馈，避免 silent failure。
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/stores/flow.ts`
- 验收条件：
  - 能力查询失败保留 toast；
  - 选择无效能力时阻止提交并提示。
- 测试点：
  - 模拟 403/timeout；
  - 选择已失效能力 key。
- 回滚点：
  - 回退新增校验与提示逻辑。

### WIN-CAP-4 - 构建与关键路径验证
- 目标：
  - 验证本次改造对主流程无破坏。
- 涉及模块 / 文件：
  - `frontend/`（构建）
- 验收条件：
  - `npm run build` 通过；
  - Flow 编辑保存/运行/状态查询可用。
- 测试点：
  - 连接 root 节点，创建 exec 节点并配置能力后 Save + Run。
- 回滚点：
  - 清理构建产物并回退改动。

### WIN-CAP-5 - Code Review（强制）
- 目标：
  - 对需求覆盖、架构、性能、可维护性逐项审查。
- 涉及模块 / 文件：
  - 本 workflow 全量改动文件
- 验收条件：
  - Review 清单形成通过/不通过结论。
- 测试点：
  - 与 plan 映射完整。
- 回滚点：
  - 若不通过，回到对应任务修复。

### WIN-CAP-6 - 归档变更（强制）
- 目标：
  - 输出可交接、可审计的变更文档。
- 涉及模块 / 文件：
  - `docs/change/2026-03-19_win-flow-capability-picker.md`
- 验收条件：
  - 包含背景、变更明细、任务映射、权衡、测试、风险与回滚。
- 测试点：
  - 文档可单独用于交接。
- 回滚点：
  - 回退该归档文档。

## 依赖关系
- `WIN-CAP-1` → `WIN-CAP-2` → `WIN-CAP-3` → `WIN-CAP-4` → `WIN-CAP-5` → `WIN-CAP-6`

## 风险与注意事项
- 能力缓存来自远端查询，存在过期风险：需容忍“当前方法不在列表”的状态；
- 节点切换与 capability 选择存在状态同步问题：必须避免 watcher 循环触发；
- 不应把 capability 视图状态写入持久化 spec，避免污染后端协议；
- 大列表能力渲染需避免重复计算：通过 computed 缓存 provider/options 映射。
