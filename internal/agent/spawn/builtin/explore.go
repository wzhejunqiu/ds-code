package builtin

import "github.com/wzhejunqiu/ds-code/internal/agent/spawn/agentdef"

// Explore is a fast read-only codebase exploration sub-agent.
func Explore() agentdef.Definition {
	return agentdef.Definition{
		Type:            agentdef.AgentTypeExplore,
		Description:     "快速只读探索代理，用于搜索代码库、查找文件、阅读代码。不可写文件。",
		Tools:           []string{"*"},
		DisallowedTools: agentdef.DisallowReadOnlyWrites,
		Model:           agentdef.ModelInherit,
		ReadOnly:        true,
		OmitHeavyRules:  true,
		PromptOverlay:   explorePromptOverlay,
	}
}

const explorePromptOverlay = `You are a read-only code explorer. Your job is to search, read, and report.
- Use glob, grep, read_file, and list_dir to find information.
- Do NOT modify any files. Do NOT run shell commands that write to the filesystem.
- Report your findings concisely. Include file paths and line numbers.`
