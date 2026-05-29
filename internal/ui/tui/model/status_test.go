package model

import (
	"context"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/session"
	tuimsg "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
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
	if _, cmd := m.Update(tuimsg.UsageUpdateMsg{}); cmd != nil {
		t.Fatalf("expected no cmd, got %v", cmd)
	}
	if !strings.Contains(m.StatusRight, "in 42") {
		t.Fatalf("after UsageUpdateMsg StatusRight = %q, want in 42", m.StatusRight)
	}
}

func TestInit_noStatusRefreshTick(t *testing.T) {
	m := New(&deps.Deps{
		Store:     session.NewMemoryStore(),
		SessionID: "sess",
	})
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned nil cmd")
	}
}
