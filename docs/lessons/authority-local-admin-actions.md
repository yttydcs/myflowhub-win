# Authority Local Admin Actions

## Summary

- “审批 / permit 只能在 authority 本机操作” 已不再是当前稳定基线。
- 现在的正式约束是：任何已登录且具备对应 `auth.*` 权限的节点，都可以通过 remote authority 链路发起审批和 permit 管理。
- 如果 remote authority 管理仍失败，优先怀疑的是消费者依赖版本、authority 路由归属或旧 guard 没有清干净，而不是继续把问题归因为“必须 authority-local”。

## Lookup Hints

- 症状关键词
  - `auth list_pending_registers: request timed out`
  - `auth list_register_permits: request timed out`
  - `requires authority-local session`
  - `routed source`
  - `source mismatch`
- 触发条件
  - `sourceId != authorityId`
  - remote authority 场景
  - Win permit / approval 页面
- 快速检查
  - 对比当前 session `nodeId` 与解析出的 `authorityId`
  - 检查是否仍依赖旧版 `github.com/yttydcs/myflowhub-subproto/auth`
  - 检查 Win 页面或 service 是否还残留 authority-local guard
  - 检查 authority 侧 `SourceID -> 当前入站连接` 的路由归属

## Symptoms

- `Registration Approvals` 或 `Permit Issuance` 在 remote authority 场景仍报：
  - `auth list_pending_registers: request timed out`
  - `auth list_register_permits: request timed out`
  - `requires authority-local session`
- authority 本机可操作，但其它已登录管理节点仍被 UI 或 service 层阻断。
- 请求看起来已经发出，但 authority 侧日志出现 routed source / source mismatch 类拒绝。

## Impact

- 已具备权限的远程管理节点无法正常审批注册或签发 / 撤销 permit。
- Win 页面会给出与当前规范冲突的限制提示，增加定位成本。
- 团队容易误把问题解释成“必须 authority-local”，从而忽略真正的版本或路由缺口。

## Trigger Conditions

- `sourceId != authorityId`
- 消费方仍依赖旧版 `myflowhub-subproto/auth`
- Win orchestration / 页面层残留 authority-local guard
- authority 尚未建立 `SourceID -> 当前入站连接` 的 descendant route index

## Root Cause

- 旧的 Win workaround 把“当时后端能力未补齐”的事实固化成了 UI 和 service 的长期 guard。
- 在 auth runtime 已补齐 remote authority admin 后，如果消费者不同时清理这些旧 guard，就会出现“协议已经支持、Win 仍自我阻断”的行为背离。
- 某些失败并非 Win 问题，而是消费者依赖版本或 authority 路由归属仍停留在旧状态。

## Investigation Trail

1. 先对照 `requirements/specs`，确认稳定目标已经改为 remote authority admin 可用。
2. 查看 auth runtime worktree，确认 admin actions 已支持以 `TargetID=authority` 转发，并保留真实 `SourceID` 做权限校验。
3. 在 Win 侧搜索 `requires authority-local session`、`authority-local`、`data-permit-remote-authority`，定位到残留 guard。
4. 删除 guard 后，再用 Go / Vitest / Wails generate / build 验证行为闭环。
5. 若运行时仍失败，再继续查依赖版本和 authority 路由归属，而不是回退到 authority-local 解释。

## Resolution

- 升级到包含 remote authority admin 的 `myflowhub-subproto/auth` 版本，并同步 Server/Win 消费方依赖。
- 清理 Win service 层和页面层的 authority-local 快速失败逻辑。
- 确认请求头保持 `SourceID=真实操作者`、`TargetID=authority`，不要引入 Win 专用 action。
- 若 authority 仍拒绝请求，优先修复 descendant route index，而不是绕过 source 校验。

## Prevention / Guardrails

- 不要再把 authority-local guard 当成长期方案；它只适用于历史版本排障。
- 行为升级后，必须同步更新 requirements/specs/lessons，避免 UI、runtime 和文档再次背离。
- remote authority 管理失败时，先查版本与 route ownership，再查页面状态；不要直接回退为“只能 authority 本机操作”。

## Related Requirements / Specs / Changes

- Requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\docs\requirements\authority-admin-console.md`
- Specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-remote-authority-admin\docs\specs\authority-admin-console.md`
  - `D:\project\MyFlowHub3\worktrees\feat-server-remote-authority-admin\docs\specs\auth.md`
- Changes:
  - [2026-03-28_win-remote-authority-admin.md](../change/2026-03-28_win-remote-authority-admin.md)
  - [2026-03-28_win-permit-remote-authority-guard.md](../change/2026-03-28_win-permit-remote-authority-guard.md)
