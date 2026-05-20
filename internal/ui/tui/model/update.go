package model

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/input"
	tuimsg "github.com/hejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/overlay"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/session"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/tcase"
	subagentui "github.com/hejunqiu/ds-code/internal/ui/tui/model/subagent"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/turn"
)

// Update handles Bubble Tea messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	debugBeforeUpdate()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, overlay.OnWindowSize(&m.State, msg.Width, msg.Height, m.syncAllViews)
	case tea.KeyMsg:
		if cmd, handled := m.updateKey(msg); handled {
			return m, cmd
		}
		return m.updateInput(msg)
	case tuimsg.StreamContentMsg:
		return m, turn.UpdateStreamContent(&m.State, msg, m.syncChatView)
	case tuimsg.StreamReasoningMsg:
		return m, turn.UpdateStreamReasoning(&m.State, msg, m.syncChatView, m.nextThinkingTickCmd)
	case tuimsg.PlanningStartMsg:
		return m, turn.UpdatePlanningStart(&m.State, m.syncChatView, m.nextThinkingTickCmd)
	case tuimsg.PlanningEndMsg:
		return m, turn.UpdatePlanningEnd(&m.State, m.syncChatView)
	case tuimsg.ThinkingTickMsg:
		return m, turn.UpdateThinkingTick(&m.State, m.syncChatView, m.nextThinkingTickCmd)
	case tuimsg.SlashOutputMsg:
		m.refreshStatus()
		return m, session.UpdateSlashOutput(&m.State, msg, m.syncChatView)
	case tuimsg.ResumeFilterTickMsg:
		return m, session.UpdateResumeFilterTick(&m.State, msg)
	case tuimsg.ResumeListMsg:
		return m, session.UpdateResumeList(&m.State, msg, &m.resumePicker)
	case tuimsg.SessionResumedMsg:
		m.refreshStatus()
		return m, session.UpdateSessionResumed(&m.State, msg, &m.resumePicker, m.syncChatView, m.syncToolView)
	case tuimsg.HistoryLoadedMsg:
		return m, session.UpdateHistoryLoaded(&m.State, msg, m.syncChatView)
	case tuimsg.ToolStartMsg:
		return m, turn.UpdateToolStart(&m.State, msg, m.syncChatView, m.syncToolView)
	case tuimsg.AssistantSegmentEndMsg:
		return m, turn.UpdateAssistantSegmentEnd(&m.State)
	case tuimsg.ToolEndMsg:
		return m, turn.UpdateToolEnd(&m.State, msg, m.syncChatView, m.syncToolView)
	case tuimsg.SubagentStartMsg:
		return m, subagentui.UpdateStart(&m.State, msg, m.syncChatView)
	case tuimsg.SubagentEndMsg:
		return m, subagentui.UpdateEnd(&m.State, msg, m.syncChatView)
	case tuimsg.SubagentToolStartMsg:
		return m, subagentui.UpdateToolStart(&m.State, msg, m.syncChatView)
	case tuimsg.SubagentToolEndMsg:
		return m, subagentui.UpdateToolEnd(&m.State, msg, m.syncChatView)
	case tuimsg.TurnStartedMsg:
		return m, turn.UpdateTurnStarted(&m.State, msg, m.syncChatView)
	case tuimsg.ContextOverlayMsg:
		return m, overlay.UpdateContext(&m.State, msg)
	case tuimsg.HelpOverlayMsg:
		return m, overlay.UpdateHelp(&m.State, msg)
	case tuimsg.TCasePickerMsg:
		return m, tcase.UpdatePicker(&m.State, msg, &m.tcasePicker)
	case tuimsg.TurnDoneMsg:
		return m, turn.UpdateTurnDone(&m.State, msg, m.syncChatView, m.refreshStatus, m.listenPrompt, statusTick)
	case tuimsg.StatusRefreshMsg:
		return m, overlay.UpdateStatusRefresh(&m.State, m.syncChatView, statusTick)
	case tuimsg.ExitConfirmTimeoutMsg:
		return m, overlay.UpdateExitConfirmTimeout(&m.State)
	case tuimsg.PromptRequestMsg:
		return m, overlay.UpdatePromptRequest(&m.State, msg, m.listenPrompt)
	case tuimsg.OverlayCloseMsg:
		return m, overlay.UpdateClose(&m.State, m.syncChatView, m.refreshStatus)
	default:
		return m.updateInput(msg)
	}
}

