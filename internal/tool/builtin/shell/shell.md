# shell

## 概述

在项目工作区目录下执行 shell 命令，支持同步执行与后台任务（启动、轮询、取消、列表）。

## 注册与可见性

| 模式 | 注册 |
|------|------|
| agent | `RegisterWrite` |
| plan | **不注册** |

可通过 `tools.defer_builtin` 延迟暴露完整 schema；需先调用 `tool_search`。

依赖 `setup.Deps.ShellJobs`（`shelljobs/manager.Manager`）；为 `nil` 时仅支持同步模式。

## 参数 Schema

| 字段 | 类型 | 说明 |
|------|------|------|
| `description` | string | 简短自然语言说明（供 TUI 展示，非执行内容） |
| `command` | string | 要执行的 shell 命令（同步或后台启动时必填） |
| `background` | boolean | `true` 时后台启动并返回 `job_id` |
| `job_id` | string | 轮询后台任务输出/状态；配合 `cancel` 可终止 |
| `cancel` | boolean | 与 `job_id` 合用，杀死后台任务 |
| `list_jobs` | boolean | 列出当前项目的后台任务 |

无单一 `required` 字段：根据操作组合校验（同步需 `command`；轮询需 `job_id` 等）。

## 用法示例

**同步：**

```json
{"description": "Run unit tests", "command": "go test ./internal/tool/builtin/..."}
```

**后台：**

```json
{"description": "Start dev server", "command": "npm run dev", "background": true}
```

```json
{"job_id": "job-abc123"}
```

```json
{"job_id": "job-abc123", "cancel": true}
```

```json
{"list_jobs": true}
```

## 返回格式

**同步成功/失败：**

```text
stdout:
...
stderr:
...
exit: ...   # 仅命令非零退出时
```

无输出：`(no output)`

**后台启动：**

```text
background job started
job_id: ...
pid: ...
status: ...
command: ...
```

**轮询：** `job_id`、`status`、`command`、`pid`、时间戳、`stdout`/`stderr` 片段（受 `background_output_max_bytes` 限制）。

## 实现细节

源文件：[`shell.go`](shell.go)

### 同步 `runSync`

1. `context.WithTimeout`：默认 `tools.shell.timeout`（120s）。
2. `exec.CommandContext($SHELL, "-c", command)`，`Dir = Perm.Workspace`。
3. 环境变量：`security.SafeSubprocessEnv`，合并配置 `env_blacklist` 与内置密钥过滤。

### 后台

委托 `shelljobs/manager`：`Start` / `Get` / `Cancel` / `List`。输出截断由 `BackgroundOutputMaxBytes` 控制。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `tools.shell.timeout` | 120s | 同步命令超时 |
| `tools.shell.max_background` | 5 | 后台任务数量上限（manager 侧） |
| `tools.shell.background_output_max_bytes` | 262144 | 单次轮询返回的最大输出 |
| `tools.shell.env_blacklist` | 如 `^AWS_` | 子进程环境变量名正则黑名单 |

## 权限与安全

- **PermissionLevel**：`Highest`
- 可执行任意 shell 命令，是攻击面最大的内建工具
- Plan 模式完全禁用
- 工作目录锁定在 `project_root`；环境变量脱敏

## 设计思想

- **单一入口**：同步与后台共用 `shell` 名称，通过参数区分，减少 LLM 工具表膨胀。
- **`description` 与 `command` 分离**：TUI 展示人类可读摘要，完整命令仍可审计。
- **后台一等公民**：长任务（编译、服务）不阻塞 agent 轮次，通过 `job_id` 轮询。

## 相关代码

- [`shell.go`](shell.go)
- [`shell_background_test.go`](shell_background_test.go)
- [`shell_display.go`](../../shell_display.go)
- [`shelljobs/manager/`](../../../shelljobs/manager/)
