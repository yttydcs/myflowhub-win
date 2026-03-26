# Authority Admin Console

## Background

- 当前 Win 端只有单一的 `Permissions` 页面，主要覆盖 authority 角色/权限策略编辑。
- `MyFlowHub-Server` 的 auth 协议已经支持待审批注册、批准/拒绝、permit 签发/撤销，但 Win 端没有对应 GUI 入口。
- 现有权限页把 authority 解析、策略编辑、运行时快照都堆叠在一页里，可读性和可操作性都偏弱。

## Goal

- 将 Win Authority 管理能力重构为一组独立入口，分别覆盖权限编排、注册审批和准入许可签发。
- 提高权限编排页面的可用性，让管理员可以更低成本地理解和修改策略。

## Scope

### Must

- 左侧导航新增一组 Authority 相关入口，分别对应：
  - 权限编排
  - 注册审批
  - 准入许可
- 英文环境下使用全英文名称和说明；中文环境下使用全文中文名称和说明。
- 权限编排页面必须继续支持 authority 解析、policy 加载/保存、运行时角色预览和单节点权限查询。
- 注册审批页面必须支持：
  - 查看待审批注册列表
  - 刷新列表
  - 批准单条注册请求
  - 拒绝单条注册请求
- 准入许可页面必须支持：
  - 签发 permit
  - 撤销 permit
  - 展示最近一次成功签发的 permit 结果

### Optional

- 为待审批列表提供轻量筛选或统计信息，只要不引入新的协议依赖。
- 为最近签发 permit 提供快捷复制或回收操作。

### Out of Scope

- 不修改 `MyFlowHub-Server` 协议。
- 不新增 permit 列表或 permit 查询能力。
- 不自动把 permit token 长期持久化到本地配置。
- 不重构 Home 页登录/注册主流程。

## Scenarios

- Authority 管理员希望在单独页面处理首次注册审批，而不是在调试页或 CLI 中手动发协议。
- 管理员希望快速给指定设备签发一次性准入 permit，并在必要时撤销。
- 管理员希望更容易理解默认角色、角色权限和节点覆盖之间的关系，并安全地保存到 authority。

## Functional Requirements

1. 应用左侧必须出现 3 个独立 Authority 管理入口。
2. 权限编排页面必须支持 authority 解析，并清楚展示当前登录身份、authority 和运行时校验状态。
3. 3 个 authority 页面都必须使用当前 session 自动解析 authority，并且不得暴露手动 authority override 输入或内部解析原因。
4. 权限编排页面必须支持编辑 `auth.default_role`、`auth.default_perms`、`auth.node_roles`、`auth.role_perms`。
5. 权限编排页面保存后必须能可选执行持久化、运行时应用、失效广播和运行时校验。
6. 注册审批页面必须能展示每条 pending request 的 `request_id`、`device_id`、申请角色、显示名、创建时间和过期时间。
7. 注册审批页面必须允许管理员对单条 pending request 执行 approve 或 reject。
8. approve 时允许角色留空，由 authority 协议自行决定默认行为；前端不得强行填充默认角色替代空值。
9. 准入许可页面必须允许管理员输入 `device_id`、`role` 和可选过期时间来签发 permit。
10. 准入许可页面必须允许管理员通过 permit token 撤销 permit。
11. permit 页面至少要展示当前会话最近一次成功签发的结果，方便立即复制或回收。

## Non-functional Requirements

- 复用现有 Win `AuthService`、`ManagementService` 和 authority orchestration 逻辑，不引入不必要的新协议层。
- 页面职责必须清晰，避免把 authority 管理重新堆回单个巨型页面。
- 所有 authority 页面都必须在未连接、未登录、缺少本地身份时显式阻止危险操作。
- 权限编排 UI 必须优先提升信息分层、表单可读性和保存反馈，不仅是把原有字段平移到新布局里。
- permit token 默认只保留会话态展示，不在本地配置里长期保存。

## Edge Cases

- authority 无法解析。
- 待审批列表为空。
- approve 的 role 为空字符串。
- reject 的 reason 为空。
- issue permit 时缺少 `device_id` 或 `role`。
- revoke permit 时 token 为空。
- permit 页面无法回读历史 permit，只能展示最近一次成功签发结果。

## Acceptance Criteria

1. 左侧导航中能看到 3 个独立 Authority 管理入口，且中英文环境分别保持全英文 / 全中文文案。
2. 权限编排页面可以完成 load/save policy、runtime 预览和节点权限查询，且交互体验明显优于原版本。
3. 注册审批页面可以完成待审批列表查询、批准和拒绝的基础链路。
4. 准入许可页面可以完成 permit 签发与撤销，并清楚展示最近一次签发结果。
5. 本轮改动不要求新增 Server 协议，也不引入 permit 历史持久化。

## Related Specs

- [authority-admin-console.md](../specs/authority-admin-console.md)
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`

## Related Changes

- [2026-03-18_win-authority-permissions-v1.md](../change/2026-03-18_win-authority-permissions-v1.md)
