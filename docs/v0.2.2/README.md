# ds-code v0.2.2 版本文档

> 版本：v0.2.2
> 状态：待实现
> 基线版本：v0.2.1
> 对应里程碑：M2 — 对齐增强
> 更新日期：2026-07-07

## 概述

v0.2.2 在 MVP 基础上补齐 Inspector（Monaco diff）、Plan 模式、子代理分轨、命令面板、系统集成，以及 **Developer ID 签名 + Apple 公证**。

总纲：[spec/](../v0.2.0/spec/)

## 文档索引

| 文档 | 说明 |
| ---- | ---- |
| [REQUIREMENTS.md](REQUIREMENTS.md) | FR-8/9/10/11 及 M2 分发/集成 |
| [DESIGN.md](DESIGN.md) | Inspector、子代理、命令面板 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | M2 gate |

## Gate

Inspector diff 可用；Plan/子代理/⌘K 可用；签名公证通过 Gatekeeper。

## 依赖

[v0.2.1](../v0.2.1/) MVP 已交付

## 下一版本

[v0.2.3](../v0.2.3/) — checkpoint、MCP/LSP、托盘、自动更新
