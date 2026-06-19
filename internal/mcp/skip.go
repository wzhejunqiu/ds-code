package mcp

// SkipReason explains why an MCP tool was not registered.
type SkipReason string

const (
	SkipBuiltinConflict      SkipReason = "builtin_conflict"
	SkipCrossServerDuplicate SkipReason = "cross_server_duplicate"
	SkipInServerDuplicate    SkipReason = "in_server_duplicate"
)

// SkippedTool records one MCP tool that was not loaded into the registry.
type SkippedTool struct {
	Server string
	Tool   string
	Reason SkipReason
}
