# ds-code v0.2.4 版本文档

> 版本：v0.2.4
> 状态：待实现（可选）
> 基线版本：v0.2.3
> 对应里程碑：M4 — HTML 输出
> 更新日期：2026-07-07

## 概述

v0.2.4 为**可选**增量：安全 PoC 通过后启用助手 **HTML 输出模式**（默认仍为 Markdown）。

总纲：[spec/](../v0.2.0/spec/)

## 前置门禁

HTML 安全 PoC（OWASP XSS 全项）通过后方可启动。见总纲 [§14](../v0.2.0/spec/DESIGN.md#14-安全)。

## 文档索引

| 文档 | 说明 |
| ---- | ---- |
| [REQUIREMENTS.md](REQUIREMENTS.md) | FR-4.8 |
| [DESIGN.md](DESIGN.md) | DOMPurify、Shadow DOM、CSP |
| [ACCEPTANCE.md](ACCEPTANCE.md) | 安全 + 功能 gate |

## Gate

安全 PoC 全绿 + HTML 流式无卡顿 + 默认 Markdown 不变。

## 依赖

[v0.2.3](../v0.2.3/) 功能完备版已交付
