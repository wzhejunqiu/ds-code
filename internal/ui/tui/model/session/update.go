package session

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/component"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/subagent"
)

func UpdateSlashOutput(s *state.State, m msg.SlashOutputMsg, syncChat func()) tea.Cmd {
	if m.Text != "" {
		s.Chat = append(s.Chat, chat.Block{Role: chat.RoleAssistant})
		s.Chat[len(s.Chat)-1].AppendContent(m.Text)
	}
	syncChat()
	return nil
}

func UpdateResumeList(s *state.State, m msg.ResumeListMsg, picker *component.Picker) tea.Cmd {
	if m.Seq != 0 && m.Seq != s.ResumeFilterSeq {
		return nil
	}
	if m.Err != nil {
		s.ErrLine = m.Err.Error()
		ClearResumePicker(s, picker)
		return nil
	}
	if m.Filter == "" && len(m.Sessions) == 0 {
		s.ErrLine = "No saved sessions."
		ClearResumePicker(s, picker)
		return nil
	}
	ApplyResumeSessions(s, m.Filter, m.Sessions, picker)
	return nil
}

func UpdateSessionResumed(s *state.State, m msg.SessionResumedMsg, picker *component.Picker, syncChat, syncTool func(), refreshStatus func()) tea.Cmd {
	s.ResumePending = false
	if m.Err != nil {
		s.ErrLine = m.Err.Error()
		return nil
	}
	if s.Deps.OnSessionEnd != nil && s.SessionID != "" {
		s.Deps.OnSessionEnd(s.SessionID)
	}
	session.DropPending(s.Deps.Store, s.SessionID)
	s.SessionID = m.SessionID
	s.Deps.SessionID = m.SessionID
	s.BindMainChat(m.Chat)
	s.ToolLines = nil
	if m.Subagents.Len() > 0 {
		s.Subagents = m.Subagents
	} else {
		s.Subagents = subagent.Registry{}
	}
	s.SubagentNav = state.SubagentNavMain
	s.ViewingSubagentID = ""
	ClearResumePicker(s, picker)
	s.ErrLine = ""
	syncChat()
	syncTool()
	if refreshStatus != nil {
		refreshStatus()
	}
	return nil
}

func UpdateHistoryLoaded(s *state.State, m msg.HistoryLoadedMsg, syncChat func(), refreshStatus func()) tea.Cmd {
	if m.Err != nil {
		s.ErrLine = m.Err.Error()
		return nil
	}
	if len(m.Chat) > 0 {
		s.BindMainChat(m.Chat)
		syncChat()
	}
	if m.Subagents.Len() > 0 {
		s.Subagents = m.Subagents
	}
	if refreshStatus != nil {
		refreshStatus()
	}
	return nil
}
