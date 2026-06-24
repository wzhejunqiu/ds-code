# glob

## 概述

按 glob 模式在工作区内查找**文件**路径（不含目录），后端为 **ripgrep 15.1.0 `--files`**（bundled 或 system/path）。内建 `glob` 工具，**非** shell `find`。

**面向大模型**：凡按文件名模式找路径的任务必须调用本 `glob` 工具，**禁止**通过 `bash` 执行 `find`；本工具已针对权限、搜索跳过目录、敏感路径与二进制过滤优化。

返回路径一律**相对项目根**（正斜杠），可直接传给 `read_file` / `grep`；**不是**相对 `path` 搜索目录。

## 注册与可见性

| 模式                    | 注册                    |
| ----------------------- | ----------------------- |
| plan / agent / subagent | `register.ExploreTools` |

## 参数 Schema（输入）

| 字段      | 类型   | 必填 | 说明                                                |
| --------- | ------ | ---- | --------------------------------------------------- |
| `pattern` | string | 是   | Glob 模式（rg `--glob`），如 `**/*.go`、`*_test.go` |
| `path`    | string | 否   | 搜索根目录（**必须是目录**），默认 `.`              |

### 输入校验

| 条件                                    | 行为                                                               |
| --------------------------------------- | ------------------------------------------------------------------ |
| `pattern` 为空                          | 错误：`pattern 为必填项`（`builtin.ErrPatternRequired`）           |
| **未**传 `path`（或 JSON 无 `path` 键） | `searchPath = "."`；**不做** stat / 目录类型 / `CheckReadablePath` |
| **显式**传 `path`（含 `"path":""`）     | 见下表                                                             |

#### 显式 `path` 校验

| 条件                     | 行为                                                    |
| ------------------------ | ------------------------------------------------------- |
| UNC（`\\` 或 `//` 开头） | 跳过 stat 与读权限检查（防 NTLM 凭证泄露），直接交给 rg |
| 路径越界 workspace       | 错误：`permission denied: path outside workspace: …`    |
| 目录不存在               | 错误：`目录不存在: <path>（当前工作目录: <cwd>）`       |
| 路径为文件               | 错误：`path 必须是目录，不能是文件: <path>`             |
| 敏感目录（S3）           | 错误：`CheckReadablePath` 拒绝                          |

## 用法示例（输入）

```json
{"pattern": "**/*.go", "path": "internal/tool/builtin"}
```

```json
{"pattern": "docs/*.md"}
```

```json
{"pattern": "*_test.go", "path": "internal/pkg"}
```

## 返回格式（输出，给大模型）

**结构**：`Found N files` + 相对项目根路径列表 + 可选截断脚标。

```
Found 2 files
internal/tool/builtin/glob/glob.go
internal/tool/builtin/grep/grep.go
```

| 场景   | 输出                                                                                       |
| ------ | ------------------------------------------------------------------------------------------ |
| 无匹配 | `Found 0 files`                                                                            |
| 截断   | `Found N` 为**本次展示**条数；正文为截断后路径 + `（结果已截断，请使用更具体的 path 或 pattern）` |

示例（全量 5 匹配，`max_results: 2`）：

```
Found 2 files
f00.go
f01.go
（结果已截断，请使用更具体的 path 或 pattern）
```

**无分页**：工具不提供 `limit`/`offset` 参数；超限时应收窄 `pattern` 或 `path` 重试。

### 相对路径契约

1. **基准**：相对**项目根**（`perm.Workspace`），非相对 `path`。
   例：`path: "internal/pkg"` → `internal/pkg/a.go`，**不是** `a.go`。
2. **格式**：`/` 分隔；无 `./` 前缀；无 workspace 绝对路径泄漏。
3. **排序**：按文件 `ModTime` 降序（相同时路径升序）。

## 实现

| 文件                           | 职责                                        |
| ------------------------------ | ------------------------------------------- |
| [`glob.go`](glob.go)           | `GlobTool`、`Schema`、`Execute`             |
| [`format_output.go`](format_output.go) | `formatGlobOutput`、截断脚标                |
| [`ripgrep.go`](ripgrep.go)     | 参数构建、exec、`--files` 解析、postProcess |
| [`text.go`](text.go)           | `RenderDesc()`、Schema 常量                 |
| [`usage.prompt`](usage.prompt) | LLM 可见工具说明                            |
| [`../rgutil/`](../rgutil/)     | 与 grep 共享的路径/format/exec 工具         |

