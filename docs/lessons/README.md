# Lessons

## Purpose

存放可复用的事故复盘、踩坑总结和预防规则。

## How To Enter This Section

- 当一次问题具备跨 workflow 复用价值时进入这里。

## What Belongs Here

- 根因分析
- 触发条件
- 预防措施

## Naming / Maintenance Rules

- 优先使用稳定名称
- 避免把一次性变更记录误写成 lessons

## Current Docs

- [authority-local-admin-actions.md](authority-local-admin-actions.md)
  - 症状：remote authority 下审批 / permit 管理仍 timeout、仍提示 `requires authority-local session`，或 authority 拒绝 routed source
  - 关键词：`authorityId != sourceId`、`routed source`、`requires authority-local session`、`list_register_permits`
- [frontend-build-babel-parser-missing.md](frontend-build-babel-parser-missing.md)
  - 症状：`failed to load config from frontend/vite.config.ts`、`Cannot find module '@babel/parser'`
  - 关键词：`@babel/parser`、`@vue/compiler-core`、`Compiling frontend`、`Cannot find module`
- [frontend-build-empty-node-modules.md](frontend-build-empty-node-modules.md)
  - 症状：`Installing frontend dependencies: Done.` 后仍报 `Cannot find module '...vite/bin/vite.js'`
  - 关键词：`vite/bin/vite.js`、`empty node_modules`、`package.json.md5`、`wails build`
- [flow-body-spec-mode-normalization.md](flow-body-spec-mode-normalization.md)
  - 症状：`foreach.body` 内高级节点明明已支持表单，但窗口里仍只显示 `Advanced JSON`，或找不到 `Add Binding`
  - 关键词：`normalizeBodySessionSnapshot`、`createBodyNodeDraft`、`specEditorMode`、`foreach.body`
- [stream-ctrl-await-mismatch.md](stream-ctrl-await-mismatch.md)
  - 症状：Stream 页面创建本地 source / consumer 超时，日志报 `stream announce: request timed out`
  - 关键词：`KindCtrl`、`SubProtoStream`、`announce_resp`、`SendCommandAndAwait`、`payload[0]`
- [stream-local-owner-ctrl-gap.md](stream-local-owner-ctrl-gap.md)
  - 症状：`stream announce` / `announce_consumer` 仍然 timeout，且服务端已把 owner 请求路由回本机 Win
  - 关键词：`routeOwnerRequest`、`delivery_prepare`、`MajorCmd`、`ACK`、`stream local owner`
- [wails-binding-proto-drift.md](wails-binding-proto-drift.md)
  - 症状：`wails generate module` / `go test` 报 `undefined: flow.DetailReq`，或 `actionDetail redeclared in this block`，或 `replacement directory ../proto-stream-subproto does not exist`
  - 关键词：`GOWORK=off`、`flow.DetailReq`、`actionDetail redeclared`、`wails generate module`、`myflowhub-proto`、`reading ..\proto-stream-subproto\go.mod`
- [wails-embed-dist-placeholder.md](wails-embed-dist-placeholder.md)
  - 症状：`pattern all:frontend/dist: cannot embed directory frontend/dist: contains no embeddable files`
  - 关键词：`go:embed all:frontend/dist`、`go mod tidy`、`Generating bindings`
