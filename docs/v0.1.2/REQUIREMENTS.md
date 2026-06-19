# v0.1.2 需求文档

> 版本：v0.1.2  
> 状态：已实现  
> 更新日期：2026-06-20

## 1. 目标

1. **路径权限收敛**：文件系统相关工具、patch、shell 路径扫描等路径访问判定，统一经 `permission.Engine` 单一 API；**`@file` / `@dir/`** 展开链路仅用 `ResolvePath`（S2），见 FR-6.9–6.10。
2. **规范化后再判定**：相对路径中的 `.` 与 `..` 不再做子串级拦截；先在工作区根下解析为绝对路径（`Clean`、symlink 求值、`ensureUnder`），再对 **Agent 工具 / shell / `read_file`** 应用 S2 边界与 S3 敏感 denylist。
3. **MCP 结果落盘**：MCP 工具完整持久化正文写入 `mcp-result/<session_id>/<stem>.txt`（`<stem>` = `spillCallFilename(tool_call_id)`）；session 仍受 `tool_result_max_chars` 截断；**模型经 `read_file` 读取当前 session 的 spill 绝对路径**获取完整 MCP 结果（见 FR-4.12–4.14）。
4. **MCP 调用参数可见**：每次 `CallTool` 在 TUI（交互模式）与 debug 日志（`-vv`）中输出本次 JSON 参数。
5. **搜索路径最大化可见**：Agent 枚举不再遵循 `.gitignore` 与框架内置 skip；**始终**跳过 `.git`（含显式 `path=.git`）；用户可配置 `tools.search.skip_dirs`；其余由**模型**决定 `path`/`pattern`。用户显式 **`@file` / `@dir/`** 见 FR-6.9–6.10。
6. **TUI 应用内选中与剪贴板**：交互 TUI 在备用屏幕模式下支持鼠标拖拽选中文本并写入系统剪贴板（对标 Claude Code fullscreen 复制体验）；见 FR-7。
7. **`read_file` 仅读文本**：非文本文件拒绝读取，向模型返回明确错误并记录 Info 日志；见 FR-8。
8. **TUI 平滑滚动**：交互 TUI 聊天区与工具面板支持流畅的多页滚轮滚动（对标 Claude Code 三层滚动架构：输入累加、分帧 drain、渲染减负）；见 FR-9。

## 2. 用户故事

### US-1：合法 `..` 段可访问

**作为** 使用 ds-code 的开发者，  
**我希望** Agent 能通过 `read_file` 读取 `internal/foo/../bar.go` 这类路径，  
**以便** 模型按惯用相对路径表达仍能访问工作区内真实文件。

**验收**：`foo/../bar` 解析后位于 `project_root` 内时，`read_file` / `grep` / `list_dir` 等读工具允许访问；`../outside` 或解析后越界仍拒绝。

### US-2：单一权限真相源

**作为** 维护者，  
**我希望** 调整 S2/S3 策略时只改 `internal/permission` 一处，  
**以便** 不会出现 patch 走了旧逻辑而 read 工具走了新逻辑的漂移。

**验收**：`grep` 无 `import "…/internal/workspace"` 直接路径校验；`IsSensitiveAbs` 不对 `internal/tool` 导出（或工具仅调 Engine 封装）。

### US-3：`.` 表示工作区根

**作为** 用户，  
**我希望** `list_dir` 默认路径 `.` 与显式 `./src` 行为一致且可预期，  
**以便** 根目录列举不被误拦。

**验收**：`path="."`、`path="./"`、`path=""`（工具默认）均解析为工作区根且通过 S3 检查（根本身不敏感）。

### US-4：shell 不误拦 git / go 惯例语法

**作为** 使用 ds-code 的开发者，  
**我希望** Agent 能执行 `git diff origin/main...v0.1.1`、`git log main..HEAD`、`go test ./...` 等命令，  
**以便** 常规开发工作流不被权限层误杀。

**验收**：上述命令在 `auto` / `readonly`（只读 git）下 **permission 层放行**；若 git 本身报 `unknown revision` 等，属 git 行为，非 ds-code 拦截。真路径逃逸（如 `cat ../outside`）仍拒绝。

### US-5：MCP 大结果可回溯

**作为** 依赖 MCP 的开发者，  
**我希望** MCP 返回超大 JSON 时，模型上下文里仍是可消化的摘要，但完整内容保存在固定路径，  
**以便** Agent 可用 `read_file` 读取 spill 全文，而不撑爆上下文。

**验收**：150k 字符 MCP 结果 + `tool_result_max_chars=100000` 时，session tool 消息正文+hint **合计** ≤100k 且含 spill 绝对路径；**当前 session** 下 `read_file path="<spill 绝对路径>"` 返回完整 150k 正文。

### US-6：MCP 调用参数可观测

**作为** 使用 MCP 的开发者，  
**我希望** 在 TUI 和 debug 日志里看到每次 MCP 调用的 JSON 参数，  
**以便** 核对模型是否传错 `query`、`changed_files`、`detail_level` 等，而不只看见工具名。

**验收**：调用 `get_impact_radius_tool` 且 args 为 `{"max_depth":2}` 时，TUI 工具行含 `max_depth`；`-vv` 日志含 `args_preview` 或（开启敏感日志时）完整 `args`。

