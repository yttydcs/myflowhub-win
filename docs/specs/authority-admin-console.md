# Authority Admin Console Spec

## Scope

- 本规范限定 Win 端 Authority 管理控制台的页面拆分、服务接口和交互边界。
- 本规范不修改 `MyFlowHub-Server` auth 协议，也不新增 permit list/query 契约。

## Interfaces / Contracts

### 1. 页面与路由契约

- Authority 管理必须拆分为 3 个独立页面：
  - `Access Policy`
  - `Registration Approvals`
  - `Permit Issuance`
- 左侧导航必须把这 3 个页面放在同一组下。
- 中文环境中的 `Access Policy` 必须显示为“访问策略”。
- 原单页 policy 能力迁移到 `Access Policy` 页面后，不得丢失 authority 解析、policy 读写和 runtime 校验链路。
- `Access Policy` 页面必须采用 tab 结构，至少包含：
  - `Current Policy`
  - `Role Management`

### 2. Wails authority orchestration 契约

- Win 端 authority orchestration 继续由现有 `PermissionService` 暴露给前端。
- `PermissionService` 至少必须提供以下接口：
  - `ResolveAuthority`
  - `LoadPolicy`
  - `SavePolicy`
  - `GetNodePerms`
  - `ListPendingRegisters`
  - `ApproveRegister`
  - `RejectRegister`
  - `IssueRegisterPermit`
  - `RevokeRegisterPermit`

### 3. Auth typed action 契约

- `AuthService` 必须提供对下列 auth action 的强类型封装：
  - `list_pending_registers`
  - `approve_register`
  - `reject_register`
  - `issue_register_permit`
  - `revoke_register_permit`
- 所有强类型封装必须保留 authority 返回的错误码 / 消息语义，不得吞错或私自改写业务状态。

### 4. policy 页面职责边界

- `Access Policy` 页面负责：
  - authority 解析
  - policy load / save
  - runtime list_roles 预览
  - `auth.get_perms` 单节点查询
- `Current Policy` tab 负责：
  - default role 选择
  - default perms 目录化勾选
  - node role overrides
  - save options
  - runtime snapshot / node perms lookup
- `Role Management` tab 负责：
  - role list 编辑
  - role 权限目录化勾选
  - role 级 unknown perms 保留展示
- 不负责待审批注册处理或 permit 生命周期管理。
- policy 页不得要求用户手动编辑权限 CSV；权限必须通过前端 catalog 勾选后再序列化为现有配置键。

### 5. approval 页面职责边界

- `Registration Approvals` 页面负责：
  - authority 解析
  - pending list 读取
  - approve / reject 动作
- approve 的 `role` 可以为空字符串；前端只做显式字段校验，不改变空值语义。
- approve / reject 成功后必须回刷 pending list。

### 6. permit 页面职责边界

- `Permit Issuance` 页面负责：
  - authority 解析
  - issue permit
  - revoke permit
  - 展示当前会话最近一次成功签发的 permit 结果
- 页面不得假装支持“历史 permit 列表”；若协议没有该能力，文案必须明确边界。

## Data Model or Protocol

### 1. 前端 authority 上下文

- 3 个 authority 页面都依赖同一组本地上下文字段：
  - `sourceId`
  - `hubId`
  - `authorityOverride`
  - `authorityId`
  - `authorityReason`
- 身份变化时，页面 store 必须清空与上一次 authority 绑定的数据，避免跨身份脏状态。

### 2. 待审批注册模型

- 前端与 Win service 之间使用以下稳定字段：
  - `requestId`
  - `deviceId`
  - `requestedRole`
  - `displayName`
  - `createdAt`
  - `expiresAt`
- 上述字段来源于 auth 协议 `PendingRegisterInfo`。

### 3. permit 结果模型

- permit 页面最近一次成功签发结果至少包含：
  - `permit`
  - `deviceId`
  - `role`
  - `expiresAt`
- 该结果只保留在运行时内存态，不写入本地 `settings.json`。

### 4. 访问策略表单模型

- 前端 `Access Policy` 编辑态至少包含：
  - `defaultRole`
  - `defaultPerms`
  - `defaultPermsUnknown`
  - `nodeRoles`
  - `rolePerms[{ role, perms, unknownPerms }]`
- `defaultPermsUnknown` 与 `rolePerms[].unknownPerms` 用于保留 catalog 之外的历史权限。
- 页面保存时必须把上述结构重新组装为既有 `Policy`：
  - `defaultPerms = known + unknown`
  - `rolePerms[].perms = known + unknown`
- 若已知权限集合中包含 `*`，前端应把它视为独占选择，不再同时保留其他已知权限项。

## Error Handling

- authority 未解析、source/hub 身份缺失时，前端必须在本地失败，不发送请求。
- 必填参数为空时，Go service 必须立即返回错误。
- auth action 返回 `code != 1` 时，必须透传为显式失败，不得静默吞掉。
- 页面动作失败时，必须保持当前可见状态稳定，不得因为一次失败清空已加载的全部数据。
- role 名重复、空 role、非法 node ID、非法分隔符等错误必须在前端显式校验。

## Security / Safety

- permit token 视为敏感准入凭证，默认不做本地持久化。
- reject / revoke 这类破坏性动作必须是显式按钮触发，不得在输入联动中隐式执行。
- authority 页面只在已连接、已登录且具备本地身份时允许提交写操作。
- catalog 外权限默认保留而不是静默丢弃，避免旧配置或未来权限点被页面误删。

## Performance Constraints

- 页面不得通过高频轮询维持 authority 数据新鲜度；刷新应由显式用户动作或成功写操作后的必要回刷触发。
- approve / reject 成功后只回刷 pending list，不连带重载 policy 或 runtime list_roles。
- permit issue / revoke 不应引入额外重复读取；只有当前页面需要的数据才刷新。
- `Access Policy` tab 切换只切前端视图，不得因为切 tab 触发额外 authority 请求。

## Related Requirements

- [authority-admin-console.md](../requirements/authority-admin-console.md)

## Related Changes

- [2026-03-18_win-authority-permissions-v1.md](../change/2026-03-18_win-authority-permissions-v1.md)
