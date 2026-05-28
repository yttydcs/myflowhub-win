# 2026-05-28_win-mcp-remote-docker-auth

## 变更背景 / 目标

为 `myflowhub-mcp` 增加可 Docker 部署的远程 HTTP MCP 入口，支持多个 Codex 会话共享同一个远程 runtime，并用固定 bearer token 控制 endpoint 访问，避免公网上任意客户端直接调用 MCP。

## 具体变更内容

- `cmd/myflowhub-mcp/main.go`
  - 新增 `--allow-remote` 和 `--auth-token`。
  - 支持 `MYFLOWHUB_MCP_AUTH_TOKEN`。
  - 在 HTTP 远程模式下，缺少 token 时提前失败。
- `internal/mcp/http_server.go`
  - 新增远程启动配置和 token 校验。
  - 配置了 token 后，在读取 body 前校验 `Authorization: Bearer <token>`。
  - token 缺失或错误时返回 `401`。
- `internal/mcp/http_server_test.go`
  - 补齐本地 loopback、远程 token、缺 token、错 token、错 scheme、远程 Origin 的测试。
- `scripts/install-codex-myflowhub-mcp.ps1`
  - HTTP 配置支持 `-AuthToken` 和 `[mcp_servers.<name>.http_headers] Authorization = "Bearer ..."`。
  - 远程 URL 场景不再提示本机 `EnsureRunning`。
  - 改进同名 server block 的幂等替换。
- Docker 部署
  - 新增 `Dockerfile.mcp`、`docker/mcp-entrypoint.sh`、`docker-compose.mcp.yml`、`.dockerignore`、`.gitattributes`。
  - 容器默认走 HTTP remote 模式，要求 `MYFLOWHUB_MCP_AUTH_TOKEN`，并挂载 `/data`。
- 文档
  - 更新 `README.md`，补充远程 Docker 部署和 Codex `http_headers` 接入方式。
  - 更新 `docs/requirements/mcp-client.md`。
  - 更新 `docs/specs/mcp-client.md`。
  - 更新 `docs/change/README.md`。

## Related Plan

- `MCP-REMOTE-1`
- `MCP-REMOTE-2`
- `MCP-REMOTE-3`
- `MCP-REMOTE-4`

## Related Requirements

- `docs/requirements/mcp-client.md`

## Related Specs

- `docs/specs/mcp-client.md`

## Related Lessons

- none

## Searchable Lessons Summary

- 远程 MCP 访问使用固定 bearer token，格式为 `Authorization: Bearer <token>`。
- Docker 入口通过环境变量注入 `MYFLOWHUB_MCP_AUTH_TOKEN`，缺失时直接失败。
- 远程部署仍需要 HTTPS / VPN / 防火墙 / 反向代理，不要把 token 当成完整公网安全方案。

## Requirements Impact

updated

## Specs Impact

updated

## Lessons Impact

none

## 经验 / 教训摘要

- 远程 HTTP MCP 的安全边界应该放在 transport 层，不能混入工具 handler。
- 固定 token 足够满足当前多会话共享需求，不必引入签发、刷新、多 token 流程。
- Docker 入口要把 token 只当配置项，不要打印到日志或嵌入调试输出。

## 测试与验证

- `$env:GOWORK='off'; go test ./internal/mcp ./internal/mcpapp -count=1`
  - 通过。
- `$env:GOWORK='off'; go build -o $env:TEMP\\myflowhub-mcp-remote-auth-test.exe ./cmd/myflowhub-mcp`
  - 通过。
- `$env:GOWORK='off'; $env:GOOS='linux'; $env:GOARCH='amd64'; go build -o $env:TEMP\\myflowhub-mcp-linux-amd64-test ./cmd/myflowhub-mcp`
  - 通过。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\\scripts\\install-codex-myflowhub-mcp.ps1 -Transport http -Url https://example.com/mcp -AuthToken test-token -WhatIf`
  - 通过，输出远程 HTTP Codex 配置和 `http_headers.Authorization`。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\\scripts\\install-codex-myflowhub-mcp.ps1 -Transport http -Listen 127.0.0.1:17688 -McpPath /mcp -AuthToken local-token -WhatIf`
  - 通过，输出本机 HTTP 配置和 `EnsureRunning` 命令。
- 进程级 smoke：
  - 启动 HTTP remote server。
  - 未带 `Authorization` 请求返回 `401`。
  - 带 `Authorization: Bearer test-token` 请求返回 `200`。
  - 结果：通过。
- `git diff --check`
  - 通过。
- Docker build / compose
  - 未运行，因为当前环境未安装 `docker` 命令。

## 潜在影响与回滚方案

- 潜在影响：
  - 远程 HTTP 模式下，用户必须自己提供 token 和网络保护层。
  - Docker 入口现在要求 `MYFLOWHUB_MCP_AUTH_TOKEN`，缺失会直接退出。
- 回滚：
  - 撤回 `cmd/myflowhub-mcp/main.go`、`internal/mcp/http_server.go`、`internal/mcp/http_server_test.go`。
  - 撤回 `scripts/install-codex-myflowhub-mcp.ps1`、`README.md`、Docker 文件和文档更新。

## 子Agent执行轨迹

- none
