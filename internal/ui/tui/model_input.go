package tui

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	uipkg "github.com/hejunqiu/ds-code/internal/ui"
	"github.com/hejunqiu/ds-code/internal/ui/slash"
)

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

	m.chat = append(m.chat, chatBlock{role: chatRoleUser})
	m.chat[len(m.chat)-1].content.WriteString(line)
	m.chat = append(m.chat, chatBlock{role: chatRoleAssistant, streaming: true, reasoningOpen: m.reasoningAll})
	m.syncChatView()
	m.running = true
	m.turnEscPending = false
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
	buf.WriteString("  Esc          Cancel turn (while running)\n")
	buf.WriteString("  Ctrl+C       Exit when idle (press twice); while running, shows Esc hint\n")
	buf.WriteString("  Ctrl+D       Same as Ctrl+C\n")
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
		return slashOutputMsg{text: "Unknown command (try /help)"}
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
