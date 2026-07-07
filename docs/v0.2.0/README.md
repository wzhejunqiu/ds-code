# ds-code v0.2.0 版本文档

> 版本：v0.2.0
> 状态：待实现
> 基线版本：v0.1.5
> 更新日期：2026-07-07

## 概述

v0.2.0 是 **ds-code 桌面产品线**的完整交付版本。实现按 **phase0~phase4** 五个阶段拆分，对应多个 git 提交；**phase 不占用独立 semver**（无 v0.2.1~v0.2.4）。

完整产品/架构设计见总纲 [spec/](spec/)；各 phase 文档为 scoped 实现切片。

## Phase 索引

| Phase | 里程碑 | 主题 | Gate |
| ----- | ------ | ---- | ---- |
| [phase0/](phase0/) | M0 | 技术验证 PoC | 单 workspace：流式 + 审批 + 停止 |
| [phase1/](phase1/) | M1 | 桌面 MVP | 多 workspace + 三栏 + P0 + 基础打包 |
| [phase2/](phase2/) | M2 | 对齐增强 | Inspector + Plan + 子代理 + 公证 |
| [phase3/](phase3/) | M3 | 深化 | checkpoint + MCP/LSP + 托盘 + 更新 |
| [phase4/](phase4/) | M4（可选） | HTML 输出 | 安全 PoC 通过后启用 |

```mermaid
flowchart LR
  spec["spec/ 总纲"]
  p0["phase0 M0"]
  p1["phase1 M1"]
  p2["phase2 M2"]
  p3["phase3 M3"]
  p4["phase4 M4"]
  release["v0.2.0 发布"]
  spec -.->|指导| p0
  p0 --> p1 --> p2 --> p3 --> p4
  p4 --> release
```

## 文档索引

| 文档 | 说明 |
| ---- | ---- |
| [spec/](spec/) | **完整产品/架构总纲**（REQUIREMENTS / DESIGN / ACCEPTANCE / ROADMAP） |
| [spec/ROADMAP.md](spec/ROADMAP.md) | phase0~phase4 实现路线与拆分原则 |
| [phase0/](phase0/) ~ [phase4/](phase4/) | 各阶段 scoped 需求 / 设计 / 验收 |

## v0.2.0 发布 Gate

- **必过**：phase0 ~ phase3 全部 gate + 回归
- **可选**：phase4（HTML 输出，安全 PoC 门禁后）

## 依赖

- 总纲：[spec/](spec/)
- CLI 基线：v0.1.5+（`permission`/`config` 契约稳定）
