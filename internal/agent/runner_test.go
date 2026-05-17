package agent_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/mock"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
)

func TestRunner_multiRoundTool(t *testing.T) {
	cfg := testConfig()
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/foo.txt", []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := session.NewMemoryStore()
	sess, err := store.NewSession("deepseek-v4-pro", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}

	perm := permission.NewEngine("auto", dir, false)
	reg := tool.NewRegistry()
	reg.Register(&builtin.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false})

	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{{
					ID:        "call_1",
					Name:      "read_file",
					Arguments: `{"path":"foo.txt"}`,
				}},
				ReasoningContent: "thinking",
				FinishReason:     "tool_calls",
			},
			{
				Content:      "found main",
				FinishReason: "stop",
			},
		},
	}

	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &agent.Runner{
		LLM:      mockLLM,
		Tools:    reg,
		Perm:     perm,
		Sessions: store,
		Context:  ctxSvc,
		Cfg:      cfg,
		MaxTurns: 5,
		Out:      &bytes.Buffer{},
	}

	res, err := r.RunTurn(context.Background(), sess.ID, "find main", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalContent != "found main" {
		t.Fatalf("content = %q", res.FinalContent)
	}
	if len(mockLLM.Calls) != 2 {
		t.Fatalf("llm calls = %d, want 2", len(mockLLM.Calls))
	}
	msgs, _ := store.ListMessages(context.Background(), sess.ID)
	if len(msgs) < 4 {
		t.Fatalf("messages = %d, want >= 4", len(msgs))
	}
}

func TestRunner_contextTooLong_retriesAfterCompact(t *testing.T) {
	cfg := testConfig()
	store := session.NewMemoryStore()
	sess, err := store.NewSession("deepseek-v4-pro", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}

	mockLLM := &mock.Client{
		Errors: []error{fmt.Errorf("context length exceeded")},
		Responses: []*llm.Response{
			{Content: "after compact", FinishReason: "stop"},
		},
	}
	perm := permission.NewEngine("auto", t.TempDir(), false)
	reg := tool.NewRegistry()
	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, LLM: mockLLM, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &agent.Runner{
		LLM:      mockLLM,
		Tools:    reg,
		Perm:     perm,
		Sessions: store,
		Context:  ctxSvc,
		Cfg:      cfg,
		MaxTurns: 5,
	}

	res, err := r.RunTurn(context.Background(), sess.ID, "hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalContent != "after compact" {
		t.Fatalf("content = %q", res.FinalContent)
	}
	if len(mockLLM.Calls) != 2 {
		t.Fatalf("llm calls = %d, want 2", len(mockLLM.Calls))
	}
}

