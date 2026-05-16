package context_test

import (
	"context"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
)

func TestBuildAPIContext_skipsSystemEvents(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: "user", Content: "hi"})
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: "system", Content: "[ds-code] rewound"})
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: "assistant", Content: "ok"})

	svc := &ctxpkg.Service{
		Cfg:   &config.Config{Context: config.ContextConfig{KeepRecentTurns: 6, WindowTokens: 1_048_576}},
		Store: store,
		Tools: tool.NewRegistry(),
	}
	view, err := svc.BuildAPIContext(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Messages) != 2 {
		t.Fatalf("messages = %d, want user+assistant only", len(view.Messages))
	}
}
