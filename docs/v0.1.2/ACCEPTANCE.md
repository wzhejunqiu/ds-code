# v0.1.2 验收标准

> 版本：v0.1.2  
> 状态：设计中  
> 更新日期：2026-06-20  
> 需求：[REQUIREMENTS.md](REQUIREMENTS.md) · 设计：[DESIGN.md](DESIGN.md)

## 1. 总体验收

- [ ] 版本号标记为 v0.1.2（release/tag 由发布流程完成）
- [ ] `make test` 通过
- [ ] `make lint` / `make vet` 无新增失败
- [ ] [SECURITY-SYNC.md](SECURITY-SYNC.md) 草稿已合入 [SECURITY.md](../v0.1.0/SECURITY.md) 与 [CONFIG.md](../v0.1.0/CONFIG.md)（**发布阻塞**；含 `tui.copy_on_select`、`tools.search.skip_dirs`、`@` 预算交叉引用 §2.5、威胁模型 §S3-S / §1.1d 行）
- [ ] [CHANGELOG.md](../../CHANGELOG.md) v0.1.2 条目
- [ ] [../v0.1.0/DESIGN.md](../v0.1.0/DESIGN.md) 权限节已补充 Engine 路径 API 一览（DESIGN §8）
- [ ] `internal/agent/README.md`、`internal/tool/builtin/README.md` 路径/MCP 相关描述已同步
- [ ] `grep`/`glob`/`list_dir` 工具 `Desc` 与 `*.md` **不含**「遵循 .gitignore」（FR-6.7）
- [ ] `read_file.md` / `DescReadFile` 已说明非文本文件拒绝（FR-8.8）
- [ ] `go.mod` 含 `github.com/oklog/ulid/v2`（NFR-19）
- [ ] `internal/tool` 下无对 `workspace.ValidateRel` / `EnsureAbsUnder` 的直接调用
- [ ] `internal/tool/globmatch` **无**对 `permission.IsSensitiveAbs` 的直接调用（FR-1.8）
- [ ] `spawn/execute.go` **所有**新建 `Engine` 分支设置 `perm.ProjectRoot = cfg.ProjectRoot`（含 readonly worktree，NFR-14）
- [ ] `spawn/execute.go` 子 `Runner` 注入 `MCPResults`（与父同一 `*resultstore.Store`，FR-4.8）
- [ ] `IsSensitiveAbs` 已降可见性或标 `Deprecated`（DESIGN §5.4）

## 2. 路径规范化（FR-2）

### AC-2.1 合法 `..` 段

**前置**：临时目录 `root`，文件 `root/pkg/util.go` 存在。

| 步骤 | 预期 |
|------|------|
| `ResolvePath("pkg/../pkg/util.go")` | 成功，返回绝对路径指向 `util.go` |
| `read_file` path=`pkg/../pkg/util.go` | 成功读取 |
| `grep` path=`pkg/../pkg` | 可搜索该目录 |
| `apply_patch` 目标 `pkg/../pkg/foo.go` | 成功（合法 `..` 段） |

### AC-2.2 逃逸仍拒绝

| 输入 | 预期 |
|------|------|
| `../outside` | `permission denied` / outside workspace |
| 绝对路径指向工作区外 | 拒绝 |
| symlink 指向工作区外（现有 fixture） | 拒绝 |

### AC-2.3 敏感路径规范化后仍拒绝

| 输入 | 预期 |
|------|------|
| `.env` | S3 拒绝 |
| `pkg/../.env` | S3 拒绝（解析后为 `root/.env`） |
| shell `cat pkg/../.env` | 拒绝 |

### AC-2.4 误拒修复

| 输入 | 预期 |
|------|------|
| 文件名 `a..b.txt` | 允许（若文件存在且非敏感） |
| `list_dir` path=`.` | 列举工作区根 |
| `list_dir` 默认空 path | 同 `.` |

### AC-2.5 shell 不误拦 git / go 惯例（v0.1.1 回归）

**背景**：v0.1.1 对含 `..` 子串的 shell token 报 `permission denied: shell path not allowed`。

| 命令 | permission 层（v0.1.2） | 说明 |
|------|-------------------------|------|
| `git diff origin/main...v0.1.1 --stat` | **放行** | token 解析后在区内 |
| `git log origin/main..v0.1.1 --format=…` | **放行** | 同上 |
| `go test ./... 2>&1` | **放行** | `./...` 非路径遍历 |
| `git diff main...v0.1.1 --stat` | **放行** | 即使本地无 `main` ref、git 后续失败，permission 不得拦截 |

**区分**：验收 permission 时用 `Engine.Check("shell", …)` 或 `checkShellDenylistPaths` 断言 **无 ErrDenied**；不要求 git/go 命令本身 exit 0。

## 3. API 收敛（FR-1）

### AC-3.1 工具层

| 检查 | 预期 |
|------|------|
| `grep` / `list_dir` / `diagnostics` / `filecandidate` | 使用 `perm.SkipSensitiveAbs` 或 `CheckReadablePath`，无 `permission.IsSensitiveAbs` 直接调用 |
| `globmatch` | **无** `IsSensitiveAbs`；Walk 经注入 `skipDir`（FR-1.8） |
| `filecandidate.ValidateGlobMatches` | 使用 `CheckAbsPath(abs, PathBoundary)`（越界 error；敏感路径由 `MakeFileCandidate` + `SkipSensitiveAbs` skip） |
| `patch/validate.go`、`patch/parser.go` | 不 import `internal/workspace` |

### AC-3.2 shell

