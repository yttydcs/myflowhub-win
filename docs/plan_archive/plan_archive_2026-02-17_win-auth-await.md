# Plan - Win 接入 SDK v1 Awaiter（Auth send+await）（PR13-WIN-Awaiter）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`refactor/win-auth-await`
- Worktree：`d:\project\MyFlowHub3\worktrees\pr13-win-auth-await\MyFlowHub-Win`
- Base：`main`
- 参考：
  - `d:\project\MyFlowHub3\target.md`
  - `d:\project\MyFlowHub3\repos.md`
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文）
- 依赖：
  - 本 workflow 同步包含 SDK 分支 `feat/sdk-await-hooks`（提供 onFrame hook，保证 Win 的 `session.frame` 事件不缺失）。

## 当前状态（事实，可审计）
- 前端通过 Wails 事件 `session.frame` 监听 Auth 响应并更新 UI：
  - `frontend/src/stores/session.ts`：仅解析 `*_login_resp` / `*_register_resp` 结尾动作。
- 后端 `AuthService` 当前仅 send，不等待响应：
  - UI 层 Promise 在“发送成功但响应失败/丢失”时无法直接感知，只能依赖后续 frame 更新。
- SDK 已具备 v1 Awaiter，但 `await.Client` 默认会拦截匹配成功帧：
  - 若 Win 不具备“全帧 tap”，将丢失 `session.frame`（UI 无法更新）。

---

## 1) 需求分析

### 目标
1) Win 侧让 Auth 的 `register/login` 变为“发送并等待响应”：
   - 正常：收到 `*_resp` 且 `data.code==1` → 返回 nil。
   - 失败：`data.code!=1` → 返回 error（优先使用 `data.msg`）。
   - 超时：返回 timeout/cancel 错误。
2) 前端保持不改：仍通过 `session.frame` 事件解析并展示登录结果（即：await 不影响事件链路）。
3) 仅覆盖 Auth（不扩展到其它子协议）。

### 范围（必须 / 不做）
#### 必须（本 PR）
- `internal/session`：
  - 底层由 `myflowhub-sdk/await.Client` 承载 connect/send/await。
  - 通过 SDK onFrame hook 把所有接收帧继续回调给 `SessionService.handleFrame`（保证事件/日志不缺失）。
- `internal/services/session`：
  - 新增 `SendCommandAndAwait`（或等价能力）供 AuthService 使用。
- `internal/services/auth`：
  - `Register` / `Login` 使用 send+await。
  - `RegisterSimple` / `LoginSimple` 增加默认 timeout（可配置化后续再做，本 PR 先固定）。

#### 不做（本 PR）
- 不改 wire（action/JSON/SubProto 不变）。
- 不调整前端 store 逻辑。
- 不把其它协议迁移到 await（VarStore/TopicBus/File/Flow 等后续另起 workflow）。

### 验收标准
- Win：`go test ./... -count=1 -p 1` 通过（Windows）。
- 手动冒烟：
  1) 启动 `hub_server`
  2) Win Connect
  3) Home 页面 Register/Login：
     - UI 仍能收到 `session.frame` 更新 `lastAuthMessage/LoggedIn`；
     - 后端调用在 timeout 内返回（失败返回 error）。

### 风险
- 若服务端某条链路的响应不携带可匹配的 `(MsgID, Action)`，await 会超时；本 workflow 仅覆盖 Auth，且 Server 已在 PR8/PR10 修复相关问题。

---

## 2) 架构设计（分析）

### 总体方案（采用）
- 在 Win 内部 `internal/session` 迁移到 SDK `await.Client`：
  - 统一连接/发送/等待语义；
  - 通过 SDK 的 onFrame hook 维持现有“每帧事件”行为，保证前端/日志不缺失。
- 在 `AuthService` 中使用 `SessionService.SendCommandAndAwait`：
  - expectAction：
    - `register` → `auth.ActionRegisterResp`
    - `login` → `auth.ActionLoginResp`
  - 解析 `auth.RespData`：以 `code==1` 判定成功，否则返回 error（msg 优先）。

### 备选对比（为什么不选）
- 备选 A：仅在 Win 内自己做 broker+deliver（不改 SDK）
  - 问题：等待语义在 Win 分叉，实现重复，未来难统一；同时会复制 SDK 内部优化（HasMsgSub 快路径）。

---

## 3.1) 计划拆分（Checklist）

## 问题清单（阻塞：否）
- 已确认：
  1) 默认 timeout：`8s`（用于 `RegisterSimple/LoginSimple`）。
  2) `code != 1` 视为错误并返回 error（前端 catch 可见）。
  3) SDK 侧同步增加 `await.Client` 的 onFrame hook（向后兼容）。

### WA1 - Session：切换到 SDK await.Client 并保持 session.frame 不缺失
- 目标：接入 SDK Awaiter 的同时，维持 Win 现有 “接收帧 → eventbus → 前端” 的链路。
- 涉及文件：
  - `internal/session/session.go`
- 验收条件：
  - Connect/Close/Send 正常；
  - 接收帧仍会触发 `SessionService.handleFrame`（日志与前端事件一致）。
- 回滚点：
  - revert 本提交（回退到 SDK v0 Session）。

### WA2 - SessionService：提供 SendCommandAndAwait
- 目标：提供最小可复用的 send+await 能力（先给 Auth 用）。
- 涉及文件：
  - `internal/services/session/service.go`
- 验收条件：
  - 不持锁等待（避免阻塞 `handleFrame`）；
  - timeout/cancel 路径明确。
- 回滚点：
  - revert 本提交。

### WA3 - Auth：Register/Login 使用 send+await
- 目标：让 Home 页面调用可同步感知成功/失败/超时。
- 涉及文件：
  - `internal/services/auth/service.go`
- 验收条件：
  - 成功 `code==1` 返回 nil；
  - 失败 `code!=1` 返回 error；
  - 默认 timeout 生效（Simple 方法）。
- 回滚点：
  - revert 本提交（回到 fire-and-forget）。

### WA4 - 回归测试（Windows）
- 命令：
  - `$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'`
  - `New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null`
  - `go test ./... -count=1 -p 1`
- 验收条件：通过。

### WA5 - Code Review（阶段 3.3）+ 归档变更（阶段 4）
- 归档文件：
  - `docs/change/2026-02-17_win-auth-await.md`
- 验收条件：
  - Review 覆盖：需求/架构/性能/安全/测试；
  - 归档包含：任务映射、关键决策、测试命令与回滚方案。
