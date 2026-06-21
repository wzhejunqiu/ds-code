# bash

## 概述

在项目工作区目录下执行 shell 命令（LLM wire 名 **`bash`**，Go 包名仍为 `shell`），支持同步执行与后台并行执行（`run_in_background`）。

## 注册与可见性

| 模式 | 注册 |
|------|------|
| agent | `RegisterWrite` |
| plan | **不注册** |

可通过 `tools.defer_builtin` 延迟暴露完整 schema；需先调用 `tool_search`（示例：`defer_builtin: ["bash"]`）。

依赖 `setup.Deps.ShellJobs`（`shelljobs/manager.Manager`）；为 `nil` 时仅支持同步模式。

## 提示词

| 文件 | 职责 |
|------|------|
| [`usage.prompt`](usage.prompt) | LLM 可见 `Description` 正文（`//go:embed` + `text/template`） |
| [`shell_cmd_description.prompt`](shell_cmd_description.prompt) | `description` 参数 JSON Schema 文案（embed 至 `SchemaShellDescription`） |
| [`text.go`](text.go) | `RenderDesc()`、其余 Schema/Err/Result 常量 |

> bash 参考实现使用 `usage.prompt` 而非 FR-0 通用的 `prompt.md` 文件名；其余工具改写仍目标为 `prompt.md`。

详见 v0.1.4 [FR-0](../../docs/v0.1.4/REQUIREMENTS.md#fr-0-工具-prompt-标准模式必遵)。

## 参数 Schema

| 字段 | 类型 | 说明 |
|------|------|------|
| `description` | string | 简短自然语言说明（供 TUI 展示，非执行内容） |
| `command` | string | 要执行的 shell 命令 |
| `run_in_background` | boolean | 默认 **false**；为 true 时在 OS 后台启动，**工具调用阻塞至完成**后返回 stdout/stderr；同轮多条可并行 |
| `timeout_ms` | integer | 可选；省略→`tools.shell.timeout`（120s）；指定→`min(ms, 600_000)`；**同步与 background 均适用**；超时强制 kill |

`command` 为必填（同步与 background 启动时）。

**Breaking（v0.1.4+）**：`background` → `run_in_background`；移除 `list_jobs`、`job_id`、`cancel`（LLM 不可见）。

## 用法示例

**同步（配置默认超时）：**

```json
{"description": "Run unit tests", "command": "go test ./internal/tool/builtin/..."}
```

**同步（per-call 超时 30 秒）：**

```json
{"description": "Quick check", "command": "make vet", "timeout_ms": 30000}
```

**后台（阻塞至完成，同轮可并行）：**

```json
{"description": "Start dev server", "command": "npm run dev", "run_in_background": true}
```

## 返回格式

**同步与 background 成功/失败（格式相同）：**

```text
stdout:
...
stderr:
...
exit: ...   # 仅命令非零退出或超时时
```

无输出：`（无输出）`

超时或 Esc 取消时，若 job/子进程已产生输出，仍会返回已写入的 stdout/stderr，并附带 `exit:` 行（如 `context deadline exceeded`）。

## Manager 生命周期

| 阶段 | 行为 |
|------|------|
| `Open` | 创建 manager（**异步** `reconcileStaleJobs` 清磁盘 stale meta，不阻塞启动）；不跨会话恢复跟踪 |
| `Start` + `Wait` | 本会话 `run_in_background`：启动子进程并阻塞至完成 |
| `Close` | 退出 ds-code 时 kill 本会话所有 running job（TUI / `-p` / `App.Close` 经 `closeShellJobs` 调用） |

Esc 取消 turn 时，turn context 取消 → `Wait` 终止对应 job。

## 实现细节

源文件：[`shell.go`](shell.go)、[`timeout.go`](timeout.go)、[`args.go`](args.go)

### 同步 `runSync`

1. `ResolveShellTimeout(cfg, timeout_ms)`：`timeout_ms` 优先，cap 10 分钟，否则 `tools.shell.timeout`。
2. `context.WithTimeout` + `exec.CommandContext($SHELL, "-c", command)`，`Dir = Perm.Workspace`。
3. 超时到期 context 取消 → 子进程被 OS/Go **强制终止**。
4. 环境变量：`security.SafeSubprocessEnv`，合并配置 `env_blacklist` 与内置密钥过滤。

### 后台 `runBackground`

1. 与 sync **相同**的 `ResolveShellTimeout` + `context.WithTimeout`。
2. 委托 `shelljobs/manager`：`Start(command, description)` → `Wait(ctx, jobID)` → `Get` 读 stdout/stderr；`Wait` 失败（超时/取消）时仍 `Get` 已写输出。
3. Runner 将 `run_in_background` bash 划入 **concurrent batch**，同轮多条并行执行。

### TUI

- bash Running（sync 与 `run_in_background`）：标题行末尾显示**递减倒计时**（如 `1:23`），由 `ShellTimeoutDeadline` + `ThinkingTick`（100ms）驱动。
- 格式化：[`internal/tool/format_countdown.go`](../../format_countdown.go) 的 `FormatTimeoutCountdown`。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `tools.shell.timeout` | 120s | 命令默认超时（可被 per-call `timeout_ms` 覆盖；sync/bg 共用） |
| `tools.shell.max_background` | 5 | 后台任务数量上限（manager 侧） |
| `tools.shell.background_output_max_bytes` | 262144 | 单次 job 输出读取上限 |
| `tools.shell.env_blacklist` | 如 `^AWS_` | 子进程环境变量名正则黑名单 |

## 权限与安全

- **PermissionLevel**：`Highest`
- 可执行任意 shell 命令，是攻击面最大的内建工具
- Plan 模式完全禁用
- 工作目录锁定在 `project_root`；环境变量脱敏

## 设计思想

- **单一入口**：同步与后台共用 `bash` wire 名，通过参数区分。
- **`description` 与 `command` 分离**：TUI 展示人类可读摘要，完整命令仍可审计。
- **并行 + 阻塞**：`run_in_background` 允许 Runner 同轮并行，但每条 tool call 仍阻塞至完成再返回结果。
- **会话绑定**：job 仅在本 ds-code 进程内跟踪；退出强制清理，不跨会话恢复。

## 相关代码

- [`shell.go`](shell.go)、[`timeout.go`](timeout.go)、[`args.go`](args.go)
- [`usage.prompt`](usage.prompt)、[`shell_cmd_description.prompt`](shell_cmd_description.prompt)、[`text.go`](text.go)、[`text_test.go`](text_test.go)
- [`shell_background_test.go`](shell_background_test.go)、[`shell_test.go`](shell_test.go)
- [`shell_display.go`](../../shell_display.go)
- [`format_countdown.go`](../../format_countdown.go)
- [`shelljobs/manager/`](../../../shelljobs/manager/)
- [`internal/agent/tool_orchestration.go`](../../../agent/tool_orchestration.go)
