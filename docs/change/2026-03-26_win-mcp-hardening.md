# Win MCP Client Hardening

## 变更背景 / 目标

- 首版 `myflowhub-mcp` 已可完成基础链路，但对长期 AI 集成来说仍有几个明显短板：
  - tool 失败主要是字符串消息，AI 不易稳定判断阻塞类型
  - `session_status` 暴露的是原始状态快照，缺少更直接的 readiness / hints
  - 启动脚本固定依赖 `go run`，部署侧对 Go 环境耦合偏强
  - Codex 安装依赖手工编辑 `config.toml`
- 本轮目标是在不改变 Hub 角色权限模型的前提下，补强 MCP client 的错误契约、状态摘要、启动链路和安装体验。

## 具体变更内容

- 更新 `internal/mcp/server.go`
  - 为 tool 错误结果引入统一结构 `code / message / hint / details`
  - `unknown tool` 也返回结构化错误
  - `initialize` 的 `tools` capability 明确声明 `listChanged=false`
- 更新 `internal/mcp/tools.go`
  - `myflowhub_session_status` 现在返回：
    - `auth`
    - `defaults`
    - `config`
    - `permissions`
    - `readiness`
    - `hints`
  - tool 失败统一收敛到：
    - `invalid_arguments`
    - `not_connected`
    - `missing_identity`
    - `write_disabled`
    - `upstream_error`
  - 保持服务端 RBAC 为授权真相，仅保留本地 `allow_write` gate
  - `varstore_set` 仍要求存在 `value` 字段，但允许空字符串作为变量值
- 更新 `scripts/start-myflowhub-mcp.ps1`
  - 新增 `-PreferSource`
  - 默认优先查找：
    - `MYFLOWHUB_MCP_EXE`
    - `build/bin/myflowhub-mcp.exe`
    - repo root 下 `myflowhub-mcp.exe`
    - repo root 下 `bin/myflowhub-mcp.exe`
  - 找不到二进制时 fallback 到 `go run ./cmd/myflowhub-mcp`
- 新增 `scripts/install-codex-myflowhub-mcp.ps1`
  - 以幂等方式安装或更新 Codex `mcp_servers.<name>`
  - 支持 `-WhatIf`
  - 已修正 `-WhatIf` 下不应创建目录的副作用
- 更新稳定文档
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`

## Requirements impact: updated

## Specs impact: updated

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-mcp-hardening\docs\requirements\mcp-client.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-mcp-hardening\docs\specs\mcp-client.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `DOCS-1`
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
- `IMPL-1`
  - `internal/mcp/server.go`
  - `internal/mcp/tools.go`
  - `internal/mcp/server_test.go`
  - `internal/mcp/tools_test.go`
- `IMPL-2`
  - `scripts/start-myflowhub-mcp.ps1`
  - `scripts/install-codex-myflowhub-mcp.ps1`
- `IMPL-3`
  - 回归验证与 README 对齐
- `ARCHIVE-1`
  - `docs/change/README.md`
  - `docs/change/2026-03-26_win-mcp-hardening.md`

## 经验 / 教训摘要

- 对 AI 直接消费的 MCP tool，字符串错误远远不够；至少要返回稳定 `code` 和明确 `hint`，否则 host 侧提示词需要补过多脆弱逻辑。
- `session_status` 不应只是“原始状态转发”，还应包含 readiness 和下一步建议，这能显著降低 AI 对流程状态的误判。
- 安装脚本如果支持 `-WhatIf`，就必须避免任何落盘副作用，包括提前建目录。

## 可复用排查线索

- 症状：
  - AI 收到错误，但无法判断是未连接、未登录还是写 gate 未开启
  - 同一台机器既能开发态 `go run`，又希望部署态优先走已编译 exe
  - Codex 安装时担心覆盖其他 MCP server 配置
- 触发条件：
  - tool 结果只有文本错误
  - host 没有 Go 环境
  - `config.toml` 中已存在同名 `mcp_servers.myflowhub`
- 关键词：
  - `invalid_arguments`
  - `missing_identity`
  - `write_disabled`
  - `install-codex-myflowhub-mcp.ps1`
  - `MYFLOWHUB_MCP_EXE`
  - `listChanged`
- 快速检查：
  - 调 `myflowhub_session_status` 看 `permissions/readiness/hints`
  - 调 `myflowhub_varstore_set` 且不启 `--allow-write`，确认返回 `write_disabled`
  - `powershell -ExecutionPolicy Bypass -File scripts/install-codex-myflowhub-mcp.ps1 -WhatIf`
  - `powershell -ExecutionPolicy Bypass -File scripts/start-myflowhub-mcp.ps1 --version`

## 关键设计决策与权衡

- 决策：不新增本地 owner/target 白名单
  - 原因：当前系统里 Hub 角色权限仍是唯一授权真相，本地只保留默认只读 gate。
- 决策：`session_status` 增加 `permissions/readiness/hints`
  - 原因：比单纯暴露原始 snapshot 更适合 AI 做分支判断。
- 决策：启动脚本优先已编译二进制，缺失时再 fallback 到源码运行
  - 原因：兼顾部署态稳定性和开发态便利性。
- 决策：安装脚本直接更新 Codex `config.toml`
  - 原因：把易错的手工配置改成可重复执行的脚本化入口。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-hardening`
  - 结果：通过
- `$env:GOWORK='off'; go test ./... -count=1`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-hardening`
  - 结果：通过
- `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-hardening`
  - 结果：通过
- `powershell -ExecutionPolicy Bypass -File scripts/start-myflowhub-mcp.ps1 --version`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-hardening`
  - 结果：通过
- `powershell -ExecutionPolicy Bypass -File scripts/install-codex-myflowhub-mcp.ps1 -ConfigPath <temp> -WhatIf`
  - 结果：通过，且未创建目录
- `powershell -ExecutionPolicy Bypass -File scripts/install-codex-myflowhub-mcp.ps1 -ConfigPath <temp> -Endpoint 127.0.0.1:9000`
  - 结果：通过，成功写入 `mcp_servers.myflowhub`

## 潜在影响与回滚方案

- 潜在影响：
  - AI host 若依赖旧的非结构化错误文本，需要切换到 `structuredContent.code` 等字段
  - 启动脚本现在会优先命中 repo 内二进制，若用户保留了旧 exe，可能不是最新源码
- 回滚方案：
  - 回退 `internal/mcp/server.go`、`internal/mcp/tools.go`、对应测试、`scripts/start-myflowhub-mcp.ps1`、`scripts/install-codex-myflowhub-mcp.ps1`
  - 删除不再需要的 `scripts/install-codex-myflowhub-mcp.ps1`
  - 若需恢复手工配置方式，移除 README 与 requirements/specs 中安装脚本说明

## 子Agent执行轨迹

- none
