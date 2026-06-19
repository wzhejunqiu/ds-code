# v0.1.2 安全与配置同步草稿

> 版本：v0.1.2  
> 状态：设计中  
> 用途：实现合入 / 发布前，将下文合入 [SECURITY.md](../v0.1.0/SECURITY.md) 与 [CONFIG.md](../v0.1.0/CONFIG.md)  
> 更新日期：2026-06-19

## 1. SECURITY.md 变更

### 1.1 威胁模型表 — S2 行

**替换**「路径遍历」缓解说明：

| 威胁 | 缓解 |
|------|------|
| 路径遍历读写工作区外文件 | `permission.ResolvePath` / `ResolveAccessPath`：`filepath.Clean` + join、`EvalSymlinks`、`ensureUnder`  jail 到 `project_root`；**不再**对相对路径做 `..` 子串拦截 |

### 1.1b 威胁模型表 — S3 行

**替换**「敏感文件泄露」缓解说明：

| 威胁 | 缓解 |
|------|------|
| 敏感文件泄露（`.env`、密钥） | 路径段级 denylist（`IsSensitiveAbs`）；Agent 枚举、`read_file`、`shell`、LSP 统一过滤；**例外**：用户提示词显式 `@file` / `@dir/` 仅 S2（§S3-S）；MCP spill 全文 0600 存 project 数据目录，仅当前 session 经 `read_file` 可读；compact 输入对 `@` 展开块不做专用脱敏（见 §1.1c、§1.3） |

### 1.1c 威胁模型表 — 新增 S3-S 行

**在威胁模型表新增一行**（与审计清单 §S3-S 对应）：

| 威胁 | 缓解 |
|------|------|
| 用户 `@` 显式绕过 S3 / compact 外发敏感内容 | 仅主会话 user message 中 `@file` / `@dir/` 可越过 S3（§S3-S）；Agent 工具、`read_file`、shell 仍 S3；`@` 展开受 `context.at_reference_max_chars` / `at_dir_max_*` 预算约束；compact 时 `sanitizeCompactInput`（S12）行级 redact **不**对 `@` 块专用剥离——旧轮 `@.env` 等**可能**进入摘要 LLM 外发输入；用户显式点名视为知情承担风险；TUI 复制可见 MCP 参数 / spill hint（与 shell 命令展示一致） |

### 1.1d 威胁模型表 — 新增 TUI 剪贴板行

**在威胁模型表新增一行**（与 FR-5.9、NFR-18 对应）：

| 威胁 | 缓解 |
|------|------|
| TUI 应用内复制泄露敏感可见内容 | 剪贴板写入 viewport **已渲染** plain text（剥离 ANSI）；**可能含** MCP 参数摘要（≤400 字符）、spill hint 绝对路径、shell 命令行——与 TUI shell 展示一致，**有意**可见；**不**复制未显示字段；与 `--allow-log-sensitive-data` 独立；失败降级提示、不 panic（FR-7.7） |

### 1.2 审计清单 — 更新 S2

| ID | 落点 | 说明 |
|----|------|------|
| S2 | `workspace.ResolveRel` + `permission.ResolvePath` | 相对/绝对路径先规范化与 symlink 求值，再 `ensureUnder`；合法 `foo/../bar` 允许；`../outside` 拒绝 |

### 1.3 审计清单 — 新增 S3-S

| ID | 落点 | 说明 |
|----|------|------|
| S3-S | `context/atref.go` | 用户提示词中显式 `@file` / `@dir/` 仅校验 S2（`ResolvePath`），**可**读取 `.env` 等 S3 路径并注入 user message；**不**应用 `textfile.IsSearchable` / grep 大小上限（仅靠 `at_dir_max_*` 预算）；Agent 枚举、`read_file`、`shell` 仍受 S3；用户显式点名视为知情承担风险。**compact**：`sanitizeCompactInput`（S12）对送入摘要 LLM 的 transcript 做行级启发式 redact，**不**对 `@` 展开块做专用剥离——旧轮 user message 中的 `@.env` 等内容可能进入 compact 摘要输入；用户承担点名风险 |