func (m *Model) updateKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if cmd, handled := subagentui.HandleNavKey(&m.State, msg, &m.subagentPicker, m.syncChatView); handled {
		return cmd, true
	}
	return overlay.HandleKey(&m.State, msg, overlay.KeyDeps{
		HandleResumeEnter: func() (tea.Cmd, bool) {
			if m.ResumePending {
				return nil, true
			}
			if len(m.ResumeSessions) > 0 {
				idx := m.resumePicker.Cursor
				if idx >= len(m.ResumeSessions) {
					idx = len(m.ResumeSessions) - 1
				}
				if idx < 0 {
					idx = 0
				}
				id := m.ResumeSessions[idx].ID
				m.input.Reset()
				session.ClearResumePicker(&m.State, &m.resumePicker)
				m.ResumePending = true
				return session.ResumeSession(&m.State, id), true
			}
			return session.FetchSessions(&m.State, m.ResumeFilter, m.ResumeFilterSeq), true
		},
		HandleResumeKey: func(k tea.KeyMsg) bool {
			return session.HandleResumeKey(&m.State, k, &m.resumePicker)
		},
		HandleCompleteKey: func(k tea.KeyMsg) bool {
			return input.HandleCompleteKey(&m.State, k, &m.completePicker, m.input.Value(), m.input.SetValue, m.input.CursorEnd)
		},
		HandleTCaseEnter: func() (tea.Cmd, bool) {
			return tcase.ConfirmSelection(&m.State, &m.tcasePicker, m.syncChatView, m.syncToolView)
		},
		HandleTCaseKey: func(k tea.KeyMsg) bool {
			return tcase.HandleKey(&m.State, &m.tcasePicker, k)
		},
		HandlePromptKey: func(k tea.KeyMsg) tea.Cmd {
			return turn.HandlePromptKey(&m.State, k.String(), m.listenPrompt)
		},
		ListenPrompt: m.listenPrompt,
		RequestCancelTurn: func() {
			turn.RequestCancel(&m.State, m.syncChatView)
		},
		ShowHelp:     input.ShowHelp,
		ShowContext:  func() tea.Cmd { return input.ShowContext(&m.State) },
		SyncChat:     m.syncChatView,
		ExitTimeout:  exitConfirmTimeoutTick,
	})
}

func (m *Model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.Running {
		var cmd tea.Cmd
		m.chatVP, cmd = m.chatVP.Update(msg)
		return m, cmd
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)
	if c := input.UpdateCompletion(&m.State, m.input.Value(), &m.completePicker, &m.resumePicker); c != nil {
		cmds = append(cmds, c)
	}

	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter && !key.Alt {
		if m.Overlay == state.OverlayResume || m.Overlay == state.OverlayTCase {
			return m, tea.Batch(cmds...)
		}
		line := strings.TrimSpace(m.input.Value())
		if line != "" {
			m.input.Reset()
			m.Overlay = state.OverlayNone
			input.ClearCompletePicker(&m.State, &m.completePicker)
			session.ClearResumePicker(&m.State, &m.resumePicker)
			cmds = append(cmds, input.SubmitLine(&m.State, line, m.syncChatView, m.syncToolView))
			if turn.NeedsPlanningTick(&m.State) {
				cmds = append(cmds, m.nextThinkingTickCmd())
			}
		}
	}

	m.chatVP, cmd = m.chatVP.Update(msg)
	cmds = append(cmds, cmd)
	if m.ToolOpen {
		m.toolVP, cmd = m.toolVP.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}
