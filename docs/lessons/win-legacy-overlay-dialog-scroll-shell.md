# win-legacy-overlay-dialog-scroll-shell

## Summary

- `MyFlowHub-Win` 里的旧式 `Overlay` 卡片弹窗如果只有 `max-w-*` 而没有外层 `max-h` 和 `overflow-hidden`，长表单或长列表会把整张卡片顶出视口。
- 可靠修复模式不是去改共享 `Overlay.vue`，而是在具体弹窗里使用统一壳层：外层卡片 `flex + max-h-[85vh] + overflow-hidden`，中部主体 `min-h-0 flex-1 overflow-y-auto`。
- 对包含输入框或可聚焦控件的滚动区，内部还要保留额外 `px/py/pr` 缓冲，否则 focus ring 很容易被裁切。

## Lookup Hints

- `max-h-[85vh]`
- `overflow-hidden`
- `overflow-y-auto`
- `legacy overlay dialog`
- `focus ring clipping`
- `data-node-vars-scroll`
- `AccessPolicy dialog`
- `FlowFieldBindingDialog`

## Symptoms

- 弹窗内容一多就继续向下增长，底部按钮或关闭动作跑出视口。
- 某个局部列表虽然能滚动，但整张卡片仍然超出屏幕。
- 输入框获得焦点时，focus ring 被滚动容器边缘切掉。

## Impact

- 用户在小窗口、高缩放或长内容场景下无法稳定访问弹窗底部操作区。
- 旧弹窗的交互表现不一致，排查时容易误以为是单个页面自己的 CSS 问题。

## Trigger Conditions

- `Overlay` 内部卡片只有 `max-w-*`，没有 `max-h-*`。
- 卡片外层没有 `overflow-hidden`。
- 滚动被放在局部列表上，但 header/footer 之间的主体层没有承担统一滚动职责。

## Root Cause

- 真正需要受限的是“卡片壳层”，而不只是其中某一个列表节点。
- 旧弹窗结构通常缺少 `flex-col` 和 `min-h-0 flex-1` 这组配套约束，导致浏览器无法把多余高度收敛到中部滚动区。
- 输入控件的外圈视觉效果超出元素盒模型时，如果滚动区贴边裁切，就会把 focus ring 一并截断。

## Investigation Trail

- 先搜索页面/组件内所有 `Overlay` 卡片，区分“已合规弹窗”和“仍只有裸 `max-w-*` 的旧弹窗”。
- 对代表性问题弹窗检查三层结构：
  - 外层卡片是否有 `max-h` 和 `overflow-hidden`
  - 中层主体是否有 `min-h-0 flex-1 overflow-y-auto`
  - 滚动区内部是否保留 padding 缓冲
- 补代表性结构测试，锁定关键 class 和 `data-*` 选择器。

## Resolution

- 在具体弹窗里统一使用局部壳层模式：
  - 外层卡片：`flex max-h-[85vh] w-full max-w-* flex-col overflow-hidden ...`
  - 主体滚动区：`mt-5 min-h-0 flex-1 overflow-y-auto`
  - 必要时在滚动区内部补 `px-1 py-1 pr-2`
- 保持共享 `Overlay.vue` 行为不变，只收敛旧弹窗模板结构。
- 用少量代表性测试和一次全局结构扫描验证没有遗留裸 `max-w` 弹窗。

## Prevention / Guardrails

- 新增 Win 弹窗时，默认先套用“外层限高 + 中部滚动 + 内层缓冲”的壳层结构，不要等到内容变长后再补救。
- 如果某个弹窗已经有局部列表滚动，也仍要检查整张卡片是否受限于 `max-h`。
- 对“想统一改所有弹窗”的需求，优先局部收敛旧结构，避免先动共享 `Overlay.vue` 导致已合规弹窗回归。

## Related Docs

- [2026-04-14_win-dialog-height-scroll-convergence.md](../change/2026-04-14_win-dialog-height-scroll-convergence.md)
- [2026-03-27_win-access-policy-role-dialog-refine.md](../change/2026-03-27_win-access-policy-role-dialog-refine.md)
