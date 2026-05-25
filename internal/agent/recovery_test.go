package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func recoveryTestConfig() *config.Config {
	return &config.Config{
		LLM:     config.LLMConfig{MaxTokens: 4096},
		Context: config.ContextConfig{ToolResultMaxChars: 100000},
		Agent:   config.AgentConfig{MaxTurns: 25},
	}
}

func TestIsLengthFinishReason(t *testing.T) {
	for _, r := range []string{"length", "LENGTH", "max_tokens"} {
		if !isLengthFinishReason(r) {
			t.Fatalf("expected length reason for %q", r)
		}
	}
	if isLengthFinishReason("stop") {
		t.Fatal("stop should not match")
	}
}

func TestFallbackModel_prefersSubagent(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.FallbackModel = "main-fb"
	cfg.LLM.Subagent.FallbackModel = "sub-fb"
	r := &Runner{Cfg: cfg}
	if got := r.fallbackModel(); got != "sub-fb" {
		t.Fatalf("expected sub-fb, got %s", got)
	}
	cfg.LLM.Subagent.FallbackModel = ""
	if got := r.fallbackModel(); got != "main-fb" {
		t.Fatalf("expected main-fb, got %s", got)
	}
}

func TestChatWithRecovery_retriesAfterCompact(t *testing.T) {
	cfg := recoveryTestConfig()
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	mockLLM := &mock.Client{
		Errors:    []error{fmt.Errorf("context length exceeded")},
		Responses: []*llm.Response{{Content: "ok"}},
	}
	perm := permission.NewEngine("auto", t.TempDir(), false)
	reg := tool.NewRegistry()
	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, LLM: mockLLM, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &Runner{LLM: mockLLM, Context: ctxSvc, Cfg: cfg, Sessions: store}

	req := llm.Request{Messages: []llm.Message{{Role: role.User, Content: "hi"}}}
	state := &LoopState{}
	resp, err := r.chatWithRecovery(context.Background(), sess.ID, req, state)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok" || len(mockLLM.Calls) != 2 {
		t.Fatalf("resp=%+v calls=%d", resp, len(mockLLM.Calls))
	}
	if !state.CompactRetried {
		t.Fatal("expected CompactRetried after context-too-long recovery")
	}
}

func TestChatWithRecovery_emptyResponseRetries(t *testing.T) {
	cfg := recoveryTestConfig()
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{Content: "", FinishReason: "stop"},
			{Content: "ok", FinishReason: "stop"},
		},
	}
	perm := permission.NewEngine("auto", t.TempDir(), false)
	reg := tool.NewRegistry()
	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, LLM: mockLLM, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &Runner{LLM: mockLLM, Context: ctxSvc, Cfg: cfg, Sessions: store}

	req := llm.Request{Messages: []llm.Message{{Role: role.User, Content: "hi"}}}
	state := &LoopState{}
	resp, err := r.chatWithRecovery(context.Background(), sess.ID, req, state)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok" || len(mockLLM.Calls) != 2 {
		t.Fatalf("resp=%+v calls=%d", resp, len(mockLLM.Calls))
	}
	if state.OutputRecoveryCount != 1 {
		t.Fatalf("expected OutputRecoveryCount=1, got %d", state.OutputRecoveryCount)
	}
}

func TestChatWithRecovery_transientNetworkRetries(t *testing.T) {
	cfg := recoveryTestConfig()
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	mockLLM := &mock.Client{
		Errors:    []error{fmt.Errorf("connection reset by peer")},
		Responses: []*llm.Response{{Content: "ok", FinishReason: "stop"}},
	}
	perm := permission.NewEngine("auto", t.TempDir(), false)
	reg := tool.NewRegistry()
	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, LLM: mockLLM, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &Runner{LLM: mockLLM, Context: ctxSvc, Cfg: cfg, Sessions: store}

	req := llm.Request{Messages: []llm.Message{{Role: role.User, Content: "hi"}}}
	state := &LoopState{}
	resp, err := r.chatWithRecovery(context.Background(), sess.ID, req, state)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok" || len(mockLLM.Calls) != 2 {
		t.Fatalf("resp=%+v calls=%d", resp, len(mockLLM.Calls))
	}
	if state.NetworkRetryCount != 1 {
		t.Fatalf("expected NetworkRetryCount=1, got %d", state.NetworkRetryCount)
	}
	if state.OutputRecoveryCount != 0 {
		t.Fatalf("expected OutputRecoveryCount=0 for network retry, got %d", state.OutputRecoveryCount)
	}
	if len(mockLLM.Calls[0].Messages) != 1 {
		t.Fatalf("network retry should not append continue message, got %d messages", len(mockLLM.Calls[0].Messages))
	}
}

func TestIsEmptyTerminalResponse(t *testing.T) {
	if !isEmptyTerminalResponse(&llm.Response{FinishReason: "stop"}) {
		t.Fatal("empty stop response should trigger recovery")
	}
	if isEmptyTerminalResponse(&llm.Response{Content: "hi", FinishReason: "stop"}) {
		t.Fatal("non-empty content should not trigger empty recovery")
	}
	if isEmptyTerminalResponse(&llm.Response{FinishReason: "length"}) {
		t.Fatal("length finish should use max_tokens path, not empty")
	}
}
