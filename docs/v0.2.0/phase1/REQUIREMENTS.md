# v0.2.0 phase1 需求文档

> 版本：v0.2.0
> 阶段：phase1（M1 MVP）
> 状态：待实现
> 前置阶段：phase0　总纲：[spec/REQUIREMENTS](../spec/REQUIREMENTS.md)
> 更新日期：2026-07-07

## 1. 目标

交付桌面 MVP，覆盖总纲 **P0** 功能，达到与 CLI/TUI 核心能力对齐的可日常使用状态。

## 2. 本阶段功能需求

### 新增于 phase1（相对 phase0）

#### FR-0 应用形态（补齐）

| ID | 描述 | 优先级 |
| -- | ---- | ------ |
| FR-0.2 | macOS ≥ 12 universal 二进制 | P0 |
| FR-0.5 | 同 repo 共享 `internal/*` | P0 |

#### FR-1 Workspace 管理（完整 P0/P1）

| ID | 描述 | 优先级 |
| -- | ---- | ------ |
| FR-1.1–FR-1.9 | 多 workspace、`WorkspaceManager`、注册表、数据隔离 | P0 |
| FR-1.10 | 懒初始化；切换不销毁后台 App | P1 |
| FR-1.11 | 同路径去重 | P1 |

#### FR-2 ~ FR-18

见总纲对应 FR；本阶段覆盖 FR-2/3/4/5/6/7/14/15/17 的 P0 项，及 FR-16.1/16.6、FR-18.1/18.5。

> `apply_patch` diff 点击 → phase2 Inspector；签名公证 → phase2

## 3. 明确不包含（phase2+）

- Inspector Monaco diff（FR-5.4、FR-8）→ phase2
- Plan 模式、子代理、命令面板 → phase2
- 签名公证（FR-18.2/18.3）→ phase2
- checkpoint、MCP/LSP 管理、托盘、自动更新 → phase3
- HTML 输出 → phase4

## 4. Gate

总纲 FR-0/1/2/3/4/5/6/7/14/15/17 的 **P0** 全部通过；REG-1~4 绿。
