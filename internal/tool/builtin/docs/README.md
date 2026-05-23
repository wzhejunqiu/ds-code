# 内建工具（builtin）文档

本目录为 `internal/tool/builtin` 包中每个 Agent 内建工具的说明文档，面向维护者与需要深入理解工具行为的开发者。

## 文档索引

| 工具 | 文档 | 权限级别 | 典型运行模式 |
|------|------|----------|--------------|
| `read_file` | [read_file.md](read_file.md) | Low | plan / agent / subagent |
| `grep` | [grep.md](grep.md) | Low | plan / agent / subagent |
| `glob` | [glob.md](glob.md) | Low | plan / agent / subagent |
| `list_dir` | [list_dir.md](list_dir.md) | Low | plan / agent / subagent |
| `diagnostics` | [diagnostics.md](diagnostics.md) | Low | plan / agent（需 LSP） |
| `web_fetch` | [web_fetch.md](web_fetch.md) | Medium | plan / agent（需配置） |
| `web_search` | [web_search.md](web_search.md) | Medium | 占位，未注册 |
| `shell` | [shell.md](shell.md) | Highest | agent only |
| `apply_patch` | [apply_patch.md](apply_patch.md) | High | agent only |
| `write_file` | [write_file.md](write_file.md) | High | agent only |
| `task` | [task.md](task.md) | Low | agent only（需 LLM） |

## 注册与运行模式

工具通过 `internal/tool/setup` 按运行模式注册到 `tool.Registry`：

```text
BuildRegistry(runMode, deps)
  ├─ RegisterReadOnly   → plan 与 agent 共有
  │    ├─ RegisterExploreTools (read_file, grep, glob, list_dir)
  │    ├─ diagnostics（lsp.enabled）
  │    └─ web_fetch（web.fetch_enabled）
  ├─ RegisterWrite      → 仅 agent（runMode != "plan"）
  │    ├─ shell, apply_patch, write_file
  └─ RegisterAgentExtras
       ├─ task（需 llm.Client）
       └─ MCP 工具（mcp__*）
```

- **Plan 模式**：只读探索 + 可选 `web_fetch` / `diagnostics`；禁止写盘与 shell。
- **Agent 模式**：完整写工具与子代理 `task`。
- **Subagent（`task` 内）**：仅 `RegisterExploreTools`，权限引擎为 `readonly`。

实现入口：`setup/setup.go`、`builtin/register_readonly.go`。

## 共同实现约定

每个工具均为实现 `tool.Tool` 接口的结构体：

```go
type Tool interface {
    Name() string
    Description() string
    Schema() map[string]any
    Execute(ctx context.Context, args json.RawMessage) (string, error)
    PermissionLevel() permission.Level
}
```

| 约定 | 说明 |
|------|------|
| `Strict` | 来自 `cfg.LLM.StrictTools`；为 true 时 JSON Schema 禁止 `additionalProperties` |
| 路径 | 读工具用 `Perm.CheckReadablePath`；写工具用 `Perm.ResolvePath` |
| 取消 | `Execute` 开头检查 `ctx.Err()`，长遍历中周期性检查 |
| 敏感路径 | `permission.IsSensitiveAbs` 跳过 `.env`、密钥等 |
| `.gitignore` | `grep` / `glob` / `list_dir` / `diagnostics` 使用 `tool.GitignoreMatcher` |
| 结果截断 | 部分工具调用 `context.TruncateToolResult`；全局见 `context.tool_result_max_chars` |

## 权限级别

定义于 `internal/permission/level.go`：`Low` < `Medium` < `High` < `Highest`。Runner 结合 `permission.mode`（`readonly` / `ask` / `auto`）决定是否向用户确认。

## 相关文档

- 配置键：[docs/CONFIG.md](../../../../docs/CONFIG.md) — `tools.*`、`web.*`、`lsp.*`
- 系统设计：[docs/DESIGN.md](../../../../docs/DESIGN.md) §9
- 产品规划：[docs/PLAN.md](../../../../docs/PLAN.md)
- TUI 展示：`internal/tool/display.go`、`internal/ui/tui/chattool/`
- 子代理：`internal/agent/subagent/README.md`
