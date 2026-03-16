# 变更背景 / 目标

Win 端在 QUIC 联调完成后，需要将依赖从开发态 pseudo-version 收敛为正式发布版本，确保发布包可复现、可追踪。

# 具体变更内容

## 修改
- `go.mod`
  - `github.com/yttydcs/myflowhub-core`
    - `v0.4.7-0.20260316021423-d992975ec6ad`
    - -> `v0.4.7`
  - `github.com/yttydcs/myflowhub-sdk`
    - `v0.1.10-0.20260316021802-bd10cd7ada93`
    - -> `v0.1.10`
- `go.sum`
  - 同步校验项，替换 pseudo-version 相关记录。

# 对应任务映射

- `QUIC-WIN-1`：Win 接入与联调（依赖收敛）
- `QUIC-REL-1`：下游版本对齐与发布

# 关键设计决策与权衡

- 仅做依赖版本收敛，不改 Win 业务代码：
  - 优点：风险最小、回滚简单；
  - 代价：Win 侧行为完全跟随 SDK/Core 实现。

# 测试与验证方式 / 结果

- 执行：
  - `GOWORK=off go mod tidy`
  - `GOWORK=off go test ./...`
- 结果：
  - 全量测试通过。

# 潜在影响与回滚方案

- 潜在影响：
  - 无功能新增，主要影响依赖解析来源。
- 回滚方案：
  - 回退本次提交并恢复旧依赖版本，重新执行全量测试。
