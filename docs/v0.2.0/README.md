# ds-code v0.2.0 版本文档

> 版本：v0.2.0
> 状态：设计中（Design）
> 基线版本：v0.1.5
> 更新日期：2026-07-04
> 目标形态：**macOS 桌面应用**（Go + Wails v3 + TypeScript）

## 概述

v0.2.0 定义 **ds-code 桌面应用**的完整产品、交互与架构设计。桌面版复用现有 Go 核心（`internal/*`），以 **Wails v3** 作为桌面壳，前端用 TypeScript 重写 UI。本版是 **权威产品设计文档**，取代 [DESKTOP.md](../DESKTOP.md) 的路线部分（后者已标记 deprecated，仅保留为历史可行性研究背景）。

六项核心设计立场（本版拍板，含后续修订）：

1. **macOS 优先且唯一**：前期只做 macOS（≥ 12 Monterey）；Windows/Linux 不在本版范围。遵循 macOS 桌面应用设计哲学（原生窗口、菜单栏、系统集成、克制的视觉与动效）。
2. **UI 全新设计，动线不照搬 TUI**：桌面端不是 TUI 的换皮。例如 **不以 ESC 取消 turn**、不以键盘序列驱动一切；改用可见控件、指针交互与 macOS 习惯（菜单栏、快捷键、拖拽）。
3. **几乎不用弹窗（modal-free 优先）**：权限审批、设置、命令、onboarding 等**尽量内联**到主界面（对话流内的审批卡片、独立设置视图、命令面板），避免打断式弹窗；仅系统级不可避免场景（如原生文件选择器）使用系统面板。
4. **Workspace 与 Agent 对话窗口**：一个 **workspace 关联一个 project**（`ProjectRoot`）；其下可创建**多个 Agent 对话窗口**（对应 TUI 的 session），统一收纳在 workspace 侧栏下。单窗口 + 左侧栏管理多个 workspace。
5. **三栏主界面**：左（workspace / Agent 对话窗口导航）+ 中（聊天主区）+ 右（上下文检查器：文件 / diff / 工具详情，可折叠）。
6. **数据与 CLI/TUI 隔离**：`ProjectID` 算法**保持不变**（`hex(SHA256(ProjectRoot))`）；桌面通过**独立数据目录** `~/.ds-code/desktop/projects/<project-id>/` 与 CLI/TUI 的 `~/.ds-code/projects/<project-id>/` 区隔，**不共享**会话、checkpoint 等运行时数据（同一 `project-id`，父目录不同）。
7. **Wails v3 + 内嵌浏览器内核**：桌面壳用 Wails v3；渲染采用**内嵌固定版本浏览器内核**（bundled Chromium），避免依赖 macOS 系统 WKWebView 版本差异。
8. **前端 React + Vite + TypeScript + Bun**：UI 技术栈已定稿；工具链 Bun（包管理/脚本）+ Vite（dev/build）；组件库 shadcn/ui + Tailwind。

