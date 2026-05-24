package subagent

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/turn"
)

type SyncFn func()

func UpdateStart(s *state.State, m msg.SubagentStartMsg, sync SyncFn) tea.Cmd {
	if !turn.EventsAllowed(s) {
		return nil
	}
	s.Subagents.Start(m.ID, m.Label, m.Prompt)
	if s.SubagentNav == state.SubagentNavDetail && s.ViewingSubagentID == m.ID {
		s.SyncDisplayedChat()
	}
	sync()
	return nil
}

func UpdateEnd(s *state.State, m msg.SubagentEndMsg, sync SyncFn) tea.Cmd {
	if !turn.EventsAllowed(s) {
		return nil
	}
	s.Subagents.End(m.ID, m.Summary, m.Err)
	if s.SubagentNav == state.SubagentNavDetail && s.ViewingSubagentID == m.ID {
		s.SyncDisplayedChat()
		sync()
	}
	return nil
}

func UpdateToolStart(s *state.State, m msg.SubagentToolStartMsg, sync SyncFn) tea.Cmd {
	if !turn.EventsAllowed(s) {
		return nil
	}
	s.Subagents.ToolStart(m.SubagentID, m.Name, m.Args, m.Command)
	if s.SubagentNav == state.SubagentNavDetail && s.ViewingSubagentID == m.SubagentID {
		s.SyncDisplayedChat()
		sync()
	}
	return nil
}

func UpdateToolEnd(s *state.State, m msg.SubagentToolEndMsg, sync SyncFn) tea.Cmd {
	if !turn.EventsAllowed(s) {
		return nil
	}
	s.Subagents.ToolEnd(m.SubagentID, m.Name, m.Args, m.Command, m.Result, m.IsError)
	if s.SubagentNav == state.SubagentNavDetail && s.ViewingSubagentID == m.SubagentID {
		s.SyncDisplayedChat()
		sync()
	}
	return nil
}
