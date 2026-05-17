package tui

// chatBlockRole identifies a row in the TUI chat transcript.
type chatBlockRole int

const (
	chatRoleUser      chatBlockRole = iota // user prompt ("> …")
	chatRoleAssistant                      // model reply (streaming or final)
	chatRoleTool                           // tool call + result in the main transcript
	chatRolePlanning                       // "Planning next moves" spinner between LLM rounds
	chatRoleInterrupt                      // turn cancelled via Esc (also restored from session history)
)

// String returns the stable role name used in tests and persisted system markers.
func (r chatBlockRole) String() string {
	switch r {
	case chatRoleUser:
		return "user"
	case chatRoleAssistant:
		return "assistant"
	case chatRoleTool:
		return "tool"
	case chatRolePlanning:
		return "planning"
	case chatRoleInterrupt:
		return "interrupt"
	default:
		return "unknown"
	}
}
