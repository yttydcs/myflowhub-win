# Plan - Win：TopicBus 改为 send+await（SDK v1 Awaiter）（PR16-WIN-TopicBus-Awaiter）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`refactor/win-topicbus-await`
- Worktree：`d:\project\MyFlowHub3\worktrees\pr16-win-topicbus-await\MyFlowHub-Win`
- Base：`main`
- 参考：
  - `d:\project\MyFlowHub3\target.md`
  - `d:\project\MyFlowHub3\repos.md`
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文）
- 依赖（本地 replace/junction）：
  - `..\MyFlowHub-Core` / `..\MyFlowHub-Proto` / `..\MyFlowHub-SDK` 均通过 Junction 指向 `repo/`，用于 `go test`。

## 约束（边界）
- 仅改 `MyFlowHub-Win`：
  - `internal/services/topicbus`：将可等待的动作改为 send+await，并补齐默认超时。
  - 如需要：新增少量通用 helper（仅限 topicbus/service.go 内部）。
- 不改 wire（action/JSON/SubProto 不变）。
- 不改前端 store（仍通过 `session.frame` 解析并更新 UI）。
- 默认 timeout：`8s`（已确认，全子协议统一）。
- `code != 1` 视为错误并返回 error（已确认，全子协议统一）。

## 当前状态（事实，可审计）
- Win 侧已接入 SDK v1 Awaiter（PR13/PR14/PR15）：
  - `SessionService.SendCommandAndAwait` 已可用；
  - `await.Client.SetOnFrame` 已接入，确保“被 Awaiter 匹配并 deliver 的帧”仍会发布 `session.frame`，前端 store 不会丢帧。
- TopicBus 当前仍为 fire-and-forget：
  - `internal/services/topicbus/service.go` 仅调用 `SessionService.SendCommand(...)`，不等待 `*_resp`。
  - 前端 `frontend/src/stores/topicbus.ts` 监听 `session.frame` 并解析 `publish` 更新事件列表。

---

## 1) 需求分析

### 目标
1) 将 TopicBus 的常用控制动作升级为 “发送并等待响应”：
   - `subscribe` → `subscribe_resp`
   - `subscribe_batch` → `subscribe_batch_resp`
   - `unsubscribe` → `unsubscribe_resp`
   - `unsubscribe_batch` → `unsubscribe_batch_resp`
   - `list_subs` → `list_subs_resp`
2) `*Simple` 方法使用默认 timeout=`8s`，避免无 ctx 调用无限等待。
3) 保持前端行为不变：仍能收到 `session.frame` 并按现有 store 逻辑更新 UI（publish 事件不受影响）。

### 范围（必须 / 不做）
#### 必须（本 PR）
- `internal/services/topicbus/service.go`
  - 上述动作改为 send+await，并按对应 resp action 匹配。
  - 解包 resp 的 `code/msg`：
    - `code==1` → 成功返回 nil
    - `code!=1` → 返回 error（msg 优先，否则带 code）
- `Subscribe*Simple/Unsubscribe*Simple/ListSubsSimple`
  - 使用 `context.WithTimeout(..., 8s)` 包装。
- 回归：`go test ./... -count=1 -p 1`（Windows）。

#### 不做（本 PR）
- `publish` 不做 await（协议无 `publish_resp`）。
- 不调整前端 store 与页面 UI。
- 不改 Server/Core/SDK/Proto。
- 不为 `Send(action, data)` 引入通用 await（避免引入不确定的 expectAction 设计）。

### 使用场景
- UI 执行 TopicBus Subscribe/Unsubscribe/ListSubs 时：
  - 希望能在超时内明确获得成功/失败，而不是只能“发了请求”；
  - 失败时在 UI 侧可立即提示（Promise reject）。
- UI 的 publish 事件仍由 `session.frame` 驱动（现状：`frontend/src/stores/topicbus.ts` 只处理 action=`publish`）。

### 功能需求
- 输入校验：
  - `topic` 非空（subscribe/unsubscribe/publish）
  - `topics` 非空（batch）
  - `name` 非空（publish）
- Await 匹配：
  - `MsgID + SubProto(TopicBus=4) + Action(*_resp)`。
- 错误语义：
  - SDK/Session 返回的连接错误/超时错误原样透传；
  - 业务错误通过 `data.code/msg` 判断并返回 error。

### 非功能需求
- 性能：避免不必要的二次解包/拷贝；等待逻辑不持有全局锁，避免影响 Connect/Close。
- 可维护性：保持变更集中在 `topicbus/service.go`，便于后续迁移 Flow/File 复用模板。
- 可回滚：单独提交，可一键 revert 回 fire-and-forget。

### 输入输出
- 输入：Wails bindings 调用 `*Simple` 或 `*(ctx)` 方法（sourceID/targetID/topic/topics/...）。
- 输出：
  - 成功：nil
  - 失败：error（含 `msg` 或 `code`）
  - 超时：context deadline exceeded

### 边界异常
- 未连接/未初始化：返回明确错误（由 `SessionService`/SDK 返回）。
- 响应 payload 无法解包或缺少 `code`：返回解包错误（协议实现问题）。
- 若服务端未返回可匹配的 `(MsgID, Action)`，await 将超时（调用方可见）。

