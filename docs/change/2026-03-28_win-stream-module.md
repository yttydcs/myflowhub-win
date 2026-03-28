# 2026-03-28 Win Stream 模块接入

## 变更背景 / 目标

- `MyFlowHub-Win` 之前没有一等 `Stream` 模块，现有 `Flow` 只负责 workflow / DAG 编排，`TopicBus` 只负责无状态事件发布订阅。
- 上游已经补出 `stream` 子协议，定义了 producer source、consumer endpoint、delivery、`music|video|text|custom` 和 `subscribe/connect/disconnect/signal`。
- 本轮目标是在 Win 中把这套能力接成独立模块，提供完整控制面、本地 runtime 观察能力，以及 `text` 可读、非 `text` 有统计的 viewer 基线。

## 具体变更内容

### 稳定文档与依赖对齐

- 新增 `docs/requirements/stream.md`
  - 固化 Win Stream 的业务目标、能力边界和验收标准
- 新增 `docs/specs/stream.md`
  - 固化模块分层、业务事件、runtime 模型和性能约束
- 更新 `docs/requirements/README.md` 与 `docs/specs/README.md`
  - 加入 Stream 文档入口
- 更新 `go.mod`
  - 临时 `replace github.com/yttydcs/myflowhub-proto => ../proto-stream-subproto`
  - 解决当前 `myflowhub-proto v0.1.5` 缺失 `protocol/stream` 的开发态编译问题

### Go Stream service 与 runtime

- 新增 `internal/services/stream/service.go`
  - 提供 `announce/withdraw/list/get/announce_consumer/withdraw_consumer/list_consumers/get_consumer/subscribe/unsubscribe/connect/disconnect/signal` 控制面 binding
  - 统一输入校验、`req_id` 兜底和业务错误处理
- 新增 `internal/services/stream/runtime.go`
  - 监听 `session.frame`
  - 解析 `SubProtoStream` 的 `KindData` / `KindAck`
  - 维护 delivery runtime 快照
  - 对 `text` 发布有界 `stream.text`
  - 对非 `text` 发布节流 `stream.stats`
  - session 断链或错误时统一关闭已知 delivery
- 新增 `internal/services/stream/events.go`
  - 定义 `stream.delivery` / `stream.text` / `stream.stats` 事件载荷
- 更新 `internal/services/stream/service_test.go`
  - 覆盖 text DATA -> `stream.text`
  - 覆盖 ACK -> `stream.stats`
  - 校验 delivery snapshot 同步
- 更新 `app.go`
  - 注册 `StreamService`
  - 把 `stream.*` 业务事件桥接到 Wails runtime

### 前端 Stream 页与状态管理

- 新增 `frontend/src/stores/stream.ts`
  - 封装 Stream bindings
  - 维护 sources / consumers / deliveries / text viewer / stats viewer 状态
  - 消费 `stream.delivery` / `stream.text` / `stream.stats`
- 新增 `frontend/src/pages/Stream.vue`
  - 提供 source/consumer 查询、创建、撤销、连接、断开、订阅、退订和 signal 控制
  - `text` 显示最近文本帧
  - `music` / `video` / `custom` 显示 delivery 状态与统计摘要
- 更新 `frontend/src/router/index.ts` 与 `frontend/src/layout/AppShell.vue`
  - 增加 `/stream` 页面与导航入口
- 新增 `frontend/src/stores/stream.test.ts`
  - 覆盖控制面结果归一化
  - 覆盖 connect/disconnect 状态变更
  - 覆盖 runtime 事件驱动的 text/stats 镜像

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `updated`

## Related requirements

- `docs/requirements/stream.md`

## Related specs

- `docs/specs/stream.md`
- `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\specs\stream.md`

## Related lessons

- `docs/lessons/wails-binding-proto-drift.md`

## 对应 plan.md 任务映射

- `WIN-DOC-1`
  - 新增 Stream requirements/specs 并更新索引
- `WIN-DEP-1`
  - 用开发态 `replace` 补齐 `protocol/stream` 依赖
- `WIN-BE-1`
  - 完成 Stream control-plane service
- `WIN-BE-2`
  - 完成 runtime、业务事件和 Wails bridge
- `WIN-FE-1`
  - 完成 `/stream` 路由、导航、页面和 store