### 1.4 审计清单 — 更新 S11

| ID | 落点 | 说明 |
|----|------|------|
| S11 | `TruncateToolResult`、`finalizeToolResult`、`@` 预算 | 超大 tool/@ 结果截断；MCP 成功调用全文写入 `~/.ds-code/projects/<id>/mcp-result/<session_id>/<stem>.txt`（0600；`<stem>` = `spillCallFilename(id)`，空 id 时为 ULID），session 回注仍受 `tool_result_max_chars`；超长时 hint 含**完整可解析绝对路径**引导 `read_file`（见 DESIGN §12.5）；**仅当前 Runner session** 经 `CheckReadablePath` 可读 spill（须 `.txt` 绝对路径；`readonly`/`ask`/`auto` 均直接放行，NFR-22）；**不**扩展至 `agents/*.output`（FR-4.7）；`shell` 不可读 spill 绝对路径；spill 可能含 MCP 返回的敏感字段，回注 LLM 后进入上下文；compact 后旧 hint **不**保留（磁盘仍在，模型无法枚举目录，见 README 已知限制） |

### 1.4b 审计清单 — `read_file` 文本判定（FR-8）

| ID | 落点 | 说明 |
|----|------|------|
| S8-R | `read_file/read_file.go`、`textfile.IsTextFile` | 非文本文件（扩展名 blocklist + 前 3072 字节 sniff）拒绝读取；返回明确 tool 错误 + Info 日志；MCP spill `.txt` 与空文件仍允许；`@file`/`@dir/` 不经此判定（FR-6.11、FR-8.7） |

### 1.5 Shell 执行模型段落补充

在「Shell 执行模型」末追加：

- spill 文件路径位于 project 数据目录（工作区外），须用 `read_file` 读取；`shell cat` 该路径仍受 S2 区外拒绝。`mcp-result/` spill 为只读区外例外，`readonly`/`ask`/`auto` 均直接放行（NFR-22）。**不**扩展至既有 `agents/*.output` 子代理摘要 spill（FR-4.7）。
- TUI 应用内复制（v0.1.2）写入剪贴板的内容为 viewport **已渲染** plain text，可能含 MCP 参数摘要、spill hint 绝对路径；与 debug 日志 `--allow-log-sensitive-data` 策略独立（见 FR-5.9、NFR-18、威胁模型 TUI 剪贴板行 §1.1d）。
- `read_file` 拒绝非文本文件（图片、二进制等）；`@file`/`@dir/` 不经 `IsTextFile`（FR-8、§1.4b）。

---

## 2. CONFIG.md 变更（草稿条目）

实现时在 [CONFIG.md](../v0.1.0/CONFIG.md) 增加或更新：

### 2.1 `context.tool_result_max_chars`

- 内建工具与 MCP 工具**共用**此上限（无 `mcp_tool_result_max_chars`）。
- MCP 超长时 session 消息含 spill 路径 hint；完整正文在 project 数据目录（见下）。

### 2.2 Project 数据目录 — `mcp-result/`

```
~/.ds-code/projects/<project_id>/mcp-result/<session_id>/<stem>.txt
```

| 项 | 说明 |
|----|------|
| `project_id` | `hex(sha256(project_root))` |
| `<stem>` | `spillCallFilename(tool_call_id)`；非空 id 文件名安全化；空 id 为 ULID；**与** `messages.tool_call_id` **可不同** |
| 权限 | 目录 0700；文件 0600 |
| 写入时机 | 每次 **成功** MCP 调用（含未超长）；失败/取消不写（FR-4.16） |
| 读取 | 仅当前 session 经 `read_file` + spill **绝对路径**（`.txt` regular file）；`readonly`/`ask`/`auto` 均直接放行 |
| 不扩展 | `agents/<parentSession>/<toolCallID>.output`（子代理摘要既有 spill，FR-4.7） |
| 清理 | **不**自动 GC；可手动删除整个 `mcp-result/` |
| 子代理 | 独立 `session_id` 子目录；父 session 不可读 |
| compact 后 | 磁盘 spill 仍在；API 上下文可能无 hint；模型无法枚举目录（见 README 已知限制） |

