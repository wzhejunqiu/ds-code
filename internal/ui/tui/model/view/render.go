package view

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/ui/theme"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/hejunqiu/ds-code/internal/ui/tui/header"
	"github.com/hejunqiu/ds-code/internal/ui/tui/layout"
	subagentui "github.com/hejunqiu/ds-code/internal/ui/tui/model/subagent"
	"github.com/hejunqiu/ds-code/internal/session/usageagg"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/hejunqiu/ds-code/internal/ui/tui/style"
)

const runningTurnHint = "Press Esc to cancel the current turn"

func ContentLineCount(s string) int {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func ChatViewportContent(s *state.State, width int) string {
	if width < 10 {
		width = 10
	}
	var sess *session.Session
	if s.HasSession {
		sess = &s.HeaderSession
	}
	hdr := header.Render(width, s.Deps.Version, s.Deps.Cfg, sess)
	if s.SubagentNav == state.SubagentNavDetail {
		if crumb := subagentui.DetailBreadcrumb(s); crumb != "" {
			hdr += "\n" + style.FooterHint.Render(crumb+"  (← back to list)")
		}
	}
	body := chat.Render(s.Chat, width, time.Now(), s.ToolDetailsVisible)
	if body == "" {
		return hdr
	}
	return hdr + "\n\n" + body
}

func SyncChat(s *state.State, chatVP, toolVP *viewport.Model, input *textinput.Model) {
	if s.Width == 0 {
		return
	}
	atBottom := chatVP.AtBottom()
	yoff := chatVP.YOffset
	Layout(s, chatVP, toolVP, input)
	text := ChatViewportContent(s, chatVP.Width)
	chatVP.SetContent(text)
	if atBottom {
		chatVP.GotoBottom()
	} else {
		chatVP.SetYOffset(yoff)
	}
}

func SyncTool(s *state.State, chatVP, toolVP *viewport.Model, input *textinput.Model) {
	toolVP.SetContent(strings.Join(s.ToolLines, "\n"))
	Layout(s, chatVP, toolVP, input)
	toolVP.GotoBottom()
}

func Layout(s *state.State, chatVP, toolVP *viewport.Model, input *textinput.Model) {
	if s.Width == 0 {
		return
	}
	const (
		footerH       = 1
		inputFrameH   = 3
		gapAfterChat  = 1
		gapAfterTool  = 1
		gapAfterInput = 1
		maxToolLines  = 5
	)
	innerW := s.Width - 2
	if innerW < 10 {
		innerW = 10
	}
	if input != nil {
		input.Width = innerW - 2
	}

	chromeH := gapAfterChat + inputFrameH + gapAfterInput + footerH
	if s.ErrLine != "" {
		chromeH += 2
	}

	toolLines := 0
	if s.ToolOpen && len(s.ToolLines) > 0 {
		toolLines = ContentLineCount(strings.Join(s.ToolLines, "\n"))
		if toolLines > maxToolLines {
			toolLines = maxToolLines
		}
		chromeH += toolLines + gapAfterTool
	}

	maxChatH := s.Height - chromeH
	if maxChatH < 1 {
		maxChatH = 1
	}

	chatLines := ContentLineCount(ChatViewportContent(s, innerW))
	chatH := chatLines
	if chatH > maxChatH {
		chatH = maxChatH
	}

	if chatVP != nil {
		chatVP.Width = innerW
		chatVP.Height = chatH
	}
	if toolVP != nil {
		toolVP.Width = innerW
		toolVP.Height = toolLines
	}
}

func RefreshStatus(s *state.State) {
	if s.Deps == nil || s.Deps.Store == nil {
		return
	}
	ctx := context.Background()
	sess, err := s.Deps.Store.Get(ctx, s.SessionID)
	if err != nil {
		s.HasSession = false
		s.StatusRight = err.Error()
		return
	}
	s.HeaderSession = sess
	s.HasSession = true

	next := ""
	if s.Deps.Context != nil {
		if bd := s.Deps.Context.CachedBreakdown(); bd != nil {
			next = fmt.Sprintf(" · ctx ~%d", bd.Total())
		} else if view, err := s.Deps.Context.BuildAPIContext(ctx, s.SessionID); err == nil {
			if b, err := ctxpkg.CountBreakdown(view); err == nil {
				next = fmt.Sprintf(" · ctx ~%d", b.Total())
			}
		}
	}
	snap, err := usageagg.TotalForSession(ctx, s.Deps.Store, s.Deps.Subagent, s.SessionID)
	if err != nil {
		snap = session.UsageSnapshotFromSession(sess)
	}
	s.StatusRight = fmt.Sprintf("in %d · out %d · cache %d%s",
		snap.PromptTokensTotal, snap.CompletionTokensTotal, snap.PromptCacheHitTokensTotal, next)
}

func Render(s *state.State, chatVP, toolVP *viewport.Model, input textinput.Model) string {
	if s.Width == 0 {
		return style.App.Render("Loading…\n")
	}
	var b strings.Builder

	if chatVP.Height > 0 {
		b.WriteString(chatVP.View())
		b.WriteString("\n")
	}

	if s.ToolOpen && toolVP.Height > 0 {
		b.WriteString(toolVP.View())
		b.WriteString("\n")
	}

	inputBody := input.View()
	if s.Running {
		inputBody = style.FooterHint.Render("Working… · Esc to cancel")
	}
	b.WriteString(layout.InputFrame(s.Width, inputBody))
	b.WriteString("\n")

	footerLeft := "? for shortcuts · ↓ subagents · Ctrl+O tool details"
	if s.ToolDetailsVisible {
		footerLeft += " (on)"
	}
	if s.SubagentNav == state.SubagentNavDetail {
		footerLeft = "← back · ↓ list · Ctrl+O tool details"
	} else if s.SubagentNav == state.SubagentNavList {
		footerLeft = "↑↓ select · Enter view · ← back"
	} else if s.Running {
		footerLeft = "Esc cancel · ↓ subagents · Ctrl+R reasoning · Ctrl+O tool details"
		if s.ToolDetailsVisible {
			footerLeft += " (on)"
		}
	}
	b.WriteString(layout.Footer(s.Width, footerLeft, s.StatusRight))

	if s.ErrLine != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(theme.Error).Render(s.ErrLine))
	}

	if s.Overlay != state.OverlayNone && s.OverlayText != "" {
		b.WriteString("\n\n")
		b.WriteString(style.Overlay.Width(s.Width - 4).Render(s.OverlayText))
	}

	return style.App.Render(b.String())
}

func RunningTurnHint() string { return runningTurnHint }
