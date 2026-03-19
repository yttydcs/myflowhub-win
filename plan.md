# Plan - Win：Flow Local 能力选择器

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`feat/win-capability-local-picker`
- Worktree：`d:\project\MyFlowHub3\repo\MyFlowHub-Win\worktrees\feat-win-capability-local-picker`
- Base：`main`

## 项目目标与当前状态
- 目标：
  - 在 Flow 页的 `local` 节点编辑中支持能力下拉选择；
  - 本地能力选择后自动回填 `spec.method`；
  - 保留手工 method 覆盖，兼容历史 flow。
- 当前状态：
  - `local` 节点仅支持手填 method；
  - 能力选择器仅覆盖 `exec` 节点；
  - 用户无法直接可视化选择“当前执行节点（executor）上的能力”。

## 范围
- 必须：
  - `kind=local` 时新增能力查询与能力下拉；
  - 按当前 Flow executor（顶栏 Executor）过滤本地可选能力；
  - 选择能力后自动回填 `selectedNode.method`；
  - 保留手工 method 输入（manual override）；
  - 不影响 `exec` 节点现有能力选择逻辑。
- 可选：
  - local 能力为空时，给出明确提示与刷新引导。
- 不做：
  - 不改后端协议；
  - 不改 DAG 执行语义；
  - 不做自动生成 args schema 填充。

## 可执行任务清单（Checklist）

### WIN-LOCAL-CAP-1 - local 能力联动建模
- 目标：
  - 在页面层增加 local 能力列表计算与选中同步逻辑。
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
- 验收条件：
  - local 选中节点切换后，能力选中态与 method 保持同步；
  - executor 变化时，local 候选能力随之变化。
- 测试点：
  - 切换节点与切换 executor，候选能力列表正确更新。
- 回滚点：
  - 回退 local 相关 computed/watcher。

### WIN-LOCAL-CAP-2 - local 节点详情 UI 改造
- 目标：
  - 将 local 节点从“纯手填”改为“能力选择优先 + 手工覆盖兜底”。
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
- 验收条件：
  - local 能力可刷新、可选择、可应用；
  - 选择后 method 自动写回并进入历史。
- 测试点：
  - 创建 local 节点，选择能力后 Save + Run 无回归。
- 回滚点：
  - 回退 local UI 区块。

### WIN-LOCAL-CAP-3 - 错误反馈与兼容
- 目标：
  - 收敛 local 能力查询/应用异常路径，保留未知 method 兼容。
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
- 验收条件：
  - 查询失败有 toast；
  - method 不在缓存能力中不强制清空。
- 测试点：
  - 403/timeout 场景提示；
  - 历史 flow 的 method 正常显示。
- 回滚点：
  - 回退新增提示逻辑。

### WIN-LOCAL-CAP-4 - 构建与关键路径验证
- 目标：
  - 验证 local 能力选择器不破坏现有流程。
- 涉及模块 / 文件：
  - `frontend/`
- 验收条件：
  - `npm run build` 完成至现有可达阶段；
  - local/exec 的保存与运行路径行为一致。
- 测试点：
  - local 选能力 + Save + Run；
  - exec 现有能力选择行为回归。
- 回滚点：
  - 清理构建产物并回退改动。

### WIN-LOCAL-CAP-5 - Code Review（强制）
- 目标：
  - 对需求、架构、性能、稳定性、测试覆盖逐项审查。
- 涉及模块 / 文件：
  - 本 workflow 全量改动文件
- 验收条件：
  - Review 清单给出通过/不通过结论。

### WIN-LOCAL-CAP-6 - 归档变更（强制）
- 目标：
  - 输出可交接、可审计的变更文档。
- 涉及模块 / 文件：
  - `docs/change/2026-03-19_win-local-capability-picker.md`
- 验收条件：
  - 包含目标、变更、任务映射、权衡、测试与回滚。

