package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) updateWindowSize(msg tea.WindowSizeMsg) tea.Cmd {
	m.width = msg.Width
	m.height = msg.Height
	m.syncChatView()
	m.syncToolView()
	if m.overlay == overlayResume && len(m.resumeSessions) > 0 {
		m.syncResumePicker()
	}
	return nil
}

func (m *model) updateContextOverlay(msg contextOverlayMsg) tea.Cmd {
	m.overlay = overlayContext
	m.overlayText = msg.text
	return nil
}

func (m *model) updateHelpOverlay(msg helpOverlayMsg) tea.Cmd {
	m.overlay = overlayHelp
	m.overlayText = msg.text
	return nil
}

func (m *model) updateStatusRefresh() tea.Cmd {
	m.refreshStatus()
	m.syncChatView()
	return statusTick()
}

func (m *model) updateExitConfirmTimeout() tea.Cmd {
	if m.exitConfirmPending && time.Since(m.exitConfirmArmedAt) >= exitConfirmTimeout {
		m.clearExitConfirm()
	}
	if m.errLine == runningTurnHint {
		m.errLine = ""
	}
	return nil
}

func (m *model) updatePromptRequest(msg promptRequestMsg) tea.Cmd {
	m.prompt = &msg.req
	m.overlay = overlayPrompt
	m.overlayText = fmt.Sprintf("Allow %s?\n%s\n[y] yes  [n] no", msg.req.Tool, truncate(msg.req.Summary, 300))
	return m.listenPrompt()
}

func (m *model) updateOverlayClose() tea.Cmd {
	m.dismissOverlay()
	m.refreshStatus()
	m.syncChatView()
	return nil
}

// dismissOverlay closes the active overlay and clears any associated picker state.
func (m *model) dismissOverlay() {
	switch m.overlay {
	case overlayComplete:
		m.overlay = overlayNone
		m.overlayText = ""
		m.clearCompletePicker()
	case overlayResume:
		m.clearResumePicker()
	case overlayPrompt:
		m.dismissPrompt()
	case overlayContext, overlayHelp:
		m.overlay = overlayNone
		m.overlayText = ""
	default:
		if m.overlay != overlayNone {
			m.overlay = overlayNone
			m.overlayText = ""
		}
	}
}
