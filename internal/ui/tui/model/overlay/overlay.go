package overlay

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	uipkg "github.com/wzhejunqiu/ds-code/internal/ui"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/view"
)

func OnWindowSize(s *state.State, width, height int, syncAll func()) tea.Cmd {
	s.Width = width
	s.Height = height
	s.NoticeScrollOffset = 0
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

func UpdateWebFetchPromptRequest(s *state.State, m msg.WebFetchPromptRequestMsg, listenWebFetch func() tea.Cmd) tea.Cmd {
	s.WebFetchPrompt = &m.Req
	s.Overlay = state.OverlayWebFetchPrompt
	summary := m.Req.URL
	if summary == "" {
		summary = m.Req.Host
	}
	s.OverlayText = fmt.Sprintf("web_fetch: %s 不在 allowlist\n%s\n[1/a] 允许本次  [2/s] 始终允许  [3/d] 拒绝", m.Req.Host, chat.Truncate(summary, 300))
	return listenWebFetch()
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
	prev := s.Overlay
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
	case state.OverlayWebFetchPrompt:
		DismissWebFetchPrompt(s)
	case state.OverlayContext, state.OverlayHelp, state.OverlaySubagentList, state.OverlayTCase:
		s.Overlay = state.OverlayNone
		s.OverlayText = ""
	default:
		if s.Overlay != state.OverlayNone {
			s.Overlay = state.OverlayNone
			s.OverlayText = ""
		}
	}
	if prev == state.OverlaySubagentList {
		s.SubagentNav = state.SubagentNavMain
		s.SyncDisplayedChat()
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

func DismissWebFetchPrompt(s *state.State) {
	if s.WebFetchPrompt == nil {
		return
	}
	select {
	case s.WebFetchPrompt.Reply <- permission.WebFetchDeny:
	default:
	}
	s.WebFetchPrompt = nil
	s.Overlay = state.OverlayNone
	s.OverlayText = ""
}
