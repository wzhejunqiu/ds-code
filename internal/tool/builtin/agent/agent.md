# agent

## 概述

LLM 可见的**子代理**入口，替代旧版 `task` 工具。接收自然语言任务与 `subagent_type`，委托 [`internal/agent/spawn`](../../../agent/spawn) 完成路由、工具池过滤、执行与结果回传。LLM 可见类型为 `general-purpose` 与 `Explore`；同步阻塞至完成或 `sync_timeout`，亦可显式 `run_in_background`。

## 注册与可见性

| 模式 | 注册条件 |
|------|----------|
| agent | `RegisterAgentExtras` 且 `setup.Deps.LLM != nil` |
| plan | **不注册** |

子代理工具池**不包含** `agent` 自身（禁止嵌套 spawn）。

## 参数 Schema（LLM 可见）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `description` | string | 是 | 短标题（TUI / 日志） |
| `prompt` | string | 是 | 交给子代理的任务说明 |
| `subagent_type` | string | 否 | `general-purpose` 或 `Explore`（[`ListToolTypes`](../../../agent/spawn/registry.go)）；省略时默认 general-purpose |
| `run_in_background` | boolean | 否 | 后台执行，父工具立即返回 async 句柄 |

Prompt 正文见 [`usage.prompt`](usage.prompt)；并行上限以渲染后的具体数字告知 LLM，不暴露 config 键名。

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
2. `Execute` 从 context 读取父 `ToolInvocation`（session_id、tool_call_id、parent_model）。
3. 信号量 `cfg.Tools.Agent.MaxParallel`（默认 3）限制并行子代理数。
4. 业务逻辑全部在 `Spawn.Handle`；本包仅做参数校验与并发控制。

## 配置项

| 键 | 默认 | 说明 |
|----|------|------|
| `tools.agent.max_parallel` | 3 | 同时运行的子代理上限（注入 prompt 为具体数字） |
| `tools.agent.sync_timeout` | 2h | 同步子代理最长等待 |
| `tools.agent.summary_max_chars` | 16000 | 同步结果摘要长度 |
| spawn 相关 | — | 见 spawn 文档 |

## 后续 TODO（非 LLM 面）

| 能力 | 状态 |
|------|------|
| Plan 子代理 | spawn 已注册；**待**用户主动发起入口 |
| verification 子代理 | 同上 |
| Fork（agent 工具路径） | skill fork（`/skill` + `context:fork`）已有；**待**完整用户发起 UX |
| worktree isolation | spawn 已实现；**待**完善后暴露给 LLM |

## 权限与安全

- **PermissionLevel**：`Low`（工具本身不直接写盘；子代理内写操作仍受权限引擎约束）
- 子代理权限模式由类型定义 + `permission.mode` 决定（readonly 类型仅探索工具）

## 设计思想

- **薄工具层**：spawn 包单入口，避免在 `agent.go` 重复路由逻辑。
- **复用主 Runner**：子代理与主会话共用 recovery、tool orchestration、Hook。
- **显式类型**：`subagent_type` enum 约束工具池与 system overlay，降低失控写操作风险。

## 相关代码

- [`agent.go`](agent.go)、[`agent_test.go`](agent_test.go)、[`usage.prompt`](usage.prompt)、[`text.go`](text.go)
- [`spawn/`](../../../agent/spawn/) — 完整设计与 API
- [`setup/setup.go`](../../setup/setup.go) — `RegisterAgentExtras`
- [`subagentstore/`](../../../session/subagentstore/) — 子代理会话持久化
