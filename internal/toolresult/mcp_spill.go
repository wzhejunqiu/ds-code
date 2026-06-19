package toolresult

import "fmt"

// MCPSavedResultHint formats the spill path hint appended to truncated MCP tool results.
func MCPSavedResultHint(displayPath string) string {
	return fmt.Sprintf("\n... [MCP 完整结果已保存至 %s；请用 read_file 读取该绝对路径（shell 无法访问）]", displayPath)
}

// ShortenSpillPathForHint returns displayPath for MCPSavedResultHint.
// The full absolute path is always returned; hint template length is handled by the caller's budget.
func ShortenSpillPathForHint(absPath string, _ int) string {
	return absPath
}
