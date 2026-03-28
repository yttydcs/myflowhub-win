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
- Win 端需要一个与 `Flow`、`TopicBus` 分离的专用模块，把这些能力变成可以查询、连接、观察和调试的本地产品能力。

## Goal

- 在 `MyFlowHub-Win` 中引入独立 `Stream` 模块，用于：
  - 查询 producer sources
  - 查询 consumer endpoints
  - 创建 / 撤销本地 source 与 consumer endpoint
  - 执行 `subscribe` / `unsubscribe`
  - 执行控制侧 `connect` / `disconnect`
  - 以类型感知方式观察 delivery 的运行状态和流内容
- 模块必须显式体现 `kind`，并与上游 `stream` 协议的 `music|video|text|custom` 保持一致。

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
    - 至少显示 metadata、payload 统计、delivery 状态
  - `video`
    - 至少显示 metadata、payload 统计、delivery 状态
  - `custom`
    - 至少显示 metadata、payload 统计、delivery 状态
- 业务 UI 不得直接在页面里解析原始 `session.frame`；应由 Go service 解析后发布 `stream.*` 业务事件。
- Win 端不得把高频大体积媒体 payload 原样穿过 Wails 事件桥作为默认路径。

### Optional

- 本地 `text` producer 编辑器，用于把文本内容写入已建立的 delivery。
- 独立 viewer window，例如视频窗口或文本窗口。
- 本地 source / consumer 草稿持久化。
- 更细的 viewer profile，例如 player / inspector / monitor。

### Not In Scope

- 本轮不实现摄像头、麦克风、屏幕采集。
- 本轮不承诺音乐 / 视频的最终播放渲染。
- 本轮不把原始二进制媒体流完整转发到 Vue/Wails 前端。
- 本轮不改 `stream` 协议 wire。
- 本轮不负责发布新的 Proto / Server / SubProto release tag；若要发版，应在后续 workflow 收口。

## Scenarios

1. 用户在 Win 中查询远端节点声明的 `music` / `video` / `text` / `custom` source 列表。
2. 用户在 Win 中为本机创建一个 `text` consumer endpoint，并查看其 descriptor。
3. 用户在 Win 中把远端 `text` source 连接到本机 `text` consumer，并实时查看文本内容。
4. 用户在 Win 中把远端 `music` 或 `video` source 连接到本机 consumer，并至少看到 delivery 已建立、持续收帧和流元数据。
5. 用户在 Win 中查询某个远端节点暴露的 consumer endpoints，作为控制侧执行 `connect`。
6. 用户在 Win 中对已建立 delivery 执行 `disconnect` 或 `signal`。

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
11. 非 `text` 流即使暂不渲染，也必须显示持续到达的帧数、字节数、最近 `pts/position`。
12. 若本地 Proto 版本不含 `stream`，Win workflow 必须显式使用开发态依赖对齐方案，而不能假设主线版本已就绪。

## Non-functional Requirements

- 架构边界：
  - `Stream` 与 `Flow`、`TopicBus`、`Debug` 分离。
- 可维护性：
  - 控制面、viewer runtime、页面 store 之间职责清楚。
- 性能：
  - 高频流量必须在 Go 本地 runtime 中限流、裁剪或汇总后再发给前端。
- 安全性：
  - 非法 `delivery_id`、非法 `kind`、非法目标节点、空 ID 输入必须显式拒绝。
- 可扩展性：
  - 后续接入音频播放、视频窗口、自定义解码器时，不应推翻当前控制面和 runtime 分层。

## Edge Cases

- Win 当前连接的 Proto 版本不包含 `stream` 定义。
- source / consumer 在 UI 打开后被远端撤销。
- `source.kind != consumer.kind` 时发起 `connect`。
- 已建立 delivery 但当前 viewer 不支持直接渲染该内容类型。
- 高频 `text` 帧导致前端渲染抖动。
- `disconnect` 成功后再次断开同一 delivery。
- 当前会话未登录或 `node_id` 未就绪时发起 source / consumer 相关动作。

## Acceptance Criteria

1. Win 出现独立 `Stream` 模块，且不复用现有 `Flow` 页面语义。
2. 用户能在 Win 中查询 sources 与 consumers，并看到 `kind` 等 descriptor 字段。
3. 用户能在 Win 中创建本地 consumer endpoint，并声明 `kind`。
4. 用户能在 Win 中执行 `connect` / `disconnect` 和 `subscribe` / `unsubscribe`。
5. `text` 类型 delivery 建立后，用户能在 Win 中看到文本内容。
6. `music` / `video` / `custom` 类型 delivery 建立后，用户至少能看到有效的 delivery 状态和数据统计。
7. 页面不直接解析原始 `session.frame`；Go service 会发布 `stream.*` 业务事件。
8. 依赖链未正式发版时，plan 会显式记录开发态对齐策略和 release 风险。

## Related Specs

- [../specs/stream.md](../specs/stream.md)
- [../../../server-stream-subproto-design/docs/specs/stream.md](../../../server-stream-subproto-design/docs/specs/stream.md)

## Related Changes

- 待本次 workflow 完成后补充。
