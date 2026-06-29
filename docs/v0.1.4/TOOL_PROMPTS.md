# v0.1.4 内建工具提示词改写台账

> 版本：v0.1.4  
> 状态：已发布（`v0.1.4`，2026-06-29）  
> 更新日期：2026-06-29  

## 已确认原则（2026-06-21）

| 项 | 你的决定 |
|----|----------|
| Description 风格 | **偏长**：像 `read_file` / `bash` 那样分点写清用法与边界 |
| 改写范围 | **8 项**内建工具本版定稿；`diagnostics` / `tool_search` / `web_search` **延后** |
| **实现模式** | **`usage.prompt` + `text.go`（embed + template）**；参照 [`shell/`](../../internal/tool/builtin/shell/) |
| 「禁止 bash 绕行」 | **每个工具改时再定** |
| 系统提示词 | [`internal/prompt/prompt.md`](../../internal/prompt/prompt.md)，同模式；**已定稿**（2026-06-29 确认） |

## 标准模式（FR-0）

每个工具子包（共享 `builtin/text.go` 除外）：

| 文件 | 职责 |
|------|------|
| `usage.prompt` | Description 正文；跨工具名用 `{{.ReadFile}}` 等 |
| `text.go` | `//go:embed usage.prompt`、`RenderDesc()`、`Schema*`、`Err*`、`Result*` |
| `<tool>.go` | `Description() { return RenderDesc() }` |
| `text_test.go` | 建议：模板已渲染、无 `{{.` 残留 |

**参考实现**：[`shell/usage.prompt`](../../internal/tool/builtin/shell/usage.prompt)、[`shell/text.go`](../../internal/tool/builtin/shell/text.go)

## 工具清单与状态

| 工具 | usage.prompt | Schema（text.go） | 状态 | 备注 |
|------|--------------|-------------------|------|------|
| `read_file` | ✅ [`usage.prompt`](../../internal/tool/builtin/read_file/usage.prompt) | ✅ `SchemaFilepath` 等 | **已定稿 + FR-0 已实现** | 参数 `filepath`（原 `path`） |
| `grep` | ✅ [`usage.prompt`](../../internal/tool/builtin/grep/usage.prompt) | ✅ 中文 Schema | **已定稿 + FR-6 已实现** | ripgrep 15.1.0 |
| `glob` | ✅ [`usage.prompt`](../../internal/tool/builtin/glob/usage.prompt) | ✅ 中文 Schema | **已定稿 + ripgrep 已实现** | 已移除 `list_dir` |
| `diagnostics` | — | `SchemaSeverity` 等 | **延后**（v0.1.4 不阻塞） | 沿用 `DescDiagnostics` |
| `web_fetch` | ✅ [`usage.prompt`](../../internal/tool/builtin/web_fetch/usage.prompt) | ✅ `SchemaURL`/`SchemaPrompt` | **已定稿 + 能力已实现** | analyze pipeline |
| `web_search` | — | `SchemaQuery` 等 | **延后**（v0.1.4 不阻塞） | 默认未注册 |
| `bash` | ✅ [`shell/usage.prompt`](../../internal/tool/builtin/shell/usage.prompt) | ✅ `text.go` | **已定稿 + FR-5 已实现** | `timeout_ms`、`run_in_background` |
| `apply_patch` | ✅ [`usage.prompt`](../../internal/tool/builtin/apply_patch/usage.prompt) | ✅ `SchemaPatchBody` 等 | **已定稿 + FR-0 已实现** | read-before-edit guard |
| `write_file` | ✅ [`usage.prompt`](../../internal/tool/builtin/write_file/usage.prompt) | ✅ `SchemaFullFileContent` 等 | **已定稿 + FR-0 已实现** | read-before-write guard |
| `tool_search` | — | `tool_name` | **延后**（v0.1.4 不阻塞） | 内联 Description |
| `agent` | ✅ [`usage.prompt`](../../internal/tool/builtin/agent/usage.prompt) | ✅ `SchemaAgent*`（中文） | **已定稿 + FR-0 已实现** | LLM 仅见 GP/Explore |
| 共享 | — | `builtin/text.go` | 沿用现有中文 Schema | 无 usage.prompt |

已定稿工具以源码中 `usage.prompt` 为准。
