package agent

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

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
