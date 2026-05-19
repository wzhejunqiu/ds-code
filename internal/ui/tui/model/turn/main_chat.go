package turn

import "github.com/hejunqiu/ds-code/internal/ui/tui/model/state"

// withMainChat runs fn while Chat/ToolLines refer to the main session transcript.
func withMainChat(s *state.State, fn func()) {
	if s.MainChat == nil {
		s.MainChat = s.Chat
	}
	s.Chat, s.ToolLines = s.MainChat, s.MainToolLines
	fn()
	s.MainChat, s.MainToolLines = s.Chat, s.ToolLines
	s.SyncDisplayedChat()
}
