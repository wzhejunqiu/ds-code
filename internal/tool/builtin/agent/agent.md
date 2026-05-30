# agent

## 概述

LLM 可见的**子代理**入口，替代旧版 `task` 工具。接收自然语言任务与 `subagent_type`，委托 [`internal/agent/spawn`](../../../agent/spawn) 完成路由、工具池过滤、执行与结果回传。支持同步、后台（async）与 Fork（共享父会话 cache 前缀）三种形态。

## 注册与可见性

| 模式 | 注册条件 |
|------|----------|
| agent | `RegisterAgentExtras` 且 `setup.Deps.LLM != nil` |
| plan | **不注册** |

子代理工具池**不包含** `agent` 自身（禁止嵌套 spawn）。

## 参数 Schema

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `description` | string | 是 | 短标题（TUI / 日志） |
| `prompt` | string | 是 | 交给子代理的任务说明 |
| `subagent_type` | string | 否 | 内置类型 enum（由 `spawn.Registry` 提供，如 explore、general-purpose） |
| `model` | string | 否 | `deepseek-v4-pro` / `deepseek-v4-flash` |
| `run_in_background` | boolean | 否 | 后台执行，父工具立即返回 async 句柄 |
| `isolation` | string | 否 | 目前支持 `worktree`（独立 git worktree） |

省略 `subagent_type` 且在交互模式下可能走 **Fork** 路径（见 spawn 文档）。

## 用法示例

```json
{
  "description": "Explore auth package",
  "prompt": "Find where JWT validation happens and list key files.",
  "subagent_type": "Explore"
}
```

```json
{
  "description": "Run tests in background",
  "prompt": "Run go test ./... and summarize failures.",
  "subagent_type": "general-purpose",
  "run_in_background": true
}
```

## 返回格式

取决于 spawn 路由结果：同步摘要、spill 指针、`async_launched` JSON、或错误信息。详见 [`spawn/README.md`](../../../agent/spawn/README.md) 中 Output / Notify 章节。

## 实现细节

源文件：[`agent.go`](agent.go)

1. `NewAgentTool` 构造 `spawn.Service`（传入父 `Registry` 用于工具池过滤）。
2. `Execute` 从 context 读取父 `ToolInvocation`（session_id、tool_call_id）。
3. 信号量 `cfg.Tools.Agent.MaxParallel`（默认 3）限制并行子代理数。
4. 业务逻辑全部在 `Spawn.Handle`；本包仅做参数校验与并发控制。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `tools.agent.max_parallel` | 3 | 同时运行的子代理上限 |
| `tools.agent.max_turns` | （见 defaults） | 子代理 Runner 最大轮次 |
| `tools.agent.summary_max_chars` | — | 同步结果摘要长度 |
| spawn 相关 | — | fork、worktree、verification 等见 spawn 文档 |

## 权限与安全

- **PermissionLevel**：`Low`（工具本身不直接写盘；子代理内写操作仍受权限引擎约束）
- 子代理权限模式由类型定义 + `permission.mode` 决定（readonly 类型仅探索工具）
- Fork 权限询问冒泡到父 TUI

## 设计思想

- **薄工具层**：spawn 包单入口，避免在 `agent.go` 重复路由逻辑。
- **复用主 Runner**：子代理与主会话共用 recovery、tool orchestration、Hook。
- **显式类型**：`subagent_type` enum 约束工具池与 system overlay，降低失控写操作风险。

## 相关代码

- [`agent.go`](agent.go)、[`agent_test.go`](agent_test.go)
- [`spawn/`](../../../agent/spawn/) — 完整设计与 API
- [`setup/setup.go`](../../setup/setup.go) — `RegisterAgentExtras`
- [`subagentstore/`](../../../session/subagentstore/) — 子代理会话持久化
