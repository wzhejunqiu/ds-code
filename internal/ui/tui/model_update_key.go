package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *model) updateKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !isExitConfirmKey(msg.String()) {
		m.clearExitConfirm()
	}
	if m.overlay == overlayResume {
		// Resume picker: Enter loads session; navigation keys go to handleResumeKey.
		if msg.Type == tea.KeyEnter && !msg.Alt {
			if len(m.resumeSessions) > 0 {
				idx := m.resumePicker.Cursor
				if idx >= len(m.resumeSessions) {
					idx = len(m.resumeSessions) - 1
				}
				if idx < 0 {
					idx = 0
				}
				id := m.resumeSessions[idx].ID
				m.input.Reset()
				m.clearResumePicker()
				return m.resumeSession(id), true
			}
			// Picker open with no matches — do not treat filter text as a session id.
			return nil, true
		}
		if m.handleResumeKey(msg) {
			return nil, true
		}
	}
	if m.overlay == overlayPrompt && m.prompt != nil {
		if m.running && msg.String() == "esc" {
			m.requestCancelTurn()
			return m.listenPrompt(), true
		}
		model, cmd := m.handlePromptKey(msg)
		_ = model
		return cmd, true
	}
	if m.overlay == overlayComplete {
		if m.handleCompleteKey(msg) {
			return nil, true
		}
	}
	if m.overlay == overlayContext || m.overlay == overlayHelp {
		switch msg.String() {
		case "esc", "q":
			m.dismissOverlay()
			return nil, true
		}
	}
	switch msg.String() {
	case "ctrl+c", "ctrl+d":
		_, cmd := m.handleExitKey(msg.String())
		return cmd, true
	case "ctrl+r":
		m.reasoningAll = !m.reasoningAll
		for i := range m.chat {
			if m.chat[i].role == chatRoleAssistant {
				m.chat[i].reasoningOpen = m.reasoningAll
			}
		}
		m.syncChatView()
		return nil, true
	case "?":
		return m.showHelpOverlay(), true
	case "ctrl+l":
		return m.showContextOverlay(), true
	case "ctrl+t":
		m.toolOpen = !m.toolOpen
		m.layout()
		return nil, true
	case "ctrl+o":
		m.toolDetailsVisible = !m.toolDetailsVisible
		m.syncChatView()
		return nil, true
	case "esc":
		if m.running {
			if m.overlay != overlayNone {
				m.dismissOverlay()
				return nil, true
			}
			m.requestCancelTurn()
			return nil, true
		}
		if m.overlay != overlayNone {
			m.dismissOverlay()
			return nil, true
		}
	}
	return nil, false
}