### 验收标准
- Subscribe/Unsubscribe/ListSubs（含 batch）：调用方 Promise 能在 timeout 内返回，并能区分成功/失败/超时。
- 前端仍能收到 `session.frame` 并按既有 store 逻辑更新 publish 事件列表。

---

## 2) 架构设计（分析）

### 总体方案（采用）
- `internal/services/topicbus` 使用 `SessionService.SendCommandAndAwait`：
  - 请求帧仍为 `MajorCmd`（逐跳进入 handler），响应帧为 `MajorOKResp`（由 Server TopicBus 统一构造）。
  - Awaiter 匹配规则：`MsgID + SubProto(TopicBus=4) + Action(*_resp)`。
- 解析响应：
  - subscribe/unsubscribe（含 batch）：解包为 `topicbus.Resp`
  - list_subs：解包为 `topicbus.ListResp`
  - 根据 `code` 判定成功/失败。
- `*Simple` 方法默认 `8s` 超时，避免无限等待。
- 保持 `session.frame` 事件链路：
  - 由 SDK v1 `onFrame` tap 保证 matched frame 也会回调 `SessionService.handleFrame`。

### 备选对比
- 备选 A：继续 fire-and-forget，仅靠 frame 更新 UI（不采用）
  - 调用方无法同步获知业务结果；错误提示与脚本化能力弱。
- 备选 B：前端实现等待（不采用）
  - 等待语义会在前端重复实现；断线/取消/超时处理与 SDK/Win 侧产生分叉。

### 模块职责
- `internal/services/session`
  - 提供 `SendCommandAndAwait`（底层委托 SDK await.Client）。
  - 发布 `session.frame` 事件（供前端 store 消费）。
- `internal/services/topicbus`
  - 子协议业务 API：输入校验、resp code 判定、返回 error。
- `frontend/src/stores/topicbus.ts`
  - 继续消费 `session.frame`（只处理 action=`publish`）更新 UI（本 PR 不改）。

### 数据 / 调用流（关键链路）
1) 前端调用：`TopicBusService.SubscribeSimple(...)`
2) Go：构造 `{action,data}` payload → `SendCommandAndAwait(ctx, sub=4, expect=subscribe_resp)`
3) SDK await：readLoop 收到 resp：
   - onFrame：发布 `session.frame`（store 不受影响）
   - deliver：匹配成功 → 唤醒等待者 → TopicBusService 解析 `code/msg` 并返回

### 接口草案
- `TopicBusService`：各方法签名保持不变；内部实现切换到 await。
- 新增内部 helper（可选）：
  - `func (s *TopicBusService) sendAndAwait(ctx context.Context, sourceID, targetID uint32, payload []byte, reqAction, respAction string, out any) error`

### 错误与安全
- 输入校验（topic/topics/name 必填）。
- `code!=1` 视为业务失败并返回 error；不依赖 header major 表达业务失败。
- 不引入新的敏感数据；不改变权限模型。

### 性能与测试策略
- 性能：topicbus 控制帧体积小、频率可控；await 额外 JSON 解包成本可忽略。
- 测试：
  - 回归：`go test ./... -count=1 -p 1`
  - 冒烟（手动）：Win Connect → Subscribe/Unsubscribe/ListSubs（timeout 内返回；UI publish 仍正常）

### 可扩展性设计点
- 为后续 Flow/File 的 await 迁移提供可复制模板（统一 timeout、统一 code/msg 判定、统一 frame tap 语义）。

---

## 3.1) 计划拆分（Checklist）

## 问题清单（阻塞：否）
- 已确认：默认 timeout=8s；`code!=1` 返回 error；本 PR 仅改 Win topicbus。

### TBA1 - TopicBus：控制动作改为 send+await
- 目标：把 TopicBus 的 subscribe/unsubscribe/list_subs（含 batch）升级为 send+await，并将 `code/msg` 显式回传给调用方。
- 涉及文件：
  - `internal/services/topicbus/service.go`
- 验收条件：
  - subscribe/subscribe_batch/unsubscribe/unsubscribe_batch/list_subs：均等待对应 `*_resp` 并按 `code` 返回。
- 测试点：
  - 断线/未连接错误可见；
  - 超时返回 context 错误。
- 回滚点：
  - revert 本提交（回到 fire-and-forget）。

### TBA2 - Simple：默认 timeout=8s
- 目标：避免 *Simple 无限等待；行为与 Auth/Management/VarPool 一致。
- 涉及文件：
  - `internal/services/topicbus/service.go`
- 验收条件：
  - 所有 *Simple 包装 `context.WithTimeout(..., 8s)`。
- 回滚点：
  - revert。

### TBA3 - 回归测试（Windows）
- 命令：
  - `$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'`
  - `New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null`
  - `go test ./... -count=1 -p 1`
- 验收条件：通过。

### TBA4 - Code Review（阶段 3.3）+ 归档变更（阶段 4）
- 归档文件：
  - `docs/change/2026-02-17_win-topicbus-await.md`
- 验收条件：
  - Review 覆盖：需求/架构/性能/安全/测试；
  - 归档包含：任务映射、关键决策、测试命令与回滚方案。

