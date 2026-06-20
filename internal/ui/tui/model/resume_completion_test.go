package model

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/input"
	tuimsg "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

func TestSlashCompletion_afterSessionResumed(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.CreateSession("m", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}

	d := depsWithStore(store, sess.ID)
	m := New(d)
	m.Width = 80
	m.Height = 24
	m.Overlay = state.OverlayResume
	m.ResumeFilterSeq = 1
	m.ResumeSessions = []session.Summary{{ID: sess.ID, Title: "hello"}}
	m.TestSyncResumePicker()

	m.input.Reset()
	m.Overlay = state.OverlayNone
	m.ResumeFilterSeq++

	staleSeq := m.ResumeFilterSeq - 1
	updated, _ := m.Update(tuimsg.ResumeListMsg{
		Filter:   "",
		Seq:      staleSeq,
		Sessions: []session.Summary{{ID: sess.ID}},
	})
	m = updated.(*Model)
	if m.Overlay == state.OverlayResume {
		t.Fatal("stale ResumeListMsg should not reopen resume overlay")
	}

	m.input.SetValue("/")
	updated, _ = m.Update(tuimsg.SessionResumedMsg{
		SessionID: sess.ID,
		Chat:      []chat.Block{{Role: chat.RoleUser, Content: strings.Repeat("line\n", 120)}},
	})
	m = updated.(*Model)

	if m.Overlay != state.OverlayComplete {
		t.Fatalf("overlay = %v, want OverlayComplete after SessionResumed with / in input", m.Overlay)
	}
	if !strings.Contains(m.OverlayText, "/help") {
		t.Fatalf("overlay = %q, want slash command list", m.OverlayText)
	}

	m.Overlay = state.OverlayNone
	m.OverlayText = ""
	m.refreshLayout()
	withoutOverlay := m.chatVP.Height()

	m.input.SetValue("/re")
	inputUpdateCompletion(m)
	m.refreshLayout()
	if m.Overlay != state.OverlayComplete {
		t.Fatalf("overlay = %v, want OverlayComplete for /re", m.Overlay)
	}
	if m.chatVP.Height() >= withoutOverlay {
		t.Fatalf("chat height = %d, want below %d when overlay open", m.chatVP.Height(), withoutOverlay)
	}
	if !strings.Contains(m.OverlayText, "/resume") {
		t.Fatalf("overlay = %q, want slash command list", m.OverlayText)
	}

	updated, _ = m.Update(tuimsg.ResumeListMsg{
		Filter:   "",
		Seq:      staleSeq,
		Sessions: []session.Summary{{ID: sess.ID}},
	})
	m = updated.(*Model)
	if m.Overlay != state.OverlayComplete {
		t.Fatalf("overlay = %v, want OverlayComplete after stale list", m.Overlay)
	}
}

func depsWithStore(store session.Store, sessionID string) *deps.Deps {
	return &deps.Deps{
		Store:     store,
		SessionID: sessionID,
		Cfg:       &config.Config{ProjectRoot: "/tmp", LLM: config.LLMConfig{Model: "m"}},
	}
}

func inputUpdateCompletion(m *Model) tea.Cmd {
	return input.UpdateCompletion(&m.State, m.input.Value(), &m.completePicker, &m.resumePicker)
}
