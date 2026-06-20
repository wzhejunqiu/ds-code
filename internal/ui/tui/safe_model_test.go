package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
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

	enter := tea.KeyPressMsg{Code: tea.KeyEnter}
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

	enter := tea.KeyPressMsg{Code: tea.KeyEnter}

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

func TestView_returnsTeaView(t *testing.T) {
	sm := newSafeModel(&Deps{
		Store:     session.NewMemoryStore(),
		SessionID: "x",
		Version:   "v",
		Cfg:       &config.Config{ProjectRoot: "/tmp", LLM: config.LLMConfig{Model: "m"}},
	})
	sm.inner.Width = 80
	sm.inner.Height = 24
	sm.inner.TestSyncChatView()

	v := sm.View()
	if !v.AltScreen {
		t.Fatal("expected AltScreen=true")
	}
	if v.MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("mouse mode = %v, want CellMotion", v.MouseMode)
	}
}

func TestFallbackView_returnsTeaView(t *testing.T) {
	sm := &safeModel{
		lastView: tea.View{
			AltScreen: true,
			MouseMode: tea.MouseModeCellMotion,
			Cursor:    tea.NewCursor(3, 2),
		},
	}
	v := sm.fallbackView()
	if !v.AltScreen {
		t.Fatal("expected AltScreen=true")
	}
	if v.MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("mouse mode = %v, want passthrough CellMotion", v.MouseMode)
	}
	if v.Cursor == nil || v.Cursor.X != 3 || v.Cursor.Y != 2 {
		t.Fatalf("cursor = %+v, want passthrough at (3,2)", v.Cursor)
	}
}

// silence deps import
var _ = deps.Deps{}
