package turn

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

func RequestCancel(s *state.State, syncView func()) {
	if s.TurnCancel != nil {
		s.TurnEscPending = false
		Cancel(s, syncView)
		return
	}
	s.TurnEscPending = true
	AppendInterruptBlock(s, syncView)
}

func Cancel(s *state.State, syncView func()) {
	if s.TurnCancel != nil {
		s.TurnCancel()
	}
	DismissPrompt(s)
	AppendInterruptBlock(s, syncView)
}

func DismissPrompt(s *state.State) {
	if s.Prompt == nil {
		return
	}
	select {
	case s.Prompt.Reply <- false:
	default:
	}
	s.Prompt = nil
	s.Overlay = state.OverlayNone
	s.OverlayText = ""
}

func AppendInterruptBlock(s *state.State, syncView func()) {
	if CurrentTurnInterrupted(s) {
		return
	}
	withMainChat(s, func() {
		now := time.Now()
		FinalizeLastAssistant(s, now)
		ClearPlanningBlock(s)
		for i := range s.Chat {
			if s.Chat[i].Role == chat.RoleTool && s.Chat[i].ToolRunning {
				s.Chat[i].ToolRunning = false
			}
		}
		s.Chat = append(s.Chat, chat.Block{Role: chat.RoleInterrupt})
	})
	syncView()
}

func EventsAllowed(s *state.State) bool {
	return s.Running && !CurrentTurnInterrupted(s)
}

// mainTranscript returns the primary session blocks (interrupt markers live here).
func mainTranscript(s *state.State) []chat.Block {
	if s.MainChat != nil {
		return s.MainChat
	}
	return s.Chat
}

func CurrentTurnInterrupted(s *state.State) bool {
	blocks := mainTranscript(s)
	lastUser := -1
	for i, b := range blocks {
		if b.Role == chat.RoleUser {
			lastUser = i
		}
	}
	if lastUser < 0 {
		return false
	}
	for i := lastUser + 1; i < len(blocks); i++ {
		if blocks[i].Role == chat.RoleInterrupt {
			return true
		}
	}
	return false
}

func PersistInterrupt(s *state.State) {
	if s.Deps == nil || s.Deps.Store == nil || s.SessionID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Deps.Store.AppendMessage(ctx, session.Message{
		SessionID: s.SessionID,
		Role:      role.System,
		Content:   chat.InterruptSessionMarker(),
	})
}

func ReplyPrompt(s *state.State, allow bool) {
	if s.Prompt == nil {
		return
	}
	select {
	case s.Prompt.Reply <- allow:
	default:
	}
	s.Prompt = nil
	s.Overlay = state.OverlayNone
	s.OverlayText = ""
}

func HandlePromptKey(s *state.State, key string, listenPrompt func() tea.Cmd) tea.Cmd {
	switch strings.ToLower(key) {
	case "y", "yes":
		ReplyPrompt(s, true)
	case "n", "no", "esc":
		ReplyPrompt(s, false)
	}
	return listenPrompt()
}
