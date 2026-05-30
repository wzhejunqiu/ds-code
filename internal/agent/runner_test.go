package agent_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/read_file"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/write_file"
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
	reg.Register(&read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false})

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
	reg.Register(&read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false})
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
	reg.Register(&write_file.WriteFileTool{Cfg: cfg, Perm: perm, Strict: false})

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
	for range 3 {
		responses = append(responses, toolResp)
	}
	responses = append(responses, &llm.Response{
		Content:      "Reached tool limit; here is progress so far.",
		FinishReason: "stop",
	})
	mockLLM := &mock.Client{Responses: responses}

	perm := permission.NewEngine("auto", t.TempDir(), false)
	reg := tool.NewRegistry()
	reg.Register(&read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false})
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

	result, err := r.RunTurn(context.Background(), sess.ID, "loop", nil)
	if err != nil {
		t.Fatalf("expected soft landing, got err: %v", err)
	}
	if result == nil || result.SubRounds != 3 {
		t.Fatalf("result = %+v, want SubRounds=3", result)
	}
	msgs, _ := store.ListMessages(context.Background(), sess.ID)
	var systemMsg, summaryAssistant string
	const summaryPrompt = "Summarize what you've accomplished, what remains unfinished, and suggested next steps. Do not call any tools."
	for _, m := range msgs {
		if m.Role == role.User && strings.Contains(m.Content, summaryPrompt) {
			t.Fatalf("unexpected persisted summary user prompt: %q", m.Content)
		}
		if m.Role == role.System && strings.Contains(m.Content, "Reached max sub-rounds") {
			systemMsg = m.Content
		}
		if m.Role == role.Assistant && m.Content == "Reached tool limit; here is progress so far." {
			summaryAssistant = m.Content
		}
	}
	if systemMsg == "" {
		t.Fatal("expected system event for max sub-rounds")
	}
	if summaryAssistant == "" {
		t.Fatal("expected summary assistant message")
	}
}

func TestRunner_exceededMaxSubRounds_onUsageUpdate(t *testing.T) {
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
		Usage:        llm.Usage{PromptTokens: 10, CompletionTokens: 1},
	}
	responses := make([]*llm.Response, 0, 4)
	for range 3 {
		responses = append(responses, toolResp)
	}
	responses = append(responses, &llm.Response{
		Content:          "summary",
		FinishReason:     "stop",
		Usage:            llm.Usage{PromptTokens: 20, CompletionTokens: 5},
	})
	mockLLM := &mock.Client{Responses: responses}

	perm := permission.NewEngine("auto", t.TempDir(), false)
	reg := tool.NewRegistry()
	reg.Register(&read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false})
	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	var usageCalls int
	var last llm.Usage
	cb := &agent.TurnCallbacks{
		OnUsageUpdate: func(u llm.Usage) {
			usageCalls++
			last = u
		},
	}
	r := &agent.Runner{
		LLM:      mockLLM,
		Tools:    reg,
		Perm:     perm,
		Sessions: store,
		Context:  ctxSvc,
		Cfg:      cfg,
		MaxTurns: 3,
	}

	if _, err := r.RunTurn(context.Background(), sess.ID, "loop", cb); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if usageCalls < 4 {
		t.Fatalf("OnUsageUpdate calls = %d, want >= 4 (3 tool rounds + summary)", usageCalls)
	}
	if last.PromptTokens != 20 || last.CompletionTokens != 5 {
		t.Fatalf("last usage = %+v, want summary round tokens", last)
	}
}

