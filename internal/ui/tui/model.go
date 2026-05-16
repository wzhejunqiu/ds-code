package tui

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hejunqiu/ds-code/internal/billing"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	uipkg "github.com/hejunqiu/ds-code/internal/ui"
	"github.com/hejunqiu/ds-code/internal/ui/slash"
)

type overlayKind int

const (
	overlayNone overlayKind = iota
	overlayContext
	overlayHelp
	overlayComplete
	overlayPrompt
)

type model struct {
	deps      *Deps
	sessionID string
	width     int
	height    int
	chatVP    viewport.Model
	toolVP    viewport.Model
	input     textarea.Model
	chat      []chatBlock
	toolLines []string
	toolOpen  bool

	overlay     overlayKind
	overlayText string
	complete    []slash.Command
	completeIdx int

	prompt *permission.PromptRequest

	running      bool
	turnCancel   context.CancelFunc
	reasoningAll bool

	statusLeft  string
	statusRight string
	errLine     string
}

func newModel(d *Deps) model {
	ta := textarea.New()
	ta.Placeholder = "Message… (/ for commands, Ctrl+C cancel, Ctrl+R reasoning)"
	ta.Focus()
	ta.CharLimit = 0
	ta.SetWidth(40)
	ta.SetHeight(3)

	chatVP := viewport.New(40, 10)
	toolVP := viewport.New(40, 4)

	m := model{
		deps:      d,
		sessionID: d.SessionID,
		chatVP:    chatVP,
		toolVP:    toolVP,
		input:     ta,
		toolOpen:  true,
	}
	m.refreshStatus()
	return m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		m.listenPrompt(),
		statusTick(),
	)
}

func statusTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return statusRefreshMsg{} })
}

func (m *model) listenPrompt() tea.Cmd {
	if m.deps.PromptCh == nil {
		return nil
	}
	ch := m.deps.PromptCh
	return func() tea.Msg {
		req, ok := <-ch
		if !ok {
			return nil
		}
		return promptRequestMsg{req: req}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		return m, nil

	case tea.KeyMsg:
		if m.overlay == overlayPrompt && m.prompt != nil {
			return m.handlePromptKey(msg)
		}
		if m.overlay == overlayComplete {
			if m.handleCompleteKey(msg) {
				return m, nil
			}
		}
		if m.overlay == overlayContext || m.overlay == overlayHelp {
			switch msg.String() {
			case "esc", "q":
				m.overlay = overlayNone
				m.overlayText = ""
				return m, nil
			}
		}
		switch msg.String() {
		case "ctrl+c":
			if m.running && m.turnCancel != nil {
				m.turnCancel()
				m.errLine = "cancelled"
				return m, nil
			}
			return m, tea.Quit
		case "ctrl+r":
			m.reasoningAll = !m.reasoningAll
			for i := range m.chat {
				if m.chat[i].role == "assistant" {
					m.chat[i].reasoningOpen = m.reasoningAll
				}
			}
			m.syncChatView()
			return m, nil
		case "ctrl+l":
			return m, m.showContextOverlay()
		case "ctrl+t":
			m.toolOpen = !m.toolOpen
			m.layout()
			return m, nil
		case "esc":
			if m.overlay != overlayNone {
				m.overlay = overlayNone
				m.overlayText = ""
				return m, nil
			}
		}

	case streamContentMsg:
		if m.running {
			m.appendAssistantContent(msg.delta)
			m.syncChatView()
		}
		return m, nil
	case streamReasoningMsg:
		if m.running {
			m.appendAssistantReasoning(msg.delta)
			m.syncChatView()
		}
		return m, nil
	case slashOutputMsg:
		if msg.text != "" {
			m.chat = append(m.chat, chatBlock{role: "assistant"})
			m.chat[len(m.chat)-1].content.WriteString(msg.text)
			m.syncChatView()
		}
		m.refreshStatus()
		return m, nil
	case toolStartMsg:
		m.toolLines = append(m.toolLines, toolLine(msg.name, "", true))
		m.syncToolView()
		return m, nil
	case toolEndMsg:
		if len(m.toolLines) > 0 {
			m.toolLines[len(m.toolLines)-1] = toolLine(msg.name, msg.preview, false)
		} else {
			m.toolLines = append(m.toolLines, toolLine(msg.name, msg.preview, false))
		}
		m.syncToolView()
		return m, nil
	case turnStartedMsg:
		m.turnCancel = msg.cancel
		return m, nil

	case contextOverlayMsg:
		m.overlay = overlayContext
		m.overlayText = msg.text
		return m, nil
	case helpOverlayMsg:
		m.overlay = overlayHelp
		m.overlayText = msg.text
		return m, nil

	case turnDoneMsg:
		m.running = false
		m.turnCancel = nil
		if len(m.chat) > 0 {
			m.chat[len(m.chat)-1].streaming = false
		}
		if msg.err != nil {
			m.errLine = msg.err.Error()
		} else {
			m.errLine = ""
		}
		m.syncChatView()
		m.refreshStatus()
		return m, tea.Batch(m.listenPrompt(), statusTick())

	case statusRefreshMsg:
		m.refreshStatus()
		return m, statusTick()

	case promptRequestMsg:
		m.prompt = &msg.req
		m.overlay = overlayPrompt
		m.overlayText = fmt.Sprintf("Allow %s?\n%s\n[y] yes  [n] no", msg.req.Tool, truncate(msg.req.Summary, 300))
		return m, m.listenPrompt()

	case overlayCloseMsg:
		m.overlay = overlayNone
		m.overlayText = ""
		m.refreshStatus()
		return m, nil
	}

	if m.running {
		var cmd tea.Cmd
		m.chatVP, cmd = m.chatVP.Update(msg)
		return m, cmd
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)
	m.updateCompletion()

	if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEnter && !msg.Alt {
		line := strings.TrimSpace(m.input.Value())
		if line != "" {
			m.input.Reset()
			m.overlay = overlayNone
			m.complete = nil
			return m, m.submitLine(line)
		}
	}

	m.chatVP, cmd = m.chatVP.Update(msg)
	cmds = append(cmds, cmd)
	if m.toolOpen {
		m.toolVP, cmd = m.toolVP.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *model) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y", "yes":
		m.prompt.Reply <- true
		m.prompt = nil
		m.overlay = overlayNone
	case "n", "no", "esc":
		m.prompt.Reply <- false
		m.prompt = nil
		m.overlay = overlayNone
	}
	return m, m.listenPrompt()
}

