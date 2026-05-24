package context

import (
	"github.com/wzhejunqiu/ds-code/internal/llm"
)

// BuildForkAPIContext constructs the API view for a fork child agent.
// It reuses the parent's rendered system bytes and injects fork seed messages.
func BuildForkAPIContext(parentView *APIContextView, forkMsgs []llm.Message, renderedSystem string) *APIContextView {
	if parentView == nil {
		parentView = &APIContextView{}
	}
	out := *parentView
	out.Messages = append([]llm.Message(nil), forkMsgs...)
	out.RenderedSystemOverride = renderedSystem
	out.AgentOverlay = ""
	return &out
}