| 命令 | 预期 |
|------|------|
| `cat readme.txt` | 允许（auto） |
| `cat pkg/../readme.txt` | 允许 |
| `cat ../secret` | 拒绝（`isOutsideWorkspaceErr`） |
| `echo hi 2>/dev/null` | 允许（非路径 token） |
| `git diff origin/main...v0.1.1 --stat` | 允许（见 AC-2.5） |
| `git log origin/main..v0.1.1` | 允许 |
| `go test ./...` | 允许 |

### AC-3.3 patch

| 步骤 | 预期 |
|------|------|
| `apply_patch` 修改工作区内文件 | 成功 |
| patch 含 `../escape` 目标 | 拒绝 |
| patch 修改 `.env` | S3 拒绝 |
| patch 目标 `pkg/../pkg/foo.go` | 成功（FR-2.7） |

## 4. MCP 结果落盘（FR-4）

### AC-4.1 spill 文件

**前置**：`tool_result_max_chars=100000`，stub MCP 返回 150000 字符。

| 步骤 | 预期 |
|------|------|
| 执行 MCP 工具 `tool_call_id=call_abc` | 创建 `…/mcp-result/<session>/call_abc.txt`（stem 与 id 一致时） |
| 文件内容 | 完整持久化正文（`FormatToolResult` 包装后），字符数 ≈ MCP 原始 150k + 包装开销 |
| 文件权限 | 0600 |
| session `messages.content` | 正文 + hint **合计** ≤ `tool_result_max_chars`（FR-4.14） |

### AC-4.2 未超长

| 步骤 | 预期 |
|------|------|
| MCP 返回 50k | spill 文件 50k；session 50k；**无** spill hint |

### AC-4.3 内建回归

| 步骤 | 预期 |
|------|------|
| `grep` 返回 150k | 无 `mcp-result/` 新文件；session 带通用 `TruncateSuffix` |

### AC-4.4 read_file 读取 spill（FR-4.12）

**前置**：MCP 调用已产生 `~/.ds-code/projects/<id>/mcp-result/<session_id>/call_abc.txt`。

| 步骤 | 预期 |
|------|------|
| `read_file` path=spill 绝对路径 | 成功，内容为完整 MCP 持久化正文 |
| `read_file` 同 session 下另一 spill 绝对路径 | 成功 |
| `read_file` spill **相对路径** | **拒绝** |
| `read_file` project 数据目录内非 `.txt` 文件（如 `sessions.db`） | **拒绝** |
| `read_file` **其他 session** 的 spill（同 project） | **拒绝**（FR-4.12） |
| `read_file` 其他 project 的 `mcp-result/…` | 拒绝 |
| worktree 子代理 spill | 落在主 `cfg.ProjectRoot` 的 `project_id` 下；**子代理**在自身回合内 `read_file` **成功**；**父** session `read_file` **拒绝**（与 AC-4.7 一致） |

### AC-4.5 spill 写入失败（FR-4.13）

| 步骤 | 预期 |
|------|------|
| stub `Save` 返回 error | session 无 spill hint；`mcp result spill failed` Warn 日志 |
| session 内容 | 普通 `TruncateToolResult`，与内建工具超长行为一致 |

### AC-4.6 MCP server 自截断（说明）

| 检查 | 预期 |
|------|------|
| CRG 返回 `Results truncated: showing 500 of 875` | spill 与 session 均含该 JSON；875 条完整列表不在 spill 内 |

### AC-4.7 子代理 spill 可读（FR-4.15）

| 步骤 | 预期 |
|------|------|
| 子代理 `task` 内 MCP 产生 spill | 文件在 `mcp-result/<子代理 session_id>/` |
| 父 Agent `read_file` 该 spill 绝对路径 | **成功**（同 project 数据目录） |
| 子代理同回合 `read_file` 该 spill | 成功 |

### AC-4.8 shell 不可读 spill（FR-4.17）

| 步骤 | 预期 |
|------|------|
| `shell command="cat <spill 绝对路径>"` | permission 拒绝 |
| 同路径 `read_file` | 成功（project 数据目录） |

### AC-4.9 MCP 失败不写 spill（FR-4.16）

| 步骤 | 预期 |
|------|------|
| stub MCP 返回 `isToolErrorBody` 正文 | 无 `mcp-result/` 新文件 |
| session 内容 | 错误正文或普通截断；**无** spill hint |

### AC-4.10 hint 预算与可读路径（FR-4.14）

| 步骤 | 预期 |
|------|------|
| 超长 MCP + 正常 spill path | `len(session content) ≤ tool_result_max_chars` |
| 从 session hint 提取路径 | `read_file path=<hint 内路径>` **成功**，正文与 spill 文件一致 |
| stub 极长绝对路径（测 `shortenSpillPathForHint`） | 仍满足 `len ≤ max`；hint 路径**非** `~`、**非**不可解析尾部；`read_file` 成功 |
| hint 路径含 `~` | **不得出现**（`shortenSpillPathForHint` 禁止） |
| `budget=0`（hint 占满 max） | session 内容仅为 hint；hint 内路径仍 `read_file` 成功 |

### AC-4.11 spill 文件名消毒（FR-4.4）

| 步骤 | 预期 |
|------|------|
| MCP `tool_call_id="call/foo"` | spill 文件 `call_foo.txt`（非 `call/foo.txt`） |
| 超长时 hint | 指向 `…/call_foo.txt` 绝对路径；`read_file` 成功 |

### AC-4.12 空 `tool_call_id` → ULID（FR-4.4）

