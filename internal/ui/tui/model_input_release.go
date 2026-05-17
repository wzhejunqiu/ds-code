//go:build !debug

package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *model) trySubmitDevSlash(cmd, _ string) (tea.Cmd, bool) {
	return nil, false
}
