# grep

## 概述

在工作区内用正则搜索文件**行内容**，后端为 **ripgrep 15.1.0**（bundled 或 system/path）。内建 `grep` 工具，**非** shell `rg`。

**面向大模型**：凡内容搜索任务必须调用本 `grep` 工具，**禁止**通过 `shell` 执行 `grep` 或 `rg`；本工具已针对权限、搜索跳过目录、敏感路径与二进制过滤优化。

## 注册与可见性

| 模式                    | 注册                    |
| ----------------------- | ----------------------- |
| plan / agent / subagent | `register.ExploreTools` |

## 参数 Schema

| 字段             | 类型    | 必填 | 说明                                                            |
| ---------------- | ------- | ---- | --------------------------------------------------------------- |
| `pattern`        | string  | 是   | 正则（**ripgrep / Rust regex** 语法），最长 512 字符            |
| `path`           | string  | 否   | 目录或单文件（rg PATH），默认 `.`                               |
| `glob`           | string  | 否   | 文件 glob 过滤（rg `--glob`），如 `*.go`                        |
| `output_mode`    | string  | 否   | 见下表；默认 `files_with_matches`                               |
| `-B`             | number  | 否   | 匹配前上下文行数（`content`）                                   |
| `-A`             | number  | 否   | 匹配后上下文行数（`content`）                                   |
| `-C` / `context` | number  | 否   | 匹配前后上下文行数（`content`）                                 |
| `-n`             | boolean | 否   | 显示行号（`content`，默认 true）                                |
| `-i`             | boolean | 否   | 大小写不敏感                                                    |
| `type`           | string  | 否   | rg `--type`（如 `go`、`js`）                                    |
| `head_limit`     | number  | 否   | 输出条数上限（三模式通用）；默认 `tools.grep.head_limit`（250） |
| `offset`         | number  | 否   | 分页偏移，默认 0                                                |
| `multiline`      | boolean | 否   | 多行模式（`-U --multiline-dotall`）                             |

### `output_mode` 取值

| 值                   | 说明                            |
| -------------------- | ------------------------------- |
| `files_with_matches` | 摘要 + 相对路径列表（**默认**） |
| `content`            | 匹配/上下文正文行               |
| `count`              | `path:count` 列表 + 全量汇总    |

## 用法示例

```json
{"pattern": "ExploreTools", "path": "internal/tool/register"}
```

```json
{"pattern": "func.*Execute", "path": "internal/tool/builtin/read_file", "glob": "*.go", "output_mode": "content"}
```

```json
{"pattern": "needle", "path": "pkg", "glob": "*.go", "output_mode": "count", "head_limit": 10}
```

## 返回格式（给大模型）

### `files_with_matches`（默认）

**结构**：`Found N files` + 路径列表 + 可选分页脚标。

```
Found 2 files
internal/foo.go
pkg/bar.go
```

| 场景   | 输出                                                                                                  |
| ------ | ----------------------------------------------------------------------------------------------------- |
| 无匹配 | `Found 0 files`                                                                                       |
| 分页   | 摘要 N 为**全量**文件数；正文为当前页路径 + `[Showing results with pagination = limit: L, offset: O]` |

示例（全量 5 文件，`head_limit: 2`）：

```
Found 5 files
f00.txt
f01.txt
[Showing results with pagination = limit: 2, offset: 0]
```

### `content`

**结构**：正文行 + 可选分页脚标（无顶部摘要）。

| 类型    | `-n: true`（默认） | `-n: false` |
| ------- | ------------------ | ----------- |
| match   | `path:line:text`   | `path:text` |
| context | `path:line-text`   | `path-text` |

```
internal/foo.go:10-import "fmt"
internal/foo.go:11:func main() {
internal/foo.go:12-    fmt.Println("hi")
```

| 场景   | 输出                |
| ------ | ------------------- |
| 无匹配 | 空字符串 `""`       |
| 分页   | 当前页行 + 分页脚标 |

### `count`

**结构**：`path:count` 列表 + `Found X occurrences across Y files` + 可选分页脚标。

```
internal/foo.go:3
pkg/bar.go:1
Found 4 occurrences across 2 files
```

| 字段         | 含义                          |
| ------------ | ----------------------------- |
| `path:count` | 该文件匹配行数（当前页）      |
| `X`          | **全量**匹配行总数            |
| `Y`          | **全量**至少命中 1 行的文件数 |

| 场景   | 输出                                 |
| ------ | ------------------------------------ |
| 无匹配 | `Found 0 occurrences across 0 files` |
| 分页   | 部分 `path:count` + 全量 X/Y + 脚标  |

结果按**命中文件最近修改时间**降序（ds-code 扩展）；`content` 同文件内按行号升序。

## 排序

postProcess 在 `head_limit`/`offset` 切片前，按文件 mtime **降序**排列（相同时路径升序）。

## 实现

| 文件                                   | 职责                                    |
| -------------------------------------- | --------------------------------------- |
| [`grep.go`](grep.go)                   | `GrepTool`、`Schema`、`Execute`         |
| [`ripgrep.go`](ripgrep.go)             | 参数构建、exec、JSON 解析、postProcess  |
| [`format_output.go`](format_output.go) | §3.4 纯文本格式化                       |
| [`rgbin/rgbin.go`](rgbin/rgbin.go)     | embed `rg.tar.gz` → `~/.ds-code/bin/rg` |
| [`text.go`](text.go)                   | `RenderDesc()`、Schema 常量             |
| [`usage.prompt`](usage.prompt)         | LLM 可见工具说明                        |

- `path` 为绝对路径时，输出统一为项目根相对路径（`/` 分隔）
- 敏感路径、`.env`、密钥等通过 `--glob '!...'` 排除
- `tools.search.skip_dirs` 在 `path=.` 时通过 `--glob` 排除；显式 `path` 可进入
- 不搜索 `.git`（宽泛 `path` 加 `!.git/**`；`path=.git` 直接返回无匹配），避免大仓库无谓遍历
- 非 searchable 扩展名（如 `.png`）在 JSON 解析后过滤
- rg 退出码 0/1 为成功；超时返回 `grep 搜索超时`

## 配置项

| 键                             | 默认        | 说明                                            |
| ------------------------------ | ----------- | ----------------------------------------------- |
| `tools.grep.head_limit`        | **250**     | per-call 未传 `head_limit` 时的回退上限         |
| `tools.grep.timeout`           | **20s**     | ripgrep 子进程超时                              |
| `tools.grep.binary`            | **bundled** | `bundled` \| `system` \| `path`                 |
| `tools.grep.binary_path`       | —           | `binary=path` 时必填                            |
| `tools.grep.respect_gitignore` | **false**   | `true` 时遵循 `.gitignore`（默认 Agent 不遵循） |

bundled 二进制解压至 `~/.ds-code/bin/rg`（SHA256 校验，篡改自愈）。

## 权限与安全

- **PermissionLevel**：`Low`
- 只读；`CheckReadablePath` 校验搜索根路径

## 相关代码与测试

- [`format_output_test.go`](format_output_test.go) — A1–A20 输出格式单元测试
- [`grep_test.go`](grep_test.go) — B1–B27 集成测试
- [`ripgrep_test.go`](ripgrep_test.go)、[`rgbin/rgbin_test.go`](rgbin/rgbin_test.go)
- [`display.go`](../../display.go) — TUI：`Found N files` / `Found X occurrences` 解析