## 依赖关系
- `WIN-LOCAL-CAP-1` → `WIN-LOCAL-CAP-2` → `WIN-LOCAL-CAP-3` → `WIN-LOCAL-CAP-4` → `WIN-LOCAL-CAP-5` → `WIN-LOCAL-CAP-6`

## 风险与注意事项
- local 能力依赖远端缓存，可能出现能力缓存与 method 不一致；
- 需避免 exec/local 两套能力选中状态相互污染；
- executor 输入为空时需使用现有默认解析逻辑，避免过滤误判。

---

# Plan - Win：Authority 权限配置独立页面（V1）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`feat/win-authority-permissions-v1`
- Worktree：`D:\project\MyFlowHub3\repo\MyFlowHub-Win\worktrees\feat-win-authority-permissions-v1`
- Base：`main`
- 设计输入文档：
  - `docs/core.md`
  - `docs/权限.md`
  - `docs/2-auth.md`
  - `docs/3-varstore.md`
  - `docs/4-topicbus.md`
  - `docs/5-file.md`
  - `docs/6-flow.md`
  - `docs/7-exec.md`

## 项目目标与当前状态
- 目标：
  - 新增 Win 端独立“权限配置”页面，管理当前连接网络的 authority 侧权限策略。
  - 支持可写 `role/perms` 策略，且同时满足“持久化生效 + 运行时立即生效”。
  - 在不改 Server 协议的前提下完成 V1（使用现有 `management config_set` + `auth perms_snapshot` 能力）。
- 当前状态：
  - 无独立权限页面；仅 `LocalHub` 页面包含本地 sidecar `auth.*` 参数编辑。
  - Win `AuthService` 仅提供登录/注册/通用 Send，无 `get_perms/list_roles/perms_snapshot` 的强类型方法。
  - `management` 已支持按目标节点执行 `config get/set/list`，可用于 authority 持久化。

## 范围
- 必须：
  - 新增独立权限页面与导航入口。
  - 新增 authority 策略读取、编辑、校验、保存能力。
  - 保存时同时执行：配置持久化（`config_set`）+ 运行时应用（`perms_snapshot`）。
  - 保存后提供最小回读验证（至少 `list_roles` 或关键节点 `get_perms`）。
- 可选：
  - authority NodeID 自动解析（默认 `hubId`）+ 手动覆盖输入。
  - 增加 `perms_invalidate(refresh=true)` 作为运行时刷新增强。
- 不做：
  - 不新增 Server 端 `policy_set` 原生动作（该项留待 V2）。
  - 不改 `varstore/topicbus/file/flow/exec` 协议行为。
  - 不在本 workflow 内进行跨仓库发布与合并。

## 可执行任务清单（Checklist）

### [x] WIN-PERM-V1-1 - Go 服务层扩展（Auth/Permission 编排）
- 目标：
  - 提供权限页面所需的强类型后端接口，避免前端直接拼 action 字符串。
- 涉及模块 / 文件：
  - `internal/services/auth/service.go`
  - 新增 `internal/services/permission/service.go`
  - `app.go`（注册/绑定新服务）
- 验收条件：
  - 支持 `GetPerms`、`ListRoles`、`PushPermsSnapshot`（可选 `InvalidatePerms`）。
  - 支持一次性编排保存：`config_set(auth.*)` + `perms_snapshot`。
  - 输入校验完整（空 key、格式错误、node id 非法等）。
- 测试点：
  - `go test ./... -count=1`
  - 至少补充 1 组编排逻辑单元测试（解析与错误路径）。
- 回滚点：
  - 回退新增 service 与 `app.go` 绑定改动。

### [x] WIN-PERM-V1-2 - 前端权限页面与状态管理
- 目标：
  - 提供独立页面实现 authority 权限策略查看与编辑。
- 涉及模块 / 文件：
  - 新增 `frontend/src/pages/Permissions.vue`
  - 新增 `frontend/src/stores/permissions.ts`
  - `frontend/src/router/index.ts`
  - `frontend/src/layout/AppShell.vue`
