package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles Bubble Tea messages for the root model.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, m.updateWindowSize(msg)
	case tea.KeyMsg:
		if cmd, handled := m.updateKey(msg); handled {
			return m, cmd
		}
		return m.updateInput(msg)
	case streamContentMsg:
		return m, m.updateStreamContent(msg)
	case streamReasoningMsg:
		return m, m.updateStreamReasoning(msg)
	case planningStartMsg:
		return m, m.updatePlanningStart()
	case planningEndMsg:
		return m, m.updatePlanningEnd()
	case thinkingTickMsg:
		return m, m.updateThinkingTick()
	case slashOutputMsg:
		return m, m.updateSlashOutput(msg)
	case resumeListMsg:
		return m, m.updateResumeList(msg)
	case sessionResumedMsg:
		return m, m.updateSessionResumed(msg)
	case historyLoadedMsg:
		return m, m.updateHistoryLoaded(msg)
	case toolStartMsg:
		return m, m.updateToolStart(msg)
	case assistantSegmentEndMsg:
		return m, m.updateAssistantSegmentEnd()
	case toolEndMsg:
		return m, m.updateToolEnd(msg)
	case turnStartedMsg:
		return m, m.updateTurnStarted(msg)
	case contextOverlayMsg:
		return m, m.updateContextOverlay(msg)
	case helpOverlayMsg:
		return m, m.updateHelpOverlay(msg)
	case turnDoneMsg:
		return m, m.updateTurnDone(msg)
	case statusRefreshMsg:
		return m, m.updateStatusRefresh()
	case exitConfirmTimeoutMsg:
		return m, m.updateExitConfirmTimeout()
	case promptRequestMsg:
		return m, m.updatePromptRequest(msg)
	case overlayCloseMsg:
		return m, m.updateOverlayClose()
	default:
		return m.updateInput(msg)
	}
}
