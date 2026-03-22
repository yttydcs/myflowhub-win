# Win Flow Call Visual Form Plan

## 项目目标与当前状态

### 目标

- 为 Win Flow 编辑器的 `call` 节点实现普通模式字段表单。
- 普通模式支持字段固定值与字段级上游数据引用。
- 对没有可用 schema 或超出普通模式表达范围的节点，直接隐藏普通模式，只保留 `Advanced JSON`。
- 架构上支持“所有 `call` 方法”接入普通模式。

### 当前状态

- 当前执行 worktree 已创建，后续实现仅在本 worktree 中进行。
- Win 编辑器已支持：
  - `call/compose`
  - `args_template`
  - `inputs`
  - `Advanced JSON`
  - 祖先节点校验
- 当前 Win 前端 capability route 尚未把 `input_schema/output_schema` 暴露给编辑器。
- 本轮已补齐本仓稳定文档：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`

## Workflow 信息

- 仓库：`MyFlowHub-Win`
- 分支：`feat/win-call-visual-form`
- Base：`main`
- Worktree：`D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win`
- 当前阶段：`4 归档变更`

## Related Requirements

- `D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`

## Related Specs

- `D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`

## Requirements Impact

- `add`

## Specs Impact

- `add`

## 可执行任务清单（Checklist）

- [x] `CALLFORM-1` 扩展 capability route 元数据与 schema 查询链路
- [x] `CALLFORM-2` 建立视觉 schema 解析与字段状态基础设施
- [x] `CALLFORM-3` 实现 call 节点普通模式字段表单与字段引用弹窗
- [x] `CALLFORM-4` 执行验证并修正问题
- [x] `CALLFORM-5` 完成 Code Review 与归档

## Task 详情

### `CALLFORM-1` 扩展 capability route 元数据与 schema 查询链路

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win`
- Plan：`D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win\plan.md`
- 目标：
  - 让 Win Flow capability query 请求带上 `include_schema=true`
  - 让前端 route 模型保留 `input_schema/output_schema/default_timeout_ms/permissions/tags`
- 涉及模块 / 文件：
  - `frontend/src/stores/flow.ts`
- Write set：
  - `frontend/src/stores/flow.ts`
- 验收条件：
  - capability 查询后，前端 state 中能读取 schema 元数据
  - 不改变既有 `method/target` 回填语义
- 测试点：
  - 静态检查 route 映射字段完整
  - 手工检查 capability route 对象包含 schema 字段
- 回滚点：
  - 回退 `frontend/src/stores/flow.ts`
- 风险与注意事项：
  - 不要把查询用的 schema 元数据错误持久化到 graph spec
- 关键上下文引用：
  - `frontend/src/stores/flow.ts`
  - `repo/MyFlowHub-Server/docs/specs/exec.md`

### `CALLFORM-2` 建立视觉 schema 解析与字段状态基础设施

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win`
- Plan：`D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win\plan.md`
- 目标：
  - 建立本地 override schema + capability `input_schema` 统一解析能力
  - 新增 JSON Pointer 读写 helper
  - 新增普通模式兼容性分析与字段状态读写 API
- 涉及模块 / 文件：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flow_method_schemas.ts`（新增）
  - `frontend/src/stores/flow_schema_resolver.ts`（新增）
  - `frontend/src/stores/flow_json_pointer.ts`（新增）
  - `frontend/src/stores/flow_visual_form.ts`（新增）
- Write set：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flow_method_schemas.ts`
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow_json_pointer.ts`
  - `frontend/src/stores/flow_visual_form.ts`
- 验收条件：
  - store 能返回当前节点是否支持普通模式
  - store 能返回字段视图模型
  - store 能对单个字段执行 literal/binding 的增删改
- 测试点：
  - 兼容性分析：未知字段、多 binding、非法祖先引用
  - JSON Pointer 读写：写入、覆盖、删除、空对象清理
- 回滚点：
  - 回退上述 store 与 helper 文件
- 风险与注意事项：
  - 普通模式不得隐式吞掉高级 JSON 字段
  - 第一版只接受受限 JSON Schema 子集
- 关键上下文引用：
  - `frontend/src/stores/flow.ts`
  - `docs/specs/flow-editor-visual-form.md`

