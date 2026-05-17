// Turn cancel/interrupt and permission prompt handling.
package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
)

// requestCancelTurn handles Esc: cancels immediately when possible, otherwise
// records intent and shows the interrupt marker until turnStartedMsg arrives.
func (m *model) requestCancelTurn() {
	if m.turnCancel != nil {
		m.turnEscPending = false
		m.cancelTurn()
		return
	}
	m.turnEscPending = true
	m.appendInterruptBlock()
}

// cancelTurn stops the in-flight agent turn (Esc). An interrupt marker is
// appended to the chat until turnDoneMsg arrives.
func (m *model) cancelTurn() {
	if m.turnCancel != nil {
		m.turnCancel()
	}
	m.dismissPrompt()
	m.appendInterruptBlock()
}

func (m *model) dismissPrompt() {
	if m.prompt == nil {
		return
	}
	select {
	case m.prompt.Reply <- false:
	default:
	}
	m.prompt = nil
	m.overlay = overlayNone
	m.overlayText = ""
}

func (m *model) appendInterruptBlock() {
	if m.currentTurnInterrupted() {
		return
	}
	now := time.Now()
	m.finalizeLastAssistant(now)
	m.clearPlanningBlock()
	for i := range m.chat {
		if m.chat[i].Role == chat.RoleTool && m.chat[i].ToolRunning {
			m.chat[i].ToolRunning = false
		}
	}
	m.chat = append(m.chat, chat.Block{Role: chat.RoleInterrupt})
	m.syncChatView()
}

// turnEventsAllowed gates stream/tool callbacks after Esc (interrupt marker present).
func (m *model) turnEventsAllowed() bool {
	return m.running && !m.currentTurnInterrupted()
}

// currentTurnInterrupted is true if an interrupt block exists after the latest user message.
func (m *model) currentTurnInterrupted() bool {
	lastUser := -1
	for i, b := range m.chat {
		if b.Role == chat.RoleUser {
			lastUser = i
		}
	}
	if lastUser < 0 {
		return false
	}
	for i := lastUser + 1; i < len(m.chat); i++ {
		if m.chat[i].Role == chat.RoleInterrupt {
			return true
		}
	}
	return false
}

func (m *model) persistTurnInterrupt() {
	if m.deps == nil || m.deps.Store == nil || m.sessionID == "" {
		return
	}
	_ = m.deps.Store.AppendMessage(context.Background(), session.Message{
		SessionID: m.sessionID,
		Role:      role.System,
		Content:   chat.InterruptSessionMarker(),
	})
}

func (m *model) replyPrompt(allow bool) {
	if m.prompt == nil {
		return
	}
	select {
	case m.prompt.Reply <- allow:
	default:
	}
	m.prompt = nil
	m.overlay = overlayNone
	m.overlayText = ""
}

func (m *model) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y", "yes":
		m.replyPrompt(true)
	case "n", "no", "esc":
		m.replyPrompt(false)
	}
	return m, m.listenPrompt()
}
