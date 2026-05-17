package tui

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hejunqiu/ds-code/internal/agent"
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
	overlayResume
	overlayPrompt
)

type model struct {
	deps      *Deps
	sessionID string
	width     int
	height    int
	chatVP    viewport.Model
	toolVP    viewport.Model
	input     textinput.Model
	chat      []chatBlock
	toolLines []string
	toolOpen  bool

	overlay     overlayKind
	overlayText string
	complete    []slash.Command
	completeIdx int

	resumeSessions []session.Summary
	resumeIdx      int
	resumeScroll   int    // first visible row in the session list overlay
	resumeFilter   string // last filter used for resume picker; avoids reset on cursor blink

	prompt *permission.PromptRequest

	running      bool
	turnCancel   context.CancelFunc
	reasoningAll bool
	toolDetailsVisible bool // Ctrl+O: expand tool args/result in chat

	statusRight string
	errLine     string

	headerSession session.Session
	hasSession    bool

	exitConfirmPending bool
	exitConfirmKey     string // "ctrl+c" or "ctrl+d"
	exitConfirmArmedAt time.Time
}

const exitConfirmTimeout = time.Second

func newModel(d *Deps) model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 40
	ti.Prompt = ""
	ti.Placeholder = ""
	ti.Cursor.Style = lipgloss.NewStyle().Reverse(true)
	ti.TextStyle = styleInputText

	chatVP := viewport.New(40, 10)
	toolVP := viewport.New(40, 4)

	m := model{
		deps:      d,
		sessionID: d.SessionID,
		chatVP:    chatVP,
		toolVP:    toolVP,
		input:     ti,
		toolOpen:  false,
	}
	m.refreshStatus()
	return m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		m.listenPrompt(),
		statusTick(),
		m.loadInitialHistory(),
	)
}

func statusTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return statusRefreshMsg{} })
}

const (
	thinkingFineTick   = 100 * time.Millisecond
	thinkingCoarseTick = time.Second
)

func thinkingTickAfter(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg { return thinkingTickMsg{} })
}

