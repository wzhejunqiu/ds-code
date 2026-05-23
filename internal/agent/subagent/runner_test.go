package subagent_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/agent/subagent"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/mock"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session/subagentstore"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/register"
)

func TestRun_readOnlySummary(t *testing.T) {
	dir := t.TempDir()
	content := "package main\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: dir,
		LLM:         config.LLMConfig{MaxTokens: 4096},
		Context:     config.ContextConfig{ToolResultMaxChars: 50_000},
		Tools: config.ToolsConfig{
			ReadFile: config.ReadFileToolConfig{MaxLines: 500},
			Grep:     config.GrepToolConfig{HeadLimit: 50},
			Glob:     config.GlobToolConfig{MaxResults: 50},
			Task:     config.TaskToolConfig{SummaryMaxChars: 8000},
		},
		Agent: config.AgentConfig{MaxTurns: 5},
	}

	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{{
					ID: "c1", Name: "read_file",
					Arguments: `{"path":"main.go"}`,
				}},
				FinishReason: "tool_calls",
			},
			{Content: "Found package main with main().", FinishReason: "stop"},
		},
	}

	sub := subagentstore.NewMemoryStore()
	run, err := sub.CreateRun(context.Background(), subagentstore.CreateRunParams{
		ParentSessionID: "test", ParentToolCallID: "tc-1", Prompt: "inspect main.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	summary, err := subagent.Run(context.Background(), cfg, mockLLM, "inspect main.go", func(reg *tool.Registry) {
		perm := permission.NewEngine("readonly", dir, false)
		register.ExploreTools(reg, cfg, perm, nil, false)
	}, sub, run, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "main") {
		t.Fatalf("summary = %q", summary)
	}
	if len(mockLLM.Calls) < 2 {
		t.Fatalf("expected multi-round LLM, got %d calls", len(mockLLM.Calls))
	}
	for _, req := range mockLLM.Calls {
		for _, def := range req.Tools {
			if def.Name == "shell" || def.Name == "write_file" || def.Name == "apply_patch" {
				t.Fatalf("subagent exposed write tool %s", def.Name)
			}
		}
	}
}
