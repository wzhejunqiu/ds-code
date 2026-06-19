# ds-code v0.1.2 版本文档

> 版本：v0.1.2  
> 状态：设计中  
> 基线版本：v0.1.1  
> 更新日期：2026-06-20

## 概述

v0.1.2 聚焦八项增量：

1. **路径访问权限收敛**（需求 1）：统一经 `permission.Engine` 判定；`.` / `..` 先规范化解析再鉴权。
2. **MCP 结果落盘 + 上下文截断**（需求 2）：MCP 回注 LLM 仍受 `tool_result_max_chars` 限制；完整结果写入 `~/.ds-code/projects/<project_id>/mcp-result/<session_id>/<stem>.txt`（`<stem>` = `spillCallFilename(tool_call_id)`，可能与 LLM 原始 id 不同）；超长时 tool 消息提示 spill **绝对路径**；**模型可用 `read_file` 读取当前会话** spill 文件以获取完整 MCP 输出（**不可**跨 session 读取；须绝对路径，不支持 `~`）。
3. **MCP 调用参数可见**（需求 3）：交互 TUI 与 debug 日志（`-vv`）输出 MCP 调用 JSON 参数；非交互 `-p` 模式无 TUI，仅日志可观测。
4. **搜索路径不再遵循 `.gitignore`**（需求 4）：Agent 枚举工具**不读** `.gitignore`、**不设**框架默认 skip；**始终**跳过 `.git`（含显式 `path=.git`，与 `skip_dirs` 不同）；用户可通过 `tools.search.skip_dirs` 追加目录；其余噪声由**模型**收窄 `path`/`pattern`。用户显式 **`@file` / `@dir/`** 不受 gitignore / S3 / `skip_dirs` 约束（FR-6.9–6.10）；Agent 工具与 shell 仍受 S3。
5. **TUI 应用内选中与剪贴板**（需求 5）：交互 TUI 在备用屏幕模式下支持鼠标拖拽选区，松手写入系统剪贴板（纯文本、无 ANSI）；对齐 Claude Code fullscreen 复制体验（FR-7）。
6. **`read_file` 仅读文本**（需求 6）：非文本文件经 `textfile.IsTextFile` 判定后拒绝（v0.1.2 内部委托 `IsSearchable`），向模型返回明确错误并写 Info 日志（FR-8）；MCP spill `.txt` 与常规源码不受影响。
7. **TUI 平滑滚动**（需求 7）：滚轮 `scrollBy` 累加 pending、分帧 proportional/adaptive drain、翻页 `scrollTo` 瞬时跳转；启用 viewport `HighPerformanceRendering` 减少跨页全屏重绘（FR-9）。

## 文档索引

| 文档 | 说明 |
|------|------|
| [REQUIREMENTS.md](REQUIREMENTS.md) | 功能与非功能需求、用户故事、范围边界 |
| [DESIGN.md](DESIGN.md) | 模块设计、API 收敛、路径解析算法、迁移映射 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | 验收标准、测试要点、手动验证步骤 |
| [SECURITY-SYNC.md](SECURITY-SYNC.md) | SECURITY / CONFIG 同步草稿（发布前合入 v0.1.0 文档） |

## 背景与动机

v0.1.1 及更早版本中，路径安全依赖多层重复逻辑：

