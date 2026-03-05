# Plan - MyFlowHub-Win：File Console 新建文件夹（mkdir）

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/file-console-mkdir`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-mkdir`
- Base：`main`
- 当前状态：已完成（待你确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 在 File Console 增加“新建文件夹”入口；
  - 前端可输入目录名并调用后端创建；
  - 本地/远端节点均支持（按 file 子协议 `write(op=mkdir)`）。
- 当前状态：
  - `New Folder` 已支持，目录创建成功后自动刷新列表；
  - 本地节点走本地 mkdir，远端节点走 `write(op=mkdir)` 并 await 响应。

## 可执行任务清单（Checklist）

- [x] `WIN-MKDIR-1` 后端 FileService 新增 mkdir API
  - 目标：新增 `CreateDirSimple`（必要校验 + await 响应）。
  - 涉及文件：
    - `internal/services/file/service.go`
    - `internal/services/file/local.go`（如需本地 helper）
  - 验收条件：
    - 非法输入返回明确错误；
    - 目标节点返回成功后接口返回 nil；
    - 失败时透传可读错误。
  - 测试点：
    - 本地创建、非法名称、目标缺失。
  - 回滚点：
    - 回滚新增 API 与调用路径。

- [x] `WIN-MKDIR-2` 前端 store 接入 mkdir 调用
  - 目标：`useFileStore` 新增 `createDir` 方法封装 Wails binding。
  - 涉及文件：
    - `frontend/src/stores/file.ts`
  - 验收条件：
    - 页面可通过 store 调用创建目录。
  - 回滚点：
    - 回滚 store 新增方法。

- [x] `WIN-MKDIR-3` File Console UI 增加“New Folder”
  - 目标：新增按钮 + 弹窗输入目录名 + 成功后刷新列表。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 可创建目录并在列表看到；
    - 输入为空/非法有提示；
    - 不影响既有 Up/Download/Offer/拖拽逻辑。
  - 测试点：
    - 根目录/子目录创建；
    - 已存在目录行为（应成功或合理提示）。
  - 回滚点：
    - 回滚页面新增弹窗与按钮。

- [x] `WIN-MKDIR-4` 回归验证与归档
  - 目标：执行 Go 测试并补充变更归档文档。
  - 涉及文件：
    - `docs/change/2026-03-05_win-file-console-mkdir.md`
  - 验收条件：
    - `go test ./...` 通过；
    - 文档包含任务映射、验证与回滚方案。

## 风险与注意事项
- `CreateDir` 需与现有 `fileSanitizeDir/fileSanitizeName` 一致，避免目录穿越。
- 若远端节点尚未升级到 `subproto/file v0.1.2`，`op=mkdir` 可能返回 `invalid op`，需给前端友好错误提示。
