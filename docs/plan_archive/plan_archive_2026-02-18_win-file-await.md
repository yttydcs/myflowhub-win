# Plan - Win：File(list/read_text) 改为 send+await（SDK v1 Awaiter）（PR18-WIN-File-Await）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`refactor/win-file-ctrl-await`
- Worktree：`d:\project\MyFlowHub3\worktrees\pr18-file-ctrl-await\MyFlowHub-Win`
- Base：`main`
- 参考：
  - `d:\project\MyFlowHub3\target.md`
  - `d:\project\MyFlowHub3\repos.md`
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文）
- 依赖（本地 replace/junction）：
  - `..\MyFlowHub-Core` / `..\MyFlowHub-Proto` / `..\MyFlowHub-SDK`

## 约束（边界）
- 仅改 File 子协议的“控制类”接口（先覆盖 `list/read_text`）：
  - `internal/services/file`：`List/ReadText` 改为 send+await（等待 `read_resp`）；
  - `*Simple` 默认 timeout=`8s`（已确认，全子协议统一）；
  - `code != 1` 视为业务失败并返回 error（已确认，全子协议统一）。
- 不改 wire（SubProto/action/KindCtrl 前缀/JSON 字段均不变）。
- 不改 File 传输状态机（`pull/offer`、DATA/ACK、`transfer.go` 大部分逻辑不动）。
- 保持事件链路不变：
  - matched frame 仍会发布 `session.frame`；
  - `FileService` 仍通过 `session.frame` 解析并发出 `file.list/file.text/file.tasks/file.offer`。

## 当前状态（事实，可审计）
- Win 已接入 SDK v1 Awaiter，且 `await.Client.SetOnFrame` 已用于“matched frame 也能发布 `session.frame`”。
- File 模块当前行为：
  - `List/ReadText` 为 fire-and-forget：只发送 `file.read`，不等待 `read_resp`；
  - UI 依赖 `file.list/file.text` 事件更新；
  - 在断线/无响应场景下，UI 可能出现“loading 不结束”的体验问题（因为没有事件回包）。
- 本轮 workflow 已确认：Server 会补齐 `read_resp/write_resp` 继承请求 `MsgID/TraceID`；SDK Awaiter 会支持 File CTRL `KindCtrl+JSON` 解包。

---

## 1) 需求分析

### 目标
1) `List/ReadText` 调用方在 `8s` 内得到明确成功/失败/超时（Promise resolve/reject）。
2) 保持 UI 行为不变：仍由事件（`file.list/file.text`）驱动目录列表与预览内容更新。

### 范围（必须 / 不做）
#### 必须（本 PR）
- 后端：
  - `internal/services/file/service.go`：`List/ReadText` 改为 `SendCommandAndAwait` 并匹配 `read_resp`。
  - `ListSimple/ReadTextSimple`：默认 `8s` timeout。
- 前端（最小兜底，避免 await 引入未处理 reject）：
  - `frontend/src/stores/file.ts`：对 `requestList/openPreview` 增加错误兜底，确保 loading 状态可收敛并展示错误信息。

#### 不做（本 PR）
- 不 await 化 `pull/offer`（传输链路保持事件驱动与任务状态机）。
- 不重构 File 的协议实现与数据结构。

### 验收标准
- `ListSimple/ReadTextSimple`：
  - 成功：返回 `nil`
  - 失败（`code!=1`）：返回 error（包含 `msg/code`）
  - 超时：返回 `context deadline exceeded`
- UI：
  - 目录列表与预览仍由 `file.list/file.text` 更新；
  - 超时/断线时 loading 能结束并展示错误。

---

## 2) 架构设计（分析）

### 总体方案（采用）
- `FileService.List/ReadText`：
  - 仍发送 `MajorCmd + SubProtoFile` 到 hub；
  - payload 仍为 `KindCtrl + JSON(action=read,data=ReadReq)`；
  - 通过 `SessionService.SendCommandAndAwait(..., expectAction=read_resp)` 等待响应；
  - 解析响应 `ReadResp.code/msg`：
    - `code==1` → 成功返回 nil
    - `code!=1` → 返回 error（msg 优先，否则携带 code）
- 事件链路保持：
  - SDK Awaiter `onFrame` tap 保障 matched frame 仍发布 `session.frame`；
  - FileService 继续从 `session.frame` 驱动 `file.list/file.text`。

### 备选对比
- 备选 A：继续 fire-and-forget（不采用）
  - 无法在超时内明确失败；断线/无回包体验差，且不利于脚本化调用。
- 备选 B：前端实现“等待响应”逻辑（不采用）
  - 等待语义会在前端重复实现，且与 SDK/后端的 cancel/timeout 分叉。

### 错误与安全
- 不改变权限与路由；错误仍以 payload `code/msg` 表达。
- send+await 仅增加等待与解包，不引入新的敏感数据流。

### 性能与测试策略
- 性能：CTRL 帧体积小；await 增加一次 JSON 解包，开销可忽略。
- 测试：
  - 回归：`go test ./... -count=1 -p 1`
  - 冒烟：Win Connect → File 刷新目录 + 文本预览（均在 `8s` 内返回）

---

## 3.1) 计划拆分（Checklist）

## 问题清单（阻塞：否）
- 已确认：File 也沿用 `*Simple` 默认 `8s` + `code!=1` 返回 error；允许本 PR 同步改 Win+SDK+Server 保持一致性。

### WFA1 - 后端：List/ReadText 改为 send+await
- 目标：`list/read_text` 等控制动作等待 `read_resp` 并显式返回业务成功/失败。
- 涉及文件：
  - `internal/services/file/service.go`
- 验收条件：
  - `List/ReadText` 等待 `read_resp`；
  - `ListSimple/ReadTextSimple` 默认 `8s` timeout；
  - `code!=1` 返回 error。
- 回滚点：
  - revert 本提交（回到 fire-and-forget）。

### WFA2 - 前端：兜底处理 await reject
- 目标：避免 await 化后出现未处理 Promise reject；确保 loading 状态收敛并展示错误。
- 涉及文件：
  - `frontend/src/stores/file.ts`
- 验收条件：
  - `requestList/openPreview` 失败时能结束 loading 并展示错误信息（不必依赖事件回包）。
- 回滚点：
  - revert 前端提交（后端仍可保留 await）。

### WFA3 - 回归测试（Windows）
- 命令：
  - `$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'`
  - `New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null`
  - `go test ./... -count=1 -p 1`
- 验收条件：通过。

### WFA4 - 冒烟（手动）
- 步骤：
  1) Win 启动并 Connect 到 server
  2) 打开 File 页面，刷新目录（list）
  3) 选择文本文件预览（read_text）
- 验收条件：
  - 两类操作均在 `8s` 内成功/失败返回；
  - UI 仍通过事件更新目录/预览；异常时 loading 能收敛。

### WFA5 - Code Review（阶段 3.3）+ 归档变更（阶段 4）
- 归档文件：
  - `docs/change/2026-02-18_win-file-await.md`
- 验收条件：
  - Review 覆盖：需求/架构/性能/安全/测试；
  - 归档包含：任务映射、关键决策、测试命令与回滚方案。

