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
- [wails-binding-proto-drift.md](wails-binding-proto-drift.md)
  - 症状：`wails generate module` / `go test` 报 `undefined: flow.DetailReq`，或 `actionDetail redeclared in this block`，或 `replacement directory ../proto-stream-subproto does not exist`
  - 关键词：`GOWORK=off`、`flow.DetailReq`、`actionDetail redeclared`、`wails generate module`、`myflowhub-proto`、`reading ..\proto-stream-subproto\go.mod`
- [wails-embed-dist-placeholder.md](wails-embed-dist-placeholder.md)
  - 症状：`pattern all:frontend/dist: cannot embed directory frontend/dist: contains no embeddable files`
  - 关键词：`go:embed all:frontend/dist`、`go mod tidy`、`Generating bindings`
