package config

// ResolveSubagentModel returns the model for subagent runs.
func (c LLMConfig) ResolveSubagentModel() string {
	if c.Subagent.Model != "" {
		return c.Subagent.Model
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
