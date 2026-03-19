# 变更背景 / 目标

Win 端缺少独立的 authority 权限配置入口，当前无法在统一页面完成“持久化策略 + 运行时立即生效”的双链路操作。
本次 V1 目标是在不改 Server 协议的前提下，基于现有能力落地独立权限页，并支持 authority 级 role/perms 编辑、保存与回读验证。

# 具体变更内容

## 新增
- `internal/services/auth/permissions.go`
  - 新增 Auth 强类型权限接口：
    - `GetPerms / GetPermsSimple`
    - `ListRoles / ListRolesSimple`
    - `PushPermsSnapshot / PushPermsSnapshotSimple`
    - `InvalidatePerms / InvalidatePermsSimple`
- `internal/services/permission/service.go`
  - 新增 `PermissionService` 编排层：
    - `ResolveAuthority`
    - `LoadPolicy`
    - `SavePolicy`
    - `GetNodePerms`
  - 新增策略解析/归一化/序列化能力（`auth.default_role`、`auth.default_perms`、`auth.node_roles`、`auth.role_perms`）。
- `internal/services/permission/service_test.go`
  - 新增策略归一化与解析单测。
- `frontend/src/stores/permissions.ts`
  - 新增权限页面 store（authority 解析、策略加载/保存、单节点权限查询、身份切换状态重置）。
- `frontend/src/pages/Permissions.vue`
  - 新增独立权限页面：authority 覆盖、策略编辑、前端基础校验、保存选项（persist/apply/invalidate/refresh/verify）、运行时角色列表、节点权限查询。

## 修改
- `app.go`
  - 注入并绑定 `PermissionService`。
- `frontend/src/router/index.ts`
  - 注册新路由：`/permissions`。
- `frontend/src/layout/AppShell.vue`
  - 侧栏新增 `Permissions` 导航入口。

# 对应 plan.md 任务映射

- `WIN-PERM-V1-1`：Go 服务层扩展（完成）
- `WIN-PERM-V1-2`：前端权限页面与状态管理（完成）
- `WIN-PERM-V1-3`：Wails bindings 与类型对齐（完成，执行 `wails generate module`）
- `WIN-PERM-V1-4`：关键路径验证（完成）
- `WIN-PERM-V1-5`：Code Review（完成）
- `WIN-PERM-V1-6`：归档文档（本文件）

# 关键设计决策与权衡

- 持久化与运行时分步执行（非事务）：
  - 采用 `management config_set(auth.*)` + `auth perms_snapshot`，可选 `perms_invalidate(refresh=true)`。
  - 优点：无需改 Server 协议即可落地；
  - 代价：两步之间存在短暂不一致窗口。
- authority 解析策略：
  - `manual override -> authority.node_id -> hubId fallback`。
  - 优点：兼容现网配置与手工运维场景。
- 校验策略：
  - 前端做“明显非法输入”拦截（空值、重复、非法分隔符），后端保留最终校验。
  - 优点：降低无效请求与回包噪音，保证服务端安全边界。

# 测试与验证方式 / 结果

- 执行：
  - `GOWORK=off go test ./... -count=1`
  - `npm install`
  - `npm run build`
  - `GOWORK=off wails generate module`
- 结果：
  - Go 全量测试通过。
  - 前端构建通过。
  - Wails 绑定生成成功（`permission.PermissionService` 可见）。
- 备注：
  - chrome-devtools 浏览器直开 Vite 页面时，因缺少 Wails 宿主 runtime，无法完成真实交互链路冒烟；已通过编译与绑定生成验证可用性。

# Code Review 结论

- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1/重复计算/多余 I/O/锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过（前端为构建级校验，运行时交互仍建议在 Wails 宿主内做一次手工冒烟）

# 潜在影响与回滚方案

- 潜在影响：
  - 页面保存时若网络波动，可能出现“已持久化但运行时未刷新”或反向情况（V1 已用分步结果暴露阶段失败）。
  - 新页面默认依赖登录态与 hubId；未登录状态下仅展示只读空态。
- 回滚方案：
  - 回退以下文件即可整体撤销本次功能：
    - `app.go`
    - `internal/services/auth/permissions.go`
    - `internal/services/permission/service.go`
    - `internal/services/permission/service_test.go`
    - `frontend/src/stores/permissions.ts`
    - `frontend/src/pages/Permissions.vue`
    - `frontend/src/router/index.ts`
    - `frontend/src/layout/AppShell.vue`
  - 回退后重跑：`GOWORK=off go test ./... -count=1`、`npm run build`。
