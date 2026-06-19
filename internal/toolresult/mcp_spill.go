package toolresult

// MCPSavedResultHint formats the spill path hint appended to truncated MCP tool results.
func MCPSavedResultHint(displayPath string) string {
	return SavedResultHint(displayPath)
}

// ShortenSpillPathForHint returns displayPath for MCPSavedResultHint.
// The full absolute path is always returned; hint template length is handled by the caller's budget.
func ShortenSpillPathForHint(absPath string, _ int) string {
	return absPath
}
