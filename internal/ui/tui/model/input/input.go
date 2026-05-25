package input

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	uipkg "github.com/wzhejunqiu/ds-code/internal/ui"
	"github.com/wzhejunqiu/ds-code/internal/ui/slash"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/component"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/overlay"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/turn"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/subagent"
)

var completePickerKeys = component.PickerKeyOpts{Tab: component.PickerTabSelectFirst}

func SyncCompleteOverlay(s *state.State, picker *component.Picker) {
	items := make([]string, len(s.Complete))
	for i, c := range s.Complete {
		items[i] = fmt.Sprintf("/%s — %s", c.Name, c.Description)
	}
	picker.SetItems(items)
	s.OverlayText = picker.View()
}

func ClearCompletePicker(s *state.State, picker *component.Picker) {
	s.Complete = nil
	s.CompleteFilterKey = ""
	picker.Clear()
}

func HandleCompleteKey(s *state.State, msg tea.KeyMsg, picker *component.Picker, inputValue string, setInput func(string), cursorEnd func()) bool {
	if CompletionReadyToSubmit(inputValue) && msg.Type == tea.KeyEnter {
		return false
	}
	if len(s.Complete) > 0 {
		SyncCompleteOverlay(s, picker)
	}
	action, handled := picker.HandleKey(msg, completePickerKeys)
	if !handled {
		return false
	}
	switch action {
	case component.PickerKeyCancel:
		overlay.Dismiss(s)
		ClearCompletePicker(s, picker)
	case component.PickerKeyConfirmFirst:
		applyCompletion(s, picker, 0, setInput, cursorEnd)
	case component.PickerKeyConfirm:
		applyCompletion(s, picker, picker.Cursor, setInput, cursorEnd)
	default:
		SyncCompleteOverlay(s, picker)
	}
	return true
}

func applyCompletion(s *state.State, picker *component.Picker, idx int, setInput func(string), cursorEnd func()) {
	if len(s.Complete) == 0 {
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s.Complete) {
		idx = len(s.Complete) - 1
	}
	setInput("/" + s.Complete[idx].Name + " ")
	cursorEnd()
	s.Overlay = state.OverlayNone
	ClearCompletePicker(s, picker)
}

func CompletionReadyToSubmit(inputValue string) bool {
	val := strings.TrimSpace(inputValue)
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

func slashCommandsEqual(a, b []slash.Command) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name {
			return false
		}
	}
	return true
}

func UpdateCompletion(s *state.State, inputValue string, picker *component.Picker, resumePicker *component.Picker) tea.Cmd {
	val := inputValue
	trimmed := strings.TrimSpace(val)
	if cmd, args, ok := slash.Parse(trimmed); ok && cmd == "resume" {
		ClearCompletePicker(s, picker)
		return session.ScheduleResumeFilter(s, strings.TrimSpace(args), resumePicker)
	}
	if s.Overlay == state.OverlayResume {
		session.ClearResumePicker(s, resumePicker)
	}
	if !strings.HasPrefix(trimmed, "/") {
		s.CompleteFilterKey = ""
		if s.Overlay == state.OverlayComplete {
			s.Overlay = state.OverlayNone
			ClearCompletePicker(s, picker)
		}
		return nil
	}
	filtered := slash.FilterCommands(val)
	if s.Overlay == state.OverlayComplete && val == s.CompleteFilterKey && slashCommandsEqual(s.Complete, filtered) {
		return nil
	}
	filterChanged := val != s.CompleteFilterKey
	s.CompleteFilterKey = val
	s.Complete = filtered
	if filterChanged || len(filtered) == 0 {
		picker.ResetSelection()
	} else {
		picker.ClampSelection()
	}
	if len(s.Complete) == 0 {
		s.Overlay = state.OverlayNone
		ClearCompletePicker(s, picker)
		return nil
	}
	s.Overlay = state.OverlayComplete
	SyncCompleteOverlay(s, picker)
	return nil
}

