package task

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

func TestTaskTool_semAcquireRespectsCancel(t *testing.T) {
	sem := newTaskSemaphore(1)
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	go func() {
		done <- sem.Acquire(ctx)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected acquire to fail after cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("acquire blocked after context cancel")
	}
	sem.Release()
}

func TestTaskTool_forwardsSubagentToolCallbacks(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
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
			Task:     config.TaskToolConfig{MaxParallel: 2, SummaryMaxChars: 8000},
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
			{Content: "done", FinishReason: "stop"},
		},
	}

	var mu sync.Mutex
	var nested []string
	parent := &agent.TurnCallbacks{
		OnSubagentToolStart: func(_, name, _, _ string) {
			mu.Lock()
			nested = append(nested, "start:"+name)
			mu.Unlock()
		},
		OnSubagentToolEnd: func(_, name, _, _, _ string, _ bool) {
			mu.Lock()
			nested = append(nested, "end:"+name)
			mu.Unlock()
		},
	}

	tool := NewTaskTool(cfg, permission.NewEngine("agent", dir, false), mockLLM, false, subagentstore.NewMemoryStore())
	ctx := agent.WithToolInvocation(
		agent.WithTurnCallbacks(context.Background(), parent),
		"parent-session", "call-task-1",
	)
	args, _ := json.Marshal(map[string]string{
		"description": "probe",
		"prompt":      "read main.go",
	})
	if _, err := tool.Execute(ctx, args); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(nested) < 2 {
		t.Fatalf("expected nested tool events, got %v", nested)
	}
}