func (m *model) handleCompleteKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "up":
		if m.completeIdx > 0 {
			m.completeIdx--
		}
		m.renderCompleteOverlay()
		return true
	case "down", "tab":
		if m.completeIdx < len(m.complete)-1 {
			m.completeIdx++
		}
		m.renderCompleteOverlay()
		return true
	case "enter":
		if len(m.complete) > 0 {
			c := m.complete[m.completeIdx]
			m.input.SetValue("/" + c.Name + " ")
			m.input.CursorEnd()
			m.overlay = overlayNone
			m.complete = nil
		}
		return true
	case "esc":
		m.overlay = overlayNone
		m.complete = nil
		return true
	}
	return false
}

func (m *model) updateCompletion() {
	val := m.input.Value()
	if !strings.HasPrefix(strings.TrimSpace(val), "/") {
		if m.overlay == overlayComplete {
			m.overlay = overlayNone
			m.complete = nil
		}
		return
	}
	m.complete = slash.FilterCommands(val)
	m.completeIdx = 0
	if len(m.complete) == 0 {
		m.overlay = overlayNone
		return
	}
	m.overlay = overlayComplete
	m.renderCompleteOverlay()
}

func (m *model) renderCompleteOverlay() {
	var b strings.Builder
	for i, c := range m.complete {
		prefix := "  "
		if i == m.completeIdx {
			prefix = "▸ "
		}
		fmt.Fprintf(&b, "%s/%s — %s\n", prefix, c.Name, c.Description)
	}
	m.overlayText = strings.TrimRight(b.String(), "\n")
}

func (m *model) submitLine(line string) tea.Cmd {
	if line == "/exit" || line == "/quit" {
		return tea.Quit
	}
	if cmd, args, ok := slash.Parse(line); ok {
		switch cmd {
		case "help":
			return m.showHelpOverlay()
		case "context":
			if strings.TrimSpace(args) == "--json" {
				return m.showContextJSON()
			}
			return m.showContextOverlay()
		default:
			return m.runSlash(line)
		}
	}

	m.chat = append(m.chat, chatBlock{role: "user"})
	m.chat[len(m.chat)-1].content.WriteString(line)
	m.chat = append(m.chat, chatBlock{role: "assistant", streaming: true, reasoningOpen: m.reasoningAll})
	m.syncChatView()
	m.running = true
	m.errLine = ""
	m.toolLines = nil
	m.syncToolView()

	d := *m.deps
	d.SessionID = m.sessionID
	return func() tea.Msg {
		go runTurnAsync(d, line, m.deps.Events)
		return nil
	}
}

func (m *model) showHelpOverlay() tea.Cmd {
	var buf bytes.Buffer
	slash.WriteHelp(&buf)
	text := buf.String()
	return func() tea.Msg {
		return helpOverlayMsg{text: text}
	}
}

func (m *model) runSlash(line string) tea.Cmd {
	d := m.deps
	return func() tea.Msg {
		var buf bytes.Buffer
		sid := m.sessionID
		handled, err := d.HandleSlash(context.Background(), &buf, &sid, line)
		m.sessionID = sid
		d.SessionID = sid
		if err != nil {
			return turnDoneMsg{err: err}
		}
		if handled {
			if out := strings.TrimSpace(buf.String()); out != "" {
				return slashOutputMsg{text: out}
			}
			return overlayCloseMsg{}
		}
		return slashOutputMsg{text: fmt.Sprintf("Unknown command (try /help)")}
	}
}

