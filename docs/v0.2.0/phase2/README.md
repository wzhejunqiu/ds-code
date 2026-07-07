# ds-code v0.2.0 — phase2（M2 对齐增强）

> 版本：v0.2.0
> 阶段：phase2
> 状态：已实现
> 前置阶段：phase1
> 对应里程碑：M2 — 对齐增强
> 更新日期：2026-07-07

## 概述

phase2 在 MVP 基础上补齐 Inspector（Monaco diff）、Plan 模式、子代理分轨、命令面板、系统集成，以及 **Developer ID 签名 + Apple 公证**。

总纲：[spec/](../spec/)

## 文档索引

| 文档 | 说明 |
| ---- | ---- |
| [REQUIREMENTS.md](REQUIREMENTS.md) | FR-8/9/10/11 及 M2 分发/集成 |
| [DESIGN.md](DESIGN.md) | Inspector、子代理、命令面板 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | M2 gate |

## Gate

Inspector diff 可用；Plan/子代理/⌘K 可用；签名公证通过 Gatekeeper。

## 依赖

[phase1](../phase1/) MVP 已交付

## 下一阶段

[phase3](../phase3/) — checkpoint、MCP/LSP、托盘、自动更新
