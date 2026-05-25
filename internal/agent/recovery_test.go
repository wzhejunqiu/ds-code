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
