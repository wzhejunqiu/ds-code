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
	if completionCmd := m.updateCompletion(); completionCmd != nil {
		cmds = append(cmds, completionCmd)
	}

	if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEnter && !msg.Alt {
		// Enter for resume is handled in updateKey (session pick or block empty list).
		if m.overlay == overlayResume {
			return m, nil
		}
		line := strings.TrimSpace(m.input.Value())
		if line != "" {
			m.input.Reset()
			m.overlay = overlayNone
			m.clearCompletePicker()
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