func TestRunner_cancelledContext(t *testing.T) {
	cfg := testConfig()
	store := session.NewMemoryStore()
	sess, err := store.NewSession("deepseek-v4-pro", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockLLM := &mock.Client{
		Responses: []*llm.Response{{Content: "unused", FinishReason: "stop"}},
	}
	perm := permission.NewEngine("auto", t.TempDir(), false)
	reg := tool.NewRegistry()
	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &agent.Runner{
		LLM:      mockLLM,
		Tools:    reg,
		Perm:     perm,
		Sessions: store,
		Context:  ctxSvc,
		Cfg:      cfg,
		MaxTurns: 5,
	}

	_, err = r.RunTurn(ctx, sess.ID, "hello", nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestRunner_cancelledDuringToolRound(t *testing.T) {
	cfg := testConfig()
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/foo.txt", []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := session.NewMemoryStore()
	sess, err := store.NewSession("deepseek-v4-pro", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inner := &mock.Client{
		Responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{{
					ID:        "call_1",
					Name:      "read_file",
					Arguments: `{"path":"foo.txt"}`,
				}},
				FinishReason: "tool_calls",
			},
			{Content: "should not run", FinishReason: "stop"},
		},
	}
	llmClient := &cancelOnNthChat{n: 2, cancel: cancel, inner: inner}
	perm := permission.NewEngine("auto", dir, false)
	reg := tool.NewRegistry()
	reg.Register(&builtin.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false})
	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &agent.Runner{
		LLM:      llmClient,
		Tools:    reg,
		Perm:     perm,
		Sessions: store,
		Context:  ctxSvc,
		Cfg:      cfg,
		MaxTurns: 5,
	}

	_, err = r.RunTurn(ctx, sess.ID, "read foo", nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

type cancelOnNthChat struct {
	n      int
	calls  int
	cancel context.CancelFunc
	inner  *mock.Client
}

func (c *cancelOnNthChat) Chat(ctx context.Context, req llm.Request) (*llm.Response, error) {
	c.calls++
	if c.calls >= c.n {
		c.cancel()
	}
	return c.inner.Chat(ctx, req)
}

func TestRunner_permissionDenied(t *testing.T) {
	cfg := testConfig()
	store := session.NewMemoryStore()
	sess, err := store.NewSession("deepseek-v4-pro", "max", "enabled", "readonly", "agent")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	perm := permission.NewEngine("readonly", dir, false)
	reg := tool.NewRegistry()
	reg.Register(&builtin.WriteFileTool{Cfg: cfg, Perm: perm, Strict: false})

	mockLLM := &mock.Client{
		Responses: []*llm.Response{{
			ToolCalls: []llm.ToolCall{{
				ID:        "call_1",
				Name:      "write_file",
				Arguments: `{"path":"out.txt","content":"x"}`,
			}},
			FinishReason: "tool_calls",
		}},
	}
	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &agent.Runner{
		LLM:      mockLLM,
		Tools:    reg,
		Perm:     perm,
		Sessions: store,
		Context:  ctxSvc,
		Cfg:      cfg,
		MaxTurns: 5,
	}

	_, err = r.RunTurn(context.Background(), sess.ID, "write", nil)
	if err != nil {
		t.Fatal(err)
	}
	msgs, _ := store.ListMessages(context.Background(), sess.ID)
	var toolMsg string
	for _, m := range msgs {
		if m.Role == role.Tool {
			toolMsg = m.Content
			break
		}
	}
	if !strings.Contains(strings.ToLower(toolMsg), "permission denied") {
		t.Fatalf("tool result = %q, want permission denial", toolMsg)
	}
}

func TestRunner_exceededMaxSubRounds(t *testing.T) {
	cfg := testConfig()
	store := session.NewMemoryStore()
	sess, err := store.NewSession("deepseek-v4-pro", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}

	toolResp := &llm.Response{
		ToolCalls: []llm.ToolCall{{
			ID:        "call_1",
			Name:      "read_file",
			Arguments: `{"path":"missing.txt"}`,
		}},
		FinishReason: "tool_calls",
	}
	responses := make([]*llm.Response, 0, 4)
	for range 4 {
		responses = append(responses, toolResp)
	}
	mockLLM := &mock.Client{Responses: responses}

	perm := permission.NewEngine("auto", t.TempDir(), false)
	reg := tool.NewRegistry()
	reg.Register(&builtin.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false})
	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &agent.Runner{
		LLM:      mockLLM,
		Tools:    reg,
		Perm:     perm,
		Sessions: store,
		Context:  ctxSvc,
		Cfg:      cfg,
		MaxTurns: 3,
	}

	_, err = r.RunTurn(context.Background(), sess.ID, "loop", nil)
	if err == nil {
		t.Fatal("expected max sub-rounds error")
	}
	if !strings.Contains(err.Error(), "exceeded max sub-rounds") {
		t.Fatalf("err = %v", err)
	}
}

func testConfig() *config.Config {
	return &config.Config{
		LLM: config.LLMConfig{MaxTokens: 4096, StrictTools: false},
		Context: config.ContextConfig{
			ToolResultMaxChars: 100000,
		},
		Tools: config.ToolsConfig{
			ReadFile: config.ReadFileToolConfig{MaxLines: 500},
			Grep:     config.GrepToolConfig{HeadLimit: 200},
		},
		Agent: config.AgentConfig{MaxTurns: 25},
	}
}