**交付范围**：本版仅交付**文档**（REQUIREMENTS / DESIGN / ACCEPTANCE），面向 CLI/TUI 功能对齐做完整设计；实现拆分到后续多个小版本（见 [REQUIREMENTS.md §7](REQUIREMENTS.md#7-分期实施路线) 分期）。

## 文档索引

| 文档                               | 说明                                                                                     |
| ---------------------------------- | ---------------------------------------------------------------------------------------- |
| [REQUIREMENTS.md](REQUIREMENTS.md) | 目标、用户故事、功能/非功能需求、范围边界、分期实施路线                                   |
| [DESIGN.md](DESIGN.md)             | 应用架构、窗口/导航模型、Workspace 管理、无弹窗策略、bridge/流式协议、系统集成、分发打包 |
| [ACCEPTANCE.md](ACCEPTANCE.md)     | 验收标准、手动验证步骤、测试清单                                                         |

## 背景与动机

### 为什么现在做桌面版

| 依据                | 说明                                                                                                  |
| ------------------- | ----------------------------------------------------------------------------------------------------- |
| 核心已与 UI 解耦    | `internal/agent` 不依赖 `internal/ui`；[`TurnCallbacks`](../../internal/agent/callbacks.go) 即事件契约 |
| 后端复用率高        | `agent`、`context`、`session`、`tool`、`permission`、`mcp`、`lsp` 等可直接复用                        |
| TUI 交互对新用户不友好 | 桌面用户更少熟悉终端；图形化 diff、文件树、拖拽、通知等 TUI 难以提供                                 |
| 一个 App = 一个 project 太受限 | CLI/TUI 每次进程绑定单个 `ProjectRoot`；开发者常同时在多个仓库工作，需要 workspace 级管理     |

### 与 DESKTOP.md 的关系

[DESKTOP.md](../DESKTOP.md) 是早期可行性研究（v0.4），其**分阶段路线（Phase 0-3）已被本版取代**并标记 deprecated。其中仍然有效的**技术结论**（流式 IPC 双层 batch、Envelope 协议、HTML 双模式安全分析）已被本版 [DESIGN.md](DESIGN.md) 吸收并落为具体设计；细节论证可回溯该历史文档。

## 术语

| 术语              | 含义                                                                                                    |
| ----------------- | ------------------------------------------------------------------------------------------------------- |
| **Workspace**     | 桌面版对一个 project 的封装，绑定一个 `ProjectRoot`；一个 workspace 对应一个后端 `app.App` 实例          |
| **Agent 对话窗口** | 一个 workspace 下的独立 agent 聊天上下文（**对应 TUI 的 session**）；可新建多个、resume；UI 上作为 workspace 子项展示 |
| **Session**       | 后端领域模型术语，与「Agent 对话窗口」一一对应；持久化在桌面独立数据目录的 `sessions.db`                  |
| **project-id**    | `hex(SHA256(ProjectRoot))`（`datadir.ProjectID`，**算法与 CLI 完全一致**）；桌面与 CLI 靠**数据目录前缀**区隔 |
| **Bridge**        | Go 侧 `TurnCallbacks` → Wails Events 的适配层（`desktop/bridge`）                                        |
| **Inspector**     | 右栏上下文检查器：文件预览 / diff / 工具调用详情                                                         |
| **审批卡片**      | 对话流内联的权限审批 UI（替代弹窗），对应后端 `permission.Prompter` / `WebFetchPrompter`                 |
| **命令面板**      | ⌘K 唤起的命令入口，承载 slash 命令与全局操作（替代命令弹窗）                                             |

## 功能地图（面向 CLI/TUI 对齐，分期落地）

| 领域              | 桌面呈现                                    | 复用后端                                        | 目标期 |
| ----------------- | ------------------------------------------- | ----------------------------------------------- | ------ |
| Workspace 管理    | 左侧栏 + 注册表持久化 + 拖拽/打开文件夹     | `app.App` × N、`datadir.ProjectID`              | M1     |
| Agent 对话窗口    | workspace 下多窗口：新建 / resume / 重命名  | `session.Store`（桌面独立 SQLite）              | M1     |
| 聊天 + 流式渲染   | 三栏中区 Markdown 流式                      | `TurnCallbacks` + bridge Envelope v1            | M1     |
| 工具卡片          | read/grep/bash/apply_patch 等卡片           | `TurnCallbacks.OnTool*`                         | M1     |
| 权限审批          | 对话流内联审批卡片（无弹窗）                | `DesktopPrompter` / `WebFetchPrompter`          | M1     |
| 停止 turn         | 可见「停止」按钮（无 ESC 动线）             | `context.Cancel`                                | M1     |
| 设置              | 独立设置视图（非弹窗）                      | `internal/config`                               | M1     |
| Onboarding        | 首启引导：API Key 环境变量指引 / 权限模式 / 打开 workspace | `config.LoadAPIKey`、设置视图                   | M1     |
| 上下文检查器      | 右栏文件/diff/工具详情，内嵌 Monaco         | `apply_patch`、checkpoint FileState             | M2     |
| Plan 模式         | 模式切换 + 只读工具集提示                   | `runmode` / Plan 工具裁剪                       | M2     |
| 子代理面板        | 侧栏 tab / 内嵌卡片，`streamId` 分轨        | `TurnCallbacks.OnSubagent*`、spawn 服务         | M2     |
| slash / 命令面板  | ⌘K 命令面板 + 输入框 `/` 补全              | `slashcmd`                                      | M2     |
| checkpoint/rewind | 时间线 rewind 视图                          | `internal/checkpoint`                           | M3     |
| MCP / LSP 管理    | 设置内的服务器状态与配置视图                | `internal/mcp`、`internal/lsp`                  | M3     |
| Token / billing   | 状态栏用量 + 会话统计                       | `OnUsageUpdate`、`internal/billing`             | M3     |
| 系统集成          | 通知 / Dock badge / 菜单栏 / 托盘 / 拖拽    | Wails v3 runtime                                | M2-M3  |
| HTML 输出模式     | 助手回复富文本（可选，默认 Markdown）       | output overlay + DOMPurify（安全 PoC 门禁）     | M4+    |

## 已知限制（本版）

| 限制             | 说明                                                                             |
| ---------------- | -------------------------------------------------------------------------------- |
| 仅 macOS         | Windows/Linux 打包与差异不在范围；代码结构预留跨平台可能，但不承诺               |
| 仅文档交付       | 本版不含实现代码；实现按 [§7 分期](REQUIREMENTS.md#7-分期实施路线) 落到后续小版本 |
| HTML 输出后置    | 默认 Markdown；HTML 富文本模式需独立安全 PoC，排在 M4+                            |
| 数据不互通       | 桌面与 CLI/TUI **不共享** sessions/checkpoint/audit 等运行时数据；同一路径下各自独立库 |
| CGO tokenizer    | 桌面默认纯 Go 字符估算；「精确计数」为设置开关，打包 CGO 细节后置                 |
| 内嵌浏览器体积   | bundled Chromium 使安装包大于纯 WKWebView 方案；体积目标见 NFR-8（< 200MB）     |
| 多窗口拆分       | 单窗口 + 侧栏为默认；workspace「拆出独立窗口」列为后期增强                        |

## 依赖与前置

- 基线：v0.1.5 已发布或合入 main（`permission`/`config`/`tracing` 契约稳定）
- 新增技术栈：Wails v3、**React + Vite + TypeScript + Bun ≥ 1.3**、内嵌 Chromium 内核（见 [DESIGN.md §3.4](DESIGN.md#34-内嵌浏览器内核)、[§12](DESIGN.md#12-前端技术栈)、[§12.5 M0 脚手架](DESIGN.md#125-m0-脚手架模板)）
- 后端复用：`cmd/ds-code/app`、`internal/agent`、`internal/permission`、`internal/session` 等**不改核心语义**，仅新增桌面适配层

## 关联文档

- 上一版本：[../v0.1.5/README.md](../v0.1.5/README.md)
- 历史可行性研究（deprecated）：[../DESKTOP.md](../DESKTOP.md)
- 安全基线：[../v0.1.0/SECURITY.md](../v0.1.0/SECURITY.md)
- 系统设计基线：[../v0.1.0/DESIGN.md](../v0.1.0/DESIGN.md)
