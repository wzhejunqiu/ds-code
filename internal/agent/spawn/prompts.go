package spawn

// SystemPromptOverlay returns additional system prompt text injected
// into the child agent's dynamic system section for the given type.
func SystemPromptOverlay(def AgentTypeDefinition) string {
	return def.PromptOverlay
}
