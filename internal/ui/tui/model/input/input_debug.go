//go:build debug

package input

import tea "github.com/charmbracelet/bubbletea"

func trySubmitDevSlash(cmd, _ string) (tea.Cmd, bool) {
	return nil, false
}
