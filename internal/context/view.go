package context

import (
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/prompt"
)

// APIContextView is the snapshot for the next API request.
type APIContextView struct {
	SystemPrompt            string
	RuntimeEnv              string
	AgentsMD                string
	Rules                   string
	Skills                  string
	GitSnapshot             string
	AgentOverlay            string
	ToolsJSON               string
	Messages                []llm.Message
	WindowTokens            int
	RenderedSystemOverride  string
}

// MergedSystem returns the single system string for the API.
func (v *APIContextView) MergedSystem() string {
	if v.RenderedSystemOverride != "" {
		return v.RenderedSystemOverride
	}
	return prompt.MergeSystem(v.SystemPrompt, v.RuntimeEnv, v.AgentsMD, v.Rules, v.Skills, v.GitSnapshot, v.AgentOverlay)
}