| 步骤 | 预期 |
|------|------|
| stub LLM 返回 `tool_calls[].id=""` 的 MCP 调用 ×2（同 session） | 创建两个不同 `…/mcp-result/<session>/<ULID>.txt` |
| 超长时 session hint | 各 hint 绝对路径含对应 ULID 文件名 |
| `read_file` 任一 hint 路径 | 返回该 call 的完整 spill 正文 |

### AC-4.13 同 `tool_call_id` 覆盖（FR-4.19）

| 步骤 | 预期 |
|------|------|
| 同 session、同非空 `tool_call_id` 两次成功 MCP | 仅一个 spill 文件；第二次**覆盖**第一次 |
| `read_file` 该 spill | 返回**第二次**完整正文 |

### AC-4.14 compact 后 spill 不可发现（已知限制）

**前置**：MCP 产生 spill 且 hint 出现在 tool 消息；随后触发 compact，旧 tool 消息被摘要替换。

| 步骤 | 预期 |
|------|------|
| compact 后 API 上下文 | **无**旧 spill hint |
| `grep`/`glob`/`list_dir` 枚举 `mcp-result/` | **不可**（工作区外 + 无 spill 例外） |
| 磁盘 | spill 文件**仍在** |
| 用户已知绝对路径 `read_file` | **仍成功**（同 session） |
| 人工定位 `session_id` | 可查 `sessions.db` `sessions` 表或 TUI `/sessions`；目录 `mcp-result/<session_id>/` 列出 spill 文件 |

### AC-4.15 子代理 `agents/` 摘要 spill（FR-4.7）

**前置**：子代理 `task` 摘要超长，产生 `…/agents/<session>/<toolCallID>.output`；父 Agent `task` 返回含 `output_file` 绝对路径 + `SavedResultHint`。

| 步骤 | 预期 |
|------|------|
| 父 Agent `read_file path=<output_file>` | **成功** |
| 父 Agent `read_file` 子 session `mcp-result/` spill | **成功**（同 project 数据目录） |
| `shell cat <output_file>` | permission 拒绝 |

### AC-4.16 spill 与权限模式（NFR-22）

**前置**：本 session 已有 `mcp-result/` spill 绝对路径。

| 模式 | `read_file` spill | `read_file path=".env"` |
|------|-------------------|-------------------------|
| `readonly` | **成功**，无 ask 弹窗 | S3 拒绝 |
| `ask` | **成功**，无 ask 弹窗 | ask 或拒绝（依现网） |
| `auto` | **成功** | S3 拒绝 |

## 5. MCP 调用参数（FR-5）

### AC-5.1 TUI 展示参数

**前置**：注册 MCP 工具 `semantic_search_nodes`（server `code-review-graph`）。

| 步骤 | 预期 |
|------|------|
| 模型调用且 args=`{"query":"permission","limit":10}` | TUI 工具 args 行含 `MCP code-review-graph · semantic_search_nodes` **且** 含 `query`、`limit` |
| args=`{}` 或空 | 仅 `MCP … · tool`，不追加 `{}` |

### AC-5.2 debug 日志

| 条件 | 预期 |
|------|------|
| `-vv`，未开敏感日志 | `mcp call tool` 含 `args_preview`（≤200 字符） |
| `-vv --allow-log-sensitive-data` | 另含完整 `args` 字段 |
| 任意 | 保留 `args_len`、`result_chars`、`duration_ms` |

### AC-5.3 回归

| 检查 | 预期 |
|------|------|
| `read_file` TUI 行 | 仍为 `Read filename`，行为不变 |
| 内建工具 debug `tool registry execute` | 仍仅 `args_chars`，不输出完整 args |

### AC-5.4 非交互模式

| 条件 | 预期 |
|------|------|
| `ds-code -p "…"` + `-vv` 调用 MCP | 无 TUI 工具块；日志含 `args_preview` |
| 同上未开敏感日志 | 无完整 `args` 字段 |

### AC-5.5 legacy `mcp__*` 与 TUI 敏感策略（FR-5.8–5.9）

| 步骤 | 预期 |
|------|------|
| 历史工具名 `mcp__graph__semantic_search_nodes` + args | TUI 含 JSON 参数摘要 |
| 未开 `--allow-log-sensitive-data` | TUI **仍**展示参数（≤400 字符）；debug 无完整 `args` |
| 侧栏 `chattool` | MCP 参数 truncate 至 60 字符 |

## 6. 搜索路径可见性（FR-6）

### AC-6.1 不再遵循 `.gitignore`

**前置**：临时项目根，`.gitignore` 含 `ignoredpkg/`，其下有 `ignored.go`。

| 步骤 | 预期 |
|------|------|
| `grep` pattern=`package` path=`ignoredpkg` | **匹配** `ignored.go` |
| `glob` pattern=`ignoredpkg/**/*.go` | 含 `ignoredpkg/ignored.go` |
| `list_dir` path=`ignoredpkg` | 列出该目录 |

### AC-6.2 `.git` 与用户 `skip_dirs`

