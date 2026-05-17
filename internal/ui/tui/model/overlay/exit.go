package overlay

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/view"
)

func IsExitConfirmKey(s string) bool {
	return s == "ctrl+c" || s == "ctrl+d"
}

func exitConfirmHintFor(key string) string {
	if key == "ctrl+c" {
		return "Press Ctrl+C again to exit"
	}
	return "Press Ctrl+D again to exit"
}

func HandleExitKey(s *state.State, key string, exitTimeout func() tea.Cmd) tea.Cmd {
	if s.Running {
		ClearExitConfirm(s)
		s.ErrLine = view.RunningTurnHint()
		return exitTimeout()
	}
	return HandleExitConfirmKey(s, key, exitTimeout)
}

func HandleExitConfirmKey(s *state.State, key string, exitTimeout func() tea.Cmd) tea.Cmd {
	if s.ExitConfirmPending {
		if s.ExitConfirmKey != key {
			return nil
		}
		return tea.Quit
	}
	s.ExitConfirmPending = true
	s.ExitConfirmKey = key
	s.ExitConfirmArmedAt = time.Now()
	s.ErrLine = exitConfirmHintFor(key)
	return exitTimeout()
}

func ClearExitConfirm(s *state.State) {
	if !s.ExitConfirmPending {
		return
	}
	hint := exitConfirmHintFor(s.ExitConfirmKey)
	s.ExitConfirmPending = false
	s.ExitConfirmKey = ""
	if s.ErrLine == hint {
		s.ErrLine = ""
	}
}
