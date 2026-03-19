# 变更背景 / 目标

- 背景：`Flow` 页此前只给 `exec` 节点提供能力选择器；`local` 节点只能手填 method。
- 目标：给 `local` 节点补齐能力选择器，按当前 executor 节点过滤能力，选择后自动回填 `method`，并保留手工覆盖兜底。

# 具体变更内容

## 修改

- `frontend/src/pages/Flow.vue`
  - 新增 local 能力状态：
    - `selectedLocalCapabilityKey`
    - `executorNodeForLocal`（从 Executor 输入与登录 hub 回退推导）
    - `localCapabilitiesForExecutor`（按 executor + `kind=local` 过滤）
  - 新增 local 能力交互方法：
    - `syncSelectedLocalCapabilityByNode`
    - `loadLocalCapabilities`
    - `applyLocalCapability`
  - 追加 watcher，保证 local 节点切换、method 变化、能力缓存刷新、executor 变化时选中态同步。
  - `Node Detail` 中 `kind=local` UI 改造为：
    - `Refresh Capabilities`
    - `Local Capability` 下拉
    - `Use Selected Capability`
    - `Method (manual override)` 输入框（兼容历史 flow）

- `plan.md`
  - 更新为本次 `WIN-LOCAL-CAP-*` 任务清单。

## 新增 / 删除

- 新增：`docs/change/2026-03-19_win-local-capability-picker.md`
- 删除：无

# 对应计划任务映射

- `WIN-LOCAL-CAP-1`：local 能力联动建模（computed + watcher）
- `WIN-LOCAL-CAP-2`：local 节点详情 UI 能力选择改造
- `WIN-LOCAL-CAP-3`：错误反馈与历史 method 兼容
- `WIN-LOCAL-CAP-4`：构建验证记录
- `WIN-LOCAL-CAP-5`：Code Review 结论
- `WIN-LOCAL-CAP-6`：本归档文档

# 关键设计决策与权衡

- 决策：复用现有 `ExecCapQuerySimple` 查询结果，不新增 local 专用接口。
  - 权衡：前端逻辑增加一层“按 executor 过滤”；但协议不变、接入成本最低。

- 决策：`local` 仍保留 method 手工输入。
  - 权衡：保证兼容历史/未知能力配置，不把 UI 选择器变成硬依赖。

- 决策：local 与 exec 维持独立选中态（`selectedLocalCapabilityKey` vs `selectedCapabilityKey`）。
  - 权衡：避免切换节点类型时状态互相污染，提升可维护性。

# 测试与验证方式 / 结果

- 执行：`cd frontend && npm run build`
  - 结果：失败（非本次改动引入）
  - 报错：`Could not resolve "../../wailsjs/go/main/App" from "src/pages/Home.vue"`
  - 结论：受当前环境缺失 `wailsjs` 生成物影响；本次改动文件已通过 Vite transform 阶段。

# Code Review（3.3）

- 需求覆盖：通过
  - `local` 节点已具备能力查询、选择、应用、手工覆盖。
- 架构合理性：通过
  - 仅前端页面层改动，协议与 store 核心结构未破坏。
- 性能风险：通过
  - 仅做数组过滤与选中同步，无新增高频网络请求。
- 可读性与一致性：通过
  - 与现有 `exec` 能力选择器交互模式保持一致。
- 可扩展性与配置化：通过
  - 后续可在同结构下补 schema 驱动 args 模板。
- 稳定性与安全：通过
  - 错误路径保留 toast；未知 method 不强制清空。
- 测试覆盖：部分通过
  - 完成构建验证尝试；受环境依赖缺失阻塞全量构建完成。

# 潜在影响与回滚方案

- 潜在影响：
  - `local` 节点编辑交互由“手填优先”变为“能力选择优先”；
  - 在 executor 未配置时，local 能力候选为空并展示提示。

- 回滚方案：
  1. 回退 `frontend/src/pages/Flow.vue` local 能力选择相关改动；
  2. 回退 `plan.md`；
  3. 删除本归档文档。
