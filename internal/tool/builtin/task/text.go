package task

const (
	DescTask = "启动只读子代理并行探索代码库，返回文本摘要。"

	SchemaTaskDescription = "本次探索的简短标签"
	SchemaTaskPrompt      = "交给子代理的详细说明"

	ErrMissingParent = "task: 缺少父会话或 tool call id"
	ErrNoSubStore    = "task: 未配置 subagent store"
)
