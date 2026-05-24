package toolresult

import (
	"fmt"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

// FormatToolResult wraps tool output for prompt safety (S5).
func FormatToolResult(name, callID, body string) string {
	return fmt.Sprintf("<tool_result name=%q id=%q>\n%s\n</tool_result>", name, callID, body)
}

// FormatToolError wraps a permission or execution error.
func FormatToolError(name, callID string, err error) string {
	return FormatToolResult(name, callID, ToolErrorPrefix+err.Error())
}

// UnpackToolBody extracts the inner body from a formatted tool result or error wrapper.
func UnpackToolBody(formatted string) (body string, isError bool) {
	const prefix = "<tool_result"
	if !strings.HasPrefix(formatted, prefix) {
		trimmed := strings.TrimSpace(formatted)
		return formatted, strings.HasPrefix(trimmed, ToolErrorPrefix) || strings.HasPrefix(trimmed, "error:")
	}
	start := strings.Index(formatted, ">\n")
	if start < 0 {
		return formatted, false
	}
	start += 2
	end := strings.LastIndex(formatted, "\n</tool_result>")
	if end < 0 || end <= start {
		body = formatted[start:]
	} else {
		body = formatted[start:end]
	}
	isError = strings.HasPrefix(body, ToolErrorPrefix) || strings.HasPrefix(body, "error:")
	return body, isError
}

// TruncateToolResult limits tool output size.
func TruncateToolResult(body string, cfg *config.Config) string {
	max := cfg.Context.ToolResultMaxChars
	if max <= 0 || len(body) <= max {
		return body
	}
	suffix := TruncateSuffix
	if len(suffix) >= max {
		return body[:max]
	}
	return body[:max-len(suffix)] + suffix
}
