# v0.1.4 内建工具提示词改写台账

> 版本：v0.1.4  
> 状态：规划中  
> 更新日期：2026-06-23  
> **维护说明**：每改一个工具，更新下表「状态」与「确认人」；草稿放在 PR/对话中，**未经你确认不得标为已定稿**。

## 已确认原则（2026-06-21）

| 项 | 你的决定 |
|----|----------|
| Description 风格 | **偏长**：像 `read_file` / `bash` 那样分点写清用法与边界 |
| 改写范围 | **全部 12 项**内建工具 + 共享 `builtin/text.go` Schema |
| **实现模式** | **`usage.prompt` + `text.go`（embed + template）**；参照 [`shell/`](../../internal/tool/builtin/shell/) |
| 「禁止 bash 绕行」 | **每个工具改时再定** |
| 系统提示词 | [`internal/prompt/prompt.md`](../../internal/prompt/prompt.md)，同模式；**逐段审定** |

## 标准模式（FR-0）

每个工具子包（共享 `builtin/text.go` 除外）：

| 文件 | 职责 |
|------|------|
| `usage.prompt` | Description 正文；跨工具名用 `{{.ReadFile}}` 等 |
| `text.go` | `//go:embed usage.prompt`、`RenderDesc()`、`Schema*`、`Err*`、`Result*` |
| `<tool>.go` | `Description() { return RenderDesc() }` |
| `text_test.go` | 建议：模板已渲染、无 `{{.` 残留 |

**参考实现**：[`shell/usage.prompt`](../../internal/tool/builtin/shell/usage.prompt)、[`shell/text.go`](../../internal/tool/builtin/shell/text.go)

> **bash 附加**：`description` 参数字段 schema 来自 `shell_cmd_description.prompt`（仅 bash 需要）。

## 改写对象（LLM 可见部分）

| 类型 | 代码位置 | 发给 LLM 的形式 |
|------|----------|-----------------|
| 工具描述 | `<tool>/usage.prompt` → `RenderDesc()` | `tools[].description` |
| 参数说明 | `<tool>/text.go` 的 `Schema*`、`builtin/text.go` | `parameters.properties.*.description` |
| 必填/错误文案 | `Err*`、`Result*` | 不发给 LLM |

## 工具清单与状态

| 工具 | usage.prompt | Schema（text.go） | 状态 | 备注 |
|------|--------------|-------------------|------|------|
| `read_file` | ✅ [`usage.prompt`](../../internal/tool/builtin/read_file/usage.prompt) | ✅ `SchemaFilepath` 等 | **已定稿 + FR-0 已实现** | 参数 `filepath`（原 `path`）；`offset`/`limit` 不变 |
| `grep` | ✅ [`usage.prompt`](../../internal/tool/builtin/grep/usage.prompt) | ✅ `SchemaPattern` 等（中文） | **已定稿 + FR-6 已实现** | ripgrep 15.1.0；`{{.Grep}}/{{.Bash}}/{{.Agent}}` 注入；输出 `Found N files` 等；不搜 `.git` |
| `glob` | ✅ [`usage.prompt`](../../internal/tool/builtin/glob/usage.prompt) | ✅ `SchemaPattern` 等（中文） | **已定稿 + ripgrep 已实现** | ripgrep `--files`；`Found N files` 输出；默认不遵循 gitignore、含隐藏文件 |
| `list_dir` | 待建 | `SchemaPathRelDefault` 等 | 待改写 | |
| `diagnostics` | 待建 | `SchemaSeverity` 等 | 待改写 | |
| `web_fetch` | 待建 | `SchemaHTTPURL` 等 | 待改写 | |
| `web_search` | 待建 | `SchemaQuery` 等 | 待改写 | 默认未注册 |
| `bash` | ✅ [`shell/usage.prompt`](../../internal/tool/builtin/shell/usage.prompt) | ✅ `text.go` | **已定稿 + FR-5 已实现** | `timeout_ms`、`run_in_background`；无 `list_jobs`；退出 kill job |
| `apply_patch` | ✅ [`usage.prompt`](../../internal/tool/builtin/apply_patch/usage.prompt) | ✅ `SchemaPatchBody` 等 | **已定稿 + FR-0 已实现** | |
| `write_file` | ✅ [`usage.prompt`](../../internal/tool/builtin/write_file/usage.prompt) | ✅ `SchemaFullFileContent` 等 | **已定稿 + FR-0 已实现** | 覆盖已有文件须先 read；*.md/README/emoji 仅 prompt |
| `tool_search` | 待建 | `tool_name` | 待改写 | 去掉 `.go` 内联 Desc |
| `agent` | 待建 | `SchemaAgent*` | 待改写 | |
| 共享 | — | `builtin/text.go` | 待审定 | 无 usage.prompt |

## 当前文案快照（基线）

定稿后删除或改为链接。已定稿工具以源码中 `usage.prompt` 为准，此处不再重复：

- [`bash`](../../internal/tool/builtin/shell/usage.prompt)
- [`read_file`](../../internal/tool/builtin/read_file/usage.prompt)
- [`grep`](../../internal/tool/builtin/grep/usage.prompt)

### glob / list_dir / …

见 git 基线或各包现有 `text.go` 中 `Desc*`。

---

**下一步**：指定下一个要改的工具；我会先给 `usage.prompt` 草稿，你确认后再按 FR-0 落代码。
