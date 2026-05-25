package spawn

import (
	"context"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

func TestDetachSpawnContext_preservesForkContext(t *testing.T) {
	parent := agent.WithRenderedSystem(context.Background(), "rendered-system")
	parent = agent.WithForkContext(parent, agent.ForkContext{
		ParentMessages: []llm.Message{{Role: role.User, Content: "parent"}},
	})
	parent = WithQuerySource(parent, QuerySourceFork)

	ctx, cancel := DetachSpawnContext(parent)
	defer cancel()

	if got := agent.RenderedSystemFromContext(ctx); got != "rendered-system" {
		t.Fatalf("RenderedSystem = %q, want rendered-system", got)
	}
	fc, ok := agent.ForkContextFromContext(ctx)
	if !ok || len(fc.ParentMessages) != 1 {
		t.Fatalf("ForkContext missing or empty: ok=%v fc=%+v", ok, fc)
	}
	if QuerySourceFromContext(ctx) != QuerySourceFork {
		t.Fatalf("QuerySource = %q, want %q", QuerySourceFromContext(ctx), QuerySourceFork)
	}

	// Parent cancellation must not cancel detached child.
	cancelParent, cancelParentFn := context.WithCancel(parent)
	defer cancelParentFn()
	_ = cancelParent
	child, childCancel := DetachSpawnContext(cancelParent)
	defer childCancel()
	cancelParentFn()
	if child.Err() != nil {
		t.Fatalf("detached child cancelled with parent: %v", child.Err())
	}
}
