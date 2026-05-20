//go:build tuitest

package slash

func tuitestCommands() []Command {
	return []Command{
		{
			Name:        "tcase",
			Args:        "[list|run <name>]",
			Description: "Pick or run built-in TUI integration scenario (harness only)",
		},
	}
}
