package subagent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/agent/subagent"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/mock"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/session/subagentstore"
)

func TestGenerateSessionTitle_returnsNormalizedTitle(t *testing.T) {
	cfg := &config.Config{
		ProjectRoot: t.TempDir(),
		LLM:         config.LLMConfig{MaxTokens: 1024},
		Agent:       config.AgentConfig{MaxTurns: 3},
		Tools:       config.ToolsConfig{Task: config.TaskToolConfig{SummaryMaxChars: 200}},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{Content: "分析 client.go 作用", FinishReason: "stop"},
		},
	}

	sub := subagentstore.NewMemoryStore()
	title, err := subagent.GenerateSessionTitle(context.Background(), cfg, mockLLM, sub, "parent-sess-1", "What does client.go do?")
	if err != nil {
		t.Fatal(err)
	}
	if title == "" {
		t.Fatal("expected non-empty title")
	}
	if strings.Contains(title, "\n") {
		t.Fatalf("title must be single line: %q", title)
	}
	if len([]rune(title)) > session.MaxTitleRunes {
		t.Fatalf("title length = %d runes, want <= %d", len([]rune(title)), session.MaxTitleRunes)
	}
}

func TestGenerateSessionTitle_truncatesLongOutput(t *testing.T) {
	cfg := &config.Config{
		ProjectRoot: t.TempDir(),
		LLM:         config.LLMConfig{MaxTokens: 1024},
		Agent:       config.AgentConfig{MaxTurns: 3},
		Tools:       config.ToolsConfig{Task: config.TaskToolConfig{SummaryMaxChars: 200}},
	}
	long := strings.Repeat("标", session.MaxTitleRunes+20)
	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{Content: long, FinishReason: "stop"},
		},
	}

	sub := subagentstore.NewMemoryStore()
	title, err := subagent.GenerateSessionTitle(context.Background(), cfg, mockLLM, sub, "parent-sess-long", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(title)) > session.MaxTitleRunes {
		t.Fatalf("title length = %d runes, want <= %d", len([]rune(title)), session.MaxTitleRunes)
	}
}

func TestGenerateSessionTitle_acceptsEnglishOutput(t *testing.T) {
	cfg := &config.Config{
		ProjectRoot: t.TempDir(),
		LLM:         config.LLMConfig{MaxTokens: 1024},
		Agent:       config.AgentConfig{MaxTurns: 3},
		Tools:       config.ToolsConfig{Task: config.TaskToolConfig{SummaryMaxChars: 200}},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{Content: "Explain client.go", FinishReason: "stop"},
		},
	}

	sub := subagentstore.NewMemoryStore()
	title, err := subagent.GenerateSessionTitle(context.Background(), cfg, mockLLM, sub, "parent-sess-2", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if title != "Explain client.go" {
		t.Fatalf("title = %q", title)
	}
}

func TestGenerateSessionTitle_idempotentSecondCall(t *testing.T) {
	cfg := &config.Config{
		ProjectRoot: t.TempDir(),
		LLM:         config.LLMConfig{MaxTokens: 1024},
		Agent:       config.AgentConfig{MaxTurns: 3},
		Tools:       config.ToolsConfig{Task: config.TaskToolConfig{SummaryMaxChars: 200}},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{Content: "缓存标题", FinishReason: "stop"},
		},
	}

	sub := subagentstore.NewMemoryStore()
	ctx := context.Background()
	first, err := subagent.GenerateSessionTitle(ctx, cfg, mockLLM, sub, "parent-sess-3", "hello")
	if err != nil {
		t.Fatal(err)
	}
	second, err := subagent.GenerateSessionTitle(ctx, cfg, mockLLM, sub, "parent-sess-3", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("title = %q, cached = %q", first, second)
	}
	if len(mockLLM.Calls) != 1 {
		t.Fatalf("LLM calls = %d, want 1 (cached title run)", len(mockLLM.Calls))
	}
}

func TestGenerateSessionTitle_disablesThinking(t *testing.T) {
	cfg := &config.Config{
		ProjectRoot: t.TempDir(),
		LLM: config.LLMConfig{
			MaxTokens: 1024,
			Subagent: config.SubagentLLMConfig{
				Thinking: config.ThinkingConfig{Type: "enabled"},
			},
		},
		Agent: config.AgentConfig{MaxTurns: 3},
		Tools: config.ToolsConfig{Task: config.TaskToolConfig{SummaryMaxChars: 200}},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{Content: "标题", FinishReason: "stop"},
		},
	}

	sub := subagentstore.NewMemoryStore()
	if _, err := subagent.GenerateSessionTitle(context.Background(), cfg, mockLLM, sub, "parent-sess-4", "hello"); err != nil {
		t.Fatal(err)
	}
	if len(mockLLM.Calls) != 1 {
		t.Fatalf("LLM calls = %d, want 1", len(mockLLM.Calls))
	}
	if mockLLM.Calls[0].ThinkingType != "disabled" {
		t.Fatalf("ThinkingType = %q, want disabled", mockLLM.Calls[0].ThinkingType)
	}
	runs, err := sub.ListRuns(context.Background(), "parent-sess-4")
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].ThinkingType != "disabled" {
		t.Fatalf("title run thinking = %+v", runs)
	}
}
