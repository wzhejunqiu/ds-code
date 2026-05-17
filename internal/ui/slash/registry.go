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
	cmds := []Command{
		{Name: "help", Description: "Show all slash commands"},
		{Name: "context", Args: "", Description: "Show session usage and next-request token breakdown"},
		{Name: "mode", Args: "[deepseek-v4-pro|flash]", Description: "View or switch model (per-session)"},
		{Name: "effort", Args: "[high|max]", Description: "View or switch reasoning effort"},
		{Name: "thinking", Args: "[on|off]", Description: "Toggle thinking mode"},
		{Name: "clear", Description: "Start a new session (history kept in DB after Phase 3)"},
		{Name: "btw", Args: "<question…>", Description: "Side-channel question (not in main thread)"},
		{Name: "compact", Description: "Manually compact API context"},
		{Name: "resume", Args: "[session_id]", Description: "Resume a saved session"},
		{Name: "plan", Description: "Enter Plan mode (read-only tools)"},
		{Name: "agent", Description: "Return to Agent mode"},
		{Name: "permissions", Description: "View or switch permission mode"},
		{Name: "checkpoint", Args: "[list|rewind n [--yes]]", Description: "Checkpoint list or rewind (requires --yes)"},
		{Name: "rewind", Args: "<n> [--yes]", Description: "Rewind workspace to checkpoint n (requires --yes)"},
		{Name: "git", Description: "Refresh git status/diff snapshot for next request"},
		{Name: "skill", Args: "[name]", Description: "List or activate a skill"},
		{Name: "task", Args: "<prompt…>", Description: "Dispatch read-only subagent (direct)"},
	}
	return append(cmds, devCommands()...)
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
