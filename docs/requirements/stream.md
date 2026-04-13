# Stream Module Requirements

## Background

- `MyFlowHub-Win` 当前没有一等 `stream` 模块。
- 现有 `Flow` 页面对应的是 workflow / DAG 编排，不承载媒体流或通用数据流。
- 现有 `TopicBus` 页面对应的是无状态事件发布订阅，不维护 producer / consumer 声明，也不提供 delivery 级连接状态。
- 上游 `stream` 子协议已经在专项 worktree 中定义了：
  - `source` / `consumer endpoint` / `delivery`
  - `music` / `video` / `text` / `custom`
  - `subscribe` / `connect` / `disconnect`
  - 二进制 `DATA` / `ACK`
- Win 端需要一个与 `Flow`、`TopicBus` 分离的专用模块，把这些能力变成可以查询、连接、观察、调试，并可在受支持场景下发送和播放真实媒体的本地产品能力。

## Goal

- 在 `MyFlowHub-Win` 中引入独立 `Stream` 模块，用于：
  - 查询 producer sources
  - 查询 consumer endpoints
  - 创建 / 撤销本地 source 与 consumer endpoint
  - 执行 `subscribe` / `unsubscribe`
  - 执行控制侧 `connect` / `disconnect`
  - 以类型感知方式观察 delivery 的运行状态和流内容
- 模块必须显式体现 `kind`，并与上游 `stream` 协议的 `music|video|text|custom` 保持一致。
- 对本地 `music` / `video` source，Win 必须支持至少两类真实输入：
  - 本地媒体文件
  - 桌面采集视频
- 接收端对支持的音视频内容必须严格边接收边播放，而不是等完整接收后再开始。

## Scope

### Must

- `Stream` 必须是 Win 中的独立模块，不能复用当前 `Flow` 页面或 `FlowService`。
- Win 后端必须提供 `stream` 控制面 service，覆盖上游公开动作：
  - `announce`
  - `withdraw`
  - `list_sources`
  - `get_source`
  - `announce_consumer`
  - `withdraw_consumer`
  - `list_consumers`
  - `get_consumer`
  - `subscribe`
  - `unsubscribe`
  - `connect`
  - `disconnect`
  - `signal`
- Win 前端必须能列出 source 与 consumer，并清楚显示它们的：
  - `kind`
  - `content_type`
  - `mode`
  - `unit_mode`
  - `tags`
  - `metadata`
- Win 必须支持创建本地 `music` / `video` source，并为其保存最小输入配置：
  - `inputKind`
  - `filePath`
- Win 必须支持的输入模式：
  - `file`
    - `music` / `video` source 可选择本地媒体文件作为输入
  - `desktop`
    - 仅 `video` source 可选择桌面采集作为输入
- 本轮 `music` / `video` 文件 source 与桌面采集 source 都必须以 `bounded + chunk` 方式工作；若配置不满足该约束，系统必须显式拒绝或降级，而不是伪装为可播放。
- Win 必须支持创建本地 consumer endpoint，并在创建时声明消费 `kind`。
- Win 必须支持控制侧把同种类的 source / consumer 连接起来，并支持断开。
- Win 必须支持 consumer 自主订阅和退订。
- Win 必须记录并展示本地已知 delivery 状态，至少包括：
  - `delivery_id`
  - `producer`
  - `consumer`
  - `consumer_id`
  - `kind`
  - 最近活动时间
  - 最近错误
- Win 必须提供类型感知的观察界面：
  - `text`
    - 直接显示文本帧 / chunk 内容
  - `music`
    - 对浏览器原生支持且可渐进播放的文件格式，必须在文件未完整接收前开始播放
    - 对不支持的格式或播放失败场景，必须明确降级到错误提示 + stats
  - `video`
    - 对浏览器原生支持且可渐进播放的文件格式，必须在文件未完整接收前开始播放
    - 对桌面采集输入，必须在采集中就让接收端开始播放，而不是等待采集结束
    - 对不支持的格式或播放失败场景，必须明确降级到错误提示 + stats
  - `custom`
    - 至少显示 metadata、payload 统计、delivery 状态
