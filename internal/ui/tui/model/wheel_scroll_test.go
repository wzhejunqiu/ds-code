package model

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/scroll"
)

func TestWheelScroll_drainsMultipleLinesPerTick(t *testing.T) {
	m := New(testDeps(true))
	m.chatVP.SetContent(strings.Repeat("line\n", 50))
	m.chatVP.Height = 10

	_, cmd := m.handleMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
		X:      0,
		Y:      0,
	})
	if cmd == nil {
		t.Fatal("expected scroll tick command")
	}
	if m.chatVP.YOffset != 0 {
		t.Fatalf("yOffset = %d, want 0 before first tick", m.chatVP.YOffset)
	}

	before := m.scroll.ChatPending
	m.handleWheelScrollTick()
	if m.chatVP.YOffset < 1 {
		t.Fatalf("yOffset = %d, want > 0 after drain", m.chatVP.YOffset)
	}
	if m.scroll.ChatPending >= before {
		t.Fatalf("pending should decrease after drain")
	}
}

func TestWheelScroll_coalescesRapidNotches(t *testing.T) {
	m := New(testDeps(true))
	m.chatVP.SetContent(strings.Repeat("line\n", 100))

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
	m.chatVP.SetContent(strings.Repeat("line\n", 100))
	m.chatVP.Height = 10
	m.scroll.ChatPending = 12
	m.scroll.BeginDrain()

	m.jumpViewport(&m.chatVP, m.chatVP.Height/2)

	if m.scroll.HasPending() {
		t.Fatal("page jump should clear pending")
	}
	if m.chatVP.YOffset <= 0 {
		t.Fatalf("yOffset = %d, want page down from 0", m.chatVP.YOffset)
	}
}
