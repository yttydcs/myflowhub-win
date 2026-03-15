# 变更背景 / 目标

对齐 SDK 重连修复版本，解决 Windows 蓝牙“断开后重连登录报 connection aborted”。

# 具体变更内容

- `go.mod` / `go.sum`
  - `github.com/yttydcs/myflowhub-sdk` 从 `v0.1.8` 升级到 `v0.1.9`。

# 对应任务映射

- Task: Win 下游依赖对齐（SDK v0.1.9）
  - 目标：获取 SDK 断开重连修复能力。
  - 验收：`go test ./...` 通过。

# 关键设计决策与权衡

- 本次仅依赖升级，不引入额外业务逻辑改动，降低回归风险。

# 测试与验证

- `go test ./...`

# 潜在影响与回滚

- 影响：会话断开/重连流程行为以 SDK v0.1.9 为准。
- 回滚：将 SDK 依赖回退到 `v0.1.8` 并复测。
