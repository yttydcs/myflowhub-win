# Showcase Display Widgets

## Background

- 当前 Win Showcase 已支持 `topic_button` 与 `var` widget，并提供 `display`、`slider`、`switch` 等基础模式。
- 在现有 Center / Editor / Viewer 架构下，Showcase 已能承载简单的操作面板，但纯展示组件仍偏单一。
- 用户希望 Showcase 能提供更丰富的展示组件，同时保持当前简洁风格。

## Goal

- 为 Win Showcase 增加更丰富的单值展示组件，提升状态面板的表达力。
- 保持当前低噪音、卡片式的简洁视觉，不把 Showcase 扩展成重型 dashboard。

## Scope

### Must

- `var` widget 支持新的显式展示模式：
  - `metric`
  - `badge`
  - `progress`
- 保持既有 `display`、`slider`、`switch`、`topic_button` 模式兼容。
- Editor 中可配置新模式，Viewer 中可一致渲染。
- `columns` 与 `canvas_percent` 布局下都能稳定工作。
- 异常值和空值必须优雅降级，不得导致页面报错或布局损坏。

### Optional

- 为新展示模式增加轻量语义色和辅助元信息，只要不破坏简洁风格。
- 复用现有数值范围配置，为 `progress` 提供上下界。

### Out of Scope

- 不新增历史图表、趋势线、多变量聚合或计算型 widget。
- 不新增新的后端接口、协议或数据源。
- 不重做 Showcase Center、布局模型和多窗口同步逻辑。

## Scenarios

- 用户希望把关键数值以更醒目的 metric 卡片方式展示。
- 用户希望把状态值以紧凑 badge 方式展示，快速判断“正常 / 异常 / 开 / 关”。
- 用户希望把数值变量映射成进度条，快速判断其处于区间中的哪个位置。
- 用户在 Editor 中配置好模式后，希望 Viewer 与 Editor 预览表现一致。

## Functional Requirements

1. `var` widget 编辑器必须允许用户显式选择 `metric`、`badge`、`progress` 三种展示模式。
2. `metric` 模式必须突出展示当前变量值，并保留足够的上下文信息用于识别来源。
3. `badge` 模式必须以紧凑标签显示当前变量值，并对常见布尔 / 状态值提供稳定的语义色。
4. `progress` 模式必须基于已配置的数值范围展示当前值进度。
5. `progress` 模式在无法解析数值时，必须提供明确降级态，而不是静默空白。
6. `auto` 模式必须继续保持当前推断规则，避免已有 widget 出现行为突变。
7. 现有 `slider`、`switch`、`topic_button` 的交互语义不得回退。
8. 新模式保存后再次加载配置，必须保持稳定。

## Non-functional Requirements

- 兼容性：
  - 旧 Showcase 配置可直接加载。
  - 未识别 mode 必须回退到安全默认值。
- 简洁性：
  - 视觉表达应聚焦单值信息，不引入大面积装饰和复杂图例。
- 可维护性：
  - Editor 与 Viewer 的展示规则必须尽量共享，避免双份逻辑漂移。
- 性能：
  - 不增加额外订阅或高频持久化。
  - 不引入可避免的重复解析或重复渲染。

## Edge Cases

- `progress` 收到空字符串、非法数值或超出区间的值。
- `progress` 的范围配置非法或缺失。
- `badge` 收到未知状态字符串。
- `metric` / `badge` / `progress` 在变量尚未有值时的展示。
- 旧配置保存了未知的 `mode` 字段值。

## Acceptance Criteria

1. 用户可以在 Editor 中创建和保存 `metric`、`badge`、`progress` 三种 `var` widget。
2. 保存后重新打开 Editor 或 Viewer，新模式会被正确保留和渲染。
3. `columns` 与 `canvas_percent` 下的渲染都保持可读且不破坏布局。
4. 非数值的 `progress` widget 不会报错，并能给出明确降级提示。
5. 现有 `display`、`slider`、`switch`、`topic_button` 行为不回退。

## Related Specs

- [showcase-display-widgets.md](../specs/showcase-display-widgets.md)
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\topicbus.md`

## Related Changes

- [2026-03-21_showcase-center-editor.md](../change/2026-03-21_showcase-center-editor.md)
- [2026-03-21_showcase-ui-simplify.md](../change/2026-03-21_showcase-ui-simplify.md)
