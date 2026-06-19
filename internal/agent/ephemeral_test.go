package agent_test

import (
	"context"
	"os"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/mcp/resultstore"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
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

func TestRunEphemeral_noMCPSpill(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	store := session.NewMemoryStore()
	sess, err := store.NewSession("deepseek-v4-pro", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	cfg := testConfig()
	cfg.ProjectRoot = root
	r := &agent.Runner{
		LLM:        mockLLM(&llm.Response{Content: "ok", FinishReason: "stop"}),
		Sessions:   store,
		Context:    &ctxpkg.Service{Cfg: cfg, Store: store, Tools: tool.NewRegistry()},
		Cfg:        cfg,
		Perm:       permission.NewEngine("ask", root, false),
		MCPResults: &resultstore.Store{ProjectRoot: root},
	}

	_, err = r.RunEphemeral(context.Background(), "btw", agent.EphemeralOpts{SessionID: sess.ID})
	if err != nil {
		t.Fatal(err)
	}
	base := datadir.DefaultMCPResultDir(root)
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatalf("RunEphemeral must not create mcp-result: %v", err)
	}
}

func mockLLM(resp *llm.Response) *mock.Client {
	return &mock.Client{Responses: []*llm.Response{resp}}
}