func SubmitLine(s *state.State, line string, syncChat, syncTool func()) tea.Cmd {
	if line == "/exit" || line == "/quit" {
		return overlay.QuitAfterWait(s)
	}
	if cmd, args, ok := slash.Parse(line); ok {
		if c, handled := trySubmitDevSlash(cmd, args); handled {
			return c
		}
		if c, handled := trySubmitTUITestSlash(cmd, args, s, syncChat, syncTool); handled {
			return c
		}
		switch cmd {
		case "help":
			return ShowHelp()
		case "context":
			if strings.TrimSpace(args) == "--json" {
				return ShowContextJSON(s)
			}
			return ShowContext(s)
		case "resume":
			id := strings.TrimSpace(args)
			if id == "" {
				return session.FetchList(s)
			}
			return session.ResumeSession(s, id)
		default:
			return RunSlash(s, line)
		}
	}

	if s.MainChat == nil {
		s.MainChat = s.Chat
	}
	s.MainChat = append(s.MainChat, chat.Block{Role: chat.RoleUser})
	s.MainChat[len(s.MainChat)-1].AppendContent(line)
	s.Subagents = subagent.Registry{}
	s.SubagentNav = state.SubagentNavMain
	s.ViewingSubagentID = ""
	s.Overlay = state.OverlayNone
	s.OverlayText = ""
	s.BindMainChat(s.MainChat)
	turn.AppendPlanningBlock(s)
	syncChat()
	s.Running = true
	s.TurnEscPending = false
	s.ErrLine = ""
	s.MainToolLines = nil
	s.ToolLines = nil
	syncTool()

	d := *s.Deps
	d.SessionID = s.SessionID
	events := s.Deps.Events
	return func() tea.Msg {
		go turn.RunAsync(d, line, events, &s.TurnWG)
		return nil
	}
}

func ShowHelp() tea.Cmd {
	var buf bytes.Buffer
	slash.WriteHelp(&buf)
	buf.WriteString("\nKeyboard shortcuts:\n\n")
	buf.WriteString("  Ctrl+O       Expand/collapse tool args & result in chat\n")
	buf.WriteString("  Ctrl+R       Expand/collapse all reasoning blocks\n")
	buf.WriteString("  Ctrl+T       Toggle tool log panel\n")
	buf.WriteString("  Ctrl+L       Context usage panel\n")
	buf.WriteString("  ↓            Subagent list (when task subagents exist)\n")
	buf.WriteString("  ←            Back from subagent list/detail\n")
	buf.WriteString("  Esc          Cancel turn (while running)\n")
	buf.WriteString("  Ctrl+C       Exit when idle (press twice); while running, shows Esc hint\n")
	buf.WriteString("  Ctrl+D       Same as Ctrl+C\n")
	return func() tea.Msg { return msg.HelpOverlayMsg{Text: buf.String()} }
}

func RunSlash(s *state.State, line string) tea.Cmd {
	d := s.Deps
	return func() tea.Msg {
		var buf bytes.Buffer
		sid := s.SessionID
		activeAgentType := s.Subagents.ResolveActiveAgentType(s.ViewingSubagentID)
		handled, err := d.HandleSlash(context.Background(), &buf, &sid, line, activeAgentType)
		s.SessionID = sid
		d.SessionID = sid
		if err != nil {
			return msg.TurnDoneMsg{Err: err}
		}
		if handled {
			if out := strings.TrimSpace(buf.String()); out != "" {
				return msg.SlashOutputMsg{Text: out}
			}
			return msg.OverlayCloseMsg{}
		}
		return msg.SlashOutputMsg{Text: "Unknown command (try /help)"}
	}
}

func ShowContext(s *state.State) tea.Cmd {
	d := s.Deps
	return func() tea.Msg {
		ctx := context.Background()
		sess, err := d.Store.Get(ctx, s.SessionID)
		if err != nil {
			return msg.TurnDoneMsg{Err: err}
		}
		view, err := d.Context.BuildAPIContext(ctx, s.SessionID)
		if err != nil {
			return msg.TurnDoneMsg{Err: err}
		}
		panel, err := uipkg.BuildContextPanelData(ctx, d.Cfg, d.Store, d.Subagent, sess, view)
		if err != nil {
			return msg.TurnDoneMsg{Err: err}
		}
		return msg.ContextOverlayMsg{Text: uipkg.FormatContextPanel(panel)}
	}
}

func ShowContextJSON(s *state.State) tea.Cmd {
	d := s.Deps
	return func() tea.Msg {
		ctx := context.Background()
		sess, err := d.Store.Get(ctx, s.SessionID)
		if err != nil {
			return msg.TurnDoneMsg{Err: err}
		}
		view, err := d.Context.BuildAPIContext(ctx, s.SessionID)
		if err != nil {
			return msg.TurnDoneMsg{Err: err}
		}
		panel, err := uipkg.BuildContextPanelData(ctx, d.Cfg, d.Store, d.Subagent, sess, view)
		if err != nil {
			return msg.TurnDoneMsg{Err: err}
		}
		text, err := uipkg.FormatContextJSON(panel)
		if err != nil {
			return msg.TurnDoneMsg{Err: err}
		}
		return msg.ContextOverlayMsg{Text: text}
	}
}
