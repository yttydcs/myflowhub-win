# Win MCP Full Chain Smoke

## 变更背景 / 目标

- `MyFlowHub-Win` 的 MCP 工具面已经覆盖 `session/auth/management/exec/flow/varstore`，但仓内 smoke 入口仍停留在基础链路：
  - `initialize`
  - `tools/list`
  - `connect`
  - `register/login`
  - `auth_get_perms`
  - `auth_list_roles`
  - `management_list_nodes`
- 这导致 authority、management config、exec capability、flow、varstore 的真实 Hub 验收还需要手工拼调用。
- 本轮目标是在不新增 MCP 工具、不扩大协议范围的前提下：
  - 把 smoke 脚本扩成 staged full-chain 验收入口
  - 默认保持安全，只在显式 opt-in 时执行 authority / write 动作
  - 把 staged smoke 契约写回 requirements/specs/README

## 具体变更内容

- 重建并扩展 `scripts/test-myflowhub-mcp-smoke.ps1`
  - 保留现有基础链路兼容路径
  - 新增阶段开关：
    - `-EnableExtendedRead`
    - `-EnableAuthoritySmoke`
    - `-EnableWriteSmoke`
  - 新增 staged smoke 相关参数：
    - `ConfigKey`
    - `NodeEchoMessage`
    - `AuthorityID`
    - `PermitDeviceID`
    - `PermitRole`
    - `PendingRequestID`
    - `PendingAction`
    - `ApprovalRole`
    - `RejectReason`
    - `ExecutorNode`
    - `FlowID`
    - `FlowName`
    - `FlowMethod`
    - `VarName`
    - `VarValue`
    - `VarOwner`
  - 在每个已启用阶段开始前先检查 `tools/list` 已暴露该阶段所需工具
  - 输出阶段摘要，失败时保留 `ConfigDir`、临时资源名、cleanup 问题和 `stderr` tail
- 扩展 read-only smoke
  - 接入：
    - `myflowhub_management_node_info`
    - `myflowhub_management_node_echo`
    - `myflowhub_management_list_subtree`
    - `myflowhub_management_config_list`
    - `myflowhub_management_config_get`
    - `myflowhub_exec_cap_query`
    - `myflowhub_flow_list`
    - `myflowhub_flow_get`
    - `myflowhub_flow_status`
  - `config_get` 优先使用 `-ConfigKey`，否则回退到首个可读 key；无可读 key 时显式记录 skipped
  - `flow_get/status` 在无显式 `FlowID` 且 `flow_list` 为空时显式记录 skipped
- 接入 authority smoke
  - 永远先跑 `myflowhub_auth_list_pending_registers`
  - permit 流要求 `PermitDeviceID + PermitRole` 成对输入
  - approve/reject 要求显式 `PendingRequestID`
  - `Permit` 在同次 smoke 中立即 revoke，避免残留
- 接入 write smoke
  - 只在 `-EnableWriteSmoke` 时让 launcher 自动追加 `--allow-write`
  - 本地强制要求：
    - `-ExecutorNode`
    - `-FlowMethod`
  - 使用临时 flow / var 名称执行：
    - `myflowhub_varstore_list`
    - `myflowhub_varstore_set`
    - `myflowhub_varstore_get`
    - `myflowhub_varstore_revoke`
    - `myflowhub_flow_set`
    - `myflowhub_flow_list`
    - `myflowhub_flow_get`
    - `myflowhub_flow_run`
    - `myflowhub_flow_status`
    - `myflowhub_flow_delete`
  - cleanup 失败不吞掉，直接把残留资源暴露给调用者
- 更新稳定文档
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`

## Requirements impact: updated

## Specs impact: updated

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\docs\requirements\mcp-client.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke\docs\specs\mcp-client.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`

## Related lessons

- none

## 对应 `plan.md` / `todo.md` 任务映射

- `DOCS-1`
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `README.md`
- `SMOKE-1`
  - `scripts/test-myflowhub-mcp-smoke.ps1`
- `SMOKE-2`
  - `scripts/test-myflowhub-mcp-smoke.ps1`
- `SMOKE-3`
  - `scripts/test-myflowhub-mcp-smoke.ps1`
- `TEST-1`
  - `scripts/test-myflowhub-mcp-smoke.ps1`
  - `README.md`
- `REVIEW-1`
  - `todo.md`
  - `docs/change/2026-04-01_win-mcp-full-chain-smoke.md`