- 验收条件：
  - 页面可加载 authority 策略（4 个核心配置：`auth.default_role`、`auth.default_perms`、`auth.node_roles`、`auth.role_perms`）。
  - 页面可保存并显示保存结果（成功/失败提示）。
  - 表单包含基本格式校验，避免提交明显非法策略。
- 测试点：
  - `npm run build`
  - 手工冒烟：加载、编辑、保存、回读。
- 回滚点：
  - 回退页面/路由/store 改动，导航恢复原样。

### [x] WIN-PERM-V1-3 - Wails bindings 与类型对齐
- 目标：
  - 确保新增 Go 接口可被前端稳定调用。
- 涉及模块 / 文件：
  - `frontend/wailsjs/**`（本地生成产物，若仓库忽略则不提交）
- 验收条件：
  - 前端调用新增后端接口无运行时 `binding unavailable` 错误。
- 测试点：
  - `GOWORK=off wails generate module`
- 回滚点：
  - 重新生成 bindings 或回退新增接口。

### [x] WIN-PERM-V1-4 - 关键路径验证
- 目标：
  - 验证“持久化 + 运行时”双生效链路可用。
- 涉及模块 / 文件：
  - 全仓（不新增业务文件）
- 验收条件：
  - 保存后，`ConfigGet(auth.*)` 与运行时查询结果（`list_roles/get_perms`）一致。
  - 错误路径可观测（超时/未连接/权限不足）。
- 测试点：
  - `GOWORK=off go test ./... -count=1`（通过）
  - `npm run build`（通过）
  - chrome-devtools 冒烟验证页面流程（受限：浏览器直开缺少 Wails runtime，页面无法完整挂载）。
- 回滚点：
  - 回退权限页相关提交。

### [x] WIN-PERM-V1-5 - Code Review（强制）
- 目标：
  - 逐项审查需求覆盖、架构、性能、稳定性、安全与测试完整性。
- 涉及模块 / 文件：
  - 本 workflow 全部改动文件。
- 验收条件：
  - 输出“通过/不通过”逐项结论。
  - 不通过项回流到 `WIN-PERM-V1-1/2/3/4` 修复。
- 测试点：
  - Review 清单完整可审计。
- 回滚点：
  - 回滚不通过改动并重审。

### [x] WIN-PERM-V1-6 - 归档文档（强制）
- 目标：
  - 归档本次权限页面 V1 的设计、实现、验证、影响与回滚方案。
- 涉及模块 / 文件：
  - `docs/change/2026-03-18_win-authority-permissions-v1.md`（文件名日期按实际提交日调整）
- 验收条件：
  - 文档包含任务映射、关键权衡、验证结果、风险与回滚。
- 测试点：
  - 文档可被其他同事独立接手执行。
- 回滚点：
  - 回退该归档文件并重新补齐。

## 依赖关系
- `WIN-PERM-V1-1` 完成后可并行推进 `WIN-PERM-V1-2` 与 `WIN-PERM-V1-3`。
- `WIN-PERM-V1-2` + `WIN-PERM-V1-3` 完成后进入 `WIN-PERM-V1-4`。
- `WIN-PERM-V1-4` 通过后进入 `WIN-PERM-V1-5`。
- `WIN-PERM-V1-5` 通过后进入 `WIN-PERM-V1-6`。

## 风险与注意事项
- V1 非原生事务：`config_set` 与 `perms_snapshot` 之间存在短窗口不一致风险。
- `management config_set` 当前为通用配置入口，缺少专用策略动作审计语义；需要在日志与变更归档中补足。
- `perms_snapshot`/`perms_invalidate` 依赖 authority 与子树链路健康，网络抖动可能导致局部延迟生效。
- 需严格限制计划外改动：仅触达权限页面与最小必要服务层。

## 待确认（进入 3.2 前）
- 保存目标 authority 解析策略：默认 `hubId` 并允许手动覆盖（建议）。
- 保存失败策略：默认“尽力执行并返回分步结果”（建议），而非强原子回滚。
