# Plan - MyFlowHub-Win：File Console 按钮可理解性 + 拖拽放置导入

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`fix/file-console-dnd-upload`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-file-console-dnd`
- Base：`main`
- 状态：已完成（待用户确认是否结束 workflow）

## 1) 目标与当前状态
- 目标：
  - 解释并修正 File Console 中 `Up` / `Download` “点击似乎无反应”的可理解性问题。
  - 支持拖拽文件到 File Console 当前目录，完成“直接放置到对应位置”的导入能力（V1：本地节点）。
- 当前状态：
  - `Up` 按钮仅在非根目录可用；根目录时禁用（无明显提示）。
  - `Download` 仅对“远端节点 + 选中文件”可用；其行为是“打开下载任务参数弹窗”，不是立即下载。
  - 当前未启用 Wails 文件拖拽（`EnableFileDrop` 未开启），无拖拽导入链路。

## 2) 任务清单（Checklist）

- [x] `FC-DND-1` 明确按钮行为与禁用原因提示（前端）
  - 目标：为 `Up` / `Download` 增加可理解提示，降低“无反应”感知。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - `Up` 在根目录显示“已在根目录”语义提示；
    - `Download` 在禁用时可看出触发条件（需远端文件被选中）。
  - 测试点：
    - 本地节点、远端节点、目录/文件选中状态切换时按钮提示变化正确。
  - 回滚点：
    - 回滚 `File.vue` 中按钮提示相关改动。

- [x] `FC-DND-2` 启用并接入 Wails 文件拖拽（前端 + 启动配置）
  - 目标：允许将外部文件拖入 File Console 列表区域。
  - 涉及文件：
    - `main.go`
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 拖拽到目录列表区域可触发导入逻辑；
    - 仅本地节点允许导入，远端节点拖拽给出明确提示。
  - 测试点：
    - 拖拽区域高亮状态可见；
    - 回调收到文件路径并进入导入流程。
  - 回滚点：
    - 回滚 `main.go` 的 DragAndDrop 配置；
    - 回滚 `File.vue` 的 OnFileDrop 监听逻辑。

- [x] `FC-DND-3` 后端导入能力（安全默认 + 错误处理）
  - 目标：新增 `ImportLocalFiles`，将拖拽源文件复制到 `BaseDir/currentDir`。
  - 涉及文件：
    - `internal/services/file/local.go`
    - `frontend/src/stores/file.ts`
  - 验收条件：
    - 仅接受常规文件；目录/非法路径被跳过并返回原因；
    - 目标路径受 `fileSanitizeDir/fileResolvePaths` 约束，禁止越界；
    - 默认不覆盖同名文件（安全默认），可返回跳过列表；
    - 成功后刷新目录列表。
  - 测试点：
    - 正常导入、重复文件跳过、目标目录非法、目录拖入跳过。
  - 回滚点：
    - 回滚新增导入 API 与前端调用。

- [x] `FC-DND-4` 单测与回归验证
  - 目标：补充导入逻辑关键路径测试，执行回归。
  - 涉及文件：
    - `internal/services/file/import_test.go`（如需新增）
  - 验收条件：
    - 新增测试通过；
    - 现有 `go test ./...` 不回归。
  - 测试点：
    - 覆盖导入成功/覆盖策略/非法输入。
  - 回滚点：
    - 回滚新增测试文件。

- [x] `FC-DND-5` Code Review（3.3）与归档（4）
  - 目标：按要求完成逐项审查并归档到 `docs/change`。
  - 涉及文件：
    - `docs/change/2026-03-05_win-file-console-dnd-upload.md`
  - 验收条件：
    - 评审结论完整（通过/不通过）；
    - 归档文档包含任务映射、权衡、验证与回滚方案。
  - 回滚点：
    - 文档层无需代码回滚。

## 3) 依赖与风险
- 依赖：
  - Wails v2 拖拽能力（`options.DragAndDrop.EnableFileDrop` + 前端 `OnFileDrop`）。
- 风险：
  - 平台差异导致拖拽事件触发行为不同；需在 Windows 运行态冒烟。
  - 大文件复制耗时导致 UI 感知延迟；V1 先保证正确性与可观测 toast，后续可扩展进度事件。

## 4) 注意事项
- 本次拖拽导入范围限定为“本地节点当前目录”；远端节点不直接写入，避免越权与协议复杂度上升。
- 默认不覆盖同名文件，避免误覆盖；若后续需要覆盖策略，回到本计划新增任务确认后再做。
