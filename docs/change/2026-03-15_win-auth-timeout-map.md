# 变更背景 / 目标

Windows 侧在蓝牙链路异常时，错误提示存在可读性不足。目标是将常见网络超时/本机中止错误转为更可理解的 UI 文案。

# 具体变更内容

- 修改 `internal/services/auth/service.go`
  - 在 `toUIError` 中新增字符串规则：
    - 包含 `timed out` → `request timed out`
    - 包含 `aborted by the software in your host machine` → `connection aborted`

# 对应任务映射

- Task: Win 认证错误可读性增强
  - 目标：让用户可区分“超时”与“连接中止”。
  - 文件：`internal/services/auth/service.go`
  - 验收：`go test ./...` 通过。

# 关键设计决策与权衡

- 仅做文案映射，不改协议与状态机，降低引入风险。

# 测试与验证

- `go test ./...`

# 潜在影响与回滚

- 影响：仅错误消息映射变化，无协议兼容风险。
- 回滚：删除 `toUIError` 的新增分支即可。
