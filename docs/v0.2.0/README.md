# ds-code v0.2.0 版本文档

> 版本：v0.2.0
> 状态：待实现
> 基线版本：v0.1.5
> 对应里程碑：M0 — 技术验证 PoC
> 更新日期：2026-07-07

## 概述

v0.2.0 是桌面产品线的**第一个可运行版本**，目标是验证技术栈可行性与核心桥接契约，**不追求完整产品功能**。

本版交付：

- Wails v3 工程骨架 + 内嵌 Chromium 渲染 PoC
- Bun + Vite + React 前端工具链 PoC
- `desktop/bridge` StreamEmitter + Envelope v1 协议
- `desktop/datadir` 桌面数据路径隔离
- `desktop/permission` 最小 DesktopPrompter
- **单 workspace** 下跑通一轮 `RunTurn`：流式聊天 + 一次写审批 + 停止取消

完整产品/架构设计见总纲 [spec/](spec/)；本版仅实现其中 M0 切片。

## 文档索引

| 文档 | 说明 |
| ---- | ---- |
| [REQUIREMENTS.md](REQUIREMENTS.md) | 本版 scoped 功能需求（引用总纲 FR 编号） |
| [DESIGN.md](DESIGN.md) | 本版实现焦点与增量设计要点 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | 本版 gate 验收标准 |
| [spec/](spec/) | **完整产品/架构总纲**（REQUIREMENTS / DESIGN / ACCEPTANCE / ROADMAP） |

## 范围边界

**In scope**

- `cmd/ds-code-desktop` 最小入口（单窗口、无菜单栏完善）
- `desktop/bridge`、`desktop/datadir`、`desktop/permission`
- 最小聊天 UI（无三栏、无 workspace 侧栏）
- 单 `ProjectRoot` 硬编码或启动参数指定

**Out of scope（留给 v0.2.1+）**

- 多 workspace / `WorkspaceManager` / `workspaces.json`
- 三栏布局、Inspector、设置视图、onboarding
- 工具卡片完整渲染（可仅文本占位）
- 签名公证、系统集成

## Gate

能打开一个项目 → 发消息 → 看流式回复 → 触发一次写操作并内联批准 → 点击停止取消 turn。

## 依赖

- 总纲：[spec/](spec/)
- CLI 基线：v0.1.5+（`permission`/`config` 契约稳定）

## 下一版本

[v0.2.1](../v0.2.1/) — M1 桌面 MVP（多 workspace + 三栏 + 完整 P0 功能）
