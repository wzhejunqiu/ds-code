package context_test

import (
	"context"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
)

func TestBuildAPIContext_keepRecentTurns(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("deepseek-v4-pro", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for i := 0; i < 8; i++ {
		_ = store.AppendMessage(ctx, session.Message{
			SessionID: sess.ID,
			Role:      "user",
			Content:   "turn",
		})
		_ = store.AppendMessage(ctx, session.Message{
			SessionID: sess.ID,
			Role:      "assistant",
			Content:   "ok",
		})
	}

	svc := &ctxpkg.Service{
		Cfg: &config.Config{
			Context: config.ContextConfig{KeepRecentTurns: 2, WindowTokens: 1_048_576},
		},
		Store: store,
		Tools: tool.NewRegistry(),
	}
	view, err := svc.BuildAPIContext(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	// 2 user turns => 2 user + 2 assistant = 4 messages
	if len(view.Messages) != 4 {
		t.Fatalf("messages = %d, want 4", len(view.Messages))
	}
}
