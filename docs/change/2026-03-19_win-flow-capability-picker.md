# 2026-03-19 Win Flow Capability Picker

## 变更背景 / 目标
- 背景：Flow 编辑器中 exec 节点此前依赖手填 `method` 再按前缀查询能力，`target` 与 `method` 需要用户手动保持一致，配置成本高且易错。
- 目标：将 exec 节点编辑主路径改为“目标节点 + 能力”联动选择，选择能力后自动回填 `target/method`，并兼容历史 flow 的手工配置。

## 具体变更内容

### 修改
- `frontend/src/pages/Flow.vue`
  - 新增 exec 能力联动计算：
    - provider 节点列表（去重排序）
    - provider + 当前值并集（兼容历史 target）
    - 按 target 过滤的能力列表
  - exec 能力查询逻辑改为全量查询（不再依赖 method 前缀），并在本地按 target 过滤。
  - 新增 `syncSelectedCapabilityByNode`，在节点切换、target/method 变化、能力列表刷新时同步 capability 选中态。
  - exec 节点详情改造为：
    - 目标节点下拉选择（保留手工输入 target 兜底）
    - 能力下拉（按目标过滤）并支持变更即应用
    - 保留 method 手工覆盖输入（兼容未知/历史能力）
  - 失败路径保持 toast 提示，避免 silent failure。

- `frontend/src/stores/flow.ts`
  - `ExecCapabilityRoute.key` 组成从 `provider|method|version` 调整为 `provider|via|method|version`，避免不同路由路径 key 冲突导致下拉选项不稳定。

- `plan.md`
  - 更新为本 workflow 的执行计划（WIN-CAP-1~WIN-CAP-6）。

### 新增 / 删除
- 新增：无
- 删除：无

## 对应 plan.md 任务映射
- `WIN-CAP-1`：`Flow.vue` 中 provider/capability 计算属性与同步逻辑。
- `WIN-CAP-2`：exec 节点详情 UI 改造（节点+能力联动，自动回填）。
- `WIN-CAP-3`：查询失败与应用失败反馈保留；无效 key 不执行。
- `WIN-CAP-4`：执行前端构建验证并记录结果。
- `WIN-CAP-5`：完成本次 Code Review 结论。
- `WIN-CAP-6`：本归档文档。

## 关键设计决策与权衡
- 决策：能力查询走全量 + 前端过滤，而非“按 method 前缀查询”。
  - 原因：支持先选节点后选能力的交互模型，降低用户输入负担。
  - 影响：列表量增大时渲染压力上升；当前通过 computed 缓存与简单过滤控制复杂度。

- 决策：exec 保留 `target` 手工输入与 `method` 手工覆盖。
  - 原因：兼容权限不足、能力未注册、历史配置等场景，避免 UI 改造后不可编辑。
  - 影响：仍允许高级用户覆盖自动选择；主路径仍以能力选择为先。

- 决策：能力 key 增加 `viaNode`。
  - 原因：同 provider/method/version 在不同路由路径下避免 key 冲突。
  - 影响：不改变后端协议，仅前端选项唯一性增强。

## Code Review（3.3）
- 需求覆盖：**通过**
  - exec 节点支持“选目标节点 + 选能力”并自动回填 `target/method`。
- 架构合理性：**通过**
  - 未改协议层；改动局限在页面状态与展示层，边界清晰。
- 性能风险：**通过**
  - 主要是前端数组过滤与去重，无额外网络风暴；查询触发仍显式按钮控制。
- 可读性与一致性：**通过**
  - 逻辑集中在 `Flow.vue`，命名与现有风格一致，保持最小侵入。
- 可扩展性与配置化：**通过**
  - 保留手工兜底，后续可无缝加入 schema 驱动 args 模板。
- 稳定性与安全：**通过**
  - 失败路径保留统一 toast；无新增敏感权限逻辑。
- 测试覆盖情况：**部分通过**
  - 已进行构建验证尝试；受环境生成物缺失影响，无法完成全量 build（见下节）。

## 测试与验证方式 / 结果
- 执行：`cd frontend && npm run build`
  - 结果：失败（非本次改动引入）
  - 报错：`Could not resolve "../../wailsjs/go/auth/AuthService" from "src/pages/Home.vue"`
  - 结论：构建阻塞点为现有 `wailsjs` 生成物缺失；本次修改文件已通过 Vue SFC 解析与 transform 阶段。

## 潜在影响与回滚方案
- 潜在影响：
  - exec 节点编辑交互路径变化（从手填优先变为能力选择优先）。
  - 能力 key 结构调整可能影响依赖旧 key 形态的本地临时状态（不影响持久化协议）。

- 回滚方案：
  1. 回退 `frontend/src/pages/Flow.vue` 本次改动，恢复旧的 method 前缀查询与单层 capability 列表；
  2. 回退 `frontend/src/stores/flow.ts` 的 capability key 变更；
  3. 按需回退 `plan.md` 与本归档文档。
