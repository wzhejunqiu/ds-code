package model

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	tuimsg "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
)

func TestUpdate_usageUpdateMsg_refreshesStatus(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.AddUsage(ctx, sess.ID, llm.Usage{PromptTokens: 42, CompletionTokens: 7}); err != nil {
		t.Fatal(err)
	}

	m := New(&deps.Deps{
		Cfg:       &config.Config{ProjectRoot: t.TempDir()},
		Store:     store,
		SessionID: sess.ID,
	})
	if !strings.Contains(m.StatusRight, "in 42") {
		t.Fatalf("initial StatusRight = %q, want in 42", m.StatusRight)
	}

	m.StatusRight = ""
	_, cmd := m.Update(tuimsg.UsageUpdateMsg{})
	if cmd == nil {
		t.Fatal("expected deferred sync cmd")
	}
	if !strings.Contains(m.StatusRight, "in 42") {
		t.Fatalf("after UsageUpdateMsg StatusRight = %q, want in 42", m.StatusRight)
	}
}

func TestInit_noPeriodicStatusTick(t *testing.T) {
	m := New(&deps.Deps{
		Cfg:       &config.Config{ProjectRoot: t.TempDir()},
		Store:     session.NewMemoryStore(),
		SessionID: "sess",
	})
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned nil cmd")
	}
	assertNoUsageUpdateMsg(t, cmd())
}

func assertNoUsageUpdateMsg(t *testing.T, msg tea.Msg) {
	t.Helper()
	if msg == nil {
		return
	}
	if _, ok := msg.(tuimsg.UsageUpdateMsg); ok {
		t.Fatal("Init must not schedule UsageUpdateMsg")
	}
	switch m := msg.(type) {
	case tea.BatchMsg:
		for _, sub := range m {
			if sub == nil {
				continue
			}
			assertNoUsageUpdateMsg(t, sub())
		}
	}
}
