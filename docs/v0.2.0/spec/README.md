# ds-code 桌面产品线 — 设计总纲（spec）

> 位置：`docs/v0.2.0/spec/`
> 版本：v0.2.0-spec
> 状态：设计中（Design）
> 基线版本：v0.1.5
> 更新日期：2026-07-07
> 目标形态：**macOS 桌面应用**（Go + Wails v3 + TypeScript）

## 概述

本目录存放 **ds-code 桌面应用完整产品、交互与架构设计总纲**（原文保留），取代 [DESKTOP.md](../../DESKTOP.md) 的路线部分。

**实现**在 v0.2.0 内按 phase0~phase4 拆分（多 git 提交，不占用 v0.2.1~v0.2.4 版本号）：

| Phase | 里程碑 | 说明 |
| ----- | ------ | ---- |
| [../phase0/](../phase0/) | M0 | 技术验证 PoC — **第一个实现阶段** |
| [../phase1/](../phase1/) | M1 | 桌面 MVP |
| [../phase2/](../phase2/) | M2 | 对齐增强 |
| [../phase3/](../phase3/) | M3 | 深化能力 |
| [../phase4/](../phase4/) | M4 | HTML 输出（可选） |

路线图详见 [ROADMAP.md](ROADMAP.md)。

## 文档索引（总纲）

| 文档 | 说明 |
| ---- | ---- |
| [REQUIREMENTS.md](REQUIREMENTS.md) | 完整目标、用户故事、FR/NFR、范围边界 |
| [DESIGN.md](DESIGN.md) | 完整架构、UI、bridge、权限、分发 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | 总纲级 DOC/IMPL 验收与里程碑门禁 |
| [ROADMAP.md](ROADMAP.md) | v0.2.0 phase0~phase4 实现路线 |

## 六项核心设计立场

1. **macOS 优先且唯一**（≥ 12 Monterey）
2. **UI 全新设计，动线不照搬 TUI**（停止按钮，不用 ESC 取消 turn）
3. **modal-free 优先**（内联审批、设置视图、命令面板）
4. **Workspace + Agent 对话窗口**（单窗口多 project）
5. **三栏主界面**（左导航 / 中聊天 / 右 Inspector）
6. **数据与 CLI/TUI 隔离**（`~/.ds-code/desktop/projects/`，`ProjectID` 算法不变）

另：**Wails v3 + 内嵌 Chromium**；**React + Vite + TS + Bun**。

## 与 DESKTOP.md 的关系

[DESKTOP.md](../../DESKTOP.md) 为早期可行性研究（deprecated）；有效技术结论已吸收进本目录 [DESIGN.md](DESIGN.md)。

## 关联

- CLI 基线：[../../v0.1.5/](../../v0.1.5/)
- 实现入口：[../README.md](../README.md)（v0.2.0）
