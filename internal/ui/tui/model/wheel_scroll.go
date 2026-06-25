package model

import (
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	tuimsg "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/view"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/scroll"
)

const wheelScrollTickEvery = 4 * time.Millisecond

func wheelScrollTickAfter() tea.Cmd {
	return tea.Tick(wheelScrollTickEvery, func(time.Time) tea.Msg {
		return tuimsg.WheelScrollTickMsg{}
	})
}

func (m *Model) viewportForTarget(target scroll.Target) *viewport.Model {
	if target == scroll.TargetTool {
		return &m.toolVP
	}
	return &m.chatVP
}

func (m *Model) clampYOffset(vp *viewport.Model, y int) int {
	totalLines := m.lineCatalog.TotalLines()
	if vp == &m.toolVP {
		totalLines = view.ContentLineCount(vp.View())
	}
	maxY := totalLines - vp.Height()
	if maxY < 0 {
		maxY = 0
	}
	if y < 0 {
		return 0
	}
	if y > maxY {
		return maxY
	}
	return y
}

func (m *Model) scrollViewportBy(vp *viewport.Model, delta int) {
	if vp == &m.chatVP {
		base := m.effectiveChatScrollY()
		m.setChatScrollY(m.clampYOffset(vp, base+delta))
		return
	}
	y := m.clampYOffset(vp, vp.YOffset()+delta)
	vp.SetYOffset(y)
}

func (m *Model) queueWheelScroll(target scroll.Target, delta int) tea.Cmd {
	if delta == 0 {
		return nil
	}
	m.scroll.ScrollBy(target, delta)
	if m.scroll.ScrollActive() {
		return nil
	}
	m.scroll.BeginDrain()
	m.scrollDeferSync = true
	return wheelScrollTickAfter()
}

func (m *Model) handleWheelScrollTick() tea.Cmd {
	target, pending, ok := m.scroll.ActiveTarget()
	if !ok {
		m.scroll.Active = false
		if m.scrollDeferSync {
			m.scrollDeferSync = false
			return m.scheduleSyncChatView()
		}
		return nil
	}

	vp := m.viewportForTarget(target)
	step := scroll.DrainStep(m.scroll.Profile(), pending, vp.Height())
	if step == 0 {
		m.scroll.ClearAll()
		return nil
	}

	absStep := step
	if absStep < 0 {
		absStep = -absStep
	}

	m.scrollViewportBy(vp, step)
	m.scroll.ApplyDrain(target, absStep)

	if m.scroll.HasPending() {
		return wheelScrollTickAfter()
	}
	if m.scrollDeferSync {
		m.scrollDeferSync = false
		return m.scheduleSyncChatView()
	}
	return nil
}

func (m *Model) jumpViewport(vp *viewport.Model, delta int) tea.Cmd {
	pending := m.scroll.ChatPending
	if vp == &m.toolVP {
		pending = m.scroll.ToolPending
	}
	if vp == &m.chatVP {
		base := m.effectiveChatScrollY()
		y := base + pending + delta
		m.scroll.ClearAll()
		m.setChatScrollY(m.clampYOffset(vp, y))
	} else {
		y := vp.YOffset() + pending + delta
		m.scroll.ClearAll()
		vp.SetYOffset(m.clampYOffset(vp, y))
	}
	if m.scrollDeferSync {
		m.scrollDeferSync = false
		return m.scheduleSyncChatView()
	}
	return nil
}

func (m *Model) viewportPageDelta(msg tea.KeyPressMsg, vp *viewport.Model) (int, bool) {
	if m.Running {
		km := vp.KeyMap
		switch {
		case key.Matches(msg, km.PageDown):
			return vp.Height(), true
		case key.Matches(msg, km.PageUp):
			return -vp.Height(), true
		case key.Matches(msg, km.HalfPageDown):
			return vp.Height() / 2, true
		case key.Matches(msg, km.HalfPageUp):
			return -(vp.Height() / 2), true
		case key.Matches(msg, km.Down):
			return 1, true
		case key.Matches(msg, km.Up):
			return -1, true
		default:
			return 0, false
		}
	}
	switch msg.String() {
	case "pgdown", "f":
		return vp.Height(), true
	case "pgup", "b":
		return -vp.Height(), true
	case "d", "ctrl+d":
		return vp.Height() / 2, true
	case "u", "ctrl+u":
		return -(vp.Height() / 2), true
	default:
		return 0, false
	}
}

func (m *Model) handleViewportScrollKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if !m.chatInteractionEnabled() {
		return nil, false
	}
	delta, ok := m.viewportPageDelta(msg, &m.chatVP)
	if !ok {
		return nil, false
	}
	return m.jumpViewport(&m.chatVP, delta), true
}