func TestRunner_exceededMaxSubRounds_summaryFailureDegraded(t *testing.T) {
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
	responses := make([]*llm.Response, 0, 3)
	for range 3 {
		responses = append(responses, toolResp)
	}
	inner := &mock.Client{Responses: responses}
	mockLLM := &summaryFailAfterLLM{inner: inner, after: 3, err: errors.New("summary LLM unavailable")}

	perm := permission.NewEngine("auto", t.TempDir(), false)
	reg := tool.NewRegistry()
	reg.Register(&read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false})
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

	const summaryPrompt = "Summarize what you've accomplished, what remains unfinished, and suggested next steps. Do not call any tools."
	result, err := r.RunTurn(context.Background(), sess.ID, "loop", nil)
	if err != nil {
		t.Fatalf("expected degraded soft landing, got err: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}

	msgs, _ := store.ListMessages(context.Background(), sess.ID)
	var systemMsg, fallbackAssistant string
	for _, m := range msgs {
		if m.Role == role.User && strings.Contains(m.Content, summaryPrompt) {
			t.Fatalf("unexpected persisted summary user prompt: %q", m.Content)
		}
		if m.Role == role.System && strings.Contains(m.Content, "Reached max sub-rounds") {
			systemMsg = m.Content
		}
		if m.Role == role.Assistant && strings.Contains(m.Content, "Could not summarize progress") {
			fallbackAssistant = m.Content
		}
	}
	if systemMsg == "" {
		t.Fatal("expected system event for max sub-rounds")
	}
	if fallbackAssistant == "" {
		t.Fatal("expected fallback assistant message")
	}
}

func TestRunner_exceededMaxSubRounds_stopHookTransition(t *testing.T) {
	dir := t.TempDir()
	dsCode := filepath.Join(dir, ".ds-code")
	if err := os.MkdirAll(dsCode, 0o700); err != nil {
		t.Fatal(err)
	}
	hookOut := filepath.Join(dir, "stop_input.json")
	script := filepath.Join(dir, "stop.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s' \"$HOOK_INPUT\" > \""+hookOut+"\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	hooksJSON := `[{"event":"Stop","command":"` + script + `"}]`
	if err := os.WriteFile(filepath.Join(dsCode, "hooks.json"), []byte(hooksJSON), 0o600); err != nil {
		t.Fatal(err)
	}

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
	for range 3 {
		responses = append(responses, toolResp)
	}
	responses = append(responses, &llm.Response{
		Content:      "summary done",
		FinishReason: "stop",
	})
	mockLLM := &mock.Client{Responses: responses}

	perm := permission.NewEngine("auto", t.TempDir(), false)
	reg := tool.NewRegistry()
	reg.Register(&read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false})
	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &agent.Runner{
		LLM:      mockLLM,
		Tools:    reg,
		Perm:     perm,
		Sessions: store,
		Context:  ctxSvc,
		Cfg:      cfg,
		MaxTurns: 3,
		Hooks:    agent.LoadHooks(dir),
	}

	if _, err := r.RunTurn(context.Background(), sess.ID, "loop", nil); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	raw, err := os.ReadFile(hookOut)
	if err != nil {
		t.Fatalf("read hook output: %v", err)
	}
	var hookInput map[string]string
	if err := json.Unmarshal(raw, &hookInput); err != nil {
		t.Fatalf("parse hook input: %v", err)
	}
	if hookInput["transition"] != "max_turns" {
		t.Fatalf("transition = %q, want max_turns", hookInput["transition"])
	}
	if !strings.Contains(hookInput["error"], "exceeded max sub-rounds") {
		t.Fatalf("error = %q, want exceeded max sub-rounds", hookInput["error"])
	}
}

func testConfig() *config.Config {
	return &config.Config{
		LLM: config.LLMConfig{MaxTokens: 4096, StrictTools: false},
		Context: config.ContextConfig{
			ToolResultMaxChars: 100000,
		},
		Tools: config.ToolsConfig{
			ReadFile: config.ReadFileToolConfig{MaxLines: 2000},
			Grep:     config.GrepToolConfig{HeadLimit: 200},
		},
		Agent: config.AgentConfig{MaxTurns: 25},
	}
}

// summaryFailAfterLLM fails Chat after a fixed number of successful inner calls.
type summaryFailAfterLLM struct {
	inner *mock.Client
	after int
	err   error
	calls int
}

func (m *summaryFailAfterLLM) Chat(ctx context.Context, req llm.Request) (*llm.Response, error) {
	m.calls++
	if m.calls > m.after {
		return nil, m.err
	}
	return m.inner.Chat(ctx, req)
}
