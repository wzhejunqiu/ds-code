# 内建工具（builtin）文档

本目录为 `internal/tool/builtin` 包中每个 Agent 内建工具的说明文档，面向维护者与需要深入理解工具行为的开发者。

## 目录结构

```text
internal/tool/builtin/
├── README.md              # 本文件：总览与注册约定
├── doc.go
├── text.go                # 共享 Schema 描述、校验错误、截断后缀
├── filecandidate.go       # grep 等共用：候选文件收集与 workspace 校验（glob 已改 ripgrep --files）
├── rgutil/                # grep / glob 共用：ripgrep 路径、exec、format、globs
├── grep_output.go         # ParseGrepOutputMode 等共用；grep 输出格式化在 grep/format_output.go
├── sort.go                # 按 ModTime 降序排序
├── read_file/             # 工具实现 + text.go（LLM 文案）+ *.md
├── grep/
│   ├── grep.go            # GrepTool、Schema、Execute
│   ├── ripgrep.go         # rg 参数、exec、JSON 解析、postProcess
│   ├── format_output.go   # §3.4 LLM 可见纯文本格式化
│   ├── text.go            # RenderDesc、Schema 常量
│   ├── usage.prompt       # LLM 工具说明模板
│   └── rgbin/             # embed rg.tar.gz → ~/.ds-code/bin/rg
├── glob/
│   ├── glob.go            # GlobTool、Schema、Execute
│   ├── ripgrep.go         # rg --files 参数、解析、postProcess
│   ├── text.go / usage.prompt
│   └── *_test.go          # G/H/I/O 测试组（见 glob.md）
├── diagnostics/
├── web_fetch/
├── web_search/            # 占位实现，未注册
├── shell/
├── apply_patch/
├── write_file/
├── tool_search/           # 延迟加载工具的 schema 查询
└── agent/                 # 子代理 spawn 入口（委托 internal/agent/spawn）
```

每个工具子包通常包含：

| 文件           | 职责                                                                |
| -------------- | ------------------------------------------------------------------- |
| `<tool>.go`    | 实现 `tool.Tool`（及可选的 `ReadOnlyTool` / `ConcurrencySafeTool`） |
| `usage.prompt` | LLM 可见 `Description` 正文（`//go:embed` + `RenderDesc()`）        |
| `text.go`      | `RenderDesc()`、`Schema*` 字段说明、错误文案                        |
| `<tool>.md`    | 设计文档（部分工具尚未单独成文，见下表）                            |
| `*_test.go`    | 单元 / 集成测试                                                     |

工具名常量统一定义于 [`name.go`](../name.go)（如 `NameReadFile = "read_file"`），各工具 `Name()` 返回对应常量。

## 文档索引

| 工具          | 文档                                                     | 权限级别 | 典型运行模式                            |
| ------------- | -------------------------------------------------------- | -------- | --------------------------------------- |
| `read_file`   | [read_file/read_file.md](read_file/read_file.md)         | Low      | plan / agent / subagent                 |
| `grep`        | [grep/grep.md](grep/grep.md)                             | Low      | plan / agent / subagent（ripgrep 后端） |
| `glob`        | [glob/glob.md](glob/glob.md)                             | Low      | plan / agent / subagent（ripgrep `--files` 后端；目录列举合并入 glob） |
| `diagnostics` | [diagnostics/diagnostics.md](diagnostics/diagnostics.md) | Low      | plan / agent（需 LSP）                  |
| `web_fetch`   | [web_fetch/web_fetch.md](web_fetch/web_fetch.md)         | Medium   | plan / agent（需配置）                  |
| `web_search`  | [web_search/web_search.md](web_search/web_search.md)     | Medium   | 占位，未注册                            |
| `bash`        | [shell/shell.md](shell/shell.md)                         | Highest  | agent only                              |
| `apply_patch` | [apply_patch/apply_patch.md](apply_patch/apply_patch.md) | High     | agent only                              |
| `write_file`  | [write_file/write_file.md](write_file/write_file.md)     | High     | agent only                              |
| `tool_search` | [tool_search/tool_search.md](tool_search/tool_search.md) | Low      | agent only（有 defer 时）               |
| `agent`       | [agent/agent.md](agent/agent.md)                         | Low      | agent only（需 LLM）                    |

子代理完整生命周期见 [`internal/agent/spawn/README.md`](../../agent/spawn/README.md)。

## 注册与运行模式

工具通过 [`setup/setup.go`](../setup/setup.go) 按运行模式注册到 `tool.Registry`：

```text
BuildRegistry(runMode, deps)
  ├─ RegisterReadOnly   → plan 与 agent 共有
  │    ├─ register.ExploreTools (read_file, grep, glob)
  │    ├─ diagnostics（lsp.enabled）
  │    └─ web_fetch（web.fetch_enabled）
  ├─ RegisterWrite      → 仅 agent（runMode != plan）
  │    ├─ shell, apply_patch, write_file
  │    └─ 可选 WrapDeferred（tools.defer_builtin 列出的名称）
  └─ RegisterAgentExtras → 仅 agent
       ├─ tool_search（Registry 引用，供 defer 查 schema）
       ├─ agent（需 llm.Client；委托 spawn.Service）
       └─ MCP 工具（裸名，由 MCP Manager 注册）
```

