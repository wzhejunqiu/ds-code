package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
)

func TestRenderChat_interruptMarker(t *testing.T) {
	blocks := []chat.Block{
		{Role: chat.RoleUser},
		{Role: chat.RoleAssistant},
		{Role: chat.RoleInterrupt},
	}
	out := chat.Render(blocks, 60, time.Now(), false)
	if !strings.Contains(out, chat.InterruptLabel()) {
		t.Fatalf("expected interrupt label in output:\n%s", out)
	}
}

func TestCancelTurn_escShowsInterruptMarker(t *testing.T) {
	cancelled := make(chan struct{}, 1)
	m := model{
		running:    true,
		turnCancel: func() { close(cancelled) },
		chat: []chat.Block{
			{Role: chat.RoleUser},
			{Role: chat.RoleAssistant, Streaming: true},
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
	out := chat.Render(m.chat, 60, time.Now(), false)
	if !strings.Contains(out, chat.InterruptLabel()) {
		t.Fatalf("expected interrupt in chat:\n%s", out)
	}
}

func TestCancelTurn_escOnlyOncePerTurn(t *testing.T) {
	m := model{
		running:    true,
		turnCancel: func() {},
		chat:       []chat.Block{{Role: chat.RoleUser}, {Role: chat.RoleAssistant}},
	}
	m.cancelTurn()
	m.cancelTurn()
	count := 0
	for _, b := range m.chat {
		if b.Role == chat.RoleInterrupt {
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
		chat: []chat.Block{
			{Role: chat.RoleUser},
			{Role: chat.RoleAssistant, Streaming: true},
		},
	}
	_, turnCancel = context.WithCancel(context.Background())
	defer turnCancel()

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
		chat:        []chat.Block{{Role: chat.RoleUser}, {Role: chat.RoleAssistant}},
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
		chat:       []chat.Block{{Role: chat.RoleUser}, {Role: chat.RoleAssistant, Streaming: true}},
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
		chat:    []chat.Block{{Role: chat.RoleUser}, {Role: chat.RoleAssistant, Streaming: true}},
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
		chat: []chat.Block{
			{Role: chat.RoleUser},
			{Role: chat.RoleAssistant},
			{Role: chat.RoleInterrupt},
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
		chat: []chat.Block{{Role: chat.RoleUser}, {Role: chat.RoleAssistant}},
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
		chat:       []chat.Block{{Role: chat.RoleUser}, {Role: chat.RoleAssistant, Streaming: true}},
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

func TestUpdate_ctrlDDuringRunningDoesNotExit(t *testing.T) {
	cancelled := make(chan struct{}, 1)
	m := model{
		running:    true,
		turnCancel: func() { close(cancelled) },
		chat:       []chat.Block{{Role: chat.RoleUser}, {Role: chat.RoleAssistant, Streaming: true}},
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = *next.(*model)

	select {
	case <-cancelled:
		t.Fatal("Ctrl+D must not cancel a running turn")
	default:
	}
	if m.currentTurnInterrupted() {
		t.Fatal("Ctrl+D must not add interrupt marker")
	}
	if m.errLine != runningTurnHint {
		t.Fatalf("errLine = %q, want %q", m.errLine, runningTurnHint)
	}
	if cmd == nil {
		t.Fatal("expected exit confirm timeout tick")
	}
}

func TestUpdate_streamContentAfterInterruptIgnored(t *testing.T) {
	m := model{
		running: true,
		chat: []chat.Block{
			{Role: chat.RoleUser},
			{Role: chat.RoleAssistant},
			{Role: chat.RoleInterrupt},
		},
	}
	next, _ := m.Update(streamContentMsg{delta: "late"})
	m = *next.(*model)
	if len(m.chat) != 3 {
		t.Fatalf("chat blocks = %d, want 3", len(m.chat))
	}
	if m.chat[1].Content.String() != "" {
		t.Fatalf("assistant content = %q, want empty", m.chat[1].Content.String())
	}
}

func TestUpdate_toolStartAfterInterruptIgnored(t *testing.T) {
	m := model{
		running: true,
		chat: []chat.Block{
			{Role: chat.RoleUser},
			{Role: chat.RoleAssistant},
			{Role: chat.RoleInterrupt},
		},
	}
	next, _ := m.Update(toolStartMsg{name: "read_file", args: "path=a.go"})
	m = *next.(*model)
	for _, b := range m.chat {
		if b.Role == chat.RoleTool {
			t.Fatal("tool block must not be added after interrupt")
		}
	}
}

func TestUpdate_turnDoneMsg_persistsInterruptToSession(t *testing.T) {
	store := session.NewMemoryStore()
	ctx := context.Background()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	m := model{
		deps:      &Deps{Store: store},
		sessionID: sess.ID,
		chat: []chat.Block{
			{Role: chat.RoleUser},
			{Role: chat.RoleAssistant},
			{Role: chat.RoleInterrupt},
		},
	}
	next, _ := m.Update(turnDoneMsg{err: context.Canceled})
	m = *next.(*model)
	if m.errLine != "" {
		t.Fatalf("errLine = %q", m.errLine)
	}
	msgs, err := store.ListMessages(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, msg := range msgs {
		if msg.Role == role.System && msg.Content == chat.InterruptSessionMarker() {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected interrupt system message in session store")
	}
}