func (m *model) showContextOverlay() tea.Cmd {
	d := m.deps
	return func() tea.Msg {
		ctx := context.Background()
		sess, err := d.Store.Get(ctx, m.sessionID)
		if err != nil {
			return turnDoneMsg{err: err}
		}
		view, err := d.Context.BuildAPIContext(ctx, m.sessionID)
		if err != nil {
			return turnDoneMsg{err: err}
		}
		panel, err := uipkg.BuildContextPanelData(d.Cfg, sess, view)
		if err != nil {
			return turnDoneMsg{err: err}
		}
		text := uipkg.FormatContextPanel(panel)
		return contextOverlayMsg{text: text}
	}
}

func (m *model) showContextJSON() tea.Cmd {
	d := m.deps
	return func() tea.Msg {
		ctx := context.Background()
		sess, err := d.Store.Get(ctx, m.sessionID)
		if err != nil {
			return turnDoneMsg{err: err}
		}
		view, err := d.Context.BuildAPIContext(ctx, m.sessionID)
		if err != nil {
			return turnDoneMsg{err: err}
		}
		panel, err := uipkg.BuildContextPanelData(d.Cfg, sess, view)
		if err != nil {
			return turnDoneMsg{err: err}
		}
		text, err := uipkg.FormatContextJSON(panel)
		if err != nil {
			return turnDoneMsg{err: err}
		}
		return contextOverlayMsg{text: text}
	}
}

func (m *model) appendAssistantContent(s string) {
	if len(m.chat) == 0 || m.chat[len(m.chat)-1].role != "assistant" {
		m.chat = append(m.chat, chatBlock{role: "assistant", streaming: true, reasoningOpen: m.reasoningAll})
	}
	m.chat[len(m.chat)-1].appendContent(s)
}

func (m *model) appendAssistantReasoning(s string) {
	if len(m.chat) == 0 || m.chat[len(m.chat)-1].role != "assistant" {
		m.chat = append(m.chat, chatBlock{role: "assistant", streaming: true, reasoningOpen: m.reasoningAll})
	}
	m.chat[len(m.chat)-1].appendReasoning(s)
}

func (m *model) syncChatView() {
	text := renderChat(m.chat, m.chatVP.Width)
	m.chatVP.SetContent(text)
	m.chatVP.GotoBottom()
}

func (m *model) syncToolView() {
	m.toolVP.SetContent(strings.Join(m.toolLines, "\n"))
	m.toolVP.GotoBottom()
}

func (m *model) layout() {
	if m.width == 0 {
		return
	}
	statusH := 1
	inputH := 4
	toolH := 4
	if !m.toolOpen {
		toolH = 0
	}
	chatH := m.height - statusH - inputH - toolH - 2
	if chatH < 3 {
		chatH = 3
	}
	m.chatVP.Width = m.width - 2
	m.chatVP.Height = chatH
	m.toolVP.Width = m.width - 2
	m.toolVP.Height = toolH
	m.input.SetWidth(m.width - 2)
}

func (m *model) refreshStatus() {
	ctx := context.Background()
	sess, err := m.deps.Store.Get(ctx, m.sessionID)
	if err != nil {
		m.statusLeft = "ds-code"
		m.statusRight = err.Error()
		return
	}
	snap := session.UsageSnapshotFromSession(sess)
	cost := billing.FormatUSD(billing.EstimateUSD(sess.Model, snap))
	m.statusLeft = fmt.Sprintf("%s · %s · thinking %s", sess.Model, sess.ReasoningEffort, sess.ThinkingType)

	next := ""
	if bd := m.deps.Context.CachedBreakdown(); bd != nil {
		next = fmt.Sprintf(" · next ~%d", bd.Total())
	} else if view, err := m.deps.Context.BuildAPIContext(ctx, m.sessionID); err == nil {
		if b, err := ctxpkg.CountBreakdown(view); err == nil {
			next = fmt.Sprintf(" · next ~%d", b.Total())
		}
	}
	m.statusRight = fmt.Sprintf("in %d out %d cache %d · %s%s",
		sess.PromptTokensTotal, sess.CompletionTokensTotal, sess.PromptCacheHitTokensTotal, cost, next)
}

func (m *model) View() string {
	if m.width == 0 {
		return "Loading…\n"
	}
	var b strings.Builder
	border := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238"))
	b.WriteString(border.Render(m.chatVP.View()))
	b.WriteString("\n")
	if m.toolOpen {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Tools"))
		b.WriteString("\n")
		b.WriteString(border.Render(m.toolVP.View()))
		b.WriteString("\n")
	}
	b.WriteString(m.input.View())
	b.WriteString("\n")
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	b.WriteString(statusStyle.Render(m.statusLeft + " │ " + m.statusRight))
	if m.errLine != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(m.errLine))
	}
	if m.overlay != overlayNone && m.overlayText != "" {
		b.WriteString("\n\n")
		overlay := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("99")).
			Padding(0, 1).
			Width(m.width - 4).
			Render(m.overlayText)
		b.WriteString(overlay)
	}
	return b.String()
}