| 步骤 | 预期 |
|------|------|
| `glob **/*` path=`.` | **不**匹配 `.git/` 下任何路径 |
| `list_dir` path=`.` | 不列 `.git/`（或 walk 未进入） |
| `grep path=.git` | **空结果** / 不进入 `.git`（FR-6.14；**无**显式 path 例外） |
| `list_dir path=.git` | **空** / 不进入（FR-6.14） |
| `skip_dirs: ["node_modules"]`，`glob **/*` path=`.` | **不**含 `node_modules/`；Walk **不**进入该目录（FR-6.14） |
| `skip_dirs: ["node_modules"]`，`grep path=node_modules` | **允许**（模型显式 path） |
| `skip_dirs: ["node_modules"]`，`glob pattern=**/* path=node_modules` | **允许**进入该子树 |
| `skip_dirs: ["node_modules"]`，`list_dir path=node_modules` | **允许**列出（显式 path 穿透 skip_dirs） |
| 无 `skip_dirs`，`glob **/*` | **可**含 `node_modules/`（若存在） |

### AC-6.3 无框架内置 skip

| 步骤 | 预期 |
|------|------|
| 存在 `.github/workflows/ci.yml` | `grep` / `glob` **可**发现（不因 `.` 前缀 blanket 跳过） |
| `read_file` path=`.github/workflows/ci.yml` | 成功（非 S3） |

### AC-6.4 安全与显式读取回归（Agent 枚举）

| 检查 | 预期 |
|------|------|
| S3 路径（`.env`） | Agent 枚举仍 skip |
| `read_file` 工作区内显式路径 | 仍 S2 + S3 |
| `read_file` 本 session `mcp-result/` spill | 成功（FR-4.12） |
| 用户 `@node_modules/foo/` | **不受** `skip_dirs` 影响，按 FR-6.9 展开 |

### AC-6.5 Plan 模式 / 子代理 explore（FR-6.1）

| 步骤 | 预期 |
|------|------|
| `--plan` 下 `grep` | 不读 `.gitignore`；`.git` 仍 skip |
| 子代理 `task` explore 工具集 | 同上（`setup.RegisterReadOnly` 注入 `searchskip`） |

### AC-6.6 `.ds-code` 等项目目录可见（FR-6.12）

| 步骤 | 预期 |
|------|------|
| 存在 `.ds-code/config.yaml` | `glob **/*` path=`.` **可**匹配（除非用户配 `skip_dirs`） |

### AC-6.7 `@dir/` / `@file` 用户显式引用（FR-6.9–6.10）

| 步骤 | 预期 |
|------|------|
| `.gitignore` 含 `ignored/`，用户 `@ignored/` | 展开（预算内） |
| 项目 `skip_dirs` 含 `node_modules`，用户 `@node_modules/pkg/` | **仍展开**（`skip_dirs` 不作用于 `@dir/`） |
| 用户 `@.github/workflows/` | 展开 |
| 用户 `@.env` 或 `@dir/` 含 `.env` | **可读**（不做 S3 拒绝；US-8） |
| `@.env` 展开后 | `read_file path=".env"` 仍拒；shell `cat .env` 仍拒 |
| `@../outside/` | 仍拒（S2） |
| 超过 `at_dir_max_files` | 提示文件过多，非静默过滤 |

### AC-6.8 `diagnostics` 与 `skip_dirs`

**前置**：`skip_dirs: ["node_modules"]`，LSP 对 `node_modules/pkg/index.ts` 有诊断。

| 步骤 | 预期 |
|------|------|
| `diagnostics` path=`.` | 结果**不含** `node_modules/` 下条目 |
| `diagnostics` path=`node_modules/pkg` | **允许**（显式 path，若 LSP 有报告） |

### AC-6.9 `list_dir` 仍 skip S3（FR-6.4）

| 步骤 | 预期 |
|------|------|
| `list_dir` path=`.` | 不列出 `.env` 等 S3 条目（`SkipSensitiveAbs`） |

### AC-6.10 `skip_dirs` 非法配置（FR-6.15）

| 配置 | 预期 |
|------|------|
| `skip_dirs: ["../outside", "/etc", ""]` | 加载 `Warn`；非法条目忽略；合法条目仍生效 |
| `skip_dirs: ["node_modules"]` | 正常生效 |

### AC-6.11 工具描述（FR-6.7）

| 检查 | 预期 |
|------|------|
| `grep` `DescGrep` / `grep.md` | **不含**「遵循 .gitignore」；提及 `.git` + `skip_dirs` + 模型应收窄 path |
| `glob` / `list_dir` 同类文档 | 同上 |

### AC-6.12 `diagnostics` 始终过滤 `.git`（DESIGN §14.3）

**前置**：LSP 对 `.git/` 下某路径有诊断报告。

| 步骤 | 预期 |
|------|------|
| `diagnostics` path=`.` | 结果**不含** `.git/` 下条目 |
| `diagnostics` path=`.git` | **空** / 不展示 `.git` 下诊断（**无**显式 path 例外，与 `grep path=.git` 一致） |

## 7. TUI 应用内选中与剪贴板（FR-7）

### AC-7.1 鼠标选区与自动复制

**前置**：macOS、Linux 桌面或 Windows/WSL（`pbcopy` / `wl-copy` / `xclip` / `clip.exe` 可用）；交互 TUI 中已有 Agent 回复（含代码块）。

| 步骤 | 预期 |
|------|------|
| 在聊天 viewport 内拖拽选中一段回复 | 选区高亮可见 |
| 松开鼠标 | 系统剪贴板更新；TUI 短暂提示复制成功 |
| 粘贴到纯文本编辑器 | 内容与选中范围一致；**无** ANSI 转义序列 |
| 粘贴代码块 | 保留换行与缩进；不含 lipgloss 边框字符（若 UI 有装饰） |

### AC-7.2 回合进行中与历史

