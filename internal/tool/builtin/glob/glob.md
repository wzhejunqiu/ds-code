# glob

## 概述

按 glob 模式在工作区内查找**文件**路径（不含目录），返回**相对项目根**的路径列表（可直接用于 `read_file` / `grep`）。结果按修改时间降序排列。自动跳过常见二进制文件（扩展名、`mimetype` magic、NUL 字节启发式）。

## 注册与可见性

| 模式                       | 注册                   |
| -------------------------- | ---------------------- |
| plan / agent / subagent | `register.ExploreTools` |

## 参数 Schema

| 字段      | 类型   | 必填 | 说明                                 |
| --------- | ------ | ---- | ------------------------------------ |
| `pattern` | string | 是   | Glob 模式，如 `**/*.go`、`*_test.go` |
| `path`    | string | 否   | 搜索基准目录，默认 `.`               |

## 用法示例

```json
{"pattern": "**/*.go", "path": "internal/tool/builtin"}
```

返回路径形如 `internal/tool/builtin/glob/glob.go`（相对项目根，而非相对 `path`）。

```json
{"pattern": "docs/*.md"}
```

## 返回格式

- 每行一个相对项目根的路径
- 按文件 `ModTime` 降序（相同时路径升序）
- 无匹配：`无匹配文件`
- 达到上限：末行 `... 已截断，共 N 条结果`

## 实现细节

源文件：[`glob.go`](glob.go)

### 匹配与收集

[`builtin.CollectGlobPattern`](../filecandidate.go)：`MatchFiles`（无预截断）→ `ValidateGlobMatches` → [`MakeFileCandidate`](../filecandidate.go)（路径相对工作区根）→ `searchskip` 过滤。

[`builtin.SortByModTimeDesc`](../sort.go) 排序后取前 `max_results` 条。

### 过滤

跳过敏感路径、`searchskip` 目录、**目录**、[`textfile.IsSearchable`](../../textfile/textfile.go) 判定的二进制。

## 配置项

| 键                       | 默认 | 说明               |
| ------------------------ | ---- | ------------------ |
| `tools.glob.max_results` | 100  | 最多返回文件路径数 |

`list_dir` 复用同一 `max_results` 作为目录项上限。

## 权限与安全

- **PermissionLevel**：`Low`
- Glob 穿越符号链接或 `..` 时在 `ValidateGlobMatches` 阶段拒绝（`GlobOutsideWorkspaceError`）

## 设计思想

- **按名找文件**：与 `grep`（按内容）互补；`**` 支持递归，贴近开发者习惯。
- **路径可复用**：与 `grep` 输出格式一致，便于链式 `read_file`。
- **mtime 优先**：最近改动的文件排在前面。
- **安全优先**：匹配后再做 workspace 边界校验。
- **只列文本向文件**：二进制结果不返回。
- **只列文件**：目录由 `list_dir` 负责。

## 相关代码

- [`glob.go`](glob.go)
- [`glob_test.go`](glob_test.go)
- [`filecandidate.go`](../filecandidate.go) — 与 grep 共用
- [`sort.go`](../sort.go)
- [`globmatch`](../../globmatch)
- [`textfile`](../../textfile)
- [`display.go`](../../display.go) — `FormatGlobDisplay`、`AppendPathResultSuffix`