### 2.3 `tools.search.skip_dirs`

```yaml
tools:
  search:
    skip_dirs: []   # 相对当前 perm.Workspace 根，如 ["node_modules", "target", ".ds-code"]
```

| 项 | 说明 |
|----|------|
| 作用范围 | 仅 **Agent 枚举**（`grep` / `glob` / `list_dir` / `diagnostics` walk） |
| 路径基准 | 条目相对**当前** `perm.Workspace`（主会话为 `project_root`；worktree 子代理为 checkout 根） |
| 默认 | `[]`（无框架内置 skip；`.git` 仍硬编码 SkipDir，**无**显式 path 例外） |
| 显式 path | 模型 `path=node_modules/...` 时**仍允许**进入该子树；**`.git` 除外**（FR-6.14） |
| 配置校验 | 条目须相对 `perm.Workspace`、无 `..`、无绝对路径；非法条目 `Warn` 并忽略（FR-6.15） |
| `@dir/` / `@file` | **不**应用 `skip_dirs` |
| `.gitignore` | v0.1.2 起 Agent **不**读取 `.gitignore` |

### 2.4 `tui.copy_on_select`

```yaml
tui:
  copy_on_select: true   # 默认 true；鼠标松手自动写剪贴板（FR-7.12）
```

| 项 | 说明 |
|----|------|
| 作用范围 | 仅交互 TUI（备用屏幕 + 鼠标选区） |
| 关闭时 | 选区保留；`Ctrl+Shift+C` / macOS `Cmd+Shift+C` 手动复制（FR-7.5） |
| 非交互 `-p` | 不适用（stdout 原生可选中） |

### 2.5 `@` 引用预算（交叉引用，行为不变）

v0.1.2 `@` S3 例外（§S3-S）仍受既有预算约束；实现时于 CONFIG 交叉引用：

| 配置项 | 说明 |
|--------|------|
| `context.at_reference_max_chars` | `@` 展开总字符上限（默认 128000） |
| `context.at_dir_max_files` | `@dir/` 最大文件数 |
| `context.at_dir_max_depth` | `@dir/` 最大遍历深度 |

`@file` / `@dir/` **不**应用 Agent 枚举的 S3 / `IsSearchable` / grep 大小上限（FR-6.11）；仅靠上述预算截断。

### 2.6 `configs/example.yaml` 注释建议

```yaml
tools:
  search:
    # 可选：永久排除 Agent 枚举扫描的目录（相对 perm.Workspace 根；不影响 @dir/）
    # skip_dirs: ["node_modules", "target", ".ds-code"]

tui:
  copy_on_select: true
```

---

## 3. CHANGELOG 要点（v0.1.2）

实现时写入 [CHANGELOG.md](../../CHANGELOG.md)：

- **路径**：`..` 子串拦截改为规范化 + `ensureUnder`；修复 shell 误拦 `git main..branch`、`go test ./...`
- **权限**：路径策略收敛至 `permission.Engine`；`@file`/`@dir/` 可越过 S3（用户显式点名，SECURITY §S3-S）
- **MCP**：结果落盘 `mcp-result/`；session 仍截断；hint 含可 `read_file` 的绝对路径；读本 session spill
- **MCP**：TUI / debug 展示调用 JSON 参数
- **搜索**：Agent 枚举不再遵循 `.gitignore`；可选 `tools.search.skip_dirs`（相对 `perm.Workspace`）；`glob **/*` Walk 注入 skip（FR-6.14）；`diagnostics` 始终过滤 `.git`
- **TUI**：应用内鼠标选区 + 剪贴板（`tui.copy_on_select`）；plain text 剥离 ANSI
- **read_file**：非文本文件拒绝（`textfile.IsTextFile`）；MCP spill `.txt` 与空文件仍可读

---

## 4. 验收勾稽

合入正式文档后，在 [ACCEPTANCE.md](ACCEPTANCE.md) §1 勾选 SECURITY / CONFIG 同步项（含 §1.1d TUI 剪贴板威胁行、§1.4b `read_file` 文本判定）。
