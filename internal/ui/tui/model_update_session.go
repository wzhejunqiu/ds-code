package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/session"
)

func (m *model) updateSlashOutput(msg slashOutputMsg) tea.Cmd {
	if msg.text != "" {
		m.chat = append(m.chat, chatBlock{role: chatRoleAssistant})
		m.chat[len(m.chat)-1].content.WriteString(msg.text)
	}
	m.refreshStatus()
	m.syncChatView()
	return nil
}

func (m *model) updateResumeList(msg resumeListMsg) tea.Cmd {
	if msg.seq != 0 && msg.seq != m.resumeFilterSeq {
		return nil
	}
	if msg.err != nil {
		m.errLine = msg.err.Error()
		m.clearResumePicker()
		return nil
	}
	if msg.filter == "" && len(msg.sessions) == 0 {
		m.errLine = "No saved sessions."
		m.clearResumePicker()
		return nil
	}
	m.applyResumeSessions(msg.filter, msg.sessions)
	return nil
}

// updateSessionResumed switches session id and transcript after /resume selection.
func (m *model) updateSessionResumed(msg sessionResumedMsg) tea.Cmd {
	m.resumePending = false
	if msg.err != nil {
		m.errLine = msg.err.Error()
		return nil
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
	return nil
}

// updateHistoryLoaded replaces m.chat when the current session history finishes loading.
func (m *model) updateHistoryLoaded(msg historyLoadedMsg) tea.Cmd {
	if msg.err != nil {
		m.errLine = msg.err.Error()
		return nil
	}
	if len(msg.chat) > 0 {
		m.chat = msg.chat
		m.syncChatView()
	}
	return nil
}
