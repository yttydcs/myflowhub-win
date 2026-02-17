# Plan Archive - Win：Management 改为 send+await（SDK v1 Awaiter）（PR14-WIN-Mgmt-Awaiter）

> 归档来源：`plan.md`（PR15 开始前归档）
> 归档日期：2026-02-17

# Plan - Win：Management 改为 send+await（SDK v1 Awaiter）（PR14-WIN-Mgmt-Awaiter）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`refactor/win-mgmt-await`
- Worktree：`d:\project\MyFlowHub3\worktrees\pr14-win-mgmt-await\MyFlowHub-Win`
- Base：`main`
- 参考：
  - `d:\project\MyFlowHub3\target.md`
  - `d:\project\MyFlowHub3\repos.md`
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文）
- 依赖（本地 replace/junction）：
  - `..\MyFlowHub-Core` / `..\MyFlowHub-Proto` / `..\MyFlowHub-SDK` 均通过 Junction 指向 `repo/`，用于 `go test`。

## 约束（边界）
- 仅改 `MyFlowHub-Win`：
  - `internal/services/management`：全量动作改为 send+await。
  - 如需要：新增少量通用 helper（仅限 management/service.go 内部）。
- 不改 wire（action/JSON/SubProto 不变）。
- 不改前端 store（仍通过 `session.frame` 解析并更新 UI）。
- 默认 timeout：`8s`（你已确认）。
- `code != 1` 视为错误并返回 error（你已确认）。

## 当前状态（事实，可审计）
- Win 侧已经接入 SDK v1 Awaiter（PR13），并通过 `await.Client.SetOnFrame` 保持 `session.frame` 事件不缺失：
  - 即：即便响应帧被 Awaiter “匹配并 deliver”，仍会被 `SessionService.handleFrame` 发布到 eventbus/Wails 前端。
- Management 当前仍为 fire-and-forget：
  - `internal/services/management/service.go` 仅调用 `SessionService.SendCommand(...)`，不等待对应 `*_resp`。
  - 前端通过 `frontend/src/stores/management.ts` 监听 `session.frame` 并解析 `*_resp` 更新 UI。

---

## 1) 需求分析（已确认）

### 目标
1) 将 Management 的常用动作升级为 “发送并等待响应”：
   - `node_echo`
   - `list_nodes`
   - `list_subtree`
   - `config_get`
   - `config_set`
   - `config_list`
2) `*Simple` 方法使用默认 timeout=`8s`，避免无 ctx 调用无限等待。
3) 保持前端行为不变：仍能收到 `session.frame` 并按现有 store 逻辑更新 UI。

### 范围（必须 / 不做）
#### 必须（本 PR）
- `internal/services/management/service.go`
  - 各请求改为 send+await，并按对应的 `*_resp` action 匹配。
  - 解包 resp 的 `code/msg`：
    - `code==1` → 成功返回 nil
    - `code!=1` → 返回 error（msg 优先，否则带 code）
- `NodeEchoSimple/ListNodesSimple/ListSubtreeSimple/Config*Simple`
  - 使用 `context.WithTimeout(..., 8s)` 包装。
- 回归：`go test ./... -count=1 -p 1`（Windows）。

#### 不做（本 PR）
- 不把其它子协议迁移到 await（VarStore/TopicBus/File/Flow 等后续另起 workflow）。
- 不调整前端 store 与页面 UI。
- 不改 Server/Core/SDK。

### 输入输出
- 输入：Wails bindings 调用 `*Simple` 或 `*(ctx)` 方法（sourceID/targetID/key/value/message）。
- 输出：
  - 成功：nil
  - 失败：error（含 `msg` 或 `code`）
  - 超时：context deadline exceeded

### 边界异常
- 未连接/未初始化：返回明确错误（由 `SessionService`/SDK 返回）。
- 响应 payload 无法解包或缺少 action/code：返回解包错误（协议实现问题）。

### 验收标准
- Management 的上述 6 个动作：调用方 Promise 能在 timeout 内返回，并能区分成功/失败/超时。
- 前端仍能收到 `session.frame` 并按既有逻辑更新列表/配置等 UI 状态。

### 风险
- 若服务端某条 management 响应不携带可匹配的 `(MsgID, Action)`（或 action 与预期不一致），await 会超时；当前 Server 的 management 使用 `kit.SendResponse`/`BuildTCPResponse`，应当继承 `MsgID/TraceID`，风险可控。

---

## 2) 架构设计（分析）

