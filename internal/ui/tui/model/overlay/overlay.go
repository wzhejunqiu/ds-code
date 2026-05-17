package overlay

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	uipkg "github.com/hejunqiu/ds-code/internal/ui"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/view"
)

func OnWindowSize(s *state.State, width, height int, syncAll func()) tea.Cmd {
	s.Width = width
	s.Height = height
	syncAll()
	return nil
}

func UpdateContext(s *state.State, m msg.ContextOverlayMsg) tea.Cmd {
	s.Overlay = state.OverlayContext
	s.OverlayText = m.Text
	return nil
}

func UpdateHelp(s *state.State, m msg.HelpOverlayMsg) tea.Cmd {
	s.Overlay = state.OverlayHelp
	s.OverlayText = m.Text
	return nil
}

func UpdateStatusRefresh(s *state.State, syncChat func(), statusTick func() tea.Cmd) tea.Cmd {
	view.RefreshStatus(s)
	syncChat()
	return statusTick()
}

func UpdateExitConfirmTimeout(s *state.State) tea.Cmd {
	if s.ExitConfirmPending && time.Since(s.ExitConfirmArmedAt) >= uipkg.ExitConfirmTimeout {
		ClearExitConfirm(s)
	}
	if s.ErrLine == view.RunningTurnHint() {
		s.ErrLine = ""
	}
	return nil
}

func UpdatePromptRequest(s *state.State, m msg.PromptRequestMsg, listenPrompt func() tea.Cmd) tea.Cmd {
	s.Prompt = &m.Req
	s.Overlay = state.OverlayPrompt
	s.OverlayText = fmt.Sprintf("Allow %s?\n%s\n[y] yes  [n] no", m.Req.Tool, chat.Truncate(m.Req.Summary, 300))
	return listenPrompt()
}

func UpdateClose(s *state.State, syncChat func(), refreshStatus func()) tea.Cmd {
	s.Overlay = state.OverlayNone
	s.OverlayText = ""
	refreshStatus()
	syncChat()
	return nil
}

// Dismiss closes the active overlay (state fields only; widgets cleared by caller).
func Dismiss(s *state.State) {
	switch s.Overlay {
	case state.OverlayComplete:
		s.Overlay = state.OverlayNone
		s.OverlayText = ""
		s.Complete = nil
		s.CompleteFilterKey = ""
	case state.OverlayResume:
		s.Overlay = state.OverlayNone
		s.ResumeSessions = nil
		s.ResumeFilter = ""
		s.ResumePending = false
		s.OverlayText = ""
	case state.OverlayPrompt:
		DismissPrompt(s)
	case state.OverlayContext, state.OverlayHelp:
		s.Overlay = state.OverlayNone
		s.OverlayText = ""
	default:
		if s.Overlay != state.OverlayNone {
			s.Overlay = state.OverlayNone
			s.OverlayText = ""
		}
	}
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