### `CALLFORM-3` 实现 call 节点普通模式字段表单与字段引用弹窗

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win`
- Plan：`D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win\plan.md`
- 目标：
  - 将当前 call 节点表单从 `args_template + input bindings` 改为字段表单
  - 提供字段级 `fx` 引用弹窗
  - 对不支持普通模式的方法或节点，仅显示 `Advanced JSON`
- 涉及模块 / 文件：
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write set：
  - `frontend/src/windows/FlowEditorWindow.vue`
- 验收条件：
  - 支持普通模式的方法展示字段表单
  - 字段引用弹窗可选择祖先节点输出 / trigger / meta
  - 不支持时，普通模式不显示
- 测试点：
  - 选择方法后展示对应字段
  - 为字段设置固定值
  - 为字段设置和清除 binding
  - 不支持方法只保留 `Advanced JSON`
- 回滚点：
  - 回退 `frontend/src/windows/FlowEditorWindow.vue`
- 风险与注意事项：
  - 不要破坏现有 `compose` 节点编辑路径
  - UI 术语统一使用“字段引用 / 上游数据引用”
- 关键上下文引用：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow.ts`

### `CALLFORM-4` 执行验证并修正问题

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win`
- Plan：`D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win\plan.md`
- 目标：
  - 对本轮改动做静态检查、构建检查和浏览器交互验证
  - 发现问题后在同一 workflow 内修正
- 涉及模块 / 文件：
  - `frontend/**`
- Write set：
  - 仅限本计划已有写集文件
- 验收条件：
  - 至少完成静态检查和一次浏览器交互验证
  - 若全量构建受基线阻塞，必须明确记录阻塞点与本次影响边界
- 测试点：
  - `frontend` 下执行 `npx vue-tsc --noEmit --pretty false`
  - `frontend` 下执行 `npm run build`
  - 如环境允许，使用 chrome-devtools 进行普通模式交互冒烟
- 回滚点：
  - 回退本轮改动提交或相关文件
- 风险与注意事项：
  - 仓库存在既有 `wailsjs` / 三方类型基线问题，需要区分本次新增问题和历史阻塞
- 关键上下文引用：
  - `guide.md`

### `CALLFORM-5` 完成 Code Review 与归档

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win`
- Plan：`D:\project\MyFlowHub3\worktrees\feat-win-call-visual-form\MyFlowHub-Win\plan.md`
- 目标：
  - 逐项完成 3.3 Code Review
  - 使用 `$docs-governor` 完成归档和 impact 复核
- 涉及模块 / 文件：
  - `docs/change/2026-03-22_win-call-visual-form.md`（预期）
  - `docs/change/README.md`
- Write set：
  - `docs/change/2026-03-22_win-call-visual-form.md`
  - `docs/change/README.md`
- 验收条件：
  - Review 结论完整
  - change 文档完整
  - 文档索引已更新
- 测试点：
  - review 结论与验证记录一致
  - 变更文档引用 `plan/requirements/specs`
- 回滚点：
  - 删除本轮归档文档并回退相关 docs 修改
- 风险与注意事项：
  - 归档前必须再次检查 requirement/spec impact
- 关键上下文引用：
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`

## 依赖关系

- `CALLFORM-1` → `CALLFORM-2` → `CALLFORM-3` → `CALLFORM-4` → `CALLFORM-5`

## 风险与注意事项

- “覆盖所有方法”依赖 capability `input_schema` 或本地 override schema，不能靠手工表单逐个硬编码。
- 当前前端普通模式必须保持保守：不支持就隐藏，不做宽松猜测。
- `flow.ts` 与 `FlowEditorWindow.vue` 共享同一草稿模型与校验语义，当前任务高度耦合，后续 3.2 如无新的独立 write set，不适合安全拆分子Agent。

## 执行结果摘要

- `CALLFORM-1`：已完成。前端 capability 查询现会请求并保留 schema 元数据。
- `CALLFORM-2`：已完成。新增 JSON Pointer、schema resolver、visual form state helper，并在 store 暴露字段级读写 API。
- `CALLFORM-3`：已完成。`call` 节点普通模式改为字段表单 + 字段级 `fx` 弹窗；`compose` 保持原模板与 binding 编辑路径。
- `CALLFORM-4`：已完成。`vue-tsc` 与 `npm run build` 已执行，本次新增的 `flow.ts` 类型问题已修复；剩余失败点为仓库基线 `wailsjs` 生成物缺失、第三方类型缺失与既有页面错误。
- `CALLFORM-5`：已完成。Code Review 已通过，进入 `docs/change` 归档。
