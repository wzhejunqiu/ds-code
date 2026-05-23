# diagnostics

## 概述

通过 **Language Server Protocol（LSP）** 获取工作区文件的编译/静态分析诊断（错误、警告等），将结果以文本摘要返回给 Agent。

## 注册与可见性

| 模式 | 注册条件 |
|------|----------|
| plan / agent | `lsp.enabled == true` 且 `setup.Deps.LSP != nil` |

LSP 关闭时不注册；调用时若被误注册也会返回 `LSP is disabled in config`。

## 参数 Schema

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `paths` | string[] | 是 | 文件或目录（相对项目根） |
| `severity` | string[] | 否 | 过滤：`error`、`warning`、`info`、`hint`；默认仅 `error` + `warning` |

## 用法示例

```json
{"paths": ["internal/tool/builtin/read_file.go"]}
```

```json
{"paths": ["internal/agent"], "severity": ["error"]}
```

## 返回格式

每条诊断：

```text
path/to/file:行:列 [severity] message
```

无问题时：`path/to/file: (no issues)`

目录遍历可能跳过无 LSP server 的扩展名，附注：`--- skip path: no LSP server for extension`

整体结果可能经 `context.TruncateToolResult` 截断。

## 实现细节

源文件：[`diagnostics.go`](../diagnostics.go)

### 流程

1. `collectFiles`：展开目录（`WalkDir`），尊重 `.gitignore`、敏感路径、`.git`；仅保留注册表中有 server 的扩展名；最多 `lsp.max_files_per_call` 个文件。
2. 按 LSP server ID 分组。
3. `LSP.EnsureClient` 启动或复用子进程（gopls、clangd、typescript-language-server 等）。
4. 每文件：`OpenFile` → 收集 `publishDiagnostics`，每文件最多 `max_issues_per_file` 条。

### 与 LSP 子系统关系

```text
diagnostics.Execute
  → lsp.Manager
  → lsp.Client (stdio JSON-RPC)
  → 用户 PATH 中的语言服务器
```

详见 `docs/DESIGN.md` §9.5。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `lsp.enabled` | true | 总开关 |
| `lsp.max_files_per_call` | 10 | 单次工具调用最多分析文件数 |
| `lsp.max_issues_per_file` | 20 | 每文件最多返回诊断条数 |
| `lsp.diagnostics_timeout` | 20s | 客户端超时 |
| `lsp.servers.*` | 见 example.yaml | 各语言启动命令 |

## 权限与安全

- **PermissionLevel**：`Low`
- 只读：仅 `didOpen` 读文件内容发给 LSP，不向磁盘写入
- 路径必须在工作区内

## 设计思想

- **不捆绑语言 SDK**：服务器由用户本机安装，ds-code 只做编排。
- **摘要而非全量 LSP**：不提供补全、跳转；专注「有没有错」。
- **按扩展名路由**：大目录调用时自动过滤无 server 的文件，控制成本。

## 相关代码

- [`diagnostics.go`](../diagnostics.go)
- [`diagnostics_integration_test.go`](../diagnostics_integration_test.go)
- [`lsp/`](../../../lsp/) — Manager、Client、语言 server 注册
