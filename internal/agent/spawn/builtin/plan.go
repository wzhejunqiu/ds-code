package builtin

import "github.com/wzhejunqiu/ds-code/internal/agent/spawn/agentdef"

// Plan is a read-only architecture planning sub-agent.
func Plan() agentdef.Definition {
	return agentdef.Definition{
		Type:            agentdef.AgentTypePlan,
		Description:     "只读架构规划代理，分析代码结构并输出实施方案。不可写文件。",
		Tools:           []string{"*"},
		DisallowedTools: agentdef.DisallowReadOnlyWrites,
		Model:           agentdef.ModelInherit,
		ReadOnly:        true,
		OmitHeavyRules:  true,
		PromptOverlay:   planPromptOverlay,
	}
}

const planPromptOverlay = `You are a software architect. Your job is to analyze code and produce implementation plans.
- Use glob, grep, read_file, and list_dir to understand the codebase.
- Do NOT modify any files.
- Output a section titled "Critical Files for Implementation" listing key files and their roles.
- Provide a step-by-step plan with trade-offs noted.`
