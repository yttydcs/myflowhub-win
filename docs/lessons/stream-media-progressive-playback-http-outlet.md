# stream-media-progressive-playback-http-outlet

## Summary

- `MyFlowHub-Win` 的 Stream 媒体播放不能把原始音视频字节直接塞过 Wails 高频事件桥。
- 严格边接收边播放的最小可行路径是：Go 侧承接 DATA、顺序写入临时 sink、再通过本地 HTTP outlet 暴露给 `<audio>` / `<video>`。
- 这条路径只对浏览器原生支持且满足 `chunk` 传输约束的媒体格式成立；其余情况必须显式降级。

## Lookup Hints

- `stream.media`
- `MediaSnapshot`
- `ConfigureSourceInputSimple`
- `media playback requires chunk unit mode`
- `progressiveMediaSink`
- `127.0.0.1`
- `PickStreamMediaFile`
- `buffering`
- `ready`
- `complete`

## Symptoms

- 接收端 delivery 已经持续收到字节，但窗口仍只显示 stats，无法开始播放。
- 播放只能在完整接收后手动打开文件，不能边接收边播。
- 把媒体帧直接透给前端后出现 UI 卡顿、内存升高或桥接频率过高。

## Impact

- 无法满足“接收端严格边接收边播放”的产品要求。
- 媒体数据与控制/观察事件混在一起，会放大 Wails bridge 的性能和稳定性风险。
- 后续桌面采集若复用错误路径，会重复踩同一类问题。

## Trigger Conditions

- 尝试用 `stream.text` / `stream.stats` 风格事件直接承载媒体字节。
- 媒体 source / delivery 不是 `chunk` 模式。
- `contentType` 与 `kind` 不匹配，或浏览器本身不支持该容器/编码的渐进播放。

## Root Cause

- Wails 事件桥适合控制面和摘要事件，不适合持续的大体积媒体 DATA。
- 浏览器原生播放器更适合消费一个持续增长的 URL 资源，而不是来自 JS 内存里的手工分片拼接。
- 因此数据面应留在 Go runtime：producer 顺序发 DATA，consumer 顺序写 sink，前端只拿状态和 URL。

## Resolution

- 为非 `text` source 增加最小输入模型：
  - `inputKind = file`
  - `filePath`
- producer 侧为每个 active delivery 启动独立文件 sender：
  - 固定 `64 KiB` chunk
  - 依赖 ACK window 控制发送节奏
  - 复用 DATA `flags` bit0 作为 EOF
- consumer 侧建立：
  - `progressiveMediaSink`
  - `mediaHTTPServer`
  - `stream.media` 状态事件
- 前端 delivery window 只根据 `stream.media.mediaUrl` 驱动 `<audio>` / `<video>`，失败时回退到错误提示 + stats。

## Prevention / Guardrails

- 后续任何 Stream 媒体能力都应继续保持“Go 数据面、前端控制/显示面”分层。
- 想支持严格边接收边播放时，先检查三件事：
  1. `unit_mode` 是否为 `chunk`
  2. `kind` / `contentType` 是否匹配
  3. 浏览器是否原生支持对应媒体格式
- 若任一条件不满足，必须显式返回错误或降级，不要假装为“可播放但还没就绪”。

## Related Docs

- [../requirements/stream.md](../requirements/stream.md)
- [../specs/stream.md](../specs/stream.md)
- [../change/2026-04-13_win-stream-media-io.md](../change/2026-04-13_win-stream-media-io.md)
