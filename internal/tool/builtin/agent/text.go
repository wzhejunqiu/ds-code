package agent

const (
	DescAgent = "启动一个子代理处理复杂多步骤任务。有 4 种类型：general-purpose（全能）、Explore（只读探索）、Plan（架构规划）、verification（验证）。可用时优先并行启动多个 agent。"

	SchemaAgentDescription = "简短描述（3-5 个词）"
	SchemaAgentPrompt      = "子代理的任务说明"
	SchemaAgentType        = "子代理类型。省略时若 fork 启用则走 Fork 路径，否则默认 general-purpose"
	SchemaAgentModel       = "可选模型覆盖（sonnet / opus / haiku）"
	SchemaAgentBackground  = "设为 true 时在后台异步运行"
	SchemaAgentIsolation   = "隔离模式：worktree（仅 general-purpose）"

	ErrMissingParent = "agent: 缺少父会话或 tool call id"
	ErrNoStore       = "agent: 未配置 agent store"
)
