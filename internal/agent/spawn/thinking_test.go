package spawn

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

func TestResolveThinkingType_forkInheritsParent(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Thinking: config.ThinkingConfig{Type: "enabled"},
			Subagent: config.SubagentLLMConfig{Thinking: config.ThinkingConfig{Type: "disabled"}},
		},
	}
	got := resolveThinkingType(cfg, AgentTypeDefinition{Type: AgentTypeFork}, "enabled", true)
	if got != "enabled" {
		t.Fatalf("fork should inherit parent thinking, got %q", got)
	}
}

func TestResolveThinkingType_nonForkDisabled(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{Thinking: config.ThinkingConfig{Type: "enabled"}},
	}
	got := resolveThinkingType(cfg, AgentTypeDefinition{Type: AgentTypeExplore}, "enabled", false)
	if got != "disabled" {
		t.Fatalf("non-fork subagent should disable thinking, got %q", got)
	}
}
