package model

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/input"
	tuimsg "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/overlay"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	subagentui "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/subagent"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/tcase"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/turn"
)

// Update handles Bubble Tea messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	debugBeforeUpdate()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd := overlay.OnWindowSize(&m.State, msg.Width, msg.Height, m.syncAllViews)
		return m, m.withHPSync(tea.Batch(cmd, m.scheduleNoticeScroll()))
	case tea.KeyMsg:
		if cmd, handled := m.updateKey(msg); handled {
			return m, cmd
		}
		return m.updateInput(msg)
	case tuimsg.StreamContentMsg:
		turn.UpdateStreamContent(&m.State, msg, func() {})
		return m, m.scheduleSyncChatView()
	case tuimsg.StreamReasoningMsg:
		cmd := turn.UpdateStreamReasoning(&m.State, msg, func() {}, m.nextThinkingTickCmd)
		return m, tea.Batch(cmd, m.scheduleSyncChatView())
	case tuimsg.PlanningStartMsg:
		cmd := turn.UpdatePlanningStart(&m.State, func() {}, m.nextThinkingTickCmd)
		return m, tea.Batch(cmd, m.scheduleSyncChatView())
	case tuimsg.PlanningEndMsg:
		turn.UpdatePlanningEnd(&m.State, func() {})
		return m, m.scheduleSyncChatView()
	case tuimsg.NoticeScrollTickMsg:
		return m, m.handleNoticeScrollTick()
	case tuimsg.WheelScrollTickMsg:
		return m, m.handleWheelScrollTick()
	case tuimsg.ThinkingTickMsg:
		if turn.NeedsThinkingTick(&m.State) || turn.NeedsPlanningTick(&m.State) {
			cmd := turn.UpdateThinkingTick(&m.State, func() {}, m.nextThinkingTickCmd)
			return m, tea.Batch(cmd, m.scheduleSyncChatView())
		}
		return m, nil
	case tuimsg.SlashOutputMsg:
		m.refreshStatus()
		session.UpdateSlashOutput(&m.State, msg, m.syncChatView)
		return m, m.syncChatViewportHP()
	case tuimsg.ResumeFilterTickMsg:
		return m, session.UpdateResumeFilterTick(&m.State, msg)
	case tuimsg.ResumeListMsg:
		return m, session.UpdateResumeList(&m.State, msg, &m.resumePicker)
	case tuimsg.SessionResumedMsg:
		session.UpdateSessionResumed(&m.State, msg, &m.resumePicker, m.syncChatViewResetting, m.syncToolView, m.refreshStatus)
		return m, m.syncChatViewportHP()
	case tuimsg.HistoryLoadedMsg:
		session.UpdateHistoryLoaded(&m.State, msg, m.syncChatViewResetting, m.refreshStatus)
		return m, m.syncChatViewportHP()
	case tuimsg.ToolStartMsg:
		turn.UpdateToolStart(&m.State, msg, m.syncChatView, m.syncToolView)
		return m, m.syncChatViewportHP()
	case tuimsg.AssistantSegmentEndMsg:
		turn.UpdateAssistantSegmentEnd(&m.State)
		return m, m.scheduleSyncChatView()
	case tuimsg.ToolEndMsg:
		turn.UpdateToolEnd(&m.State, msg, m.syncChatView, m.syncToolView)
		return m, m.syncChatViewportHP()
	case tuimsg.SubagentStartMsg:
		subagentui.UpdateStart(&m.State, msg, m.syncChatView)
		return m, m.syncChatViewportHP()
	case tuimsg.SubagentEndMsg:
		cmd := subagentui.UpdateEnd(&m.State, msg, m.syncChatView)
		m.refreshStatus()
		m.syncChatView()
		return m, m.withHPSync(cmd)
	case tuimsg.SubagentToolStartMsg:
		subagentui.UpdateToolStart(&m.State, msg, m.syncChatView)
		return m, m.syncChatViewportHP()
	case tuimsg.SubagentToolEndMsg:
		subagentui.UpdateToolEnd(&m.State, msg, m.syncChatView)
		return m, m.syncChatViewportHP()
	case tuimsg.TurnStartedMsg:
		turn.UpdateTurnStarted(&m.State, msg, m.syncChatView)
		return m, m.syncChatViewportHP()
	case tuimsg.ContextOverlayMsg:
		return m, overlay.UpdateContext(&m.State, msg)
	case tuimsg.HelpOverlayMsg:
		return m, overlay.UpdateHelp(&m.State, msg)
	case tuimsg.TCasePickerMsg:
		return m, tcase.UpdatePicker(&m.State, msg, &m.tcasePicker)
	case tuimsg.TurnDoneMsg:
		m.mdSegmentCache.Reset()
		turn.UpdateTurnDone(&m.State, msg, m.syncChatView, m.refreshStatus, m.listenPrompt)
		return m, m.syncChatViewportHP()
	case tuimsg.UsageUpdateMsg:
		m.refreshStatus()
		return m, m.scheduleSyncChatView()
	case chatSyncFlushMsg:
		m.chatSyncScheduled = false
		m.syncChatView()
		return m, m.syncChatViewportHP()
	case tuimsg.ExitConfirmTimeoutMsg:
		return m, overlay.UpdateExitConfirmTimeout(&m.State)
	case tuimsg.PromptRequestMsg:
		return m, overlay.UpdatePromptRequest(&m.State, msg, m.listenPrompt)
	case tuimsg.OverlayCloseMsg:
		overlay.UpdateClose(&m.State, m.syncChatView, m.refreshStatus)
		return m, m.syncChatViewportHP()
	case copyResultMsg:
		return m.handleCopyResult(msg)
	case copyToastClearMsg:
		return m.handleCopyToastClear()
	case tea.MouseMsg:
		m.updatePlainLines()
		return m.handleMouse(msg)
	default:
		return m.updateInput(msg)
	}
}

func (m *Model) updateKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if cmd, handled := m.handleManualCopyKey(msg); handled {
		return cmd, true
	}
	if cmd, handled := subagentui.HandleNavKey(&m.State, msg, &m.subagentPicker, m.syncChatViewResetting); handled {
		return m.withHPSync(cmd), true
	}
	cmd, handled := overlay.HandleKey(&m.State, msg, overlay.KeyDeps{
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
		ShowHelp:    input.ShowHelp,
		ShowContext: func() tea.Cmd { return input.ShowContext(&m.State) },
		SyncChat:    m.syncChatView,
		ExitTimeout: exitConfirmTimeoutTick,
	})
	if handled {
		return m.withHPSync(cmd), true
	}
	return nil, false
}

func (m *Model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if cmd, handled := m.handleViewportScrollKey(key); handled {
			return m, cmd
		}
	}

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
