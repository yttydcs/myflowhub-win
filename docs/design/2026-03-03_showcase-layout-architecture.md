# 架构设计（V1）- Showcase 独立窗口 + Columns 布局 + 实时同步

## 总体方案

### 目标能力
- 独立访问：新增 `#/showcase-window?screenId=...` Viewer 路由，可通过 `window.open` 打开新窗口。
- 布局（V1）：仅实现 `columns`（响应式列数 + widget `colSpan`）。
- 实时同步：任意窗口保存 ShowcaseConfig 后，所有窗口自动 reload 并刷新订阅。

### 选型与理由
- **配置同步**：采用 Wails `runtime.EventsEmit` 广播 `showcase.config_changed`。
  - 理由：现有日志窗口已证明 Wails 事件可跨窗口分发（`#/log-window`）。
  - 备选：走 `corebus` 新增事件并桥接到 runtime；V1 直接 emit 更简单，后续如需“配置变更原因/差分”再引入 bus 事件。
- **响应式 columns 算法**：用 `ResizeObserver` 监听容器宽度，计算 `columns = clamp(1, maxColumns, floor((w+gap)/(minColumnWidth+gap)))`，再用 CSS Grid 渲染。
  - 理由：纯 CSS `auto-fit/minmax()` 难以“同时自动列数 + 上限 maxColumns”，且需要 JS 才能配合 `colSpan` clamp 的一致性。
  - 备选：CSS-only（`repeat(auto-fit, minmax())`）+ `max-width` hack；可读性/可控性较差。
- **DnD 排序**：V1 使用原生 HTML5 Drag&Drop（拖拽手柄触发），`drop` 时落盘保存。
  - 理由：避免引入额外依赖；满足“必须有排序”且不在 `dragover` 期间写入配置。
  - 备选：引入 sortable 库（如 vuedraggable）；后续若要更强的拖拽体验可再评估。

## 模块职责

### Go（后端）
- `App.ShowcaseConfig()`：读取 profile-scoped `showcase.config`，解析 + normalize 后返回。
- `App.SaveShowcaseConfig(cfg)`：normalize + 持久化；成功后 `runtime.EventsEmit(ctx, "showcase.config_changed", {...})`。
- `normalizeShowcaseConfig/Screen/Widget`：向后兼容（缺字段补默认）；对 `maxColumns/minColumnWidth/colSpan` 做 clamp；保证最小可用配置。

### 前端（Vue）
- `frontend/src/stores/showcase.ts`
  - 负责：类型定义、normalize、load/save、订阅管理（VarPool subscribe/unsubscribe）、TopicBus publish、处理 `showcase.config_changed` 自动 reload。
  - 设计点：store 内维护“订阅的 screenId 模式”（跟随 currentScreenId 或固定 screenId），用于 Viewer 固定 screen 与自动同步。
- `frontend/src/pages/Showcase.vue`（Designer）
  - 负责：Screen/Widget 管理、Columns 配置入口、widget `colSpan` 编辑、DnD 排序、打开 Viewer 窗口。
- `frontend/src/windows/ShowcaseWindow.vue`（Viewer）
  - 负责：读取 `screenId`（query），固定渲染对应 screen；不存在则显示 “Screen not found” 并停止；接收实时同步刷新 UI。

## 数据 / 调用流

### 打开 Viewer
1) Designer 点击 “Open Window”
2) `window.open(base + "#/showcase-window?screenId=...")`
3) Viewer `onMounted`：
   - `showcase.setIdentity(...)`
   - `showcase.load()`
   - `showcase.setFixedScreenId(screenId)`（或等价调用）
   - `showcase.enter()`（订阅对应 screen 的 var widgets）

### 保存与实时同步
1) Designer 调用 `showcase.save()` → Go `SaveShowcaseConfig`
2) Go 持久化成功 → `EventsEmit("showcase.config_changed")`
3) 所有窗口 store `EventsOn("showcase.config_changed")`：
   - coalesce（可选）→ `load()` → `refreshSubscriptions()`（leave/enter）
4) Viewer/Designer UI 随 config 更新重渲染

## 接口草案（仅内部事件 + schema）

### Runtime Event
- `showcase.config_changed` payload（V1 简化）：
  - `{ ts: string, profile: string }` 或 `{}`（前端只需要“发生变化”信号）

### 配置 schema（扩展字段）
- `ShowcaseScreen.layout = { mode:"columns", columns:{ maxColumns:number, minColumnWidth:number, gap:number } }`
- `ShowcaseWidget.layout = { colSpan:number }`

## 错误与安全
- UI 强校验：
  - `maxColumns`：建议 1~12
  - `minColumnWidth`：建议 200~1200（单位 px）
  - `colSpan`：建议 1~maxColumns
- Viewer：
  - `screenId` 缺失/不存在：显示 “Screen not found” 并停止订阅/渲染（不 fallback）。
- 控制权限：
  - 复用既有 `ensureReady()`：未登录（无 selfNodeId）禁止 publish / var set。

## 性能与测试策略
- 性能关键点：
  - DnD 仅在 `drop` 时保存配置，避免频繁 I/O 与订阅抖动。
  - `showcase.config_changed` 触发的 reload 可做最小合并（如短窗口内只 reload 一次），避免“连续保存”导致多次 reload。
  - V1 采用 leave/enter 全量刷新订阅；后续可做订阅 diff 减少抖动。
- 测试策略：
  - Go：新增/更新单测覆盖 normalize（默认值、clamp、向后兼容、emit 不 panic）。
  - 前端：`npm run build`；手工验证多窗口同步、响应式列数、DnD 排序不触发控制误操作。

## 可扩展性设计点
- `layout.mode` 预留：`absolute` / `scale`；V2 只需扩展 schema + 新 renderer。
- `showcase.config_changed` payload 预留：后续可携带 `changedScreenIds` 或 config hash 以支持增量更新。
- widget 层面预留更多 layout 字段（如 `rowSpan`、对齐、分组等）。

