# grep

## 概述

在工作区内用正则搜索文件**行内容**，通过 `output_mode` 控制返回形态。实现为纯 Go 遍历，不依赖外部 `rg`。

**面向大模型**：凡内容搜索任务必须调用本 `grep` 工具，**禁止**通过 `shell` 执行 `grep` 或 `rg`；本工具已针对权限、`.gitignore` 与二进制过滤优化。

## 注册与可见性

| 模式                       | 注册                   |
| -------------------------- | ---------------------- |
| plan / agent / task 子代理 | `RegisterExploreTools` |

## 参数 Schema

| 字段          | 类型   | 必填 | 说明                                                           |
| ------------- | ------ | ---- | -------------------------------------------------------------- |
| `pattern`     | string | 是   | 正则表达式（Go `regexp` 语法），匹配**行内容**，最长 512 字符  |
| `path`        | string | 否   | 目录、单文件或 glob（如 `*.go`、`internal/**/*.go`），默认 `.` |
| `output_mode` | string | 否   | 输出模式，见下表；默认 `files_with_matches`                    |

### `output_mode` 取值

| 值                   | 说明                                          |
| -------------------- | --------------------------------------------- |
| `files_with_matches` | 每行一个相对路径（**默认**）                  |
| `content`            | 每行 `path:行号:trimmed 行内容`               |
| `count`              | 单个整数，全工作区总匹配行数（见 head_limit） |

## 用法示例

```json
{"pattern": "RegisterExploreTools", "path": "internal/tool"}
```

```json
{"pattern": "func.*Execute", "path": "internal/tool/builtin/read_file.go", "output_mode": "content"}
```

```json
{"pattern": "needle", "path": "pkg/*.go", "output_mode": "count"}
```

## 返回格式

### `files_with_matches`（默认）

- 每行一个相对路径
- 无匹配：`无匹配`
- 达到 `head_limit`（文件数）：末尾 `... 已截断，共 N 个文件`

### `content`

- 每行：`path/to/file:行号:trimmed 行内容`
- 无匹配：`无匹配`
- 达到 `head_limit`（匹配行数）：末尾截断提示

### `count`

- 单个整数，如 `42`；无匹配为 `0`
- **不受** `head_limit` 限制，扫描全部候选后返回完整总数
- 无截断后缀

结果按**命中文件最近修改时间**降序；`content` 模式同文件内按行号升序。

## 实现细节

源文件：[`grep.go`](grep.go)

### 候选收集

| `path` 类型                | 实现                                                                                                 |
| -------------------------- | ---------------------------------------------------------------------------------------------------- |
| 精确路径（无 `*`/`?`/`[`） | `CheckReadablePath` → 单文件或 `WalkDir` 递归                                                        |
| glob 路径                  | [`globmatch.SplitPath`](../../globmatch/globmatch.go) + [`MatchFiles`](../../globmatch/globmatch.go) |

过滤：`.git`、敏感路径、`.gitignore`、大于 2MiB、[`textfile.IsSearchable`](../../textfile/textfile.go) 二进制跳过。文件路径相对项目根，由 [`builtin.MakeFileCandidate`](../filecandidate.go) 统一计算（与 `glob` 一致）。

### 搜索

1. 候选按 `ModTime` 降序（相同时按路径升序），[`builtin.SortByModTimeDesc`](../sort.go)。
2. 按批并发（`min(8, NumCPU)`）`ReadFile` + 逐行 `regexp.MatchString`。
3. `content` / `files_with_matches`：达到 `head_limit` 后停止；`count` 扫完全部候选。

## 配置项

| 键                      | 默认 | 说明                                                                  |
| ----------------------- | ---- | --------------------------------------------------------------------- |
| `tools.grep.head_limit` | 200  | `content` 最多匹配行数；`files_with_matches` 最多文件数；`count` 忽略 |

## 权限与安全

- **PermissionLevel**：`Low`
- 只读；与 `read_file` 相同的路径校验；glob path 做 workspace 边界校验

## 设计思想

- **可预测、零依赖**：不调用系统 `grep`/`rg`，行为在沙箱内完全可控。
- **与探索工具一致**：尊重 `.gitignore`，跳过常见二进制，避免 `node_modules` 等污染结果。
- **mtime 优先**：最近改动的文件排在前面，便于 Agent 优先关注活跃代码。
- **硬上限**：`content` / `files_with_matches` 受 `head_limit` 约束；`count` 仍受单文件大小等安全边界限制。

## 相关代码

- [`grep.go`](grep.go)
- [`grep_test.go`](grep_test.go)
- [`globmatch`](../../globmatch) — 与 `glob` 工具共用的路径匹配
- [`textfile`](../../textfile) — 二进制/文本判定
- [`display.go`](../../display.go) — `FormatGrepDisplay`、`AppendGrepResultSuffix`（TUI：`· N paths` / `· N matches`）
