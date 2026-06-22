# agent

核心 **Agent 循环**：一条用户消息可经历多轮 LLM **子轮次**（sub-round），每子轮次为一次 `PrepareRequest → LLM.Chat → 工具执行` 循环，直到模型不再返回 `tool_calls` 或达到 `MaxTurns`。

本包负责编排 LLM 调用、工具执行、错误恢复、流式回调、生命周期 Hook，以及通过 `spawn` 子包调度子代理（`agent` 工具）。

## 包结构

| 文件 | 说明 |
|------|------|
| [runner.go](./runner.go) | `Runner` 定义、`executeTool` |
| [runner_turn.go](./runner_turn.go) | `RunTurn` / `RunTurnSeeded` 主入口 |
| [runner_loop.go](./runner_loop.go) | 子轮次流式、工具 UI 回调、终局处理 |
| [loop_state.go](./loop_state.go) | `LoopPhase` / `Transition` / `LoopState` |
| [recovery.go](./recovery.go) | `chatWithRecovery` 多策略 LLM 错误恢复 |
| [tool_orchestration.go](./tool_orchestration.go) | 工具分批（并发读 / 串行写） |
| [callbacks.go](./callbacks.go) | `TurnCallbacks` 流式与 UI 事件 |
| [context.go](./context.go) | ctx 传播：`TurnCallbacks`、`ToolInvocation`、`ActiveTurn` |
| [fork_context.go](./fork_context.go) | Fork 子代理的父会话上下文 |
| [hooks.go](./hooks.go) | `HookManager` 生命周期脚本 |
| [hook_input.go](./hook_input.go) | `HOOK_INPUT` JSON 结构 |
| [hook_apply.go](./hook_apply.go) | PreToolUse 参数改写 |
| [session_hooks.go](./session_hooks.go) | SessionStart / SessionEnd |
| [checkpoint.go](./checkpoint.go) | 写操作前快照、`RewindCheckpoint` |
| [ephemeral.go](./ephemeral.go) | `RunEphemeral`（/btw 旁路） |
| [stream_timing.go](./stream_timing.go) | reasoning 墙钟时间统计 |
| [usage_billing.go](./usage_billing.go) | assistant 消息用量与计费快照 |
| [text.go](./text.go) | 常量（btw、max_turns、摘要截断） |

