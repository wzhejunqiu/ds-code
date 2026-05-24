package spawn_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/register"
)

func TestSyncPromote_returnsBeforeCompletion(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot:     dir,
		ProjectDataDir:  dir,
		LLM:             config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:         config.ContextConfig{ToolResultMaxChars: 50_000},
		Tools:           config.ToolsConfig{Agent: config.AgentToolConfig{AutoBackgroundAfter: 1, SummaryMaxChars: 8000}},
		Agent:           config.AgentConfig{MaxTurns: 5},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{{Content: "done", FinishReason: "stop"}},
		Delay:     3 * time.Second,
	}
	main := session.NewMemoryStore()
	parent, _ := main.CreateSession("m", "max", "enabled", "auto", "agent")
	sub := subagentstore.NewMemoryStore()
	perm := permission.NewEngine("readonly", dir, false)
	reg := tool.NewRegistry()
	register.ExploreTools(reg, cfg, perm, nil, false)

	svc := spawn.NewService(cfg, perm, reg, mockLLM, sub)
	before, _ := main.ListMessages(context.Background(), parent.ID)

	out, err := svc.Handle(context.Background(), agent.ToolInvocation{
		SessionID: parent.ID, ToolCallID: "call-promote",
	}, spawn.Params{SubagentType: "Explore", Description: "slow", Prompt: "work"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "async_launched") {
		t.Fatalf("expected promote response, got %q", out)
	}
	after, _ := main.ListMessages(context.Background(), parent.ID)
	if len(after) != len(before) {
		t.Fatalf("main session should not gain messages on promote, before=%d after=%d", len(before), len(after))
	}
}
