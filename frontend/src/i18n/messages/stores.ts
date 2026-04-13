// Context: defines localized stores copy used by the Win frontend.

import type { LocaleMessages } from "../types";

export const storesZhCN: LocaleMessages = {
  Management: "管理",
  "App binding '{method}' unavailable": "应用绑定“{method}”不可用",
  "Flow binding '{method}' unavailable": "Flow 绑定“{method}”不可用",
  "TopicBus binding '{method}' unavailable": "主题总线绑定“{method}”不可用",
  "VarPool binding '{method}' unavailable": "变量池绑定“{method}”不可用",
  "Log binding '{method}' unavailable": "日志绑定“{method}”不可用",
  "Management binding '{method}' unavailable": "管理绑定“{method}”不可用",
  "Preset binding '{method}' unavailable": "预设绑定“{method}”不可用",
  "Root node is required.": "必须提供根节点。",
  "Root node must be a positive number.": "根节点必须是正整数。",
  "Login required to query devices.": "查询设备前需要先登录。",
  "Hub ID missing.": "缺少 Hub ID。",
  "Failed to load profile state.": "加载配置状态失败。",
  "Unable to switch profile.": "切换配置失败。",
  "Login required to send Management requests.": "发送管理请求需要先登录。",
  "Target node is required.": "目标节点不能为空。",
  "Target node must be a positive number.": "目标节点必须是正数。",
  "Select a node to load config.": "请选择节点以加载配置。",
  "Select a node to update config.": "请选择节点以更新配置。",
  "Target ID must be a valid number.": "目标 ID 必须是有效数字。",
  "Login required to send TopicBus requests.": "发送主题总线请求前需要先登录。",
  "Topic is required.": "主题不能为空。",
  "Name is required.": "名称不能为空。",
  "Max events must be a positive number.": "最大事件数必须是正整数。",
  "Hub target is unavailable. Re-login and retry.":
    "Hub 目标不可用，请重新登录后重试。",
  "Login required to send VarPool requests.": "发送变量池请求前需要先登录。",
  "Owner NodeID is required.": "拥有者节点 ID 不能为空。",
  "Owner NodeID must be a positive number.": "拥有者节点 ID 必须是正整数。",
  "VarPool list failed (code={code})": "变量池列表加载失败（code={code}）",
  "Variable name is required.": "变量名不能为空。",
  "Owner is required.": "Owner 不能为空。",
  "Variable value is required.": "变量值不能为空。",
  "Failed to update log pause state.": "更新日志暂停状态失败。",
  "Failed to load logs.": "加载日志失败。",
  "Log Window": "日志窗口",
  "Live Log Stream": "实时日志流",
  "Scroll to Bottom": "滚动到底部",
  "TopicBus Window": "主题总线窗口",
  "Listening for every known topic from this moment onward.":
    "从当前时刻开始监听所有已知主题。",
  "Listening only for the selected topic from this moment onward.":
    "从当前时刻开始仅监听所选主题。",
  "Self {id}": "自身 {id}",
  "Target {id}": "目标 {id}",
  "Events in window: {count}": "窗口内事件：{count}",
  "Cache limit: {count}": "缓存上限：{count}",
  "Your own publish action will not echo back here.":
    "你自己发布的消息不会在这里回显。",
  Receive: "接收",
  "Scroll to Latest": "滚动到最新",
  Clear: "清空",
  "Waiting for new TopicBus events in this window.":
    "正在等待该窗口中的新主题总线事件。",
  Send: "发送",
  "Publish Event": "发布事件",
  Publish: "发布",
  "event name": "事件名称",
  "JSON or plain text": "JSON 或纯文本",
  "Choose a topic before sending from the aggregate window.":
    "在聚合窗口发送前请先选择主题。",
  "Topic is locked to this channel for safer publishing.":
    "当前主题已锁定到该频道，发布更安全。",
  "Event published. This window will not echo your own message.":
    "事件已发布。该窗口不会回显你自己的消息。",
  "Failed to publish event.": "发布事件失败。",
  "Failed to initialize TopicBus window.": "初始化主题总线窗口失败。",
  "Node Variables": "节点变量",
  "Lists public variables on a node. Click Add Watch to include it in your watched list.":
    "列出节点上的公开变量。点击“新增监听”可加入你的观察列表。",
  "Owner {owner}": "拥有者 {owner}",
  Load: "加载",
  Search: "搜索",
  "Filter by name...": "按名称筛选...",
  "{count} results": "{count} 条结果",
  "Page {page} / {total}": "第 {page} / {total} 页",
  "Already watched": "已在观察中",
  "Already watched.": "已在观察中。",
  "Not watched": "未观察",
  Watched: "已观察",
  "No matches.": "没有匹配项。",
  "Load a node to view variables.": "加载一个节点以查看变量。",
  Prev: "上一页",
  Next: "下一页",
  "Showing {shown} of {total}": "显示 {shown} / {total}",
  "Connect before listing node variables.": "列出节点变量前请先连接。",
  "Login required to list node variables.": "列出节点变量前需要先登录。",
  "Failed to load node variables.": "加载节点变量失败。",
  "Watch added.": "已加入观察。",
  "Failed to add watch.": "添加观察失败。",
  "Connect to a session before sending TopicBus requests.":
    "发送主题总线请求前请先连接会话。",
  "Login to a node before using TopicBus operations.":
    "使用主题总线操作前请先登录节点。",
  "Primary Action": "主要操作",
  Secondary: "次要操作",
  "Module Checklist": "模块检查清单",
  "Wire session events to the live panel.": "将会话事件接入实时面板。",
  "Connect Wails services to the action controls.":
    "把 Wails 服务接到操作控件上。",
  "Ship the final data tables and inspectors.": "补齐最终的数据表格与检查器。",
  "Next Milestone": "下一里程碑",
  "This view is a placeholder until the full module UI is delivered in the next milestones. It already inherits the global layout, navigation, and profile context.":
    "这个视图目前只是占位，完整模块 UI 会在后续里程碑交付。它已经继承了全局布局、导航和配置上下文。",
  "Stream binding '{method}' unavailable": "Stream 绑定“{method}”不可用",
  "Login required to use Stream controls.": "使用 Stream 控件前需要先登录。",
  "Saved Sources": "已保存 Source",
  "Persistent local list": "本地持久化列表",
  "Saved Consumers": "已保存 Consumer",
  "Restored after login": "登录后自动恢复",
  "Manage your saved local sources, keep the main list compact, and open a separate input studio only when you need to send content.":
    "管理已保存的本地 Source，让主列表保持简洁，只在需要发送内容时打开独立输入界面。",
  "Manage your saved local sources in a compact list and open a dedicated input window only when you need to send content.":
    "用紧凑列表管理已保存的本地 Source，只在需要发送内容时再打开独立输入窗口。",
  "Keep local consumers in a simple list, review current bindings, and subscribe through a dedicated dialog instead of inline forms.":
    "用简洁列表管理本地 Consumer，查看当前绑定关系，并通过独立弹窗完成订阅。",
  "Keep local consumers in a simple list, review current bindings, and subscribe through a dedicated dialog only when you need to change them.":
    "用简洁列表管理本地 Consumer，查看当前绑定关系，只在需要调整时再打开订阅弹窗。",
  "Browse remote catalogs, connect compatible endpoints, and inspect runtime deliveries from the control tab.":
    "在控制页浏览远端目录、连接兼容端点，并查看运行中的 delivery。",
  "Browse remote catalogs, connect compatible endpoints, and open runtime output windows from the control tab.":
    "在控制页浏览远端目录、连接兼容端点，并从这里打开运行时输出窗口。",
  "Choose control pairs through a focused dialog, then inspect runtime deliveries and open output windows from the control tab.":
    "通过聚焦弹窗选择控制配对，然后在控制页查看运行中的 delivery 并打开输出窗口。",
  Source: "源",
  Consumer: "消费者",
  Control: "控制",
  "Local Sources": "本地 Source",
  "Keep local sources persistent and focused. Add or remove sources here, then open a separate studio only when you need to input content.":
    "让本地 Source 持久化并保持聚焦。在这里新增或移除，需要输入内容时再打开独立工作区。",
  "Keep local sources persistent and focused. Add or remove sources here, then open a dedicated input window only when you need to send content.":
    "让本地 Source 保持持久化且聚焦。在这里新增或移除，需要发送内容时再打开独立输入窗口。",
  "{count} bindings": "{count} 个绑定",
  "No descriptor details": "没有描述信息",
  "No active bindings": "暂无活跃绑定",
  Input: "输入",
  "Input Window": "输入窗口",
  "No local sources yet. Use New Source to create one.":
    "还没有本地 Source，点击“新建 Source”即可创建。",
  "Source Details": "Source 详情",
  "Keep the list simple. Use this side panel to review metadata and current bindings before opening the input studio.":
    "保持列表简洁，在这里查看 metadata 和当前绑定关系，再决定是否打开输入工作区。",
  "Current bindings": "当前绑定",
  "Nothing is bound to this source yet.": "这个 Source 目前还没有绑定对象。",
  "Open Input Studio": "打开输入工作区",
  "Remove Source": "移除 Source",
  "Select a local source to inspect its metadata and bindings.":
    "选择一个本地 Source 查看它的 metadata 和绑定关系。",
  "Local Consumers": "本地 Consumer",
  "Store local consumers as a compact list. Review current bindings here and open a separate dialog only when you want to subscribe to a source.":
    "把本地 Consumer 保持为紧凑列表，在这里查看绑定关系，只有要订阅时才打开独立弹窗。",
  "No current source bindings": "当前没有绑定任何 Source",
  "No local consumers yet. Use New Consumer to create one.":
    "还没有本地 Consumer，点击“新建 Consumer”即可创建。",
  "Consumer Details": "Consumer 详情",
  "This side panel only shows what the consumer is currently bound to. Use the subscription dialog when you need to change it.":
    "这个侧栏只显示当前绑定到了哪些 Source，需要调整时请使用订阅弹窗。",
  "Current source bindings": "当前 Source 绑定",
  "Unknown source": "未知 Source",
  "This consumer is not bound to any source.":
    "这个 Consumer 目前没有绑定任何 Source。",
  "Open Subscribe Dialog": "打开订阅弹窗",
  "Remove Consumer": "移除 Consumer",
  "Select a local consumer to review its bindings.":
    "选择一个本地 Consumer 查看它的绑定关系。",
  "Query producer catalogs and keep control selections focused here.":
    "查询 producer 目录，并在这里保持控制面的选择焦点。",
  "Query consumer endpoints and compare them before connecting.":
    "查询 consumer 端点，并在连接前完成比对。",
  "Leave target aligned with Hub unless you are routing control requests elsewhere.":
    "除非你明确要把控制请求打到别处，否则 target 保持为 Hub 即可。",
  "Choose one source and one consumer, then connect or subscribe from this focused panel.":
    "选择一个 Source 和一个 Consumer，然后在这个聚焦面板里执行 connect 或 subscribe。",
  "Selected Pair": "当前配对",
  "Select Pair": "选择配对",
  "Change Pair": "更换配对",
  "Use a focused picker for remote source and consumer selection, instead of leaving both catalogs on the page.":
    "使用聚焦选择框完成远端 Source 和 Consumer 的选择，而不是把两个目录常驻在页面上。",
  "Selected Source": "已选 Source",
  "No source selected yet.": "当前还没有选中的 Source。",
  "Selected Consumer": "已选 Consumer",
  "No consumer selected yet.": "当前还没有选中的 Consumer。",
  "Subscribe only works when the selected consumer belongs to this node.":
    "只有当选中的 Consumer 属于当前节点时，才能执行 subscribe。",
  "Inspect deliveries and send lightweight runtime signals from one place.":
    "在同一个区域查看 delivery，并发送轻量运行时信号。",
  "Observed {count}": "观察到 {count}",
  "Stream control plane refreshed.": "Stream 控制面已刷新。",
  "Failed to refresh Stream control plane.": "刷新 Stream 控制面失败。",
  "Frames {frames} · Bytes {bytes}": "帧 {frames} · 字节 {bytes}",
  "Frames {frames} · Bytes {bytes} · ACK {ack} · Flags {flags}":
    "帧 {frames} · 字节 {bytes} · ACK {ack} · Flags {flags}",
  "Producer {producer} · Consumer {consumer}":
    "Producer {producer} · Consumer {consumer}",
  "No known deliveries yet.": "当前还没有已知 delivery。",
  "Output Window": "输出窗口",
  "Selected Delivery": "已选 Delivery",
  "Waiting for text frames...": "正在等待文本帧...",
  "Create a persistent local source without filling the main page with form fields.":
    "创建持久化本地 Source，而不是在主页面堆满表单。",
  "Create a persistent local consumer endpoint without expanding the main list into a form page.":
    "创建持久化本地 Consumer 端点，而不是把主列表展开成表单页面。",
  "Source Input Studio": "Source 输入工作区",
  "Use a dedicated workspace for source input so the list page stays compact.":
    "把 Source 输入放进独立工作区，让列表页面保持紧凑。",
  "{count} active bindings": "{count} 个活跃绑定",
  "Text input": "文本输入",
  "Type text for the active deliveries of this source...":
    "为这个 Source 的活跃 deliveries 输入文本...",
  "Send Text": "发送文本",
  "Direct input is currently available only for text sources. Other kinds still use the control plane and runtime viewer.":
    "当前只有 text Source 支持直接输入，其他类型仍通过控制面和运行时查看器使用。",
  "Recent sends": "最近发送",
  "Sent to {count} deliveries": "已发送到 {count} 个 delivery",
  "No text has been sent from this studio yet.": "这个工作区还没有发送过文本。",
  "No text has been sent from this window yet.": "这个窗口里还没有发送过文本。",
  "Subscribe Consumer": "订阅 Consumer",
  "Choose a source from the current node or another node, then subscribe without expanding the main consumer list into a form.":
    "从当前节点或其他节点选择一个 Source，然后完成订阅，而不需要把主列表展开成表单。",
  "No subscription sources loaded yet.": "还没有加载到可订阅的 Source。",
  "Sources refreshed.": "Source 列表已刷新。",
  "Failed to query sources.": "查询 Source 失败。",
  "Consumers refreshed.": "Consumer 列表已刷新。",
  "Failed to query consumers.": "查询 Consumer 失败。",
  "Runtime deliveries refreshed.": "运行时 delivery 已刷新。",
  "Failed to load deliveries.": "加载 delivery 失败。",
  "Local stream lists refreshed.": "本地 Stream 列表已刷新。",
  "Failed to refresh local stream lists.": "刷新本地 Stream 列表失败。",
  "Subscription sources refreshed.": "订阅 Source 列表已刷新。",
  "Failed to load subscription sources.": "加载订阅 Source 列表失败。",
  "Local source created.": "本地 Source 已创建。",
  "Failed to create local source.": "创建本地 Source 失败。",
  "Local consumer created.": "本地 Consumer 已创建。",
  "Failed to create local consumer.": "创建本地 Consumer 失败。",
  "Text sent to source.": "文本已发送到 Source。",
  "Failed to send text to source.": "发送文本到 Source 失败。",
  "Select a source before subscribing.": "订阅前请先选择一个 Source。",
  "Failed to subscribe.": "订阅失败。",
  "Delivery connected.": "Delivery 已连接。",
  "Failed to connect delivery.": "连接 Delivery 失败。",
  "Delivery disconnected.": "Delivery 已断开。",
  "Failed to disconnect delivery.": "断开 Delivery 失败。",
  "Delivery unsubscribed.": "Delivery 已退订。",
  "Failed to unsubscribe delivery.": "退订 Delivery 失败。",
  "Signal sent.": "信号已发送。",
  "Failed to send signal.": "发送信号失败。",
  "Source removed.": "Source 已移除。",
  "Failed to remove source.": "移除 Source 失败。",
  "Consumer removed.": "Consumer 已移除。",
  "Failed to remove consumer.": "移除 Consumer 失败。",
  "Failed to load Stream settings.": "加载 Stream 设置失败。",
  "Stream auto-restore incomplete.": "Stream 自动恢复未完全成功。",
  "Stream input window was blocked by browser popup policy.":
    "Stream 输入窗口被浏览器弹窗策略阻止。",
  "Stream output window was blocked by browser popup policy.":
    "Stream 输出窗口被浏览器弹窗策略阻止。",
  "Stream Source Window": "Stream 输入窗口",
  "Send text into a local source from a dedicated window.":
    "在独立窗口中向本地 Source 发送文本。",
  "Stream Delivery Window": "Stream 输出窗口",
  "Inspect a single runtime delivery in a dedicated window.":
    "在独立窗口中查看单个运行时 delivery。",
  "Source Input Window": "Source 输入窗口",
  "Send text from a dedicated window so the main Stream page stays focused on the list.":
    "在独立窗口中发送文本，让主 Stream 页面继续只负责列表。",
  "Control Pair Picker": "控制配对选择框",
  "Choose a source and a consumer from the current control target, then connect or subscribe without expanding the main page.":
    "从当前控制目标中选择一个 Source 和一个 Consumer，然后完成 connect 或 subscribe，而不需要把主页面展开成目录页。",
  "Select Source": "选择 Source",
  "Select Consumer": "选择 Consumer",
  "Loading Stream source window...": "正在加载 Stream 输入窗口...",
  "Source not found in local catalog.": "本地目录中找不到该 Source。",
  "Open this window from the Source list after a local source is created.":
    "请先创建本地 Source，再从 Source 列表打开这个窗口。",
  "Failed to initialize Source input window.": "初始化 Source 输入窗口失败。",
  "No active consumers are currently bound to this source.":
    "当前没有活跃 Consumer 绑定到这个 Source。",
  "This source uses a local media file as input. Once a delivery becomes active, the producer starts sending it immediately in chunk mode.":
    "这个 Source 使用本地媒体文件作为输入。一旦有 delivery 进入 active，producer 就会以 chunk 模式立即开始发送。",
  "Configured file": "已配置文件",
  "No media file configured yet.": "当前还没有配置媒体文件。",
  "Replace File": "替换文件",
  "Media file configured for source.": "已为 Source 配置媒体文件。",
  "Failed to configure media file for source.": "为 Source 配置媒体文件失败。",
  "Delivery Output Window": "Delivery 输出窗口",
  "Inspect one runtime delivery in a dedicated window.":
    "在独立窗口中查看单个运行时 delivery。",
  "Loading Stream delivery window...": "正在加载 Stream 输出窗口...",
  "Delivery not found in runtime state.": "运行时状态中找不到该 delivery。",
  "Open this window from Runtime Deliveries after a connection or subscription is active.":
    "请在连接或订阅生效后，从 Runtime Deliveries 打开这个窗口。",
  "Failed to initialize Delivery output window.":
    "初始化 Delivery 输出窗口失败。",
  "No runtime stats available yet.": "当前还没有运行时统计信息。",
  "Playback failed for this media stream.": "该媒体流播放失败。",
  "Buffering media stream...": "正在缓冲媒体流...",
  "Progressive playback active. Received {bytes} bytes.":
    "渐进播放已启动，已接收 {bytes} 字节。",
  "Runtime summary": "运行时摘要",
  Frames: "帧数",
  Bytes: "字节数",
  Updated: "更新时间",
  "Source ID is required.": "必须提供 Source ID。",
  "Text content is required.": "文本内容不能为空。",
  "Source not found.": "找不到 Source。",
  "Only text sources support direct input.": "只有 text Source 支持直接输入。",
  "Only media sources support file input.": "只有媒体 Source 支持文件输入。",
  "Selected media file kind does not match the source kind.":
    "所选媒体文件类型与 Source 类型不匹配。",
  "Selected media file content type does not match the source content type.":
    "所选媒体文件内容类型与 Source 内容类型不匹配。",
  "A media file is required for non-text sources.":
    "非 text Source 必须配置媒体文件。",
  "Selected file is not a supported audio or video file.":
    "所选文件不是受支持的音频或视频文件。",
  "Failed to select a media file.": "选择媒体文件失败。",
  "Media file": "媒体文件",
  "Select a local media file": "选择一个本地媒体文件",
  "Choose File": "选择文件",
  "Clear File": "清除文件",
  "This round only supports file-backed media input with bounded/chunk delivery.":
    "本轮只支持基于文件的媒体输入，并要求使用 bounded/chunk 传输。",
  "Desktop capture is only available for video sources.":
    "桌面采集只适用于 video Source。",
  "Only desktop video sources support capture input.":
    "只有桌面 video Source 支持采集输入。",
  "Desktop capture requires at least one active delivery.":
    "桌面采集至少需要一个活跃 delivery。",
  "Capture payload is required unless the chunk is final.":
    "除非当前 chunk 是 final，否则必须提供采集 payload。",
  "Input mode": "输入模式",
  "Local File": "本地文件",
  "Desktop Capture": "桌面采集",
  "File-backed media input uses bounded/chunk delivery.":
    "基于文件的媒体输入使用 bounded/chunk 传输。",
  "Desktop capture starts from the Source Window with a user click and does not persist a previous screen selection.":
    "桌面采集需要在 Source Window 中由用户点击启动，且不会持久化上一次选择的屏幕内容。",
  "This round captures desktop video only. System audio is not included.":
    "本轮只采集桌面视频，不包含系统音频。",
  "Requesting desktop capture permission...": "正在请求桌面采集权限...",
  "Stopping desktop capture...": "正在停止桌面采集...",
  "Desktop capture is live for {count} deliveries. Sent {chunks} chunks / {bytes} bytes.":
    "桌面采集已对 {count} 个 delivery 生效，已发送 {chunks} 个 chunk / {bytes} 字节。",
  "Desktop capture is idle. Start it from this window when deliveries are active.":
    "桌面采集当前空闲，请在有活跃 delivery 时从这个窗口启动。",
  "Desktop capture stopped.": "桌面采集已停止。",
  "Failed to stop desktop capture.": "停止桌面采集失败。",
  "Desktop capture is not supported in this runtime.":
    "当前运行时不支持桌面采集。",
  "Desktop capture recorder failed.": "桌面采集录制器失败。",
  "Desktop sharing stopped by the system.": "桌面共享已被系统停止。",
  "Desktop capture started.": "桌面采集已启动。",
  "Failed to start desktop capture.": "启动桌面采集失败。",
  "No active deliveries remain. Capture stopped.":
    "已经没有活跃 delivery，采集已停止。",
  "Desktop source configuration changed. Capture stopped.":
    "桌面 Source 配置已变化，采集已停止。",
  "Source window closed. Desktop capture stopped.":
    "Source 窗口已关闭，桌面采集已停止。",
  "Desktop capture sends live media chunks directly into the current active deliveries.":
    "桌面采集会把实时媒体 chunk 直接送入当前活跃 deliveries。",
  "Capture status": "采集状态",
  "Capture summary": "采集摘要",
  "Chunks {chunks} · Bytes {bytes} · Deliveries {count}":
    "Chunk {chunks} · 字节 {bytes} · Delivery {count}",
  "Local preview": "本地预览",
  "Preview appears here after the desktop capture starts.":
    "桌面采集启动后，本地预览会显示在这里。",
  "Start Capture": "开始采集",
  "Stop Capture": "停止采集",
  "Capture session": "采集会话",
  Deliveries: "Delivery 数",
};
