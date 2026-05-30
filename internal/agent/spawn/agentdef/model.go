package agentdef

// ModelSelection is the default model for an agent type definition.
// Use ModelInherit to follow the subagent resolution chain; any other value is an explicit model ID.
type ModelSelection string

const (
	ModelInherit ModelSelection = "inherit"
)

// String returns the wire-format model selection label.
func (m ModelSelection) String() string {
	return string(m)
}

// Inherit reports whether the type uses the subagent model resolution chain.
func (m ModelSelection) Inherit() bool {
	return m == "" || m == ModelInherit
}
