package agent_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/mock"
	"github.com/hejunqiu/ds-code/internal/permission"
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
