package context

import "github.com/hejunqiu/ds-code/internal/llm"

// APIContextView is the snapshot for the next API request.
type APIContextView struct {
	SystemPrompt string
	AgentsMD     string
	Rules        string
	Skills       string
	GitSnapshot  string
	ToolsJSON    string
	Messages     []llm.Message
	WindowTokens int
}

// MergedSystem returns the single system string for the API.
func (v *APIContextView) MergedSystem() string {
	return mergeSystem(v.SystemPrompt, v.AgentsMD, v.Rules, v.Skills, v.GitSnapshot)
}
