# v0.1.1 需求文档

> 版本：v0.1.1  
> 状态：已实现  
> 更新日期：2026-06-19

## 1. 目标

1. MCP 工具在 ds-code tool registry 中以 **裸名**（MCP server 提供的原始 `tool.name`）注册，与 AGENTS.md / Cursor 文档一致。
2. 命名冲突时 **跳过加载、不阻断启动**，并向用户给出可见提示。
3. TUI 首屏 header **logo 右侧**新增统一启动提示区，收纳 MCP 跳过摘要与 `--allow-log-sensitive-data` 警告。

## 2. 用户故事

### US-1：按 AGENTS.md 使用 MCP 工具

**作为** 在本项目运行 ds-code 的开发者，  
**我希望** Agent 能直接调用 `semantic_search_nodes`、`get_architecture_overview` 等裸名工具，  
**以便** 无需记忆 `mcp__code-review-graph__` 前缀，AGENTS.md 与 ds-code 行为一致。

**验收**：配置 `code-review-graph` MCP 后，`tool_search({"tool_name":"semantic_search_nodes"})` 返回完整 schema；LLM 可直接以裸名调用 MCP 工具。

### US-2：内建工具优先

**作为** 用户，  
**我希望** 当 MCP 工具与内建工具同名时，内建工具始终生效、MCP 同名工具不被加载，  
**以便** 不会出现 `grep`/`read_file` 等被 MCP 意外覆盖的情况。

**验收**：MCP server 若暴露名为 `grep` 的工具，registry 中仅保留内建 `grep`；启动提示列出跳过项及原因 `builtin_conflict`。

### US-3：跨 server 同名安全

**作为** 用户，  
**我希望** 两个 MCP server 提供同名工具时，**两个 server 的该工具都不加载**，  
**以便** 避免不可预测的「先到先得」行为。

**验收**：server A 与 server B 均提供 `search` 时，registry 中无 `search`（MCP 来源）；两条跳过记录，原因 `cross_server_duplicate`；ds-code 正常启动。

### US-3b：单 server 内重复安全

**作为** 用户，  
**我希望** 同一 MCP server 的 `ListTools` 返回重复裸名时，不会启动失败或静默覆盖，  
**以便** 行为可预测。

**验收**：stub server 返回两个 `dup_tool` 时，registry 中至多一个 MCP `dup_tool`；第二条记 `in_server_duplicate`；ds-code 正常启动。

### US-4：启动期可见告警

**作为** TUI 用户，  
**我希望** 在首屏 header logo 右侧看到启动提示（MCP 跳过、敏感日志警告等），  
**以便** 一眼了解当前环境与风险，而不必翻 log 或 footer。

**验收**：`-vv --allow-log-sensitive-data` 时 header 右侧显示敏感日志警告；存在 MCP 跳过时显示摘要；窄终端下 notices 降级到 info 下方全宽显示。

## 3. 功能需求

### FR-1 MCP 裸名注册

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-1.1 | `adapterTool.Name()` 返回 MCP 原始工具名，不再返回 `mcp__{server}__{tool}` | P0 |
| FR-1.2 | `CallTool` 仍向 MCP server 传递原始工具名 | P0 |
| FR-1.3 | `Description()` 可保留 `[MCP {server}]` 前缀，便于 LLM 区分来源 | P1 |
| FR-1.4 | `tool_search`、LLM `tools` 列表、Execute 均按裸名工作 | P0 |

