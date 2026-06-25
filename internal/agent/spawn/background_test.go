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

func TestSyncTimeout_returnsError(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot:    dir,
		ProjectDataDir: dir,
		LLM:            config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:        config.ContextConfig{ToolResultMaxChars: 50_000},
		Tools: config.ToolsConfig{Agent: config.AgentToolConfig{
			SyncTimeout:     50 * time.Millisecond,
			SummaryMaxChars: 8000,
		}},
		Agent: config.AgentConfig{MaxTurns: 5},
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

	out, err := svc.Handle(context.Background(), agent.ToolInvocation{
		SessionID: parent.ID, ToolCallID: "call-timeout",
	}, spawn.Params{SubagentType: "Explore", Description: "slow", Prompt: "work"}, true)
	if err == nil {
		t.Fatalf("expected sync timeout error, got output %q", out)
	}
	if strings.Contains(out, "async_launched") {
		t.Fatalf("sync timeout must not promote to async, got %q", out)
	}
}

func TestBackgroundManager_enqueuePrioLater(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot:    dir,
		ProjectDataDir: dir,
		LLM:            config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:        config.ContextConfig{ToolResultMaxChars: 50_000},
		Tools:          config.ToolsConfig{Agent: config.AgentToolConfig{SummaryMaxChars: 8000}},
		Agent:          config.AgentConfig{MaxTurns: 5},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{{Content: "done", FinishReason: "stop"}},
	}
	sub := subagentstore.NewMemoryStore()
	perm := permission.NewEngine("readonly", dir, false)
	reg := tool.NewRegistry()
	register.ExploreTools(reg, cfg, perm, nil, false)

	run, err := sub.CreateRun(context.Background(), subagentstore.CreateRunParams{
		ParentSessionID:  "parent",
		ParentToolCallID: "tc-async",
		AgentType:        "Explore",
		SpawnKind:        subagentstore.SpawnAsync,
		Label:            "bg",
		Prompt:           "work",
		Model:            "m",
		Background:       true,
	})
	if err != nil {
		t.Fatal(err)
	}

	q := spawn.NewNotificationQueue()
	bm := spawn.NewBackgroundManager(q)
	def, err := spawn.NewRegistry().Resolve("Explore")
	if err != nil {
		t.Fatal(err)
	}
	bm.Start(agent.WithActiveTurn(context.Background()), cfg, mockLLM, run, def, perm, reg, sub, nil, nil, nil, nil)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && bm.RunningCount() > 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if bm.RunningCount() > 0 {
		t.Fatal("background agent did not finish in time")
	}

	later := q.Drain(spawn.PrioLater)
	if len(later) != 1 {
		t.Fatalf("expected 1 PrioLater notification, got %d", len(later))
	}
	if later[0].AgentID != run.ID {
		t.Fatalf("AgentID = %q, want %q", later[0].AgentID, run.ID)
	}
	if next := q.Drain(spawn.PrioNext); len(next) != 0 {
		t.Fatalf("expected no PrioNext, got %d", len(next))
	}
}

func TestBackgroundManager_enqueuePrioNextWhenIdle(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot:    dir,
		ProjectDataDir: dir,
		LLM:            config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:        config.ContextConfig{ToolResultMaxChars: 50_000},
		Tools:          config.ToolsConfig{Agent: config.AgentToolConfig{SummaryMaxChars: 8000}},
		Agent:          config.AgentConfig{MaxTurns: 5},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{{Content: "done", FinishReason: "stop"}},
	}
	sub := subagentstore.NewMemoryStore()
	perm := permission.NewEngine("readonly", dir, false)
	reg := tool.NewRegistry()
	register.ExploreTools(reg, cfg, perm, nil, false)

	run, err := sub.CreateRun(context.Background(), subagentstore.CreateRunParams{
		ParentSessionID:  "parent",
		ParentToolCallID: "tc-async",
		AgentType:        "Explore",
		SpawnKind:        subagentstore.SpawnAsync,
		Label:            "bg",
		Prompt:           "work",
		Model:            "m",
		Background:       true,
	})
	if err != nil {
		t.Fatal(err)
	}

	q := spawn.NewNotificationQueue()
	bm := spawn.NewBackgroundManager(q)
	def, err := spawn.NewRegistry().Resolve("Explore")
	if err != nil {
		t.Fatal(err)
	}
	bm.Start(context.Background(), cfg, mockLLM, run, def, perm, reg, sub, nil, nil, nil, nil)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && bm.RunningCount() > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	next := q.Drain(spawn.PrioNow)
	if len(next) != 1 {
		t.Fatalf("expected 1 PrioNow notification, got %d", len(next))
	}
	if later := q.Drain(spawn.PrioLater); len(later) != 0 {
		t.Fatalf("expected no PrioLater, got %d", len(later))
	}
}

func TestBackgroundManager_enqueuePrioNowAfterTurnEnds(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot:    dir,
		ProjectDataDir: dir,
		LLM:            config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:        config.ContextConfig{ToolResultMaxChars: 50_000},
		Tools:          config.ToolsConfig{Agent: config.AgentToolConfig{SummaryMaxChars: 8000}},
		Agent:          config.AgentConfig{MaxTurns: 5},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{{Content: "done", FinishReason: "stop"}},
		Delay:     150 * time.Millisecond,
	}
	sub := subagentstore.NewMemoryStore()
	perm := permission.NewEngine("readonly", dir, false)
	reg := tool.NewRegistry()
	register.ExploreTools(reg, cfg, perm, nil, false)

	run, err := sub.CreateRun(context.Background(), subagentstore.CreateRunParams{
		ParentSessionID:  "parent",
		ParentToolCallID: "tc-async",
		AgentType:        "Explore",
		SpawnKind:        subagentstore.SpawnAsync,
		Label:            "bg",
		Prompt:           "work",
		Model:            "m",
		Background:       true,
	})
	if err != nil {
		t.Fatal(err)
	}

	q := spawn.NewNotificationQueue()
	bm := spawn.NewBackgroundManager(q)
	def, err := spawn.NewRegistry().Resolve("Explore")
	if err != nil {
		t.Fatal(err)
	}
	parentCtx := agent.WithActiveTurn(context.Background())
	bm.Start(parentCtx, cfg, mockLLM, run, def, perm, reg, sub, nil, nil, nil, nil)
	agent.WithoutActiveTurn(parentCtx)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && bm.RunningCount() > 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if bm.RunningCount() > 0 {
		t.Fatal("background agent did not finish in time")
	}

	now := q.Drain(spawn.PrioNow)
	if len(now) != 1 {
		t.Fatalf("expected 1 PrioNow notification after turn ended, got %d", len(now))
	}
	if now[0].AgentID != run.ID {
		t.Fatalf("AgentID = %q, want %q", now[0].AgentID, run.ID)
	}
	if later := q.Drain(spawn.PrioLater); len(later) != 0 {
		t.Fatalf("expected no PrioLater, got %d", len(later))
	}
}
