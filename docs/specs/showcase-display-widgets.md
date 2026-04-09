# Showcase Display Widgets Spec

## Scope

- 本规范限定 Win Showcase 中 `var` widget 的展示模式扩展与渲染约束。
- 本规范不修改 TopicBus / VarStore 协议，也不新增新的运行时接口。
- `line_chart` 的样本数据仅存在于前端内存，不属于 `ShowcaseConfig` 持久化数据本体。

## Interfaces / Contracts

### 1. 展示模式枚举

- `ShowcaseVarWidget.mode` 必须支持以下取值：
  - `auto`
  - `display`
  - `metric`
  - `badge`
  - `progress`
  - `line_chart`
  - `slider`
  - `switch`

### 2. `auto` 推断契约

- `auto` 必须保持现有推断语义：
  - `bool` / `boolean` -> `switch`
  - 数值类型 -> `slider`
  - 其他 -> `display`
- 新增显示模式不得改变 `auto` 的既有结果。

### 3. 展示模式职责边界

- `display`
  - 紧凑文本显示原始值。
- `metric`
  - 以更突出的主值卡片展示原始值。
- `badge`
  - 以紧凑状态标签展示原始值。
- `progress`
  - 以进度条展示当前数值在配置区间中的相对位置。
- `line_chart`
  - 以折线图展示当前 session 内数值变量的前端内存样本趋势。
- `slider`
  - 继续作为交互控件，允许用户写回数值变量。
- `switch`
  - 继续作为交互控件，允许用户写回布尔 / 枚举变量。

### 4. 区间配置复用

- `progress` 模式复用 `ShowcaseVarWidget.slider.min/max` 作为显示区间。
- `slider.step` 与 `slider.throttleMs` 仅在 `slider` 模式中具备交互语义。
- 若区间非法，前后端 normalize 必须回退为默认有效区间。

### 5. 折线图配置契约

- `line_chart` 持久化配置放在 `ShowcaseVarWidget.chart`：
  - `rangeMs`
  - `bucketMs`
- `rangeMs` 表示可见时间范围。
- `bucketMs` 表示聚合粒度。
- `line_chart` 必须支持保存上述配置，但样本点本身不得进入持久化模型。
- 若配置非法，前后端 normalize 必须回退到安全默认值。

### 6. Editor / Viewer 一致性

- Editor 中的 widget 预览与 Viewer 中的 widget 展示必须使用同一套模式判断和视觉语义。
- 两种布局模式下都必须遵守同样的 mode 渲染契约。

## Data Model or Protocol

### 1. 持久化模型

`ShowcaseVarWidget` 持久化结构继续使用现有模型，不新增版本号升级要求：

```ts
type ShowcaseVarWidget = {
  ownerId: number
  name: string
  mode: "auto" | "display" | "metric" | "badge" | "progress" | "line_chart" | "slider" | "switch"
  visibility: string
  type: string
  slider: {
    min: number
    max: number
    step: number
    throttleMs: number
  }
  switch: {
    onValue: string
    offValue: string
  }
  chart: {
    rangeMs: number
    bucketMs: number
  }
}
```

### 2. Normalize 契约

- Go 与前端都必须接受上述新 mode。
- 未识别 mode 必须回退为 `auto`。
- 旧配置缺失 mode 时，必须按现有规则回退到 `auto`。
- `slider` 区间配置继续沿用现有 normalize 规则。
- `line_chart.chart` 缺失或非法时，必须回退到默认 `rangeMs/bucketMs`。

### 3. 状态语义约束

- `badge` 的语义色规则必须是确定性的，同一输入在 Editor / Viewer 中结果一致。
- `progress` 计算结果必须 clamp 到 `0% ~ 100%`。
- 当变量当前值不可解析为数值时，`progress` 只渲染降级态，不得触发写操作。
- `line_chart` 只消费前端内存中的数值样本，不得发起新的后端历史查询。
- `line_chart` 在样本不足两点、时间窗内无样本或当前值非数值时，只渲染降级态。

## Error Handling

- 未识别 mode：
  - normalize 为 `auto`
- `progress` 非数值值：
  - 渲染为空态提示
  - 不中断整个 screen 渲染
- `line_chart` 无有效样本：
  - 渲染为空态提示
  - 不中断整个 screen 渲染
- 变量尚未有值：
  - 新模式显示统一的未就绪提示，而不是空白卡片

## Security / Safety

- `metric`、`badge`、`progress`、`line_chart` 都是只读展示模式，不得调用 `SetSimple`、`SendSimple` 或 TopicBus 发送路径。
- 现有 `slider`、`switch`、`topic_button` 的 ready-check、输入校验和错误提示必须保持不变。

## Performance Constraints

- 新模式不得新增订阅种类或订阅数量。
- 模式判断、数值解析和状态色计算应尽量集中复用，避免 Editor / Viewer 双份重复逻辑。
- 不得通过高频 save 或额外 runtime event 来驱动展示模式刷新。
- `line_chart` 的样本缓存必须有 retention 上限和裁剪策略，避免前端内存无界增长。
- `line_chart` 的粒度聚合应基于已有内存样本完成，不得引入额外 I/O。

## Related Requirements

- [showcase-display-widgets.md](../requirements/showcase-display-widgets.md)

## Related Changes

- [2026-03-21_showcase-center-editor.md](../change/2026-03-21_showcase-center-editor.md)
- [2026-03-21_showcase-ui-simplify.md](../change/2026-03-21_showcase-ui-simplify.md)