1. `internal/workspace.ResolveRel` 对相对路径做 `strings.Contains(rel, "..")` 子串拦截，导致 `foo/../bar` 等**解析后仍在工作区内**的合法路径被拒绝。
2. `permission.IsSensitiveAbs`、`wspkg.EnsureAbsUnder`、`wspkg.ValidateRel` 在 grep/glob/list_dir/diagnostics/patch/apply/filecandidate/context 等处被**直接调用**，策略演进时易遗漏。
3. `shell_sensitive_paths.checkPathCandidate` 在 `ResolvePath` 失败时额外用 `strings.Contains(rel, "..")` 判定，与 workspace 层语义不一致。
4. **v0.1.1 生产误拦**（shell 工具）：`git diff origin/main...v0.1.1`、`git log origin/main..v0.1.1`、`go test ./...` 等常见命令因 token 中含 `..` **子串**被报 `permission denied: shell path not allowed`——实为 git revision range / go 包模式，非路径遍历。
5. **v0.1.1 MCP 结果被截断**：`Runner.executeSingleTool` 对内建工具与 MCP 工具统一调用 `TruncateToolResult`（默认 `context.tool_result_max_chars=100000`），大 JSON（如 `get_impact_radius_tool`）在写入 `messages` 前被截断，模型在后续轮次看不到完整 MCP 输出。
6. **v0.1.1 MCP 参数不可见**：TUI 仅显示 `MCP code-review-graph · get_impact_radius_tool`，不展示 `{"query":…}` 等实参；debug 日志 `mcp call tool` 仅有 `args_len`，排障时需猜模型传了什么。
7. **v0.1.1 搜索过度依赖 `.gitignore`**：`grep` / `glob` / `list_dir` / `diagnostics` 经 `GitignoreMatcher` 过滤，Agent 无法感知被 `.gitignore` 或「隐藏目录」规则挡住的源码；**`@dir/` 亦被误伤**。v0.1.2 去掉 gitignore 与内置目录 skip，改由模型根据任务自行选择搜索范围。
8. **v0.1.1 TUI 输出不可复制**：TUI 使用 Bubble Tea 备用屏幕 + 鼠标捕获，终端原生拖拽复制失效；Claude Code fullscreen 已通过应用内选区 + 剪贴板解决，ds-code 尚无等价能力。
9. **v0.1.1 `read_file` 可读二进制**：`grep`/`glob` 经 `textfile.IsSearchable` 跳过二进制，但 `read_file` 无同等校验，模型误读 `.png`/`.wasm` 等会浪费 token 或得到乱码行。
10. **v0.1.1 TUI 滚轮不流畅**：鼠标选区接入后滚轮事件被拦截；若一次跳多行或缺少分帧 drain，长 transcript 滚动卡顿、不连贯（对标 Claude Code 多页平滑滚动体验缺失）。

本版本 **S2 工作区边界不变**；**S3 在 Agent 工具 / shell / `read_file` 路径上不变**，但新增两处显式例外：（1）用户提示词中的 **`@file` / `@dir/`** 仅校验 S2，可读取 `.env` 等敏感路径（FR-6.10）；（2）**MCP spill** 经 `read_file` 只读放行，且仅限**当前 session**（FR-4.12）。需求 1 修正路径实现；需求 2/3 改善 MCP 可观测性与结果完整性；需求 4 调整 Agent 枚举可见性策略；需求 5 补齐 TUI 复制体验；需求 6 对齐 `read_file` 与 Agent 枚举的文本判定；需求 7 补齐 TUI 多页平滑滚动。

## 变更摘要

### 需求 1：路径权限

| 领域 | v0.1.1 | v0.1.2 |
|------|--------|--------|
| `..` / `.` 处理 | 相对路径含 `..` 子串即拒绝 | `filepath.Clean` + join + symlink 解析后 `ensureUnder` |
| 路径权限入口 | 分散在 workspace / permission / 各 tool | **统一** `permission.Engine` 路径 API |
| 工具层敏感过滤 | 直接调 `IsSensitiveAbs` | 经 Engine 封装（或 package-private） |
| patch 路径校验 | 直接调 `wspkg.ValidateRel` / `EnsureAbsUnder` | 经 Engine 注入的 resolve 回调 |
| shell 常见命令 | `git …/main..branch`、`go test ./...` 误拦 | 规范化后区内则放行；真越界仍拒 |

### 需求 2：MCP 结果

| 领域 | v0.1.1 | v0.1.2 |
|------|--------|--------|
| MCP 回注 LLM | `tool_result_max_chars` 截断，无落盘 | **仍截断**；超长时 suffix 指向 spill 文件 |
| MCP 完整结果 | 截断后丢失 | 写入 `mcp-result/<session_id>/<stem>.txt`（空 LLM id → ULID） |
| `read_file` 读 spill | 不可读（路径在工作区外） | **可**读本 project **当前 session** 的 `mcp-result/<session_id>/*.txt`；跨 session / 跨 project 拒绝 |
| spill 写入失败 | — | 回退普通截断、无 hint；debug 日志 `Warn` |
| session 截断 + hint | — | 正文截断与 hint **合计** ≤ `tool_result_max_chars` |
| spill `project_id` | — | 始终 `hex(sha256(cfg.ProjectRoot))`；子代理 session 独立分目录 |
| 内建工具 | `tool_result_max_chars` | **不变**（无 spill） |
| TUI 结果预览 | `chattool` 本地截断 | **不变**（仅 UI） |
| spill 磁盘占用 | — | 不自动 GC（与会话/checkpoint 一致；用户可手动删除 `mcp-result/`） |
| MCP server 自截断 | 由 server 决定 | ds-code 原样写入 spill 文件后截断回注 |

