package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func isExitConfirmKey(s string) bool {
	return s == "ctrl+c" || s == "ctrl+d"
}

func exitConfirmHintFor(key string) string {
	if key == "ctrl+c" {
		return "Press Ctrl+C again to exit"
	}
	return "Press Ctrl+D again to exit"
}

func (m *model) handleExitKey(key string) (tea.Model, tea.Cmd) {
	if m.running {
		m.clearExitConfirm()
		m.errLine = runningTurnHint
		return m, exitConfirmTimeoutTick()
	}
	return m.handleExitConfirmKey(key)
}

func (m *model) handleExitConfirmKey(key string) (tea.Model, tea.Cmd) {
	if m.exitConfirmPending {
		if m.exitConfirmKey != key {
			return m, nil
		}
		return m, tea.Quit
	}
	m.exitConfirmPending = true
	m.exitConfirmKey = key
	m.exitConfirmArmedAt = time.Now()
	m.errLine = exitConfirmHintFor(key)
	return m, exitConfirmTimeoutTick()
}

func (m *model) clearExitConfirm() {
	if !m.exitConfirmPending {
		return
	}
	hint := exitConfirmHintFor(m.exitConfirmKey)
	m.exitConfirmPending = false
	m.exitConfirmKey = ""
	if m.errLine == hint {
		m.errLine = ""
	}
}
