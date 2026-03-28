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
- 页面不得暴露手动 authority override 输入，也不展示内部 authority 解析 reason。

### 2. Wails authority orchestration 契约

- Win 端 authority orchestration 继续由现有 `PermissionService` 暴露给前端。
- `PermissionService` 至少必须提供以下接口：
  - `ResolveAuthority`
  - `LoadPolicy`
  - `SavePolicy`
  - `GetNodePerms`
  - `ListPendingRegisters`
  - `ListRegisterPermits`
  - `ApproveRegister`
  - `RejectRegister`
  - `IssueRegisterPermit`
  - `RevokeRegisterPermit`

### 3. Auth typed action 契约

- `AuthService` 必须提供对下列 auth action 的强类型封装：
  - `list_pending_registers`
  - `list_register_permits`
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
  - 默认准入摘要展示
  - 通过弹窗编辑 default role
  - 通过弹窗编辑 default perms 权限列表
  - 紧凑 node override 列表展示
  - 通过弹窗新建或编辑单条 node override
  - 在默认准入卡和节点覆盖卡附近提供就近保存入口
  - 紧凑 operations panel
  - 在 operations panel 中承载 save options、runtime snapshot、node perms lookup
  - runtime details 默认折叠，仅在需要时展开
  - policy 加载中时展示页面内 loading notice
- `Role Management` tab 负责：
  - 紧凑单行 role 列表展示
  - 在页头提供就近保存入口
  - 通过单列弹窗编辑单个 role
  - role 权限的权限列表式 add / remove 编辑
  - 通过独立权限选择列表追加权限，而不是直接改写现有权限行
  - role 级 unknown perms 保留展示
  - policy 加载中时展示页面内 loading notice
- `Role Management` tab 的 role 列表与 role 权限列表必须保持当前紧凑单行风格，作为长期 UI 基线：
  - 优先保证单行扫读节奏，而不是回退为高密度多行块状卡片
  - role 列表行应把名称、基线标识、摘要信息和主要操作收敛在同一行或同一视觉节奏内
  - role 权限列表行应保持“权限名 + 分组/简述 + 删除动作”的紧凑行结构，而不是重新引入矩阵勾选或可直接改写的行内表单
- `Current Policy` 与 `Role Management` 中的就近保存入口必须复用同一条 `savePolicy(saveOptions)` 链路，不得引入局部保存协议。
- 不负责待审批注册处理或 permit 生命周期管理。
- policy 页不得要求用户手动编辑权限 CSV；权限必须通过前端 catalog 勾选后再序列化为现有配置键。

### 5. approval 页面职责边界

- `Registration Approvals` 页面负责：
  - authority 解析
  - pending list 读取
  - approve / reject 动作
- pending request 主视图必须是紧凑摘要列表，不得为每条 request 持续展开 approve / reject 输入区。
- 审批详细输入必须集中在单个 `review dialog` 中；`role` 与 `reason` 共享同一聚焦面，避免拆成两个并列编辑块。
- approve 的 `role` 可以为空字符串；前端只做显式字段校验，不改变空值语义。
- approve / reject 成功后必须回刷 pending list。

### 6. permit 页面职责边界

- `Permit Issuance` 页面负责：
  - authority 解析
  - 读取活动 permit 列表
  - issue permit
  - 从活动 permit 列表行内 revoke permit
- 主视图必须采用“顶部动作栏 + 活动 permit 列表”结构，不得继续维持 latest-only 结果卡。
- issue 的详细输入必须放在按需打开的独立 dialog 中；revoke 直接从列表行内触发，不再要求用户手动输入 token。
- 页面只展示活动 permit，不展示已消费 / 已撤销 / 已过期的 permit 历史。
- 当解析出的 `authorityId != sourceId` 时，页面必须进入 authority-local 受限态：
  - 不再自动请求 permit 列表
  - 不再把 timeout 作为普通加载失败展示
  - 显式提示当前需在 authority 节点本机登录后再进行 permit 管理

