package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/session"
)

func contentLineCount(s string) int {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func (m *model) chatViewportContent(width int) string {
	if width < 10 {
		width = 10
	}
	header := renderHeader(width, m.deps.Version, m.deps.Cfg, m.headerSessionPtr())
	chat := renderChat(m.chat, width, time.Now(), m.toolDetailsVisible)
	if chat == "" {
		return header
	}
	return header + "\n\n" + chat
}

func (m *model) syncChatView() {
	if m.width == 0 {
		return
	}
	atBottom := m.chatVP.AtBottom()
	yoff := m.chatVP.YOffset

	m.layout()
	text := m.chatViewportContent(m.chatVP.Width)
	m.chatVP.SetContent(text)
	if atBottom {
		m.chatVP.GotoBottom()
	} else {
		m.chatVP.SetYOffset(yoff)
	}
}

func (m *model) syncToolView() {
	m.toolVP.SetContent(strings.Join(m.toolLines, "\n"))
	m.layout()
	m.toolVP.GotoBottom()
}

// layout sizes the chat/tool viewports to their content (capped by terminal
// height) so the input and footer sit directly under the transcript instead of
// being pinned to the bottom of the screen.
func (m *model) layout() {
	if m.width == 0 {
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
	innerW := m.width - 2
	if innerW < 10 {
		innerW = 10
	}
	m.input.Width = innerW - 2

	chromeH := gapAfterChat + inputFrameH + gapAfterInput + footerH
	if m.errLine != "" {
		chromeH += 2
	}

	toolLines := 0
	if m.toolOpen && len(m.toolLines) > 0 {
		toolLines = contentLineCount(strings.Join(m.toolLines, "\n"))
		if toolLines > maxToolLines {
			toolLines = maxToolLines
		}
		chromeH += toolLines + gapAfterTool
	}

	maxChatH := m.height - chromeH
	if maxChatH < 1 {
		maxChatH = 1
	}

	chatLines := contentLineCount(m.chatViewportContent(innerW))
	chatH := chatLines
	if chatH > maxChatH {
		chatH = maxChatH
	}

	m.chatVP.Width = innerW
	m.chatVP.Height = chatH
	m.toolVP.Width = innerW
	m.toolVP.Height = toolLines
}

func (m *model) headerSessionPtr() *session.Session {
	if !m.hasSession {
		return nil
	}
	s := m.headerSession
	return &s
}

func (m *model) refreshStatus() {
	if m.deps == nil || m.deps.Store == nil {
		return
	}
	ctx := context.Background()
	sess, err := m.deps.Store.Get(ctx, m.sessionID)
	if err != nil {
		m.hasSession = false
		m.statusRight = err.Error()
		return
	}
	m.headerSession = sess
	m.hasSession = true

	next := ""
	if m.deps.Context != nil {
		if bd := m.deps.Context.CachedBreakdown(); bd != nil {
			next = fmt.Sprintf(" · ctx ~%d", bd.Total())
		} else if view, err := m.deps.Context.BuildAPIContext(ctx, m.sessionID); err == nil {
			if b, err := ctxpkg.CountBreakdown(view); err == nil {
				next = fmt.Sprintf(" · ctx ~%d", b.Total())
			}
		}
	}
	m.statusRight = fmt.Sprintf("in %d · out %d · cache %d%s",
		sess.PromptTokensTotal, sess.CompletionTokensTotal, sess.PromptCacheHitTokensTotal, next)
}

func (m *model) View() string {
	if m.width == 0 {
		return styleApp.Render("Loading…\n")
	}
	var b strings.Builder

	if m.chatVP.Height > 0 {
		b.WriteString(m.chatVP.View())
		b.WriteString("\n")
	}

	if m.toolOpen && m.toolVP.Height > 0 {
		b.WriteString(m.toolVP.View())
		b.WriteString("\n")
	}

	inputBody := m.input.View()
	if m.running {
		inputBody = styleFooterHint.Render("Working… · Esc to cancel")
	}
	b.WriteString(renderInputFrame(m.width, inputBody))
	b.WriteString("\n")

	footerLeft := "? for shortcuts · Ctrl+O tool details"
	if m.toolDetailsVisible {
		footerLeft += " (on)"
	}
	if m.running {
		footerLeft = "Esc cancel · Ctrl+R reasoning · Ctrl+O tool details"
		if m.toolDetailsVisible {
			footerLeft += " (on)"
		}
	}
	b.WriteString(renderFooter(m.width, footerLeft, m.statusRight))

	if m.errLine != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorError).Render(m.errLine))
	}

	if m.overlay != overlayNone && m.overlayText != "" {
		b.WriteString("\n\n")
		overlay := styleOverlay.Width(m.width - 4).Render(m.overlayText)
		b.WriteString(overlay)
	}

	return styleApp.Render(b.String())
}
