# v0.1.1 验收标准

> 版本：v0.1.1  
> 状态：已实现  
> 更新日期：2026-06-19  
> 需求：[REQUIREMENTS.md](REQUIREMENTS.md) · 设计：[DESIGN.md](DESIGN.md)

## 1. 总体验收

- [ ] 版本号标记为 v0.1.1（release/tag 由发布流程完成）
- [ ] `make test` 通过
- [ ] `make lint` / `make vet` 无新增失败
- [x] 文档 [CONFIG.md](../v0.1.0/CONFIG.md)、[DESIGN.md](../v0.1.0/DESIGN.md) 已同步 MCP 裸名描述
- [x] [DESIGN.md §10](DESIGN.md#10-已知风险与修复方向) 所列风险均有对应实现与测试（见 §6）

## 2. MCP 裸名注册

### AC-2.1 正常连通

**前置**：`.ds-code/config.yaml` 配置 `code-review-graph`，`uvx` 可用。

| 步骤 | 预期 |
|------|------|
| 启动 `bin/ds-code -vv` | 日志含 `mcp_servers: 1`、`MCP ready`、`mcp discover tools` 且 `tool_count > 0` |
| LLM 请求中 `tools` 数量 | 大于 10（内建 + MCP 裸名） |
| `tool_search({"tool_name":"semantic_search_nodes"})` | 返回完整 Schema，非 `unknown tool` |
| 直接调用 `semantic_search_nodes` | 执行成功并返回图谱结果 |

### AC-2.2 无 MCP 配置

| 步骤 | 预期 |
|------|------|
| 移除或清空 `mcp.servers` | 日志 `MCP disabled (no servers configured)` |
| `tools` 数量 | 与 v0.1.0 agent 模式内建集相同（默认配置约 10：含 `lsp.enabled: true` 时的 `diagnostics`；关闭 LSP 或无 `web_fetch` 时数量相应减少） |
| 行为 | 与 v0.1.0 一致 |

### AC-2.3 不再使用 mcp__ 前缀

| 步骤 | 预期 |
|------|------|
| `tool_search({"tool_name":"mcp__code-review-graph__semantic_search_nodes"})` | `unknown tool` |
| registry 中不存在 `mcp__*` 键名 | 单元测试或 debug 日志确认 |

## 3. 冲突跳过

### AC-3.1 内建工具冲突

**模拟**：MCP server 暴露工具名 `grep`（或测试 stub）。

| 步骤 | 预期 |
|------|------|
| 启动 ds-code | **不**失败 |
| registry.Get("grep") | 内建 grep |
| SkippedTools | 含 `{tool: grep, reason: builtin_conflict}` |
| 日志 | `mcp tool skipped` warn |

### AC-3.2 跨 server 同名

**模拟**：两个 stub server 均提供 `search`。

| 步骤 | 预期 |
|------|------|
| 启动 ds-code | **不**失败 |
| registry 中无 MCP 来源的 `search` | 是 |
| SkippedTools | **两条**，均为 `cross_server_duplicate` |
| 两个 server 的 `search` 均不可用 | 是 |

### AC-3.3 配置级错误仍失败

| 步骤 | 预期 |
|------|------|
| 两个 `mcp.servers` 同名 `name` | 启动失败，duplicate server name |
| MCP command 不存在 / 连接失败 | 启动失败（与 v0.1.0 一致） |

### AC-3.4 单 server 内工具重名

**模拟**：单个 stub server 的 `ListTools` 返回两个 `read_dup`（或测试等价场景）。

| 步骤 | 预期 |
|------|------|
| 启动 ds-code | **不**失败 |
| registry 中 MCP 来源的 `read_dup` | 至多 **1** 个（第一条保留） |
| SkippedTools | 含 `{tool: read_dup, reason: in_server_duplicate}` |
| 日志 | `buildTools` 完成后出现 `mcp tool skipped` |

### AC-3.5 跳过日志完整性

| 步骤 | 预期 |
|------|------|
| 同时存在 `cross_server_duplicate` 与 `builtin_conflict` | project log 中**两类**跳过均有 `mcp tool skipped` |
| 日志时间点 | 在 `MCP ready` 与首次 `buildTools` **之后**，非仅在 `ensureMCP` 后 |

## 4. TUI Header 消息通知区

### AC-4.1 敏感日志警告

**前置**：`bin/ds-code -vv --allow-log-sensitive-data`

| 步骤 | 预期 |
|------|------|
| header 消息通知区 | 完整中文警告可见；**自动换行**，无 `` 乱码、无单行 `...` 截断 |
| footer 下方 | **无**独立 `SensitiveLogWarn` 红条 |
| stderr（TUI 前） | 仍有一行警告（scrollback） |

### AC-4.2 MCP 跳过提示

**前置**：存在至少一条 SkippedTool。

| 步骤 | 预期 |
|------|------|
| 消息通知区 | 显示 MCP 跳过摘要 + **全部** `tool@server (原因)` 行 |
| 样式 | warn 级别（红色）；首行带 `⚠ ` |

### AC-4.3 无告警时

| 步骤 | 预期 |
|------|------|
| 无 MCP 跳过、未开 sensitive log | 通知区空白；布局不崩 |
| 宽度 20–200 列 | 不 panic（fuzz 测试） |

### AC-4.4 窄终端

| 步骤 | 预期 |
|------|------|
| 宽度 &lt; 72 列 | 通知区在身份区下方全宽换行，不覆盖 logo |

### AC-4.5 通知区自动滚动

**前置**：换行后通知总行数 > 8（如多条 MCP 跳过 + 长敏感日志文案）。

| 步骤 | 预期 |
|------|------|
| 初始 | 显示前 8 行 + `通知 1–8 / n`（无键盘快捷键提示） |
| 等待自动滚动 tick（约 4s）或测试注入 `NoticeScrollTickMsg` | scroll offset 递增，可见后续通知行 |
| 滚至末尾 | offset 归零，循环展示 |
| 终端 resize | scroll offset 归零 |

**集成测试**：`internal/tuitest/notice_test.go` — `TestHarness_headerNoticeAutoScroll`（`-tags=tuitest`）。

### AC-4.6 非 TUI 模式

| 步骤 | 预期 |
|------|------|
| `bin/ds-code -p ...` 且存在 MCP 跳过 | 无 header；project log 含 `mcp tool skipped` |

## 5. 权限、展示与 defer

### AC-5.1 defer_mcp

| 步骤 | 预期 |
|------|------|
| 默认 `defer_mcp: true` | MCP 裸名工具以 stub schema 注册 |
| `tool_search` 裸名 | 返回完整 schema |
| Execute 裸名 | 正常调用 MCP |

### AC-5.2 写工具权限

| 步骤 | 预期 |
|------|------|
| MCP 工具名含写启发式（如 `write_*`）且已注册 | `Manager.IsWriteTool` 返回 true |
| 内建只读工具 `grep` / `read_file` | `Manager.IsWriteTool` 返回 **false**（不因裸名启发式误判） |
| ask 模式下调用 MCP 写工具 | 触发权限 prompt |
| ask 模式下 MCP 写工具参数摘要 | `formatArgsSummary` 输出 JSON 摘要（非空），不依赖 `mcp__` 前缀 |

### AC-5.3 TUI 裸名 MCP 展示

| 步骤 | 预期 |
|------|------|
| TUI 中调用 `semantic_search_nodes` | 工具行标题为 `MCP code-review-graph · semantic_search_nodes`（或等价 server 名） |
| Resume 含 `mcp__*` 的历史块 | 仍可读展示（legacy 解析） |

### AC-5.4 历史 Session

| 步骤 | 预期 |
|------|------|
| Resume v0.1.0 session（含 `mcp__*` tool_calls） | 启动成功；历史消息可展示 |
| 同 session 新发起 MCP 调用 | 须使用裸名；`mcp__*` 新调用返回 `unknown tool` |

## 6. 单元测试清单

实现 PR 须包含或更新：

- [x] `TestManager_DiscoverTools_bareNames`
- [x] `TestManager_DiscoverTools_builtinConflictSkipped`
- [x] `TestManager_DiscoverTools_crossServerDuplicateBothSkipped`
- [x] `TestManager_DiscoverTools_inServerDuplicateSkipped`
- [x] `TestManager_DiscoverTools_duplicateServerNameStillFails`
- [x] `TestManager_IsWriteTool_builtinNotMisclassified`
- [x] `TestManager_IsWriteTool_builtinConflictNotWrite`
- [x] `TestDisplay_MCPBareName`
- [x] `TestDisplay_LegacyMCPPrefix`（resume 历史）
- [x] `TestPermission_FormatArgsSummary_MCPBareName`
- [x] `TestWrapCells_utf8Safe` / `TestBuildNoticeLines_wrapAndPrefix`
- [x] `TestMaxScrollOffset` / `TestRenderNotificationZone_scrollHint` / `TestAdvanceScrollOffset_wraps`
- [x] `TestHarness_headerNoticeAutoScroll`（`-tags=tuitest`）
- [x] `TestHeader_NoticesWideAndNarrow`
- [x] `TestToolSearch_bareMCPName`
- [x] `TestMCPSkipped_LoggedAfterBuildTools`

## 7. 手动验证脚本（参考）

```bash
# 1. 构建
make build

# 2. 日志验证 MCP 连通
bin/ds-code -vv 2>&1 | head -1   # 另开终端看 project log

# 3. 非交互 probe（需 API key）
bin/ds-code -vv --permission-mode readonly -p \
  '只调用 tool_search，tool_name 为 semantic_search_nodes，输出原文'

# 4. 敏感日志 + header
bin/ds-code -vv --allow-log-sensitive-data
# 目视 header 右侧警告，footer 下无重复红条
```

## 8. 发布检查

- [x] CHANGELOG / release note 注明：`mcp__` 前缀已移除、MCP 工具改用裸名（breaking for scripts）、**resume 旧 session 须用裸名发起新 MCP 调用**
- [ ] AGENTS.md **无需**随版本修改
- [ ] 示例配置 [configs/example.yaml](../../configs/example.yaml) MCP 注释可补充裸名说明（可选）
