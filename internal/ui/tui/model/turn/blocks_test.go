package turn

import (
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
)

func TestAppendPlanningBlock_preservesExistingTimer(t *testing.T) {
	started := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	s := &state.State{
		Chat: []chat.Block{{
			Role:              chat.RolePlanning,
			Streaming:         true,
			PlanningStartedAt: started,
		}},
	}
	AppendPlanningBlock(s)
	if len(s.Chat) != 1 {
		t.Fatalf("blocks = %d, want 1", len(s.Chat))
	}
	if !s.Chat[0].PlanningStartedAt.Equal(started) {
		t.Fatalf("PlanningStartedAt reset to %v, want %v", s.Chat[0].PlanningStartedAt, started)
	}
}

func TestAppendPlanningBlock_replacesNonPlanningTail(t *testing.T) {
	s := &state.State{
		Chat: []chat.Block{{Role: chat.RoleUser}},
	}
	AppendPlanningBlock(s)
	if len(s.Chat) != 2 || s.Chat[1].Role != chat.RolePlanning {
		t.Fatalf("chat = %+v", s.Chat)
	}
}
