package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
)

func TestViewWithWidthBeforeWindowSizeMsg(t *testing.T) {
	sm := newSafeModel(&Deps{
		Store:     session.NewMemoryStore(),
		SessionID: "x",
		Version:   "v",
		Cfg:       &config.Config{ProjectRoot: "/tmp", LLM: config.LLMConfig{Model: "m"}},
	})
	sm.inner.Width = 80
	sm.inner.Height = 24
	sm.inner.TestSyncChatView()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View panicked: %v", r)
		}
	}()
	_ = sm.View()
}

func TestResumeDoubleEnterIgnoredWhilePending(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.CreateSession("m", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	sm := newSafeModel(&Deps{Store: store})
	m := sm.inner
	m.ResumeSessions = []session.Summary{{ID: sess.ID}}
	m.Overlay = state.OverlayResume
	m.TestSyncResumePicker()

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd1 := sm.Update(enter)
	sm = updated.(*safeModel)
	if cmd1 == nil {
		t.Fatal("expected resume cmd")
	}
	if !sm.inner.ResumePending {
		t.Fatal("expected resumePending after first enter")
	}

	updated2, cmd2 := sm.Update(enter)
	sm = updated2.(*safeModel)
	if cmd2 != nil {
		t.Fatal("second enter should be ignored while resume is pending")
	}
	if !sm.inner.ResumePending {
		t.Fatal("resumePending should stay true until sessionResumedMsg")
	}
}

func TestResumeDoubleEnterViewDoesNotPanic(t *testing.T) {
	store := session.NewMemoryStore()
	ctx := context.Background()
	sess, err := store.CreateSession("m", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: role.User, Content: "hello"})
	_ = store.AppendMessage(ctx, session.Message{
		SessionID: sess.ID,
		Role:      role.Assistant,
		Content:   "# Title\n\n**bold** body",
	})

	cfg := &config.Config{ProjectRoot: "/tmp", LLM: config.LLMConfig{Model: "m"}}
	sm := newSafeModel(&Deps{Store: store, SessionID: sess.ID, Version: "test", Cfg: cfg})
	sm.inner.Width = 120
	sm.inner.Height = 40
	sm.inner.TestInputSetValue("/resume")
	sm.inner.Overlay = state.OverlayResume
	sm.inner.ResumeSessions = []session.Summary{{ID: sess.ID, Title: "hello"}}
	sm.inner.TestSyncResumePicker()

	enter := tea.KeyMsg{Type: tea.KeyEnter}

	updated, resumeCmd := sm.Update(enter)
	if s, ok := updated.(*safeModel); ok {
		sm = s
	}
	if resumeCmd == nil {
		t.Fatal("expected resume cmd on first enter")
	}

	// Second enter while resume is in flight must not start another resume or corrupt state.
	updated2, cmd2 := sm.Update(enter)
	if s, ok := updated2.(*safeModel); ok {
		sm = s
	}
	if cmd2 != nil {
		t.Fatal("second enter should be ignored while resume is pending")
	}

	if msg := resumeCmd(); msg != nil {
		updated3, _ := sm.Update(msg)
		if s, ok := updated3.(*safeModel); ok {
			sm = s
		}
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View panicked: %v", r)
		}
	}()
	_ = sm.View()
}

// silence deps import
var _ = deps.Deps{}
