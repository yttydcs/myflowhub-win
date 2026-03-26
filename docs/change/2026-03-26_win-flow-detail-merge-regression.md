# 2026-03-26_win-flow-detail-merge-regression

## 变更背景 / 目标
- `main` 上的 merge 结果把 `internal/services/flow/detail_types.go` 中的 canonical local detail 类型，与 `internal/services/flow/service.go` 中旧的内联 detail 定义同时保留下来。
- 当前直接后果是 `wails generate module`、`go test ./internal/services/flow` 和 `go test ./...` 都因 `actionDetail` / `DetailReq` / `DetailResp` 重复声明而失败。
- 本轮目标是在不改变稳定 `flow.detail` 契约的前提下，恢复唯一事实来源和构建链路。

## 具体变更内容
- 后端：
  - 删除 `internal/services/flow/service.go` 顶部被 merge 带回的 `actionDetail` / `actionDetailResp`
  - 删除 `internal/services/flow/service.go` 中重复的 `DetailReq` / `DetailResp` / `DetailNode` 定义
  - 保持 `internal/services/flow/detail_types.go` 作为唯一 canonical detail 类型来源
- 测试：
  - `internal/services/flow/service_test.go` 新增 `TestDetailRespJSONShape`
  - 锁定 `DetailResp` 当前 canonical JSON 形状，包括 `executor_node` 和 `node` 指针反序列化
- 文档：
  - 更新 `docs/lessons/wails-binding-proto-drift.md`
  - 补充 merge 回归场景下的 `redeclared in this block` 排查线索

## Requirements impact
- none

## Specs impact
- none

## Lessons impact
- updated

## Related requirements
- [flow-editor-run-detail.md](../requirements/flow-editor-run-detail.md)

## Related specs
- [flow-editor-run-detail.md](../specs/flow-editor-run-detail.md)

## Related lessons
- [wails-binding-proto-drift.md](../lessons/wails-binding-proto-drift.md)

## 对应 plan.md 任务映射
- `FIX-1`
  - `internal/services/flow/service.go`
- `TEST-1`
  - `internal/services/flow/service_test.go`
- `ARCHIVE-1`
  - `docs/change/2026-03-26_win-flow-detail-merge-regression.md`
  - `docs/lessons/wails-binding-proto-drift.md`
  - `docs/lessons/README.md`
  - `docs/change/README.md`

## 经验 / 教训摘要
- 对“本地 typed payload 兜底”这类修复，真正需要守住的是单一 canonical 来源，而不只是“代码里存在一个可用定义”。
- merge 回归会把“缺失 proto 依赖”问题变成“重复本地定义”问题，两者都会卡死 Wails bindings 和 Go 编译，但排查入口不同。
- 对这类 Wails/Go 编译链问题，`GOWORK=off` 仍然是 worktree 内的标准验证姿势。

## 可复用排查线索
- 症状：
  - `wails generate module` 报 `actionDetail redeclared in this block`
  - `go test ./internal/services/flow` 报 `DetailResp redeclared in this block`
  - `go test ./...` 同时拖挂根包、`cmd/myflowhub-mcp`、`internal/mcp`、`internal/mcpapp`
- 触发条件：
  - 本地 detail payload 已经拆到独立文件
  - 后续 merge 或 cherry-pick 又把旧的内联定义带回 `service.go`
- 关键词：
  - `actionDetail redeclared`
  - `DetailReq redeclared`
  - `DetailResp redeclared`
  - `detail_types.go`
  - `wails generate module`
  - `GOWORK=off`
- 快速检查：
  - 先执行 `$env:GOWORK='off'; go test ./internal/services/flow`
  - 检查 `internal/services/flow/service.go` 是否仍声明 `actionDetail*` 和 `Detail*`
  - 检查 `internal/services/flow/detail_types.go` 是否已经存在同名 canonical 定义

## 关键设计决策与权衡
- 决策：保留 `detail_types.go`，删除 `service.go` 中的重复 detail 定义
  - 原因：这和 `2026-03-26_win-flow-detail-bindings.md` 已归档的修复方向、当前 stable spec，以及现有 `DetailResp` 形状保持一致。
  - 代价：仍需未来在 shared proto 补齐后再评估是否收敛。
- 决策：补一个 JSON 形状测试，而不是只依赖编译通过
  - 原因：编译能防止重名，但不能显式锁住 `DetailResp` 当前 canonical 字段形状。

## 测试与验证方式 / 结果
- `$env:GOWORK='off'; go test ./internal/services/flow`
  - 结果：通过
- `$env:GOWORK='off'; go test ./...`
  - 结果：通过
- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：仍打印重复 `Not found: time.Time` 警告，但退出码为 `0`
- `git diff --check`
  - 结果：通过

## 潜在影响
- 本轮不改变用户可见 `flow.detail` 行为，只恢复当前主线的构建与绑定生成能力。
- 后续如果 shared proto 正式新增 `flow.detail`，仍需再评估本地定义的收敛路径。

## 回滚方案
- 回退 `internal/services/flow/service.go` 的去重改动
- 回退 `internal/services/flow/service_test.go` 中新增的 JSON 形状测试
- 回退本次 `docs/change` 与 `docs/lessons` 更新

## 子Agent执行轨迹
- none
