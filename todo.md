# Plan - MyFlowHub-Win：Offer 远端目录可调 + 树状节点选择

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/file-console-offer-dir-node-picker`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-offer-dir-node-picker`
- Base：`main`
- 当前状态：已完成（阶段 1 → 4 已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 在 `Send Offer` 弹窗支持设置远端落地目录（`req.dir`）；
  - 目标节点从“手输 ID”升级为“树状选择”（风格参考 Devices）；
  - 后端支持“本地源目录”和“远端目标目录”分离，避免强绑定。
- 当前状态：
  - 现有 `StartOffer` 仅使用单一 `dir`：既用于本地找源文件，也用于远端落地目录；
  - Offer 弹窗仅有 `Target Node ID` 输入框，不支持树状选择；
  - 需要补齐后端参数分离 + 前端选择器能力。

## 可执行任务清单（Checklist）

- [x] `OFFER-DIR-1` 后端新增 StartOfferToDir（源目录/目标目录分离）
  - 目标：新增后端 API，允许发送端独立传入 `sourceDir` 与 `remoteDir`。
  - 涉及模块 / 文件：
    - `internal/services/file/transfer.go`
  - 验收条件：
    - `sourceDir` 用于本地文件定位；
    - `remoteDir` 用于 `protocol.WriteReq.Dir`；
    - 旧 `StartOffer` 保持兼容（默认 `remoteDir=sourceDir`）。
  - 测试点：
    - 有效 source/remote 组合可正常发起；
    - remote 非法路径返回清晰错误；
    - 兼容旧调用路径不回归。
  - 回滚点：
    - 删除新增 API 并恢复 `StartOffer` 原逻辑。

- [x] `OFFER-DIR-2` RetryTask 适配分离目录语义
  - 目标：确保 offer 任务重试时不丢失源目录与目标目录语义。
  - 涉及模块 / 文件：
    - `internal/services/file/transfer.go`
  - 验收条件：
    - 发生失败后 Retry 可再次发起；
    - sourceDir 与 remoteDir 维持正确映射。
  - 测试点：
    - 修改 remoteDir 后发起任务，模拟失败后重试。
  - 回滚点：
    - 恢复 `RetryTask` 旧路径。

- [x] `OFFER-DIR-3` 前端 store 对接新后端能力
  - 目标：`useFileStore.startOffer` 增加 `remoteDir` 参数并调用新 binding。
  - 涉及模块 / 文件：
    - `frontend/src/stores/file.ts`
  - 验收条件：
    - 可从页面传入 `sourceDir + remoteDir` 并成功调用；
    - 输入为空时保持可用默认行为（根目录）。
  - 测试点：
    - remoteDir 与 sourceDir 不同时调用参数正确。
  - 回滚点：
    - 回滚 store 方法签名与调用逻辑。

- [x] `OFFER-DIR-4` File Console Offer 弹窗新增远端目录输入
  - 目标：在弹窗新增 `Remote Dir` 输入，默认回填当前目录，支持修改。
  - 涉及模块 / 文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 用户可编辑远端目录；
    - 发送时使用编辑值；
    - 校验/错误提示友好。
  - 测试点：
    - 当前目录发送（默认值）；
    - 改为其他合法目录发送；
    - 非法目录返回错误提示。
  - 回滚点：
    - 回滚弹窗字段与相关状态逻辑。

- [x] `OFFER-DIR-5` 树状目标节点选择器（类似 Devices）
  - 目标：提供树状节点选择交互，替代“只手输数字”。
  - 涉及模块 / 文件：
    - `frontend/src/components/file/OfferNodeTreePicker.vue`（新增）
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 可展开/折叠节点；
    - 可点击节点选中目标；
    - 选中值与发送逻辑联动。
  - 测试点：
    - 根节点加载、子节点展开、失败重试；
    - 选中后发送到目标节点。
  - 回滚点：
    - 删除新增组件并恢复旧 ID 输入方案。

- [x] `OFFER-DIR-6` 回归验证、Code Review、归档
  - 目标：完成关键路径验证，输出评审结论并归档。
  - 涉及模块 / 文件：
    - `internal/services/file/transfer.go`
    - `frontend/src/stores/file.ts`
    - `frontend/src/pages/File.vue`
    - `frontend/src/components/file/OfferNodeTreePicker.vue`
    - `docs/change/2026-03-06_win-file-offer-dir-node-picker.md`（按当天命名）
  - 验收条件：
    - 后端/前端关键路径验证通过；
    - Review 覆盖需求、架构、性能、可读性、扩展性、稳定性、安全、测试；
    - 归档文档包含任务映射、权衡、验证、回滚方案。
  - 测试点：
    - `go test ./...`；
    - `frontend` 构建检查（环境允许）；
    - 手工验证 Offer 目标目录与树状选择流程。
  - 回滚点：
    - 回滚本次新增/修改文件。

## 依赖关系
- `OFFER-DIR-2` 依赖 `OFFER-DIR-1`；
- `OFFER-DIR-4`/`OFFER-DIR-5` 依赖 `OFFER-DIR-3`；
- `OFFER-DIR-6` 依赖前置任务全部完成。

## 风险与注意事项
- Management binding 不可用时树加载会失败，需明确错误提示与重试入口；
- 若只改前端不改后端，会出现“远端目录输入无效”的行为偏差；
- 修改 `StartOffer` 相关路径后，需重点回归 `RetryTask`；
- 保持协议字段兼容，不新增协议 action/op。
