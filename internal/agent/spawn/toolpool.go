package spawn

import "github.com/wzhejunqiu/ds-code/internal/tool"

// FilterToolRegistry builds a filtered tool.Registry for a child agent,
// applying Layer 1 (global block), Layer 2 (type disallowed), and
// optional Layer 2.5 (async whitelist).
func FilterToolRegistry(parent *tool.Registry, def AgentTypeDefinition, background bool) *tool.Registry {
	child := tool.NewRegistry()
	globalBlock := globalDisallowedTools()

	for _, t := range parent.All() {
		if isBlocked(t.Name(), globalBlock) {
			continue
		}
		if isBlocked(t.Name(), def.DisallowedTools) {
			continue
		}
		if background && !isAsyncTool(t.Name()) {
			continue
		}
		child.Register(t)
	}
	return child
}

// globalDisallowedTools returns tools forbidden for ALL child agents (Layer 1).
func globalDisallowedTools() []tool.Name {
	return []tool.Name{tool.NameAgent}
}

func isBlocked(name string, blocked []tool.Name) bool {
	for _, b := range blocked {
		if b.Matches(name) {
			return true
		}
	}
	return false
}

// isAsyncTool checks whether a tool is permitted for background (async) agents.
func isAsyncTool(name string) bool {
	return asyncAllowed[tool.Name(name)]
}

var asyncAllowed = map[tool.Name]bool{
	tool.NameReadFile:    true,
	tool.NameGlob:        true,
	tool.NameGrep:        true,
	tool.NameListDir:     true,
	tool.NameDiagnostics: true,
	tool.NameWebFetch:    true,
	tool.NameWebSearch:   true,
	tool.NameShell:       true,
	tool.NameWriteFile:   true,
	tool.NameApplyPatch:  true,
}
