# Plan - HeaderTcp v2（32B）+ Core 路由统一（Win）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`refactor/hdrtcp-v2`
- Worktree：`d:\\project\\MyFlowHub3\\worktrees\\hdrtcp-v2\\MyFlowHub-Win`
- 目标 PR：PR1（跨 3 个 repo 同步提交/合并）

## 项目目标
1) 配合 Core/Server 完成 **HeaderTcp v2（32B）big-bang** 升级，Win 端能正常与 Server 互通。  
2) Win 端不再“重新实现一套协议机制”：transport codec 与协议类型尽量复用 Core/Server（或后续的协议库），减少漂移。  
3) 输出可执行的冒烟验证步骤：能启动 Win 并连接到 Server，跑通最小链路（本 PR 仅验证关键路径，Linux 构建暂忽略）。

## 范围
### 必须（PR1）
- 适配 Core 的 `IHeader` v2 接口变更（编译通过）
- transport 编解码切换到 HeaderTcp v2（32B）
- 更新 Win 侧构造 header 的位置（`Major/SubProto/Source/Target/MsgID/Timestamp/...`），确保与新路由规则兼容
- 冒烟验证步骤（启动 + 连接 server）

### 不做（本 PR）
- Linux 构建/验收（用户已允许忽略）
- UI/前端大重构（与本次 wire 升级无关）
- 子协议拆独立 repo/go module（另起 PR2+）

## 已确认的关键决策（来自阶段 2）
- 兼容策略：**S3 / big-bang**；切换后 **v1 不再兼容**。
- HeaderTcp v2：**32B（+8B）**。
- 路由框架规则：**MajorCmd 不由 Core 自动转发，必须进 handler；MajorMsg/OK/Err 走 Core 快速转发**。
- 语义基线：`TargetID=0` 仅表示“下行广播不回父”，不能表示上送父节点。
- Win 定位：UI 以 SDK/hook/事件流方式调用，尽量不复刻协议细节。

## 问题清单（阻塞：是）
> 与 Core/Server 共用的 wire 细节确认项；未确认禁止进入阶段 3.2。

1) HeaderTcp v2 `magic` 值（建议 `0x4D48`）是否确认？
2) `hop_limit` 默认值/语义是否确认？（建议默认 `16`，转发递减）
3) `trace_id` 生成策略是否确认？（建议发送侧自动补齐随机 uint32；响应继承；转发不改）
4) `timestamp` 单位是否确认？（建议保持 Unix 秒 `uint32`）

## 任务清单（Checklist）

### W1 - transport codec 适配 HeaderTcp v2（32B）
- 目标：Win 端收发使用 v2 头部，消除与 Server 的重复实现/漂移点（优先复用 Core 的 header codec）。
- 涉及模块/文件（预期）：
  - `internal/services/transport/codec.go`（编解码）
  - `internal/session/session.go`、`internal/services/session/service.go`（连接/登录链路 header 构造）
  - 其他服务中直接构造 `header.HeaderTcp` 的位置（`rg WithMajor` 排查）
- 验收条件：
  - `go test ./...` 通过（若本 repo 有测试）。
  - 本地可与同批次的 Server worktree 建立连接并完成一次最小交互（见 W3）。
- 测试点：
  - 收包：能解出 v2 头（magic/ver/hdr_len 校验通过），payload 长度正确。
  - 发包：server 能识别并响应。
- 回滚点：
  - 将 v2 适配独立提交；可 revert。

### W2 - 与 Core 路由规则对齐（Major/Target 语义）
- 目标：确保 Win 端各协议调用在 `Major` 使用上符合新框架规则，避免依赖旧的 Core “自动转发 Cmd” 行为。
- 涉及模块/文件（预期）：
  - `internal/services/**`（尤其 file：CTRL/DATA/ACK）
- 验收条件：
  - 控制面请求使用 `MajorCmd`；通知/数据使用 `MajorMsg`；响应使用 `MajorOKResp/MajorErrResp`（以实际协议约定为准）。
- 回滚点：
  - 每个协议按提交拆分，可单独回滚。

### W3 - 冒烟验证步骤（启动并连接 server）
- 目标：提供“可复制粘贴执行”的验证步骤，覆盖最小链路：启动 server → 启动 win → 连接 → 触发至少一个 Cmd → 得到 OK/Err 响应。
- 验收条件：
  - 步骤可在干净环境复现（仅依赖本仓库与同批次 Core/Server worktree）。
- 测试点（建议最小集合）：
  - connect/disconnect
  - login（或 register/login）
  - varstore list/get（任选其一，确保 Cmd→OK）
- 回滚点：
  - 文档变更独立提交；可 revert。

**步骤（Windows，本 PR 批次 worktree）**
1) 启动 HubServer（终端 1）：
   - `cd d:\project\MyFlowHub3\worktrees\hdrtcp-v2\MyFlowHub-Server`
   - `go run ./cmd/hub_server -addr 127.0.0.1:9000 -node-id 1`

2) 启动 Win App（终端 2）：
   - `cd d:\project\MyFlowHub3\worktrees\hdrtcp-v2\MyFlowHub-Win`
   - `wails dev`

3) App 内操作（Home/Auth）：
   - Address：`127.0.0.1:9000` → 点击 **Connect**
   - Device ID：填写任意非空字符串（例如 `smoke-1`）
   - 首次运行：点击 **Register**（NodeID=0），从响应/日志中记录分配到的 `node_id`
   - 随后：点击 **Login**（使用上一步的 `node_id`）
   - 期望：状态显示 Connected；Logs 中无明显 error

4) 触发至少一个业务 Cmd→OK（VarPool/VarStore）：
   - 进入 **VarPool** 页面，执行一次 **List**（或 **Get**）
   - 期望：收到 OK 响应（Frame/Logs 中能看到对应返回），且界面数据刷新

5) 退出验证：
   - App 点击 **Disconnect**（或关闭窗口）
   - 终端 1 Ctrl+C 停止 HubServer

### W4 - Code Review（阶段 3.3）与归档（阶段 4）
- 目标：完成 Review 清单并在本 worktree 根目录创建 `docs/change/2026-02-10_hdrtcp-v2.md`。
- 验收条件：
  - Review 逐项“通过/不通过”结论明确；不通过则回到阶段 3.2 修正。
  - 归档文档包含：背景/目标、具体变更、任务映射（W1-W3）、关键决策与权衡、测试结果、影响与回滚方案。

## 依赖关系
- 依赖同批次 Core/Server 的 v2 头部升级；任一端未升级将无法互通。

## 风险与注意事项
- go.mod 的 `replace` 路径需保持指向同批次 worktree（避免误链接到旧 header）。
- 这是 wire 破坏性变更：建议在本地联调通过后再推远端 PR。
