# subagent

只读**嵌套 agent**，供 `task` 工具及需要在不改动主会话的前提下做探索的 slash 流程使用。

`Run` 会构建隔离环境：

- `subagentstore.Store`（`subagent_runs` / `subagent_messages` 表，与主 `sessions` / `messages` 分离）
- 通过 `sessionStore` 适配器供 `agent.Runner` 追加 transcript 与 `AddUsage`（仅累加对应 run 行）
- `permission.Engine`，模式为 `"readonly"`
- 由调用方 `RegisterFunc` 填充的全新 `tool.Registry`
- `agent.Runner`，`Out: io.Discard`；经 `task` 调用时可传入 `TurnCallbacks`，仅转发嵌套工具事件到主 TUI

对用户 prompt 执行一次 `RunTurn`；返回 `FinalContent` 或 `FinalReasoning`，并按 `cfg.Tools.Task.SummaryMaxChars` 截断。主会话仅收到一条 `tool_name=task` 摘要；**`BuildAPIContext` 不读取 subagent 表**。

`task` 工具在 `CreateRun` 时写入 `parent_tool_call_id`（= 主 LLM `tool_call.id`），TUI `/resume` 通过 `LoadSubagentRegistry` 恢复 ↓ 列表与详情。

复用与主 agent 相同的子轮次循环，轮次上限为 `min(8, cfg.Agent.MaxTurns)`（在配置生效时）。

整会话用量展示（状态栏、`/context`、compact 阈值）通过 `usageagg.TotalForSession` 聚合主 `sessions.*_total` 与同 parent 下 `SUM(subagent_runs)`。
