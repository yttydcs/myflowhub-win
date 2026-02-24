# Plan - Devices：节点信息弹窗（management/node_info）

## Workflow 信息
- 范围：跨仓库（Proto + SubProto + Server + Win）
- 分支（各仓一致）：`feat/node-info`
- Worktrees：
  - Proto：`d:\project\MyFlowHub3\worktrees\node-info\MyFlowHub-Proto`
  - SubProto：`d:\project\MyFlowHub3\worktrees\node-info\MyFlowHub-SubProto`
  - Server：`d:\project\MyFlowHub3\worktrees\node-info\MyFlowHub-Server`
  - Win：`d:\project\MyFlowHub3\worktrees\node-info\MyFlowHub-Win`
- Base：`main`
- Workspace（本地联调用）：`d:\project\MyFlowHub3\worktrees\node-info\go.work`
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）

## 约束（边界）
- UI：使用页面内 Modal（不做多 OS 窗口）。
- 数据来源：
  - 远端节点：必须由 **目标节点本地采集并回包**（不允许 Hub/Win 侧推断/缓存伪造）。
  - Win 自身节点（sourceID==targetID）：由 Win 本地采集并返回（不走网络，避免 timeout）。
- 协议：仅新增 management action（wire **新增**，不改既有 action/schema/header 语义）。
- 展示：不强约束字段集合；以 `items[key]=value` 的 KV 形式返回并展示。
- 版本：优先展示 semver（若可得），否则 fallback 展示 commit/(devel) 等可定位信息（你已确认允许 fallback）。
- 发布：按顺序发布依赖与 tag（你已确认）：
  1) `myflowhub-proto` -> `v0.1.1`
  2) `myflowhub-subproto/management` -> `management/v0.1.1`
  3) `myflowhub-server` -> `v0.0.2`（触发 Release 出包）
  4) `myflowhub-win` 更新依赖并合并

## 当前状态（事实，可审计）
- `myflowhub-proto/protocol/management` 仅有 `node_echo/list_nodes/list_subtree/config_*`，缺少“节点基础信息”查询。
- Win `Devices` 目前只展示 nodeId/children，不支持点击查看详情。
- Win `ManagementService` 对 `ListNodes/ListSubtree` 已做 source==target 的短路；但无 `node_info`。
- `hub_server` release workflow 已存在（tag `v*` 触发打包），但节点信息能力尚未实现。

---

## 3.1) 计划拆分（Checklist）

### T1. Workspace 准备（已完成）
- 目标：创建跨仓 worktrees，并提供本地联调 go.work。
- 涉及：
  - `d:\project\MyFlowHub3\worktrees\node-info\*`
- 验收：
  - 4 个 worktree 均存在且分支为 `feat/node-info`
  - `worktrees/node-info/go.work` 可被 go 命令识别（在 worktree 下运行 `go env GOWORK` 可看到该文件路径）
- 回滚点：删除 `worktrees/node-info/` 与各 repo 的 worktree 绑定（`git worktree remove` + `prune`）。

### T2. Proto：新增 management `node_info`（并发布 `v0.1.1`）
- 目标：提供 wire source-of-truth（action 常量 + Req/Resp struct）。
- 涉及文件（预期）：
  - `MyFlowHub-Proto/protocol/management/types.go`
- 设计要点：
  - 新增：`ActionNodeInfo`、`ActionNodeInfoResp`
  - 新增：`NodeInfoReq{}`、`NodeInfoResp{Code, Msg, Items map[string]string}`
- 验收：
  - `go test ./...`（Proto 仓）通过（至少可编译）
  - tag `v0.1.1` 推送到 GitHub
- 测试点：
  - KV 字段可为空，但 `Code/Msg` 行为与现有 resp 一致（`Code==1` 代表 ok）
- 回滚点：
  - 未推 tag：`git revert` / reset
  - 已推 tag：不强制删除远端 tag（避免破坏下游）；改发补丁 tag（如 `v0.1.2`）并同步下游依赖。

