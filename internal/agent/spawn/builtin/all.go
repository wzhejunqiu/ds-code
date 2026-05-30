package builtin

import "github.com/wzhejunqiu/ds-code/internal/agent/spawn/agentdef"

// All returns every built-in and synthetic agent definition.
func All() []agentdef.Definition {
	return []agentdef.Definition{
		GeneralPurpose(),
		Explore(),
		Plan(),
		Verification(),
		Fork(),
	}
}
