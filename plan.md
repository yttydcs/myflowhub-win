# Plan - Win：Flow 项目中心 + 独立编辑窗口 + 部署管理

## Workflow 信息
- Repo：`MyFlowHub-Win`
- Branch：`feat/win-project-center`
- Worktree：`D:/project/MyFlowHub3/repo/MyFlowHub-Win/repo/MyFlowHub-Win/worktrees/feat-win-project-center`
- Base：`main`

## 项目目标与当前状态
- 目标：
  - Flow 页面首屏改为本地项目列表（按 profile 持久化）。
  - 每个项目支持 `Edit` 与 `Deploy`。
  - `Edit` 在独立窗口打开，支持并行编辑多个项目。
  - 顶部 `Current Deployments` 可按 nodeId 查询并支持删除部署。
- 当前状态：
  - `Flow.vue` 为单页混合“列表+编辑器+状态”，列表来自远端 executor。
  - 无本地项目存储 API。
  - FlowService 无 delete 接口。

## 可执行任务清单（Checklist）
- [x] WIN-PRJ-1 新增本地项目持久化 API（App 层，profile 隔离）
- [x] WIN-PRJ-2 FlowService 增加 Delete/DeleteSimple
- [x] WIN-PRJ-3 新建 Flow 项目中心 store（本地项目 + 部署查询/删除）
- [x] WIN-PRJ-4 页面与路由重构（项目中心页 + 编辑窗口页）
- [x] WIN-PRJ-5 部署弹窗（含触发器回写、覆盖二次确认、仅 set）
- [x] WIN-PRJ-6 Current Deployments（nodeId + 设备树选择 + 删除）
- [x] WIN-PRJ-7 验证与回归

## 任务明细

### WIN-PRJ-1 本地项目持久化 API
- 目标：按 profile 持久化 Flow 项目清单与项目内容。
- 涉及模块/文件：
  - `app.go`（bindings 暴露）
  - 新增 `app_flow_projects.go`（建议）
  - `frontend/wailsjs/go/main/App.d.ts` / `App.js`（生成或同步）
- 验收条件：
  - 提供读取/保存本地 Flow 项目状态接口。
  - 数据按 `store.CurrentProfile()` 隔离。
- 测试点：
  - 切换 profile 后项目隔离。
- 回滚点：
  - 回退新增 API 与绑定。

### WIN-PRJ-2 FlowService delete 接口
- 目标：Win 侧可调用 `flow.delete`。
- 涉及模块/文件：
  - `internal/services/flow/service.go`
  - `frontend/src/stores/flow.ts`（或新 store）
- 验收条件：
  - 提供 `Delete/DeleteSimple`，输入校验与错误处理对齐现有风格。
- 测试点：
  - delete 请求可返回成功/失败信息。
- 回滚点：
  - 回退 delete 接口。

### WIN-PRJ-3 新建项目中心 store
- 目标：分离“本地项目模型”与“远端部署操作”。
- 涉及模块/文件：
  - 新增 `frontend/src/stores/flowProjects.ts`（建议）
- 验收条件：
  - 本地项目 CRUD、trigger 回写策略（方案 B）
  - 远端 list/get/set/delete 的调用编排
- 测试点：
  - 项目保存、读取、部署、删除链路可用。
- 回滚点：
  - 回退新 store，恢复旧 flow.ts 直连模式。

### WIN-PRJ-4 页面与路由重构
- 目标：
  - `/flow` 变为项目中心页面。
  - 新增 `/flow-editor-window`（window layout）承载纯编辑器。
- 涉及模块/文件：
  - `frontend/src/router/index.ts`
  - `frontend/src/pages/Flow.vue`（重构为项目中心）
  - 新增 `frontend/src/windows/FlowEditorWindow.vue`（建议）
  - 复用/拆分编辑器组件（从现有 Flow.vue 抽出）
- 验收条件：
  - 项目中心不承载复杂编辑面板。
  - 编辑窗口可并行打开多个项目。
- 测试点：
  - `window.open` 多窗口并发编辑。
- 回滚点：
  - 回退路由和页面拆分。

### WIN-PRJ-5 部署弹窗
- 目标：部署动作只 `set`，部署时可编辑 trigger 并回写项目默认 trigger（方案 B）。
- 涉及模块/文件：
  - 项目中心页面与相关 store
- 验收条件：
  - 支持手输 nodeId 与从设备树选择。
  - 目标 node 存在同 `flow_id` 时二次确认覆盖。
  - 部署不触发 run。
- 测试点：
  - 覆盖确认流程、触发器回写正确。
- 回滚点：
  - 回退部署弹窗与覆盖检查。

### WIN-PRJ-6 Current Deployments
- 目标：顶部区域按 nodeId 查看部署，并支持删除部署。
- 涉及模块/文件：
  - 项目中心页面与相关 store
  - 设备树选择复用 `Devices`/`ManagementService` 能力
- 验收条件：
  - 展示至少：`flow_id/name/trigger/last_status/last_run_id`。
  - 删除部署带二次确认。
- 测试点：
  - list + trigger 展示、delete 成功后刷新。
- 回滚点：
  - 回退 current deployments 区域。

### WIN-PRJ-7 验证与回归
- 目标：保证关键路径可执行。
- 涉及模块/文件：
  - `frontend/` + `internal/services/flow/` + `app_*`
- 验收条件：
  - `go test ./... -count=1`
  - `frontend` 构建通过（环境允许时）。
- 测试点：
  - 新建项目 -> 编辑窗口修改 -> 部署 -> 当前部署查看 -> 删除部署。
- 回滚点：
  - 回退本 workflow 提交。

## 依赖关系
- 依赖 Proto/SubProto delete 能力落地。
- 与 Server docs workflow 并行；发布前需对齐文档。

## 风险与注意事项
- 风险：若 trigger 展示强依赖逐条 get，可能引入 N+1 请求；需做按需/并发限流。
- 风险：多窗口并发编辑同项目可能产生覆盖冲突；需 last-write-wins + 二次提示。
- 注意：严格保持“部署仅 set，不 run”。

