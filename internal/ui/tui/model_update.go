package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/session"
)

// Update handles Bubble Tea messages for the root model.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncChatView()
		m.syncToolView()
		if m.overlay == overlayResume && len(m.resumeSessions) > 0 {
			m.renderResumeOverlay()
		}
		return m, nil

	case tea.KeyMsg:
		if !isExitConfirmKey(msg.String()) {
			m.clearExitConfirm()
		}
		if m.overlay == overlayResume {
			if msg.Type == tea.KeyEnter && !msg.Alt {
				if len(m.resumeSessions) > 0 {
					idx := m.resumeIdx
					if idx >= len(m.resumeSessions) {
						idx = len(m.resumeSessions) - 1
					}
					if idx < 0 {
						idx = 0
					}
					id := m.resumeSessions[idx].ID
					m.input.Reset()
					m.clearResumePicker()
					return m, m.resumeSession(id)
				}
				// Picker open with no matches — do not treat filter text as a session id.
				return m, nil
			}
			if m.handleResumeKey(msg) {
				return m, nil
			}
		}
		if m.overlay == overlayPrompt && m.prompt != nil {
			if m.running && msg.String() == "esc" {
				m.requestCancelTurn()
				return m, m.listenPrompt()
			}
			return m.handlePromptKey(msg)
		}
		if m.overlay == overlayComplete {
			if m.handleCompleteKey(msg) {
				return m, nil
			}
		}
		if m.overlay == overlayContext || m.overlay == overlayHelp {
			switch msg.String() {
			case "esc", "q":
				m.overlay = overlayNone
				m.overlayText = ""
				return m, nil
			}
		}
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			return m.handleExitKey(msg.String())
		case "ctrl+r":
			m.reasoningAll = !m.reasoningAll
			for i := range m.chat {
				if m.chat[i].role == chatRoleAssistant {
					m.chat[i].reasoningOpen = m.reasoningAll
				}
			}
			m.syncChatView()
			return m, nil
		case "?":
			return m, m.showHelpOverlay()
		case "ctrl+l":
			return m, m.showContextOverlay()
		case "ctrl+t":
			m.toolOpen = !m.toolOpen
			m.layout()
			return m, nil
		case "ctrl+o":
			m.toolDetailsVisible = !m.toolDetailsVisible
			m.syncChatView()
			return m, nil
		case "esc":
			if m.running {
				if m.overlay != overlayNone {
					m.overlay = overlayNone
					m.overlayText = ""
					return m, nil
				}
				m.requestCancelTurn()
				return m, nil
			}
			if m.overlay != overlayNone {
				m.overlay = overlayNone
				m.overlayText = ""
				return m, nil
			}
		}

	case streamContentMsg:
		if m.turnEventsAllowed() {
			m.clearPlanningBlock()
			m.appendAssistantContent(msg.delta)
			m.syncChatView()
		}
		return m, nil
	case streamReasoningMsg:
		if m.turnEventsAllowed() {
			m.clearPlanningBlock()
			var cmd tea.Cmd
			if m.appendAssistantReasoning(msg.delta) {
				cmd = m.nextThinkingTickCmd()
			}
			m.syncChatView()
			return m, cmd
		}
		return m, nil
	case planningStartMsg:
		if m.turnEventsAllowed() {
			m.appendPlanningBlock()
			m.syncChatView()
			return m, m.nextThinkingTickCmd()
		}
		return m, nil
	case planningEndMsg:
		m.clearPlanningBlock()
		m.syncChatView()
		return m, nil
	case thinkingTickMsg:
		if m.needsThinkingTick() || m.needsPlanningTick() {
			m.syncChatView()
			return m, m.nextThinkingTickCmd()
		}
		return m, nil
	case slashOutputMsg:
		if msg.text != "" {
			m.chat = append(m.chat, chatBlock{role: chatRoleAssistant})
			m.chat[len(m.chat)-1].content.WriteString(msg.text)
		}
		m.refreshStatus()
		m.syncChatView()
		return m, nil

	case resumeListMsg:
		if msg.err != nil {
			m.errLine = msg.err.Error()
			m.overlay = overlayNone
			return m, nil
		}
		m.resumeSessions = msg.sessions
		m.resumeFilter = ""
		m.resumeIdx = 0
		m.resumeScroll = 0
		if len(m.resumeSessions) == 0 {
			m.errLine = "No saved sessions."
			m.overlay = overlayNone
			return m, nil
		}
		m.overlay = overlayResume
		m.renderResumeOverlay()
		return m, nil

	case sessionResumedMsg:
		if msg.err != nil {
			m.errLine = msg.err.Error()
			return m, nil
		}
		session.DropPending(m.deps.Store, m.sessionID)
		m.sessionID = msg.sessionID
		m.deps.SessionID = msg.sessionID
		m.chat = msg.chat
		m.toolLines = nil
		m.clearResumePicker()
		m.errLine = ""
		m.refreshStatus()
		m.syncChatView()
		m.syncToolView()
		return m, nil

	case historyLoadedMsg:
		if msg.err != nil {
			m.errLine = msg.err.Error()
			return m, nil
		}
		if len(msg.chat) > 0 {
			m.chat = msg.chat
			m.syncChatView()
		}
		return m, nil
	case toolStartMsg:
		if !m.turnEventsAllowed() {
			return m, nil
		}
		m.appendToolBlock(msg.name, msg.args, msg.command, "", true, false)
		m.toolLines = append(m.toolLines, toolLine(msg.name, msg.args, msg.command, "", true, false))
		m.syncChatView()
		m.syncToolView()
		return m, nil
	case assistantSegmentEndMsg:
		if !m.turnEventsAllowed() {
			return m, nil
		}
		m.finalizeLastAssistant(time.Now())
		return m, nil
	case toolEndMsg:
		if !m.turnEventsAllowed() {
			return m, nil
		}
		m.finishToolBlock(msg.name, msg.args, msg.command, msg.result, msg.isError)
		m.toolLines = m.toolLines[:0]
		for _, b := range m.chat {
			if b.role == chatRoleTool {
				preview := b.toolResult
				if preview == "" && b.toolRunning {
					preview = "…"
				}
				m.toolLines = append(m.toolLines, toolLine(b.toolName, b.toolArgs, b.toolCommand, preview, b.toolRunning, b.toolError))
			}
		}
		m.syncChatView()
		m.syncToolView()
		return m, nil
	case turnStartedMsg:
		m.turnCancel = msg.cancel
		if m.turnEscPending {
			m.turnEscPending = false
			m.cancelTurn()
		}
		return m, nil

	case contextOverlayMsg:
		m.overlay = overlayContext
		m.overlayText = msg.text
		return m, nil
	case helpOverlayMsg:
		m.overlay = overlayHelp
		m.overlayText = msg.text
		return m, nil

	case turnDoneMsg:
		m.running = false
		m.turnCancel = nil
		m.turnEscPending = false
		m.clearPlanningBlock()
		now := time.Now()
		m.finalizeLastAssistant(now)
		for i := range m.chat {
			if m.chat[i].role == chatRoleTool && m.chat[i].toolRunning {
				m.chat[i].toolRunning = false
			}
		}
		m.applyTurnMetrics(msg.result)
		if m.currentTurnInterrupted() {
			m.errLine = ""
			m.persistTurnInterrupt()
		} else if msg.err != nil {
			if errors.Is(msg.err, context.Canceled) {
				m.errLine = "turn cancelled"
			} else {
				m.errLine = msg.err.Error()
			}
		} else {
			m.errLine = ""
		}
		m.refreshStatus()
		m.syncChatView()
		return m, tea.Batch(m.listenPrompt(), statusTick())

	case statusRefreshMsg:
		m.refreshStatus()
		m.syncChatView()
		return m, statusTick()

	case exitConfirmTimeoutMsg:
		if m.exitConfirmPending && time.Since(m.exitConfirmArmedAt) >= exitConfirmTimeout {
			m.clearExitConfirm()
		}
		if m.errLine == runningTurnHint {
			m.errLine = ""
		}
		return m, nil

	case promptRequestMsg:
		m.prompt = &msg.req
		m.overlay = overlayPrompt
		m.overlayText = fmt.Sprintf("Allow %s?\n%s\n[y] yes  [n] no", msg.req.Tool, truncate(msg.req.Summary, 300))
		return m, m.listenPrompt()

	case overlayCloseMsg:
		m.overlay = overlayNone
		m.overlayText = ""
		m.refreshStatus()
		m.syncChatView()
		return m, nil
	}

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
