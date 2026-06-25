package config

// ResolveSubagentModel returns the model for subagent runs.
// When llm.subagent.model is unset, parentModel is used before falling back to llm.model.
func (c LLMConfig) ResolveSubagentModel(parentModel string) string {
	if c.Subagent.Model != "" {
		return c.Subagent.Model
	}
	if parentModel != "" {
		return parentModel
	}
	return c.Model
}

// ResolveSubagentThinkingType returns thinking.type for subagent runs.
func (c LLMConfig) ResolveSubagentThinkingType() string {
	if c.Subagent.Thinking.Type != "" {
		return c.Subagent.Thinking.Type
	}
	return c.Thinking.Type
}

// ResolveSubagentReasoningEffort returns reasoning_effort for subagent runs.
func (c LLMConfig) ResolveSubagentReasoningEffort() string {
	if c.Subagent.ReasoningEffort != "" {
		return c.Subagent.ReasoningEffort
	}
	return c.ReasoningEffort
}
