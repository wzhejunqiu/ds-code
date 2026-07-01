package session

import "github.com/wzhejunqiu/ds-code/internal/permissionmode"

// PermissionMode is the session-level permission policy (readonly / ask / auto).
type PermissionMode = permissionmode.Mode

const (
	PermissionModeReadonly = permissionmode.Readonly
	PermissionModeAsk      = permissionmode.Ask
	PermissionModeAuto     = permissionmode.Auto
)

// ParsePermissionMode parses a config or DB string into PermissionMode.
func ParsePermissionMode(s string) (PermissionMode, error) {
	return permissionmode.Parse(s)
}
