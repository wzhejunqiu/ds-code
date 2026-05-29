package spawn_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/register"
)

func TestService_Handle_syncExplore(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot:    dir,
		ProjectDataDir: dir,
		LLM:            config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:        config.ContextConfig{ToolResultMaxChars: 50_000},
		Tools:          config.ToolsConfig{Agent: config.AgentToolConfig{AutoBackgroundAfter: 0}},
		Agent:          config.AgentConfig{MaxTurns: 5},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{{Content: "found it", FinishReason: "stop"}},
	}
	sub := subagentstore.NewMemoryStore()
	perm := permission.NewEngine("readonly", dir, false)
	reg := tool.NewRegistry()
	register.ExploreTools(reg, cfg, perm, nil, false)
	svc := spawn.NewService(cfg, perm, reg, mockLLM, sub)

	out, err := svc.Handle(context.Background(), agent.ToolInvocation{
		SessionID: "sess", ToolCallID: "tc-sync",
	}, spawn.Params{SubagentType: "Explore", Description: "find", Prompt: "search"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "found it") {
		t.Fatalf("unexpected sync output: %q", out)
	}
}

func TestService_Handle_asyncDrainNotification(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot:    dir,
		ProjectDataDir: dir,
		LLM:            config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:        config.ContextConfig{ToolResultMaxChars: 50_000},
		Agent:          config.AgentConfig{MaxTurns: 5},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{{Content: "done", FinishReason: "stop"}},
	}
	sub := subagentstore.NewMemoryStore()
	perm := permission.NewEngine("readonly", dir, false)
	reg := tool.NewRegistry()
	register.ExploreTools(reg, cfg, perm, nil, false)
	svc := spawn.NewService(cfg, perm, reg, mockLLM, sub)

	_, err := svc.Handle(context.Background(), agent.ToolInvocation{
		SessionID: "sess", ToolCallID: "tc-async",
	}, spawn.Params{SubagentType: "Explore", Description: "bg", Prompt: "work", RunInBackground: true}, true)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if got := svc.DrainNotifications(spawn.PrioNow); len(got) > 0 {
			return
		}
		if got := svc.DrainNotifications(spawn.PrioLater); len(got) > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected async completion notification")
}

func TestBuildForkAPIContext_prefix(t *testing.T) {
	parentView := &ctxpkg.APIContextView{WindowTokens: 128000}
	calls := []llm.ToolCall{{ID: "c1", Name: "agent"}}
	msgs := spawn.BuildForkMessages(nil, calls, "task")
	forkView := ctxpkg.BuildForkAPIContext(parentView, msgs, "rendered-system")
	if forkView.RenderedSystemOverride != "rendered-system" {
		t.Fatal("expected rendered system override")
	}
	if len(forkView.Messages) != 2 {
		t.Fatalf("expected tool+user, got %d", len(forkView.Messages))
	}
	if forkView.Messages[0].Role != role.Tool || forkView.Messages[0].Content != spawn.ForkPlaceholder {
		t.Fatal("expected tool placeholder message")
	}
}