| 步骤 | 预期 |
|------|------|
| Agent 回合进行中（`Working…`） | 滚轮上翻后仍可选中已渲染历史并复制 |
| 流式输出进行中已建立选区 | 选区**可能**随新渲染行错位；不追未到达 token（FR-7.9） |
| 按 Esc 取消回合 | 取消优先；不误触复制 |

### AC-7.3 工具面板

**前置**：`Ctrl+T` 打开工具日志面板。

| 步骤 | 预期 |
|------|------|
| 在工具面板内拖拽选中 | 可选中并复制（FR-7.2） |

### AC-7.4 Copy on select 关闭

**前置**：`tui.copy_on_select: false` 或 `/config` 关闭。

| 步骤 | 预期 |
|------|------|
| 拖拽选中后松手 | **不**自动写剪贴板 |
| `Ctrl+Shift+C`（Linux/Windows）或 `Cmd+Shift+C`（macOS） | 将当前选区写入剪贴板 |

### AC-7.5 剪贴板不可用

**前置**：无 `pbcopy`/`xclip` 等且 OSC 52 被终端拒绝（或 stub 测试）。

| 步骤 | 预期 |
|------|------|
| 完成选区并触发复制 | TUI 显示失败提示；进程不 panic、不卡死 |

### AC-7.6 非交互回归

| 步骤 | 预期 |
|------|------|
| `ds-code -p "hello"` | stdout 仍可用终端原生方式选中（FR-7.13） |

### AC-7.7 浮层冲突（FR-7.8，P1）

**前置**：打开 `/help` 或权限 prompt 浮层。

| 步骤 | 预期 |
|------|------|
| 在聊天 viewport 拖拽 | **不**建立聊天选区（首期：禁用聊天选区即可） |
| 关闭浮层后 | 聊天选区恢复正常 |

### AC-7.8 复制可见 MCP 参数（FR-5.9 + NFR-18）

| 步骤 | 预期 |
|------|------|
| 选中含 MCP 工具行的 viewport 文本并复制 | 剪贴板含**已渲染** JSON 参数摘要（≤400 字符） |
| 复制内容 | **不含**未显示的 debug-only 字段 |

| 复制内容 | **不含**未显示的 debug-only 字段 |

## 8. TUI 平滑滚动（FR-9）

**前置**：交互 TUI 中聊天区内容超过一屏（多轮 Agent 对话）；终端支持鼠标滚轮（`tea.WithMouseCellMotion` 已启用）。

### AC-8.1 滚轮平滑多页

| 步骤 | 预期 |
|------|------|
| 在聊天 viewport 内快速向下滚轮 | 可见**连续中间帧**；非一次跳半屏以上 |
| 慢速滚轮 | 手感连贯；无明显抖动 |
| 连续快速滚轮（burst） | pending 累加后分多帧释放；最终到位；不 panic |

### AC-8.2 翻页瞬时跳转

| 步骤 | 预期 |
|------|------|
| 按 `PgDn` / `PgUp` | **瞬时**半页跳转；无 drain 动画拖尾 |
| 滚轮 drain 进行中按 `PgUp` | pending **立即清空**；按翻页规则瞬时跳位 |
| `↑` / `↓` 单行滚动 | 瞬时 1 行；不走 pending drain |

### AC-8.3 工具面板与浮层

| 步骤 | 预期 |
|------|------|
| `Ctrl+T` 打开工具面板，在面板内滚轮 | `toolVP` 平滑滚动 |
| 打开 `/help` 浮层时滚轮 | **忽略**（与 FR-7.8 一致） |
| Agent 回合进行中滚轮上翻 | 可浏览历史（FR-9.10 / FR-7.9） |

### AC-8.4 选区与 HP 渲染（FR-9.5–9.6）

| 步骤 | 预期 |
|------|------|
| 无选区时滚轮 | HP 路径生效；帧率优于全量重绘 |
| 拖拽建立选区 | 选区高亮正常；可边滚边选 |
| 选区活跃时滚轮 | 允许滚动；高亮不丢失（HP 临时关闭可接受） |

### AC-8.5 终端 profile（FR-9.3，P1）

| 环境 | 预期 |
|------|------|
| iTerm2 / Ghostty / Terminal.app | proportional drain（大 burst 多帧追平） |
| VS Code / Cursor 集成终端 | adaptive drain（小 pending 一帧释放） |

### AC-8.6 环境变量（FR-9.8，P1）

| 步骤 | 预期 |
|------|------|
| `DS_CODE_SCROLL_SPEED=0.5` | 滚轮累积步长约为默认一半 |
| `DS_CODE_SCROLL_SPEED=2` | 滚轮累积步长约为默认两倍 |

## 9. `read_file` 仅读文本（FR-8）

### AC-9.1 非文本拒绝

**前置**：工作区内存在 `assets/logo.png`（有效 PNG 头）与 `main.go`。

| 步骤 | 预期 |
|------|------|
| `read_file` path=`assets/logo.png` | tool **错误**；消息含 `无法读取非文本文件` 与用户传入的 `path` |
| `read_file` path=`main.go` | 成功；输出含行号前缀 |
| debug 日志（`-v` 及以上） | PNG 拒绝时含 `read_file skipped non-text file` 及 `path`、`abs` |
| 错误响应 | **不**含 PNG 二进制或 base64 片段 |

### AC-9.2 边界与回归

