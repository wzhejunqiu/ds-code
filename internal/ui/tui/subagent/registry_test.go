package subagent

import "testing"

func TestResolveActiveAgentType_viewingDetail(t *testing.T) {
	r := &Registry{}
	r.Start("a1", "label", "prompt", "Explore", false)
	if got := r.ResolveActiveAgentType("a1"); got != "Explore" {
		t.Fatalf("viewing detail = %q, want Explore", got)
	}
}

func TestResolveActiveAgentType_singleRunning(t *testing.T) {
	r := &Registry{}
	r.Start("a1", "label", "prompt", "Explore", true)
	if got := r.ResolveActiveAgentType(""); got != "Explore" {
		t.Fatalf("single running = %q, want Explore", got)
	}
}

func TestResolveActiveAgentType_multipleRunning(t *testing.T) {
	r := &Registry{}
	r.Start("a1", "one", "p1", "Explore", true)
	r.Start("a2", "two", "p2", "Plan", true)
	if got := r.ResolveActiveAgentType(""); got != "" {
		t.Fatalf("multiple running should return empty, got %q", got)
	}
}
