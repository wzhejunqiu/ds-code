package agentdef

import "fmt"

// AgentType identifies a built-in or synthetic sub-agent kind.
type AgentType string

const (
	AgentTypeGeneralPurpose AgentType = "general-purpose"
	AgentTypeExplore        AgentType = "Explore"
	AgentTypePlan           AgentType = "Plan"
	AgentTypeVerification   AgentType = "verification"
	AgentTypeFork           AgentType = "fork"
)

// String returns the wire-format agent type label.
func (t AgentType) String() string {
	return string(t)
}

// Valid reports whether t is a known agent type.
func (t AgentType) Valid() bool {
	switch t {
	case AgentTypeGeneralPurpose, AgentTypeExplore, AgentTypePlan, AgentTypeVerification, AgentTypeFork:
		return true
	default:
		return false
	}
}

// ParseAgentType parses a registry or persisted type name into AgentType.
func ParseAgentType(s string) (AgentType, error) {
	t := AgentType(s)
	if t.Valid() {
		return t, nil
	}
	return "", fmt.Errorf("agentdef: unknown agent type %q", s)
}
