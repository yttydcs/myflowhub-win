# Plan - MyFlowHub-Win：VarPool 联动适配 VarStore 回应语义

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`refactor/varstore-hop-align`
- Worktree：`d:\project\MyFlowHub3\worktrees\varstore-hop-align\win`
- Base：`main`
- 关联仓库：`MyFlowHub-SubProto`、`MyFlowHub-SDK`、`MyFlowHub-Server`

## 项目目标与当前状态
- 目标：确保 Win 侧 VarPool 在 VarStore 新回程策略（逐跳 Cmd 响应）下行为一致、UI 不回归。
- 当前状态：SDK await 已完成 VarStore `MajorCmd` 响应兼容；Win 侧待在升级 SDK/SubProto 版本后做冒烟验证，并按需做最小适配（WIN-1/WIN-2）。

## 依赖关系
- 依赖 SDK await 对 VarStore `MajorCmd` 响应的兼容。
- 依赖 SubProto 新规则（`set_resp` value、list 空集合语义、subscriber 规则）。

## 风险与注意事项
- 避免 UI 侧重复处理 owner 通知与请求响应导致状态闪动。
- 避免对无关子协议 await 行为产生副作用。

## 可执行任务清单（Checklist）

### WIN-1 VarPool 服务层兼容检查与最小改造
- 目标：确认并修复 VarPool 服务对响应 major/action 的假设，保证请求-响应闭环稳定。
- 涉及模块/文件：`internal/services/varpool/service.go`、必要的 await 调用封装文件。
- 验收条件：get/set/list/revoke/subscribe 在新回程语义下可正常返回。
- 测试点：VarPool 服务层单测或最小集成测试。
- 回滚点：恢复服务层匹配策略改动。

### WIN-2 前端状态语义校准
- 目标：前端 store/UI 与新规则一致：list 空集合成功展示、set 失败不误刷新缓存、notify 不重复提示。
- 涉及模块/文件：`frontend/src/stores/varpool.ts`、必要页面文件。
- 验收条件：UI 行为与服务返回码一致，不出现空列表误报错误。
- 测试点：手动冒烟 + （如存在）前端测试用例。
- 回滚点：回退 store 侧处理分支。

### WIN-3 归档变更
- 目标：沉淀 Win 联动改造内容与验证结果。
- 涉及模块/文件：`docs/change/2026-03-06_varstore-hop-align-win.md`
- 验收条件：文档覆盖 WIN-1~WIN-2，含风险与回滚步骤。
- 测试点：文档命令/操作可复现。
- 回滚点：文档改动可独立回退。
