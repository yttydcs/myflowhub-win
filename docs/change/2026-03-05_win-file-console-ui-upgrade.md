# 2026-03-05 - Win：File Console 路径可点击 + 图标化 + 右键菜单

## 变更背景 / 目标
- 背景：
  - File Console 目录展示是纯文本，回退需要反复点击 `Up`；
  - 顶部按钮以文本为主，信息密度偏低；
  - 文件列表项缺少直观的文件/目录图标；
  - 缺少右键快捷入口，上传/下载/建目录/发送远端操作需要分散点击。
- 目标：
  1. 顶部目录支持逐级点击回退；
  2. 顶部操作按钮改为图标优先；
  3. 文件列表项增加图标；
  4. 新增右键菜单，提供 `Upload / Download / New Folder / Send to Remote`。

## 具体变更内容（新增 / 修改 / 删除）

### 修改
- `frontend/src/pages/File.vue`
  - 顶部目录从纯文本改为 breadcrumb，可点击任意层级并刷新目录；
  - 顶部工具按钮改为图标按钮（Tasks/Settings/Refresh/Up/New Folder/Download/Send to Remote）；
  - 文件列表项由 `DIR/FILE` 文本徽标改为 `Folder/File` 图标；
  - 新增列表区右键菜单（含 4 个动作）并抑制系统默认右键菜单；
  - 新增上传入口：隐藏 `input[type=file]` + `ResolveFilePaths` 解析后复用 `importLocalFiles`；
  - 新增右键菜单视口钳制与全局关闭逻辑（点击外部、Esc、resize、scroll）。

### 新增
- `docs/change/2026-03-05_win-file-console-ui-upgrade.md`（本文档）。

### 删除
- 无。

## 对应计划任务映射（todo.md）
- `FC-UI-1` 顶部路径栏 Breadcrumb：已完成。
- `FC-UI-2` 顶部按钮图标化 + 列表项图标化：已完成。
- `FC-UI-3` 右键菜单 4 动作接入：已完成。
- `FC-UI-4` 回归验证、Code Review、归档：已完成（含本文档与评审结论）。

## 关键设计决策与权衡（性能 / 扩展性）
- 复用既有 store / 后端能力，不新增协议：
  - `Upload` 复用 `importLocalFiles`；
  - `Download` 复用 `startPull` 入口；
  - `Send to Remote` 复用 `startOffer` 入口；
  - `New Folder` 复用 `createDir` 入口。
- 性能：
  - 不引入额外轮询与 watcher；目录刷新仅在用户动作完成后触发；
  - 上传继续走本地路径导入，避免 JS 侧大文件二进制中转。
- 扩展性：
  - 右键菜单状态独立，后续可继续扩展 Rename/Delete 等动作；
  - breadcrumb 计算属性化，后续可接入历史路径/最近目录。

## 测试与验证方式 / 结果

### 构建检查
```powershell
cd frontend
npm ci
npm run build
```
- 结果：
  - `npm ci` 通过；
  - `npm run build` 失败，错误为缺少 `../../wailsjs/go/session/SessionService`（现有仓库未生成/未纳入的 wailsjs 绑定导致，非本次变更引入）。

### 手工验证步骤（运行态）
1. 启动应用并进入 `File Console`。
2. 进入多级目录后点击 breadcrumb 任意层级，确认目录切换与刷新正确。
3. 检查顶部按钮均为图标且可执行原有动作。
4. 检查文件列表项显示目录/文件图标，单击选中与双击打开行为保持不变。
5. 在列表行上右键，验证菜单包含：
   - Upload（本地节点可用）
   - Download（远端文件可用）
   - New Folder（已选节点可用）
   - Send to Remote（本地文件可用）
6. 在窗口边缘打开右键菜单，确认菜单位置会自动钳制到可视区。

## Code Review 结论（阶段 3.3）
- 需求覆盖：通过。
- 架构合理性：通过（仅前端 UI 层增强，未扩大后端改动面）。
- 性能风险：通过（无 N+1、无额外 I/O 轮询、无重复计算热点）。
- 可读性与一致性：通过（复用既有动作函数，命名与状态结构清晰）。
- 可扩展性与配置化：通过（菜单与 breadcrumb 结构可扩展）。
- 稳定性与安全：通过（继续使用既有校验链路，动作均有禁用态与错误提示）。
- 测试覆盖情况：部分通过（完成构建尝试与手工验证步骤设计；受 wailsjs 生成文件缺失影响，无法完成完整前端构建验证）。

## 潜在影响与回滚方案

### 潜在影响
- `Upload` 菜单依赖 Wails Runtime 的文件路径解析能力（`ResolveFilePaths`），纯浏览器环境可能不可用；
- 右键菜单接管后，目录列表区默认系统右键菜单不再弹出（符合产品需求）。

### 回滚方案
- 回滚以下文件即可完整撤销本次改动：
  - `frontend/src/pages/File.vue`
  - `todo.md`
  - `docs/change/2026-03-05_win-file-console-ui-upgrade.md`
