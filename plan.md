# Plan - Win：升级依赖并发布 RFCOMM 写出修复版本

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`chore/win-rfcomm-write-bump`
- Worktree：`d:\project\MyFlowHub3\worktrees\chore-win-rfcomm-write-bump\repo\MyFlowHub-Win`
- Base：`main`
- 依赖仓：
  - `MyFlowHub-Core`
  - `MyFlowHub-SDK`

## 项目目标与当前状态
- 目标：
  - 将 Win 桌面端升级到包含 RFCOMM 写出修复的新 Core / SDK 版本；
  - 做最小必要验证并准备新的 release；
  - 让 B 电脑通过 release 包获得可用的 RFCOMM 注册链路。
- 当前状态：
  - Win 端当前使用 `myflowhub-core v0.4.2`、`myflowhub-sdk v0.1.4`；
  - 真实测试中可以连接 RFCOMM，但 register 发送后常出现长时间无响应；
  - 根因已定位为底层帧发送短写，不是 UI 按钮或 payload 长度问题。

## 范围
- 必须：
  - 升级 Win 对 Core / SDK 的依赖版本；
  - 做最小必要构建/测试验证；
  - 归档本次依赖升级与 release 准备；
  - 推送代码并创建新的 Win release tag。
- 可选：
  - 若发现版本联动问题，补最小必要的兼容性调整。
- 不做：
  - 不新增 UI 功能；
  - 不改 Home 页面交互；
  - 不改 auth 业务逻辑。

## 可执行任务清单（Checklist）

### WIN-RFCOMM-1 - 升级 Core / SDK 依赖
- 目标：
  - 将 `go.mod` / `go.sum` 对齐到修复后的 Core 与 SDK 版本。
- 涉及模块 / 文件：
  - `go.mod`
  - `go.sum`
- 验收条件：
  - `go list -m` 显示依赖已升级；
  - 不引入计划外业务改动。
- 测试点：
  - `go mod tidy`
  - `go list -m github.com/yttydcs/myflowhub-core`
  - `go list -m github.com/yttydcs/myflowhub-sdk`
- 回滚点：
  - 回退依赖版本提交。

### WIN-RFCOMM-2 - 执行最小验证
- 目标：
  - 证明升级后的 Win 仓库可通过最小必要测试/构建校验。
- 涉及模块 / 文件：
  - 全仓测试
- 验收条件：
  - `GOWORK=off go test ./... -count=1` 通过；
  - 如需额外 release 准备文件，仅做最小化处理。
- 测试点：
  - `GOWORK=off go test ./... -count=1`
- 回滚点：
  - 回退依赖升级；删除临时验证产物。

### WIN-RFCOMM-3 - Code Review（强制）
- 目标：
  - 审查依赖升级范围、兼容性、稳定性与验证证据。
- 涉及模块 / 文件：
  - 本 workflow 全部改动文件
- 验收条件：
  - 形成逐项通过/不通过结论。
- 测试点：
  - Review 结论完整。
- 回滚点：
  - 修订依赖升级或取消发版。

### WIN-RFCOMM-4 - 归档与发布准备
- 目标：
  - 生成归档文档，并准备新的 release tag。
- 涉及模块 / 文件：
  - `docs/change/2026-03-15_win-rfcomm-write-bump.md`
- 验收条件：
  - 归档文档说明版本来源、测试、影响与回滚；
  - 明确本次 release 目的是携带 RFCOMM 写出修复。
- 测试点：
  - 归档内容完整可交接。
- 回滚点：
  - 回退归档文档。

## 依赖关系
- `WIN-RFCOMM-1` 完成后进入 `WIN-RFCOMM-2`
- `WIN-RFCOMM-2` 完成后进入 `WIN-RFCOMM-3`
- `WIN-RFCOMM-3` 通过后进入 `WIN-RFCOMM-4`

## 风险与注意事项
- Win release 必须基于已经推送的 Core / SDK tag，否则依赖无法在 `GOWORK=off` 下稳定解析；
- 本仓应保持“仅依赖升级 + 发布准备”的最小改动原则；
- 若 GitHub release workflow 依赖 tag 命名规则，需要按既有版本策略递增。
