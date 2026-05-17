package turn

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
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
	now := time.Now()
	FinalizeLastAssistant(s, now)
	ClearPlanningBlock(s)
	for i := range s.Chat {
		if s.Chat[i].Role == chat.RoleTool && s.Chat[i].ToolRunning {
			s.Chat[i].ToolRunning = false
		}
	}
	s.Chat = append(s.Chat, chat.Block{Role: chat.RoleInterrupt})
	syncView()
}

func EventsAllowed(s *state.State) bool {
	return s.Running && !CurrentTurnInterrupted(s)
}

func CurrentTurnInterrupted(s *state.State) bool {
	lastUser := -1
	for i, b := range s.Chat {
		if b.Role == chat.RoleUser {
			lastUser = i
		}
	}
	if lastUser < 0 {
		return false
	}
	for i := lastUser + 1; i < len(s.Chat); i++ {
		if s.Chat[i].Role == chat.RoleInterrupt {
			return true
		}
	}
	return false
}

func PersistInterrupt(s *state.State) {
	if s.Deps == nil || s.Deps.Store == nil || s.SessionID == "" {
		return
	}
	_ = s.Deps.Store.AppendMessage(context.Background(), session.Message{
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
