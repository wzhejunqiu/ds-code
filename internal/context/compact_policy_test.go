package context

import (
	"context"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestShouldCompact_promptTotal(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "", "", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.UpdateSession(ctx, sess.ID, func(st *session.Session) error {
		st.PromptTokensTotal = 900
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	svc := &Service{
		Cfg: &config.Config{
			Context: config.ContextConfig{
				WindowTokens:          1000,
				CompactThresholdRatio: 0.8,
			},
		},
		Store: store,
		Tools: tool.NewRegistry(),
	}
	got, err := store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !svc.shouldCompact(ctx, sess.ID, got) {
		t.Fatal("expected compact when prompt total exceeds threshold")
	}
}

func TestShouldCompact_breakdownTotal(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "", "", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	svc := &Service{
		Cfg: &config.Config{
			Context: config.ContextConfig{
				WindowTokens:          100,
				CompactThresholdRatio: 0.5,
			},
		},
		Store: store,
		Tools: tool.NewRegistry(),
	}
	got, err := store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !svc.shouldCompact(ctx, sess.ID, got) {
		t.Fatal("expected compact on first breakdown when threshold is very low")
	}
	if !svc.userTurnCounted {
		t.Fatal("expected userTurnCounted after first breakdown")
	}
}

func TestShouldCompact_cachedBreakdownBelowThreshold(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "", "", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	svc := &Service{
		Cfg: &config.Config{
			Context: config.ContextConfig{
				WindowTokens:          1000,
				CompactThresholdRatio: 0.8,
			},
		},
		Store: store,
		Tools: tool.NewRegistry(),
	}
	svc.userTurnCounted = true
	svc.userTurnBreakdown = &ContextBreakdown{
		SystemPrompt: 10,
		Tools:        10,
		Conversation: 10,
	}
	got, err := store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if svc.shouldCompact(ctx, sess.ID, got) {
		t.Fatal("expected no compact when cached breakdown is below threshold")
	}
}
