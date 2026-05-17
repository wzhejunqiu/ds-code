//go:build debug

package slash

func devCommands() []Command {
	return []Command{
		{Name: "debug-panic", Args: "", Description: "Test TUI crash recovery on next update (debug build only)"},
	}
}
