package model

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
	"github.com/hejunqiu/ds-code/internal/ui/tui/deps"
	tuimsg "github.com/hejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/turn"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/view"
)

func TestCancelTurn_escShowsInterruptMarker(t *testing.T) {
	cancelled := make(chan struct{}, 1)
	m := New(&deps.Deps{})
	m.Running = true
	m.TurnCancel = func() { close(cancelled) }
	m.Chat = []chat.Block{
		{Role: chat.RoleUser},
		{Role: chat.RoleAssistant, Streaming: true},
	}
	turn.Cancel(&m.State, m.syncChatView)

	select {
	case <-cancelled:
	default:
		t.Fatal("expected turn cancel func invoked")
	}
	if !turn.CurrentTurnInterrupted(&m.State) {
		t.Fatal("expected interrupt marker in current turn")
	}
	out := chat.Render(m.Chat, 60, time.Now(), false)
	if !strings.Contains(out, chat.InterruptLabel()) {
		t.Fatalf("expected interrupt in chat:\n%s", out)
	}
}

func TestUpdate_escCancelsRunningTurn(t *testing.T) {
	var turnCancel context.CancelFunc
	m := New(&deps.Deps{})
	m.Running = true
	m.TurnCancel = func() {
		if turnCancel != nil {
			turnCancel()
		}
	}
	m.Chat = []chat.Block{
		{Role: chat.RoleUser},
		{Role: chat.RoleAssistant, Streaming: true},
	}
	_, turnCancel = context.WithCancel(context.Background())
	defer turnCancel()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(*Model)

	if !turn.CurrentTurnInterrupted(&m.State) {
		t.Fatal("expected interrupt marker after Esc")
	}
	if m.TurnEscPending {
		t.Fatal("pending flag should be clear when cancel func was available")
	}
}

func TestUpdate_ctrlCDuringRunningDoesNotCancelTurn(t *testing.T) {
	cancelled := make(chan struct{}, 1)
	m := New(&deps.Deps{})
	m.Running = true
	m.TurnCancel = func() { close(cancelled) }
	m.Chat = []chat.Block{{Role: chat.RoleUser}, {Role: chat.RoleAssistant, Streaming: true}}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = next.(*Model)

	select {
	case <-cancelled:
		t.Fatal("Ctrl+C must not cancel a running turn")
	default:
	}
	if turn.CurrentTurnInterrupted(&m.State) {
		t.Fatal("Ctrl+C must not add interrupt marker")
	}
	if m.ErrLine != view.RunningTurnHint() {
		t.Fatalf("errLine = %q, want %q", m.ErrLine, view.RunningTurnHint())
	}
}

func TestUpdate_turnDoneMsg_persistsInterruptToSession(t *testing.T) {
	store := session.NewMemoryStore()
	ctx := context.Background()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	m := New(&deps.Deps{Store: store})
	m.SessionID = sess.ID
	m.Chat = []chat.Block{
		{Role: chat.RoleUser},
		{Role: chat.RoleAssistant},
		{Role: chat.RoleInterrupt},
	}
	next, _ := m.Update(tuimsg.TurnDoneMsg{Err: context.Canceled})
	m = next.(*Model)
	if m.ErrLine != "" {
		t.Fatalf("errLine = %q", m.ErrLine)
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

func TestUpdate_escDuringPermissionPromptCancelsTurn(t *testing.T) {
	reply := make(chan bool, 1)
	m := New(&deps.Deps{})
	m.Running = true
	m.TurnCancel = func() {}
	m.Overlay = state.OverlayPrompt
	m.Prompt = &permission.PromptRequest{Tool: "shell", Reply: reply}
	m.Chat = []chat.Block{{Role: chat.RoleUser}, {Role: chat.RoleAssistant, Streaming: true}}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(*Model)
	if !turn.CurrentTurnInterrupted(&m.State) {
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
