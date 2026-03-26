# Plan - MyFlowHub-Win MCP Client Hardening

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/win-mcp-hardening`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-hardening`
- Current Stage: `4 completed / waiting workflow end confirmation`

## Goal And Current State
- Goal:
  - 在不改变 Hub 角色权限模型的前提下，补强 Win MCP client 的可判读错误、状态自检、启动链路和 Codex 安装体验。
- Current State:
  - 首版 MCP client 已可完成 `connect/login/management/varstore` 基础链路。
  - 当前本地写保护仅有 `allow_write` 总开关，且这符合当前“服务端 RBAC 为真相、本地只做默认只读 gate”的边界。
  - 当前 tool 错误主要是字符串消息，不利于 AI 稳定判断阻塞类型。
  - 当前启动脚本固定依赖 `go run`，Codex 安装也依赖手动编辑配置。

## Docs Governance
- Requirements impact: clarify
- Specs impact: clarify
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-hardening\docs\requirements\mcp-client.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-hardening\docs\specs\mcp-client.md`
- Related lessons:
  - none

## Scope
- Must:
  - 为 MCP tools 返回统一结构化错误，明确区分 `invalid_arguments`、`not_connected`、`missing_identity`、`write_disabled`、`upstream_error`
  - 强化 `myflowhub_session_status`，明确暴露 auth/readiness/local write gate
  - 将 `scripts/start-myflowhub-mcp.ps1` 改为优先启动已编译二进制，缺失时再 fallback 到 `go run`
  - 在 `scripts/` 下新增 Codex MCP 安装脚本
  - 更新 requirements/specs/README
- Optional:
  - 对 MCP server 做最小兼容性补强，但不引入完整 SDK 迁移
- Out of scope:
  - 新增本地 owner/target 白名单
  - 改动 Hub 角色权限系统
  - 扩展新的协议工具集

## Parallelism Assessment
- 本轮实现写集高度重叠在 `internal/mcp/*`、`scripts/*`、`docs/*`，不适合派发子 Agent。
- Owner: 主 Agent

## Executable Checklist
- [x] DOCS-1 更新 `docs/requirements/mcp-client.md` 与 `docs/specs/mcp-client.md`
- [x] IMPL-1 统一 MCP tool 错误模型与状态摘要
- [x] IMPL-2 增强启动脚本并新增 Codex 安装脚本
- [x] IMPL-3 更新 README 与必要测试/验证
- [x] REVIEW-1 完成 3.3 代码复核
- [x] ARCHIVE-1 归档到 `docs/change`

## Task Details
### DOCS-1 - 文档契约更新
- Goal:
  - 把结构化错误、状态摘要、二进制优先启动和安装脚本写入稳定文档。
- Files:
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
- Acceptance:
  - 文档能准确描述新的 AI 使用与安装链路
- Tests:
  - 文档自检
- Rollback:
  - 回退文档到增强前状态

### IMPL-1 - Tool 错误模型与状态摘要
- Goal:
  - 让 AI 能从 MCP 结果稳定判断阻塞类型，并从 `session_status` 获取下一步行动线索。
- Files:
  - `internal/mcp/server.go`
  - `internal/mcp/tools.go`
  - `internal/mcp/tools_test.go`
  - `internal/mcp/server_test.go`
- Acceptance:
  - `session_status` 返回额外 readiness/permissions 摘要
  - tool 失败返回结构化 `code/message/hint/details`
  - 未连接、缺省身份、写入关闭、上游失败可区分
- Tests:
  - `go test ./internal/mcp -count=1`
- Rollback:
  - 回退错误模型与状态摘要实现

### IMPL-2 - 启动与安装链路
- Goal:
  - 降低对开发环境 Go 的强依赖，并提供可复用的 Codex 安装入口。
- Files:
  - `scripts/start-myflowhub-mcp.ps1`
  - `scripts/install-codex-myflowhub-mcp.ps1`
- Acceptance:
  - 启动脚本优先寻找本地 exe，缺失时 fallback 到 `go run`
  - 安装脚本可把 MCP 配置写入 Codex `config.toml`
- Tests:
  - `powershell -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 --version`
  - `powershell -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -WhatIf`
- Rollback:
  - 删除新增安装脚本并恢复启动脚本

### IMPL-3 - 回归验证与说明
- Goal:
  - 保证 README、脚本和 MCP 行为一致，并覆盖关键路径回归。
- Files:
  - `README.md`
  - 受影响测试文件
- Acceptance:
  - README 能直接指导 build / start / install
  - 关键测试与 smoke 通过
- Tests:
  - `$env:GOWORK='off'; go test ./... -count=1`
  - `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
- Rollback:
  - 回退说明与测试补充

## Risks And Notes
- PowerShell 下原地更新 TOML 需避免误删其他 MCP server 配置。
- 启动脚本的二进制查找路径必须可预测，避免与 GUI 产物混淆。
- 结构化错误要保持向后兼容：`content.text` 仍需可读，`structuredContent` 需更稳定。

阻塞：否
已完成 Stage 4，等待用户确认是否结束 workflow