### 需求 3：MCP 调用参数

| 领域 | v0.1.1 | v0.1.2 |
|------|--------|--------|
| TUI 工具行 | `MCP {server} · {tool}` | 同上 + **紧凑 JSON 参数**（主工具块 ≤400 字符；侧栏 `chattool` 仍 truncate 60） |
| TUI 敏感参数 | — | **有意**与 shell 命令行一致：TUI 可见性优先，**不**受 `--allow-log-sensitive-data` 门控 |
| debug 日志 | 仅 `args_len` | `args_preview`（默认截断至 200 字符）；`--allow-log-sensitive-data` 时完整 `args` |
| legacy `mcp__*` 历史工具名 | 仅展示 | TUI 同样展示 JSON 参数（FR-5.8） |
| 非交互 `-p` | 无 TUI | 同上 debug 日志；LLM `tool_calls` 不变 |
| LLM API | 已有 `tool_calls[].arguments` | **不变**（本需求面向用户可观测性） |

### 需求 4：搜索路径与可见性

| 领域 | v0.1.1 | v0.1.2 |
|------|--------|--------|
| 枚举过滤来源 | 加载 `.gitignore` + 隐藏目录（`.` 开头，除 `.ds-code`） | **不读** `.gitignore`；**无**框架内置 skip |
| 噪声目录 | gitignore / 隐藏规则 | **模型**收窄 `path`/`pattern`；用户可选 `tools.search.skip_dirs` |
| 仍保留 | S3 敏感路径、二进制/超大文件 | **不变** |
| `.git` | 常随 gitignore/遍历跳过 | **始终** SkipDir（含显式 `path=.git`；与 `skip_dirs` 显式 path 例外不同） |
| `glob **/*` + `skip_dirs` | — | Walk 阶段 SkipDir（`globmatch` 注入 `skipDir`，FR-6.14） |
| `read_file` / `@file` | 显式路径 | 工作区内仍 S2+S3；**另可**读本 session `mcp-result/` spill |
| `@file` / `@dir/` | 曾受 gitignore/S3 过滤 | 用户显式引用：**仅 S2**；见 FR-6.9–6.10 |
| `.ds-code/`、`/.github/` 等 | 隐藏目录常跳过 | Agent 枚举**可**发现（行为变更） |
| Plan / 子代理 explore | 同 gitignore | 同步改用 `searchskip`（`setup.go`） |
| 配置 | — | 可选 `tools.search.skip_dirs`（默认 `[]`） |
| spill 读取方式 | — | 仅 `read_file`（绝对路径）；`shell cat` 区外路径仍拒；`readonly`/`ask`/`auto` 均直接放行 |
| `agents/` 摘要 spill | — | **不**扩展 `read_file`；与 `mcp-result/` 并存（FR-4.7） |
| 子代理 spill | — | 独立 `session_id` 目录；父 Agent **不可**读；MCP 数据不向父传递 |
| compact 后 hint | — | 被摘要的旧轮 spill 路径**不**保留在 API 上下文（见已知限制） |

### 需求 5：TUI 应用内选中与剪贴板

| 领域 | v0.1.1 | v0.1.2 |
|------|--------|--------|
| 终端复制 | 备用屏幕下原生拖拽不可用 | 应用内鼠标选区 + 松手写剪贴板 |
| 复制内容 | — | plain text（剥离 ANSI/lipgloss） |
| 剪贴板后端 | — | `pbcopy` / `wl-copy` / `xclip` / OSC 52 等 |
| Copy on select | — | 默认开启；`tui.copy_on_select` 可关 |
| 回合进行中 | — | 仍可复制已渲染历史（滚轮上翻） |
| 工具面板 | — | `Ctrl+T` 打开时同样可选中复制（FR-7.2） |
| 非交互 `-p` | stdout 可选中 | **不变**（FR-7.13） |
| classic / transcript | — | **不在范围**（FR-3.7–3.8） |

### 需求 6：`read_file` 仅读文本