### US-7：Agent 能搜到 gitignore 内的源码

**作为** 使用 ds-code 的开发者，  
**我希望** Agent 用 `grep` / `glob` 能发现 `.gitignore` 中列出的目录下的源码，  
**以便** 模型对 monorepo、生成物旁路源码等有完整感知。

**验收**：项目 `.gitignore` 含 `internal/generated/` 时，`grep` 可搜到其下 `.go` 文件。

### US-8：用户显式 `@` 可读敏感路径

**作为** 在提示词写 `@.env` 或 `@.ssh/config` 的开发者，  
**我希望** 该路径按我的点名预加载进上下文，  
**以便** 调试配置时不必绕过 ds-code 安全策略。

**验收**：`@.env`、`@.ssh/config` 可展开进 user message；`read_file path=".env"` 与 shell `cat .env` **仍** S3 拒绝。

### US-9：`@dir/` 用户点名即读

**作为** 在提示词里写 `@node_modules/foo/` 或 `@internal/generated/` 的用户，  
**我希望** 该目录下文件按我的指定全部预加载（在预算内），  
**以便** 不被 gitignore 或 S3 静默跳过。

**验收**：`.gitignore` 忽略的路径，当用户输入 `@that_dir/` 时均尝试展开；仅因 `at_dir_max_files` 等预算截断时提示，不因策略过滤静默省略。

### US-10：TUI 输出可复制

**作为** 在 Cursor / iTerm2 等终端中使用 ds-code 的开发者，  
**我希望** 在 TUI 聊天区用鼠标选中 Agent 回复或工具输出，并自动复制到系统剪贴板，  
**以便** 在备用屏幕模式下也能把代码、命令、分析结论粘贴到其他应用，而不必退出 ds-code 或查 SQLite。

**验收**：在交互 TUI 中拖拽选中聊天 viewport 内可见文本，松手后系统剪贴板含**纯文本**（无 ANSI 转义）；粘贴到编辑器内容与选中范围一致。剪贴板不可用时给出可见错误提示，不 panic。

### US-11：可配置永久排除噪声目录

**作为** 在 monorepo 中日常使用的开发者，  
**我希望** 通过 `tools.search.skip_dirs` 永久排除 `node_modules`、`target` 等目录，  
**以便** Agent 每次 `grep path=.` 时不会扫穿巨型目录，同时仍可用显式 `path=node_modules/pkg` 深入排查。

**验收**：配置 `skip_dirs: ["node_modules"]` 后，`glob **/*` path=`.` 不匹配 `node_modules/`；`grep path=node_modules/pkg` 仍允许。

### US-12：compact 后 spill 可人工恢复

**作为** 长会话中使用 MCP 的开发者，  
**我希望** 了解 compact 后旧 spill hint 丢失时的恢复方式，  
**以便** 在模型无法自行发现磁盘 spill 时，仍能手动定位完整 MCP 结果。

**验收**：文档（README 已知限制、SECURITY §S11）说明：compact 前应用 `read_file` 读 spill；compact 后若无近轮 hint，须查 `~/.ds-code/projects/<id>/mcp-result/<session_id>/`；Agent **无法** `list_dir`/`grep` 枚举该目录。

### US-13：`read_file` 拒绝非文本文件

**作为** 使用 ds-code 的开发者，  
**我希望** Agent 调用 `read_file` 读取图片、二进制或可执行文件时被明确拒绝，  
**以便** 模型不会把乱码灌进上下文，且我能从日志看到拒绝原因。

**验收**：`read_file path="logo.png"` 返回 tool 错误（含「无法读取非文本文件」）；debug 日志含 `read_file skipped non-text file`；`read_file path="main.go"` 与 MCP spill `.txt` 仍成功。

### US-14：TUI 长会话滚轮流畅

**作为** 在 Cursor / iTerm2 / Ghostty 等终端中使用 ds-code 的开发者，  
**我希望** 在长 transcript 中用鼠标滚轮或触控板平滑上翻/下翻多页历史，  
**以便** 浏览 Agent 回复与工具输出时不卡顿、不一跳数屏，且翻页键（PgUp/PgDn）仍能瞬时跳转。

**验收**：聊天区内容超过一屏时，快速滚轮产生连续中间帧（非一次跳多页）；PgUp/PgDn 半页瞬时到位、无 drain 动画；滚轮进行中按 PgUp 清空 pending 并跳页；工具面板（`Ctrl+T`）滚轮行为一致。

## 3. 功能需求