[`spawn/`](spawn/) 子包见 [spawn/README.md](spawn/README.md) 及 [spawn 源文件索引](spawn/README.md#包结构)。

## 架构总览

```
                    ┌─────────────────────────────────────┐
                    │           app.App 组装               │
                    │  LLM / Tools / Perm / Sessions /    │
                    │  Context / Checkpoints / Hooks /    │
                    │  spawn.Service                      │
                    └─────────────────┬───────────────────┘
                                      │
                    ┌─────────────────▼───────────────────┐
                    │              Runner                  │
                    │  RunTurn / RunEphemeral / Rewind     │
                    └─────────────────┬───────────────────┘
           ┌──────────────────────────┼──────────────────────────┐
           │                          │                          │
    ┌──────▼──────┐           ┌───────▼───────┐          ┌──────▼──────┐
    │ context.    │           │  llm.Client   │          │ tool.       │
    │ Service     │           │  (stream)     │          │ Registry    │
    │ PrepareReq  │           └───────────────┘          │ Execute     │
    └─────────────┘                                      └─────────────┘
           │                          │
    ┌──────▼──────┐           ┌───────▼───────┐
    │ session.    │           │ permission.   │
    │ Store       │           │ Engine        │
    │ (append-only)│          └───────────────┘
    └─────────────┘
```

**Runner** 是编排中心，不直接持有业务逻辑，而是串联：

| 依赖 | 职责 |
|------|------|
| [`context.Service`](../context/service.go) | 构建 API 上下文（system + messages）、@ 展开、compact |
| [`llm.Client`](../llm/types.go) | 流式 Chat，返回 content / reasoning / tool_calls |
| [`tool.Registry`](../tool/registry.go) | 工具定义与 Execute |
| [`permission.Engine`](../permission/engine.go) | 工具权限检查（readonly/ask/auto + S3 denylist） |
| [`session.Store`](../session/store.go) | 消息只增持久化、用量累计 |
| [`checkpoint.Store`](../checkpoint/store.go) | 写操作前文件快照 |
| [`HookManager`](./hooks.go) | `.ds-code/hooks.json` 生命周期脚本 |
| [`spawn.Service`](./spawn/service.go) | `agent` 工具 → 子代理调度 |

### 组装（[`cmd/ds-code/app/runner.go`](../../cmd/ds-code/app/runner.go)）

[`App.newRunner`](../../cmd/ds-code/app/runner.go) 负责依赖注入，主 Runner 与子代理共用同一套 LLM / 权限 / Hook，但 session 存储分离：

```
openStore → permission.Engine → buildTools（含 agent 工具）
    → context.Service（AgentsMD、Rules、AtExpander）
    → agent.Runner{ MaxTurns, Hooks, Checkpoints, Audit }
    → 若存在 agent 工具：
         spawn.Service.Hooks / ParentContext 指向 runner
         runner.DrainNotifications      ← PrioNow + PrioNext（拼入下条 user 消息前）
         runner.DrainNotificationsLater ← PrioLater（写入 session 为 user 行）
         CleanupExpiredWorktrees（启动时清 TTL 过期 worktree）
```

TUI 通过 [`deps.Runner`](../ui/tui/deps/deps.go) 调用 [`RunTurn`](./runner_turn.go)；非交互 CLI（`-p`）传 `cb=nil`，终局内容写入 [`Runner.Out`](./runner.go)。

## Runner 结构体

定义见 [`runner.go`](./runner.go)。

```go
type Runner struct {
    LLM, Tools, Perm, Sessions, Context, Cfg
    MaxTurns         int              // 默认 25（agent.max_turns）
    Out              io.Writer        // 无 TurnCallbacks 时的 CLI 输出
    Audit            *audit.Logger
    Checkpoints      *checkpoint.Store
    Hooks            *HookManager
    DrainNotifications      NotificationFunc       // RunTurn 开头
    DrainNotificationsLater DrainNotificationsLaterFunc
    ForSubagent      bool             // true → 5xx 时用 llm.subagent.fallback_model
    sessionStarted   map[string]bool  // SessionStart 去重
}
```

## 核心概念

### 用户回合 vs 子轮次

| 术语 | 含义 |
|------|------|
| **用户回合（User Turn）** | 用户发送一条消息到 agent 给出最终回复的完整过程 |
| **子轮次（Sub-round）** | 用户回合内的一次 LLM 请求；若返回 `tool_calls` 则执行工具后继续下一子轮次 |

一条用户消息通常包含多个子轮次，例如：读文件 → 改文件 → 总结回复。

### LoopState

[`LoopState`](./loop_state.go) 在单次用户回合内跨子轮次传递可变状态，主要用于 [`chatWithRecovery`](./recovery.go)：

- **Phase**：`Prepare → LLM → Decide → Tools → Update`（当前实现以 Round 计数为主，Phase 为扩展点）
- **Transition**：记录为何重试或结束（`compact_retry`、`max_turns`、`model_fallback` 等）
- **EphemeralTail**：仅注入 API、不写入 SQLite 的尾部消息（如 MaxTurns 摘要 prompt）

### 消息持久化（每子轮次）

| 阶段 | 写入 session 的内容 |
|------|---------------------|
| 用户提交 | `role=user`（@ 展开后） |
| LLM 返回 tool_calls | `role=assistant` + `ToolCallsJSON` + content/reasoning/usage |
| 每个工具完成 | `role=tool` + `ToolCallID` + 格式化结果体 |
| LLM 终局（无工具） | `role=assistant` + 最终 content/reasoning/`TurnDurationMS` |
| MaxTurns | `role=system` 事件 + 摘要 assistant（摘要 prompt 本身不持久化） |

工具结果经 [`toolresult.FormatToolResult`](../toolresult/format.go) 包装为 `<tool_result name=... id=...>` XML，持久化前由 [`finalizeToolResult`](./mcp_spill.go) / [`TruncateToolResult`](../toolresult/format.go) 处理：内建工具仅截断；MCP 成功调用另将全文写入 `mcp-result/<session_id>/` 并在超长时附加可 `read_file` 的绝对路径 hint。[`runToolCalls`](./runner_loop.go) 读回 session 时用 [`UnpackToolBody`](../toolresult/format.go) 解析 UI 展示。

## Runner 入口

| API | 源文件 | 用途 |
|-----|--------|------|
| `RunTurn` | [runner_turn.go](./runner_turn.go) | 交互式聊天回合：追加用户消息，跑完整子轮次循环 |
| `RunTurnSeeded` | [runner_turn.go](./runner_turn.go) | 不追加用户消息（Fork 子代理预置历史后启动） |
| `RunEphemeral` | [ephemeral.go](./ephemeral.go) | `/btw` 旁路问答——无工具，可选不写历史 |
| `RewindCheckpoint` | [checkpoint.go](./checkpoint.go) | 通过 checkpoint 撤销文件写入 |
| `EndSessionHooks` | [session_hooks.go](./session_hooks.go) | 会话结束时触发 `SessionEnd` hook |

### TurnResult

定义见 [`runner.go`](./runner.go)。

```go
type TurnResult struct {
    FinalContent           string
    FinalReasoning         string
    FinalReasoningDuration time.Duration
    TurnDuration           time.Duration
    Usage                  llm.Usage
    SubRounds              int  // 实际执行的子轮次数
}
```

## RunTurn 主循环

实现：[`runner_turn.go`](./runner_turn.go)（主循环）、[`runner_loop.go`](./runner_loop.go)（流式 / 工具 UI / 终局）。

```
DrainNotifications（PrioNow 异步 agent 通知）
        │
        ▼
ExpandUserText（@ 引用展开）→ AppendMessage(user)
        │
        ▼
Context.BeginUserTurn()
        │
        ▼
┌─── for round = 0 .. MaxTurns-1 ───────────────────────┐
│  PhasePrepare: Context.PrepareRequest                  │
│  PhaseLLM:     chatWithRecovery（流式）                 │
│  若无 tool_calls → finishTerminalRound → 返回          │
│  appendAssistantWithTools                              │
│  PhaseTools:   runToolCalls（分批执行）                 │
│  DrainNotificationsLater（PrioLater 通知）               │
│  OnAssistantSegmentEnd / OnPlanningStart（round>0）    │
└────────────────────────────────────────────────────────┘
        │
        ▼（达到 MaxTurns）
finishMaxTurnsExceeded（soft landing 摘要）
```

### 子轮次间 UI 信号

- **round 0**：Planning UI 由 TUI 在 [`submitLine`](../ui/tui/run.go) 时启动
- **round > 0**：agent 通过 [`OnPlanningStart`](./callbacks.go) / `OnPlanningEnd` 显示「Planning next moves」
- **工具执行前**：[`OnAssistantSegmentEnd`](./callbacks.go) 结束当前可见 assistant 分段

### 流式处理

[`RunTurn`](./runner_turn.go) 始终 `Stream: true`。[`attachStreamHandlers`](./runner_loop.go) 将 `llm.StreamDelta` 映射到 [`TurnCallbacks`](./callbacks.go)：

| Delta | 回调 |
|-------|------|
| Content | `OnContentDelta`（并结束 planning UI） |
| Reasoning | `OnReasoningDelta` |

[`streamTiming`](./stream_timing.go) 记录 reasoning 墙钟时间（首个 reasoning delta → 首个 content delta），写入 `ReasoningDurationMS` 与 `TurnResult.FinalReasoningDuration`。

若流式过程中未输出 content，终局时从 `resp.Content` 补发一次 `OnContentDelta`。

### LLM 请求字段

每个子轮次构造的 `llm.Request` 要点：

| 字段 | 来源 |
|------|------|
| `MergedSystem` / `Messages` | `Context.PrepareRequest` |
| `Model` / `ThinkingType` / `ReasoningEffort` | 当前 `session.Session` |
| `Tools` | `Tools.Definitions()`（MaxTurns 摘要请求无 tools） |
| `MaxTokens` | PrepareRequest 返回值 |
| `UserID` | [`cacheScope`](./runner.go)(sessionID) = sha256(sessionID) hex，用于 prompt cache 分桶 |
| `StrictTools` | `cfg.llm.strict_tools` |

[`Context.BeginUserTurn` / `EndUserTurn`](../context/service.go) 包裹整次用户回合，重置 per-turn token breakdown 缓存（compact 条件 A 用）。

## chatWithRecovery：LLM 错误恢复

[`chatWithRecovery`](./recovery.go) 包装 `LLM.Chat`，按错误类型依次尝试恢复策略（同一子轮次内重试，不增加 round 计数）：

| 错误类型 | 策略 | Transition |
|----------|------|------------|
| Context too long | ① `CompactAPIContext` 重试 ② `ForceAggressiveSnip` 重试 | `compact_retry` / `snip_retry` |
| Max tokens / length | ① MaxTokens 升至 64K ② 追加 continue 消息重试（最多 3 次） | `max_tokens_escalate` / `output_recovery` |
| Empty response | 追加 continue 消息重试 | `output_recovery` |
| Transient network | 直接重试（最多 3 次） | `network_retry` |
| Rate limit | 指数退避重试（最多 3 次） | `rate_limit_retry` |
| Server 5xx | 指数退避 → 切换 fallback model（子代理用 `llm.subagent.fallback_model`） | `model_fallback` |

compact/snip 后通过 [`mergePreparedMessages`](./recovery.go)(view.Messages, state.EphemeralTail) 重建请求，保留 API-only 尾部消息。

## 工具执行

### 单工具流水线（[`executeTool`](./runner.go)）

```
PreToolUse hook（可改写 args）
        │
        ▼
shell → security classifier 审计
        │
        ▼
Perm.Check
        │
        ▼
recordCheckpoint（apply_patch / write_file）
        │
        ▼
Tools.Execute（ctx 携带 ToolInvocation）
        │
        ▼
PostToolUse hook
        │
        ▼
FormatToolResult / FormatToolError
```

`agent` 工具且 `fork_enabled` 时，[`enrichAgentForkContext`](./runner.go) 在 Execute 前注入父会话 API 视图与 [`ForkContext`](./fork_context.go)。

内置 [`agent` 工具](../tool/builtin/agent/agent.go) 通过信号量限制并发 spawn 数（`tools.agent.max_parallel`，默认 3），再委托 [`spawn.Service.Handle`](./spawn/service.go)。

### 分批编排（[`tool_orchestration.go`](./tool_orchestration.go)）

[`partitionToolCalls`](./tool_orchestration.go) 将同一 assistant 消息中的多个 tool_call 拆成 batch：

- **并发 batch**：相邻的只读 + concurrency-safe 工具（如 read_file、grep），最多 10 个 goroutine
- **串行 batch**：写工具或非 concurrency-safe 工具，逐个执行

每个工具：[`executeSingleTool`](./tool_orchestration.go) → 截断 → [`persistToolResult`](./tool_orchestration.go) 写入 session。

[`runToolCalls`](./runner_loop.go) 负责 UI 回调（`OnToolStart` / `OnToolEnd`），包括 apply_patch 多行展示、read_file 行号后缀等。每个 sub-round 开始时 [`readGateForSubRound`](./readfile_gate.go) 注入 [`readgate`](../tool/readgate/gate.go)：`apply_patch` 要求 update/delete 路径在更早 sub-round 已 `read_file`，且禁止同批 read+patch 同一文件。

### 取消

`context.Context` 取消会在子轮次之间、工具 batch 之间、并发 goroutine 内中止执行；取消时触发 `HookStop`。

## Context 传播

[`RunTurn`](./runner_turn.go) 通过 `context.Context` 向嵌套工具传递状态（[`context.go`](./context.go)）：

| 机制 | 源文件 | 用途 |
|------|--------|------|
| `WithActiveTurn` / `InActiveTurn` | [context.go](./context.go) | 标记父 Runner 是否在用户回合中（通知优先级） |
| `WithTurnCallbacks` | [context.go](./context.go) | 子代理 UI 回调转发 |
| `WithToolInvocation` | [context.go](./context.go) | 工具获知 parent sessionID + toolCallID |
| `WithForkContext` / `WithRenderedSystem` | [fork_context.go](./fork_context.go) | Fork 子代理继承父会话消息与 system |

`WithActiveTurn` 使用 refcount：父 Runner 与子 Runner 嵌套时仍视为 active turn，PrioLater 通知不会误升为 PrioNow。

[`SubagentToolCallbacks`](./context.go) 将子代理工具事件映射到 `OnSubagentToolStart` / `OnSubagentToolEnd`。

## TurnCallbacks

定义见 [`callbacks.go`](./callbacks.go)。TUI 在 [`run.go`](../ui/tui/run.go) 实现这些可选 hook，映射为 Bubble Tea 消息：

```go
type TurnCallbacks struct {
    OnContentDelta, OnReasoningDelta
    OnToolStart, OnToolEnd
    OnAssistantSegmentEnd
    OnPlanningStart, OnPlanningEnd
    OnSubagentStart, OnSubagentEnd
    OnSubagentToolStart, OnSubagentToolEnd
    OnUsageUpdate
}
```

## 生命周期 Hook

实现：[`hooks.go`](./hooks.go)、[`hook_input.go`](./hook_input.go)、[`hook_apply.go`](./hook_apply.go)。

`.ds-code/hooks.json` 定义外部 shell 脚本，通过 `HOOK_INPUT` 环境变量接收 JSON：

| 事件 | 触发时机 |
|------|----------|
| `SessionStart` | 会话首条用户消息 |
| `SessionEnd` | [`EndSessionHooks`](./session_hooks.go) |
| `PreToolUse` | 工具执行前（stdout JSON 可改写 shell command / args） |
| `PostToolUse` | 工具执行后 |
| `SubagentStart` / `SubagentStop` | 子代理启停 |
| `Stop` | 用户回合正常结束、MaxTurns、或 ctx 取消 |

`HookStop` 的 `transition` 字段标识结束原因；`max_turns` 时 `error` 为说明性文本，非失败。

### PreToolUse 改写协议

Hook stdout 若为合法 JSON，可改写即将执行的工具参数：

```json
{ "command": "safe-cmd", "args": { "path": "/new/path" } }
```

- `command` 仅对 `shell` 工具有效
- `args` 合并进工具参数字典（[`hook_apply.go`](./hook_apply.go)）

## MaxTurns Soft Landing

实现：[`finishMaxTurnsExceeded`](./runner_loop.go)、常量 [`text.go`](./text.go)。

达到 `MaxTurns` 时：

1. 写入 history-only system 事件
2. 构造无工具的摘要 LLM 请求（摘要 prompt 仅注入 API，**不**持久化为 user 行）
3. `HookStop` 携带 `transition: "max_turns"`
4. 摘要 LLM 失败时写入 fallback assistant 消息

## Checkpoint

写类工具（`apply_patch`、`write_file`）执行前，[`recordCheckpoint`](./checkpoint.go) 捕获目标文件快照到 [`checkpoint.Store`](../checkpoint/store.go)。

[`RewindCheckpoint`](./checkpoint.go) 恢复工作区并追加 system 事件消息。

## RunEphemeral（/btw）

实现：[`ephemeral.go`](./ephemeral.go)。

旁路问答通道：

- 无工具、非流式、`ThinkingType: disabled`
- 默认不写 session 历史；可选 `IncludeRecentTurns` 带入近期 API 上下文
- 可选 `CountTowardSession` 累计用量

## spawn 子包：子代理调度

`agent` 工具的全部逻辑在 [`spawn.Service.Handle`](./spawn/service.go) 中完成，替代旧 `subagent` 包。详细设计见 [`spawn/README.md`](spawn/README.md)。

### 路由决策（[`router.go`](./spawn/router.go)）

```
subagent_type 省略 + fork_enabled + interactive?
  ├─ 是 → Fork（继承父工具池与 system，共享 prompt cache 前缀）
  └─ 否 → force_background（verification）或 run_in_background?
            ├─ 是 → Async（后台 goroutine）
            └─ 否 → Sync（同步等待，可超时 promote 为 Async）
```

Fork 防护：禁止从 Fork 子代理再 Fork；Fork 子消息含 `fork-boilerplate` 标签用于递归检测。

### Fork 消息构造（[`fork.go`](./spawn/fork.go)）

[`BuildForkMessages`](./spawn/fork.go) 为共享 prompt cache 前缀而设计：

1. 复制父 API 消息直到触发 fork 的 assistant（含全部 `tool_use` 块）
2. 对每个 parent tool_call 追加 `role=tool`，内容为固定占位符 `ForkPlaceholder`（**所有 fork 子代理字节相同**，保证 cache 前缀一致）
3. 追加 `role=user`：`ForkBoilerplate` + `[directive: "..."]` 包裹的任务 prompt

子代理 `ThinkingType` 继承父会话（与常量子代理默认 `disabled` 不同）。

### Skill Fork（[`skill.go`](./spawn/skill.go)）

`.ds-code/skills/<name>/SKILL.md` 若 front matter 含 `context: fork`，TUI slash 走 [`spawn.Service.FromSkill`](./spawn/skill.go)：不经过 LLM `agent` 工具参数，直接复用 Fork 路径，`QuerySource=skill:fork`。

### 内置 Agent 类型（[`registry.go`](./spawn/registry.go)）

| 类型 | 特点 |
|------|------|
| `general-purpose` | 全工具（除 agent），可读写 |
| `Explore` | 只读探索，省略 AGENTS.md/rules |
| `Plan` | 只读规划 |
| `verification` | 只读、强制后台、对抗性验证 |
| `fork`（合成） | 继承父会话，bubble 权限 |

### ExecuteRun 流程（[`execute.go`](./spawn/execute.go)）

1. 按类型过滤工具池（[`FilterToolRegistry`](./spawn/toolpool.go)）：全局禁 agent → 类型 disallow → 后台白名单
2. 配置权限引擎（readonly / inherit / bubble；worktree 时 rebind workspace）
3. 创建 `sessionStore` 适配器（映射到 `subagentstore`）
4. 构造子 `context.Service`（Fork 用 `ForkView`；常规模型用 `AgentOverlay`）
5. 创建子 `Runner`（`ForSubagent: true`）
6. `RunTurn` 或 `RunTurnSeeded`（Fork）
7. 返回截断后的 `FinalContent` 摘要

### 子代理 Model 与 MaxTurns（[`model.go`](./spawn/model.go)）

Model 优先级：`agent` 工具参数 → 类型定义（非 `inherit`）→ `llm.subagent.model` → 主模型。

子代理 `MaxTurns` 默认 8（`llm.subagent.max_turns`），主 Runner 默认 25（`agent.max_turns`）。

### System Overlay 与 Agent Memory

- **[prompts.go](./spawn/prompts.go)**：Explore / Plan / verification 注入额外 system 段（行为约束、VERDICT 格式等）
- **[memory.go](./spawn/memory.go)**：从 `~/.ds-code/agent-memory/{type}/` 读取最近 3 个 slot 文件（`user|feedback|project|reference`），截断至 ~7000 字符，包在 `<agent-memory>` 注入子代理 system

### 工具池三层过滤（[`toolpool.go`](./spawn/toolpool.go)）

| 层 | 规则 |
|----|------|
| Layer 1 | 全局禁止 `agent`（防嵌套 spawn） |
| Layer 2 | 类型 `DisallowedTools`（如 Explore 禁 write_file） |
| Layer 2.5 | 后台 agent 仅允许 async 白名单（read/grep/glob/shell/write 等） |

Worktree 子 agent 通过 `tool.RebindRegistryPerm` 将 workspace 切到 worktree 路径。

### 结果交付（[`output.go`](./spawn/output.go)）

子代理摘要超过 `tools.agent.summary_max_chars`（默认 16000 字符）或 1 MiB 时 **spill** 到 `{project_data}/agents/{session}/{tool_call_id}.output`，父工具返回 `{"status":"completed","output_file":"..."}` + `SavedResultHint`；否则 inline 返回 `[description]\n{summary}`。

### 后台 Agent 与通知

- **[BackgroundManager](./spawn/background.go)**：管理 in-flight 异步 agent，支持 Kill
- **Sync → Async promote**：[`runSync`](./spawn/service.go) 超过 `auto_background_after` 秒未完成则转为后台，立即返回 `async_launched`
- **[NotificationQueue](./spawn/notify.go)**：三级优先级
  - `PrioNow`：空闲时立即注入
  - `PrioNext`：下次 RunTurn 开头
  - `PrioLater`：父回合工具执行后（[`DrainNotificationsLater`](./runner_turn.go)）

通知格式为 `<task-notification>` XML：

- **PrioNow / PrioNext**：`DrainNotifications` 拼接到下一条 user 消息**文本前**（不单独写 DB 行）
- **PrioLater**：`DrainNotificationsLater` 以 `role=user` 追加 `<task-notification>` 到 session（下一轮 PrepareRequest 可见）

后台 goroutine 通过 [`DetachSpawnContext`](./spawn/context.go) 脱离父 ctx 取消，但保留 ForkContext 等 value；父 ctx 取消不会 kill 已 promote 的后台 agent。

### Worktree 隔离

`isolation: worktree` 时（仅 general-purpose）在 `{project_data}/worktrees/` 创建 git worktree（sparse checkout + node_modules 等 symlink）。

| 时机 | 行为 |
|------|------|
| 失败 / killed | [`cleanupWorktreeImmediate`](./spawn/worktree_cleanup.go) 立即删除 |
| 会话结束 | [`CleanupSessionWorktrees`](./spawn/worktree_cleanup.go) |
| 启动 / 定期 | [`CleanupExpiredWorktrees`](./spawn/worktree_cleanup.go)（默认 TTL 24h，`tools.agent.worktree_ttl`） |

## 配置项速查

| 配置键 | 作用 |
|--------|------|
| `agent.max_turns` | 主 Runner 子轮次上限（默认 25） |
| `llm.subagent.max_turns` | 子代理子轮次上限（默认 8） |
| `llm.subagent.model` / `fallback_model` | 子代理模型与 5xx 降级 |
| `tools.agent.fork_enabled` | 省略 subagent_type 时走 Fork |
| `tools.agent.auto_background_after` | Sync 超时 promote 为 Async（秒，默认 120） |
| `tools.agent.max_parallel` | 并发 agent 工具上限（默认 3） |
| `tools.agent.summary_max_chars` | 摘要 inline 字符上限 |
| `tools.agent.worktree_*` | worktree TTL、sparse paths、symlink dirs |
| `context.tool_result_max_chars` | 工具结果持久化截断 |
| `btw.max_tokens` | `/btw` 默认 max tokens |

## 用量与计费

每个 assistant 消息通过 [`enrichAssistantUsage`](./usage_billing.go) 附加：

- `PromptTokens` / `CompletionTokens` / `PromptCacheHitTokens`
- `ModelID` + `PricingSnapshotJSON`
- `EstimatedCostCNY`

子轮次结束后 [`applySubRoundUsage`](./runner_loop.go) 更新 session 累计用量并触发 `OnUsageUpdate`。

## 相关包

| 包 / 文件 | 关系 |
|-----------|------|
| [`internal/context`](../context/service.go) | API 上下文构建、compact、@ 展开 |
| [`internal/ui/tui`](../ui/tui/run.go) | `TurnCallbacks` → Bubble Tea 消息 |
| [`internal/session/subagentstore`](../session/subagentstore/types.go) | 子代理 run 与消息持久化 |
| [`internal/tool/builtin/agent`](../tool/builtin/agent/agent.go) | `agent` 工具入口，委托 `spawn.Service` |
| [`internal/toolresult`](../toolresult/format.go) | 工具结果 XML 包装与截断（[context 包 re-export](../context/toolresult.go)） |
| [`cmd/ds-code/app`](../../cmd/ds-code/app/runner.go) | Runner 组装、通知 drain、worktree 清理 |

## 测试

| 文件 | 覆盖 |
|------|------|
| [runner_test.go](./runner_test.go) / [runner_loop_test.go](./runner_loop_test.go) / [runner_stream_test.go](./runner_stream_test.go) / [runner_planning_test.go](./runner_planning_test.go) | RunTurn、流式、planning UI、checkpoint+hook |
| [recovery_test.go](./recovery_test.go) | chatWithRecovery 各分支、fallback model |
| [tool_orchestration.go](./tool_orchestration.go)（经 runner_test） | 并发读 batch |
| [spawn/*_test.go](spawn/) | 路由、Fork、后台、worktree、integration |
| [hooks_test.go](./hooks_test.go) / [hook_apply_test.go](./hook_apply_test.go) | Hook JSON、PreToolUse 改写 |

集成测试：[spawn/integration_test.go](./spawn/integration_test.go)（端到端 spawn）；TUI 集成在 [`internal/tuitest`](../../internal/tuitest/stack.go)（`-tags=tuitest`）。

## 设计约束

- **消息只增**：`session.Store` 无 Update/Delete；compact 不删历史行
- **双层消息模型**：SQLite 全量历史 + 内存 API 上下文（compact 后替换为摘要 + 近 N 轮）
- **权限 S3 denylist 始终生效**：无论 readonly/ask/auto
- **子代理禁止嵌套 agent 工具**：工具池 Layer 1 全局过滤
- **TUI 取消**：Esc 取消当前轮次（ctx 贯穿）；子轮次间与工具间均响应取消
