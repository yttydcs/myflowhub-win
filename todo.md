# Todo - Win 下游依赖对齐 Core/SDK 新版本

## 目标与状态
- 目标：将 Win 依赖对齐到 `myflowhub-core v0.4.4` 与 `myflowhub-sdk v0.1.6`，确保发行包包含最新 RFCOMM 修复链路。
- 当前状态：Win `main` 位于 `v0.0.5`，依赖为 `core v0.4.3`、`sdk v0.1.5`。

## 任务清单
- [ ] WIN-1 更新 `go.mod` 依赖到 `core v0.4.4`、`sdk v0.1.6`
- [ ] WIN-2 运行测试验证（`go test ./... -count=1`）
- [ ] WIN-3 更新变更归档 `docs/change/2026-03-15_bump-core-v0.4.4-win.md`
- [ ] WIN-4 提交、合并、打 tag（`v0.0.6`）

## 验收条件
- 依赖版本准确，行为无计划外改动。
- 测试通过。
- 归档文档完整。

## 回滚点
- 回退 `go.mod` 与归档文档。
