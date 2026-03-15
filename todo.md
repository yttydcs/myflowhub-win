# Todo - Win 下游依赖对齐 Core v0.4.5 / SDK v0.1.7

## 目标与状态
- 目标：将 Win 依赖对齐到 `myflowhub-core v0.4.5` 与 `myflowhub-sdk v0.1.7`，确保蓝牙注册/登录链路包含最新修复。
- 当前状态：Win `main` 位于 `v0.0.6`，依赖为 `core v0.4.4`、`sdk v0.1.6`。

## 任务清单
- [ ] WIN-1 更新 `go.mod` 依赖到 `core v0.4.5`、`sdk v0.1.7`
- [ ] WIN-2 运行测试验证（`go test ./... -count=1`）
- [ ] WIN-3 更新变更归档 `docs/change/2026-03-15_bump-core-v0.4.5-win.md`
- [ ] WIN-4 提交、合并、打 tag（`v0.0.7`）

## 验收条件
- 依赖版本准确，行为无计划外改动。
- 测试通过。
- 归档文档完整。

## 回滚点
- 回退 `go.mod` 与归档文档。
