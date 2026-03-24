# Flow Editor Draft Reliability

## Background

- Win Flow 编辑器已经支持 DAG、`call/compose` 和 `Advanced JSON`，但在结构收敛前，窗口壳层长期承担过多职责。
- 编辑器缺少 `dirty-state`、关闭保护和本地草稿恢复；一旦刷新、崩溃或误关窗口，用户容易丢失未保存的图编辑结果。

## Goal

- 为 Win Flow 编辑器提供稳定的“已保存 / 未保存”状态反馈。
- 在不引入后台 autosave 的前提下，降低用户丢稿风险。
- 让 editor shell 的生命周期、保存边界和恢复边界可解释、可审计。

## Scope

### Must

- 编辑器窗口必须显示当前 graph 是否存在未保存改动。
- graph 有未保存改动时，关闭窗口或刷新页面必须触发退出保护。
- 编辑器必须支持基于本地草稿的恢复提示。
- 恢复逻辑必须基于原始 editor graph state，而不是仅基于可通过校验的导出 graph。
- 成功手动保存后，编辑器必须回到 clean 状态，并清理对应本地恢复草稿。
- editor shell 必须把主要 UI 分区收敛为独立组件边界，避免继续堆积到单体窗口文件。

### Optional

- 提供显式“恢复上次草稿”入口，而不只是在加载时弹出确认。
- 后续可再扩展 route 切换保护、显式恢复历史和更细粒度状态展示。

### Out of Scope

- 本轮不引入后台 autosave 到项目持久层。
- 本轮不处理多窗口并发编辑同一 project 的冲突合并。
- 本轮不修改 Flow 运行时协议。

## Scenarios

- 用户编辑 graph 后直接刷新页面。
- 用户编辑 graph 后误关 editor window。
- 用户在图处于中间不合法状态时离开页面，重新打开后希望继续修改。
- 用户恢复本地草稿后，再手动保存项目。

## Functional Requirements

1. 编辑器必须基于当前 graph 内容计算 `dirty-state`，且不得把选择状态或历史游标当作脏变更。
2. graph 有未保存改动时，浏览器关闭或刷新必须触发标准 `beforeunload` 提示。
3. 编辑器必须按 `projectId` 保存本地恢复草稿。
4. 本地恢复草稿必须保留原始 editor state，包括尚未通过完整 graph 校验的中间编辑状态。
5. 恢复提示只应在当前已保存 project 基线与本地草稿记录的基线一致时出现。
6. 若用户拒绝恢复，当前恢复记录必须清理，避免每次打开都重复弹出。
7. 手动保存成功后，`dirty-state` 必须清零，且当前 project 的本地恢复草稿必须清理。
8. editor shell 的主职责必须收敛为：
   - 生命周期与快捷键
   - 项目加载 / 保存
   - 顶层状态装配
   - 顶层事件协调

## Non-functional Requirements

- 可靠性：
  - 本地恢复必须是 best-effort，不得因存储失败阻塞编辑器主路径。
- 性能：
  - 编辑过程不得引入额外后端 I/O。
  - 本地恢复写入应避免每次细小变更都立即同步写盘。
- 可维护性：
  - editor shell 组件边界应稳定，后续 capability / visual form 增强不应再回到单体窗口文件中硬堆。
- 可解释性：
  - 保存、未保存和恢复提示的边界必须能用明确规则描述。

## Edge Cases

- graph 处于中间非法状态，但用户仍希望下次继续编辑。
- 本地恢复记录损坏或 JSON 不合法。
- 本地恢复记录对应的是旧的已保存基线。
- 浏览器 / 容器环境不提供 `localStorage`。
- 用户保存成功后立即关闭窗口，不应再次看到旧草稿恢复提示。

## Acceptance Criteria

1. 编辑器头部可以稳定显示当前 project 的保存状态和最近保存时间。
2. graph 有未保存改动时，刷新或关闭窗口会触发浏览器级保护。
3. 未保存改动在重新打开 editor 时可提示恢复。
4. 恢复后的 graph 即使处于未保存状态，也能继续编辑并再次手动保存。
5. `FlowEditorWindow.vue` 收敛为窗口壳层，主要交互分区移入独立组件。

## Related Specs

- [flow-editor-draft-reliability.md](../specs/flow-editor-draft-reliability.md)
- [flow-editor-visual-form.md](../specs/flow-editor-visual-form.md)

## Related Changes

- [2026-03-24_win-flow-editor-shell-reliability.md](../change/2026-03-24_win-flow-editor-shell-reliability.md)
