# glob

## 概述

按 glob 模式在工作区内查找**文件**路径（不含目录），返回相对项目根的路径列表。

## 注册与可见性

| 模式 | 注册 |
|------|------|
| plan / agent / task 子代理 | `RegisterExploreTools` |

## 参数 Schema

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `pattern` | string | 是 | Glob 模式，如 `**/*.go`、`*_test.go` |
| `path` | string | 否 | 搜索基准目录，默认 `.` |

## 用法示例

```json
{"pattern": "**/*.go", "path": "internal/tool/builtin"}
```

```json
{"pattern": "docs/*.md"}
```

## 返回格式

- 每行一个相对路径
- 无匹配：`No files matched.`
- 达到上限：末行 `... truncated at N matches`

## 实现细节

源文件：[`glob.go`](../glob.go)

### 两种匹配路径

| 模式 | 实现 |
|------|------|
| 不含 `**` | `filepath.Glob(filepath.Join(root, pattern))` |
| 含 `**` | 自定义 `globDoubleStar`：`WalkDir` + 后缀/`filepath.Match` |

### 后处理

对每个候选绝对路径：

1. `validateGlobMatches`：`EvalSymlinks` 后 `EnsureAbsUnderWorkspace`，防止 glob 逃出工作区。
2. 跳过敏感路径、`.gitignore` 项、**目录**（只保留文件）。
3. 计数达到 `max_results` 后截断。

`globDoubleStar` 内部预取 `limit*4` 条候选再过滤，以平衡性能与结果量。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `tools.glob.max_results` | 200 | 最多返回文件路径数 |

`list_dir` 复用同一 `max_results` 作为目录项上限。

## 权限与安全

- **PermissionLevel**：`Low`
- Glob 穿越符号链接或 `..` 时在 `validateGlobMatches` 阶段拒绝（`GlobOutsideWorkspaceError`）

## 设计思想

- **按名找文件**：与 `grep`（按内容）互补；`**` 支持递归，贴近开发者习惯。
- **安全优先**：匹配后再做 workspace 边界校验，而非仅信任 glob 字符串。
- **只列文件**：目录由 `list_dir` 负责，职责分离。

## 相关代码

- [`glob.go`](../glob.go)
- [`glob_test.go`](../glob_test.go)
- [`display.go`](../../display.go) — `FormatGlobDisplay`、`AppendPathResultSuffix`
