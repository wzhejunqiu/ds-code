package view

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/theme"
	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/session/usageagg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/header"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/layout"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/markdown"
	subagentui "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/subagent"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/style"
)

const runningTurnHint = "Press Esc to cancel the current turn"

// SyncCaches holds optional render caches for chat sync.
type SyncCaches struct {
	Chat   *chat.RenderCache
	MD     *markdown.SegmentCache
	Header *HeaderCache
}

// HeaderCache caches the header segment inside the chat viewport.
type HeaderCache struct {
	text string
	key  headerCacheKey
}

type headerCacheKey struct {
	width          int
	hasSession     bool
	costCNY        float64
	subagentNav    state.SubagentNav
	model          string
	thinking       string
	projectRoot    string
	version        string
	breadcrumb     string
}

// Invalidate clears the cached header.
func (c *HeaderCache) Invalidate() {
	if c == nil {
		return
	}
	c.text = ""
	c.key = headerCacheKey{}
}

func ContentLineCount(s string) int {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func buildHeader(s *state.State, width int) string {
	if width < 10 {
		width = 10
	}
	var sess *session.Session
	if s.HasSession {
		sess = &s.HeaderSession
	}
	hdr := header.Render(width, s.Deps.Version, s.Deps.Cfg, sess, s.HeaderCostCNY)
	if s.SubagentNav == state.SubagentNavDetail {
		if crumb := subagentui.DetailBreadcrumb(s); crumb != "" {
			hdr += "\n" + style.FooterHint.Render(crumb+"  (← back to list)")
		}
	}
	return hdr
}

func headerKey(s *state.State, width int) headerCacheKey {
	key := headerCacheKey{
		width:       width,
		hasSession:  s.HasSession,
		costCNY:     s.HeaderCostCNY,
		subagentNav: s.SubagentNav,
		version:     s.Deps.Version,
	}
	if s.HasSession {
		key.model = s.HeaderSession.Model
		key.thinking = s.HeaderSession.ThinkingType
	}
	if s.Deps != nil && s.Deps.Cfg != nil {
		key.projectRoot = s.Deps.Cfg.ProjectRoot
	}
	if s.SubagentNav == state.SubagentNavDetail {
		key.breadcrumb = subagentui.DetailBreadcrumb(s)
	}
	return key
}

func buildHeaderCached(s *state.State, width int, cache *HeaderCache) string {
	key := headerKey(s, width)
	if cache != nil && cache.text != "" && cache.key == key {
		return cache.text
	}
	text := buildHeader(s, width)
	if cache != nil {
		cache.text = text
		cache.key = key
	}
	return text
}

func buildChatBody(s *state.State, width int, caches *SyncCaches) (text string, lineCount int) {
	if width < 10 {
		width = 10
	}
	var chatCache *chat.RenderCache
	var mdCache *markdown.SegmentCache
	if caches != nil {
		chatCache = caches.Chat
		mdCache = caches.MD
	}
	text = chat.RenderCached(s.Chat, width, time.Now(), s.ToolDetailsVisible, chatCache, mdCache)
	lineCount = ContentLineCount(text)
	return text, lineCount
}

func joinViewportContent(hdr, body string) string {
	if body == "" {
		return hdr
	}
	return hdr + "\n\n" + body
}

func buildViewportContent(s *state.State, width int, caches *SyncCaches) (content string, lineCount int) {
	if width < 10 {
		width = 10
	}
	hdr := buildHeaderCached(s, width, cacheHeader(caches))
	body, _ := buildChatBody(s, width, caches)
	content = joinViewportContent(hdr, body)
	lineCount = ContentLineCount(content)
	return content, lineCount
}

// ChatViewportContent renders header+body (legacy helper for tests).
func ChatViewportContent(s *state.State, width int) string {
	content, _ := buildViewportContent(s, width, nil)
	return content
}

func SyncChat(s *state.State, chatVP, toolVP *viewport.Model, input *textinput.Model, caches *SyncCaches) {
	if s.Width == 0 {
		return
	}
	innerW := s.Width - 2
	if innerW < 10 {
		innerW = 10
	}

	atBottom := chatVP.AtBottom()
	yoff := chatVP.YOffset

	content, chatLines := buildViewportContent(s, innerW, caches)

	Layout(s, chatVP, toolVP, input, chatLines)
	chatVP.SetContent(content)
	if atBottom {
		chatVP.GotoBottom()
	} else {
		chatVP.SetYOffset(yoff)
	}
}

func cacheHeader(caches *SyncCaches) *HeaderCache {
	if caches == nil {
		return nil
	}
	return caches.Header
}

func SyncTool(s *state.State, chatVP, toolVP *viewport.Model, input *textinput.Model, caches *SyncCaches) {
	toolVP.SetContent(strings.Join(s.ToolLines, "\n"))
	innerW := s.Width - 2
	if innerW < 10 {
		innerW = 10
	}
	_, chatLines := buildViewportContent(s, innerW, caches)
	Layout(s, chatVP, toolVP, input, chatLines)
	toolVP.GotoBottom()
}

func Layout(s *state.State, chatVP, toolVP *viewport.Model, input *textinput.Model, chatLines int) {
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
	if s.SensitiveLogWarn != "" {
		chromeH += ContentLineCount(s.SensitiveLogWarn) + 1
	}
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

	chatH := chatLines
	if chatH > maxChatH {
		chatH = maxChatH
	}

	if chatVP != nil {
		chatVP.Width = innerW
		chatVP.Height = chatH
		delta := chatH / 3
		if delta < 3 {
			delta = 3
		}
		chatVP.MouseWheelDelta = delta
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
		if view, err := s.Deps.Context.BuildAPIContext(ctx, s.SessionID); err == nil {
			if b, err := ctxpkg.CountBreakdown(view); err == nil {
				next = fmt.Sprintf(" · ctx ~%d", b.Total())
			}
		}
	}
	snap, err := usageagg.TotalForSession(ctx, s.Deps.Store, s.Deps.Subagent, s.SessionID)
	if err != nil {
		snap = session.UsageSnapshotFromSession(sess)
	}
	cost, err := usageagg.EstimateCostForSession(ctx, s.Deps.Store, s.Deps.Subagent, s.SessionID)
	if err != nil {
		s.HeaderCostCNY = 0
	} else {
		s.HeaderCostCNY = cost.TotalCNY
	}
	costLabel := ""
	if s.HeaderCostCNY > 0 {
		costLabel = " · " + billing.FormatCNY(s.HeaderCostCNY)
	}
	s.StatusRight = fmt.Sprintf("in %d · out %d · cache %d%s%s",
		snap.PromptTokensTotal, snap.CompletionTokensTotal, snap.PromptCacheHitTokensTotal, costLabel, next)
}

// FooterLeft builds the left footer hint string (exported for tests).
func FooterLeft(s *state.State) string {
	footerLeft := "? for shortcuts · ↓ subagents · Ctrl+O tool details"
	if s.Deps != nil && s.Deps.BackgroundAgents != nil {
		if n := s.Deps.BackgroundAgents(); n > 0 {
			footerLeft = fmt.Sprintf("%d agents running in background · ", n) + footerLeft
		}
	}
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
	return footerLeft
}

func Render(s *state.State, chatVP, toolVP *viewport.Model, input *textinput.Model) string {
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

	var inputBody string
	if input != nil {
		inputBody = input.View()
	}
	if s.Running {
		inputBody = style.FooterHint.Render("Working… · Esc to cancel")
	}
	b.WriteString(layout.InputFrame(s.Width, inputBody))
	b.WriteString("\n")

	footerLeft := FooterLeft(s)
	b.WriteString(layout.Footer(s.Width, footerLeft, s.StatusRight))

	if s.SensitiveLogWarn != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(theme.Error).Render(s.SensitiveLogWarn))
	}

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