- Source Window 中的桌面采集必须通过显式用户点击触发，不能自动绕过系统权限授权。
- 桌面采集开始后，系统必须把编码后的媒体 chunk 直接送入当前 delivery，而不是先录制完整文件再回放。
- 桌面采集开始时，系统必须以“启动采集时已经 active 的 deliveries”作为当前采集会话的固定目标集合。
- 桌面采集停止、共享被系统中止、窗口关闭或 source 配置变化时，系统必须显式发送终止信号或完成收口，不能留下假活跃状态。
- 业务 UI 不得直接在页面里解析原始 `session.frame`；应由 Go service 解析后发布 `stream.*` 业务事件。
- Win 端不得把高频大体积媒体 payload 原样穿过 Wails 事件桥作为默认路径。

### Optional

- 本地 `text` producer 编辑器，用于把文本内容写入已建立的 delivery。
- 独立 viewer window，例如视频窗口或文本窗口。
- 本地 source / consumer 草稿持久化。
- 更细的 viewer profile，例如 player / inspector / monitor。
- Source Window 显示桌面采集本地预览、推送统计和当前固定 delivery 集合摘要。

### Not In Scope

- 本轮不实现摄像头采集。
- 本轮不实现麦克风采集。
- 本轮不实现桌面采集的系统音频。
- 本轮不持久化用户上一次选择的具体屏幕 / 窗口。
- 本轮不保证“采集中途新增 delivery 无缝加入当前录制会话”。
- 本轮不把原始二进制媒体流完整转发到 Vue/Wails 前端。
- 本轮不改 `stream` 协议 wire。
- 本轮不引入转码、重封装或外部编解码器下载。
- 本轮只承诺浏览器原生可渐进播放的音视频文件格式。
- 本轮不负责发布新的 Proto / Server / SubProto release tag；若要发版，应在后续 workflow 收口。

## Scenarios

1. 用户在 Win 中查询远端节点声明的 `music` / `video` / `text` / `custom` source 列表。
2. 用户在 Win 中为本机创建一个 `text` consumer endpoint，并查看其 descriptor。
3. 用户在 Win 中把远端 `text` source 连接到本机 `text` consumer，并实时查看文本内容。
4. 用户在 Win 中创建本地 `music` 或 `video` source，选择一个本地媒体文件作为输入，连接到本地或远端 consumer 后，接收端在文件尚未完整接收时开始播放。
5. 用户在 Win 中创建本地 `video` source，输入模式选择 `desktop capture`，在 Source Window 中点击开始采集并选择桌面 / 窗口后，当前 active delivery 在采集中开始播放桌面内容。
6. 用户在 Win 中查询某个远端节点暴露的 consumer endpoints，作为控制侧执行 `connect`。
7. 用户在 Win 中对已建立 delivery 执行 `disconnect` 或 `signal`。

## Functional Requirements

1. Win 必须有独立 `Stream` 导航入口。
2. Win 必须有独立 `StreamService`，而不是把逻辑塞进 `DebugService` 或 `FlowService`。
3. `StreamService` 必须对所有请求做输入校验，并将 `code != 1` 统一转为可读错误。
4. Win store 必须以业务状态为主，不依赖页面内重复解码协议 payload。
5. 本地 consumer 创建时必须要求输入 `kind`。
6. source / consumer 列表查询结果必须把 `kind` 原样暴露到前端。
7. `connect` 和 `subscribe` 成功后，Win 必须记录新 delivery。
8. `disconnect`、`unsubscribe`、`withdraw`、`withdraw_consumer` 成功后，Win 必须同步移除或标记对应 delivery。
9. Win 必须区分“控制面成功但当前 viewer 不能原生渲染”与“协议失败”。
10. `text` 流内容必须能被查看，且保留最近若干条记录或片段。
11. `music` / `video` 文件 source 必须通过现有 `delivery` 路径发送文件内容，而不是只声明 descriptor。
12. 本轮 `music` / `video` 文件 source 只承诺 `bounded + chunk` 传输；若 `unit_mode != chunk` 或相关配置不满足播放约束，系统必须显式报错或降级。
13. 接收端对支持的 `music` / `video` 文件必须严格边接收边播放，不能等完整落盘后再开始。
14. `video` source 必须支持 `desktop` 输入模式，并且仅限 `video` 使用。
15. 桌面采集必须由 Source Window 中的用户手势触发，并显式检测 `getDisplayMedia()` 与 `MediaRecorder` 是否可用。
16. 桌面采集开始后，系统必须把编码后的媒体 chunk 直接推入现有 `delivery` 路径，不能先接收完整文件。
17. 桌面采集的首个有效 chunk 必须能使接收端进入新的播放会话；再次开始新采集会话时，接收端必须重置到新的播放资源，而不是把新旧会话拼成同一个媒体文件。
18. 桌面采集只承诺覆盖“启动采集时已 active 的 deliveries”；采集中途新增 delivery 若需加入，允许要求用户重启采集。
19. 若桌面采集启动时没有 active delivery，系统必须立即拒绝并提示，而不是假装采集成功。
20. 非 `text` 流即使不能播放，也必须显示持续到达的帧数、字节数、最近 `pts/position`。
21. 若本地 Proto 版本不含 `stream`，Win workflow 必须显式使用开发态依赖对齐方案，而不能假设主线版本已就绪。

