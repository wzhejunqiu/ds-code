package model

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hejunqiu/ds-code/internal/ui/tui/component"
	"github.com/hejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/session"
	subagentui "github.com/hejunqiu/ds-code/internal/ui/tui/model/subagent"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/view"
	"github.com/hejunqiu/ds-code/internal/ui/tui/style"
)

// Model is the Bubble Tea model for the ds-code TUI.
type Model struct {
	state.State
	chatVP         viewport.Model
	toolVP         viewport.Model
	input          textinput.Model
	completePicker component.Picker
	resumePicker   component.Picker
	subagentPicker component.Picker
}

// New builds a Model from runtime dependencies.
func New(d *deps.Deps) *Model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 40
	ti.Prompt = ""
	ti.Placeholder = ""
	ti.Cursor.Style = lipgloss.NewStyle().Reverse(true)
	ti.TextStyle = style.InputText

	m := &Model{
		State: state.State{
			Deps:      d,
			SessionID: d.SessionID,
		},
		chatVP: viewport.New(40, 10),
		toolVP: viewport.New(40, 4),
		input:  ti,
	}
	view.RefreshStatus(&m.State)
	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.listenPrompt(),
		statusTick(),
		session.LoadInitialHistory(&m.State),
	)
}

func (m *Model) listenPrompt() tea.Cmd {
	if m.Deps == nil || m.Deps.PromptCh == nil {
		return nil
	}
	ch := m.Deps.PromptCh
	return func() tea.Msg {
		req, ok := <-ch
		if !ok {
			return nil
		}
		return msg.PromptRequestMsg{Req: req}
	}
}

func (m *Model) syncChatView() {
	view.SyncChat(&m.State, &m.chatVP, &m.toolVP, &m.input)
}

func (m *Model) syncToolView() {
	view.SyncTool(&m.State, &m.chatVP, &m.toolVP, &m.input)
}

func (m *Model) syncAllViews() {
	m.syncChatView()
	m.syncToolView()
	if m.Overlay == state.OverlayResume && len(m.ResumeSessions) > 0 {
		session.SyncResumePicker(&m.State, &m.resumePicker)
	}
	if m.Overlay == state.OverlaySubagentList && m.Subagents.Len() > 0 {
		subagentui.SyncListPicker(&m.State, &m.subagentPicker)
	}
}

func (m *Model) layout() {
	view.Layout(&m.State, &m.chatVP, &m.toolVP, &m.input)
}

func (m *Model) refreshStatus() {
	view.RefreshStatus(&m.State)
}

func (m *Model) View() string {
	return view.Render(&m.State, &m.chatVP, &m.toolVP, m.input)
}
