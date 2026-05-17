package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.running {
		var cmd tea.Cmd
		m.chatVP, cmd = m.chatVP.Update(msg)
		return m, cmd
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)
	m.updateCompletion()

	if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEnter && !msg.Alt {
		if m.overlay == overlayResume {
			return m, nil
		}
		line := strings.TrimSpace(m.input.Value())
		if line != "" {
			m.input.Reset()
			m.overlay = overlayNone
			m.complete = nil
			m.clearResumePicker()
			return m, m.submitLine(line)
		}
	}

	m.chatVP, cmd = m.chatVP.Update(msg)
	cmds = append(cmds, cmd)
	if m.toolOpen {
		m.toolVP, cmd = m.toolVP.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}
