# Win FlowCanvas 节点选择修复（对齐 VueFlow 事件签名）

## 变更背景 / 目标
- 背景：Win → Flow 页面中，选择某个 Flow 后点击画布节点，右侧长期停留在 `Select a node to edit its details.`，无法编辑节点详情；拖拽节点后也不会触发位置写回。
- 目标：恢复画布交互事件（node/edge/pane click、node drag stop），确保：
  - 点击节点可选中并在右侧 `Node Detail` 展示详情；
  - 点击空白可清空选择；
  - 点击边可选中边；
  - 拖拽节点松开后会写回坐标并可保存。

## 具体变更内容
- 修改：`frontend/src/components/flow/FlowCanvas.vue`
  - 对齐 `@vue-flow/core` 事件签名：`nodeClick/edgeClick/nodeDragStop` 回调改为接收单个事件对象（`NodeMouseEvent` / `EdgeMouseEvent` / `NodeDragEvent`），从 `payload.node/payload.edge` 读取数据。
  - 事件监听统一改为 kebab-case：`@node-click/@edge-click/@pane-click/@node-drag-stop`（避免模板属性大小写/风格差异导致监听失效）。

## Plan 任务映射
- `CANVAS-1`：完成。修复 `FlowCanvas.vue` 事件监听与参数解析。
- `CANVAS-2`：待用户执行。按 plan 冒烟验证点击/拖拽行为。
- `CANVAS-3`：完成。归档本文档。

## 关键设计决策与权衡
- 选择：在 `FlowCanvas.vue` 做最小变更对齐第三方库事件结构，不改 store/协议/数据结构。
  - 收益：变更面小、风险低；后续升级 `@vue-flow/core` 时更容易定位事件层问题。
- 选择：监听使用 kebab-case，而不是依赖 camelCase。
  - 收益：更符合 Vue 生态常见写法；对模板大小写折叠更稳健。

## 测试与验证方式 / 结果
- 执行：`npm ci`（`frontend/`）
  - 结果：通过。
- 执行：`npm run build`（`frontend/`）
  - 结果：失败（仓库现状问题，非本次改动引入）。报错：`Could not resolve \"../../wailsjs/go/session/SessionService\"`（缺少生成物 `frontend/wailsjs`）。
- 手工冒烟（需要你执行）：见 `plan.md` 的 `CANVAS-2`。

## Code Review 结论（3.3）
- 需求覆盖：通过。节点/边/空白点击与拖拽结束事件均已修复到可触发路径。
- 架构合理性：通过。修复集中在 view 层组件，不引入跨层耦合。
- 性能风险：通过。仅常量级字段读取与 trim/Number 转换，无新增 I/O 或循环热点。
- 可读性与一致性：通过。使用 `@vue-flow/core` 类型明确事件结构，避免 `any` 误用。
- 可扩展性与配置化：通过。事件解析逻辑集中，后续新增其它事件（doubleClick/contextMenu）可复用同模式。
- 稳定性与安全：通过。不涉及权限/协议/网络请求逻辑变更。
- 测试覆盖情况：部分通过。受仓库现状（缺少 `wailsjs` 生成物）限制，无法取得完整前端构建绿色；已提供可交接的手工冒烟步骤。

## 潜在影响与回滚方案
- 潜在影响：
  - 若未来 `@vue-flow/core` 事件字段命名变化（例如 `payload.node` 结构变更），需同步调整解析。
- 回滚方案：
  - 回滚 `frontend/src/components/flow/FlowCanvas.vue` 的本次提交即可恢复旧行为。

