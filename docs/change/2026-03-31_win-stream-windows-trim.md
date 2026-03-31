# 2026-03-31 Win Stream Windows Trim

## Context

- Workflow: `feat/win-stream-windows-trim`
- Requirements impact: `none`
- Specs impact: `none`
- Goal:
  - 移除 Stream 顶部 4 块统计卡
  - 压缩 source / consumer / delivery 列表信息密度
  - 把输入与输出改成真正的独立窗口，而不是主页面内工作区

## Changes

### 主页面收敛

- `frontend/src/pages/Stream.vue`
  - 删除顶部 `Saved Sources / Saved Consumers / Known Deliveries / Last Runtime Event` 统计区
  - 把 `Source` / `Consumer` tab 改成纯列表主导，只保留名称、`kind` 和当前绑定摘要
  - 移除页内 `Source Input Studio` Overlay
  - 移除 `Control` tab 中的内嵌 delivery 详情查看块
  - 为 source 行增加 `Input Window`
  - 为 runtime delivery 行增加 `Output Window`
  - popup 被拦截时显式 toast 提示

### 独立窗口

- `frontend/src/windows/StreamSourceWindow.vue`
  - 新增 source 输入窗口
  - 依据 `sourceId` 加载本地 source
  - `text` source 可发送文本并保留窗口内最近发送记录
  - source 不存在或类型不支持时显示空态
- `frontend/src/windows/StreamDeliveryWindow.vue`
  - 新增 delivery 输出窗口
  - 依据 `deliveryId` 查看运行时 delivery
  - `text` delivery 显示文本帧
  - 其他类型显示 stats 摘要
  - delivery 不存在时显示空态
- `frontend/src/router/index.ts`
  - 新增 `/stream-source-window`
  - 新增 `/stream-delivery-window`

### 文案与测试

- `frontend/src/i18n/messages/stores.ts`
  - 补充精简页面、窗口标题、空态、popup blocked 与 runtime 文案的 `zh-CN`
- `frontend/src/pages/Stream.test.ts`
  - 更新为校验顶部统计卡已移除
  - 校验 source 输入改为打开独立窗口
  - 校验 delivery 输出改为打开独立窗口
  - 保留 consumer 订阅弹窗关键路径测试

## Validation

- `git diff --check`
  - 通过
  - 仅有 Git 的 LF/CRLF 预警，无 diff 格式错误
- `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 通过
- `npm exec vitest run src/pages/Stream.test.ts`
  - 未通过执行
  - 当前 worktree 缺少前端依赖，`vitest` / `@vitejs/plugin-vue` 不存在
- `npm run build`
  - 未通过执行
  - 当前 worktree 缺少 `frontend/node_modules/vite/bin/vite.js`

## Rollback

- 回退以下文件即可撤销本轮前端改动：
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/router/index.ts`
  - `frontend/src/windows/StreamSourceWindow.vue`
  - `frontend/src/windows/StreamDeliveryWindow.vue`
  - `frontend/src/i18n/messages/stores.ts`
  - `frontend/src/pages/Stream.test.ts`
