# grep

## 概述

在工作区内用正则搜索文件**行内容**，返回 `path:line:content` 形式的匹配列表。实现为纯 Go 遍历，不依赖外部 `rg`。

## 注册与可见性

| 模式 | 注册 |
|------|------|
| plan / agent / task 子代理 | `RegisterExploreTools` |

## 参数 Schema

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `pattern` | string | 是 | 正则表达式（Go `regexp` 语法），最长 512 字符 |
| `path` | string | 否 | 搜索根目录或单文件，默认 `.` |

## 用法示例

```json
{"pattern": "RegisterExploreTools", "path": "internal/tool"}
```

```json
{"pattern": "func.*Execute", "path": "internal/tool/builtin/read_file.go"}
```

## 返回格式

- 每行匹配：`path/to/file:行号:trimmed 行内容`
- 无匹配：`no matches`
- 达到 `head_limit`：`... truncated at N matches`

## 实现细节

源文件：[`grep.go`](../grep.go)

1. `filepath.WalkDir` 自 `path` 解析后的绝对根递归。
2. **跳过**：`.git` 目录、敏感路径、`GitignoreMatcher` 忽略的文件。
3. **单文件限制**：大于 2MiB 的文件跳过；单行超过 64KiB 跳过。
4. 整文件读入内存后按 `\n` 分行，逐行 `regexp.MatchString`。
5. 匹配数达到 `head_limit` 时用 sentinel `errStopWalk` 终止遍历。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `tools.grep.head_limit` | 200 | 最多返回的匹配条数 |

## 权限与安全

- **PermissionLevel**：`Low`
- 只读；与 `read_file` 相同的路径校验

## 设计思想

- **可预测、零依赖**：不调用系统 `grep`/`rg`，行为在沙箱内完全可控。
- **与探索工具一致**：尊重 `.gitignore`，避免 `node_modules` 等污染结果。
- **硬上限**：匹配数、文件大小、模式长度均有界，防止 ReDoS 与 token 爆炸（复杂正则仍需谨慎）。

## 相关代码

- [`grep.go`](../grep.go)
- [`grep_test.go`](../grep_test.go)
- [`display.go`](../../display.go) — `FormatGrepDisplay`、`AppendGrepResultSuffix`