- `ARCHIVE-1`
  - `docs/change/README.md`
  - `docs/change/2026-04-01_win-mcp-full-chain-smoke.md`

## 经验 / 教训摘要

- 真实 Hub smoke 不应把“能连上”当作“全链路可用”；阶段化验证能把风险和副作用拆开。
- authority / write smoke 如果不做本地前置校验，会把明显缺参错误变成真实环境上的无意义状态操作。
- write smoke 的最小安全基线不是“尽量清理”，而是“资源命名可追踪 + cleanup 失败可见”。

## 可复用排查线索

- 症状：
  - MCP 基础 smoke 通过，但 authority / flow / varstore 的真实 Hub 验收仍不清楚怎么跑
  - 写 smoke 看似成功，但共享环境出现临时 flow / var 残留
  - authority smoke 直接改了真实请求，事后却无法追踪是脚本哪一步做的
- 触发条件：
  - 仍在使用旧版只覆盖基础链路的 smoke 脚本
  - `--allow-write` 未显式打开
  - `ExecutorNode` / `FlowMethod` / `PendingRequestID` 等关键前置参数缺失
- 关键词：
  - `EnableExtendedRead`
  - `EnableAuthoritySmoke`
  - `EnableWriteSmoke`
  - `cleanup failed`
  - `ConfigGetKey`
  - `WriteFlowID`
  - `WriteVarName`
- 快速检查：
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Help`
  - `tools/list` 中确认 staged smoke 所需工具全部存在
  - 先跑 base / extended read，再决定是否启用 authority / write
  - 写 smoke 失败时优先看 `WriteFlowID`、`WriteVarName` 和 `CleanupIssues`

## 关键设计决策与权衡

- 决策：继续扩展现有 smoke 脚本，而不是另起一份 full-smoke 脚本
  - 原因：保持启动逻辑、JSON-RPC 驱动、错误处理和帮助文本集中，减少分叉维护。
- 决策：authority / write 只做显式 opt-in
  - 原因：真实 Hub 上这些动作天然有副作用，默认路径必须保持可安全重复执行。
- 决策：write smoke 只要求 `ExecutorNode` 和 `FlowMethod`，其余 flow/var 名称默认生成临时值
  - 原因：把环境相关的必要前提压到最小，同时保证生成资源可追踪可清理。
- 决策：`config_get` 允许在“无可读 key”时显式 skipped
  - 原因：真实 Hub 的 management config 暴露面可能随角色不同而变化，脚本要把这种差异说清，而不是伪造成功。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./internal/mcp -count=1`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
  - 结果：通过
- `$env:GOWORK='off'; go build ./cmd/myflowhub-mcp`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
  - 结果：通过
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Help`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
  - 结果：通过
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
  - 结果：按预期失败，明确提示 `Login mode requires -ConfigDir...`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir C:\temp\missing-smoke -EnableWriteSmoke -FlowMethod demo::run`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
  - 结果：按预期失败，明确提示 `Write smoke requires -ExecutorNode...`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir C:\temp\missing-smoke -EnableAuthoritySmoke -PendingAction approve`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
  - 结果：按预期失败，明确提示 `Pending approval smoke requires -PendingRequestID...`
- `git diff --check`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-full-chain-smoke`
  - 结果：无错误；仅提示工作区文件会在后续被 git 规范成 `CRLF`
- 真实 Hub 端到端 staged smoke
  - 结果：本轮未在当前环境实际连真实 Hub 执行；需用户在目标 Hub 环境下按 README 示例分阶段验证

## 潜在影响与回滚方案

- 潜在影响：
  - 维护者现在会看到一个更长的 smoke 帮助页和更多 staged 参数
  - authority / write smoke 若被显式启用，将真实影响共享 Hub 状态
  - 文档现在把 smoke 合同从“基础链路”升级为“分阶段全链路”，后续改脚本必须同步维护这三处稳定文档
- 回滚方案：
  - 回退 `scripts/test-myflowhub-mcp-smoke.ps1`
  - 回退 `docs/requirements/mcp-client.md`、`docs/specs/mcp-client.md`、`README.md`
  - 删除 `docs/change/2026-04-01_win-mcp-full-chain-smoke.md`
  - 从 `docs/change/README.md` 移除对应索引项

## 子Agent执行轨迹

- none
