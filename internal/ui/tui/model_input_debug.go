//go:build debug

package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *model) trySubmitDevSlash(cmd, _ string) (tea.Cmd, bool) {
	if cmd != "debug-panic" {
		return nil, false
	}
	ArmTestPanic("update")
	m.errLine = "debug: next update will panic once (TUI should recover)"
	return nil, true
}
