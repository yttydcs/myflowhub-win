# 2026-03-20 Win Flow 编辑器适配 call-only 节点模型

## 变更背景 / 目标
SubProto 已收敛为 Flow 新写入仅接受 `kind=call`。Win 编辑器仍生成 `local/exec`，会导致保存被后端拒绝（400）。
本次目标是让 Win 端 Flow 编辑器与后端规范对齐，并保留历史数据读取兼容。

## 具体变更内容

### 新增
- 无。

### 修改
- `frontend/src/stores/flow.ts`
  - `FlowNodeDraft.kind` 改为 `call`。
  - `mapNode` 兼容读取旧 `local/exec/call`，统一映射为 `call`。
  - `addNode` 改为只创建 `call` 节点。
  - `buildSpec/buildGraph` 固定输出 `kind=call`，`target>0` 才写入远程目标。
  - 能力应用入口新增 `applyCallCapability`（保留 `applyExecCapability` 别名以防旧引用）。
- `frontend/src/pages/Flow.vue`
  - 移除节点 `kind` 切换 UI，固定展示 `call`。
  - 移除 Add Node 中的 kind 选择。
  - 合并 local/exec 分叉编辑区为统一 call 编辑区：
    - `target=0` 表示本地调用（执行在 executor 节点）。
    - `target>0` 表示远程调用。
  - 能力选择统一按当前筛选目标（target 或 executor）过滤并回填 `method/target`。

### 删除
- 删除页面层 local 专用能力选择状态与分支逻辑。

## 对应 plan.md 任务映射
- WIN-CALL-1（Store 模型切换）→ `frontend/src/stores/flow.ts`
- WIN-CALL-2（页面统一化）→ `frontend/src/pages/Flow.vue`
- WIN-CALL-3（验证与归档）→ 本文档

## 关键设计决策与权衡
- 采用“前端统一 call + 读取兼容旧 kind”策略：
  - 优点：新保存数据完全符合后端规范；历史数据不丢失可编辑。
  - 代价：页面上不再显式区分 local/exec，改由 `target` 表达调用位置。
- 保留 `applyExecCapability` 别名：
  - 优点：降低重构期间引用断裂风险。
  - 后续：可在全仓迁移完成后移除别名。

## 性能与可扩展性
- 性能：未增加额外网络请求；仍复用 `ExecCapQuerySimple` 单次查询后前端过滤。
- 可扩展：后续新增调用策略可继续挂在 `call` 节点字段，不再扩展 kind 分支。

## 测试与验证方式 / 结果
- 执行：`npm run build`（目录：`frontend/`）
- 结果：失败，原因为既有环境缺失 `../../wailsjs/go/main/App`（与本次改动无关，仓库已有阻塞项）。

## 3.3 Code Review 结论
- 需求覆盖：通过（call-only 写入、旧数据读取兼容、UI 收敛均已实现）。
- 架构合理性：通过（状态模型与页面模型一致，职责清晰）。
- 性能风险：通过（无新增 N+1/重复请求/重计算热点）。
- 可读性一致性：通过（删除双分支后逻辑收敛）。
- 可扩展性与配置化：通过（后续围绕 call 节点扩展即可）。
- 稳定性与安全：通过（仅前端编辑语义调整，不放宽权限/校验）。
- 测试覆盖：有条件通过（受既有 `wailsjs` 缺失影响，构建全量验证未闭环）。

## 潜在影响
- 用户将不再在 UI 上直接选择 `local/exec`，需通过 `target` 理解调用位置。
- 若 executor 未配置且 target=0，本地能力筛选为空（页面已有提示）。

## 回滚方案
1. 回退 `frontend/src/stores/flow.ts` 的 call-only 映射与构图逻辑。
2. 回退 `frontend/src/pages/Flow.vue` 的统一 call 编辑区与节点创建逻辑。
3. 恢复原 local/exec 双分支 UI。