| 步骤 | 预期 |
|------|------|
| `read_file` 空文件 `empty.txt` | 成功（空输出或 offset 提示） |
| `read_file` MCP spill 绝对路径 | 成功（FR-4.12 + FR-8.6） |
| `read_file` `.pdf` / `.wasm` / blocked 扩展名 | 拒绝（与 `IsTextFile` → `IsSearchable` blocklist 一致） |
| `@file logo.png`（用户引用） | **仍展开**（不经 `read_file`，FR-8.7） |
| S3 路径 `.env` | 权限拒绝（**先于**文本判定） |
| 超大非文本文件 | 若 `Stat` 未超限但 `IsTextFile` 为 false → 拒绝（**不**读入全文） |

### AC-9.3 工具描述（FR-8.8）

| 检查 | 预期 |
|------|------|
| `read_file.md` | 说明非文本文件将被拒绝；与 grep/glob 文本策略一致 |
| `DescReadFile` | 提及无法读取二进制/媒体文件（或等价表述） |

## 10. 测试清单

- [ ] `TestValidateRel_allowsDotDotInside`（新）
- [ ] `TestValidateRel_rejectsTraversal`（仍用 `../outside`）
- [ ] `TestValidateRel_rejectsSymlinkEscape`（回归）
- [ ] `TestEngine_checkReadablePath_dotDotInside`（新）
- [ ] `TestEngine_shell_dotDotInside`（新）
- [ ] `TestEngine_resolvePath_blocksTraversal` 更新断言（不误拒 `foo/../bar`）
- [ ] `TestCheckPathCandidate_allowsGitRevisionRange`（新：`origin/main..v0.1.1`、`origin/main...v0.1.1`）
- [ ] `TestCheckPathCandidate_allowsGoTestEllipsis`（新：`./...`）
- [ ] `TestCheckPathCandidate_blocksRelativeTraversal`（新：`../outside`）
- [ ] `TestEngine_shell_gitDiffGoTest`（新：整命令 `Check` 无 ErrDenied）
- [ ] `TestMCPResultStore_Save`（新）
- [ ] `TestFinalizeToolResult_mcpSpillAndTruncate`（新）
- [ ] `TestSpillCallFilename_emptyUsesDistinctULID`（新：FR-4.4，两次空 id 不同文件名）
- [ ] `TestFinalizeToolResult_mcpUnderLimit`（新）
- [ ] `TestFinalizeToolResult_builtinNoSpill`（回归）
- [ ] `TestFinalizeToolResult_mcpSpillHintBudget`（新：正文+hint ≤ max）
- [ ] `TestFinalizeToolResult_spillSaveFailed`（新：无 hint + Warn）
- [ ] `TestCheckReadablePath_mcpSpillFile`（新：本 session spill 可读）
- [ ] `TestCheckReadablePath_mcpSpillOtherSession`（新：同 project 其他 session **可读**）
- [ ] `TestCheckReadablePath_otherProjectDenied`（新：其他 project_id 拒绝）
- [ ] `TestCheckReadablePath_agentsOutputAllowed`（新：FR-4.7，`agents/*.output` 可读）
- [ ] `TestCheckReadablePath_sessionsDB`（新：project 数据目录 `.db` 可读）
- [ ] `TestFinalizeToolResult_mcpErrorNoSpill`（新：FR-4.16）
- [ ] `TestFinalizeToolResult_mcpSuccessBodyStartsWithError`（新：`isToolErrorBody` 不误判成功响应）
- [ ] `TestEngine_shell_deniesSpillAbsPath`（新：FR-4.17）
- [ ] `TestCheckReadablePath_mcpSpillOtherSessionReadable`（新：FR-4.15 父可读子 spill）
- [ ] `TestSpawnExecute_worktreeSetsProjectRoot`（新：NFR-14，inherit worktree）
- [ ] `TestSpawnExecute_readonlyWorktreeSetsProjectRoot`（新：readonly worktree `ProjectRoot`）
- [ ] `TestSpawnExecute_childInheritsMCPResults`（新：FR-4.8）
- [ ] `TestFinalizeToolResult_mcpHintBudgetLongPath`（新：FR-4.14 边界）
- [ ] `TestFinalizeToolResult_mcpHintPathReadable`（新：hint 路径 `read_file` 成功）
- [ ] `TestShortenSpillPathForHint_noTildeNoTruncate`（新：禁止 `~` / 不可解析尾部）
- [ ] `TestMCPResultStore_overwriteSameCallID`（新：FR-4.19）
- [ ] `TestSpillCallFilename_sanitizesSlashes`（新：FR-4.4，`call/foo` → `call_foo.txt`）
- [ ] `TestCheckReadablePath_mcpSpillRelativePathDenied`（新：相对路径拒绝）
- [ ] `TestCheckReadablePath_projectDataDirDenied`（新：目录路径拒绝）
- [ ] `TestCheckReadablePath_mcpSpillBudgetZeroHintOnly`（新：FR-4.14 budget=0）
- [ ] `TestCheckReadablePath_agentsOutputAllowed`（新：FR-4.7）
- [ ] `TestCheckReadablePath_mcpSpillReadonlyMode`（新：NFR-22，readonly 无 ask）
- [ ] `TestCompactAPIContext_spillHintNotInSummary`（新：AC-4.14）
- [ ] `TestRunEphemeral_noMCPSpill`（新：NFR-20 / FR-3.11）
- [ ] `TestApplyPatch_allowsDotDotInside`（新：FR-2.7）
- [ ] `TestGlobmatch_skipDirSkipsNodeModules`（新：FR-6.14 Walk 不进入 skip_dirs）
- [ ] `TestGlobTool_explicitSkipDirPath`（新：AC-6.2，`glob path=node_modules` 穿透）
- [ ] `TestListDir_explicitSkipDirPath`（新：AC-6.2，`list_dir path=node_modules` 穿透）
- [ ] `TestGrepTool_explicitGitPathEmpty`（新：FR-6.14）
- [ ] `TestSearchSkip_invalidConfigEntries`（新：FR-6.15）
- [ ] `TestDiagnosticsTool_respectsSkipDirs`（新）
- [ ] `TestDiagnosticsTool_alwaysFiltersGit`（新：AC-6.12）
- [ ] `TestListDir_skipsSensitiveEntries`（新：FR-6.4）
- [ ] `TestFormatMCPCallDisplay_withArgs`（新）
- [ ] `TestFormatMCPCallDisplay_emptyArgs`（新）
- [ ] `TestLogMCPCall_argsPreview`（新）
- [ ] `TestSearchSkip_alwaysSkipsGit`（新）
- [ ] `TestSearchSkip_userSkipDirs`（新）
- [ ] `TestGrepTool_noGitignoreFilter`（新）
- [ ] `TestGlobTool_respectsSkipDirs`（新）
- [ ] `TestAtExpander_dirIgnoresSkipDirs`（新：`@dir/` 不受 skip_dirs）
- [ ] `TestAtExpander_dirIgnoresGitignore`（新：`@dir/` 不受 gitignore）
- [ ] `TestAtExpander_dirAllowsSensitive`（新：`@.env` 可读）
- [ ] `TestAtExpander_dirAllowsNodeModules`（新）
- [ ] `TestAtExpander_sensitiveDenied` **删除或改写**（v0.1.2：`@.env` 应成功）
- [ ] `TestAtExpander_dirSkipsSensitiveFiles` **删除或改写**（v0.1.2：`@dir/` 不 skip S3）
- [ ] `TestAtExpander_dirSkipsSensitiveDirectory` **删除或改写**（v0.1.2：`@./` 可进入 `.ssh`）
- [ ] `TestGrepTool_respectsGitignoreInSubdirectory` **删除**（v0.1.2 不再遵循 gitignore）
- [ ] `TestGrepTool_planModeNoGitignore`（新：Plan 模式）
- [ ] `TestDisplay_MCPCallDisplay_legacyPrefix`（新：FR-5.8）
- [ ] `TestSelection_plainTextFromStyled`（新：FR-7.3，ANSI 剥离）
- [ ] `TestClipboard_write_macOS` / `TestClipboard_write_linux`（新：FR-7.6，按 GOOS 条件）
- [ ] `TestSelection_viewportHitTest`（新：FR-7.1，坐标→文本）
- [ ] `TestSelection_copyOnSelect`（新：FR-7.4）
- [ ] `TestSelection_runningTurnAllowsHistory`（新：FR-7.9）
- [ ] `TestSelection_overlayDisablesChatSelect`（新：FR-7.8）
- [ ] `TestSelection_copiesVisibleMCPArgs`（新：FR-5.9 + NFR-18）
- [ ] `TestGrepTool_descNoGitignore`（新：FR-6.7）
- [ ] `TestIsTextFile_matchesIsSearchable`（新：FR-8.4，委托一致）
- [ ] `TestReadFile_rejectsNonText`（新：FR-8，PNG/PDF）
- [ ] `TestReadFile_allowsEmptyFile`（新：FR-8.5）
- [ ] `TestReadFile_allowsMCPSpill`（新：FR-8.6，与 spill 集成）
- [ ] `TestReadFile_nonTextLogsInfo`（新：FR-8.3，log capture）
- [ ] `TestDrain_proportional_*` / `TestDrain_adaptive_*`（新：FR-9.3，`internal/ui/tui/scroll`）
- [ ] `TestWheelScroll_*` / `TestScroll_jumpBy_clearsPending`（新：FR-9.1–9.2）

