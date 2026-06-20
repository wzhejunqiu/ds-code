# ds-code v0.1.4 版本文档

> 版本：v0.1.4  
> 状态：规划中  
> 基线版本：v0.1.3  
> 更新日期：2026-06-21

## 概述

v0.1.4 的 **核心目标** 是：**逐一把所有内建工具的 LLM 提示词改写完**——包括各工具的 `Description()` 以及 JSON Schema 字段 `description`（`internal/tool/builtin/**/text.go` 与共享 `builtin/text.go`）。

系统提示词（`internal/prompt/prompt.md`）可作为配套工作同步推进，但 **文案以你（产品/维护者）为主导**；实现者在改写任何提示词前须先与你确认方向与草稿，不得自行臆造大段规范。

本版本 **不改变** 工具执行语义、权限策略或 TUI 行为。

## 文档索引

| 文档 | 说明 |
|------|------|
| [REQUIREMENTS.md](REQUIREMENTS.md) | 功能需求、工具清单、协作流程 |
| [DESIGN.md](DESIGN.md) | 文案组织、`bash` 改名、模板注入等技术设计 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | 逐工具验收清单 |
| [TOOL_PROMPTS.md](TOOL_PROMPTS.md) | **各工具改写状态与待确认项**（随讨论更新） |

## 内建工具改写范围（12 项）

| 工具 | 文件 | 状态 |
|------|------|------|
| `read_file` | `read_file/text.go` | 进行中（已有 Cursor 风格长描述草稿） |
| `grep` | `grep/text.go` | 待改写 |
| `glob` | `glob/text.go` | 待改写 |
| `list_dir` | `list_dir/text.go` | 待改写 |
| `diagnostics` | `diagnostics/text.go` | 待改写 |
| `web_fetch` | `web_fetch/text.go` | 待改写 |
| `web_search` | `web_search/text.go` | 待改写（占位，默认未注册） |
| `bash`（原 `shell`） | `shell/text.go` | 待改写 |
| `apply_patch` | `apply_patch/text.go` | 待改写 |
| `write_file` | `write_file/text.go` | 待改写 |
| `tool_search` | `tool_search/tool_search.go` + 拟增 `text.go` | 待改写 |
| `agent` | `agent/text.go` | 待改写 |
| 共享 schema | `builtin/text.go` | 待改写 |

## 协作约定（重要）

1. **你先定调**：每个工具的用途边界、禁止事项、与 Cursor/Codex 对齐程度，由你拍板。
2. **改写前必问**：Agent/实现者给出草稿或选项后，等你确认再落代码。
3. **一次一工具或一小批**：避免大批量静默替换导致风格漂移。
4. **系统提示词**：`prompt.md` 内容由你提供或逐段审定；工具名占位符（`{{.Bash}}` 等）由代码注入，正文不硬编码 wire 名。

## 配套技术项（非文案核心，但同版交付）

| 项 | 说明 |
|----|------|
| `shell` → `bash` | LLM 可见工具名与 Cursor 对齐；配置键 `tools.shell` 不变 |
| 工具名注入 | 提示词中引用其它工具时，用 `tool.Name*` / `fmt.Sprintf` / 模板，避免硬编码 |
| 系统提示词载体 | `prompt.md` + embed（若你审定后的 base 在本版合入） |

## 已知限制

| 限制 | 说明 |
|------|------|
| 历史会话 | SQLite 中 `name: "shell"` 的 tool_call 不做别名兼容 |
| 配置迁移 | `tools.defer_builtin` 须写 `bash` 而非 `shell` |
| MCP 工具 | 本版不改 MCP `Description()` 格式 |
| 子代理 overlay | `agentdef.PromptOverlay` 不在本版范围 |

## 关联文档

- 上一版本：[../v0.1.3/README.md](../v0.1.3/README.md)
- 内建工具实现说明：[../../internal/tool/builtin/README.md](../../internal/tool/builtin/README.md)
