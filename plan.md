# Plan - Win：Flow 编辑器适配 call-only 节点模型

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`refactor/win-call-node`
- Worktree：`d:\project\MyFlowHub3\repo\MyFlowHub-Win\worktrees\refactor-win-call-node`
- Base：`main`

## 1) 需求分析

### 目标
- 让 Win Flow 编辑器与 SubProto 新规范对齐：新写入仅生成 `kind=call` 节点。
- 去除 UI 中 `local/exec` 二分编辑心智，统一为单一调用节点。

### 范围
- 必须：
  - 前端节点模型改为 `call`。
  - 保存 DAG 时仅写 `kind=call`。
  - 旧数据读取兼容：读到 `local/exec` 时映射为 `call` 节点显示/保存。
  - 能力选择器保留可用：可按当前 target 过滤并回填 `method/target`。
  - 更新 Win 变更文档。
- 可选：
  - 在节点详情中提示 `target=0/空` 表示本地调用。
- 不做：
  - 不改 Go 服务层协议封装。
  - 不改 SubProto 后端逻辑。

### 使用场景
- 用户在 Win 中新建 Flow，添加节点，选择能力，保存后可被 SubProto `flow.set` 接受。
- 用户加载历史含 `local/exec` 的 Flow，编辑后可保存为新 `call` 格式。

### 功能需求
- 新增节点默认 `kind=call`。
- 节点详情移除 kind 切换；以 `target` 决定本地/远程行为。
- 能力选择器统一：
  - `target>0` 时按 target 过滤；
  - `target<=0` 时按 executor 作为本地能力集合。

### 非功能需求
- 可读性：减少分支与重复状态。
- 可扩展：后续新增调用策略集中在 call 节点。
- 稳定性：保留手工 method/target override，不阻断高级用法。

### 输入输出
- 输入：Flow 页面编辑状态（nodes/edges/trigger）。
- 输出：`flow.set` 请求中的 graph 节点均为 `kind=call`。

### 边界异常
- target 非法值、args 非法 JSON、method 为空仍按现有校验报错。
- 能力查询失败保持 toast 提示，不清空用户手填输入。

### 验收标准
- Win 端保存时不再产生 `local/exec` kind。
- 选择能力后 method/target 正确写回。
- 旧 flow 读取后可再次保存并通过后端校验。

### 风险
- 若遗漏某处旧 kind 写入路径，仍会触发后端 400。
- 前端能力筛选逻辑变更可能影响既有 exec 选择体验。

阻塞：否

---

## 2) 架构设计（分析）

### 总体方案
- 将 `frontend/src/stores/flow.ts` 中节点类型改为单一 `call`，并在序列化层固定输出 `kind=call`。
- 将 `frontend/src/pages/Flow.vue` 节点详情改为 call 统一视图，保留 target/method/args 编辑。

### 模块职责
- `flow.ts`：数据模型、读写映射、校验与保存。
- `Flow.vue`：交互层与能力选择 UI。

### 数据/调用流
1. `Get` 返回 graph：`mapNode` 将 `local/exec/call` 映射为内部 `call`。
2. 用户编辑节点：统一操作 `method/target/args`。
3. `Save` 构图：`buildGraph` 固定 `kind=call`。
4. 能力查询：沿用 `ExecCapQuerySimple`，按 target/executor 过滤候选。

### 接口草案
- 无协议字段变更。
- UI 语义：`target<=0` 代表本地调用。

### 错误与安全
- 复用现有输入校验。
- 不放宽任何 auth/permission 检查。

### 性能与测试策略
- 不新增网络请求次数。
- 优先执行前端构建与类型检查（`npm run build`）。

### 可扩展性设计点
- 未来可在 call 节点增加 version/policy 字段，不再引入新 kind。

阻塞：否

---

## 3.1) 计划拆分（Checklist）

### WIN-CALL-1 - Store 模型切换为 call-only
- 目标：统一节点类型与读写映射。
- 涉及文件：
  - `frontend/src/stores/flow.ts`
- 验收条件：
  - `buildGraph` 仅输出 `kind=call`。
  - `mapNode` 兼容旧 `local/exec`。
- 测试点：
  - 加载旧数据并保存，payload 为 call。
- 回滚点：revert 本任务提交。

### WIN-CALL-2 - Flow 页面节点编辑统一化
- 目标：去除 local/exec 分叉 UI，改为 call 统一配置。
- 涉及文件：
  - `frontend/src/pages/Flow.vue`
- 验收条件：
  - 新增节点无 kind 选择。
  - 节点详情可设置 target/method/args，能力选择可用。
- 测试点：
  - 选择能力、切换 target、手工覆盖 method。
- 回滚点：revert 本任务提交。

### WIN-CALL-3 - 验证与归档
- 目标：完成构建验证与变更文档。
- 涉及文件：
  - `docs/change/2026-03-20_win-call-node-unify.md`
- 验收条件：
  - 构建结果记录清楚。
  - 文档包含任务映射、权衡、回滚。
- 回滚点：revert 本任务提交。
