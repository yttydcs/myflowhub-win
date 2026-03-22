# 变更归档：Win Flow Call Visual Form

## 变更背景 / 目标

- 现有 Win Flow 编辑器已经支持 `call/compose`、`args_template`、`inputs` 和 `Advanced JSON`，但 `call` 节点普通路径仍要求用户直接编辑底层 JSON。
- 本轮目标是为 `call` 节点提供字段级普通模式，并让前置节点输出、trigger、flow meta、run meta 能以图形化方式绑定到后续字段。
- 对超出普通模式表达范围的节点，编辑器必须保守退回 `Advanced JSON`，不能静默吞掉高级配置。

## Requirements / Specs Impact

- Requirements impact：`updated`
- Specs impact：`updated`
- Lessons impact：`none`
- Related requirements：
  - `D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`
- Related specs：
  - `D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`

## 具体变更内容（新增 / 修改 / 删除）

### 新增

- `frontend/src/stores/flow_json_pointer.ts`
  - 提供 JSON Pointer 的 token escape、append、读写、删除和叶子字段收集能力。
- `frontend/src/stores/flow_method_schemas.ts`
  - 定义普通模式字段 schema 类型，并为 `varstore::get/set/revoke` 提供本地 override schema。
- `frontend/src/stores/flow_schema_resolver.ts`
  - 解析本地 override schema 与 capability `input_schema`，输出统一视觉 schema。
- `frontend/src/stores/flow_visual_form.ts`
  - 提供普通模式兼容性分析、字段状态构建、literal/binding 写回 helper。

### 修改

- `frontend/src/stores/flow.ts`
  - capability query 请求增加 `include_schema=true`。
  - `ExecCapabilityRoute` 补齐 `defaultTimeoutMs`、`permissions`、`tags`、`inputSchema`、`outputSchema`。
  - 新增当前节点视觉 schema 解析、schema 默认值应用、字段级 literal/binding 读写 API。
  - 对字段 binding 写回增加祖先节点与 JSON Pointer 校验。
- `frontend/src/windows/FlowEditorWindow.vue`
  - `call` 节点普通模式切换为字段表单。
  - 每个字段右侧提供 `fx` 入口，可配置祖先节点结果、trigger、flow meta、run meta 绑定。
  - 普通模式不再直接暴露 `Args Template (JSON)` 与整块 `Input Bindings` 给 `call` 节点。
  - 当方法无 schema 或节点当前配置超出普通模式能力时，表单区仅显示不可用提示，并引导使用 `Advanced JSON`。
  - `compose` 节点继续保留原有 `template + input bindings` 编辑路径。
- `plan.md`
  - 更新 workflow 当前阶段和任务完成状态，补充执行结果摘要。

### 删除

- 无运行时契约删除。

## 对应 plan / todo 任务映射

- `CALLFORM-1`
  - `frontend/src/stores/flow.ts`
- `CALLFORM-2`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flow_json_pointer.ts`
  - `frontend/src/stores/flow_method_schemas.ts`
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow_visual_form.ts`
- `CALLFORM-3`
  - `frontend/src/windows/FlowEditorWindow.vue`
- `CALLFORM-4`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
- `CALLFORM-5`
  - `plan.md`
  - `docs/change/2026-03-22_win-call-visual-form.md`
  - `docs/change/README.md`

## 关键设计决策与权衡

- 普通模式只作为编辑层投影，不引入第二套持久化 spec 模型。
  - 好处：运行时契约仍然稳定落在 `args_template + inputs`。
  - 代价：普通模式必须严格做兼容性分析，遇到未知字段或多 binding 时直接回退。
- schema 解析采用“本地 override + capability input_schema 兜底”。
  - 好处：高频方法能有更细的字段体验，同时保留“覆盖所有方法”的扩展路线。
  - 代价：当前只接受受限 JSON Schema 子集，复杂 schema 仍需走 `Advanced JSON`。
- `compose` 不强行复用 call 字段表单。
  - 好处：变更范围最小，避免破坏现有模板绑定路径。
  - 代价：本轮普通模式只覆盖 `call` 节点。

### 性能要点

- capability schema 仅在已有 capability query 结果中读取，不引入额外网络往返。
- 普通模式字段状态基于当前节点草稿即时计算，避免新增持久化镜像或重复序列化。
- JSON Pointer 读写只针对单个字段路径更新，避免每次编辑重建整份绑定列表语义。

### 可扩展性要点

- 后续可继续为更多高频方法在 `flow_method_schemas.ts` 中增加 override schema。
- capability 若提供更完整 `input_schema`，现有 resolver 可直接复用，无需继续加专用页面。
- 字段 binding source 已抽象为 `VisualBindingSource`，后续新增 source kind 时可沿 store/UI 两层扩展。

