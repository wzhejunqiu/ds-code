package agentdef

import "github.com/wzhejunqiu/ds-code/internal/tool"

// DisallowAgentOnly blocks nested agent spawns (Layer 2; Layer 1 also blocks agent globally).
var DisallowAgentOnly = []tool.Name{tool.NameAgent}

// DisallowReadOnlyWrites blocks write tools for read-only agent types.
var DisallowReadOnlyWrites = []tool.Name{
	tool.NameAgent,
	tool.NameWriteFile,
	tool.NameApplyPatch,
}
