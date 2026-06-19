# ds-code v0.1.1 版本文档

> 版本：v0.1.1  
> 状态：已实现  
> 基线版本：v0.1.0  
> 更新日期：2026-06-19

## 概述

v0.1.1 聚焦 **MCP 工具裸名注册** 与 **TUI 启动提示区**，解决 ds-code 与 Cursor/AGENTS.md 工具命名不一致导致的 `unknown tool` 问题，并在首屏 header **消息通知区** 展示启动期告警（换行可读、多条可滚动；含 MCP 冲突跳过与敏感日志警告）。

## 文档索引

| 文档 | 说明 |
|------|------|
| [REQUIREMENTS.md](REQUIREMENTS.md) | 功能与非功能需求、用户故事、范围边界 |
| [DESIGN.md](DESIGN.md) | 模块设计、数据流、接口变更、TUI 布局 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | 验收标准、测试要点、手动验证步骤 |

## 背景与动机

v0.1.0 将 MCP 工具注册为 `mcp__{server}__{tool}` 规范化名称，而项目 `AGENTS.md`、Cursor MCP 集成均使用 **裸名**（如 `semantic_search_nodes`）。Agent 按 AGENTS.md 调用 `tool_search` 时返回 `unknown tool`，MCP 能力形同虚设。

同时，`--allow-log-sensitive-data` 警告目前挤在 footer 下方，与 header 右侧大量空白未利用；MCP 跳过信息缺少用户可见通道。

## 变更摘要

| 领域 | v0.1.0 | v0.1.1 |
|------|--------|--------|
| MCP registry 名称 | `mcp__{server}__{tool}` | MCP 原始工具名（裸名） |
| 与内建工具冲突 | 无专门处理（可能静默覆盖） | 跳过 MCP 工具 + 提示 |
| 跨 MCP server 同名 | 启动失败（duplicate tool name） | 双方均不加载 + 提示，不阻断启动 |
| 单 MCP server 内同名 | 启动失败 | 第一条保留，其余跳过 + `in_server_duplicate` |
| 敏感日志警告 | footer 下方红色 banner | header **消息通知区**（换行 + 可滚动） |
| AGENTS.md | 需改工具名才能用 | **无需修改** |
| 历史 session | — | 新 MCP 调用须裸名；`mcp__*` 历史仅展示 |

## 实现注意

评审识别的风险与修复方向见 [REQUIREMENTS.md §7](REQUIREMENTS.md#7-已知风险与修复方向)、[DESIGN.md §10](DESIGN.md#10-已知风险与修复方向)。实现 PR 须对照 [ACCEPTANCE.md](ACCEPTANCE.md) §3.4–3.5、§5.3–5.4 验收。

## 依赖与前置

- 用户需在 `~/.ds-code/config/config.yaml` 或 `.ds-code/config.yaml` 配置 `mcp.servers`（与 v0.1.0 相同）
- MCP 子进程可用（如 `uvx code-review-graph serve`）
- 无新增 CLI  flag 或配置键（本版本不改 CONFIG schema）

## 关联文档

- 全局设计：[../v0.1.0/DESIGN.md](../v0.1.0/DESIGN.md) §13 MCP
- 配置说明：[../v0.1.0/CONFIG.md](../v0.1.0/CONFIG.md) §5.8 `mcp`
- 项目 Agent 指令：[../../AGENTS.md](../../AGENTS.md)
