package context_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/deepseek"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

func TestCountBreakdown_invariant(t *testing.T) {
	view := &context.APIContextView{
		SystemPrompt: "You are ds-code",
		AgentsMD:     "do things",
		ToolsJSON:    `[{"type":"function"}]`,
		WindowTokens: deepseek.ContextWindowTokens,
		Messages: []llm.Message{
			{Role: role.User, Content: "hello"},
			{Role: role.Assistant, Content: "hi"},
			{Role: role.Tool, Name: "task", Content: "subagent result"},
		},
	}
	bd, err := context.CountBreakdown(view)
	if err != nil {
		t.Fatal(err)
	}
	if bd.Total() <= 0 {
		t.Fatalf("total = %d", bd.Total())
	}
	if bd.SystemPrompt <= 0 {
		t.Fatal("expected system prompt count > 0")
	}
	if bd.Subagents <= 0 {
		t.Fatal("expected subagent count > 0")
	}
}
