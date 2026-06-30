package model

import (
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/component"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/header"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/markdown"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	subagentui "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/subagent"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/tcase"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/view"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/scroll"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/selection"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/style"
)

// Model is the Bubble Tea model for the ds-code TUI.
type Model struct {
	state.State
	chatVP            viewport.Model
	toolVP            viewport.Model
	input             textinput.Model
	completePicker    component.Picker
	resumePicker      component.Picker
	subagentPicker    component.Picker
	tcasePicker       component.Picker
	chatRenderCache   chat.RenderCache
	mdSegmentCache    markdown.SegmentCache
	headerCache       view.HeaderCache
	chatSyncScheduled bool

	plainLines     []string
	toolPlainLines []string
	lineCatalog    chat.LineCatalog
	chatScrollY    int
	selRange       selection.Range
	selDragging    bool
	selTarget      int
	lastClickAt    time.Time
	lastClickPt    selection.Point

	scroll          scroll.Controller
	scrollDeferSync bool
	mouseLeakBuf    string
}

const (
	selTargetNone = iota
	selTargetChat
	selTargetTool
)

// New builds a Model from runtime dependencies.
func New(d *deps.Deps) *Model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 0
	ti.SetWidth(40)
	ti.Prompt = ""
	ti.Placeholder = ""
	inputStyles := textinput.DefaultLightStyles()
	inputStyles.Focused.Text = style.InputText
	inputStyles.Blurred.Text = style.InputText
	ti.SetStyles(inputStyles)

	m := &Model{
		State: state.State{
			Deps:           d,
			SessionID:      d.SessionID,
			StartupNotices: append([]header.Notice(nil), d.StartupNotices...),
		},
		chatVP: viewport.New(viewport.WithWidth(40), viewport.WithHeight(10)),
		toolVP: viewport.New(viewport.WithWidth(40), viewport.WithHeight(4)),
		input:  ti,
		scroll: scroll.NewController(),
	}
	view.RefreshStatus(&m.State)
	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.listenPrompt(),
		m.listenWebFetchPrompt(),
		session.LoadInitialHistory(&m.State),
		m.scheduleNoticeScroll(),
		func() tea.Msg { return tea.RequestWindowSize() },
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

func (m *Model) listenWebFetchPrompt() tea.Cmd {
	if m.Deps == nil || m.Deps.WebFetchPromptCh == nil {
		return nil
	}
	ch := m.Deps.WebFetchPromptCh
	return func() tea.Msg {
		req, ok := <-ch
		if !ok {
			return nil
		}
		return msg.WebFetchPromptRequestMsg{Req: req}
	}
}

func (m *Model) syncToolView() {
	view.SyncTool(&m.State, &m.chatVP, &m.toolVP, &m.input, m.syncCaches())
	m.updateToolPlainLines()
}

func (m *Model) syncAllViews() {
	m.syncChatView()
	m.syncToolView()
	if m.Overlay == state.OverlayResume && m.ResumeSessions != nil {
		session.SyncResumePicker(&m.State, &m.resumePicker)
	}
	if m.Overlay == state.OverlaySubagentList && m.Subagents.Len() > 0 {
		subagentui.SyncListPicker(&m.State, &m.subagentPicker)
	}
	if m.Overlay == state.OverlayTCase && len(m.TCaseItems) > 0 {
		tcase.SyncPicker(&m.State, &m.tcasePicker)
	}
}

func (m *Model) View() tea.View {
	var sel *view.SelectionOverlay
	if m.selRange.Active() {
		sel = &view.SelectionOverlay{
			ChatPlain:   m.plainLines,
			ChatRange:   m.selRange,
			ToolPlain:   m.toolPlainLines,
			ToolRange:   m.selRange,
			ChatActive:  m.selTarget == selTargetChat,
			ToolActive:  m.selTarget == selTargetTool,
			ChatScrollY: m.effectiveChatScrollY(),
		}
	}
	v := tea.NewView(view.Render(&m.State, &m.chatVP, &m.toolVP, &m.input, sel))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
