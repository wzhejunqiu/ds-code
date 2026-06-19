package model

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/ui/clipboard"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/view"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/selection"
)

type copyResultMsg struct {
	err error
}

type copyToastClearMsg struct{}

func (m *Model) copyOnSelectEnabled() bool {
	if m.Deps == nil || m.Deps.Cfg == nil {
		return true
	}
	return m.Deps.Cfg.TUI.CopyOnSelect
}

func (m *Model) updatePlainLines() {
	innerW := m.Width - 2
	if innerW < 10 {
		innerW = 10
	}
	m.plainLines = view.ChatPlainContent(&m.State, innerW, m.syncCaches())
}

func (m *Model) updateToolPlainLines() {
	plain := selection.StripANSI(strings.Join(m.ToolLines, "\n"))
	m.toolPlainLines = selection.LinesFromContent(plain)
}

func (m *Model) selectionLines() []string {
	if m.selTarget == selTargetTool {
		return m.toolPlainLines
	}
	return m.plainLines
}

func (m *Model) mapMousePoint(msg tea.MouseMsg) (selection.Point, bool) {
	if msg.Y < 0 || msg.X < 0 {
		return selection.Point{}, false
	}
	chatH := m.chatVP.Height
	if msg.Y < chatH {
		m.selTarget = selTargetChat
		line := msg.Y + m.chatVP.YOffset
		if line < 0 {
			line = 0
		}
		col := msg.X
		if col < 0 {
			col = 0
		}
		return selection.Point{Line: line, Col: col}, true
	}
	if m.ToolOpen && m.toolVP.Height > 0 {
		toolTop := chatH + 1
		if msg.Y >= toolTop && msg.Y < toolTop+m.toolVP.Height {
			m.selTarget = selTargetTool
			line := (msg.Y - toolTop) + m.toolVP.YOffset
			if line < 0 {
				line = 0
			}
			col := msg.X
			if col < 0 {
				col = 0
			}
			return selection.Point{Line: line, Col: col}, true
		}
	}
	return selection.Point{}, false
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.Overlay != state.OverlayNone || m.Prompt != nil {
		return m, nil
	}
	pt, ok := m.mapMousePoint(msg)
	if !ok {
		return m, nil
	}

	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			m.selDragging = true
			m.selRange = selection.Range{Start: pt, End: pt}
		}
	case tea.MouseActionMotion:
		if m.selDragging && msg.Button == tea.MouseButtonLeft {
			m.selRange.End = pt
		}
	case tea.MouseActionRelease:
		if msg.Button == tea.MouseButtonLeft && m.selDragging {
			m.selDragging = false
			m.selRange.End = pt
			if m.copyOnSelectEnabled() && m.selRange.Active() {
				text := selection.Extract(m.selectionLines(), m.selRange)
				return m, m.asyncCopy(text)
			}
		}
	}
	return m, nil
}

func (m *Model) handleManualCopyKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.copyOnSelectEnabled() {
		return nil, false
	}
	switch msg.String() {
	case "ctrl+shift+c", "cmd+shift+c":
		if !m.selRange.Active() {
			return nil, true
		}
		return m.copySelection(), true
	default:
		return nil, false
	}
}

func (m *Model) copySelection() tea.Cmd {
	if !m.selRange.Active() {
		return nil
	}
	text := selection.Extract(m.selectionLines(), m.selRange)
	return m.asyncCopy(text)
}

func (m *Model) asyncCopy(text string) tea.Cmd {
	return func() tea.Msg {
		err := clipboard.Write(text)
		return copyResultMsg{err: err}
	}
}

func (m *Model) handleCopyResult(msg copyResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.StatusRight = "复制失败"
	} else {
		m.StatusRight = "已复制"
	}
	return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return copyToastClearMsg{}
	})
}

func (m *Model) handleCopyToastClear() (tea.Model, tea.Cmd) {
	m.refreshStatus()
	return m, nil
}
