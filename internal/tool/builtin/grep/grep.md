# grep

## 概述

在工作区内用正则搜索文件**行内容**，返回 `path:line:content` 形式的匹配列表。实现为纯 Go 遍历，不依赖外部 `rg`。

**面向大模型**：凡内容搜索任务必须调用本 `grep` 工具，**禁止**通过 `shell` 执行 `grep` 或 `rg`；本工具已针对权限、`.gitignore` 与二进制过滤优化。

## 注册与可见性

| 模式                       | 注册                   |
| -------------------------- | ---------------------- |
| plan / agent / task 子代理 | `RegisterExploreTools` |

## 参数 Schema

| 字段      | 类型   | 必填 | 说明                                                                 |
| --------- | ------ | ---- | -------------------------------------------------------------------- |
| `pattern` | string | 是   | 正则表达式（Go `regexp` 语法），匹配**行内容**，最长 512 字符        |
| `path`    | string | 否   | 目录、单文件或 glob（如 `*.go`、`internal/**/*.go`），默认 `.`       |

## 用法示例

```json
{"pattern": "RegisterExploreTools", "path": "internal/tool"}
```

```json
{"pattern": "func.*Execute", "path": "internal/tool/builtin/read_file.go"}
```

```json
{"pattern": "needle", "path": "pkg/*.go"}
```

## 返回格式

- 每行匹配：`path/to/file:行号:trimmed 行内容`
- 无匹配：`no matches`
- 达到 `head_limit`：`... truncated at N matches`
- 结果按**命中文件最近修改时间**降序（同文件内按行号升序）

## 实现细节

源文件：[`grep.go`](grep.go)

### 候选收集

| `path` 类型 | 实现 |
| ----------- | ---- |
| 精确路径（无 `*`/`?`/`[`） | `CheckReadablePath` → 单文件或 `WalkDir` 递归 |
| glob 路径 | [`globmatch.SplitPath`](../../globmatch/globmatch.go) + [`MatchFiles`](../../globmatch/globmatch.go) |

过滤：`.git`、敏感路径、`.gitignore`、大于 2MiB、[`textfile.IsSearchable`](../../textfile/textfile.go) 二进制跳过。

### 搜索

1. 候选按 `ModTime` 降序（相同时按路径升序）。
2. 按批并发（`min(8, NumCPU)`）`ReadFile` + 逐行 `regexp.MatchString`。
3. 凑满 `head_limit` 后停止后续批次。

## 配置项

| 键                      | 默认 | 说明               |
| ----------------------- | ---- | ------------------ |
| `tools.grep.head_limit` | 200  | 最多返回的匹配条数 |

## 权限与安全

- **PermissionLevel**：`Low`
- 只读；与 `read_file` 相同的路径校验；glob path 做 workspace 边界校验

## 设计思想

- **可预测、零依赖**：不调用系统 `grep`/`rg`，行为在沙箱内完全可控。
- **与探索工具一致**：尊重 `.gitignore`，跳过常见二进制，避免 `node_modules` 等污染结果。
- **mtime 优先**：最近改动的文件匹配行排在前面，便于 Agent 优先关注活跃代码。
- **硬上限**：匹配数、文件大小、模式长度均有界，防止 ReDoS 与 token 爆炸（复杂正则仍需谨慎）。

## 相关代码

- [`grep.go`](../grep.go)
- [`grep_test.go`](../grep_test.go)
- [`globmatch`](../../globmatch) — 与 `glob` 工具共用的路径匹配
- [`textfile`](../../textfile) — 二进制/文本判定
- [`display.go`](../../display.go) — `FormatGrepDisplay`、`AppendGrepResultSuffix`
