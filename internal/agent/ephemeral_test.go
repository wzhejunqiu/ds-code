package agent_test

import (
	"context"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestRunEphemeral_noToolsNoHistoryWrite(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("deepseek-v4-pro", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: role.User, Content: "secret"})

	mockLLM := &mock.Client{
		Responses: []*llm.Response{{Content: "side answer", FinishReason: "stop"}},
	}
	cfg := testConfig()
	cfg.BTW.MaxTokens = 1024
	r := &agent.Runner{
		LLM:      mockLLM,
		Sessions: store,
		Context:  &ctxpkg.Service{Cfg: cfg, Store: store, Tools: tool.NewRegistry()},
		Cfg:      cfg,
		Perm:     permission.NewEngine("ask", t.TempDir(), false),
	}

	res, err := r.RunEphemeral(ctx, "quick?", agent.EphemeralOpts{SessionID: sess.ID})
	if err != nil {
		t.Fatal(err)
	}
	if res.Content != "side answer" {
		t.Fatalf("content = %q", res.Content)
	}
	if len(mockLLM.Calls) != 1 {
		t.Fatalf("calls = %d", len(mockLLM.Calls))
	}
	if len(mockLLM.Calls[0].Tools) != 0 {
		t.Fatal("btw must not attach tools")
	}
	msgs, _ := store.ListMessages(ctx, sess.ID)
	if len(msgs) != 1 {
		t.Fatalf("history rows = %d, want 1 (no btw write)", len(msgs))
	}
}
