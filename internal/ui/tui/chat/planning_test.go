package chat

import (
	"strings"
	"testing"
	"time"
)

func TestRenderPlanningBlock(t *testing.T) {
	started := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	now := started.Add(2 * time.Second)
	blocks := []Block{{
		Role:              RolePlanning,
		Streaming:         true,
		PlanningStartedAt: started,
	}}
	out := Render(blocks, 60, now, false)
	if !strings.Contains(out, "Planning next moves") {
		t.Fatalf("missing planning label:\n%s", out)
	}
	if !strings.Contains(out, "2s") {
		t.Fatalf("missing elapsed duration:\n%s", out)
	}
}
