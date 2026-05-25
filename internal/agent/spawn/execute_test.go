package spawn

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

func TestResolveSubagentMaxTurns_fromConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Subagent.MaxTurns = 12
	if got := resolveSubagentMaxTurns(cfg); got != 12 {
		t.Fatalf("expected 12, got %d", got)
	}
}

func TestResolveSubagentMaxTurns_fallback(t *testing.T) {
	cfg := &config.Config{}
	if got := resolveSubagentMaxTurns(cfg); got != 8 {
		t.Fatalf("expected fallback 8, got %d", got)
	}
}
