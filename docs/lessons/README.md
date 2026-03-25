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

- [wails-embed-dist-placeholder.md](wails-embed-dist-placeholder.md)
  - 症状：`pattern all:frontend/dist: cannot embed directory frontend/dist: contains no embeddable files`
  - 关键词：`go:embed all:frontend/dist`、`go mod tidy`、`Generating bindings`