func exitConfirmTimeoutTick() tea.Cmd {
	return tea.Tick(exitConfirmTimeout, func(time.Time) tea.Msg { return exitConfirmTimeoutMsg{} })
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
		m.syncChatView()
		m.syncToolView()
		if m.overlay == overlayResume && len(m.resumeSessions) > 0 {
			m.renderResumeOverlay()
		}
		return m, nil

	case tea.KeyMsg:
		if !isExitConfirmKey(msg.String()) {
			m.clearExitConfirm()
		}
		if m.overlay == overlayResume {
			if msg.Type == tea.KeyEnter && !msg.Alt {
				if len(m.resumeSessions) > 0 {
					idx := m.resumeIdx
					if idx >= len(m.resumeSessions) {
						idx = len(m.resumeSessions) - 1
					}
					if idx < 0 {
						idx = 0
					}
					id := m.resumeSessions[idx].ID
					m.input.Reset()
					m.clearResumePicker()
					return m, m.resumeSession(id)
				}
				// Picker open with no matches — do not treat filter text as a session id.
				return m, nil
			}
			if m.handleResumeKey(msg) {
				return m, nil
			}
		}
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
				m.clearExitConfirm()
				return m, nil
			}
			return m.handleExitConfirmKey("ctrl+c")
		case "ctrl+d":
			return m.handleExitConfirmKey("ctrl+d")
		case "ctrl+r":
			m.reasoningAll = !m.reasoningAll
			for i := range m.chat {
				if m.chat[i].role == "assistant" {
					m.chat[i].reasoningOpen = m.reasoningAll
				}
			}
			m.syncChatView()
			return m, nil
		case "?":
			return m, m.showHelpOverlay()
		case "ctrl+l":
			return m, m.showContextOverlay()
		case "ctrl+t":
			m.toolOpen = !m.toolOpen
			m.layout()
			return m, nil
		case "ctrl+o":
			m.toolDetailsVisible = !m.toolDetailsVisible
			m.syncChatView()
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
			m.clearPlanningBlock()
			m.appendAssistantContent(msg.delta)
			m.syncChatView()
		}
		return m, nil
	case streamReasoningMsg:
		if m.running {
			m.clearPlanningBlock()
			var cmd tea.Cmd
			if m.appendAssistantReasoning(msg.delta) {
				cmd = m.nextThinkingTickCmd()
			}
			m.syncChatView()
			return m, cmd
		}
		return m, nil
	case planningStartMsg:
		if m.running {
			m.appendPlanningBlock()
			m.syncChatView()
			return m, m.nextThinkingTickCmd()
		}
		return m, nil
	case planningEndMsg:
		m.clearPlanningBlock()
		m.syncChatView()
		return m, nil
	case thinkingTickMsg:
		if m.needsThinkingTick() || m.needsPlanningTick() {
			m.syncChatView()
			return m, m.nextThinkingTickCmd()
		}
		return m, nil
	case slashOutputMsg:
		if msg.text != "" {
			m.chat = append(m.chat, chatBlock{role: "assistant"})
			m.chat[len(m.chat)-1].content.WriteString(msg.text)
		}
		m.refreshStatus()
		m.syncChatView()
		return m, nil

	case resumeListMsg:
		if msg.err != nil {
			m.errLine = msg.err.Error()
			m.overlay = overlayNone
			return m, nil
		}
		m.resumeSessions = msg.sessions
		m.resumeFilter = ""
		m.resumeIdx = 0
		m.resumeScroll = 0
		if len(m.resumeSessions) == 0 {
			m.errLine = "No saved sessions."
			m.overlay = overlayNone
			return m, nil
		}
		m.overlay = overlayResume
		m.renderResumeOverlay()
		return m, nil

	case sessionResumedMsg:
		if msg.err != nil {
			m.errLine = msg.err.Error()
			return m, nil
		}
		session.DropPending(m.deps.Store, m.sessionID)
		m.sessionID = msg.sessionID
		m.deps.SessionID = msg.sessionID
		m.chat = msg.chat
		m.toolLines = nil
		m.clearResumePicker()
		m.errLine = ""
		m.refreshStatus()
		m.syncChatView()
		m.syncToolView()
		return m, nil

	case historyLoadedMsg:
		if msg.err != nil {
			m.errLine = msg.err.Error()
			return m, nil
		}
		if len(msg.chat) > 0 {
			m.chat = msg.chat
			m.syncChatView()
		}
		return m, nil
	case toolStartMsg:
		m.appendToolBlock(msg.name, msg.args, msg.command, "", true, false)
		m.toolLines = append(m.toolLines, toolLine(msg.name, msg.args, msg.command, "", true, false))
		m.syncChatView()
		m.syncToolView()
		return m, nil
	case assistantSegmentEndMsg:
		m.finalizeLastAssistant(time.Now())
		return m, nil
	case toolEndMsg:
		m.finishToolBlock(msg.name, msg.args, msg.command, msg.result, msg.isError)
		m.toolLines = m.toolLines[:0]
		for _, b := range m.chat {
			if b.role == "tool" {
				preview := b.toolResult
				if preview == "" && b.toolRunning {
					preview = "…"
				}
				m.toolLines = append(m.toolLines, toolLine(b.toolName, b.toolArgs, b.toolCommand, preview, b.toolRunning, b.toolError))
			}
		}
		m.syncChatView()
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
		m.clearPlanningBlock()
		now := time.Now()
		m.finalizeLastAssistant(now)
		for i := range m.chat {
			if m.chat[i].role == "tool" && m.chat[i].toolRunning {
				m.chat[i].toolRunning = false
			}
		}
		m.applyTurnMetrics(msg.result)
		if msg.err != nil {
			m.errLine = msg.err.Error()
		} else {
			m.errLine = ""
		}
		m.refreshStatus()
		m.syncChatView()
		return m, tea.Batch(m.listenPrompt(), statusTick())

	case statusRefreshMsg:
		m.refreshStatus()
		m.syncChatView()
		return m, statusTick()

	case exitConfirmTimeoutMsg:
		if m.exitConfirmPending && time.Since(m.exitConfirmArmedAt) >= exitConfirmTimeout {
			m.clearExitConfirm()
		}
		return m, nil

	case promptRequestMsg:
		m.prompt = &msg.req
		m.overlay = overlayPrompt
		m.overlayText = fmt.Sprintf("Allow %s?\n%s\n[y] yes  [n] no", msg.req.Tool, truncate(msg.req.Summary, 300))
		return m, m.listenPrompt()

	case overlayCloseMsg:
		m.overlay = overlayNone
		m.overlayText = ""
		m.refreshStatus()
		m.syncChatView()
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
		if m.overlay == overlayResume {
			return m, nil
		}
		line := strings.TrimSpace(m.input.Value())
		if line != "" {
			m.input.Reset()
			m.overlay = overlayNone
			m.complete = nil
			m.clearResumePicker()
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

func isExitConfirmKey(s string) bool {
	return s == "ctrl+c" || s == "ctrl+d"
}

func exitConfirmHintFor(key string) string {
	if key == "ctrl+c" {
		return "Press Ctrl+C again to exit"
	}
	return "Press Ctrl+D again to exit"
}

func (m *model) handleExitConfirmKey(key string) (tea.Model, tea.Cmd) {
	if m.exitConfirmPending {
		if m.exitConfirmKey != key {
			return m, nil
		}
		return m, tea.Quit
	}
	m.exitConfirmPending = true
	m.exitConfirmKey = key
	m.exitConfirmArmedAt = time.Now()
	m.errLine = exitConfirmHintFor(key)
	return m, exitConfirmTimeoutTick()
}

func (m *model) clearExitConfirm() {
	if !m.exitConfirmPending {
		return
	}
	hint := exitConfirmHintFor(m.exitConfirmKey)
	m.exitConfirmPending = false
	m.exitConfirmKey = ""
	if m.errLine == hint {
		m.errLine = ""
	}
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
	case "down":
		if m.completeIdx < len(m.complete)-1 {
			m.completeIdx++
		}
		m.renderCompleteOverlay()
		return true
	case "tab":
		if len(m.complete) > 0 {
			m.applyCompletion(0)
		}
		return true
	case "enter":
		if m.completionReadyToSubmit() {
			return false
		}
		if len(m.complete) > 0 {
			m.applyCompletion(m.completeIdx)
		}
		return true
	case "esc":
		m.overlay = overlayNone
		m.complete = nil
		return true
	}
	return false
}

