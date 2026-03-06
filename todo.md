# TODO - Win(VarPool)：Node Vars 查询超时修复（list 不应直连 target=owner）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`fix/nodevars-list-timeout`
- Worktree：`d:\project\MyFlowHub3\worktrees\fix-win-nodevars-list-timeout`
- Base：`main`
- 背景：拓扑 `2 -> 1 -> 9 -> 11(metrics)` 中，节点 2 在 Devices 页点击 Node 11 的 `Vars` 时提示：
  - `Failed to load node variables.`
  - `varpool list: request timed out`

## 项目目标与当前状态

### 目标
1) Node Vars 查询不再超时：在上述拓扑下，Node 11 的变量名列表可在超时前返回（成功或空列表），不弹出 timeout 错误。
2) 行为对齐 `repo/MyFlowHub-Server/docs/3-varstore.md` 的规范：`list` 请求应发送到“当前连接的 Hub/目标节点”（默认 hubId），而不是直发到 owner 节点。

### 当前状态（事实，可审计）
- 前端 `listOwnerNames(ownerId)` 当前调用 `ListSimple(sourceID, targetID=ownerId, { owner: ownerId })`，会构造 `MajorCmd + TargetID=ownerId` 的帧。
- Server 侧 `varstore`（v0.1.1）在 `TargetID!=local` 时会按 header target 前置转发 cmd，导致 list 被转发到叶子节点 owner。
- `metrics node`（Node 11）是 leaf client，只会发送 `varstore set` 到 hub 发布指标，并不会处理 `list/get` 请求，因此不会回包，最终触发超时。

## 可执行任务清单（Checklist）

- [x] `NODEVARS-1`：修正 Node Vars list 的 target 选择（对齐规范）
  - 目标：`listOwnerNames(ownerId)` 改为 `targetID=state.defaultTargetId(hubId)`（直连“直接父/Hub”的缓存），payload 仍为 `{ owner }`。
    - 说明：避免复用可编辑的 `Target ID` 输入，防止用户误设为 owner 再次触发 leaf 超时；需要调试时仍可在 VarPool 页手动 list/get。
  - 涉及文件：
    - `frontend/src/stores/varpool.ts`
  - 验收条件：
    - Node Vars 对话框对任意 owner 执行 Load，不再出现 `request timed out`；
    - owner 无缓存/无变量时返回空列表（不报错）。
  - 测试点：
    - 复现拓扑下：Node 11 Vars 可正常加载（有值或空列表均可接受，关键是“不超时”）；
    - 将 VarPool 页 `Target ID` 改为其它节点（如 9/1），Node Vars 查询仍可用且不超时。
  - 回滚点：
    - revert 本任务提交。

- [x] `NODEVARS-2`：最小回归验证（构建/类型检查）
  - 目标：确保 TS 改动不破坏构建与运行。
  - 验收条件：
    - `cd frontend && npm run build` 通过（若项目已有更合适的 lint/test 命令，可替换）。
  - 回滚点：
    - revert 本任务提交。

- [x] `NODEVARS-3`：Code Review + 归档（强制）
  - 目标：完成审查闭环与变更归档。
  - 涉及文件：
    - `docs/change/2026-03-06_win-varpool-nodevars-list-timeout.md`
  - 验收条件：
    - Review 通过；
    - 归档文档包含：问题原因、方案选择、任务映射、验证方式与回滚方案。

## 依赖关系
- `NODEVARS-1 -> NODEVARS-2 -> NODEVARS-3`

## 风险与注意事项
- Node Vars 的 list 在规范语义上依赖祖先链缓存（`up_set`），若 hub 重启而 metrics node 未重新 publish，可能出现“空列表但不超时”；这属于数据刷新问题，不属于本次“超时”回归范围。