## Data Model or Protocol

### 1. 前端 authority 上下文

- 3 个 authority 页面都依赖同一组本地上下文字段：
  - `sourceId`
  - `hubId`
  - `authorityId`
- authority 由共享 store 基于当前 session 自动解析；页面只读展示最终 `authorityId`。
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

### 3. 活动 permit 模型

- 前端与 Win service 之间使用以下稳定字段：
  - `permit`
  - `deviceId`
  - `role`
  - `issuedBy`
  - `issuedAt`
  - `expiresAt`
- 上述字段来源于 auth 协议 `RegisterPermitInfo`。

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
- 页面还必须维护局部弹窗编辑态：
  - 默认准入编辑弹窗
  - 节点覆盖编辑弹窗
  - 角色编辑弹窗
  - 角色权限选择弹窗
- `defaultPermsUnknown` 与 `rolePerms[].unknownPerms` 用于保留 catalog 之外的历史权限。
- 页面保存时必须把上述结构重新组装为既有 `Policy`：
  - `defaultPerms = known + unknown`
  - `rolePerms[].perms = known + unknown`
- 若已知权限集合中包含 `*`，前端应把它视为独占选择，不再同时保留其他已知权限项。
- 权限列表编辑器必须禁止自由输入任意权限字符串；新增权限只能从内置 catalog option 中选择。
- 角色编辑弹窗中的已选权限列表必须为只读展示，每项仅保留删除动作。
- 角色编辑弹窗中的权限列表样式必须保持紧凑单行摘要，避免把权限项渲染为高块状卡片或多列编辑区。
- 节点覆盖编辑器必须校验 `nodeId > 0`、role 非空且 nodeId 不重复。
- 删除角色时，若该角色仍被默认准入或 node override 引用，前端必须阻止删除并显式提示。
- 编辑角色名时，前端应同步更新当前页面中 default role 与 node override 对旧角色名的引用，避免产生悬空引用。

## Error Handling

- authority 未解析、source/hub 身份缺失时，前端必须在本地失败，不发送请求。
- 必填参数为空时，Go service 必须立即返回错误。
- auth action 返回 `code != 1` 时，必须透传为显式失败，不得静默吞掉。
- 页面动作失败时，必须保持当前可见状态稳定，不得因为一次失败清空已加载的全部数据。
- permit 管理动作在 `sourceId != authorityId` 的 remote authority 场景下，Win orchestration 层必须快速返回 authority-local 错误，而不是继续等待 auth timeout。
- `loadPolicy()` 进行中时，页面必须展示稳定的页面内 loading notice，不能只依赖按钮 disabled 状态传达反馈。
- role 名重复、空 role、非法 node ID、非法分隔符等错误必须在前端显式校验。

## Security / Safety

- permit token 视为敏感准入凭证，默认不做本地持久化。
- reject / revoke 这类破坏性动作必须是显式按钮触发，不得在输入联动中隐式执行。
- authority 页面只在已连接、已登录且具备本地身份时允许提交写操作。
- catalog 外权限默认保留而不是静默丢弃，避免旧配置或未来权限点被页面误删。
- 角色编辑弹窗的滚动容器必须预留足够内边距，避免 focus ring 被 overflow 裁切。

## Performance Constraints

- 页面不得通过高频轮询维持 authority 数据新鲜度；刷新应由显式用户动作或成功写操作后的必要回刷触发。
- approve / reject 成功后只回刷 pending list，不连带重载 policy 或 runtime list_roles。
- permit issue / revoke 不应引入额外重复读取；只有当前页面需要的数据才刷新。
- `Access Policy` tab 切换只切前端视图，不得因为切 tab 触发额外 authority 请求。

## Related Requirements

- [authority-admin-console.md](../requirements/authority-admin-console.md)

## Related Changes

- [2026-03-18_win-authority-permissions-v1.md](../change/2026-03-18_win-authority-permissions-v1.md)
