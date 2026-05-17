package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
)

func TestResumeDoubleEnterIgnoredWhilePending(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.CreateSession("m", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	m := model{
		deps:           &Deps{Store: store},
		resumeSessions: []session.Summary{{ID: sess.ID}},
	}
	m.overlay = overlayResume
	m.syncResumePicker()

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd1 := m.Update(enter)
	m1 := updated.(*model)
	if cmd1 == nil {
		t.Fatal("expected resume cmd")
	}
	if !m1.resumePending {
		t.Fatal("expected resumePending after first enter")
	}

	updated2, cmd2 := m1.Update(enter)
	m2 := updated2.(*model)
	if cmd2 != nil {
		t.Fatal("second enter should be ignored while resume is pending")
	}
	if !m2.resumePending {
		t.Fatal("resumePending should stay true until sessionResumedMsg")
	}
}

func TestViewWithWidthBeforeWindowSizeMsg(t *testing.T) {
	sm := newSafeModel(&Deps{
		Store:     session.NewMemoryStore(),
		SessionID: "x",
		Version:   "v",
		Cfg:       &config.Config{ProjectRoot: "/tmp", LLM: config.LLMConfig{Model: "m"}},
	})
	sm.inner.width = 80
	sm.inner.height = 24
	sm.inner.syncChatView()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View panicked: %v", r)
		}
	}()
	_ = sm.View()
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
	sm.inner.width = 120
	sm.inner.height = 40
	sm.inner.input.SetValue("/resume")
	sm.inner.overlay = overlayResume
	sm.inner.resumeSessions = []session.Summary{{ID: sess.ID, Title: "hello"}}
	sm.inner.syncResumePicker()

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	for i := 0; i < 2; i++ {
		updated, cmd := sm.Update(enter)
		if s, ok := updated.(*safeModel); ok {
			sm = s
		}
		if cmd != nil {
			if msg := cmd(); msg != nil {
				updated2, _ := sm.Update(msg)
				if s, ok := updated2.(*safeModel); ok {
					sm = s
				}
			}
		}
		sm.inner.width = 80
		sm.inner.syncChatView()
		_ = sm.View()
	}
}
