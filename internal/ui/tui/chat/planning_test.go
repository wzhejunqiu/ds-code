package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestRenderPlanningBlock(t *testing.T) {
	started := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	now := started.Add(2 * time.Second)
	blocks := []Block{{
		Role:              RolePlanning,
		Streaming:         true,
		PlanningStartedAt: started,
	}}
	out := Render(blocks, 60, now, false, tool.DisplayContext{})
	if !strings.Contains(out, "规划下一步行动") {
		t.Fatalf("missing planning label:\n%s", out)
	}
	if !strings.Contains(out, "2s") {
		t.Fatalf("missing elapsed duration:\n%s", out)
	}
}
