# v0.1.4 内建工具提示词改写台账

> 版本：v0.1.4  
> 状态：规划中  
> 更新日期：2026-06-21  
> **维护说明**：每改一个工具，更新下表「状态」与「确认人」；草稿放在「待确认草稿」列或 PR 描述中，**未经你确认不得标为已定稿**。

## 已确认原则（2026-06-21）

| 项 | 你的决定 |
|----|----------|
| Description 风格 | **偏长**：像 `read_file` 那样分点写清用法与边界（各工具可长短略有差异） |
| 改写范围 | **全部 12 项**内建工具 + 共享 `builtin/text.go` Schema，本版都要改 |
| 「禁止 bash 绕行」 | **每个工具改时再定**，不预先统一套话 |
| 系统提示词 `prompt.md` | **纳入 v0.1.4**，但你 **逐段审定** 后才合入；与工具 Desc 重复处改时再决定保留层 |

## 改写对象

每个工具包含：

| 类型 | 代码位置 | 发给 LLM 的形式 |
|------|----------|-----------------|
| 工具描述 | `Desc*` / `Description()` | `tools[].description` |
| 参数说明 | `Schema*`、`builtin/text.go` 共享常量 | `tools[].parameters.properties.*.description` |
| 必填/错误文案 | `Err*`、`Result*` | 不发给 LLM（本表不跟踪，除非你要统一语气） |

## 工具清单与状态

| 工具 | Desc | Schema 字段 | 状态 | 备注 |
|------|------|-------------|------|------|
| `read_file` | `DescReadFile`（多行） | `SchemaOffset`、`SchemaLimitFmt`、`builtin.SchemaPathFileRelOrAbs` | **草稿** | 已有 Cursor 风格长文；目录列举用 `%s` 注入 bash 名 |
| `grep` | `DescGrep` | `SchemaRegexPattern`、`SchemaGrepPath`、`SchemaOutputMode` | 待你定稿 | 已改「禁止 bash 绕行」一句 |
| `glob` | `DescGlob` | `builtin.SchemaGlobPattern` 等 | 待你定稿 | |
| `list_dir` | `DescListDir` | `builtin.SchemaPathRelDefault` 等 | 待你定稿 | |
| `diagnostics` | `DescDiagnostics` | `SchemaSeverity`、`builtin.SchemaPathsRelRoot` | 待你定稿 | |
| `web_fetch` | `DescWebFetch` | `builtin.SchemaHTTPURL` | 待你定稿 | |
| `web_search` | `DescWebSearch` | `SchemaQuery` | 待你定稿 | 默认未注册 |
| `bash` | `DescShell` | `SchemaShellDescription`、`SchemaShellCommand`、background/job 等 | 待你定稿 | wire 名 bash；包名仍 shell |
| `apply_patch` | `DescApplyPatch` | `builtin.SchemaPatchBody` | 待你定稿 | |
| `write_file` | `DescWriteFile` | `builtin.SchemaFullFileContent` 等 | 待你定稿 | |
| `tool_search` | 目前在 `.go` 内联 | `tool_name` | 待你定稿 | 建议抽到 `text.go` |
| `agent` | `DescAgent` | `SchemaAgent*` 全套 | 待你定稿 | |
| 共享 | — | `builtin/text.go` 全部 `Schema*` | 待你定稿 | 与各工具一并审定 |

## 当前文案快照（基线 v0.1.3 → 改动前）

便于对比讨论；**定稿后本表改为「已定稿文案」链接或删除快照**。

### read_file

```
读取本地文件系统中的文件。你可通过该工具直接访问任意文件。假定本工具能够读取设备上的所有文件。若用户提供文件路径，则默认该路径有效。读取不存在的文件也属正常操作，此时工具会返回错误信息。

用法：
- 本工具仅支持读取文件，无法读取目录。如需读取目录内容，请通过 {bash} 工具执行 ls 命令。
- 若你已知道需要文件的哪一部分，就只读那一部分；对大文件尤为重要。
- 无法读取二进制或媒体文件。
```

### grep

```
在工作区内用正则搜索文件内容；始终跳过 .git，可选 tools.search.skip_dirs；搜索前应收窄 path 避免盲目全库扫描。凡内容搜索任务必须调用本 grep 工具，禁止通过 bash 工具执行 grep 或 rg。
```

### glob

```
按 Glob 模式查找项目根目录下的文件（例如 **/*.go）。始终跳过 .git，可选 tools.search.skip_dirs；搜索前应收窄 path。
```

### list_dir

```
列出相对项目根目录的路径下的文件与目录。始终跳过 .git，可选 tools.search.skip_dirs；列举前应收窄 path。
```

### bash（DescShell，wire 名待统一）

```
在项目工作区运行 shell 命令。background=true 可后台启动，用 job_id 轮询或取消。
```

### apply_patch / write_file

```
应用 Codex 风格补丁（*** Begin Patch ... *** End Patch）到工作区文件。编辑已有文件请优先于 write_file。

创建新文件或覆盖整个文件。编辑已有文件请优先使用 apply_patch。
```

### agent

```
启动一个子代理处理复杂多步骤任务。有 4 种类型：general-purpose（全能）、Explore（只读探索）、Plan（架构规划）、verification（验证）。可用时优先并行启动多个 agent。
```

### tool_search

```
按名称查找工具的完整定义（仅用于延迟加载的工具）
```

## 待你回答的问题（改写前）

见 REQUIREMENTS §2 协作流程；每工具开工前在 issue/PR/对话中确认：

1. 描述长度：偏 Cursor 长文（如 read_file）还是一行精炼？
2. 是否写「禁止用 bash 绕行」：哪些工具必须写、写到什么程度？
3. 交叉引用：工具名用字面量还是注入（改名安全）？
4. `list_dir` vs `glob` vs `grep` 的分工话术是否要在每个 Desc 里重复？
5. `agent` 四种类型的说明要不要扩写（与 spawn overlay 分工）？

---

**下一步**：请指定 **第一个** 要开工的工具（或你直接贴该工具的 Desc/Schema 定稿）。我会只出该工具的草稿供你改，确认后再写代码。
