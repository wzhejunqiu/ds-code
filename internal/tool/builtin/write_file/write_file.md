# write_file

## 概述

**新建**文件或**整文件覆盖**已有文件。修改已有文件内容应优先使用 `apply_patch`。

## 注册与可见性

| 模式 | 注册 |
|------|------|
| agent | `RegisterWrite` |
| plan | **不注册** |

可通过 `tools.defer_builtin` 延迟暴露完整 schema；需先调用 `tool_search`。

## 参数 Schema

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `path` | string | 是 | 相对项目根的路径 |
| `content` | string | 是 | 完整文件内容 |

## 用法示例

```json
{
  "path": "internal/foo/bar.go",
  "content": "package bar\n\nfunc Bar() {}\n"
}
```

## 返回格式

```text
wrote path/to/file (N bytes)
```

## 实现细节

源文件：[`write_file.go`](write_file.go)

1. `Perm.ResolvePath` 解析目标路径。
2. 若目标文件已存在，经 [`readgate`](../../readgate/gate.go) 校验须先 `read_file`（新建文件豁免）。
3. `os.MkdirAll` 创建父目录（权限 `0755`）。
4. `os.WriteFile` 写入，模式 `0644`。

无内置 diff、无备份逻辑（checkpoint 由 Runner 层负责）。

## 先读后写（read guard）

覆盖已有文件前，Runner 强制校验（与 `apply_patch` 共用 [`readgate`](../../readgate/gate.go)）：

1. **跨 sub-round**：目标文件须在本 session **更早的 sub-round** 内已成功 `read_file`（新建文件豁免）。
2. **同 sub-round**：同一条 assistant 回复中不能对**同一文件**既 `read_file` 又 `write_file`。
3. **子代理**：已读集合按 `session_id` 隔离；Fork 子 session 可从 seed 消息水合历史 read。
4. **实现**：[`readgate`](../../readgate/gate.go) + [`agent/readfile_gate.go`](../../../agent/readfile_gate.go)；`runToolCalls` 注入 sub-round 快照与同批 read 集合。

错误文案见 [`text.go`](text.go) 中 `ErrMustReadFirstFmt`、`ErrSameBatchReadWriteFmt`。

## LLM 描述（FR-0）

| 文件 | 职责 |
|------|------|
| [`usage.prompt`](usage.prompt) | Description 正文 |
| [`text.go`](text.go) | embed、`RenderDesc()`、`Err*`、`Result*` |
| [`write_file.go`](write_file.go) | `Description() { return RenderDesc() }` |

## 配置项

无专用 `tools.write_file.*` 键；受全局 `context.tool_result_max_chars` 等间接影响。

## 权限与安全

- **PermissionLevel**：`High`
- 可覆盖任意工作区内非敏感文件；敏感路径由 `ResolvePath` / 引擎规则拒绝

## 设计思想

- **简单写整文件**：适合新文件、生成配置、用户明确要求全文重写。
- **刻意不用于小改**：产品规则要求改已有文件走 `apply_patch`，降低误删未改动行的风险。
- **最小实现**：不做流式写入或部分更新，保持工具语义清晰。

## 相关代码

- [`write_file.go`](write_file.go)
- [`usage.prompt`](usage.prompt)、[`text.go`](text.go)
- [`display.go`](../../display.go) — `FormatWriteFileDisplay`