## 11. 手动验证

```bash
# 在项目根启动 ds-code
bin/ds-code --permission-mode auto

# 通过 Agent 或调试入口验证：
# 1. read_file path="internal/../internal/go.mod"  → 成功
# 2. read_file path="../etc/passwd"                 → 拒绝
# 3. list_dir path="."                              → 列出根目录

# shell（v0.1.1 误拦回归）：
# 4. shell command="git diff origin/main...HEAD --stat"  → permission 放行（git 是否成功另论）
# 5. shell command="go test ./... -count=1"              → permission 放行
# 6. shell command="cat ../outside"                      → permission 拒绝

# MCP（需求 2）：
# 7. 调用大结果 MCP 工具 → 检查 mcp-result/<session>/<stem>.txt 存在且完整
# 8. session tool 消息含 spill 路径 hint（仅超长时；正文+hint ≤ max）
# 9. read_file spill 绝对路径 → 返回完整 MCP 正文（非 session 截断版）
# 9b. read_file spill 相对路径 → 拒绝
# 10. 同 session 另一 spill → read_file 仍可读
# 11. 其他 session spill → read_file 拒绝
# 12. shell cat spill 绝对路径 → permission 拒绝；read_file 同路径成功
# 12b. tool_call_id 含 `/` → spill 为消毒 stem（如 call_foo.txt）

# MCP 参数（需求 3）：
# 13. -vv 下调用 MCP → 日志可见 args_preview
# 14. TUI 工具块 → 标题行含 JSON 参数摘要（侧栏可能仅 60 字符）
# 15. ds-code -p "…" -vv → 无 TUI，日志仍含 args_preview
# 16. legacy mcp__* 工具名 → TUI 仍展示参数

# 搜索可见性（需求 4）：
# 17. .gitignore 忽略目录下 grep → 可搜到
# 18. glob path=. → 不匹配 .git/
# 18b. grep path=.git → 空/不进入
# 19. 配置 skip_dirs: [node_modules] → glob Walk 跳过；@node_modules/ 仍可展开
# 20. .github/workflows/*.yml → grep 可发现
# 21. --plan 模式 grep → 同样不读 gitignore
# 22. diagnostics path=. + skip_dirs → 不展示 skip 目录下 LSP 诊断
# 22b. grep Desc / grep.md 不含「遵循 .gitignore」

# @dir/ / @file（FR-6.9–6.10）：
# 23. @ignored/（.gitignore 中）→ 应展开
# 24. @node_modules/some-pkg/ → 应展开
# 25. @.env → 应展开；read_file .env 仍拒

# 子代理 spill（FR-4.15）：
# 26. task 子代理 MCP spill → 父 read_file 成功；子代理回合内可读

# compact + spill（AC-4.14）：
# 26b. compact 后旧 spill hint 不在上下文；grep/list_dir 无法发现 mcp-result/
# 26c. session_id 可查 sessions.db 或 TUI /sessions

# agents/ 摘要 spill（AC-4.15）：
# 26d. 子代理超长 summary → task 返回 output_file + SavedResultHint；read_file 该路径成功

# 搜索 skip_dirs 显式 path（AC-6.2）：
# 26e. skip_dirs 含 node_modules → glob path=node_modules 仍允许

# diagnostics + .git（AC-6.12）：
# 26f. diagnostics path=.git → 空（即使 LSP 有 .git 下报告）

# TUI 复制（需求 5）：
# 27. 拖拽选中 Agent 回复 → 松手 → 粘贴到编辑器，内容与选中一致且无 ANSI
# 28. 回合进行中滚轮上翻 → 仍可复制历史
# 29. Ctrl+T 工具面板 → 可选中复制
# 30. tui.copy_on_select=false → 松手不复制；Ctrl+Shift+C 手动复制
# 31. 打开 /help 浮层时聊天区不可选（FR-7.8）
# 32. 复制 MCP 工具行 → 剪贴板含可见 JSON 参数摘要

# TUI 平滑滚动（需求 7）：
# 33. 长 transcript 快速滚轮 → 连续中间帧，非一跳多屏
# 34. PgUp/PgDn → 瞬时半页，无 drain 拖尾
# 35. 滚轮中按 PgUp → pending 清空并跳页
# 36. Ctrl+T 工具面板内滚轮 → 面板平滑滚动
# 37. /help 浮层打开时滚轮 → 忽略

# read_file 文本限制（需求 6）：
# 38. read_file path=*.png → 错误「无法读取非文本文件」；日志含 skipped non-text
# 39. read_file path=*.go → 成功
# 40. read_file spill .txt → 仍成功
```

