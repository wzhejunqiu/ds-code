//go:build tuitest

package slash

func tuitestCommands() []Command {
	return []Command{
		{
			Name:        "tcase",
			Args:        "[list|run <name>]",
			Description: "Run built-in TUI integration test scenario (harness only)",
		},
	}
}
