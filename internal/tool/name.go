package tool

import "github.com/wzhejunqiu/ds-code/internal/toolname"

// Name is the canonical LLM-visible identifier for a registered builtin tool.
// Each builtin tool's Name() returns the matching constant below.
type Name string

const (
	NameReadFile    Name = "read_file"
	NameWriteFile   Name = "write_file"
	NameApplyPatch  Name = "apply_patch"
	NameShell       Name = toolname.Bash
	NameGlob        Name = "glob"
	NameGrep        Name = "grep"
	NameListDir     Name = "list_dir"
	NameDiagnostics Name = "diagnostics"
	NameWebFetch    Name = "web_fetch"
	NameWebSearch   Name = "web_search"
	NameAgent       Name = "agent"
	NameToolSearch  Name = "tool_search"
)

// String returns the wire-format tool name.
func (n Name) String() string {
	return string(n)
}

// Matches reports whether name equals this tool name.
func (n Name) Matches(name string) bool {
	return n.String() == name
}

// ReservedNames lists all built-in tool name constants (including names not registered when disabled).
// For documentation and tests only; MCP Discover must not use this to pre-skip tools.
func ReservedNames() map[string]struct{} {
	names := []Name{
		NameReadFile, NameWriteFile, NameApplyPatch, NameShell, NameGlob, NameGrep,
		NameListDir, NameDiagnostics, NameWebFetch, NameWebSearch, NameAgent, NameToolSearch,
	}
	out := make(map[string]struct{}, len(names))
	for _, n := range names {
		out[n.String()] = struct{}{}
	}
	return out
}
