package tool

// DeferredTool is implemented by tools whose full schema should be lazy-loaded
// to reduce system prompt size. Only a stub schema is sent to the LLM;
// the full schema is retrieved via tool_search when needed.
type DeferredTool interface {
	Tool
	// StubSchema returns a minimal schema (name + one-line description) for the LLM.
	StubSchema() map[string]any
	// ShouldDefer reports whether this tool should use deferred loading.
	ShouldDefer() bool
}
