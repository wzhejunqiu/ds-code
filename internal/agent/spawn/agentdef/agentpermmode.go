package agentdef

import (
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/session"
)

// AgentPermMode selects how a sub-agent relates to the parent permission engine.
// Distinct from session.PermissionMode (readonly / ask / auto): only readonly overlaps
// in name; here it means the sub-agent always uses a readonly engine.
type AgentPermMode string

const (
	AgentPermModeInherit  AgentPermMode = "inherit"  // reuse parent permission.Engine
	AgentPermModeReadonly AgentPermMode = "readonly" // dedicated readonly engine for sub-agent
	AgentPermModeBubble   AgentPermMode = "bubble"   // inherit engine; asks bubble to parent Prompter
)

// String returns the wire-format agent permission mode label.
func (m AgentPermMode) String() string {
	return string(m)
}

// Valid reports whether m is a known agent permission mode (empty defaults to inherit).
func (m AgentPermMode) Valid() bool {
	return m == "" || m == AgentPermModeInherit || m == AgentPermModeReadonly || m == AgentPermModeBubble
}

// ParseAgentPermMode parses a registry string into AgentPermMode.
func ParseAgentPermMode(s string) (AgentPermMode, error) {
	m := AgentPermMode(s)
	if m.Valid() {
		return m, nil
	}
	return "", fmt.Errorf("agentdef: invalid agent permission mode %q", s)
}

// ToSessionMode maps agent perm mode to session.PermissionMode when projecting a synthetic session.
// Only readonly has a session-level equivalent; inherit/bubble defer to the parent session policy.
func (m AgentPermMode) ToSessionMode() session.PermissionMode {
	if m == AgentPermModeReadonly {
		return session.PermissionModeReadonly
	}
	return ""
}
