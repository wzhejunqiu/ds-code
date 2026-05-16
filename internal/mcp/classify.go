package mcp

import (
	"strings"

	"github.com/hejunqiu/ds-code/internal/permission"
	mcpsdk "github.com/mark3labs/mcp-go/mcp"
)

// ClassifyPermission maps MCP tool metadata to ds-code permission levels.
func ClassifyPermission(t mcpsdk.Tool) permission.Level {
	if t.Annotations.ReadOnlyHint != nil && *t.Annotations.ReadOnlyHint {
		return permission.LevelLow
	}
	if isWriteMCPToolName(t.Name) {
		if t.Annotations.DestructiveHint != nil && *t.Annotations.DestructiveHint {
			return permission.LevelHighest
		}
		return permission.LevelHigh
	}
	if t.Annotations.OpenWorldHint != nil && *t.Annotations.OpenWorldHint {
		return permission.LevelMedium
	}
	return permission.LevelLow
}

// IsWriteMCPToolName reports whether an MCP tool (original name) performs writes.
func IsWriteMCPToolName(tool string) bool {
	return isWriteMCPToolName(tool)
}

func isWriteMCPToolName(name string) bool {
	lower := strings.ToLower(name)
	writePrefixes := []string{
		"write_", "create_", "delete_", "remove_", "move_", "rename_",
		"edit_", "patch_", "apply_", "execute", "run_", "shell",
		"upload_", "mkdir", "rmdir",
	}
	for _, p := range writePrefixes {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// IsWriteTool reports whether a normalized registry tool name requires write permission.
func IsWriteTool(normalized string) bool {
	_, tool, ok := ParseToolName(normalized)
	if !ok {
		return false
	}
	return isWriteMCPToolName(tool)
}
