# agent

核心 **agent 循环**：一条用户消息可经历多轮 LLM **子轮次**（tool call → tool 结果 → 再次 LLM），直到模型不再返回 `tool_calls` 或达到 `MaxTurns`。

## Runner（`runner.go`）

`Runner` 串联 LLM、工具、权限、会话存储与 context 服务。主要入口：

| API | 用途 |
|-----|------|
| `RunTurn` | 交互式聊天回合（TUI 或带回调的 CLI） |
| `RunEphemeral` | `/btw` 旁路问答——无工具，可选不写历史 |
| `RewindCheckpoint` | 通过 checkpoint 撤销文件写入 |

### RunTurn 流程

```
追加用户消息（@ 展开后）
        │
        ▼
┌─── for round = 0 .. MaxTurns-1 ───┐
│  PrepareRequest → LLM.Chat（流式）  │
│  追加 assistant 消息               │
│  若无 tool_calls → 返回结果        │
│  对每个 tool_call：                │
│    权限 → checkpoint → Execute     │
│    追加 tool 消息                  │
│  OnAssistantSegmentEnd（回调）     │
└────────────────────────────────────┘
```

**子轮次** = 上述循环中的一次 LLM 请求。单条用户消息常包含多轮子轮次（例如读文件 → 改文件 → 回复）。

首轮 planning 由 TUI 在 `submitLine` 时插入；子轮次之间（`round > 0`）由 agent 触发：

- `OnPlanningStart` / `OnPlanningEnd` — TUI 显示「Planning next moves」，直到下一段流式输出开始。
- `OnAssistantSegmentEnd` — TUI 在工具行之前结束当前可见的 assistant 分段。

### 流式（`TurnCallbacks`）

`RunTurn` 始终请求 `Stream: true`。`req.OnStream` 接收 `llm.StreamDelta`：

- **Content** → `OnContentDelta`（并结束 planning UI）。
- **Reasoning** → `OnReasoningDelta`（模型提供 thinking 流时）。

`streamTiming` 记录 reasoning 墙钟时间（首个 reasoning delta → 首个 content delta），用于 `TurnResult.FinalReasoningDuration` 与持久化的 `ReasoningDurationMS`。

若提供商仅在流结束时返回正文，且过程中未流式输出，则仍会从最终 `resp.Content` 调用一次 `OnContentDelta`。

### 上下文过长

遇到 `deepseek.IsContextTooLong` 时，调用一次 `Context.CompactAPIContext` 并重试同一子轮次请求。

### 工具执行（`executeTool`）

顺序：`Perm.Check` → `recordCheckpoint`（写类工具）→ 审计日志 → `Tools.Execute` → 格式化的 tool 消息体（写入 session 时会截断）。

`context.Context` 取消会在子轮次之间、工具之间中止执行。

### MaxTurns soft landing

达到 `MaxTurns` 时写入 history-only 的 system 事件，再发起**无工具**的摘要 LLM 请求（摘要 prompt 仅注入 API 消息，**不**持久化为 user 行）。`HookStop` 的 `HOOK_INPUT` 含 `transition: "max_turns"` 与 `error` 说明超限。Hook 脚本判断 stop 原因应读 `transition`，勿单独以 `error` 非空视为失败。摘要 LLM 失败时写入 fallback assistant 消息并完成 soft landing。

## 相关包

- **`internal/agent/subagent`** — `task` 工具 / slash 子 agent 使用的短时只读 `Runner`（见 subagent README）。
- **`internal/ui/tui`** — 将 `TurnCallbacks` 映射为 Bubble Tea 消息（`run.go`）。
