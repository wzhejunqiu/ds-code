package slash

// Command describes a slash command for help and completion.
type Command struct {
	Name        string
	Args        string
	Description string
	Phase       string // empty = implemented in current phase
}

// All returns the canonical command registry (single source of truth).
func All() []Command {
	return []Command{
		{Name: "help", Description: "Show all slash commands"},
		{Name: "context", Args: "", Description: "Visualize API context token breakdown", Phase: "4"},
		{Name: "mode", Args: "[deepseek-v4-pro|flash]", Description: "View or switch model (per-session)"},
		{Name: "effort", Args: "[high|max]", Description: "View or switch reasoning effort"},
		{Name: "thinking", Args: "[on|off]", Description: "Toggle thinking mode"},
		{Name: "clear", Description: "Start a new session (history kept in DB after Phase 3)"},
		{Name: "btw", Args: "<question…>", Description: "Side-channel question (not in main thread)", Phase: "4"},
		{Name: "compact", Description: "Manually compact API context", Phase: "3"},
		{Name: "resume", Args: "[session_id]", Description: "Resume a saved session", Phase: "3"},
		{Name: "plan", Description: "Enter Plan mode (read-only tools)", Phase: "6"},
		{Name: "agent", Description: "Return to Agent mode", Phase: "6"},
		{Name: "permissions", Description: "View or switch permission mode"},
		{Name: "checkpoint", Args: "[list|rewind n]", Description: "Checkpoint list or rewind", Phase: "7"},
		{Name: "git", Description: "Refresh git status/diff snapshot for next request"},
		{Name: "skill", Args: "<name>", Description: "Activate a skill", Phase: "6"},
		{Name: "task", Args: "<description…>", Description: "Dispatch read-only subagent", Phase: "6"},
	}
}

// Lookup returns a command by name.
func Lookup(name string) (Command, bool) {
	for _, c := range All() {
		if c.Name == name {
			return c, true
		}
	}
	return Command{}, false
}
