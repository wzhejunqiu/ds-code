# ds-code v0.1.4 版本文档

> 版本：v0.1.4  
> 状态：已发布（`v0.1.4`，2026-06-29）  
> 基线版本：v0.1.3  
> 更新日期：2026-06-29

## 概述

v0.1.4 的 **核心目标** 是：**逐一把内建工具的 LLM 提示词改写完**——包括各工具的 `Description()` 以及 JSON Schema 字段 `description`。

**统一实现模式**（参照已落地的 `bash`）：每个工具子包使用 **`usage.prompt` + `text.go`（`//go:embed` + `text/template`）**；正文在 Markdown 中编辑，工具名通过 `{{.ReadFile}}` 等占位符注入。详见 [REQUIREMENTS.md §4 FR-0](REQUIREMENTS.md#fr-0-工具-prompt-标准模式必遵) 与 [DESIGN.md §3](DESIGN.md#3-工具-prompt-标准模式)。

本版 **8 项**内建工具已完成 FR-0 改写；`diagnostics`、`tool_search`、`web_search` 的 prompt **延后至后续版本**（不阻塞 v0.1.4 发布）。

本版本 **不改变** 除 **`bash`（FR-5）**、**`grep`（FR-6）**、**`glob`（ripgrep）** 及 **移除 `list_dir`** 外的工具执行语义、权限策略或 TUI 行为（TUI 另有 sticky scroll、bash 倒计时等增强）。

## 文档索引

| 文档 | 说明 |
|------|------|
| [REQUIREMENTS.md](REQUIREMENTS.md) | 功能需求、工具清单、协作流程 |
| [DESIGN.md](DESIGN.md) | 文案组织、`bash` 改名、`grep` ripgrep、模板注入等技术设计 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | 逐工具验收清单 |
| [TOOL_PROMPTS.md](TOOL_PROMPTS.md) | **各工具改写状态与待确认项** |

## 内建工具改写范围（11 项）

| 工具 | 提示词文件（标准布局） | 状态 |
|------|------------------------|------|
| `read_file` | [`read_file/usage.prompt`](../../internal/tool/builtin/read_file/usage.prompt) + `text.go` | **已定稿**（FR-0；参数 `filepath`） |
| `grep` | [`grep/usage.prompt`](../../internal/tool/builtin/grep/usage.prompt) + `text.go` | **已定稿**（FR-0 + FR-6 ripgrep） |
| `glob` | [`glob/usage.prompt`](../../internal/tool/builtin/glob/usage.prompt) + `text.go` | **已定稿**（FR-0 + ripgrep `--files`；已移除 `list_dir`） |
| `diagnostics` | — | **延后**（沿用 `DescDiagnostics` 常量） |
| `web_fetch` | [`web_fetch/usage.prompt`](../../internal/tool/builtin/web_fetch/usage.prompt) + `text.go` | **已定稿** |
| `web_search` | — | **延后**（默认未注册） |
| `bash` | [`shell/usage.prompt`](../../internal/tool/builtin/shell/usage.prompt) + `text.go` | **已定稿**（参考实现 + FR-5） |
| `apply_patch` | [`apply_patch/usage.prompt`](../../internal/tool/builtin/apply_patch/usage.prompt) + `text.go` | **已定稿** |
| `write_file` | [`write_file/usage.prompt`](../../internal/tool/builtin/write_file/usage.prompt) + `text.go` | **已定稿** |
| `tool_search` | — | **延后**（内联 Description） |
| `agent` | [`agent/usage.prompt`](../../internal/tool/builtin/agent/usage.prompt) + `text.go` | **已定稿** |
| 共享 schema | `builtin/text.go`（无 usage.prompt） | 沿用现有中文 Schema |

## 配套技术项（同版交付）

| 项 | 说明 |
|----|------|
| `shell` → `bash` | LLM 可见工具名与 Cursor 对齐；配置键 `tools.shell` 不变 |
| 工具名注入 | `usage.prompt` 内 `{{.Bash}}` 等；`text.go` 从 `tool.Name*` 填模板 |
| `bash` 参数改造 | `timeout_ms`、`run_in_background`；移除 `list_jobs` |
| **`grep` ripgrep** | 后端改为 ripgrep 15.1.0；见 [FR-6](REQUIREMENTS.md#fr-6-grep-工具-ripgrep-后端p0) |
| **`glob` ripgrep** | 后端改为 ripgrep `--files`；见 [`glob/glob.md`](../../internal/tool/builtin/glob/glob.md) |
| TUI 倒计时 | sync bash Running 标题递减倒计时 |
| read guard | `apply_patch` / `write_file` 编辑前须先 `read_file` |

## 已知限制

| 限制 | 说明 |
|------|------|
| 延后 prompt | `diagnostics`、`tool_search`、`web_search` 未迁至 FR-0 `usage.prompt` |
| 系统提示词 | `internal/prompt/prompt.md` 部分章节待后续审定 |
| 历史会话 | SQLite 中 `name: "shell"` 的 tool_call 不做别名兼容 |
| 历史参数 | `background` / `list_jobs` 不做别名；须用 `run_in_background` |
| 配置迁移 | `tools.defer_builtin` 须写 `bash` 而非 `shell` |
| `grep` breaking | 输出格式、`path`+`glob` 分离、正则方言改为 ripgrep；见 CHANGELOG |
| `glob` breaking | 输出格式改为 `Found N files`；后端改为 ripgrep `--files`；见 CHANGELOG |
| `list_dir` | 已移除；目录列举改用 `glob` |
| MCP 工具 | 本版不改 MCP `Description()` 格式 |
| 子代理 overlay | `agentdef.PromptOverlay` 不在本版范围 |

## 关联文档

- 上一版本：[../v0.1.3/README.md](../v0.1.3/README.md)
- 内建工具实现说明：[../../internal/tool/builtin/README.md](../../internal/tool/builtin/README.md)
- `grep` 实现规格：[../../internal/tool/builtin/grep/grep.md](../../internal/tool/builtin/grep/grep.md)
- `glob` 实现规格：[../../internal/tool/builtin/glob/glob.md](../../internal/tool/builtin/glob/glob.md)
