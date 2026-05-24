package overlay

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/turn"
)

type KeyDeps struct {
	HandleResumeEnter   func() (tea.Cmd, bool)
	HandleResumeKey     func(tea.KeyMsg) bool
	HandleCompleteKey   func(tea.KeyMsg) bool
	HandleTCaseEnter    func() (tea.Cmd, bool)
	HandleTCaseKey      func(tea.KeyMsg) bool
	HandlePromptKey     func(tea.KeyMsg) tea.Cmd
	ListenPrompt        func() tea.Cmd
	RequestCancelTurn   func()
	ShowHelp            func() tea.Cmd
	ShowContext         func() tea.Cmd
	SyncChat            func()
	ExitTimeout         func() tea.Cmd
}

func HandleKey(s *state.State, msg tea.KeyMsg, d KeyDeps) (tea.Cmd, bool) {
	if !IsExitConfirmKey(msg.String()) {
		ClearExitConfirm(s)
	}
	if s.Overlay == state.OverlayResume {
		if msg.Type == tea.KeyEnter && !msg.Alt {
			if cmd, ok := d.HandleResumeEnter(); ok {
				return cmd, true
			}
		}
		if d.HandleResumeKey(msg) {
			return nil, true
		}
	}
	if s.Overlay == state.OverlayPrompt && s.Prompt != nil {
		if s.Running && msg.String() == "esc" {
			d.RequestCancelTurn()
			return d.ListenPrompt(), true
		}
		return d.HandlePromptKey(msg), true
	}
	if s.Overlay == state.OverlayComplete {
		if d.HandleCompleteKey(msg) {
			return nil, true
		}
	}
	if s.Overlay == state.OverlayTCase {
		if msg.Type == tea.KeyEnter && !msg.Alt && d.HandleTCaseEnter != nil {
			if cmd, ok := d.HandleTCaseEnter(); ok {
				return cmd, true
			}
		}
		if d.HandleTCaseKey != nil && d.HandleTCaseKey(msg) {
			return nil, true
		}
	}
	if s.Overlay == state.OverlayContext || s.Overlay == state.OverlayHelp {
		switch msg.String() {
		case "esc", "q":
			Dismiss(s)
			return nil, true
		}
	}
	switch msg.String() {
	case "ctrl+c", "ctrl+d":
		return HandleExitKey(s, msg.String(), d.ExitTimeout), true
	case "ctrl+r":
		s.ReasoningAll = !s.ReasoningAll
		for i := range s.Chat {
			if s.Chat[i].Role == chat.RoleAssistant {
				s.Chat[i].ReasoningOpen = s.ReasoningAll
			}
		}
		d.SyncChat()
		return nil, true
	case "?":
		return d.ShowHelp(), true
	case "ctrl+l":
		return d.ShowContext(), true
	case "ctrl+t":
		s.ToolOpen = !s.ToolOpen
		return nil, true
	case "ctrl+o":
		s.ToolDetailsVisible = !s.ToolDetailsVisible
		d.SyncChat()
		return nil, true
	case "esc":
		if s.SubagentNav == state.SubagentNavDetail || s.SubagentNav == state.SubagentNavList {
			return nil, false
		}
		if s.ErrLine != "" && strings.HasPrefix(s.ErrLine, "TUI ") {
			s.ErrLine = ""
			return nil, true
		}
		if s.Running {
			if s.Overlay != state.OverlayNone {
				Dismiss(s)
				return nil, true
			}
			d.RequestCancelTurn()
			return nil, true
		}
		if s.Overlay != state.OverlayNone {
			Dismiss(s)
			return nil, true
		}
	}
	return nil, false
}

// HandlePromptKey wraps turn.HandlePromptKey for overlay KeyDeps wiring.
func HandlePromptKey(s *state.State, msg tea.KeyMsg, listenPrompt func() tea.Cmd) tea.Cmd {
	return turn.HandlePromptKey(s, msg.String(), listenPrompt)
}
