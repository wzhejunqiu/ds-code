package subagent

import (
	"charm.land/bubbletea/v2"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/turn"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/subagent"
)

type SyncFn func()

func eventsAllowed(s *state.State, id string) bool {
	if turn.CurrentTurnInterrupted(s) {
		return false
	}
	if s.Running {
		return true
	}
	rec := s.Subagents.Get(id)
	return rec != nil && rec.Status == subagent.StatusRunning
}

func UpdateStart(s *state.State, m msg.SubagentStartMsg, sync SyncFn) tea.Cmd {
	if turn.CurrentTurnInterrupted(s) {
		return nil
	}
	s.Subagents.Start(m.ID, m.Label, m.Prompt, m.AgentType, m.Background)
	if s.SubagentNav == state.SubagentNavDetail && s.ViewingSubagentID == m.ID {
		s.SyncDisplayedChat()
	}
	sync()
	return nil
}

func UpdateEnd(s *state.State, m msg.SubagentEndMsg, sync SyncFn) tea.Cmd {
	if !eventsAllowed(s, m.ID) {
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
	if !eventsAllowed(s, m.SubagentID) {
		return nil
	}
	disp := tool.FromRegistry(s.Deps.Runner.Tools)
	s.Subagents.ToolStart(m.SubagentID, m.Name, m.Args, m.Command, time.Time{}, disp)
	if s.SubagentNav == state.SubagentNavDetail && s.ViewingSubagentID == m.SubagentID {
		s.SyncDisplayedChat()
		sync()
	}
	return nil
}

func UpdateToolEnd(s *state.State, m msg.SubagentToolEndMsg, sync SyncFn) tea.Cmd {
	if !eventsAllowed(s, m.SubagentID) {
		return nil
	}
	disp := tool.FromRegistry(s.Deps.Runner.Tools)
	s.Subagents.ToolEnd(m.SubagentID, m.Name, m.Args, m.Command, m.Result, m.IsError, disp)
	if s.SubagentNav == state.SubagentNavDetail && s.ViewingSubagentID == m.SubagentID {
		s.SyncDisplayedChat()
		sync()
	}
	return nil
}
