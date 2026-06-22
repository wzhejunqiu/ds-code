# ds-code v0.1.4 版本文档

> 版本：v0.1.4  
> 状态：规划中  
> 基线版本：v0.1.3  
> 更新日期：2026-06-21

## 概述

v0.1.4 的 **核心目标** 是：**逐一把所有内建工具的 LLM 提示词改写完**——包括各工具的 `Description()` 以及 JSON Schema 字段 `description`。

**统一实现模式**（参照已落地的 `bash`）：每个工具子包使用 **`prompt.md` + `text.go`（`//go:embed` + `text/template`）**；正文在 Markdown 中编辑，工具名通过 `{{.ReadFile}}` 等占位符注入。详见 [REQUIREMENTS.md §4 FR-0](REQUIREMENTS.md#fr-0-工具-prompt-标准模式必遵) 与 [DESIGN.md §3](DESIGN.md#3-工具-prompt-标准模式)。

系统提示词（`internal/prompt/prompt.md`）采用同一套 embed + template 机制；**文案以你为主导**，改写前须确认。

本版本 **不改变** 除 `bash`（FR-5）外的工具执行语义、权限策略或 TUI 行为。

## 文档索引

| 文档 | 说明 |
|------|------|
| [REQUIREMENTS.md](REQUIREMENTS.md) | 功能需求、工具清单、协作流程 |
| [DESIGN.md](DESIGN.md) | 文案组织、`bash` 改名、模板注入等技术设计 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | 逐工具验收清单 |
| [TOOL_PROMPTS.md](TOOL_PROMPTS.md) | **各工具改写状态与待确认项**（随讨论更新） |

## 内建工具改写范围（12 项）

| 工具 | 提示词文件（标准布局） | 状态 |
|------|------------------------|------|
| `read_file` | [`read_file/prompt.md`](../../internal/tool/builtin/read_file/prompt.md) + `text.go` | **已定稿**（FR-0；参数 `filepath`） |
| `grep` | `grep/prompt.md` + `text.go` | 待改写 |
| `glob` | `glob/prompt.md` + `text.go` | 待改写 |
| `list_dir` | `list_dir/prompt.md` + `text.go` | 待改写 |
| `diagnostics` | `diagnostics/prompt.md` + `text.go` | 待改写 |
| `web_fetch` | `web_fetch/prompt.md` + `text.go` | 待改写 |
| `web_search` | `web_search/prompt.md` + `text.go` | 待改写（占位，默认未注册） |
| `bash` | [`shell/prompt.md`](../../internal/tool/builtin/shell/prompt.md) + `text.go` | **参考实现**；草稿待你确认 |
| `apply_patch` | `apply_patch/prompt.md` + `text.go` | 待改写 |
| `write_file` | `write_file/prompt.md` + `text.go` | 待改写 |
| `tool_search` | `tool_search/prompt.md` + `text.go` | 待改写 |
| `agent` | `agent/prompt.md` + `text.go` | 待改写 |
| 共享 schema | `builtin/text.go`（无 prompt.md） | 待改写 |

## 协作约定（重要）

1. **你先定调**：每个工具的用途边界、禁止事项、与 Cursor/Codex 对齐程度，由你拍板。
2. **改写前必问**：Agent/实现者给出草稿或选项后，等你确认再落代码。
3. **一次一工具或一小批**：避免大批量静默替换导致风格漂移。
4. **系统提示词**：`internal/prompt/prompt.md` 与工具层同模式；内容由你审定。
5. **工具 Description**：各子包 `internal/tool/builtin/<tool>/prompt.md`；**禁止**在 `text.go` 用大段 `const Desc*` 或 `fmt.Sprintf` 硬编码正文（Schema / Err / Result 除外）。

## 配套技术项（非文案核心，但同版交付）

| 项 | 说明 |
|----|------|
| `shell` → `bash` | LLM 可见工具名与 Cursor 对齐；配置键 `tools.shell` 不变 |
| 工具名注入 | `prompt.md` 内 `{{.Bash}}` 等；`text.go` 从 `tool.Name*` 填模板 |
| 参考实现 | [`internal/tool/builtin/shell/`](../../internal/tool/builtin/shell/)（`prompt.md` + `RenderDesc()`） |
| `bash` 参数改造 | `timeout_ms`、`run_in_background`；移除 `list_jobs` |
| TUI 倒计时 | sync bash Running 标题递减倒计时 |
| 系统提示词 | [`internal/prompt/`](../../internal/prompt/)（同一 embed + template 模式） |

## 已知限制

| 限制 | 说明 |
|------|------|
| 历史会话 | SQLite 中 `name: "shell"` 的 tool_call 不做别名兼容 |
| 历史参数 | `background` / `list_jobs` 不做别名；须用 `run_in_background` |
| 配置迁移 | `tools.defer_builtin` 须写 `bash` 而非 `shell` |
| MCP 工具 | 本版不改 MCP `Description()` 格式 |
| 子代理 overlay | `agentdef.PromptOverlay` 不在本版范围 |

## 关联文档

- 上一版本：[../v0.1.3/README.md](../v0.1.3/README.md)
- 内建工具实现说明：[../../internal/tool/builtin/README.md](../../internal/tool/builtin/README.md)