### 总体方案（采用）
- `internal/services/management` 使用 `SessionService.SendCommandAndAwait`：
  - 请求帧仍为 `MajorCmd`（逐跳进入 handler），响应帧为 `MajorOKResp`（由 Server 统一构造）。
  - Awaiter 匹配规则：`MsgID + SubProto(Management=1) + Action(*_resp)`。
- 解析 `resp.Message.Data` 为对应的 proto struct（来自 `myflowhub-proto/protocol/management`），根据 `code` 判定成功/失败。
- `*Simple` 方法默认 `8s` 超时，避免无限等待。
- 保持 `session.frame` 事件链路：
  - 由 PR13 引入的 `await.Client.onFrame` 保证 matched frame 也会回调 `SessionService.handleFrame`。

### 备选对比
- 备选 A：继续 fire-and-forget，仅靠 frame 更新 UI（不采用）
  - 问题：调用方无法同步获得业务成功/失败；错误体验差，且不利于后续自动化脚本/测试。
- 备选 B：前端自己实现等待（不采用）
  - 问题：等待语义在前端分叉，无法与 SDK/Win/CLI 统一；也更难处理断线/取消/超时。

### 模块职责
- `internal/services/session`
  - 提供 `SendCommandAndAwait`：统一发送 + 等待 + 超时/取消语义（底层委托 SDK Awaiter）。
  - 发布 `session.frame` 事件（供前端 store 消费）。
- `internal/services/management`
  - 为 Management 子协议提供业务级 API（输入校验、resp code 判定、返回 error）。
- `frontend/src/stores/management.ts`
  - 继续消费 `session.frame` 更新 UI（本 PR 不改）。

### 数据 / 调用流（关键链路）
1) 前端调用 Wails binding：`ManagementService.NodeEchoSimple(...)`
2) Go：构造 `{action,data}` payload → `SessionService.SendCommandAndAwait(ctx, sub=1, expect=node_echo_resp)`
3) SDK await：发送请求；readLoop 收到 resp：
   - onFrame：先发布 `session.frame` → 前端 store 仍能更新 UI
   - deliver：匹配成功 → 唤醒等待者 → ManagementService 解析 `code/msg` 并返回

### 接口草案
- `ManagementService`：各方法保持签名不变；内部实现切换到 await。
- 新增内部 helper（可选）：
  - `func (s *ManagementService) sendAndAwait(ctx, sourceID, targetID uint32, reqAction, respAction string, data any) error`

### 错误与安全
- 仍执行输入校验（message/key 非空等）。
- `code!=1` 视为业务失败并返回 error；不依赖 header major 表达业务失败。
- 不引入新的敏感数据；不改变权限模型。

### 性能与测试策略
- 性能：management 请求频率低，await 增量成本可忽略；主要收益是更确定的交互语义。
- 测试：
  - 回归：`go test ./...`
  - 冒烟（手动）：Win Connect → NodeEcho/ListNodes/ConfigList（timeout 内返回；UI 仍更新）

### 可扩展性设计点
- 本 PR 为 “子协议逐步迁移到 await” 提供可复制模板；后续可按相同方式迁移 VarStore/TopicBus 等。

---

## 3.1) 计划拆分（Checklist）

## 问题清单（阻塞：否）
- 已确认：默认 timeout=8s；`code!=1` 返回 error；仅改 Win management。

### MWA1 - Management：全量动作改为 send+await
- 目标：把 management 的 6 个动作升级为 send+await，并将 `code/msg` 显式回传给调用方。
- 涉及文件：
  - `internal/services/management/service.go`
- 验收条件：
  - node_echo/list_nodes/list_subtree/config_get/config_set/config_list：均等待对应 `*_resp` 并按 `code` 返回。
- 测试点：
  - 断线/未连接错误可见；
  - 超时返回 context 错误。
- 回滚点：
  - revert 本提交（回到 fire-and-forget）。

### MWA2 - Simple：默认 timeout=8s
- 目标：避免 *Simple 无限等待；行为与 Auth 一致。
- 涉及文件：
  - `internal/services/management/service.go`
- 验收条件：
  - 所有 *Simple 包装 `context.WithTimeout(..., 8s)`。
- 回滚点：
  - revert。

### MWA3 - 回归测试（Windows）
- 命令：
  - `$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'`
  - `New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null`
  - `go test ./... -count=1 -p 1`
- 验收条件：通过。

### MWA4 - Code Review（阶段 3.3）+ 归档变更（阶段 4）
- 归档文件：
  - `docs/change/2026-02-17_win-mgmt-await.md`
- 验收条件：
  - Review 覆盖：需求/架构/性能/安全/测试；
  - 归档包含：任务映射、关键决策、测试命令与回滚方案。

