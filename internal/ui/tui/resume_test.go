package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/session"
)

func TestResumePickerEnterDoesNotSubmitFilterAsSessionID(t *testing.T) {
	store := session.NewMemoryStore()
	ctx := context.Background()
	sess, err := store.CreateSession("deepseek-v4-pro", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: "user", Content: "hi"})

	m := model{
		deps: &Deps{Store: store},
	}
	m.overlay = overlayResume
	m.resumeSessions = nil
	m.input.SetValue("/resume nomatch")

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := m.Update(enter)
	m2 := updated.(*model)

	if cmd != nil {
		t.Fatal("expected no cmd when picker has no matches")
	}
	if m2.errLine != "" {
		t.Fatalf("unexpected error: %q", m2.errLine)
	}
	if m2.overlay != overlayResume {
		t.Fatalf("overlay = %v, want overlayResume", m2.overlay)
	}
}

func TestResumePickerEnterResumesSelectedSession(t *testing.T) {
	store := session.NewMemoryStore()
	ctx := context.Background()
	sess, err := store.CreateSession("deepseek-v4-pro", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: "user", Content: "hello"})

	m := model{
		deps: &Deps{Store: store},
	}
	m.overlay = overlayResume
	m.resumeSessions = []session.Summary{{ID: sess.ID, Title: "hello"}}
	m.input.SetValue("/resume")

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(enter)
	if cmd == nil {
		t.Fatal("expected resume cmd")
	}
	msg := cmd()
	res, ok := msg.(sessionResumedMsg)
	if !ok {
		t.Fatalf("msg type %T", msg)
	}
	if res.err != nil {
		t.Fatal(res.err)
	}
	if res.sessionID != sess.ID {
		t.Fatalf("sessionID = %q", res.sessionID)
	}
	if len(res.chat) != 1 {
		t.Fatalf("chat blocks = %d", len(res.chat))
	}
	if res.chat[0].content.String() != "hello" {
		t.Fatalf("chat content = %q", res.chat[0].content.String())
	}
}