- `WIN-FE-2`
  - 完成 `text` viewer 与 generic stats viewer
- `WIN-VAL-1`
  - 执行 Go / Wails / 前端验证
- `WIN-REVIEW-1`
  - 完成 3.3 checklist
- `WIN-ARCHIVE-1`
  - 归档到 `docs/change`

## 经验 / 教训摘要

- `Stream` 不应混入现有 `Flow` 或 `TopicBus`；producer/consumer/delivery 是另一套产品语义。
- 高频 stream runtime 应先在 Go 侧做解码、节流和有界缓存，再桥接到前端；否则 Wails 事件桥很快会被媒体 payload 压垮。
- 即使 worktree 或非最新 tag 已经包含目标协议包，也要检查 `go.mod` 实际解析到的 semver tag 是否真的带了该包；缺口要在 workflow 中显式记录，而不是假设主线已就绪。

## 可复用排查线索

- 症状
  - Win 无法编译 `protocol/stream`
  - `Stream` 页能连上控制面但没有文本或统计更新
  - `wails generate module` 因 proto 漂移失败
- 触发条件
  - 当前依赖仍锁在 `myflowhub-proto v0.1.5`
  - `session.frame` 没有进入 `SubProtoStream` runtime
  - 页面直接依赖原始帧而不是 `stream.*` 业务事件
- 关键词
  - `protocol/stream`
  - `replace github.com/yttydcs/myflowhub-proto`
  - `stream.delivery`
  - `stream.text`
  - `stream.stats`
  - `DeliverySnapshot`
- 快速检查
  - 查看 `go.mod` 是否仍带开发态 `replace`
  - 查看 `app.go` 是否桥接 `stream.delivery` / `stream.text` / `stream.stats`
  - 查看 `internal/services/stream/runtime.go` 是否只解析 `SubProtoStream`
  - 查看前端是否只消费 `stream.*` 业务事件，而不是直接解析 `session.frame`

## 关键设计决策与权衡

- 决策：`Stream` 作为独立模块，而不是并入 `Flow`
  - 原因：避免把 workflow 编排和流式 delivery 模型混在同一套 UI/服务语义里
- 决策：Go runtime 负责解析 `SubProtoStream`，前端只消费业务事件
  - 原因：协议细节集中管理，更容易做节流、有界缓存和错误兜底
- 决策：首版只让 `text` 真正可读，`music` / `video` / `custom` 先展示统计摘要
  - 原因：先把控制面、观察面和扩展接口做对，再留真实播放器到后续 workflow
- 决策：开发态允许临时 `replace`，但把它明确标记为 release blocker
  - 原因：先让 Win 模块实现和验证闭环，避免被上游 semver 缺口卡死

## 测试与验证方式 / 结果

- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Win/frontend`
  - `npm test -- src/stores/stream.test.ts`
  - 结果：通过（1 个文件，2 个用例）
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：仍会打印 `Not found: time.Time`，但退出码为 0
- `MyFlowHub-Win/frontend`
  - `npm run build`
  - 结果：通过
  - 备注：Vite 继续提示主 bundle 超过 `500 kB`，本轮未处理 code splitting

## 潜在影响与回滚方案

- 潜在影响
  - Win 现在暴露新的 `StreamService`、`/stream` 页面和 `stream.*` 事件
  - 当前开发分支仍依赖临时 proto `replace`，正式 release 前必须切回正式 semver
  - `music` / `video` / `custom` 目前只有观察能力，没有真实播放
- 回滚方案
  - 回退 `go.mod`
  - 回退 `internal/services/stream/`
  - 回退 `app.go`
  - 回退 `frontend/src/router/index.ts`
  - 回退 `frontend/src/layout/AppShell.vue`
  - 回退 `frontend/src/pages/Stream.vue`
  - 回退 `frontend/src/stores/stream.ts`
  - 回退 `frontend/src/stores/stream.test.ts`
  - 回退 `docs/requirements/stream.md`
  - 回退 `docs/specs/stream.md`
  - 回退 `docs/requirements/README.md`
  - 回退 `docs/specs/README.md`
  - 回退 `docs/lessons/wails-binding-proto-drift.md`
  - 回退本归档和 `docs/change/README.md`

## 子Agent执行轨迹

- 未使用子Agent
