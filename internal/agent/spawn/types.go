package spawn

import "github.com/wzhejunqiu/ds-code/internal/agent/spawn/agentdef"

// Shared agent definition types (defined in agentdef for use by builtin without import cycles).
type (
	AgentType           = agentdef.AgentType
	AgentPermMode       = agentdef.AgentPermMode
	AgentTypeDefinition = agentdef.Definition
	ModelSelection      = agentdef.ModelSelection
)

const (
	AgentTypeGeneralPurpose = agentdef.AgentTypeGeneralPurpose
	AgentTypeExplore        = agentdef.AgentTypeExplore
	AgentTypePlan           = agentdef.AgentTypePlan
	AgentTypeVerification   = agentdef.AgentTypeVerification
	AgentTypeFork           = agentdef.AgentTypeFork

	AgentPermModeInherit  = agentdef.AgentPermModeInherit
	AgentPermModeReadonly = agentdef.AgentPermModeReadonly
	AgentPermModeBubble   = agentdef.AgentPermModeBubble

	ModelInherit = agentdef.ModelInherit
)

// ParseAgentType parses a registry or persisted type name into AgentType.
func ParseAgentType(s string) (AgentType, error) {
	return agentdef.ParseAgentType(s)
}
