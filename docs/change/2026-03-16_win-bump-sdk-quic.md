# 变更背景 / 目标

为 Win 端接入 `quic://` 传输链路，本次仅做依赖对齐，不改 UI/业务逻辑代码。  
目标是让 Win 通过最新 SDK/Core 直接具备 QUIC 连接能力，并保持现有 TCP/RFCOMM 行为不回退。

# 具体变更内容

## 新增
- 无新增业务代码文件。

## 修改
- `go.mod`
  - `github.com/yttydcs/myflowhub-core` 升级到 `v0.4.7-0.20260316021423-d992975ec6ad`
  - `github.com/yttydcs/myflowhub-sdk` 升级到 `v0.1.10-0.20260316021802-bd10cd7ada93`
  - 同步间接依赖版本（`x/crypto`、`x/net`、`x/text`）
- `go.sum`
  - 补齐 QUIC 相关传递依赖校验项（含 `quic-go` 等）

## 删除
- 无。

# 对应计划任务映射

- `QUIC-WIN-1`：Win 接入与联调（先完成依赖层接入）
- `QUIC-REL-1`：下游版本对齐与发布准备

# 关键设计决策与权衡

- 选择“仅升级依赖，不改 Win 业务层代码”：
  - 优点：变更最小、回滚简单、风险集中在依赖解析层；
  - 代价：QUIC 行为完全依赖 SDK/Core 实现，Win 侧仅做透传验证。
- 保持 `GOWORK=off` 进行模块测试：
  - 避免根 `go.work` 干扰 worktree 独立依赖解析，保证发布可复现。

# 测试与验证方式 / 结果

- 执行：
  - `GOWORK=off go mod tidy`
  - `GOWORK=off go test ./...`
- 结果：
  - 全量测试通过（含主包与已有服务测试），未出现新增失败。

# 潜在影响与回滚方案

- 潜在影响：
  - 依赖升级可能引入传递包行为差异（TLS/网络栈相关）。
- 回滚方案：
  - 回退本次提交，将 `go.mod/go.sum` 恢复到升级前版本；
  - 重新执行 `GOWORK=off go test ./...` 验证回滚稳定性。
