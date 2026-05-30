# apply_patch

## 概述

应用 **Codex 风格** 的 patch 文档，对已有文件做增量编辑、新增或删除文件。是修改已有代码的首选方式；失败时原子回滚。

## 注册与可见性

| 模式 | 注册 |
|------|------|
| agent | `RegisterWrite` |
| plan | **不注册** |

可通过 `tools.defer_builtin` 延迟暴露完整 schema；需先调用 `tool_search`。

## 参数 Schema

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `patch` | string | 是 | 完整 patch 文本（`*** Begin Patch` … `*** End Patch`） |

## Patch 格式（Codex）

```text
*** Begin Patch
*** Update File: path/to/file.go
@@
 context line (optional)
-old line
+new line
*** Add File: new/file.go
+line1
+line2
*** Delete File: obsolete.go
*** End Patch
```

要点：

- 更新块以 `@@` 开始，其后为上下文与 `-`/`+` 行（unified 风格，非标准 unified diff 文件头）。
- 支持 `*** Move to:` 重命名。
- `*** End of File` 标记 EOF 相关块。
- 可用 `<<'EOF'` heredoc 包裹（解析器 `unwrapHeredoc`）。

完整语法见 [`patch/parser.go`](../../../patch/parser.go)。

## 用法示例

```json
{
  "patch": "*** Begin Patch\n*** Update File: sample.go\n@@\n func Hello() string {\n \treturn \"hello\"\n }\n+// harness\n*** End Patch\n"
}
```

集成测试剧本：`docs/TUI_TCASE_SCRIPTS.md` — `tool-patch-single`、`tool-patch-multi`。

## 返回格式

成功：`applied: {summary}`（由 `patch/apply` 生成的变更摘要）

失败：解析错误、路径越界、上下文不匹配、变更行数超限等，**不写盘**。

## 实现细节

源文件：[`apply_patch.go`](apply_patch.go) → [`patch/apply/`](../../../patch/apply/)

1. `patch.Parse(patchText, workspace)` 解析为 `[]FileChange`。
2. 若配置 `max_changed_lines > 0`，先 `patch.CountChangedLines` 校验总量。
3. 对每个文件：备份 → 应用 add/update/delete/move → 记录已写路径。
4. 任一步失败：`defer` 中 `rollback()` 恢复备份或删除新建文件。

路径解析通过 `t.Perm.ResolvePath` 注入，并 `EnsureAbsUnderWorkspace`。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `tools.apply_patch.max_changed_lines` | 2000 | 单 patch 允许变更的行数上限 |

## 权限与安全

- **PermissionLevel**：`High`
- 写操作；Runner 可在执行前创建 checkpoint（见 `docs/SECURITY.md`）
- 不实现独立 `edit_file` 工具，编辑语义统一于此

## 设计思想

- **对齐 Codex**：与 OpenAI Codex CLI 的 patch 格式一致，便于迁移 prompt 与模型习惯。
- **事务性**：多文件 patch 全成功或全回滚，避免半拉子工程状态。
- **优于 write_file 改已有文件**：保留上下文块匹配，减少误覆盖整文件。

## 相关代码

- [`apply_patch.go`](apply_patch.go)
- [`parser.go`](../../../patch/parser.go)、[`apply/apply.go`](../../../patch/apply/apply.go)
- [`display.go`](../../display.go) — `ApplyPatchStarts`、`ApplyPatchFileDisplay`
- [`stats.go`](../../../patch/stats.go) — 变更行数统计（TUI `+N`/`-N`）
