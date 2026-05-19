package context

import (
	"github.com/hejunqiu/ds-code/internal/config"
	toolresultpkg "github.com/hejunqiu/ds-code/internal/toolresult"
)

// FormatToolResult wraps tool output for prompt safety (S5).
func FormatToolResult(name, callID, body string) string {
	return toolresultpkg.FormatToolResult(name, callID, body)
}

// FormatToolError wraps a permission or execution error.
func FormatToolError(name, callID string, err error) string {
	return toolresultpkg.FormatToolError(name, callID, err)
}

// UnpackToolBody extracts the inner body from a formatted tool result or error wrapper.
func UnpackToolBody(formatted string) (body string, isError bool) {
	return toolresultpkg.UnpackToolBody(formatted)
}

// TruncateToolResult limits tool output size.
func TruncateToolResult(body string, cfg *config.Config) string {
	return toolresultpkg.TruncateToolResult(body, cfg)
}
