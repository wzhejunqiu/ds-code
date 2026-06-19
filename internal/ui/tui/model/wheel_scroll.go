package model

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	tuimsg "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
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
	step := scroll.DrainStep(m.scroll.Profile(), pending, vp.Height)
	if step == 0 {
		m.scroll.ClearAll()
		return nil
	}

	absStep := step
	if absStep < 0 {
		absStep = -absStep
	}

	if step > 0 {
		vp.LineDown(absStep)
	} else {
		vp.LineUp(absStep)
	}
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
	y := vp.YOffset + m.scroll.ChatPending + delta
	m.scroll.ClearAll()
	vp.SetYOffset(y)
	if m.scrollDeferSync {
		m.scrollDeferSync = false
		return m.scheduleSyncChatView()
	}
	return nil
}

func (m *Model) viewportPageDelta(msg tea.KeyMsg, vp *viewport.Model) (int, bool) {
	if m.Running {
		km := vp.KeyMap
		switch {
		case key.Matches(msg, km.PageDown):
			return vp.Height, true
		case key.Matches(msg, km.PageUp):
			return -vp.Height, true
		case key.Matches(msg, km.HalfPageDown):
			return vp.Height / 2, true
		case key.Matches(msg, km.HalfPageUp):
			return -(vp.Height / 2), true
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
		return vp.Height, true
	case "pgup", "b":
		return -vp.Height, true
	case "d", "ctrl+d":
		return vp.Height / 2, true
	case "u", "ctrl+u":
		return -(vp.Height / 2), true
	default:
		return 0, false
	}
}

func (m *Model) handleViewportScrollKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.Overlay != state.OverlayNone || m.Prompt != nil {
		return nil, false
	}
	delta, ok := m.viewportPageDelta(msg, &m.chatVP)
	if !ok {
		return nil, false
	}
	return m.jumpViewport(&m.chatVP, delta), true
}