| 领域 | v0.1.1 | v0.1.2 |
|------|--------|--------|
| 二进制 / 媒体文件 | `read_file` 可读（乱码或浪费 token） | **拒绝**；返回 `read_file: 无法读取非文本文件: …` |
| 判定逻辑 | 无 | `textfile.IsTextFile`（v0.1.2 委托 `IsSearchable`；与 grep/glob 解耦便于后续分化） |
| 日志 | 无 | `Info`：`read_file skipped non-text file` + `path`/`abs` |
| MCP spill `.txt` | — | **仍可读**（ds-code 写入的 UTF-8 文本） |
| `@file` / `@dir/` | — | **不变**（不经 `read_file` 工具，FR-6.11） |
| 空文件 | 可读 | **仍可读**（`IsTextFile` 对 0 字节返回 true） |

### 需求 7：TUI 平滑滚动

| 领域 | v0.1.1 | v0.1.2（初版） | v0.1.2（目标） |
|------|--------|----------------|----------------|
| 滚轮滚动 | viewport 默认 3 行/ notch 或不可用 | 选区接入后滚轮失效或一跳多屏 | **`scrollBy` pending + 分帧 drain** |
| 翻页键 | viewport 瞬时跳页 | 同左 | **`scrollTo` 清空 pending 后瞬时跳** |
| 渲染 | 每帧全屏 lipgloss 重绘 | 同左 | **HP 模式 + `SyncScrollArea` 边缘更新** |
| 终端适配 | — | — | 原生 proportional / 集成终端 adaptive |
| 速度调节 | — | — | 可选 `DS_CODE_SCROLL_SPEED` |
| 虚拟列表 | — | — | **不引入**（`RenderCache` 已按 block 缓存） |

## 已知限制

以下为本版本**有意接受**或**暂不处理**的行为，实现与验收须与之一致：

