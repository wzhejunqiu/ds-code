# list_dir

## 概述

列出工作区内某一目录的**直接子项**（非递归），用于快速查看目录结构。

## 注册与可见性

| 模式 | 注册 |
|------|------|
| plan / agent / task 子代理 | `RegisterExploreTools` |

## 参数 Schema

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `path` | string | 否 | 目录路径，默认 `.` |

## 用法示例

```json
{"path": "internal/tool/builtin"}
```

```json
{}
```

## 返回格式

- 子目录：`name/`
- 文件：`name`
- 空目录：`(empty)`
- 达到上限：`... truncated at N entries`

不返回隐藏规则外的 `.git`（显式跳过）。

## 实现细节

源文件：[`list_dir.go`](../list_dir.go)

1. `Perm.CheckReadablePath` 解析目录。
2. `os.ReadDir` 读取一层。
3. 过滤：`.git`、`.gitignore` 忽略项、敏感绝对路径。
4. 条目数上限与 `tools.glob.max_results` 相同（默认 100）。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `tools.glob.max_results` | 100 | 最多列出的条目数（与 glob 共用） |

## 权限与安全

- **PermissionLevel**：`Low`
- 只读；无法列出工作区外路径

## 设计思想

- **浅层列举**：避免 `WalkDir` 整树扫描，适合「看看这个文件夹有什么」。
- **与 glob 配对**：`list_dir` 导航 + `glob` 按模式批量找文件。
- **输出极简**：仅名称，不含 size/mtime，减少 token。

## 相关代码

- [`list_dir.go`](../list_dir.go)
- [`glob_test.go`](../glob_test.go)（与 list_dir 联合测试）
- [`display.go`](../../display.go) — `AppendPathResultSuffix`
