# Plan - 协议仓库拆分（Proto）+ Win 上移为应用层（Win）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`refactor/proto-extract`
- Worktree：`d:\project\MyFlowHub3\worktrees\proto-extract\MyFlowHub-Win`
- 目标 PR：PR1（跨多个 repo 同步提交/合并）

## 项目目标（PR1）
1) Win 作为更上层应用：不“像 server 一样”实现协议机制，仅通过 SDK/协议库调用 server 能力。
2) 移除对 `MyFlowHub-Server` 仓库的 Go 依赖：Win 只依赖 Core + Proto（避免因 Server 内部重构导致 Win 被迫同步）。

## 已确认信息
- `MyFlowHub-Proto` module：`github.com/yttydcs/myflowhub-proto`
- wire：action 名称/消息结构/SubProto 值均不变（策略 A）

## 范围
### 必须（PR1）
- 将所有 `github.com/yttydcs/myflowhub-server/protocol/*` import 切换为 `github.com/yttydcs/myflowhub-proto/protocol/*`
- 更新 `go.mod`：移除 `myflowhub-server` require/replace；新增 `myflowhub-proto` require/replace
- 回归验证：`go test ./...`

### 不做（本 PR）
- UI/前端大改
- 协议 wire 改动（action/结构不变）
- 将 Win 内部 transport.EncodeMessage 全量迁移到 Proto（可作为 PR2：SDK 化）

## 问题清单（阻塞：否）
- 无

## 任务清单（Checklist）

### W1 - 切换协议 import 到 Proto
- 目标：Win 不再通过 Server repo 获取协议 types/常量。
- 涉及模块/文件（预期）：
  - `internal/services/*`（auth/varpool/topicbus/file/flow/management 等）
- 验收条件：
  - `rg "myflowhub-server/protocol" internal` 无结果。
  - `go test ./...` 编译通过（后续整体验证）。
- 回滚点：
  - revert 相关 import 变更。

### W2 - go.mod 解耦 Server 依赖
- 目标：`go.mod` 不再 require/replace `myflowhub-server`。
- 涉及模块/文件：
  - `go.mod`
- 验收条件：
  - `go test ./...` 通过。
- 回滚点：
  - revert `go.mod`。

### W3 - 全量回归
- 目标：确保 Win 在依赖切换后保持稳定。
- 验收条件：
  - `go test ./... -count=1` 通过。

## 依赖关系
- 依赖 Proto 仓库提供对应 `protocol/*` 包。

## 风险与注意事项
- 若 Win 某处使用了 Server 的“实现层”代码（非 protocol），需要回到阶段 3.1 增补任务并确认（本 PR 目标是彻底移除 Server 依赖）。