| 限制 | 说明 |
|------|------|
| **两套 spill 目录并存** | `mcp-result/<session>/<stem>.txt`（MCP 工具结果）与 `agents/<session>/<toolCallID>.output`（子代理摘要 spill）物理目录不同；`read_file` 经 `resolveProjectDataRead` **统一放行**本 project 数据目录 `~/.ds-code/projects/<project_id>/` 下 regular file（见 DESIGN §12.8b）。 |
| **spill 仅 `read_file`** | spill 位于 `~/.ds-code/projects/…`（工作区外）。`shell cat <spill 绝对路径>` 在 permission 层拒绝；MCP / 子代理 tool result 附 `SavedResultHint` 引导模型 `read_file`。`readonly`/`ask`/`auto` 下读 project 数据目录 **均直接放行**（NFR-22）。 |
| **子代理 MCP 不向父传递** | 子代理有独立 `session_id`；MCP spill 写入 `mcp-result/<子代理 session>/`。父 Agent **看不到**子 tool 消息与 MCP spill hint，但 **可** `read_file` 子 session spill（同 project 数据目录）。父 Agent 仍应依赖子代理 FinalContent / `output_file` 摘要，而非假设可见子 tool 流。 |
| **compact 丢失 spill hint** | 触发 compact 后，旧轮 tool 消息（含 spill 路径 hint）被摘要替换；磁盘 spill 仍在，但 API 上下文可能不再含路径。模型须在 compact 前读取 spill，或依赖近 N 轮未 compact 消息（见 DESIGN §12.11）。**compact 后**模型无法 `list_dir`/`grep` 发现 project 数据目录。人工恢复：主会话 `session_id` 见 TUI `/sessions` 或 `~/.ds-code/projects/<id>/sessions.db`；spill 在 `mcp-result/<session_id>/` 或 `agents/<session_id>/`。 |
| **hint 路径须可 `read_file`** | hint 中嵌入的路径必须是 `read_file` 可解析的**完整绝对路径**（`resolveProjectDataRead` 用 `filepath.Clean`，**不**展开 `~`）；为控制 hint 长度仅可缩短正文，**禁止**尾部截断到不可读路径（见 DESIGN §12.5、FR-4.14）。 |
| **每次成功 MCP 均落盘** | 未超长结果也写 spill（简化实现、便于调试）；高频 MCP 增加磁盘 IO（NFR-11；粗估：10 次/轮 × 500k 字符 ≈ 5MB/轮，视 MCP 响应而定）。同非空 `tool_call_id` 重试/recovery **覆盖**同名 `<stem>.txt`（FR-4.19）。 |
| **spill 文件名 ≠ LLM id** | `messages.tool_call_id` 存 LLM 原始 id；磁盘文件名为 `spillCallFilename(id)`（如 `call/foo` → `call_foo.txt`）。hint 须指向**实际 spill 绝对路径**。 |
| **`.git` 不可 Agent 枚举** | 即使 `grep path=.git` / `list_dir path=.git` 也不进入 `.git`（FR-6.14）；用户 `@.git/` 仍可展开（FR-6.9）。 |
| **`glob **/*` + `skip_dirs`** | `glob`/`grep` 的 `**` 路径经 `globmatch` Walk 时，`skip_dirs` 在 Walk 阶段 `SkipDir`（Phase H 注入）；结果层仍经 `ignored()` 过滤。与 `grep` 目录 Walk 语义对齐（见 DESIGN §14.4.1）。 |
| **`@` + compact 外发摘要** | 旧轮 user message 中 `@.env` 等展开块**可能**进入 compact 摘要 LLM 输入；S12 行级 redact **不**对 `@` 块专用剥离（SECURITY §S3-S）。用户点名敏感路径须知情。 |
| **TUI 复制含可见敏感内容** | 复制内容为 viewport **已渲染** plain text；含 MCP 参数摘要、spill hint 绝对路径、shell 命令行等（与 FR-5.9 一致）；不复制未显示字段（NFR-18）。 |
| **`@` 引用语法** | 仅匹配 `@([a-zA-Z0-9_./\-]+)`；含空格、Unicode 等路径须用 Agent 工具或引号外显式说明。 |
| **`@.git/` / 超大目录** | 用户可 `@.git/` 或 `@node_modules/`；`@dir/` 仅列路径（不含正文），受 `at_dir_max_*` 条目上限约束。 |
| **`@dir/` 语义** | `@dir/` 注入目录文件列表，**不**预读文件正文；需全文时用 `@file` 或 Agent `read_file`。用户原文中的 `@path` 保留在 prompt 中。 |
| **MCP 失败 / 取消** | 失败（`isToolErrorBody`）与 context 取消 mid-flight **不写** spill（FR-4.16）。`isToolErrorBody` 对正文以 `error:` 开头的**成功** MCP 响应可能误判（见 DESIGN §12.5、§10）。 |
| **审计日志** | `audit.jsonl` 仍仅存 args 哈希；MCP `args_preview` 不进 audit（行为不变）。 |
| **TUI 复制范围** | 仅聊天 viewport 与工具面板；输入框、浮层内复制策略见 FR-7.8；不实现 transcript 刷回 scrollback。流式输出进行中选区可能错位（FR-7.9）。 |
| **远程剪贴板** | SSH/tmux 下依赖 OSC 52 或 tmux paste buffer；部分终端默认禁用 OSC 52，复制可能失败并提示（FR-7.6–7.7）。 |
| **TUI 滚轮与选区** | 文本选区活跃时 HP 滚动临时关闭，回退全量渲染（FR-9.6）；快速滚轮 + 拖拽选区并存时以选区高亮为准。 |
| **集成终端手感** | VS Code / Cursor 内置终端与 iTerm2/Ghostty 使用不同 drain 曲线（FR-9.3）；极端 burst 可能 snap 截断 pending（>30 行）。 |

## 依赖与前置

- 基线：v0.1.1 已发布或合入 main
- 需求 2 固定目录 `mcp-result/`（见 [DESIGN.md §12](DESIGN.md#12-mcp-结果落盘与上下文截断需求-2)）；**不**新增 `mcp_tool_result_max_chars`；新增依赖 `github.com/oklog/ulid/v2`（NFR-19）
- **发布阻塞**：实现合入前须同步 [SECURITY.md](../v0.1.0/SECURITY.md) 与 [CONFIG.md](../v0.1.0/CONFIG.md)；草稿见 [SECURITY-SYNC.md](SECURITY-SYNC.md)
- worktree 子代理须在 `spawn/execute.go` **所有**新建 `Engine` 分支注入 `perm.ProjectRoot = cfg.ProjectRoot`；子 `Runner` 须继承父 `MCPResults` Store（见 DESIGN §5.3、§12.5）

## 关联文档

- 安全/配置同步草稿：[SECURITY-SYNC.md](SECURITY-SYNC.md)（实现时写入 v0.1.0 正式文档）
- 安全基线：[../v0.1.0/SECURITY.md](../v0.1.0/SECURITY.md) §S2、§S3、§S3-S（`@` 例外，实现时新增）、§S11
- 全局设计：[../v0.1.0/DESIGN.md](../v0.1.0/DESIGN.md)
- 上一版本：[../v0.1.1/README.md](../v0.1.1/README.md)
