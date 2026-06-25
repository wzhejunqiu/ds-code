package model

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/scroll"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/selection"
)

func TestWheelScroll_drainsMultipleLinesPerTick(t *testing.T) {
	m := New(testDeps(true))
	seedChatLines(m, 50)
	m.chatVP.SetHeight(10)
	_, cmd := m.handleMouse(tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 0, Y: 2})
	if cmd == nil {
		t.Fatal("expected scroll tick command")
	}
	if m.chatScrollY != 0 {
		t.Fatalf("chatScrollY = %d, want 0 before first tick", m.chatScrollY)
	}

	before := m.scroll.ChatPending
	m.handleWheelScrollTick()
	if m.chatScrollY < 1 {
		t.Fatalf("chatScrollY = %d, want > 0 after drain", m.chatScrollY)
	}
	if m.scroll.ChatPending >= before {
		t.Fatalf("pending should decrease after drain")
	}
}

func TestWheelScroll_coalescesRapidNotches(t *testing.T) {
	m := New(testDeps(true))
	seedChatLines(m, 100)

	m.queueWheelScroll(scroll.TargetChat, 3)
	m.queueWheelScroll(scroll.TargetChat, 3)

	if m.scroll.ChatPending != 6 {
		t.Fatalf("pending = %d, want 6", m.scroll.ChatPending)
	}
	if !m.scroll.ScrollActive() {
		t.Fatal("expected active scroll animation")
	}
}

func TestWheelScroll_capsPending(t *testing.T) {
	m := New(testDeps(true))
	m.scroll.ChatPending = scroll.PendingMax
	m.queueWheelScroll(scroll.TargetChat, 10)
	if m.scroll.ChatPending != scroll.PendingMax {
		t.Fatalf("pending = %d, want clamp cap %d", m.scroll.ChatPending, scroll.PendingMax)
	}
}

func TestScroll_jumpBy_clearsPending(t *testing.T) {
	m := New(testDeps(true))
	seedChatLines(m, 100)
	m.chatVP.SetHeight(10)
	m.scroll.ChatPending = 12
	m.scroll.BeginDrain()

	m.jumpViewport(&m.chatVP, m.chatVP.Height()/2)

	if m.scroll.HasPending() {
		t.Fatal("page jump should clear pending")
	}
	if m.chatScrollY <= 0 {
		t.Fatalf("chatScrollY = %d, want page down from 0", m.chatScrollY)
	}
}

func TestWheelScroll_drainReturnsCmdWhenDeferred(t *testing.T) {
	m := New(testDeps(true))
	seedChatLines(m, 50)
	m.chatVP.SetHeight(10)
	m.scrollDeferSync = true

	m.queueWheelScroll(scroll.TargetChat, 5)
	cmd := m.handleWheelScrollTick()
	if cmd == nil {
		t.Fatal("expected sync command when scrollDeferSync is set")
	}
}

func TestChatInteractionEnabled_overlayBlocks(t *testing.T) {
	m := New(testDeps(true))
	m.Overlay = state.OverlayHelp
	if m.chatInteractionEnabled() {
		t.Fatal("overlay should disable chat interaction")
	}
}

func TestWheelScroll_worksAfterCopySelection(t *testing.T) {
	m := New(testDeps(true))
	seedChatLines(m, 50)
	m.chatVP.SetWidth(80)
	m.chatVP.SetHeight(10)
	m.selDragging = false
	m.selRange = selection.Range{
		Start: selection.Point{Line: 0, Col: 0},
		End:   selection.Point{Line: 0, Col: 5},
	}

	before := m.chatScrollY
	_, cmd := m.handleMouse(tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 5, Y: 2})
	if cmd == nil {
		t.Fatal("expected wheel scroll tick command after copy selection")
	}
	for i := 0; i < 8 && m.chatScrollY <= before; i++ {
		m.handleWheelScrollTick()
	}
	if m.chatScrollY <= before {
		t.Fatalf("wheel down after copy: chatScrollY = %d, want > %d", m.chatScrollY, before)
	}
}

func TestRunningMode_chatVPScrollKey(t *testing.T) {
	m := New(testDeps(true))
	seedChatLines(m, 80)
	m.chatVP.SetHeight(10)
	m.Running = true
	m.chatScrollY = 0

	before := m.chatScrollY
	updated, _ := m.updateInput(tea.KeyPressMsg{Code: tea.KeyPgDown})
	m = updated.(*Model)
	if m.chatScrollY <= before {
		t.Fatalf("chatScrollY = %d, want > %d after page down while running", m.chatScrollY, before)
	}
}

func TestJumpViewport_clampsTotalLines(t *testing.T) {
	m := New(testDeps(true))
	m.Width = 80
	m.Height = 24
	seedChatLines(m, 80)
	m.syncChatView()
	m.chatVP.SetHeight(10)

	m.jumpViewport(&m.chatVP, 9999)
	if !scroll.IsPinnedBottom(m.chatScrollY) {
		t.Fatalf("chatScrollY = %d, want pinned bottom at max", m.chatScrollY)
	}

	m.jumpViewport(&m.chatVP, -9999)
	if m.chatScrollY != 0 {
		t.Fatalf("chatScrollY = %d, want 0", m.chatScrollY)
	}
}
