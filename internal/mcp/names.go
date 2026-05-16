package mcp

import (
	"fmt"
	"regexp"
	"strings"
)

const prefix = "mcp__"

var serverNameRE = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// ValidateServerName checks MCP server id used in tool names.
func ValidateServerName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("mcp: server name is required")
	}
	if !serverNameRE.MatchString(name) {
		return fmt.Errorf("mcp: invalid server name %q (use [a-z][a-z0-9_-]*)", name)
	}
	return nil
}

// ToolName returns the normalized registry name: mcp__{server}__{tool}.
func ToolName(server, tool string) string {
	return prefix + server + "__" + tool
}

// ParseToolName splits a normalized MCP tool name.
func ParseToolName(name string) (server, tool string, ok bool) {
	if !strings.HasPrefix(name, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(name, prefix)
	i := strings.Index(rest, "__")
	if i <= 0 {
		return "", "", false
	}
	server = rest[:i]
	tool = rest[i+2:]
	if tool == "" {
		return "", "", false
	}
	return server, tool, true
}

// IsMCPTool reports whether name is a normalized MCP tool.
func IsMCPTool(name string) bool {
	_, _, ok := ParseToolName(name)
	return ok
}
