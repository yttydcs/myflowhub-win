# 变更说明：Devices 点击节点弹窗展示 node_info

## 变更背景 / 目标
在 `Devices` 树形视图中，点击任意节点希望能查看其“基本信息”（平台、版本号等）。同时你要求该信息应来自**节点本身**：
- 远端节点：由目标节点本地采集并通过 management `node_info` 回包；
- Win 自身节点（sourceID==targetID）：Win 本地采集并返回，避免发包后超时。

## 具体变更内容（新增 / 修改 / 删除）
- 修改：
  - `internal/services/management/service.go`
    - 新增 `NodeInfo/NodeInfoSimple`
    - `sourceID==targetID` 时短路返回本机信息 KV（不走网络）
  - `frontend/src/pages/Devices.vue`
    - 节点行点击打开 Modal
    - 自动加载并以 KV 列表展示 `items`
    - 支持 Reload/Close，错误通过 Toast + 弹窗内错误态呈现

## 对应 plan.md 任务映射
- `plan.md`
  - T5. Win：Devices 点击节点弹出 Modal + 查询 node_info（自节点短路）

## 关键设计决策与权衡（性能 / 扩展性）
- UI 采用 Modal（而非多 OS 窗口）：
  - 改动最小、交互直观；后续若需要再抽成独立页面/窗口也可演进。
- 返回与展示均采用 KV：
  - 不强依赖固定字段，便于未来扩展（例如增加 uptime、能力列表等）。
- 性能：
  - 按需查询（点击才请求），避免对树上每个节点做预拉取。

## 测试与验证方式 / 结果
- 已在本地执行：`go test ./...`
- 冒烟建议：
  1) 启动 `hub_server`，Win 连接并登录；
  2) 打开 `Devices`，点击任意节点行；
  3) 确认弹窗展示 `platform/go_version/...` 等字段；
  4) 点击 Win 自身节点时不应再出现 timeout（应直接返回本机信息）。

## 潜在影响与回滚方案
- 影响：新增管理动作查询入口；若远端节点不支持 `node_info`，仍可能超时（UI 会提示失败）。
- 回滚：revert 本提交；UI 变更集中在 `Devices.vue`，易于回滚。