## Non-functional Requirements

- 架构边界：
  - `Stream` 与 `Flow`、`TopicBus`、`Debug` 分离。
- 可维护性：
  - 控制面、viewer runtime、页面 store、source-window capture runner 之间职责清楚。
- 性能：
  - 高频流量必须在 Go 本地 runtime 中限流、裁剪或汇总后再发给前端。
- 安全性：
  - 非法 `delivery_id`、非法 `kind`、非法目标节点、空 ID 输入必须显式拒绝。
- 兼容性：
  - 桌面采集能力必须显式依赖当前 WebView / 浏览器运行时支持；不支持时要明确报错。
- 可扩展性：
  - 后续接入音频播放、视频窗口、自定义解码器或更强 capture 能力时，不应推翻当前控制面和 runtime 分层。

## Edge Cases

- Win 当前连接的 Proto 版本不包含 `stream` 定义。
- source / consumer 在 UI 打开后被远端撤销。
- `source.kind != consumer.kind` 时发起 `connect`。
- 已建立 delivery 但当前 viewer 不支持直接渲染该内容类型。
- 媒体 source / delivery 不是 `chunk` 模式，但用户仍尝试走文件输入或严格边接收边播放。
- 高频 `text` 帧导致前端渲染抖动。
- `disconnect` 成功后再次断开同一 delivery。
- 当前会话未登录或 `node_id` 未就绪时发起 source / consumer 相关动作。
- 当前运行时不支持 `getDisplayMedia()` 或 `MediaRecorder`。
- 用户弹出桌面共享授权后取消共享。
- 用户开始共享后立刻被系统 / 浏览器终止共享。
- 桌面采集 source 在录制中被 withdraw、窗口关闭，或配置从 `desktop` 改回 `file`。
- 新 delivery 在采集中途加入，但缺少当前会话初始化片段。

## Acceptance Criteria

1. Win 出现独立 `Stream` 模块，且不复用现有 `Flow` 页面语义。
2. 用户能在 Win 中查询 sources 与 consumers，并看到 `kind` 等 descriptor 字段。
3. 用户能在 Win 中创建本地 consumer endpoint，并声明 `kind`。
4. 用户能在 Win 中执行 `connect` / `disconnect` 和 `subscribe` / `unsubscribe`。
5. `text` 类型 delivery 建立后，用户能在 Win 中看到文本内容。
6. 本地 `music` / `video` source 创建后，用户能为其配置本地文件输入并保存。
7. 本地 `video` source 创建后，用户能为其配置 `desktop` 输入并在 Source Window 中开始 / 停止桌面采集。
8. 支持的 `music` / `video` delivery 建立后，接收端能在文件未完整接收前开始播放。
9. 桌面采集开始后，当前 active delivery 能在采集中开始播放桌面内容。
10. 不支持播放的 `music` / `video` / `custom` 类型 delivery 仍能看到有效的 delivery 状态和数据统计，并得到明确降级提示。
11. 页面不直接解析原始 `session.frame`；Go service 会发布 `stream.*` 业务事件。
12. 依赖链未正式发版时，plan 会显式记录开发态依赖对齐策略和 release 风险。

## Related Specs

- [../specs/stream.md](../specs/stream.md)
- [../../../server-stream-subproto-design/docs/specs/stream.md](../../../server-stream-subproto-design/docs/specs/stream.md)

## Related Changes

- [../change/2026-04-13_win-stream-desktop-capture.md](../change/2026-04-13_win-stream-desktop-capture.md)
- [../change/2026-04-13_win-stream-media-io.md](../change/2026-04-13_win-stream-media-io.md)