## 测试与验证方式 / 结果

- 工作目录：
  - `D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win\frontend`
- `npm ci`
  - 结果：通过
- `npx vue-tsc --noEmit --pretty false`
  - 结果：失败
  - 本次新增问题：
    - `src/stores/flow.ts` 两处字段 binding 类型收束错误，已在本轮修复
  - 当前剩余阻塞：
    - 缺失 `wailsjs` 生成类型：`src/pages/*`、`src/stores/*`、`src/windows/*`
    - 第三方类型缺失：`d3-*`、`radix-vue`、Bluetooth DOM types
    - 既有业务错误：`src/pages/Presets.vue`
  - 影响边界：
    - 过滤后 `FlowEditorWindow.vue` 仅剩 `wailsjs` 生成类型缺失，不存在本轮新增独立模板/逻辑类型错误
- `npm run build`
  - 结果：失败
  - 当前阻塞：
    - `src/pages/TopicBus.vue` 无法解析 `../../wailsjs/go/main/App`
  - 影响边界：
    - 构建失败点不在本轮修改文件
    - Vite 已进入全局 transform/build 流程，未发现本轮编辑文件引入的独立语法错误
- 浏览器交互验证
  - 结果：未执行
  - 原因：前端构建与类型环境被仓库既有 `wailsjs` 生成物缺失阻塞

## 3.3 Code Review 结论

- 需求覆盖：通过。`call` 节点普通模式、字段固定值、字段级上游数据引用、`Advanced JSON` 回退和“支持所有方法的架构路径”均已覆盖。
- 架构合理性：通过。schema 解析、兼容性分析和写回逻辑集中在 store/helper，UI 只负责交互投影，没有引入新的运行时契约。
- 性能风险：通过。未引入新的远程调用或重复序列化热点；字段编辑只更新单字段路径与单字段 binding。
- 可读性与一致性：通过。普通模式与 `Advanced JSON`、`call` 与 `compose` 的边界清晰，写回 API 命名直接对应行为。
- 可扩展性与配置化：通过。本地 override schema、capability schema resolver 和 `VisualBindingSource` 为后续方法扩展与新 binding source 预留了清晰入口。
- 稳定性与安全：通过。字段 binding 在写回前校验祖先节点、meta 字段和 JSON Pointer；普通模式遇到未知字段或多 binding 会显式回退，而不是静默覆盖。
- 测试覆盖情况：通过但存在基线阻塞。已完成依赖安装、类型检查和构建尝试；剩余失败均属于仓库现有基础设施/类型问题。
- 子Agent治理与审计：通过。本轮未使用子Agent；原因是当前环境规则要求未获用户显式授权不得派发，且 store 与窗口组件写集高度耦合。

## 潜在影响与回滚方案

### 潜在影响

- capability `input_schema` 尚未加载或方法 schema 不受支持时，`call` 节点普通模式会明确回退到 `Advanced JSON`。
- 普通模式现在对“超出 schema 的字段”和“同字段多 binding”更保守，旧节点若存在这些情况，会看到不可用提示而不是继续在普通模式中编辑。

### 回滚方案

- 直接回退以下文件即可撤销本轮实现：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flow_json_pointer.ts`
  - `frontend/src/stores/flow_method_schemas.ts`
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow_visual_form.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `plan.md`
  - `docs/change/2026-03-22_win-call-visual-form.md`
  - `docs/change/README.md`

## 子Agent执行轨迹

- 本轮未使用子Agent。
- 原因：
  - 当前会话未获得用户对子Agent的显式授权；
  - `flow.ts`、helper 和 `FlowEditorWindow.vue` 共享同一编辑语义，拆分后文件写集和集成成本过高。
- Task ID → Agent → Worktree → 文件 → 验收结果
  - `CALLFORM-1` → 主Agent → `D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win` → `frontend/src/stores/flow.ts` → 通过
  - `CALLFORM-2` → 主Agent → `D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win` → `frontend/src/stores/flow.ts`, `frontend/src/stores/flow_json_pointer.ts`, `frontend/src/stores/flow_method_schemas.ts`, `frontend/src/stores/flow_schema_resolver.ts`, `frontend/src/stores/flow_visual_form.ts` → 通过
  - `CALLFORM-3` → 主Agent → `D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win` → `frontend/src/windows/FlowEditorWindow.vue` → 通过
  - `CALLFORM-4` → 主Agent → `D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win` → `frontend/src/stores/flow.ts`, `frontend/src/windows/FlowEditorWindow.vue` → 通过
  - `CALLFORM-5` → 主Agent → `D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win` → `plan.md`, `docs/change/2026-03-22_win-call-visual-form.md`, `docs/change/README.md` → 通过
