package builtin

import "github.com/wzhejunqiu/ds-code/internal/agent/spawn/agentdef"

// Fork is a synthetic type used when subagent_type is omitted and fork is enabled.
// It uses the parent's exact tool pool and rendered system prompt.
func Fork() agentdef.Definition {
	return agentdef.Definition{
		Type:            agentdef.AgentTypeFork,
		Description:     "Fork 子代理，继承父会话的工具池和 system prompt，共享 prompt cache 前缀。",
		Tools:           []string{"*"},
		DisallowedTools: agentdef.DisallowAgentOnly,
		Model:           agentdef.ModelInherit,
		PermissionMode:  agentdef.AgentPermModeBubble,
	}
}
