# v0.2.2 设计文档

> 版本：v0.2.2（M2 对齐增强）
> 状态：待实现
> 总纲：[spec/DESIGN](../v0.2.0/spec/DESIGN.md)
> 更新日期：2026-07-07

## 1. 本版定位

在 v0.2.1 三栏 MVP 上填满 Inspector，补齐 Plan/子代理/命令/集成与签名公证。

## 2. 实现焦点

| 主题 | 总纲章节 |
| ---- | -------- |
| Inspector Monaco diff | §6.1.2、§10 |
| Plan 模式 | §11 |
| 子代理 `streamId` 分轨 | §6.5、§8.3 |
| ⌘K 命令面板 | §6.1.6、§7.4 |
| 通知/Dock/拖拽 | §13 |
| 签名公证 | §15 |
| `internal/ui/port`（建议） | §2.2、§16 |

## 3. 不改动

Envelope v1、`workspaces.json` schema 不变。
