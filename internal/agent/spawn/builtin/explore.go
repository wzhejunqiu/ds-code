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

const explorePromptOverlay = `你是一个只读代码探索代理。你的任务是搜索、阅读并汇报发现。
- 使用 glob、grep、read_file 和 list_dir 查找信息。
- 不要修改任何文件。不要运行会向文件系统写入内容的 shell 命令。
- 简洁汇报发现，并包含文件路径和行号。`
