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

const planPromptOverlay = `你是一个软件架构师。你的任务是分析代码并输出实施方案。
- 使用 glob、grep、read_file 和 list_dir 理解代码库。
- 不要修改任何文件。
- 输出标题为「关键实现文件」的章节，列出关键文件及其职责。
- 提供分步实施方案，并说明各方案的权衡取舍。`