func (m *model) applyCompletion(idx int) {
	if len(m.complete) == 0 {
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.complete) {
		idx = len(m.complete) - 1
	}
	c := m.complete[idx]
	m.input.SetValue("/" + c.Name + " ")
	m.input.CursorEnd()
	m.overlay = overlayNone
	m.complete = nil
}

// completionReadyToSubmit reports whether Enter should run the current slash
// line instead of inserting another completion suffix. Partial prefixes like
// "/c" are not ready; a full registered command such as "/context" is.
func (m *model) completionReadyToSubmit() bool {
	val := strings.TrimSpace(m.input.Value())
	cmd, _, ok := slash.Parse(val)
	if !ok {
		return false
	}
	if _, known := slash.Lookup(cmd); !known {
		return false
	}
	base := "/" + cmd
	return val == base || strings.HasPrefix(val, base+" ")
}

func (m *model) updateCompletion() {
	val := m.input.Value()
	trimmed := strings.TrimSpace(val)
	if cmd, args, ok := slash.Parse(trimmed); ok && cmd == "resume" {
		m.updateResumePicker(strings.TrimSpace(args))
		return
	}
	if m.overlay == overlayResume {
		m.clearResumePicker()
	}
	if !strings.HasPrefix(trimmed, "/") {
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
		case "resume":
			id := strings.TrimSpace(args)
			if id == "" {
				return m.fetchResumeList()
			}
			return m.resumeSession(id)
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
	buf.WriteString("\nKeyboard shortcuts:\n\n")
	buf.WriteString("  Ctrl+O       Expand/collapse tool args & result in chat\n")
	buf.WriteString("  Ctrl+R       Expand/collapse all reasoning blocks\n")
	buf.WriteString("  Ctrl+T       Toggle tool log panel\n")
	buf.WriteString("  Ctrl+L       Context usage panel\n")
	buf.WriteString("  Ctrl+C       Cancel turn (while running) / exit (press twice when idle)\n")
	buf.WriteString("  Ctrl+D       Exit (press twice when idle)\n")
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

func (m *model) appendPlanningBlock() {
	m.clearPlanningBlock()
	m.chat = append(m.chat, chatBlock{
		role:              "planning",
		streaming:         true,
		planningStartedAt: time.Now(),
	})
}

func (m *model) clearPlanningBlock() {
	for i := len(m.chat) - 1; i >= 0; i-- {
		if m.chat[i].role == "planning" {
			m.chat = append(m.chat[:i], m.chat[i+1:]...)
			return
		}
	}
}

func (m *model) needsPlanningTick() bool {
	if !m.running || len(m.chat) == 0 {
		return false
	}
	return m.chat[len(m.chat)-1].role == "planning"
}

func (m *model) appendToolBlock(name, args, command, result string, running, isError bool) {
	if len(m.chat) > 0 {
		last := &m.chat[len(m.chat)-1]
		if last.role == "assistant" {
			last.finalizeReasoning(time.Now())
			last.streaming = false
		}
	}
	m.chat = append(m.chat, chatBlock{
		role:        "tool",
		toolName:    name,
		toolArgs:    args,
		toolCommand: command,
		toolResult:  result,
		toolRunning: running,
		toolError:   isError,
	})
}

func (m *model) finishToolBlock(name, args, command, result string, isError bool) {
	for i := len(m.chat) - 1; i >= 0; i-- {
		if m.chat[i].role != "tool" || !m.chat[i].toolRunning {
			continue
		}
		m.chat[i].toolName = name
		m.chat[i].toolArgs = args
		m.chat[i].toolCommand = command
		m.chat[i].toolResult = result
		m.chat[i].toolRunning = false
		m.chat[i].toolError = isError
		return
	}
	m.appendToolBlock(name, args, command, result, false, isError)
}

func (m *model) finalizeLastAssistant(at time.Time) {
	for i := len(m.chat) - 1; i >= 0; i-- {
		if m.chat[i].role != "assistant" {
			continue
		}
		m.chat[i].finalizeReasoning(at)
		m.chat[i].streaming = false
		return
	}
}

func (m *model) appendAssistantContent(s string) {
	blk := m.ensureStreamingAssistant()
	blk.finalizeReasoning(time.Now())
	blk.streaming = true
	blk.appendContent(s)
}

// applyTurnMetrics attaches final turn stats to the assistant block that holds the
// visible reply. After tool rounds there may be a trailing empty assistant block;
// prefer the last one that actually has content.
func (m *model) applyTurnMetrics(result *agent.TurnResult) {
	if result == nil {
		return
	}
	idx := -1
	for i := len(m.chat) - 1; i >= 0; i-- {
		if m.chat[i].role != "assistant" {
			continue
		}
		if idx < 0 {
			idx = i
		}
		if m.chat[i].content.Len() > 0 {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	if result.FinalReasoningDuration > 0 && m.chat[idx].reasoningDuration == 0 {
		m.chat[idx].reasoningDuration = result.FinalReasoningDuration
	}
	if result.TurnDuration > 0 {
		m.chat[idx].turnDuration = result.TurnDuration
	}
}

func (m *model) ensureStreamingAssistant() *chatBlock {
	needNew := len(m.chat) == 0
	if !needNew {
		switch m.chat[len(m.chat)-1].role {
		case "tool", "user", "planning":
			needNew = true
		case "assistant":
			// Keep streaming into the current assistant until a tool/user/planning
			// block breaks the segment. Do not split on content+!streaming alone — that
			// created a trailing empty assistant and hid turn duration on the wrong block.
		default:
			needNew = true
		}
	}
	if needNew && len(m.chat) > 0 {
		if prev := &m.chat[len(m.chat)-1]; prev.role == "assistant" {
			prev.finalizeReasoning(time.Now())
		}
	}
	if needNew {
		m.chat = append(m.chat, chatBlock{role: "assistant", streaming: true, reasoningOpen: m.reasoningAll})
	}
	return &m.chat[len(m.chat)-1]
}

func (m *model) appendAssistantReasoning(s string) bool {
	blk := m.ensureStreamingAssistant()
	started := false
	if blk.reasoningStartedAt.IsZero() {
		blk.reasoningStartedAt = time.Now()
		started = true
	}
	blk.appendReasoning(s)
	return started
}

func (m *model) needsThinkingTick() bool {
	if !m.running || len(m.chat) == 0 {
		return false
	}
	blk := m.chat[len(m.chat)-1]
	return blk.role == "assistant" && !blk.reasoningStartedAt.IsZero() && blk.reasoningEndedAt.IsZero()
}

func (m *model) thinkingElapsed() time.Duration {
	if len(m.chat) == 0 {
		return 0
	}
	blk := m.chat[len(m.chat)-1]
	if blk.reasoningStartedAt.IsZero() {
		return 0
	}
	end := blk.reasoningEndedAt
	if end.IsZero() {
		end = time.Now()
	}
	d := end.Sub(blk.reasoningStartedAt)
	if d < 0 {
		return 0
	}
	return d
}

func (m *model) nextThinkingTickCmd() tea.Cmd {
	if m.thinkingElapsed() < thinkingFineDuration {
		return thinkingTickAfter(thinkingFineTick)
	}
	return thinkingTickAfter(thinkingCoarseTick)
}

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
		footerH         = 1
		inputFrameH     = 3
		gapAfterChat    = 1
		gapAfterTool    = 1
		gapAfterInput   = 1
		maxToolLines    = 5
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

func (m *model) handleResumeKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "up":
		m.resumeMoveSelection(-1)
		return true
	case "down", "tab":
		m.resumeMoveSelection(1)
		return true
	case "pgup":
		m.resumePageSelection(-1)
		return true
	case "pgdown":
		m.resumePageSelection(1)
		return true
	case "enter":
		// Handled before handleResumeKey in the KeyMsg block.
		return true
	case "esc":
		m.clearResumePicker()
		return true
	}
	return false
}

func (m *model) clearResumePicker() {
	m.overlay = overlayNone
	m.resumeSessions = nil
	m.resumeFilter = ""
	m.resumeIdx = 0
	m.resumeScroll = 0
	m.overlayText = ""
}

func (m *model) updateResumePicker(filter string) {
	// textinput emits updates on cursor blink; do not reset selection unless filter changed.
	if m.overlay == overlayResume && filter == m.resumeFilter && len(m.resumeSessions) > 0 {
		return
	}
	filterChanged := m.resumeFilter != filter
	m.resumeFilter = filter

	list, err := m.listResumeSessions(filter)
	if err != nil {
		m.errLine = err.Error()
		m.clearResumePicker()
		return
	}
	m.resumeSessions = list
	if filterChanged || len(list) == 0 {
		m.resumeIdx = 0
		m.resumeScroll = 0
	} else if m.resumeIdx >= len(list) {
		m.resumeIdx = len(list) - 1
	}
	if len(list) == 0 {
		m.overlayText = "No matching sessions."
		m.overlay = overlayResume
		return
	}
	m.overlay = overlayResume
	m.renderResumeOverlay()
}

func (m *model) fetchResumeList() tea.Cmd {
	d := m.deps
	return func() tea.Msg {
		list, err := d.Store.ListSessions(context.Background(), resumeListMax)
		return resumeListMsg{sessions: list, err: err}
	}
}

func (m *model) loadInitialHistory() tea.Cmd {
	if m.deps == nil || m.deps.Store == nil || m.sessionID == "" {
		return nil
	}
	d := m.deps
	sid := m.sessionID
	reasoningOpen := m.reasoningAll
	return func() tea.Msg {
		chat, err := loadSessionChat(d.Store, sid, reasoningOpen)
		return historyLoadedMsg{chat: chat, err: err}
	}
}

func (m *model) resumeSession(id string) tea.Cmd {
	d := m.deps
	reasoningOpen := m.reasoningAll
	return func() tea.Msg {
		ctx := context.Background()
		if _, err := d.Store.Get(ctx, id); err != nil {
			return sessionResumedMsg{err: err}
		}
		chat, err := loadSessionChat(d.Store, id, reasoningOpen)
		if err != nil {
			return sessionResumedMsg{err: err}
		}
		return sessionResumedMsg{sessionID: id, chat: chat}
	}
}

func (m *model) refreshStatus() {
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
		inputBody = styleFooterHint.Render("Working…")
	}
	b.WriteString(renderInputFrame(m.width, inputBody))
	b.WriteString("\n")

	footerLeft := "? for shortcuts · Ctrl+O tool details"
	if m.toolDetailsVisible {
		footerLeft += " (on)"
	}
	if m.running {
		footerLeft = "Ctrl+C cancel · Ctrl+D exit (twice) · Ctrl+R reasoning · Ctrl+O tool details"
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