- **Plan 模式**：只读探索 + 可选 `web_fetch` / `diagnostics`；禁止写盘与 shell。
- **Agent 模式**：完整写工具、子代理 `agent`、`tool_search` 与 MCP。
- **Subagent（`agent` 内）**：按类型过滤工具池（[`spawn/toolpool.go`](../../agent/spawn/toolpool.go)）；Explore/Plan 类型为 readonly，不含 `agent` 自身。

实现入口：[`setup/setup.go`](../setup/setup.go)、[`register/explore.go`](../register/explore.go)。

## 延迟加载（defer）

为缩小发给 LLM 的工具 schema 体积，写工具可通过 `tools.defer_builtin` 延迟暴露完整参数：

| 配置                  | 说明                                                                   |
| --------------------- | ---------------------------------------------------------------------- |
| `tools.defer_builtin` | 工具名列表，如 `["bash", "apply_patch"]`；命中项经 `WrapDeferred` 注册 |
| `tool_search`         | agent 模式下始终注册；LLM 用其按名称拉取完整 schema 后再调用目标工具   |

机制：

1. `Registry.Definitions()` 对 `DeferredTool` 仅发送 `StubSchema`（含 `_note` 提示先调 `tool_search`）。
2. `Registry.FullSchema(name)` 与 `tool_search.Execute` 返回完整 JSON Schema。
3. 实际 `Execute` 仍走内层工具，行为与未 defer 时一致。

MCP 侧另有 `tools.defer_mcp`（见 MCP Manager），与本节内建 defer 独立。

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

### 可选接口

| 接口                  | 作用                                                        |
| --------------------- | ----------------------------------------------------------- |
| `ReadOnlyTool`        | `IsReadOnly() bool` — 探索类工具返回 `true`                 |
| `ConcurrencySafeTool` | `IsConcurrencySafe() bool` — 可与其它只读工具并行执行       |
| `DeferredTool`        | `StubSchema` / `ShouldDefer` — 由 `WrapDeferred` 包装写工具 |

Runner 在 [`tool_orchestration.go`](../../agent/tool_orchestration.go) 中将**相邻**且同时满足 `IsToolReadOnly` + `IsToolConcurrencySafe` 的调用合并为并发批次（上限 10）；写工具、`agent` 等无上述接口的实现始终串行。

### 其它约定

| 约定     | 说明                                                                                                                                 |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `Strict` | 来自 `cfg.LLM.StrictTools`；为 true 时 JSON Schema 禁止 `additionalProperties`                                                       |
| 路径     | 读工具用 `Perm.CheckReadablePath`；写工具用 `Perm.CheckWritablePath` / `ResolveAccessPath`                                           |
| 取消     | `Execute` 开头检查 `ctx.Err()`，长遍历中周期性检查                                                                                   |
| 敏感路径 | `permission.SkipSensitiveAbs` 跳过 `.env`、密钥等（`@` 引用例外，见 SECURITY §S3-S）                                                 |
| 搜索跳过 | `grep` / `glob` / `diagnostics` 使用 `searchskip.Matcher`（`.git` + `tools.search.skip_dirs`）；**不**读取 `.gitignore` |
| 结果截断 | 部分工具调用 `context.TruncateToolResult`；全局见 `context.tool_result_max_chars`                                                    |
| LLM 文案 | 共享常量见 [`text.go`](text.go)；各工具专有字符串见子包 `text.go`                                                                    |

## 权限级别

定义于 [`permission/level.go`](../../permission/level.go)：`Low` < `Medium` < `High` < `Highest`。Runner 结合 `permission.mode`（`readonly` / `ask` / `auto`）决定是否向用户确认。

## 相关代码

- [`registry.go`](../registry.go) — `Tool` 接口、`Registry`、`DeferredTool`
- [`deferred.go`](../deferred.go)、[`deferred_wrapper.go`](../deferred_wrapper.go) — 延迟加载包装
- [`concurrency.go`](../concurrency.go) — `ReadOnlyTool` / `ConcurrencySafeTool` 判定
- [`schema.go`](../schema.go) — `ObjectSchema`
- [`setup/setup.go`](../setup/setup.go) — 按运行模式注册工具
- [`display.go`](../display.go) — TUI 工具行展示
- [`runner_loop.go`](../../agent/runner_loop.go) — 工具调用与展示回调

## 相关文档

- 配置键：[docs/v0.1.0/CONFIG.md](../../../docs/v0.1.0/CONFIG.md) — `tools.*`、`web.*`、`lsp.*`
- 系统设计：[docs/v0.1.0/DESIGN.md](../../../docs/v0.1.0/DESIGN.md) §9
- 产品规划：[docs/v0.1.0/PLAN.md](../../../docs/v0.1.0/PLAN.md)
- TUI 展示：[`display.go`](../display.go)、[`ui/tui/chattool/`](../../ui/tui/chattool/)
- 子代理：[`agent/spawn/README.md`](../../agent/spawn/README.md)
