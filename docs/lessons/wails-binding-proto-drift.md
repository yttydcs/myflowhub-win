# wails-binding-proto-drift

## Summary
- 当 Win Wails service 直接引用 shared proto 中尚未存在的新 req/resp 类型，或 merge 后同时保留两套本地 detail 定义时，`GOWORK=off` 下的 `go test` 和 `wails generate module` 会直接编译失败。此类问题应通过单一 canonical 本地 typed payload 保持 JSON 契约不变并隔离 proto 漂移。

## Lookup Hints
- `undefined: flow.DetailReq`
- `undefined: flow.DetailResp`
- `undefined: flow.ActionDetail`
- `actionDetail redeclared in this block`
- `DetailResp redeclared in this block`
- `wails generate module`
- `GOWORK=off`
- `myflowhub-proto`

## Symptoms
- `wails generate module` 在 Go 编译阶段报未定义符号。
- `go test ./internal/services/<module>` 报 shared proto 中某个 req/resp 类型不存在。
- `wails generate module` 或 `go test ./...` 报 `redeclared in this block`，提示 `service.go` 和 `detail_types.go` 同时定义了 `actionDetail` / `DetailReq` / `DetailResp`。
- 开发者先看到 worktree 不在 `go.work` 的错误，切到 `GOWORK=off` 后才暴露真实缺失符号。

## Impact
- Wails bindings 无法生成。
- 前端依赖的 `frontend/wailsjs` 无法刷新。
- 相关 Go 包无法通过基础编译验证。

## Trigger Conditions
- 新增或修改 Win service public method 时，直接使用了 shared proto 中尚未发布的类型或常量。
- 把本地 typed payload 从 `service.go` 拆到独立文件后，后续 merge/cherry-pick 又把旧的内联定义带回了 `service.go`。
- worktree 校验未显式使用 `GOWORK=off`，导致父级 `go.work` 先拦截真实错误。

## Root Cause
- Win 仓库的实现节奏先于 shared proto 基线；service 层把“未来 proto 类型”直接暴露到当前可编译接口，导致 bindings 生成和 Go 编译都依赖一个不存在的符号集。
- 后续集成没有保持单一 canonical 本地 detail 类型来源，merge 结果把独立 `detail_types.go` 和旧的 `service.go` 内联定义同时保留下来，导致重复声明。

## Investigation Trail
- 先在 worktree 中执行 `wails generate module`，看到父级 `go.work` 模块外错误。
- 切换到 `$env:GOWORK='off'` 后复现真实编译失败。
- 对 `repo/MyFlowHub-Proto/protocol/flow/types.go` 与当前 server 代码树做搜索，确认不存在 `DetailReq` / `DetailResp` / `ActionDetail*`。
- 对照已有 `internal/services/auth/authority.go`，确认本地 typed payload 是仓库内已采用的稳定修复模式。
- 如果报 `redeclared in this block`，继续对 `internal/services/flow/service.go` 与 `internal/services/flow/detail_types.go` 做对照，确认是否存在两套 detail action/type 定义。
- 用 `git blame` / `git log -S "type DetailReq struct"` 追踪是哪次 merge 或后续功能提交把旧定义带回了 `service.go`。

## Resolution
- 在 Win 侧新增本地 exported typed payload 和 action 常量。
- 保持 JSON 字段契约不变，只替换 service 方法的 Go 类型依赖。
- 补充最小单元测试，并用 `GOWORK=off` 执行 `go test ./...` 与 `wails generate module`。
- 如果已存在独立 `detail_types.go`，则删除 `service.go` 中重复的 `actionDetail*` / `DetailReq` / `DetailResp` 定义，保持单一 canonical 来源。

## Prevention / Guardrails
- 新增 Win Wails binding 前，先检查 shared proto 是否已定义对应 req/resp/action。
- worktree 下所有 Go / Wails 验证默认使用 `GOWORK=off`。
- 如果 shared proto 还没准备好，但前端/Win 需要先落地，优先使用 Win 本地 typed payload，并在 spec 中澄清这是实现边界而非协议扩展。
- 当本地 typed payload 已经被抽到独立文件后，后续 merge review 要显式检查旧文件里的内联定义是否已经删除，避免重新形成双源定义。

## Related Docs
- [2026-03-26_win-flow-detail-bindings.md](../change/2026-03-26_win-flow-detail-bindings.md)
- [2026-03-26_win-flow-detail-merge-regression.md](../change/2026-03-26_win-flow-detail-merge-regression.md)
- [2026-03-26_win-authority-console-refactor.md](../change/2026-03-26_win-authority-console-refactor.md)
- [flow-editor-run-detail.md](../specs/flow-editor-run-detail.md)