## 12. 非目标确认

- [ ] `LoadSkill` 仍拒绝 skill 名含 `..`（未改行为）
- [ ] worktree slug 含 `..` 仍拒绝（未改行为）
- [ ] code-review-graph 内部 500 节点上限未在本仓库修改（见 AC-4.6）
- [ ] `shell` 内用户执行的 `rg`/`grep` 忽略策略不受 FR-6 影响（FR-3.6）
- [ ] `@file` / `@dir/` 不受 gitignore/S3/`skip_dirs` 过滤（FR-6.9–6.10）；`read_file` / shell 仍 S3
- [ ] **无**框架内置 skip；`.git` 始终 SkipDir（**含**显式 `path=.git`，FR-6.14）；`tools.search.skip_dirs` 可选（FR-6.3–6.5）
- [ ] `glob **/*` + `skip_dirs` Walk 阶段 SkipDir（FR-6.14）；`globmatch` 无 `IsSensitiveAbs`（FR-1.8）
- [ ] compact 后旧轮 spill hint **不**保留在 API 上下文；模型**无法** `list_dir`/`grep` 发现 `mcp-result/`（磁盘仍在；见 README 已知限制、AC-4.14）
- [ ] `RunEphemeral`（`/btw`）不写 spill（FR-3.11、NFR-20）
- [ ] `agents/*.output` 子代理摘要 spill：`read_file` **放行**（FR-4.7；`resolveProjectDataRead`）
- [ ] `audit.jsonl` 不记录 MCP `args_preview`（S10 不变）
- [ ] 每次成功 MCP 均写 spill（含未超长，FR-4.18）；同非空 `tool_call_id` 覆盖（FR-4.19）
- [ ] TUI transcript 刷回 scrollback / classic 无备用屏幕渲染器**未**实现（FR-3.7–3.8）
- [ ] `read_file` 非文本拒绝（`IsTextFile`）；`@file`/`@dir/` **不**应用文本判定（FR-8.7、FR-6.11）
- [ ] 子代理 prompt **不**展开 `@`（FR-3.10；主会话仍走 `AtExpander`）
- [ ] 非交互 `-p` 不引入 TUI 选区逻辑（FR-7.13）
- [ ] TUI 双击/键盘扩展选区（FR-7.10–7.11，P2）**未**实现时可勾选为延期
- [ ] TUI 浮层内独立选区（FR-7.8 完整版）**未**实现时可勾选为延期（首期仅禁用聊天选区）
- [ ] TUI React 式虚拟列表**未**实现（FR-9.12；`RenderCache` 已覆盖 block 缓存）
- [ ] `DS_CODE_SCROLL_SPEED` / 终端 profile 检测（FR-9.8、AC-8.5–8.6）**未**实现时可勾选为延期（P1）
