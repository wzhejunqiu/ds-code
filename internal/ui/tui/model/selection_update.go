package model

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/ui/clipboard"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/view"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/scroll"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/selection"
)

const (
	doubleClickWindow = 400 * time.Millisecond
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
	if len(m.lineCatalog.PlainLines()) > 0 {
		m.plainLines = m.lineCatalog.PlainLines()
		return
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

func (m *Model) mapMousePoint(mouse tea.Mouse) (selection.Point, bool) {
	if mouse.Y < 0 || mouse.X < 0 {
		return selection.Point{}, false
	}
	chatH := m.chatVP.Height()
	if mouse.Y < chatH {
		m.selTarget = selTargetChat
		line := mouse.Y + m.chatScrollY
		if line < 0 {
			line = 0
		}
		col := mouse.X
		if col < 0 {
			col = 0
		}
		return selection.Point{Line: line, Col: col}, true
	}
	if m.ToolOpen && m.toolVP.Height() > 0 {
		toolTop := chatH + 1
		if mouse.Y >= toolTop && mouse.Y < toolTop+m.toolVP.Height() {
			m.selTarget = selTargetTool
			line := (mouse.Y - toolTop) + m.toolVP.YOffset()
			if line < 0 {
				line = 0
			}
			col := mouse.X
			if col < 0 {
				col = 0
			}
			return selection.Point{Line: line, Col: col}, true
		}
	}
	return selection.Point{}, false
}

func (m *Model) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if !m.chatInteractionEnabled() {
		return m, nil
	}
	delta := m.scroll.ComputeWheelStep(msg, time.Now())
	if delta == 0 {
		return m, nil
	}
	mouse := msg.Mouse()
	chatH := m.chatVP.Height()
	if mouse.Y < chatH {
		return m, m.queueWheelScroll(scroll.TargetChat, delta)
	}
	if m.ToolOpen && m.toolVP.Height() > 0 {
		toolTop := chatH + 1
		if mouse.Y >= toolTop && mouse.Y < toolTop+m.toolVP.Height() {
			return m, m.queueWheelScroll(scroll.TargetTool, delta)
		}
	}
	return m, nil
}

func (m *Model) handleMouse(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		return m.handleMouseWheel(msg)
	case tea.MouseClickMsg:
		if !m.chatInteractionEnabled() {
			return m, nil
		}
		pt, ok := m.mapMousePoint(msg.Mouse())
		if !ok {
			return m, nil
		}
		if msg.Button == tea.MouseLeft {
			if m.isDoubleClick(msg) {
				m.expandWordSelection(pt)
				return m, nil
			}
			m.selDragging = true
			m.selRange = selection.Range{Start: pt, End: pt}
			m.lastClickAt = time.Now()
			m.lastClickPt = pt
		}
	case tea.MouseMotionMsg:
		if !m.chatInteractionEnabled() {
			return m, nil
		}
		if m.selDragging && msg.Button == tea.MouseLeft {
			pt, ok := m.mapMousePoint(msg.Mouse())
			if ok {
				m.selRange.End = pt
			}
		}
	case tea.MouseReleaseMsg:
		if !m.chatInteractionEnabled() {
			return m, nil
		}
		if msg.Button == tea.MouseLeft && m.selDragging {
			m.selDragging = false
			pt, ok := m.mapMousePoint(msg.Mouse())
			if ok {
				m.selRange.End = pt
			}
			if m.copyOnSelectEnabled() && m.selRange.Active() {
				text := selection.Extract(m.selectionLines(), m.selRange)
				return m, m.copyText(text)
			}
		}
	}
	return m, nil
}

func (m *Model) isDoubleClick(msg tea.MouseClickMsg) bool {
	if m.lastClickAt.IsZero() {
		return false
	}
	mouse := msg.Mouse()
	pt, ok := m.mapMousePoint(mouse)
	if !ok {
		return false
	}
	return time.Since(m.lastClickAt) < doubleClickWindow &&
		pt.Line == m.lastClickPt.Line && pt.Col == m.lastClickPt.Col
}

func (m *Model) expandWordSelection(pt selection.Point) {
	lines := m.selectionLines()
	if pt.Line < 0 || pt.Line >= len(lines) {
		return
	}
	line := lines[pt.Line]
	start, end := selection.WordBounds(line, pt.Col)
	m.selRange = selection.Range{
		Start: selection.Point{Line: pt.Line, Col: start},
		End:   selection.Point{Line: pt.Line, Col: end},
	}
	m.selDragging = false
}

func (m *Model) handleManualCopyKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
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

func (m *Model) handleSelectionKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if !m.selRange.Active() {
		return nil, false
	}
	if !msg.Mod.Contains(tea.ModShift) {
		return nil, false
	}
	var delta int
	switch msg.String() {
	case "up", "shift+up":
		delta = -1
	case "down", "shift+down":
		delta = 1
	case "home", "shift+home":
		m.extendSelectionToLineEdge(true)
		return nil, true
	case "end", "shift+end":
		m.extendSelectionToLineEdge(false)
		return nil, true
	default:
		return nil, false
	}
	lines := m.selectionLines()
	end := m.selRange.End
	newLine := end.Line + delta
	if newLine < 0 {
		newLine = 0
	}
	if newLine >= len(lines) {
		newLine = len(lines) - 1
	}
	if newLine < 0 {
		return nil, true
	}
	m.selRange.End = selection.Point{Line: newLine, Col: end.Col}
	if delta < 0 && newLine < m.chatScrollY {
		return m.jumpViewport(&m.chatVP, delta), true
	}
	if delta > 0 && newLine >= m.chatScrollY+m.chatVP.Height() {
		return m.jumpViewport(&m.chatVP, delta), true
	}
	return nil, true
}

func (m *Model) extendSelectionToLineEdge(home bool) {
	lines := m.selectionLines()
	end := m.selRange.End
	if end.Line < 0 || end.Line >= len(lines) {
		return
	}
	col := 0
	if !home {
		col = len(lines[end.Line])
	}
	m.selRange.End = selection.Point{Line: end.Line, Col: col}
}

func (m *Model) copySelection() tea.Cmd {
	if !m.selRange.Active() {
		return nil
	}
	text := selection.Extract(m.selectionLines(), m.selRange)
	return m.copyText(text)
}

func (m *Model) copyText(text string) tea.Cmd {
	if text == "" {
		return nil
	}
	return tea.Batch(
		tea.SetClipboard(text),
		func() tea.Msg {
			if err := clipboard.Write(text); err != nil {
				return copyResultMsg{err: err}
			}
			return copyResultMsg{}
		},
	)
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
