# Plan - MyFlowHub-Win：File Console 路径可点击 + 图标化 + 右键菜单

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/file-console-ui-upgrade`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-file-console-ui`
- Base：`main`
- 当前状态：已完成（阶段 1~4 完成，待你确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 顶部目录路径支持逐级点击回退；
  - 顶部按钮改为以图标为主；
  - 文件列表项增加目录/文件图标；
  - 新增右键菜单，支持：上传、下载、创建文件夹、发送到远程设备。
- 当前状态：
  - 已有 `createDir`、`startPull`、`startOffer`、`importLocalFiles` 能力；
  - File Console 已支持目录浏览、双击进入、拖拽上传（本地节点）、下载与 offer（按钮入口）；
  - 待补齐入口与交互层（路径栏、图标按钮、右键菜单）。

## 可执行任务清单（Checklist）

- [x] `FC-UI-1` 顶部路径栏改为可点击 Breadcrumb
  - 目标：把 `Dir: xxx` 改为可点击目录段，支持跳转到任意上级目录。
  - 涉及模块 / 文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 根目录显示 `/`；
    - 子目录显示可点击段（例：`/a/b/c`）；
    - 点击任意段后目录列表刷新为目标目录。
  - 测试点：
    - 从根目录进入多级目录后逐级回退；
    - 点击中间层级直接跳转。
  - 回滚点：
    - 回滚 `File.vue` breadcrumb 相关计算与模板区块。

- [x] `FC-UI-2` 顶部按钮图标化 + 列表项图标化
  - 目标：顶部操作按钮以图标呈现，文件列表使用文件/目录图标替代文本徽标。
  - 涉及模块 / 文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 顶部按钮主要由图标呈现，保留 title 提示；
    - 文件/目录前显示语义化图标，原有选择与双击行为不变。
  - 测试点：
    - 按钮禁用态、hover 态、title 提示仍正确；
    - 图标与文字布局在桌面窗口下不重叠。
  - 回滚点：
    - 回滚 `File.vue` 顶栏按钮与列表项渲染修改。

- [x] `FC-UI-3` 新增目录列表右键菜单（Upload/Download/New Folder/Send to Remote）
  - 目标：在列表项/列表区域接管右键菜单并接入现有动作能力。
  - 涉及模块 / 文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 右键可弹出自定义菜单，系统默认菜单被抑制；
    - 菜单包含 4 个动作，且按上下文禁用不适用项；
    - 点击动作后可触发对应流程（上传导入、下载弹窗、新建文件夹弹窗、offer 弹窗）。
  - 测试点：
    - 本地/远端节点切换后的可用性变化；
    - 选中文件/目录/无选中三种状态下的禁用逻辑；
    - 菜单靠近窗口边缘时位置自动修正。
  - 回滚点：
    - 回滚 `File.vue` 右键菜单状态、模板与动作分发逻辑。

- [x] `FC-UI-4` 回归验证、Code Review 与归档
  - 目标：完成关键路径验证、输出评审结论并生成归档文档。
  - 涉及模块 / 文件：
    - `frontend/src/pages/File.vue`
    - `docs/change/2026-03-05_win-file-console-ui-upgrade.md`（如日期变化按当天命名）
  - 验收条件：
    - 手工验证通过（路径点击、图标按钮、右键四动作）；
    - Code Review 覆盖需求/架构/性能/可读性/扩展性/稳定性/测试；
    - 归档文档包含任务映射、权衡、验证与回滚方案。
  - 测试点：
    - `frontend` 构建或等价静态检查（环境允许时）；
    - 运行态手工验证步骤记录完整。
  - 回滚点：
    - 回滚上述前端文件与归档文档即可撤销本次改动。

## 依赖关系
- `FC-UI-2` 依赖 `FC-UI-1` 的顶部布局稳定；
- `FC-UI-3` 可与 `FC-UI-2` 并行开发，但提交前需整体联调；
- `FC-UI-4` 依赖前 3 项完成后执行。

## 风险与注意事项
- 当前仓库未提交 `frontend/wailsjs` 生成文件，若本地缺少生成步骤可能影响前端类型检查；
- 上传动作若走文件选择器，需要将 `File` 对象解析为系统路径后再调用 `ImportLocalFiles`；
- 右键菜单必须与现有拖拽区域共存，避免影响 `OnFileDrop` 行为；
- 保持最小改动原则，不改后端协议和 store 接口签名。
