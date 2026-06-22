# read_file

## 概述

在工作区内读取单个文本文件内容，支持按行范围分页，避免大文件一次性灌入上下文。

LLM 可见 `Description` 由 [`usage.prompt`](usage.prompt) 经 `RenderDesc()` 渲染；参数说明见 [`text.go`](text.go) 的 `Schema*`。

## 注册与可见性

| 模式                    | 注册                    |
| ----------------------- | ----------------------- |
| plan / agent / subagent | `register.ExploreTools` |

## 参数 Schema

| 字段       | 类型    | 必填 | 说明                                                  |
| ---------- | ------- | ---- | ----------------------------------------------------- |
| `filepath` | string  | 是   | 文件的绝对路径（LLM 提示要求绝对路径；实现层仍解析相对路径） |
| `offset`   | integer | 否   | 起始行号（1-based），默认从第 1 行                    |
| `limit`    | integer | 否   | 最多读取行数；省略时从文件开头读取（最多 max_lines 行） |

## 用法示例

```json
{"filepath": "/abs/path/to/internal/tool/builtin/read_file.go"}
```

```json
{"filepath": "/abs/path/to/docs/v0.1.0/CONFIG.md", "offset": 100, "limit": 50}
```

## 返回格式

- 每行：`{行号}|{行内容}\n`
- 若 `limit` 超过配置上限：末尾附加 `... 已按 {N} 行截断；请调整 offset/limit 继续`
- 若范围内之后还有行：`... 还有 {N} 行未显示`
- 若 `offset` 超出文件长度：`（空：offset {N} 超出文件长度 {M}）`

TUI 通过 `FormatReadFileDisplay` / `ReadFileLineRange` 解析行号范围展示。

## 实现细节

源文件：[`read_file.go`](read_file.go)

1. **权限**：`Perm.CheckReadablePath` 解析路径并拒绝敏感文件；`permission.Engine` 从 `filepath` 参数取路径。
2. **体积预检**：`os.Stat` 检查文件字节数 ≤ `tools.read_file.max_bytes`（默认 2MiB），超限直接报错，不读入内存。
3. **文本判定**：`textfile.IsTextFile` 在打开文件前拒绝二进制/媒体文件（PNG 等）；空文件与 MCP spill `.txt` 允许。
4. **行范围**：`resolveReadOffsetLimit` 计算 `[readStart, readEnd]`；`limit` 会被钳制到 `max_lines`。
5. **扫描**：`bufio.Scanner`，单行 buffer 上限 1MiB；跳过 `readStart` 之前的行，读到 `readEnd` 后继续扫描以统计 `moreAfter`。

## 配置项

| 键                          | 默认    | 说明                                         |
| --------------------------- | ------- | -------------------------------------------- |
| `tools.read_file.max_lines` | 2000    | 单次最多返回行数（默认读取整个文件时的上限） |
| `tools.read_file.max_bytes` | 2097152 | 文件大小上限（字节）                         |

## 权限与安全

- **PermissionLevel**：`Low`
- 只读；工作区内路径经 `CheckReadablePath` → `ResolveAccessPath`（S2+S3）
- **实现层例外**：`~/.ds-code/projects/<当前 project_id>/` 下 regular file 经 `resolveProjectDataRead` 放行（MCP / 子代理 spill、`sessions.db` 等）；project 数据目录路径跳过 `IsTextFile` 扩展名 blocklist
- LLM 可见 Description **不**枚举可读范围；spill 路径引导见 tool result 中的 `SavedResultHint`

## 设计思想

- **默认全读**：仅传 `filepath` 时从文件开头读取（最多 max_lines 行）；超大文件用 `offset`/`limit` 分段。
- **行号前缀**：输出带 `N|` 前缀，便于与 `apply_patch` 的 `@@` 上下文及 TUI 引用对齐。
- **先 Stat 后读**：超大文件在打开前拒绝，保护内存与 token 预算。
- **仅文本**：工作区内二进制与媒体文件由 `IsTextFile` 拒绝；project 数据目录 regular file 不受扩展名 blocklist 限制

## 相关代码

- [`read_file.go`](read_file.go)
- [`usage.prompt`](usage.prompt) — LLM 可见 Description 正文
- [`text.go`](text.go) — `RenderDesc()`、`Schema*`、`Err*`、`Result*`
- [`text_test.go`](text_test.go)
- [`read_file_test.go`](read_file_test.go)
- [`display.go`](../../display.go) — `FormatReadFileDisplay`、`ReadFileLineRange`、`AppendReadFileLineRange`