### FR-1 路径权限 API 收敛

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-1.1 | 定义 `permission.Engine` 为**唯一对外**路径策略入口（读、写、枚举跳过） | P0 |
| FR-1.2 | `internal/tool/*`、`internal/context/*`、`internal/patch/*` 不得直接 import `workspace` 做权限校验（`patch/apply` 通过注入 `resolve func`） | P0 |
| FR-1.3 | 目录遍历中的敏感目录跳过（原 `IsSensitiveAbs` + `SkipDir`）改为 `Engine.SkipSensitiveAbs(abs)` | P0 |
| FR-1.4 | `ValidateGlobMatches` 改用 `CheckAbsPath(abs, PathBoundary)`（**仅 S2** 越界报错）；`MakeFileCandidate` 敏感过滤改用 `SkipSensitiveAbs`（skip，非整次失败） | P0 |
| FR-1.5 | `shell_sensitive_paths` 的 `checkPathCandidate` 移除独立 `strings.Contains(..)` 分支，与 FR-2 共用解析 | P0 |
| FR-1.6 | `workspace` 包保留纯路径代数（S2），`IsSensitiveAbs` 保留在 `permission` 包内，不对 tool 层公开 | P1 |
| FR-1.7 | shell 解析失败时：**越界/遍历**（`ensureUnder` 失败）拒绝；**非路径 token**（如 `2>/dev/null`、git ref 解析后在区内）放行；不得因去掉子串 `..` 而放行 `cat ../outside` | P0 |
| FR-1.8 | `internal/tool/globmatch` 移除对 `permission.IsSensitiveAbs` 的直接调用；Walk 时 `.git` / S3 skip 改由注入的 `skipDir func(rel string) bool`（来自 `searchskip` + `SkipSensitiveAbs`）；`MatchFiles` 签名扩展为 `MatchFiles(root, pattern, limit, skipDir)` | P0 |

### FR-2 规范化后判定 `.` / `..`

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-2.1 | 删除 `ResolveRel` 中 `strings.Contains(rel, "..")` 预拦截 | P0 |
| FR-2.2 | 相对路径统一：`abs = Clean(Join(ws, rel))` → `resolvePath`（symlink）→ `ensureUnder(ws, abs)` | P0 |
| FR-2.3 | 绝对路径逻辑不变：`Clean` → `EvalSymlinks` → `ensureUnder` | P0 |
| FR-2.4 | 解析后越界返回与现网一致的 `permission denied` / `outside workspace` 语义 | P0 |
| FR-2.5 | 解析后命中 S3 仍拒绝（含 `proj/../.env` 规范化到工作区内 `.env`） | P0 |
| FR-2.6 | 含 `..` **子串**但非路径组件的 token（如 `origin/main..v0.1.1`、`./...`、`a..b.txt`）规范化后在工作区内则允许 | P0 |
| FR-2.7 | `apply_patch` 对工作区内含合法 `..` 段的目标路径（如 `pkg/../pkg/foo.go`）与 FR-2.1–2.2 一致：解析后区内且非 S3 则允许 | P0 |

### FR-3 明确不在本需求范围

| ID | 描述 |
|----|------|
| FR-3.1 | `context/skills.go` 对 skill **名称** 的 `..` 拦截（非文件路径 API）— 保持现状或另开需求 |
| FR-3.2 | `worktree/manager.go` 对 slug/dir 的 `..` 拦截 — 属 worktree 命名安全，非通用路径 API |
| FR-3.3 | 新增可配置 denylist、扩大 S3 规则集 |
| FR-3.4 | MCP 工具参数的路径校验（仍走 `Engine.Check`，本需求不扩展 MCP schema） |
| FR-3.5 | 用户自定义 `.gitignore` 作为 ds-code 枚举依据 — 本版本**移除**该行为（见 FR-6） |
| FR-3.6 | `shell` 内用户自行执行 `rg`/`grep` 的忽略策略 — 不受 FR-6 约束 |
| FR-3.7 | Transcript 模式 / 将整个会话刷回终端 scrollback（Claude `Ctrl+o` + `[`）— 另开需求 |
| FR-3.8 | 关闭备用屏幕或 classic 无备用屏幕渲染器 — 另开需求 |
| FR-3.9 | 按住 `Shift`/`Option`/`Fn` 绕过鼠标捕获、走终端原生选区 — 可选后续；本版本以应用内选区为主 |
| FR-3.10 | 子代理 prompt 的 `@` 展开 — `spawn/execute.go` 子 `ctxSvc` 无 `AtExpander`（v0.1.1 现状）；本版本不改动 |
| FR-3.11 | `RunEphemeral`（`/btw`）无 tools、不写 spill（S13）；本需求 MCP spill 链路**不适用** |

> **说明**：`@file` / `@dir/` **在 v0.1.2 范围内**，见 FR-6.9–6.10；上表仅列与路径 API / TUI 等**无关**的排除项。

