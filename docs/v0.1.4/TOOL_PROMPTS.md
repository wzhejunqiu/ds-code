# v0.1.4 内建工具提示词改写台账

> 版本：v0.1.4  
> 状态：规划中  
> 更新日期：2026-06-21  
> **维护说明**：每改一个工具，更新下表「状态」与「确认人」；草稿放在 PR/对话中，**未经你确认不得标为已定稿**。

## 已确认原则（2026-06-21）

| 项 | 你的决定 |
|----|----------|
| Description 风格 | **偏长**：像 `read_file` / `bash` 那样分点写清用法与边界 |
| 改写范围 | **全部 12 项**内建工具 + 共享 `builtin/text.go` Schema |
| **实现模式** | **`prompt.md` + `text.go`（embed + template）**；参照 [`shell/`](../../internal/tool/builtin/shell/) |
| 「禁止 bash 绕行」 | **每个工具改时再定** |
| 系统提示词 | [`internal/prompt/prompt.md`](../../internal/prompt/prompt.md)，同模式；**逐段审定** |

## 标准模式（FR-0）

每个工具子包（共享 `builtin/text.go` 除外）：

| 文件 | 职责 |
|------|------|
| `prompt.md` | Description 正文；跨工具名用 `{{.ReadFile}}` 等 |
| `text.go` | `//go:embed prompt.md`、`RenderDesc()`、`Schema*`、`Err*`、`Result*` |
| `<tool>.go` | `Description() { return RenderDesc() }` |
| `text_test.go` | 建议：模板已渲染、无 `{{.` 残留 |

**参考实现**：[`shell/usage.prompt`](../../internal/tool/builtin/shell/usage.prompt)、[`shell/text.go`](../../internal/tool/builtin/shell/text.go)

> **bash 例外**：Description 正文文件名为 `usage.prompt`（非通用 `prompt.md`）；`description` 参数字段 schema 来自 `shell_cmd_description.prompt`。其余工具仍按 `prompt.md` + `text.go` 迁移。

## 改写对象（LLM 可见部分）

| 类型 | 代码位置 | 发给 LLM 的形式 |
|------|----------|-----------------|
| 工具描述 | `<tool>/prompt.md` → `RenderDesc()` | `tools[].description` |
| 参数说明 | `<tool>/text.go` 的 `Schema*`、`builtin/text.go` | `parameters.properties.*.description` |
| 必填/错误文案 | `Err*`、`Result*` | 不发给 LLM |

## 工具清单与状态

| 工具 | prompt.md | Schema（text.go） | 状态 | 备注 |
|------|-----------|-------------------|------|------|
| `read_file` | 待建 | `SchemaOffset` 等 | 待迁移 FR-0 + 定稿 | 现有 `const DescReadFile` 待迁入 |
| `grep` | 待建 | `SchemaRegexPattern` 等 | 待改写 | |
| `glob` | 待建 | `SchemaGlobPattern` 等 | 待改写 | |
| `list_dir` | 待建 | `SchemaPathRelDefault` 等 | 待改写 | |
| `diagnostics` | 待建 | `SchemaSeverity` 等 | 待改写 | |
| `web_fetch` | 待建 | `SchemaHTTPURL` 等 | 待改写 | |
| `web_search` | 待建 | `SchemaQuery` | 待改写 | 默认未注册 |
| `bash` | ✅ [`shell/usage.prompt`](../../internal/tool/builtin/shell/usage.prompt) | ✅ `text.go` | **已定稿 + FR-5 已实现** | `timeout_ms`、`run_in_background`；无 `list_jobs`；退出 kill job |
| `apply_patch` | 待建 | `SchemaPatchBody` 等 | 待改写 | |
| `write_file` | 待建 | `SchemaFullFileContent` 等 | 待改写 | |
| `tool_search` | 待建 | `tool_name` | 待改写 | 去掉 `.go` 内联 Desc |
| `agent` | 待建 | `SchemaAgent*` | 待改写 | |
| 共享 | — | `builtin/text.go` | 待审定 | 无 prompt.md |

## 当前文案快照（基线）

定稿后删除或改为链接。`bash` 以 [`shell/usage.prompt`](../../internal/tool/builtin/shell/usage.prompt) 为准，此处不再重复。

### read_file（待迁入 prompt.md）

```
读取本地文件系统中的文件。…
用法：
- 本工具仅支持读取文件…请通过 {bash} 工具执行 ls 命令。
…
```

### grep / glob / list_dir / …

见 git 基线或各包现有 `text.go` 中 `Desc*`。

---

**下一步**：指定下一个要改的工具；我会先给 `prompt.md` 草稿，你确认后再按 FR-0 落代码。
