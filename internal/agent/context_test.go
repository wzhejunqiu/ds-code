package agent_test

import (
	"context"
	"testing"

	"github.com/hejunqiu/ds-code/internal/agent"
)

func TestSubagentToolCallbacks_forwardsToolEvents(t *testing.T) {
	parent := &agent.TurnCallbacks{}
	var subStarts, subEnds int
	parent.OnSubagentToolStart = func(_, _, _, _ string) { subStarts++ }
	parent.OnSubagentToolEnd = func(_, _, _, _, _ string, _ bool) { subEnds++ }
	sub := agent.SubagentToolCallbacks(parent, "sa-1")
	sub.OnToolStart("read_file", "path=x", "")
	sub.OnToolEnd("read_file", "path=x", "", "ok", false)
	if subStarts != 1 || subEnds != 1 {
		t.Fatalf("sub starts=%d ends=%d", subStarts, subEnds)
	}
}

func TestTurnCallbacksFromContext(t *testing.T) {
	parent := &agent.TurnCallbacks{}
	ctx := agent.WithTurnCallbacks(context.Background(), parent)
	if got := agent.TurnCallbacksFromContext(ctx); got != parent {
		t.Fatal("expected same callbacks pointer")
	}
	if got := agent.TurnCallbacksFromContext(context.Background()); got != nil {
		t.Fatal("expected nil without context value")
	}
}
