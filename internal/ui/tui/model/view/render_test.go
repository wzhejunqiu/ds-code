package view

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

func TestFooterLeft_backgroundAgentsBadge(t *testing.T) {
	n := 0
	s := &state.State{
		Deps: &deps.Deps{
			BackgroundAgents: func() int {
				return n
			},
		},
	}
	if got := FooterLeft(s); strings.Contains(got, "agents running") {
		t.Fatalf("expected no badge when zero, got %q", got)
	}
	n = 2
	got := FooterLeft(s)
	if !strings.Contains(got, "2 agents running in background") {
		t.Fatalf("expected badge, got %q", got)
	}
}
