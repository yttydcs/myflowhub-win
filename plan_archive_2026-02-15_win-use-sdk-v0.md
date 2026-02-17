# Plan - Win 接入 MyFlowHub-SDK v0（Session/Transport）（PR2-WIN-1）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`refactor/win-sdk-v0`
- Worktree：`d:\project\MyFlowHub3\worktrees\pr2-win-sdk-v0\MyFlowHub-Win`
- 参考总目标：
  - `d:\project\MyFlowHub3\target.md`
  - `d:\project\MyFlowHub3\repos.md`
- 约束：commit 信息使用中文（前缀如 `refactor:` 可英文）

## 当前状态
- Win 当前存在“客户端底层能力”的重复实现：
  - `internal/session/session.go`：connect/send/readLoop + trace_id/hop_limit 默认值补齐
  - `internal/services/transport/codec.go`：`{action,data}` JSON envelope 编码
- `MyFlowHub-SDK`（`github.com/yttydcs/myflowhub-sdk`）已完成 v0：
  - `session`：connect/close/send/readLoop（HeaderTcp v2）
  - `transport`：EncodeMessage/DecodeMessage
- 目标态约束（已确认）：Win 是更上层应用，应通过统一 SDK/Client 调用能力，而不是在 Win 内重复实现协议机制。

> 环境备注（不进 git）：本仓库 `go.mod` 已使用 `replace ../MyFlowHub-Core`、`../MyFlowHub-Proto`。  
> 本 workflow 需要额外提供 `../MyFlowHub-SDK` 目录（本地联调 replace），已在 worktree 外层通过 Junction 指向 `repo/MyFlowHub-SDK`。

---

## 1) 需求分析

### 目标
1) Win 引入并依赖 `MyFlowHub-SDK v0`，收敛客户端侧底层能力到统一实现。
2) **保持 UI/绑定层不变**：Wails bindings 与 `internal/services/*` 的对外 API 不变。
3) 薄封装替换：
   - `internal/session`：改为委托 `myflowhub-sdk/session`
   - `internal/services/transport`：改为委托 `myflowhub-sdk/transport`

### 范围
#### 必须（本 PR）
- `go.mod` 引入 `github.com/yttydcs/myflowhub-sdk`（开发期通过 `replace ../MyFlowHub-SDK` 联调）。
- `internal/services/transport/codec.go`：保持 `EncodeMessage` API 不变，内部改为调用 SDK。
- `internal/session/session.go`：保持 `Session` 对外 API 不变，内部改为调用 SDK（Connect/Close/Send）；保留 `Login`/`LoginLegacy`（如存在）为兼容入口。
- 回归：`go test ./... -count=1 -p 1` 通过。

#### 不做（本 PR）
- 不移除现有 `internal/session` 与 `internal/services/transport` 包路径（保持最小 diff，后续再删）。
- 不新增 SDK v1（Broker/Awaiter）或各子协议强类型 client。
- 不改协议 wire（action/JSON/SubProto 不变）。

### 使用场景
- 用户在 Win 里连接 HubServer 并发起管理/认证/文件等请求时：
  - HeaderTcp v2 编解码一致
  - trace_id/hop_limit 默认规则一致
  - `{action,data}` envelope 编码一致

### 验收标准
- Win `go.mod` 已引入 `myflowhub-sdk` 且构建通过。
- `internal/session` 与 `internal/services/transport` 已变为薄封装（不再维护重复实现）。
- `go test ./... -count=1 -p 1` 通过。

### 风险
- SDK 与 Win 之间存在行为细节差异（例如 Close 时是否触发 error event）；本 PR 以“保持功能可用”为目标，并在归档文档中明确差异点与回滚方案。

---

## 2) 架构设计（分析）

### 总体方案（含选型理由 / 备选对比）
#### 方案 A（采用）：保留 Win 内部包路径，内部委托 SDK
- 优点：改动面最小；`internal/services/*` 与 bindings 无需调整；便于“小步多 PR”。
- 缺点：短期内仍存在一层转发包装（但可在后续 PR 删除并直连 SDK）。

#### 方案 B（不采用，本 PR 不做）：全量改 import，删除 internal/session 与 internal/services/transport
- 优点：结构更干净。
- 缺点：触及文件多，PR 变大且回滚成本高。

### 模块职责
- `myflowhub-sdk/session`：统一 TCP Session + HeaderTcp v2 编解码 + 默认字段补齐
- `myflowhub-sdk/transport`：统一 `{action,data}` envelope 编解码
- `myflowhub-win/internal/session`：Win 内部兼容壳（薄封装）
- `myflowhub-win/internal/services/transport`：Win 内部兼容壳（薄封装）

### 性能与测试策略
- 性能关键点：
  - 读循环依旧使用 `bufio.Reader`（降低 syscall）
  - DecodeMessage 使用 `json.RawMessage`（减少反序列化）
- 回归：
  - `GOTMPDIR=d:\project\MyFlowHub3\.tmp\gotmp; go test ./... -count=1 -p 1`

---

## 3.1) 计划拆分（Checklist）

## 问题清单（阻塞：否）
- 无（范围、依赖与验收已明确）。

### WSDK1 - 引入 myflowhub-sdk 依赖
- 目标：Win 具备 SDK 依赖与本地 replace 联调能力。
- 涉及文件：
  - `go.mod`
  - `go.sum`（由 tidy 生成）
- 验收条件：
  - `go test ./...` 编译通过。
- 测试点：
  - `go mod tidy` 后依赖解析正确。
- 回滚点：
  - revert `go.mod/go.sum`。

### WSDK2 - transport 薄封装委托 SDK
- 目标：`internal/services/transport.EncodeMessage` 复用 SDK 实现。
- 涉及文件：
  - `internal/services/transport/codec.go`
- 验收条件：
  - 编译通过；现有调用点无需改动。
- 测试点：
  - `go test ./...`（编译覆盖）。
- 回滚点：
  - revert 本提交。

### WSDK3 - session 薄封装委托 SDK
- 目标：`internal/session.Session` 复用 SDK Session 实现，避免重复维护底层逻辑。
- 涉及文件：
  - `internal/session/session.go`
- 验收条件：
  - `internal/services/session` 无需改动即可工作。
- 测试点：
  - `go test ./...`（编译覆盖）。
- 回滚点：
  - revert 本提交。

### WSDK4 - 全量回归
- 目标：确保最小改动不破坏功能。
- 验收条件：
  - `go test ./... -count=1 -p 1` 通过。
- 回滚点：
  - revert 本 PR。

### WSDK5 - Code Review + 归档
- 目标：可审计、可交接。
- 涉及文件：
  - `docs/change/2026-02-15_win-use-sdk-v0.md`
- 验收条件：
  - 包含：需求覆盖/架构/性能/安全/测试结论与回滚方案。
