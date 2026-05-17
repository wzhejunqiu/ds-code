# ds-code 安全说明

本文档对应 [PLAN.md · 安全审计清单](PLAN.md#安全审计清单) 与 [DESIGN.md §15](DESIGN.md#15-安全设计)。

## 威胁模型（简要）

| 威胁 | 缓解 |
|------|------|
| Prompt 注入经 tool 结果覆盖 system | Tool 输出带边界；`mergeSystem` 固定顺序；用户消息不能替换 system |
| 路径遍历读写工作区外文件 | `permission.ResolvePath`：`..` 拦截、symlink 解析、jail 到 `project_root` |
| 敏感文件泄露（`.env`、密钥） | 路径段级 denylist（`IsSensitiveAbs`）；读工具、LSP、`shell` 统一过滤；`auto` 可读普通文件但仍受 denylist；compact 摘要脱敏（S12） |
| API Key 进入仓库或配置 | 仅 `DS_CODE_DEEPSEEK_API_KEY` / `DEEPSEEK_API_KEY`；YAML 禁止 `api_key` |
| SSRF（Web 工具） | `web.fetch_enabled` 默认关；`web.allowlist` 必填 |
| MCP 写操作绕过权限 | MCP 工具走同一 `permission.Engine`；写工具检测器 |
| 子代理提权写盘 | `task` 子 Runner 仅只读工具集（S14） |
| 会话数据跨项目泄露 | `project_id = sha256(project_root)` 分库；DB/checkpoint/audit 按项目目录隔离 |

## 审计清单 S1–S14

| ID | 落点 | 说明 |
|----|------|------|
| S1 | `internal/config` | API Key 仅环境变量；日志不打印 key |
| S2 | `permission.ResolvePath` | 路径逃逸拦截 |
| S3 | `permission.IsSensitiveAbs` / `CheckReadablePath` | 按路径段匹配：`.env`、`.envrc`、`.env.*`、`.aws`、`.ssh`、`.docker`、`.kube`、`.gnupg`、`credentials/`、`secrets/`、密钥文件名等（启发式，非完备） |
| S4 | `permission.checkSensitiveShell` | denylist 路径扫描 + 高危 shell 模式；`auto` 下 shell 可读普通文件但不可访问敏感路径；`context` 取消杀子进程 |
| S5 | `context` tool 格式化 + system merge | 用户不能覆盖 system |
| S6 | `internal/mcp` + `Perm.Check` | MCP 写操作统一权限 |
| S7 | `session.OpenDefaultStore` | 按项目分库；`sessions.db` 权限 0600 |
| S8 | CI `govulncheck` | 依赖漏洞扫描 |
| S9 | `context.Context` | LLM、工具、LSP、子进程贯穿取消 |
| S10 | `internal/audit` | `audit.enabled` → 固定 `audit.jsonl`；仅存 args 哈希 |
| S11 | `TruncateToolResult`、`@` 预算 | 超大 tool/@ 结果截断 |
| S12 | `sanitizeCompactInput` | compact 摘要 redact 密钥行 |
| S13 | `RunEphemeral` | `/btw` 无 tools、不写 `messages` |
| S14 | `subagent` + `RegisterExploreTools` | 子代理无 write/shell |

## Checkpoint

- 写操作（`apply_patch`、`write_file`）前捕获受影响文件快照至 `~/.ds-code/projects/<id>/checkpoints/<session>/`.
- `/checkpoint rewind N` 恢复工作区；历史层追加 `role=system` 事件，**不**进入 API `mergeSystem`。
- 单文件快照上限 4 MiB；`shell` 不自动 checkpoint（工作区变更不可可靠还原）。

## Shell 后台任务

- 后台命令在 `project_root` 下执行，输出写入 `~/.ds-code/projects/<id>/shell-jobs/<job_id>/`。
- 并发上限 `tools.shell.max_background`（默认 5）；轮询/列表在 `readonly` 下允许，启动与 `cancel` 仍受写权限约束。
- 退出 ds-code 时**不**自动杀死后台任务（长任务可继续运行）；`cancel` 或进程自然结束。

## 报告问题

如发现安全问题，请通过项目维护者私下渠道报告，勿在公开 issue 中粘贴密钥或完整审计日志。
