# Win Legacy Dialog Height And Scroll Convergence

## 变更背景 / 目标

- 用户先反馈 `VarPool` 页 `Node Variables` 弹窗在长列表场景下没有视口内限高，内容会把整张卡片顶出屏幕。
- 继续盘点后确认这不是单点问题，`MyFlowHub-Win` 里还有一批旧式 `Overlay` 卡片弹窗仍只有 `max-w-*`，缺少外层 `max-h` 约束。
- 本轮目标是把这些旧弹窗统一收敛到“视口内限高 + 外层裁切 + 中部内部滚动”的稳定壳层，同时不改共享 `Overlay.vue`，不改变现有业务交互。

## 具体变更内容

- `frontend/src/components/varpool/NodeVarsDialog.vue`
  - 将 `Node Variables` 弹窗改为 `max-h-[85vh] + flex-col + overflow-hidden` 的卡片壳层。
  - 把变量列表主体移入独立滚动区，并预留内边距避免 focus ring 被裁切。
- `frontend/src/pages/VarPool.vue`
- `frontend/src/pages/Devices.vue`
- `frontend/src/pages/AccessPolicy.vue`
- `frontend/src/layout/AppShell.vue`
  - 将核心旧弹窗收敛到同一壳层模式，保持原有表单、按钮和关闭行为不变。
- `frontend/src/pages/File.vue`
- `frontend/src/pages/Flow.vue`
- `frontend/src/pages/Showcase.vue`
- `frontend/src/pages/ShowcaseCenter.vue`
- `frontend/src/pages/Stream.vue`
  - 将页面级旧式 overlay 卡片统一补齐视口内限高与内部滚动，不再让整张卡片随内容继续长高。
- `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
- `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - 让 flow editor 旧弹窗结构向已验证的 picker/dialog 模式对齐，同时保留现有焦点管理和 aria 语义。
- 测试
  - 新增 `frontend/src/components/varpool/NodeVarsDialog.test.ts`
  - 更新 `frontend/src/pages/AccessPolicy.test.ts`
  - 更新 `frontend/src/components/flow/editor/FlowFieldBindingDialog.test.ts`
  - 为代表性旧弹窗补充限高和内部滚动断言。

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `updated`

## Related requirements

- 未找到与 Win 旧弹窗壳层收敛直接对应的稳定 requirements 文档。

## Related specs

- `D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll\docs\specs\flow-editor-accessibility.md`

## Related lessons

- `D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll\docs\lessons\win-legacy-overlay-dialog-scroll-shell.md`
- `D:\project\MyFlowHub3\worktrees\fix-win-varpool-node-vars-dialog-scroll\docs\lessons\README.md`

## 对应 plan.md 任务映射

- `DHC-1`
  - 收敛 `VarPool` / `Devices` / `AccessPolicy` / `AppShell` 等核心旧弹窗。
- `DHC-2`
  - 收敛 `File` / `Flow` / `Showcase` / `ShowcaseCenter` / `Stream` 等页面级旧弹窗。
- `DHC-3`
  - 收敛 flow editor 旧弹窗组件。
- `DHC-4`
  - 补代表性结构测试并完成定向验证。

## 经验 / 教训摘要

- 对这类旧式 `Overlay` 卡片，真正的根因通常不是“内部列表没滚”，而是“整张卡片没有外层 `max-h` 和 `overflow-hidden`”。
- 只把滚动加在某个局部列表上并不够；如果卡片本体不受限，长表单和大列表仍然会把 footer 挤出视口。
- 滚动区内部如果紧贴裁切边界，输入框或按钮的 focus ring 很容易被切掉，需要额外 `px/py` 缓冲。
- 这类问题更适合做局部模板收敛，而不是直接改共享 `Overlay.vue`，因为各弹窗的宽度、滚动位置和 footer 结构并不完全一致。

## 可复用排查线索

- 症状：
  - overlay 弹窗打开后随内容继续长高，底部按钮跑出视口
  - 列表本身能滚，但整张卡片仍超出屏幕
  - 输入框 focus ring 或按钮阴影被滚动容器边缘裁切
- 触发条件：
  - 旧弹窗卡片只有 `max-w-*`，没有 `max-h-*`
  - 卡片外层没有 `overflow-hidden`
  - 滚动放在局部列表上，但中层主体没有 `min-h-0 flex-1 overflow-y-auto`
- 关键词：
  - `max-h-[85vh]`
  - `overflow-hidden`
  - `overflow-y-auto`
  - `data-node-vars-dialog`
  - `data-node-vars-scroll`
  - `legacy overlay dialog`
- 快速检查：
  - 搜索 `Overlay` 内部卡片是否仍只有裸 `max-w-*`
  - 检查滚动区是否位于 header/footer 之间的中部主体
  - 检查滚动区内部是否保留足够的 `px/py/pr` 缓冲

## 关键设计决策与权衡

- 不改共享 `Overlay.vue`，只在各弹窗局部模板中收敛：
  - 优点是变更面最小，不会影响已合规弹窗。
  - 代价是需要逐页盘点旧弹窗，而不是一处改完。
- 使用统一的 `max-h-[85vh]` 壳层而不是为每个弹窗单独定制：
  - 优点是收敛标准清晰，测试断言也更稳定。
  - 代价是个别短弹窗也会显式具备同一结构，但视觉影响可控。
- 用少量代表性测试锁住结构契约，而不是给每个弹窗都补一套测试：
  - 优点是覆盖关键风险点且维护成本更低。
  - 代价是其余弹窗仍主要依赖结构审计和手工冒烟。

## 测试与验证方式 / 结果

- `npm exec vitest run src/pages/AccessPolicy.test.ts`
  - 通过，`5` 个测试通过。
- `npm exec vitest run src/components/flow/editor/FlowFieldBindingDialog.test.ts`
  - 通过，`1` 个测试通过。
- `npm exec vitest run src/components/varpool/NodeVarsDialog.test.ts`
  - 通过，`1` 个测试通过。
- `rg -n -U -P '<Overlay[\\s\\S]{0,900}?<div[^>]*class="(?:(?!max-h-)[^"])*max-w-[^"]*"' frontend/src/pages frontend/src/components`
  - 无命中，说明本轮扫描范围内未残留“overlay 卡片仍只有裸 `max-w`”的旧结构。
- `git diff --check`
  - 通过；仅有仓库当前 LF/CRLF 提示，无 patch 级格式错误。

## 潜在影响与回滚方案

- 潜在影响：
  - 一些原本高度不受限的旧弹窗现在会更稳定地被限制在视口内，长内容改为内部滚动。
  - 若某个局部滚动区放置位置判断错误，可能出现 header/footer 跟着滚动的问题；本轮已通过代表性测试和结构审计降低该风险。
- 回滚方案：
  - 回退本轮涉及的 Vue 页面/组件与测试文件。
  - 删除 `docs/change/2026-04-14_win-dialog-height-scroll-convergence.md`
  - 删除 `docs/lessons/win-legacy-overlay-dialog-scroll-shell.md`
  - 回退 `docs/change/README.md` 与 `docs/lessons/README.md`

## 子Agent执行轨迹

- 未使用子Agent
