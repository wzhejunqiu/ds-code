# ds-code v0.2.0 — phase1（M1 MVP）

> 版本：v0.2.0
> 阶段：phase1
> 状态：待实现
> 前置阶段：phase0
> 对应里程碑：M1 — 桌面 MVP
> 更新日期：2026-07-07

## 概述

phase1 交付**可日常使用的桌面 MVP**：在 phase0 技术验证基础上，补齐总纲中所有 **P0 功能**——多 workspace、三栏 UI、Agent 对话窗口、工具卡片、完整权限审批、设置、onboarding 与基础打包。

完整设计见总纲 [spec/](../spec/)。

## 文档索引

| 文档 | 说明 |
| ---- | ---- |
| [REQUIREMENTS.md](REQUIREMENTS.md) | 本阶段 scoped FR（P0） |
| [DESIGN.md](DESIGN.md) | WorkspaceManager、三栏 UI、本阶段增量要点 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | M1 gate 验收 |

## Gate

FR-0/1/2/3/4/5/6/7/14/15/17 全部 P0 绿；产出 universal `.app` + `.dmg`（未公证）。

## 依赖

- [phase0](../phase0/) 已交付（bridge、datadir、Chromium PoC、工具链）
- 总纲：[spec/](../spec/)

## 下一阶段

[phase2](../phase2/) — Inspector、Plan、子代理、命令面板、签名公证