- ripgrep `--files --glob '<pattern>'` 列文件；stdout 经 `rgutil.RelPathFromWorkspace` 归一化
- 敏感路径、`skip_dirs`、`.git` 通过 `--glob '!...'` 排除（与 grep 一致）
- post-filter：`SkipSensitiveAbs` + `textfile.IsSearchable`；`include_hidden: false` 时过滤点分路径段
- `path=.git` 早退 `Found 0 files`；rg 退出码 0/1 为成功；超时返回 `glob 搜索超时`

## 配置项

| 键                              | 默认          | 说明                                   |
| ------------------------------- | ------------- | -------------------------------------- |
| `tools.glob.max_results`        | **100**       | 最多返回文件路径数                     |
| `tools.glob.respect_gitignore`  | **false**     | `true` 时遵循 `.gitignore`             |
| `tools.glob.include_hidden`     | **true**      | `false` 时不返回隐藏路径段（`.` 前缀） |
| `tools.grep.binary` / `timeout` | bundled / 20s | ripgrep 二进制与超时（glob 复用）      |

## 权限与安全

- **PermissionLevel**：`Low`
- 只读；显式 `path` 时 `CheckReadablePath`（UNC 除外）

## 相关代码与测试

| 文件                                             | 组别       | 覆盖                                                    |
| ------------------------------------------------ | ---------- | ------------------------------------------------------- |
| [`format_output_test.go`](format_output_test.go) | **G1–G7**  | 输出格式单元测试（`Found N files`、截断脚标、相对路径） |
| [`input_test.go`](input_test.go)                 | **H1–H7**  | 输入解析、`validateExplicitPath`、隐藏路径判定          |
| [`io_test.go`](io_test.go)                       | **G8–G15** | 集成 I/O：精确 JSON 输入 → 输出/错误                    |
| [`glob_test.go`](glob_test.go)                   | **B***     | 行为集成（安全、skip_dirs、gitignore、mtime 等）        |
| [`ripgrep_test.go`](ripgrep_test.go)             | **C1**     | `buildGlobRipgrepArgs` 参数快照                         |
| [`text_test.go`](text_test.go)                   | —          | `usage.prompt` 渲染无模板泄漏                           |
| [`display.go`](../../display.go)                 | **D***     | TUI：`AppendPathResultSuffix` 解析 `Found N files`      |

**自动化**：`go test ./internal/tool/builtin/glob/...`

### 测试用例索引

#### G 组 — 输出格式

| ID      | 输入 / 条件                                    | 期望输出                          |
| ------- | ---------------------------------------------- | --------------------------------- |
| G1      | 1 个路径                                       | `Found 1 files\ninternal/foo.go`  |
| G2      | 2 个路径                                       | `Found 2 files\n…`                |
| G3      | 无路径                                         | `Found 0 files`                   |
| G4      | 全量 5、`max_results` 2                        | `Found 2 files` + 2 行 + 截断脚标 |
| G5      | `limit` 0（不截断）                            | 无截断脚标                        |
| G6      | 子目录路径                                     | `internal/pkg/a.go`（相对项目根） |
| G7      | 路径无 `./` 前缀                               | `Found 1 files\nfoo.go`           |
| G8–G12  | 集成：单文件/多文件/无匹配/path 收窄/隐式 path | 精确字符串匹配                    |
| G13–G15 | 集成：空 pattern / path 为文件 / path 不存在   | 对应错误文案                      |

#### H 组 — 输入校验

| ID  | 条件                    | 期望                 |
| --- | ----------------------- | -------------------- |
| H1  | JSON 无 `path` 键       | `explicitPath=false` |
| H2  | `"path":""`             | `explicitPath=true`  |
| H3  | `"path":"internal/pkg"` | 解析正确             |
| H4  | 显式 path 目录不存在    | `目录不存在`         |
| H5  | 显式 path 为文件        | `必须是目录`         |
| H6  | 显式 path 为有效目录    | 通过                 |
| H7  | UNC 路径                | 跳过 stat，通过      |
