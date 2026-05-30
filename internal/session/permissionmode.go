package session

import "fmt"

// PermissionMode is the session-level permission policy (readonly / ask / auto).
type PermissionMode string

const (
	PermissionModeReadonly PermissionMode = "readonly"
	PermissionModeAsk      PermissionMode = "ask"
	PermissionModeAuto     PermissionMode = "auto"
)

// String returns the wire-format permission mode label.
func (m PermissionMode) String() string {
	return string(m)
}

// Valid reports whether m is a known permission mode (empty is allowed for unset).
func (m PermissionMode) Valid() bool {
	return m == "" || m == PermissionModeReadonly || m == PermissionModeAsk || m == PermissionModeAuto
}

// ParsePermissionMode parses a config or DB string into PermissionMode.
func ParsePermissionMode(s string) (PermissionMode, error) {
	m := PermissionMode(s)
	if m.Valid() {
		return m, nil
	}
	return "", fmt.Errorf("session: invalid permission_mode %q", s)
}
