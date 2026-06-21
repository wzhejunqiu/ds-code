# ds-code 安全说明

本文档对应 [PLAN.md · 安全审计清单](PLAN.md#安全审计清单) 与 [DESIGN.md §15](DESIGN.md#15-安全设计)。

## 威胁模型（简要）

| 威胁 | 缓解 |
|------|------|
| Prompt 注入经 tool 结果覆盖 system | Tool 输出带边界；`mergeSystem` 固定顺序；用户消息不能替换 system |
| 路径遍历读写工作区外文件 | `permission.ResolvePath` / `ResolveAccessPath`：`filepath.Clean` + join、`EvalSymlinks`、`ensureUnder` jail 到 `project_root`；**不再**对相对路径做 `..` 子串拦截 |
| 敏感文件泄露（`.env`、密钥） | 路径段级 denylist（`SkipSensitiveAbs`）；Agent 枚举、`read_file`、`shell`、LSP 统一过滤；**例外**：用户提示词显式 `@file` / `@dir/` 仅 S2（§S3-S）；MCP spill 全文 0600 存 project 数据目录，同 project 数据目录 regular file 经 `read_file` 可读；compact 输入对 `@` 展开块不做专用脱敏 |
| 用户 `@` 显式绕过 S3 / compact 外发敏感内容 | 仅主会话 user message 中 `@file` / `@dir/` 可越过 S3（§S3-S）；Agent 工具、`read_file`、shell 仍 S3；`@` 展开受 `context.at_reference_max_chars` / `at_dir_max_*` 预算约束；compact 时 `sanitizeCompactInput`（S12）行级 redact **不**对 `@` 块专用剥离；用户显式点名视为知情承担风险；TUI 复制可见 MCP 参数 / spill hint |
| TUI 应用内复制泄露敏感可见内容 | 剪贴板写入 viewport **已渲染** plain text（剥离 ANSI）；**可能含** MCP 参数摘要（≤400 字符）、spill hint 绝对路径、shell 命令行；与 `--allow-log-sensitive-data` 独立；失败降级提示、不 panic |
| API Key 进入仓库或配置 | 仅 `DS_CODE_DEEPSEEK_API_KEY` / `DEEPSEEK_API_KEY`；YAML 禁止 `api_key` |
| SSRF（Web 工具） | `web.fetch_enabled` 默认关；`web.allowlist` 必填；通配符 `*.example.com` 仅匹配 `example.com` 及其子域 |
| MCP 写操作绕过权限 | MCP 工具走同一 `permission.Engine`；写工具检测器 |
| 子代理提权写盘 | `task` 子 Runner 仅只读工具集（S14） |
| 会话数据跨项目泄露 | `project_id = sha256(project_root)` 分库；DB/checkpoint/audit 按项目目录隔离 |

## 审计清单 S1–S14

| ID | 落点 | 说明 |
|----|------|------|
| S1 | `internal/config` | API Key 仅环境变量；日志不打印 key |
| S2 | `workspace.ResolveRel` + `permission.ResolvePath` | 相对/绝对路径先规范化与 symlink 求值，再 `ensureUnder`；合法 `foo/../bar` 允许；`../outside` 拒绝 |
| S3 | `permission.SkipSensitiveAbs` / `CheckReadablePath` | 按路径段匹配：`.env`、`.envrc`、`.env.*`、`.aws`、`.ssh`、`.docker`、`.kube`、`.gnupg`、`credentials/`、`secrets/`、密钥文件名等（启发式，非完备） |
| S3-S | `context/atref.go` | 用户提示词中显式 `@file` / `@dir/` 仅校验 S2（`ResolvePath`）。**`@file`** 可读取 `.env` 等 S3 路径并注入全文；**`@dir/`** 仅列路径（不含 S3 文件正文）。用户原文中的 `@path` 保留在 prompt 中。Agent 枚举、`read_file`、`shell` 仍受 S3；compact 时 `sanitizeCompactInput`（S12）**不**对 `@` 展开块做专用剥离 |
| S4 | `permission.checkSensitiveShell` | denylist 路径扫描 + 高危 shell 模式；`auto` 下 shell 可读普通文件但不可访问敏感路径；`context` 取消杀子进程 |
| S5 | `context` tool 格式化 + system merge | 用户不能覆盖 system |
| S6 | `internal/mcp` + `Perm.Check` | MCP 写操作统一权限 |
| S7 | `session.OpenDefaultStore` | 按项目分库；`sessions.db` 权限 0600 |
| S8 | CI `govulncheck` | 依赖漏洞扫描 |
| S9 | `context.Context` | LLM、工具、LSP、子进程贯穿取消 |
| S10 | `internal/audit` | `audit.enabled` → 固定 `audit.jsonl`；仅存 args 哈希 |
| S11 | `TruncateToolResult`、`finalizeToolResult`、`@` 预算 | 超大 tool/@ 结果截断；MCP 成功调用全文写入 `~/.ds-code/projects/<id>/mcp-result/<session_id>/<stem>.txt`（0600）；session 回注仍受 `tool_result_max_chars`；超长时 hint 含完整可解析绝对路径引导 `read_file`；`read_file` 经 `resolveProjectDataRead` 可读本 project 数据目录 regular file（含任意 session spill、`agents/*.output`、`sessions.db`；`readonly`/`ask`/`auto` 均直接放行） |
| S8-R | `read_file/read_file.go`、`textfile.IsTextFile` | 非文本文件（扩展名 blocklist + 前 3072 字节 sniff）拒绝读取；MCP spill `.txt` 与空文件仍允许；`@file`/`@dir/` 不经此判定 |
| S12 | `sanitizeCompactInput` | compact 摘要 redact 密钥行 |
| S13 | `RunEphemeral` | `/btw` 无 tools、不写 `messages` |
| S14 | `subagent` + `RegisterExploreTools` | 子代理无 write/shell |

## Checkpoint

- 写操作（`apply_patch`、`write_file`）前捕获受影响文件快照至 `~/.ds-code/projects/<id>/checkpoints/<session>/`.
- `/checkpoint rewind N --yes` 或 `/rewind N --yes` 恢复工作区（无 `--yes` 时仅提示，不执行）；历史层追加 `role=system` 事件，**不**进入 API `mergeSystem`。
- 单文件快照上限 4 MiB；`shell` 不自动 checkpoint（工作区变更不可可靠还原）。

## Shell 执行模型

- 同步与后台命令均通过 `$SHELL -c <command>` 执行；在 `permission.mode=auto`（`--dangerously-auto`）下无写操作确认，但 **S3 敏感路径 denylist 与高危命令模式仍会拦截**（含 `bash -c`、`nc` 反向 shell、`/dev/tcp`、launchctl/crontab 等扩展模式；仍为启发式，非沙箱）。
- 非 TTY 脚本请勿使用默认 `ask`；应显式 `--permission-mode readonly` 或了解风险后使用 `--dangerously-auto`。
- spill 文件路径位于 project 数据目录（工作区外），须用 `read_file` 读取；`shell cat` 该路径仍受 S2 区外拒绝。同 project 数据目录 regular file（含 `mcp-result/` 任意 session spill、`agents/*.output`）为只读区外例外，`readonly`/`ask`/`auto` 均直接放行。
- TUI 应用内复制（v0.1.2）写入剪贴板的内容为 viewport **已渲染** plain text，可能含 MCP 参数摘要、spill hint 绝对路径；与 debug 日志 `--allow-log-sensitive-data` 策略独立。
- `read_file` 拒绝非文本文件（图片、二进制等）；`@file`/`@dir/` 不经 `IsTextFile`。

## Shell 后台任务

- 后台命令在 `project_root` 下执行，输出写入 `~/.ds-code/projects/<id>/shell-jobs/<job_id>/`。
- 并发上限 `tools.shell.max_background`（默认 5）。
- `run_in_background` 工具调用会阻塞至 job 完成；同轮多条可并行（Runner concurrent batch）。
- **退出 ds-code 时强制 kill 本会话仍在运行的 shell job**（TUI / 非交互 `-p` 均经 `closeShellJobs` → `Manager.Close`）。
- **不跨会话恢复 job**：`Open` 时**异步** reconcile 磁盘 stale meta（不阻塞启动；`Close` 前等待 reconcile 结束），不重新纳入 manager。
- LLM 不可见 `job_id` / `cancel`；用户按 Esc 取消 turn 会终止当前 job。

## 报告问题

如发现安全问题，请通过项目维护者私下渠道报告，勿在公开 issue 中粘贴密钥或完整审计日志。
