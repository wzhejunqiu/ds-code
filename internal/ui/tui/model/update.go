package model

import (
	"strings"

	tea "charm.land/bubbletea/v2"
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
		return m, tea.Batch(cmd, m.scheduleNoticeScroll())
	case tea.KeyPressMsg:
		events, passthrough, pending := input.AccumulateLeakedMouseKeys(&m.mouseLeakBuf, msg)
		if pending {
			var cmds []tea.Cmd
			for _, mm := range events {
				_, cmd := m.handleMouse(mm)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			return m, tea.Batch(cmds...)
		}
		if len(events) > 0 {
			var cmds []tea.Cmd
			for _, mm := range events {
				_, cmd := m.handleMouse(mm)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			if passthrough.Text != "" {
				msg = passthrough
				if cmd, handled := m.updateKey(msg); handled {
					return m, tea.Batch(append(cmds, cmd)...)
				}
				updated, cmd := m.updateInput(msg)
				return updated, tea.Batch(append(cmds, cmd)...)
			}
			return m, tea.Batch(cmds...)
		}
		if passthrough.Text != "" {
			msg = passthrough
		}
		if cmd, handled := m.updateKey(msg); handled {
			return m, cmd
		}
		return m.updateInput(msg)
	case tea.PasteMsg:
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
		if turn.NeedsThinkingTick(&m.State) || turn.NeedsPlanningTick(&m.State) || turn.NeedsBashTimeoutTick(&m.State) {
			cmd := turn.UpdateThinkingTick(&m.State, func() {}, m.nextThinkingTickCmd)
			return m, tea.Batch(cmd, m.scheduleSyncChatView())
		}
		return m, nil
	case tuimsg.SlashOutputMsg:
		m.refreshStatus()
		session.UpdateSlashOutput(&m.State, msg, m.syncChatView)
		return m, m.scheduleSyncChatView()
	case tuimsg.ResumeFilterTickMsg:
		return m, session.UpdateResumeFilterTick(&m.State, msg)
	case tuimsg.ResumeListMsg:
		session.UpdateResumeList(&m.State, msg, &m.resumePicker)
		m.refreshLayout()
		return m, nil
	case tuimsg.SessionResumedMsg:
		session.UpdateSessionResumed(&m.State, msg, &m.resumePicker, m.syncChatAfterLoad, m.syncToolView, m.refreshStatus)
		var cmds []tea.Cmd
		cmds = append(cmds, m.scheduleSyncChatView())
		if c := input.UpdateCompletion(&m.State, m.input.Value(), &m.completePicker, &m.resumePicker); c != nil {
			cmds = append(cmds, c)
		}
		m.refreshLayout()
		return m, tea.Batch(cmds...)
	case tuimsg.HistoryLoadedMsg:
		session.UpdateHistoryLoaded(&m.State, msg, m.syncChatAfterLoad, m.refreshStatus)
		return m, m.scheduleSyncChatView()
	case tuimsg.ToolStartMsg:
		turn.UpdateToolStart(&m.State, msg, m.syncChatView, m.syncToolView)
		return m, m.scheduleSyncChatView()
	case tuimsg.AssistantSegmentEndMsg:
		turn.UpdateAssistantSegmentEnd(&m.State)
		return m, m.scheduleSyncChatView()
	case tuimsg.ToolEndMsg:
		turn.UpdateToolEnd(&m.State, msg, m.syncChatView, m.syncToolView)
		return m, m.scheduleSyncChatView()
	case tuimsg.SubagentStartMsg:
		subagentui.UpdateStart(&m.State, msg, m.syncChatView)
		return m, m.scheduleSyncChatView()
	case tuimsg.SubagentEndMsg:
		cmd := subagentui.UpdateEnd(&m.State, msg, m.syncChatView)
		m.refreshStatus()
		m.syncChatView()
		return m, cmd
	case tuimsg.SubagentToolStartMsg:
		subagentui.UpdateToolStart(&m.State, msg, m.syncChatView)
		return m, m.scheduleSyncChatView()
	case tuimsg.SubagentToolEndMsg:
		subagentui.UpdateToolEnd(&m.State, msg, m.syncChatView)
		return m, m.scheduleSyncChatView()
	case tuimsg.BackgroundAgentCompleteMsg:
		cmd := input.TryAutoResumeTurn(&m.State, m.syncChatView, m.syncToolView)
		return m, tea.Batch(cmd, m.scheduleSyncChatView())
	case tuimsg.TurnStartedMsg:
		turn.UpdateTurnStarted(&m.State, msg, m.syncChatView)
		return m, m.scheduleSyncChatView()
	case tuimsg.ContextOverlayMsg:
		return m, overlay.UpdateContext(&m.State, msg)
	case tuimsg.HelpOverlayMsg:
		return m, overlay.UpdateHelp(&m.State, msg)
	case tuimsg.TCasePickerMsg:
		return m, tcase.UpdatePicker(&m.State, msg, &m.tcasePicker)
	case tuimsg.TurnDoneMsg:
		m.mdSegmentCache.Reset()
		turn.UpdateTurnDone(&m.State, msg, m.syncChatView, m.refreshStatus, m.listenPrompt)
		resumeCmd := input.TryAutoResumeTurn(&m.State, m.syncChatView, m.syncToolView)
		return m, tea.Batch(m.scheduleSyncChatView(), resumeCmd)
	case tuimsg.UsageUpdateMsg:
		m.refreshStatus()
		return m, m.scheduleSyncChatView()
	case chatSyncFlushMsg:
		m.chatSyncScheduled = false
		m.syncChatView()
		return m, nil
	case tuimsg.ExitConfirmTimeoutMsg:
		return m, overlay.UpdateExitConfirmTimeout(&m.State)
	case tuimsg.PromptRequestMsg:
		return m, overlay.UpdatePromptRequest(&m.State, msg, m.listenPrompt)
	case tuimsg.WebFetchPromptRequestMsg:
		return m, overlay.UpdateWebFetchPromptRequest(&m.State, msg, m.listenWebFetchPrompt)
	case tuimsg.OverlayCloseMsg:
		overlay.UpdateClose(&m.State, m.syncChatView, m.refreshStatus)
		return m, m.scheduleSyncChatView()
	case copyResultMsg:
		return m.handleCopyResult(msg)
	case copyToastClearMsg:
		return m.handleCopyToastClear()
	case tea.MouseClickMsg, tea.MouseReleaseMsg, tea.MouseMotionMsg, tea.MouseWheelMsg:
		m.updatePlainLines()
		return m.handleMouse(msg)
	default:
		return m.updateInput(msg)
	}
}

func (m *Model) updateKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if cmd, handled := m.handleManualCopyKey(msg); handled {
		return cmd, true
	}
	if cmd, handled := m.handleSelectionKey(msg); handled {
		return cmd, true
	}
	if cmd, handled := subagentui.HandleNavKey(&m.State, msg, &m.subagentPicker, m.syncChatViewResetting); handled {
		return cmd, true
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
		HandleResumeKey: func(k tea.KeyPressMsg) bool {
			return session.HandleResumeKey(&m.State, k, &m.resumePicker)
		},
		HandleCompleteKey: func(k tea.KeyPressMsg) bool {
			return input.HandleCompleteKey(&m.State, k, &m.completePicker, m.input.Value(), m.input.SetValue, m.input.CursorEnd)
		},
		HandleTCaseEnter: func() (tea.Cmd, bool) {
			return tcase.ConfirmSelection(&m.State, &m.tcasePicker, m.syncChatView, m.syncToolView)
		},
		HandleTCaseKey: func(k tea.KeyPressMsg) bool {
			return tcase.HandleKey(&m.State, &m.tcasePicker, k)
		},
		HandlePromptKey: func(k tea.KeyPressMsg) tea.Cmd {
			return turn.HandlePromptKey(&m.State, k.String(), m.listenPrompt)
		},
		HandleWebFetchPromptKey: func(k tea.KeyPressMsg) tea.Cmd {
			return turn.HandleWebFetchPromptKey(&m.State, k.String(), m.listenWebFetchPrompt)
		},
		ListenPrompt:         m.listenPrompt,
		ListenWebFetchPrompt: m.listenWebFetchPrompt,
		RequestCancelTurn: func() {
			turn.RequestCancel(&m.State, m.syncChatView)
		},
		ShowHelp:    input.ShowHelp,
		ShowContext: func() tea.Cmd { return input.ShowContext(&m.State) },
		SyncChat:    m.syncChatView,
		ExitTimeout: exitConfirmTimeoutTick,
	})
	if handled {
		return cmd, true
	}
	return nil, false
}

func (m *Model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
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
	m.refreshLayout()

	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "enter" && !key.Mod.Contains(tea.ModAlt) {
		if m.Overlay == state.OverlayResume || m.Overlay == state.OverlayTCase {
			return m, tea.Batch(cmds...)
		}
		line := strings.TrimSpace(m.input.Value())
		if line != "" {
			m.input.Reset()
			m.Overlay = state.OverlayNone
			input.ClearCompletePicker(&m.State, &m.completePicker)
			session.ClearResumePicker(&m.State, &m.resumePicker)
			m.refreshLayout()
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
