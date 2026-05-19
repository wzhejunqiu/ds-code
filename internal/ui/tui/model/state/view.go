package state

import (
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/hejunqiu/ds-code/internal/ui/tui/subagent"
)

// BindMainChat sets the primary session transcript (main agent view).
func (s *State) BindMainChat(chat []chat.Block) {
	s.MainChat = chat
	s.MainToolLines = nil
	if s.SubagentNav == SubagentNavMain || s.SubagentNav == SubagentNavList {
		s.Chat = s.MainChat
		s.ToolLines = s.MainToolLines
	}
}

// SyncDisplayedChat copies the active view (main or selected subagent) into Chat/ToolLines.
func (s *State) SyncDisplayedChat() {
	switch s.SubagentNav {
	case SubagentNavDetail:
		if rec := s.Subagents.Get(s.ViewingSubagentID); rec != nil {
			s.Chat = rec.Chat
			s.ToolLines = rec.ToolLines
			return
		}
		s.SubagentNav = SubagentNavMain
		fallthrough
	default:
		s.Chat = s.MainChat
		s.ToolLines = s.MainToolLines
	}
}

// ViewingSubagent returns the record when in detail navigation.
func (s *State) ViewingSubagent() *subagent.Record {
	if s.SubagentNav != SubagentNavDetail {
		return nil
	}
	return s.Subagents.Get(s.ViewingSubagentID)
}
