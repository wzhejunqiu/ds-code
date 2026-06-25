package input

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/turn"
)

// TryAutoResumeTurn starts a notification-driven turn when the TUI is idle.
func TryAutoResumeTurn(s *state.State, syncChat, syncTool func()) tea.Cmd {
	if !autoResumeAllowed(s) {
		return nil
	}
	if !needsNotificationResume(s) {
		return nil
	}
	return SubmitAutoResume(s, syncChat, syncTool)
}

func autoResumeAllowed(s *state.State) bool {
	if s == nil || s.Deps == nil || s.Deps.Runner == nil {
		return false
	}
	if s.Running || s.Prompt != nil || s.Overlay != state.OverlayNone {
		return false
	}
	if s.Deps.BackgroundAgents != nil && s.Deps.BackgroundAgents() > 0 {
		return false
	}
	return true
}

func needsNotificationResume(s *state.State) bool {
	if s.Deps.HasPendingNotifications != nil && s.Deps.HasPendingNotifications() {
		return true
	}
	if s.Deps.Store == nil || s.SessionID == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	msgs, err := s.Deps.Store.ListMessages(ctx, s.SessionID)
	if err != nil || len(msgs) == 0 {
		return false
	}
	last := msgs[len(msgs)-1]
	return spawn.SessionAwaitingNotificationResume(last.Content, string(last.Role))
}

// SubmitAutoResume runs a turn to consume pending async agent notifications.
func SubmitAutoResume(s *state.State, syncChat, syncTool func()) tea.Cmd {
	if s.MainChat == nil {
		s.MainChat = s.Chat
	}
	s.BindMainChat(s.MainChat)
	turn.AppendPlanningBlock(s)
	syncChat()
	s.Running = true
	s.TurnEscPending = false
	s.ErrLine = ""
	s.MainToolLines = nil
	s.ToolLines = nil
	syncTool()

	d := *s.Deps
	d.SessionID = s.SessionID
	events := s.Deps.Events
	return func() tea.Msg {
		go turn.RunAsync(d, "", events, &s.TurnWG)
		return nil
	}
}