### FR-4 MCP 结果落盘与截断

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-4.1 | 每次 MCP 工具调用成功后，将 **完整** 持久化正文（`toolresult.FormatToolResult` 包装后的字符串，与写入 `messages` 截断前一致；**非** MCP SDK 层 `mcp/server.formatToolResult` 原始拼接）写入 spill 文件，再应用 `TruncateToolResult` | P0 |
| FR-4.2 | spill 路径固定：`~/.ds-code/projects/<project_id>/mcp-result/<session_id>/<stem>.txt`；`<stem>` = `spillCallFilename(tc.ID)`；`project_id = hex(sha256(cfg.ProjectRoot))` | P0 |
| FR-4.3 | 目录 `mcp-result/`、`mcp-result/<session_id>/` 权限 **0700**；文件 **0600**（与 `sessions.db` 一致） | P0 |
| FR-4.4 | spill 文件名 stem 由 `spillCallFilename(rawID)` 生成：非空 id 做文件名安全化（替换 `/`、`\`、`..`、NUL 等）；**空 id 每次 `Save` 独立生成 ULID**（`github.com/oklog/ulid/v2`），避免多 call 抢占同一文件 | P0 |
| FR-4.5 | 仅当 `len(full) > tool_result_max_chars` 时，截断 suffix 追加 spill **绝对路径** 与 `read_file` 提示；未超长时 session 与文件内容一致，**不**追加多余提示 | P0 |
| FR-4.6 | 内建工具仍仅 `TruncateToolResult`，**不**写 `mcp-result/` | P0 |
| FR-4.7 | 子代理摘要 spill（`agents/<session>/<toolCallID>.output`）与 MCP spill 一并经 `resolveProjectDataRead` 放行；`task` spill 时返回 `output_file` + `SavedResultHint`（见 DESIGN §12.8b） | P1 |
| FR-4.8 | 子代理 Runner（`ForSubagent`）与主 Runner 共用同一 spill 规则、`resultstore.Store` 实例与路径布局；`spawn/execute.go` 构造 `childRunner` 时须设置 `MCPResults: parentRunner.MCPResults`（或等价注入） | P0 |
| FR-4.9 | **不**引入 `mcp_tool_result_max_chars`；MCP 与内建共用 `context.tool_result_max_chars` | P0 |
| FR-4.10 | MCP server 端自截断内容仍原样进入 spill 文件；ds-code 不修改外部 MCP | P1 |
| FR-4.11 | spill 的 `project_id` 始终取自 `cfg.ProjectRoot`（**非** worktree 的 `perm.Workspace`）；子代理使用各自 `session_id` 子目录 | P0 |
| FR-4.12 | `read_file` 经 `CheckReadablePath` → `resolveProjectDataRead` **允许只读**本 project 下 `~/.ds-code/projects/<project_id>/` 内 **regular file**（**须绝对路径**，不展开 `~`）；**拒绝**相对路径、其他 project、目录路径；工作区内路径仍走 S2+S3 | P0 |
| FR-4.13 | spill 写入失败（磁盘满、权限等）：回退 `TruncateToolResult`、**不**追加 hint；`logging.L().Warn("mcp result spill failed", …)` | P0 |
| FR-4.14 | 超长时：先以 spill **完整绝对路径**（`Save` 返回值）计算 `hint = SavedResultHint(displayPath)`；`displayPath` 经 `shortenSpillPathForHint` **仅**缩短 hint 模板总长，且须保证 `read_file path=displayPath` 可成功（**不**用 `~`、**不**尾部截断到不可解析路径；见 DESIGN §12.5）；正文截断至 `tool_result_max_chars - len(hint)` 再拼接；**合计** ≤ `tool_result_max_chars` | P0 |
| FR-4.15 | **子代理 spill 可读**：子代理 MCP spill 写入 `mcp-result/<子代理 session_id>/`；父 Agent 经 `resolveProjectDataRead` **可** `read_file` 同 project 任意 session 的 spill；子代理在自身 `RunTurn` 内同样可读 | P0 |
| FR-4.16 | MCP 调用失败（`isToolErrorBody`）或 context 取消 mid-flight：**不写** spill；session 走普通截断或错误正文 | P0 |
| FR-4.17 | spill 完整正文**仅**经 `read_file` 读取；`shell` 访问 spill 绝对路径（工作区外）仍拒绝 | P1 |
| FR-4.18 | 每次**成功** MCP 调用均写 spill（含未超长），便于调试与路径一致；不采用「仅超长落盘」 | P1 |
| FR-4.19 | 同一 session 内相同非空 `tool_call_id` 再次 spill 时**覆盖**同名文件（LLM 重试 / recovery 重放）；空 id 仍每次 ULID 新文件 | P1 |

### FR-5 MCP 调用参数输出

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-5.1 | TUI `DisplaySummary` 对 MCP 裸名工具：在 `MCP {server} · {tool}` 后追加 `formatArgsJSON(rawArgs)` 单行预览（复用现有 400 字符上限） | P0 |
| FR-5.2 | 无参数或 `{}` / `null` 时仅显示标题行，不追加空 JSON | P0 |
| FR-5.3 | `logMCPCall` 增加 `args_preview`（compact JSON，默认截断至 200 字符） | P0 |
| FR-5.4 | `logging.AllowSensitiveData()` 为 true 时，debug 日志额外输出完整 `args` 字段（`logging.FieldString`） | P1 |
| FR-5.5 | 内建工具 `DisplaySummary` 与 `Registry.Execute` 日志行为不变 | P0 |
| FR-5.6 | 不向 LLM 重复注入参数（已在 assistant `tool_calls` 中）；本需求仅用户可观测通道 | P0 |
| FR-5.7 | 非交互 `-p` 模式无 TUI；MCP 参数经 `-vv` debug 日志（`args_preview`）可观测 | P1 |
| FR-5.8 | legacy `mcp__{server}__{tool}` 历史工具名在 TUI 同样展示 JSON 参数（`FormatMCPCallDisplay`） | P1 |
| FR-5.9 | TUI 主工具块展示 MCP 参数（≤400 字符）**有意**不受 `--allow-log-sensitive-data` 门控，与 shell 命令行展示一致；侧栏 `chattool.Line` 仍 truncate 60 | P1 |

### FR-6 搜索路径可见性（替代 `.gitignore` 枚举）

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-6.1 | **Agent 枚举**内建工具（`grep`、`glob`、`list_dir`、`diagnostics`）**不再**调用 `GitignoreMatcher` / 不加载 `.gitignore` | P0 |
| FR-6.2 | **移除** `GitignoreMatcher.Ignored` 中对「`.` 开头目录/文件（除 `.ds-code`）」的 blanket 跳过 | P0 |
| FR-6.3 | **不**内置框架或目录默认 skip 列表（无 `node_modules` / `target/` 等硬编码）；其余噪声**主要**由**模型**通过工具参数控制 | P0 |
| FR-6.4 | 仍保留：**S3 敏感路径**、**`.git` 目录始终 SkipDir**（含显式 `path=.git`；git 对象非可搜索源码，**无** `skip_dirs` 式显式 path 例外）、二进制跳过、`grep` 文件大小上限 | P0 |
| FR-6.5 | 新增可选配置 `tools.search.skip_dirs`（相对**当前** `perm.Workspace` 根的路径段，默认 `[]`）；**不**引入 `framework_skips_enabled` 或框架预设 | P1 |
| FR-6.6 | `write_file` / `apply_patch` **不变**（显式路径，仍 S2+S3）；`read_file` 工作区内路径仍 S2+S3，**另**可读本 project `mcp-result/<session_id>/` spill（FR-4.12）；`@dir/` / `@file` 走 FR-6.9 | P0 |
| FR-6.7 | 工具描述/文档（`grep` desc、`*.md`）去掉「遵循 .gitignore」；补充「模型应收窄 path，避免盲目全库搜索」 | P1 |
| FR-6.8 | `GitignoreMatcher` / `LoadGitignore` 自 Agent 枚举与 `@` 链路移除；无引用则删除 | P1 |
| FR-6.9 | **`@dir/` / `@file` 用户显式引用**：移除 `GitignoreMatcher`、Walk 中 S3/`.git` 过滤；**不**应用 `searchskip`/`skip_dirs`；路径解析用 `ResolvePath`（仅 S2）；仍保留上下文预算 | P0 |
| FR-6.10 | **S3 在 `@` 链路的例外**（实现时写入 SECURITY §S3-S）：仅用户提示词中的 `@file` / `@dir/` 可越过 S3；Agent 工具、`read_file`、shell **不变**；compact 摘要输入经 S12 行级 redact，**不**对 `@` 展开块专用剥离；`@` 展开内容进入 user message，由用户承担点名风险 | P0 |
| FR-6.11 | **`@file` / `@dir/`** 展开**不**应用 Agent 枚举的 `textfile.IsSearchable` / grep 文件大小上限；仅靠 `at_dir_max_*` 与 `at_reference_max_chars` 预算约束（用户显式意图；二进制文件亦可能进入上下文） | P1 |
| FR-6.12 | 移除隐藏目录 blanket skip 后，`.ds-code/`、`.github/` 等可被 Agent 枚举发现；属预期行为变更（项目配置通常不含 API key，见 S1）；`configs/example.yaml` 可注释建议 `skip_dirs: [".ds-code"]` | P1 |
| FR-6.13 | **`@` 引用语法**：仅 `atRefPattern` 匹配的字符集（`[a-zA-Z0-9_./\-]+`）；不支持含空格或 Unicode 的路径字面值——文档化，本版本不扩展正则 | P2 |
| FR-6.14 | **`glob **/*` + `skip_dirs`**：`globmatch.MatchFiles` Walk 时注入 `searchskip.SkipDir`，与 `grep`/`list_dir` 一致在 Walk 阶段跳过；`CollectGlobPattern` 结果层仍经 `ignored()` 二次过滤 | P0 |
| FR-6.15 | **`skip_dirs` 配置校验**：条目须为相对 `perm.Workspace` 的 slash 路径段（无 `..`、无绝对路径）；非法条目加载时 `Warn` 并忽略 | P1 |

### FR-7 TUI 应用内选中与剪贴板

> **背景**：ds-code TUI 使用 Bubble Tea 备用屏幕（`WithAltScreen`）+ 鼠标捕获（`WithMouseCellMotion`），终端原生拖拽复制失效。本需求在应用内实现选区与剪贴板写入，对齐 Claude Code fullscreen 模式的核心复制路径（不实现 classic 渲染器或 transcript 刷回 scrollback，见 FR-3.7–3.8）。

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-7.1 | 聊天 **viewport**（含 header + 主 transcript）支持鼠标拖拽建立文本选区；选区高亮可见 | P0 |
| FR-7.2 | 工具面板（`Ctrl+T` 打开时）同样支持选区与复制 | P1 |
| FR-7.3 | 写入剪贴板内容为 **plain text**：复制前剥离 ANSI/SGR 转义与 lipgloss 样式，保留换行与可见字符 | P0 |
| FR-7.4 | 默认 **Copy on select**：鼠标松手（button release）后自动写入系统剪贴板 | P0 |
| FR-7.5 | 关闭 Copy on select 时，选区保留至用户按 `Ctrl+Shift+C`（或 macOS 终端 `Cmd+Shift+C`）手动复制 | P1 |
| FR-7.6 | 剪贴板后端按平台择优：`pbcopy`（macOS）、`wl-copy`（Wayland）、`xclip`/`xsel`（X11）、PowerShell `Set-Clipboard` / `clip.exe`（Windows/WSL）；均不可用时尝试 **OSC 52** | P0 |
| FR-7.7 | 复制成功/失败在 TUI 给出短暂提示（toast 或 footer 一行）；失败不阻塞 UI | P0 |
| FR-7.8 | 选区与复制**不**作用于输入框（`textinput` 仍由 bubbles 处理光标/编辑）；浮层（`/help`、`/context`、权限 prompt 等）打开时优先浮层内选区，否则忽略聊天选区 | P1 |
| FR-7.9 | Agent 回合进行中（`running==true`）仍可选中**已渲染**的聊天内容（用户可滚轮上翻复制历史）；Esc 取消回合优先于复制快捷键；流式输出进行中已建立选区**可能**随新渲染行错位或失效（不追未到达 token，见 DESIGN §15.1） | P0 |
| FR-7.10 | 双击选中词、三击选中行（词界与常见终端一致，路径/token 尽量整段选中） | P2 |
| FR-7.11 | 键盘扩展选区：`Shift+↑/↓` 扩展；选区触顶/触底时滚动 viewport；`Shift+Home`/`Shift+End` 扩至行首/行尾 | P2 |
| FR-7.12 | 配置项 `tui.copy_on_select`（默认 `true`）；可在 `/config` 或项目/用户 YAML 切换（实现细节见 DESIGN §15） | P1 |
| FR-7.13 | 非交互 `-p` 模式**不在范围**（stdout 本就可选中）；本需求仅交互 TUI | P0 |
| FR-7.14 | 不修改 `WithAltScreen()` 默认行为；不引入 classic 无备用屏幕渲染器 | P0 |

### FR-8 `read_file` 仅读文本

> **背景**：v0.1.1 中 `grep`/`glob`/`list_dir` 经 `textfile.IsSearchable` 跳过二进制，但 `read_file` 无同等校验，模型误读 `.png`、`.wasm`、`.pdf` 等会浪费 token 或得到无意义行内容。本需求与 Agent 枚举对齐，**复用**同一判定函数。

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-8.1 | `read_file` 在 `CheckReadablePath` 与 `Stat`（体积预检）**之后**、打开文件**之前**，调用 `textfile.IsTextFile(abs)`；返回 `false` 时**不**读入文件内容 | P0 |
| FR-8.2 | 非文本时向模型返回 tool 错误：`read_file: 无法读取非文本文件: {path}`（`path` 为调用参数中的原始路径，便于模型纠错） | P0 |
| FR-8.3 | 同时写 **Info** 日志：`read_file skipped non-text file`，字段至少含 `path`（用户参数）、`abs`（解析后绝对路径）；**不**将文件内容写入日志 | P0 |
| FR-8.4 | 新增 [`textfile.IsTextFile`](../../internal/tool/textfile/textfile.go) 作为 `read_file` **专用**入口；v0.1.2 实现委托 `IsSearchable`；`grep`/`glob`/`list_dir` **仍**直接调 `IsSearchable`；**不**新增配置开关 | P0 |
| FR-8.5 | 空文件（0 字节）**仍允许**读取（与 `IsTextFile` / `IsSearchable` 现有语义一致） | P0 |
| FR-8.6 | MCP spill 文件（`.txt`、ds-code 写入的 UTF-8 文本）**不受影响**；权限放行（FR-4.12）先于文本判定，spill 路径须通过 `IsTextFile` | P0 |
| FR-8.7 | **`@file` / `@dir/`** 用户引用**不在范围**（不经 `read_file.Execute`；仍不应用 `IsSearchable`，见 FR-6.11） | P0 |
| FR-8.8 | 更新 `read_file.md` 工具描述：说明非文本文件将被拒绝，建议使用 `glob`/`grep` 发现路径后再读源码 | P1 |

### FR-9 TUI 平滑滚动

> **背景**：v0.1.2 引入 TUI 鼠标选区后，滚轮事件被 `handleMouse` 拦截；若直接跳转 viewport 或使用过大的 `MouseWheelDelta`（如 `chatH/3`），长 transcript 滚动手感卡顿、一次跳多行。本需求参考 Claude Code 终端多页滚动三层架构：**输入累加**（`scrollBy`）、**分帧 drain**（pending 队列按比例释放）、**渲染减负**（Bubble Tea `HighPerformanceRendering` + `SyncScrollArea`）；翻页键走 **`scrollTo` 瞬时跳转**，与滚轮语义分离。

| ID | 描述 | 优先级 |
|----|------|--------|
| FR-9.1 | 滚轮 / 触控板事件调用 **`scrollBy`**：累加 `pendingScrollDelta`，**不**立即修改 `YOffset` | P0 |
| FR-9.2 | 翻页键（PgUp/PgDn、HalfPage、↑/↓ 等经 viewport KeyMap）及未来 `Ctrl+U`/`Ctrl+D` 扩展走 **`scrollTo` / `jumpBy`**：瞬时写入 `YOffset` 并**清空** pending | P0 |
| FR-9.3 | **分帧 drain**：pending 非零时以高频 tick（约 4ms，~250fps 等效）每帧释放有限行数；原生终端用 **proportional**（每帧至少 4 行，否则取 pending 的 3/4，单帧 cap `viewportH-1`）；集成终端（VS Code / Cursor）用 **adaptive**（pending ≤5 一帧释放；更大时 2–3 行/帧；pending >30 snap 截断） | P0 |
| FR-9.4 | 聊天 **viewport**（`chatVP`）与工具面板 **viewport**（`toolVP`）均支持滚轮平滑滚动；按鼠标 Y 坐标路由到对应 viewport | P0 |
| FR-9.5 | 启用 `viewport.HighPerformanceRendering`；drain 帧通过 `tea.SyncScrollArea` / `ScrollUp` / `ScrollDown` 减少全屏重绘（Bubble Tea 对 DECSTBM 硬件滚动的等效路径） | P0 |
| FR-9.6 | **文本选区活跃时**（`selDragging` 或 `selRange.Active()`）临时关闭 HP 渲染，回退全量 `View()` 以正确显示 selection highlight；选择结束后恢复 HP 并全量 sync 一次 | P1 |
| FR-9.7 | drain 期间**不**触发 `syncChatView` / `buildViewportContent`（仅 YOffset 变化）；滚动活跃时暂停 chat sync flush，结束后补一次 flush | P1 |
| FR-9.8 | 滚轮输入支持终端 profile 区分与 burst 累加；可选环境变量 `DS_CODE_SCROLL_SPEED`（默认 `1.0`）调节基准倍率 | P1 |
| FR-9.9 | 浮层（`/help`、`/context`、权限 prompt 等）打开时忽略滚轮（与 FR-7.8 一致） | P0 |
| FR-9.10 | Agent 回合进行中（`running==true`）滚轮上翻历史仍可用（与 FR-7.9 一致） | P0 |
| FR-9.11 | 非交互 `-p` 模式**不在范围** | P0 |
| FR-9.12 | **不**实现 React 式虚拟列表（`chat.RenderCache` 已按 block 缓存；全量拼接仅在内容变更时执行） | P0 |

## 4. 非功能需求

| ID | 描述 |
|----|------|
| NFR-1 | `make test` 全绿；新增/调整 `workspace`、`permission`、builtin tool 单测 |
| NFR-2 | 不降低 symlink 逃逸防护（现有 `TestValidateRel_rejectsSymlinkEscape` 等须保持） |
| NFR-3 | 错误信息对用户/模型仍包含原始 `rel` 参数，便于纠错 |
| NFR-4 | 路径解析链路不引入额外 `Stat`（与 v0.1.1 同级）；MCP spill 每次成功调用一次 `WriteFile` 可接受 |
| NFR-5 | MCP spill 可能占用磁盘；不自动 GC（与会话/checkpoint 策略一致，后续可另开清理需求） |
| NFR-6 | spill 路径在 project 数据目录内；`read_file` 经 `CheckReadablePath` + `SpillSessionID` 可读（见 DESIGN §12.6）；`shell` 不可读 |
| NFR-7 | debug 日志 `args_preview` 默认不含完整参数；完整 `args` 需 `--allow-log-sensitive-data`（TUI 展示策略见 FR-5.9） |
| NFR-8 | 移除 `LoadGitignore` 全树 Walk 后，启动少一次 IO（较 v0.1.1） |
| NFR-9 | 模型若 `grep path=.` 扫到 `node_modules` 等，靠 `head_limit`/截断/提示收窄 path；用户可用 `skip_dirs` 永久排除常扫目录 |
| NFR-10 | shell `isOutsideWorkspaceErr` 使用 `errors.Is(err, workspace.ErrOutsideWorkspace)` 等 typed error，避免字符串匹配脆弱性（见 DESIGN §6.2） |
| NFR-11 | 每次成功 MCP 调用一次 `WriteFile`（含未超长）；用户可手动删除 `mcp-result/`（与 checkpoint/shell-jobs 一致） |
| NFR-12 | compact 摘要（S12）对 transcript 行级启发式 redact；**不**对 user message 中 `@` 展开块做专用剥离（旧轮 `@.env` 等可进入 compact 摘要 LLM 输入）；**不**在 compact 摘要中保留 spill 路径（见 Out of scope） |
| NFR-13 | `SpillSessionID` 在 `RunTurn` 入口设置一次（覆盖该轮所有 tool batch，含并发 MCP） |
| NFR-14 | worktree / 子代理路径：`perm.Workspace` 可为 checkout 路径；`perm.ProjectRoot` **始终** `cfg.ProjectRoot`（`spawn/execute.go` **所有**新建 `Engine` 分支，含 readonly worktree，均须设置） |
| NFR-15 | 选区渲染与复制路径不显著增加 TUI 帧延迟；拖拽过程中允许节流重绘 |
| NFR-16 | 剪贴板写入在 goroutine 中执行，不阻塞 Bubble Tea 主循环 |
| NFR-17 | OSC 52 / 外部剪贴板命令失败时降级为提示，**不**将 spill 路径或敏感 tool 参数额外写入日志（除非已有 `-vv` 策略） |
| NFR-18 | 复制纯文本不得夹带 MCP 参数、权限 token 等**未在 viewport 显示的隐藏字段**；**已渲染**的 MCP 参数摘要、spill hint 路径、shell 命令行可复制（与 FR-5.9 一致） |
| NFR-19 | 新增依赖 `github.com/oklog/ulid/v2`（spill 空 id 文件名，见 DESIGN §12.4） |
| NFR-20 | `RunEphemeral`（`/btw`）不触发 MCP spill 写入 |
| NFR-21 | `read_file` 文本判定在 `Stat` 之后执行；`IsTextFile` 当前委托 `IsSearchable`（sniff 前 3072 字节），与 grep 同级 |
| NFR-22 | `read_file` 读本 session `mcp-result/` spill 视为**只读区外例外**；`readonly` / `ask` / `auto` 均**直接放行**，不触发 ask 弹窗（工作区内 S3 路径规则不变） |
| NFR-23 | 滚轮 drain 帧不显著增加 TUI 帧延迟；drain 期间避免全量 lipgloss 重绘聊天正文（依赖 FR-9.5 HP 路径） |
| NFR-24 | pending 队列有上限（默认 48 行量级），防止极端 burst 无限累积 |
| NFR-25 | 翻页瞬时跳转与滚轮 drain **互斥**：`scrollTo` 必须清空 pending，避免动画与 snap 冲突 |

## 5. 范围边界

**In scope**

- `internal/workspace/{path,errors}.go`（`ErrOutsideWorkspace` sentinel）
- `internal/permission/{engine,workspace,sensitive,shell_sensitive_paths}.go`（`ProjectRoot`、`SpillSessionID` 字段）
- `internal/tool/builtin/*`、`internal/tool/globmatch/*`（`MatchFiles` 注入 `skipDir`；移除 `IsSensitiveAbs` 直接调用，FR-1.8）
- `internal/patch/{validate,apply,parser.go}`（`ValidatePath` 改经注入校验函数，去除 `workspace` import）
- `internal/checkpoint/rewind.go`（`patch.ValidatePath` 改调 Engine 注入）
- `internal/context/{atref.go,atref_test.go}`（`@` 移除 gitignore/S3 过滤；改写冲突单测，见 FR-6.9）
- `internal/agent/tool_orchestration.go`、`cmd/ds-code/app`（MCP spill 注入 + 截断；`RunTurn` 入口设置 `perm.SpillSessionID`）
- `internal/agent/spawn/execute.go`（**所有**新建 `Engine` 分支 `perm.ProjectRoot = cfg.ProjectRoot`；子 `Runner` 注入 `MCPResults` + `SpillSessionID`）
- `internal/toolresult/{format,text}.go`、`internal/context/toolresult.go`
- `internal/datadir/paths.go`、`internal/mcp/resultstore/store.go`（spill 路径与写入）
- `internal/mcp/server.go`、`internal/tool/display.go`（MCP 参数展示与日志）
- `internal/tool/gitignore.go`（删除）
- `internal/tool/searchskip/`（`.git` 硬编码 + `tools.search.skip_dirs`）
- `internal/config/types.go`、`configs/example.yaml`（`tools.search.skip_dirs`、`tui.copy_on_select`）
- `cmd/ds-code/app/tools.go`、`internal/tool/setup/setup.go`、`internal/tool/register/explore.go`（`Gitignore` → `searchskip.Matcher`；Plan/子代理同步）
- `internal/ui/tui/run.go`、`internal/ui/tui/model/*`（鼠标选区状态、与 viewport 坐标映射）
- `internal/ui/tui/scroll/`（`scrollBy`/`scrollTo`、pending drain、终端 profile、wheel step）
- `internal/ui/clipboard/` 或等价包（平台剪贴板 + OSC 52 降级）
- `internal/tool/textfile/textfile.go`（`IsTextFile`，FR-8.4）
- `internal/tool/builtin/read_file/{read_file.go,read_file.md,text.go}`（FR-8 调用 `IsTextFile` + 日志）
- `internal/tool/builtin/grep/grep.md`、`glob/glob.md`、`list_dir/list_dir.md` 等工具描述（FR-6.7）

**Out of scope（v0.1.2 其他需求另文档）**

- Skills / worktree **命名校验**（FR-3.1/3.2；非通用路径 API）
- 权限模式（readonly/ask/auto）行为变更
- Checkpoint **快照**路径策略（仅 rewind 中 `ValidatePath` 调用随 patch 迁移）
- Audit 路径
- 修改 code-review-graph 等外部 MCP server 的内置节点数上限
- spill 文件自动 GC / 清理 CLI
- **`read_file` 扩展至 `agents/*.output`**（子代理摘要既有 spill；见 FR-4.7）
- **compact 后保留 MCP spill 路径 hint**（旧轮 tool 消息被摘要替换后路径丢失；磁盘 spill 仍在；模型无法枚举 `mcp-result/`；见 README 已知限制）
- **`@` 展开内容的 compact 专用脱敏**（S12 行级启发式仍作用于 compact 输入；不对 `@` 块做专用剥离；若需另开需求）
- **子代理 prompt `@` 展开**（FR-3.10；主会话 user message 仍走 `AtExpander`）
- **框架预设 skip 列表**（npm/Spring 等 marker 检测；由模型 + 用户 `skip_dirs` 承担）
- **TUI transcript 导出 / classic 渲染器 / 终端原生选区绕过**（FR-3.7–3.9）
