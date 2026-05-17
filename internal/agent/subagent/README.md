# subagent

只读**嵌套 agent**，供 `task` 工具及需要在不改动主会话的前提下做探索的 slash 流程使用。

`Run` 会构建隔离环境：

- 内存 `session.Store`（非用户当前会话）
- `permission.Engine`，模式为 `"readonly"`
- 由调用方 `RegisterFunc` 填充的全新 `tool.Registry`
- `agent.Runner`，`Out: io.Discard`，且**无** `TurnCallbacks`（不向 TUI 推流）

对用户 prompt 执行一次 `RunTurn`；返回 `FinalContent` 或 `FinalReasoning`，并按 `cfg.Tools.Task.SummaryMaxChars` 截断。

复用与主 agent 相同的子轮次循环，轮次上限为 `min(8, cfg.Agent.MaxTurns)`（在配置生效时）。