### T3. SubProto(management)：实现 `node_info` handler（并发布 `management/v0.1.1`）
- 目标：目标节点本地采集并回包（平台/版本等）。
- 涉及文件（预期）：
  - `MyFlowHub-SubProto/management/actions.go`（注册）
  - `MyFlowHub-SubProto/management/management.go`（forward error 支持）
  - `MyFlowHub-SubProto/management/types.go`（常量/别名）
  - `MyFlowHub-SubProto/management/action_node_info.go`（新建：采集与响应）
- 采集建议（KV，不强制）：
  - `node_id`（来自 `srv.NodeID()`）
  - `platform`（`runtime.GOOS/GOARCH`）
  - `go_version`（`runtime.Version()`）
  - `module` / `version`（`debug.ReadBuildInfo()`）
  - `commit` / `vcs_time` / `vcs_modified`（BuildInfo settings）
- 验收：
  - handler 对本地请求返回 `Code==1` 且包含至少 `platform/node_id`（或可替代关键字段）
  - tag `management/v0.1.1` 推送到 GitHub
- 测试点：
  - action 未识别/转发失败时：能返回 `node_info_resp(code!=1,msg=...)`（避免纯 timeout）
- 回滚点同 T2（tag 已推则走补丁 tag）。

### T4. Server：接入新能力 + 集成测试 + 发布 `v0.0.2`
- 目标：Server 依赖升级并可对外发布可用二进制。
- 涉及文件（预期）：
  - `MyFlowHub-Server/go.mod`（升级 `myflowhub-proto v0.1.1`、`myflowhub-subproto/management v0.1.1`）
  - `MyFlowHub-Server/tests/integration_management_node_info_test.go`（新建或扩展现有集成测试）
- 验收：
  - `go test ./...`（Server 仓）通过
  - 打 tag `v0.0.2` 后 GitHub Release 成功产出 zip（windows/linux amd64）
- 回滚点：
  - tag 未推：revert + 删除本地 tag
  - tag 已推：发 `v0.0.3` 修复（不建议回收远端 release）。

### T5. Win：Devices 点击节点弹出 Modal + 查询 node_info（自节点短路）
- 目标：在 Devices 树里点击节点显示基本信息（KV）。
- 涉及文件（预期）：
  - `MyFlowHub-Win/internal/services/management/service.go`（新增 `NodeInfo/NodeInfoSimple` + source==target 本地采集）
  - `MyFlowHub-Win/frontend/src/pages/Devices.vue`（节点行点击 -> Modal + 展示 KV）
  - `MyFlowHub-Win/frontend/src/stores/devices.ts`（可选：封装 node_info 调用；若不封装则在页面内调用 Wails binding）
  - `MyFlowHub-Win/go.mod`（升级 `myflowhub-proto v0.1.1`）
- 验收（冒烟）：
  - 打开 `Devices`，点击任意节点可弹窗；加载中/成功/失败态清晰
  - 点击 `+/-` 仅展开收起，不会误触打开弹窗（事件冒泡控制）
  - 点击 Win 自身节点不再出现 timeout（应返回本地信息或空但 `Code==1`）
- 回滚点：单 PR 可 revert；UI 变更点集中在 Devices 页面。

### T6. 合并与发布顺序（跨仓）
- 目标：保证下游依赖可用（避免 Win/Server 指向不存在的 tag）。
- 顺序（强制）：
  1) 合并 Proto -> push branch -> 打 tag `v0.1.1` -> push tag
  2) 合并 SubProto(management) -> push -> 打 tag `management/v0.1.1` -> push tag
  3) 合并 Server -> push -> 打 tag `v0.0.2` -> push tag（触发 Release）
  4) 合并 Win -> push
- 验收：各仓 `origin/main` 与 tag 均存在且 CI/Release 成功（Server 以 GitHub Release 为准）。
- 回滚点：若发布后发现问题，以补丁版本推进（避免删除已被依赖的 tag/release）。

---

## 风险与注意事项
- 旧节点未实现 `node_info`：可能仍表现为 timeout；Win 需用 Toast 清晰提示，不应卡死 UI。
- `debug.ReadBuildInfo()` 的 `Main.Version` 在某些构建方式下可能为 `(devel)`：因此必须同时返回 `commit/vcs_time` 等 fallback 字段。

