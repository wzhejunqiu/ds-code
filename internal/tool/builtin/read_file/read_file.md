# read_file

## 概述

在工作区内读取单个文本文件内容，支持按行范围分页，避免大文件一次性灌入上下文。

## 注册与可见性

| 模式 | 注册 |
|------|------|
| plan / agent / subagent | `register.ExploreTools` |

## 参数 Schema

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `path` | string | 是 | 相对项目根的路径，或落在工作区内的绝对路径 |
| `offset` | integer | 否 | 起始行号（1-based），默认从第 1 行 |
| `limit` | integer | 否 | 最多读取行数；未指定时使用配置的 `max_lines` |

## 用法示例

```json
{"path": "internal/tool/builtin/read_file.go"}
```

```json
{"path": "docs/CONFIG.md", "offset": 100, "limit": 50}
```

## 返回格式

- 每行：`{行号}|{行内容}\n`
- 若 `limit` 超过配置上限：末尾附加 `... truncated to max_lines; adjust offset/limit to continue`
- 若范围内之后还有行：`... N more lines not shown`
- 若 `offset` 超出文件长度：`(empty: offset N beyond file length M)`

TUI 通过 `FormatReadFileDisplay` / `ReadFileLineRange` 解析行号范围展示。

## 实现细节

源文件：[`read_file.go`](read_file.go)

1. **权限**：`Perm.CheckReadablePath` 解析路径并拒绝敏感文件。
2. **体积预检**：`os.Stat` 检查文件字节数 ≤ `tools.read_file.max_bytes`（默认 2MiB），超限直接报错，不读入内存。
3. **行范围**：`resolveReadOffsetLimit` 计算 `[readStart, readEnd]`；`limit` 会被钳制到 `max_lines`。
4. **扫描**：`bufio.Scanner`，单行 buffer 上限 1MiB；跳过 `readStart` 之前的行，读到 `readEnd` 后继续扫描以统计 `moreAfter`。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `tools.read_file.max_lines` | 500 | 单次最多返回行数 |
| `tools.read_file.max_bytes` | 2097152 | 文件大小上限（字节） |

## 权限与安全

- **PermissionLevel**：`Low`
- 只读；路径不得逃出工作区（含符号链接解析，见 `permission.Engine`）

## 设计思想

- **分页优先**：鼓励模型用 `offset`/`limit` 分段读大文件，而不是依赖截断尾巴。
- **行号前缀**：输出带 `N|` 前缀，便于与 `apply_patch` 的 `@@` 上下文及 TUI 引用对齐。
- **先 Stat 后读**：超大文件在打开前拒绝，保护内存与 token 预算。

## 相关代码

- [`read_file.go`](read_file.go)
- [`read_file_test.go`](read_file_test.go)
- [`display.go`](../../display.go) — `FormatReadFileDisplay`、`ReadFileLineRange`、`AppendReadFileLineRange`
