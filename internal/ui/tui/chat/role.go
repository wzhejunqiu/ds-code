package chat

// Role identifies a row in the TUI chat transcript.
type Role int

const (
	RoleUser Role = iota
	RoleAssistant
	RoleTool
	RolePlanning
	RoleInterrupt
)

// String returns the stable role name used in tests and persisted system markers.
func (r Role) String() string {
	switch r {
	case RoleUser:
		return "user"
	case RoleAssistant:
		return "assistant"
	case RoleTool:
		return "tool"
	case RolePlanning:
		return "planning"
	case RoleInterrupt:
		return "interrupt"
	default:
		return "unknown"
	}
}
