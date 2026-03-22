# 2026-03-22 Devices Node Display Name Follow-up

## 变更背景 / 目标

在已有 `Devices` 节点昵称展示能力之上，继续补齐两个用户仍会直接撞上的缺口：

- `Edit` 配置弹窗在首次命名时看不到 `node.display_name`
- Win 自身节点的本地 `node_info` 和 auth 请求还没有复用同一份显示名

本次目标是在不新增页面、不改 `config_set` 标准行为的前提下，把这两条链路补齐。

## 具体变更内容

- `frontend/src/stores/management.ts`
  - 在 `config_list` 返回 keys 的基础上 synthetic 合成并去重 `node.display_name`
  - 保持现有 `config_get` / `config_set` 回显路径不变
- `frontend/src/pages/Devices.vue`
  - Config 标题优先使用当前已加载的 `node.display_name`
  - 仍回退设备树缓存与 `Node {nodeId}`
- `internal/services/management/service.go`
  - self `node_info` 短路返回时补 `items["display_name"]`
- `internal/services/auth/service.go`
  - register/login JSON 可选追加 `display_name`
  - 显示名读取遵循 `raw node.display_name -> 当前 profile scoped 值` 的回退顺序
- `app.go`
  - 为 `AuthService` 注入本地 `Store`
- 新增测试
  - `internal/services/auth/service_test.go`
  - `internal/services/management/service_test.go`

## 对应计划任务映射

- `WIN1`
- `WIN2`
- `WIN3`

## 关键设计决策与权衡

- 继续复用现有 `Edit` 配置弹窗，而不是新增独立“昵称编辑”入口。
- auth 扩展只加兼容 JSON 字段，不改 Wails 暴露方法签名，也不要求旧对端理解该字段。
- 本地显示名读取优先 raw key，兼容 self `config_set` 直接落盘 `settings.json` 的语义。

## Requirements / Specs 影响检查

- Requirements impact：`none`
- Specs impact：`none`
- Related requirements：
  - [management-node-display-name.md](/D:/project/MyFlowHub3/worktrees/MyFlowHub3-feat-node-display-name-followup/docs/requirements/management-node-display-name.md)
- Related specs：
  - [management-config-layering.md](/D:/project/MyFlowHub3/worktrees/MyFlowHub3-feat-node-display-name-followup/docs/specs/management-config-layering.md)
- Lessons：`none`

## 测试与验证方式 / 结果

- `GOWORK=off go test ./... -count=1`：通过
- `npm run build`：失败
  - 阻塞点：`frontend/src/pages/Home.vue` 依赖的 `../../wailsjs/go/session/SessionService` 缺失
  - 结论：属于仓内既有生成物问题，与本次 `Devices` / management / auth 改动无关

## 潜在影响与回滚方案

### 潜在影响

- Win 前端整仓 build 仍不能作为完整验收口径，直到 `wailsjs` 生成物恢复。
- `node.display_name` 本地读取逻辑目前在 auth / management 各有一份，后续若被更多服务消费可再抽共享 helper。

### 回滚方案

- 回退 `frontend/src/stores/management.ts`
- 回退 `frontend/src/pages/Devices.vue`
- 回退 `internal/services/management/service.go`
- 回退 `internal/services/auth/service.go`
- 回退 `app.go`
- 回退新增测试文件

## 子 Agent 执行轨迹

- `WIN1` / `WIN2` -> `Erdos (019d1618-0ff3-7312-96d1-bda0ab0c8c39)` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-node-display-name-followup`
  - 文件：`frontend/src/stores/management.ts`、`frontend/src/pages/Devices.vue`、`internal/services/auth/service.go`、`internal/services/management/service.go`、`app.go`、测试文件
  - 验收：Win Go 测试通过；前端 build 失败点已明确为既有环境问题
