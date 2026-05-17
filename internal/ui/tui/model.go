package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/ui/slash"
	"github.com/hejunqiu/ds-code/internal/ui/tui/component"
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

	// Slash completion: domain rows parallel to completePicker.Items indices.
	complete       []slash.Command
	completePicker component.Picker

	// /resume picker: summaries parallel to resumePicker.Items indices.
	resumeSessions []session.Summary
	resumePicker   component.Picker
	resumeFilter   string // skips re-list when only textinput cursor blinked

	prompt *permission.PromptRequest

	running            bool
	turnCancel         context.CancelFunc
	turnEscPending     bool // Esc pressed before turnStartedMsg delivered cancel func
	reasoningAll       bool
	toolDetailsVisible bool // Ctrl+O: expand tool args/result in chat

	statusRight string
	errLine     string

	headerSession session.Session
	hasSession    bool

	exitConfirmPending bool
	exitConfirmKey     string // "ctrl+c" or "ctrl+d"
	exitConfirmArmedAt time.Time
}

const (
	exitConfirmTimeout = time.Second
	runningTurnHint    = "Press Esc to cancel the current turn"
)

const (
	thinkingFineTick   = 100 * time.Millisecond
	thinkingCoarseTick = time.Second
)

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

func thinkingTickAfter(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg { return thinkingTickMsg{} })
}

func exitConfirmTimeoutTick() tea.Cmd {
	return tea.Tick(exitConfirmTimeout, func(time.Time) tea.Msg { return exitConfirmTimeoutMsg{} })
}

func (m *model) listenPrompt() tea.Cmd {
	if m.deps == nil || m.deps.PromptCh == nil {
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
