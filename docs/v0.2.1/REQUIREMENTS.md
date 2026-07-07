# v0.2.1 需求文档

> 版本：v0.2.1（M1 MVP）
> 状态：待实现
> 前置：[v0.2.0](../v0.2.0/)　总纲：[spec/REQUIREMENTS](../v0.2.0/spec/REQUIREMENTS.md)
> 更新日期：2026-07-07

## 1. 目标

交付桌面 MVP，覆盖总纲 **P0** 功能，达到与 CLI/TUI 核心能力对齐的可日常使用状态。

## 2. 本版功能需求

### 新增于 v0.2.1（相对 v0.2.0）

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

见总纲对应 FR；本版覆盖 FR-2/3/4/5/6/7/14/15/17 的 P0 项，及 FR-16.1/16.6、FR-18.1/18.5。

> `apply_patch` diff 点击 → v0.2.2 Inspector；签名公证 → v0.2.2

## 3. 明确不包含（v0.2.2+）

- Inspector Monaco diff（FR-5.4、FR-8）→ v0.2.2
- Plan 模式、子代理、命令面板 → v0.2.2
- 签名公证（FR-18.2/18.3）→ v0.2.2
- checkpoint、MCP/LSP 管理、托盘、自动更新 → v0.2.3
- HTML 输出 → v0.2.4

## 4. Gate

总纲 FR-0/1/2/3/4/5/6/7/14/15/17 的 **P0** 全部通过；REG-1~4 绿。
