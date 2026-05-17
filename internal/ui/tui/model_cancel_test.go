package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/permission"
)

func TestRenderChat_interruptMarker(t *testing.T) {
	blocks := []chatBlock{
		{role: "user"},
		{role: "assistant"},
		{role: "interrupt"},
	}
	out := renderChat(blocks, 60, time.Now(), false)
	if !strings.Contains(out, interruptLabel) {
		t.Fatalf("expected interrupt label in output:\n%s", out)
	}
}

func TestCancelTurn_escShowsInterruptMarker(t *testing.T) {
	cancelled := make(chan struct{}, 1)
	m := model{
		running:    true,
		turnCancel: func() { close(cancelled) },
		chat: []chatBlock{
			{role: "user"},
			{role: "assistant", streaming: true},
		},
	}
	m.cancelTurn()

	select {
	case <-cancelled:
	default:
		t.Fatal("expected turn cancel func invoked")
	}
	if !m.currentTurnInterrupted() {
		t.Fatal("expected interrupt marker in current turn")
	}
	out := renderChat(m.chat, 60, time.Now(), false)
	if !strings.Contains(out, interruptLabel) {
		t.Fatalf("expected interrupt in chat:\n%s", out)
	}
}

func TestCancelTurn_escOnlyOncePerTurn(t *testing.T) {
	m := model{
		running:    true,
		turnCancel: func() {},
		chat:       []chatBlock{{role: "user"}, {role: "assistant"}},
	}
	m.cancelTurn()
	m.cancelTurn()
	count := 0
	for _, b := range m.chat {
		if b.role == "interrupt" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("interrupt blocks = %d, want 1", count)
	}
}

func TestUpdate_escCancelsRunningTurn(t *testing.T) {
	var turnCancel context.CancelFunc
	m := model{
		running: true,
		turnCancel: func() {
			if turnCancel != nil {
				turnCancel()
			}
		},
		chat: []chatBlock{
			{role: "user"},
			{role: "assistant", streaming: true},
		},
	}
	_, turnCancel = context.WithCancel(context.Background())

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = *next.(*model)

	if !m.currentTurnInterrupted() {
		t.Fatal("expected interrupt marker after Esc")
	}
	if m.turnEscPending {
		t.Fatal("pending flag should be clear when cancel func was available")
	}
}

func TestUpdate_escClosesOverlayBeforeCancel(t *testing.T) {
	m := model{
		running:     true,
		turnCancel:  func() {},
		overlay:     overlayHelp,
		overlayText: "help",
		chat:        []chatBlock{{role: "user"}, {role: "assistant"}},
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = *next.(*model)
	if m.overlay != overlayNone {
		t.Fatalf("overlay = %v, want none", m.overlay)
	}
	if m.currentTurnInterrupted() {
		t.Fatal("overlay close should not cancel turn")
	}
}

func TestUpdate_ctrlCDuringRunningDoesNotCancelTurn(t *testing.T) {
	cancelled := make(chan struct{}, 1)
	m := model{
		running:    true,
		turnCancel: func() { close(cancelled) },
		chat:       []chatBlock{{role: "user"}, {role: "assistant", streaming: true}},
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = *next.(*model)

	select {
	case <-cancelled:
		t.Fatal("Ctrl+C must not cancel a running turn")
	default:
	}
	if m.currentTurnInterrupted() {
		t.Fatal("Ctrl+C must not add interrupt marker")
	}
	if m.errLine != runningTurnHint {
		t.Fatalf("errLine = %q, want %q", m.errLine, runningTurnHint)
	}
}

func TestUpdate_escBeforeTurnCancelArrives(t *testing.T) {
	var cancel context.CancelFunc
	m := model{
		running: true,
		chat:    []chatBlock{{role: "user"}, {role: "assistant", streaming: true}},
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = *next.(*model)
	if !m.turnEscPending || !m.currentTurnInterrupted() {
		t.Fatal("expected pending Esc cancel and interrupt marker")
	}

	ctx, c := context.WithCancel(context.Background())
	cancel = c
	next, _ = m.Update(turnStartedMsg{cancel: cancel})
	m = *next.(*model)
	if m.turnEscPending {
		t.Fatal("pending flag should be cleared")
	}
	if ctx.Err() == nil {
		t.Fatal("expected context cancelled when turnStarted honors pending Esc")
	}
}

func TestUpdate_turnDoneMsg_canceledWithInterruptClearsErrLine(t *testing.T) {
	m := model{
		chat: []chatBlock{
			{role: "user"},
			{role: "assistant"},
			{role: "interrupt"},
		},
	}
	next, _ := m.Update(turnDoneMsg{err: context.Canceled})
	m = *next.(*model)
	if m.errLine != "" {
		t.Fatalf("errLine = %q, want empty when interrupt marker present", m.errLine)
	}
}

func TestUpdate_turnDoneMsg_canceledWithoutInterruptShowsErrLine(t *testing.T) {
	m := model{
		chat: []chatBlock{{role: "user"}, {role: "assistant"}},
	}
	next, _ := m.Update(turnDoneMsg{err: context.Canceled})
	m = *next.(*model)
	if m.errLine != "turn cancelled" {
		t.Fatalf("errLine = %q, want turn cancelled", m.errLine)
	}
}

func TestUpdate_escDuringPermissionPromptCancelsTurn(t *testing.T) {
	reply := make(chan bool, 1)
	m := model{
		running:    true,
		turnCancel: func() {},
		overlay:    overlayPrompt,
		prompt:     &permission.PromptRequest{Tool: "shell", Reply: reply},
		chat:       []chatBlock{{role: "user"}, {role: "assistant", streaming: true}},
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = *next.(*model)
	if !m.currentTurnInterrupted() {
		t.Fatal("expected interrupt marker when Esc during permission prompt")
	}
	select {
	case ok := <-reply:
		if ok {
			t.Fatal("expected permission reply false to unblock runner")
		}
	default:
		t.Fatal("expected reply sent to permission channel")
	}
}
