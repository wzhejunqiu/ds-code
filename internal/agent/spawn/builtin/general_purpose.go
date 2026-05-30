package builtin

import "github.com/wzhejunqiu/ds-code/internal/agent/spawn/agentdef"

// GeneralPurpose is the default full-capability sub-agent.
func GeneralPurpose() agentdef.Definition {
	return agentdef.Definition{
		Type:            agentdef.AgentTypeGeneralPurpose,
		Description:     "全能子代理，可读写文件、执行 shell、使用全部工具。适用于多步骤复杂任务。",
		Tools:           []string{"*"},
		Model:           agentdef.ModelInherit,
		DisallowedTools: agentdef.DisallowAgentOnly,
	}
}