### FR-2 冲突检测与跳过

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-2.1 | **内建冲突**：registry 中已存在同名工具（含 `tool_search`、`agent` 等）→ 跳过该 MCP 工具 | P0 |
| FR-2.2 | **跨 server 重复**：同一裸名出现在 ≥2 个 server → **所有** 提供该名的 server 均不加载该工具 | P0 |
| FR-2.3 | 跳过不导致 `NewManagerFromConfig` / `DiscoverTools` 失败 | P0 |
| FR-2.4 | **server 名重复**（配置错误）仍启动失败，行为与 v0.1.0 一致 | P0 |
| FR-2.5 | 暴露 `Manager.SkippedTools()` 供 app/TUI 消费 | P0 |
| FR-2.6 | 每条跳过写入 project log（`Warn` 级别）；**须在 `buildTools` / `MCP.Register` 完成之后**输出（见 [DESIGN.md §5](DESIGN.md#5-app-层接线)、[§10.4](DESIGN.md#104-跳过日志时机)） | P1 |
| FR-2.7 | **单 server 内重复**：同一 `ListTools` 响应中同名 `tool.name` 出现 ≥2 次 → 该 server 的该名**均不加载**，记 `in_server_duplicate`，不阻断启动（见 [DESIGN.md §4.3](DESIGN.md#43-discovertools-两遍算法)） | P0 |
| FR-2.8 | `ToolCount()` / 启动日志中的 MCP 工具数反映**最终进入 registry 的数量**，不含已跳过项（见 [DESIGN.md §4.9](DESIGN.md#49-toolcount-语义)） | P1 |

跳过原因枚举：

| Reason | 含义 |
|--------|------|
| `builtin_conflict` | 与已注册内建工具同名（在 `Register` 阶段判定） |
| `cross_server_duplicate` | 多个 MCP server 提供同名裸名 |
| `in_server_duplicate` | 同一 MCP server 的 `ListTools` 返回重复裸名 |

### FR-3 权限与展示适配

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-3.1 | `Manager.IsWriteTool(name)` **仅对已成功注册的 MCP 裸名**返回 true：`byName` 命中 adapter 时按 adapter 权限级 + `isWriteMCPToolName(mcpTool)` 判定；**不得**对任意裸名做全局启发式回退（避免误判内建工具，见 [DESIGN.md §4.6](DESIGN.md#46-权限写工具检测)、[§10.1](DESIGN.md#101-iswritetool-裸名回退)） | P0 |
| FR-3.2 | TUI 工具行展示 `MCP {server} · {tool}`；经 `registry.MCPServerForTool` 解析 server，**展示层须能访问 Registry**（见 [DESIGN.md §4.8](DESIGN.md#48-registry-mcp-元数据tui-展示)） | P1 |
| FR-3.3 | `tools.defer_mcp` 行为不变：defer 的是裸名 MCP 工具 | P0 |
| FR-3.4 | ask 模式下裸名 MCP 写工具的参数摘要：`formatArgsSummary` 通过 `registry.IsMCPTool` 识别，不再依赖 `mcp__` 前缀（见 [DESIGN.md §4.6](DESIGN.md#46-权限写工具检测)） | P0 |

### FR-4 TUI Header 消息通知区

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-4.1 | header **双栏**：左身份区（logo/版本/模型/路径）；右 **消息通知区** 承载启动告警 | P0 |
| FR-4.2 | Notice 支持 `warn` / `info`；每条为独立通知块，块间空行 | P0 |
| FR-4.3 | `--allow-log-sensitive-data`（且 `-vv`）警告迁入通知区，**不再**占用 footer banner | P0 |
| FR-4.4 | MCP 跳过摘要列出全部 `tool@server` 明细（组装不截断条数） | P0 |
| FR-4.5 | 通知文案 **自动换行**（列宽内 UTF-8 安全折行），**禁止**单行字节截断/乱码 | P0 |
| FR-4.6 | 换行后总行数 > 8 时，通知区 **自动竖向滚动**（约 4s 一步，循环）；底部显示 `通知 x–y / n` | P1 |
| FR-4.7 | 窄屏（`< 72` 列）通知区移至身份区下方、全宽换行展示 | P1 |
| FR-4.8 | TUI 启动前 stderr 敏感日志警告保留（`MaybeWarnSensitiveLog`） | P1 |

## 4. 非功能需求

| ID | 描述 |
|----|------|
| NFR-1 | 无 MCP 配置时行为与 v0.1.0 一致（0 个 MCP 工具，无 notices） |
| NFR-2 | 冲突跳过逻辑可单元测试，不依赖真实 MCP 子进程 |
| NFR-3 | header 宽度 fuzz 测试不 panic（延续 `header_width_test` 惯例） |
| NFR-4 | 不新增必填配置项；不破坏现有 `.ds-code/config.yaml` MCP 配置格式 |
| NFR-5 | 非 TUI（`-p` 等非交互）无 header notices；MCP 跳过信息通过 project log（`mcp tool skipped`）可见，与 v0.1.0 日志渠道一致 |

## 5. 范围外（Non-Goals）

- 不实现 Cursor `.cursor/mcp.json` 自动导入
- 不为 MCP 裸名提供 `mcp__` 前缀别名（彻底移除规范化名作为主键，不做双注册）
- 不改动 AGENTS.md 工具名写法
- 不在本版本增加 `/mcp` slash 命令或 MCP 热重载
- 不改动 Plan 模式工具集（Plan 仍不注册 MCP）

## 6. 兼容性说明

### 破坏性变更

对 **依赖 `mcp__{server}__{tool}` 名称** 的外部脚本、测试或用户习惯为破坏性变更。v0.1.1 起 LLM 与 `tool_search` 仅认裸名。

### 历史 Session（resume）

v0.1.0 持久化的 assistant `tool_calls` 可能含 `mcp__*` 工具名；升级后 registry 仅认裸名。

| 场景 | v0.1.1 预期 |
|------|-------------|
| Resume 旧 session、继续对话 | 新 tool call 须用裸名；历史中 `mcp__*` 名称仅作上下文展示，**不可重放执行** |
| 旧 session TUI 历史块 | 仍可按 `mcp__` 前缀展示（`display.go` 保留只读解析，见 [DESIGN.md §4.8](DESIGN.md#48-registry-mcp-元数据tui-展示)） |
| 推荐做法 | 升级后 MCP 相关任务**新建 session**；或在 release note 中说明旧 session 限制 |

本版本**不做** `mcp__` 别名双注册（见 Non-Goals）。

### 文档需同步（实现阶段）

- [../v0.1.0/CONFIG.md](../v0.1.0/CONFIG.md)：`defer_mcp`、MCP 工具名描述
- [../v0.1.0/DESIGN.md](../v0.1.0/DESIGN.md) §13：命名规范
- [../../CLAUDE.md](../../CLAUDE.md)、[../../internal/tool/builtin/README.md](../../internal/tool/builtin/README.md)

### 无需修改

- [../../AGENTS.md](../../AGENTS.md)
- 用户 MCP YAML 配置结构

## 7. 已知风险与修复方向

实现前须对照 [DESIGN.md §10](DESIGN.md#10-已知风险与修复方向) 逐项落实。

| 风险 | 修复方向（摘要） |
|------|------------------|
| `IsWriteTool` 裸名全局启发式误判内建工具 | 仅 `byName` 命中 MCP adapter 时判定；未注册名返回 false |
| TUI `DisplaySummary` 无 Registry，裸名 MCP 无法展示 server | `Runner` 持有 Registry；展示函数增加 `MCPLookup` 或 `*Registry` 参数 |
| 单 server `ListTools` 重名未定义 | Pass 1 统计 + Pass 2 跳过，原因 `in_server_duplicate` |
| `SkippedTools` 在 `ensureMCP` 后日志，漏记 `builtin_conflict` | 日志与 `StartupNotices` 均在 `buildTools` 完成之后组装 |
| `ToolCount` 含已跳过工具 | 改为 `RegisteredToolCount` 或 Register 后维护计数 |
| `ReservedNames` 预检误伤未注册内建名（如 `web_search`） | Discover **禁止**用 `ReservedNames` 预跳过；仅以 `Register` 时 `reg.Get` 为准 |
| Resume v0.1.0 session 工具名不一致 | 文档说明 + 历史展示保留 `mcp__` 解析；新调用仅裸名 |
| 非 TUI 无 header 跳过提示 | 依赖 project log；见 NFR-5 |
