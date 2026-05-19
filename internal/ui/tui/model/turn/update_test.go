package turn

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
)

func hasPlanningRole(blocks []chat.Block) bool {
	for _, b := range blocks {
		if b.Role == chat.RolePlanning {
			return true
		}
	}
	return false
}

func TestUpdatePlanningEnd_clearsMainChatWhileViewingSubagent(t *testing.T) {
	s := &state.State{
		Running:           true,
		MainChat:          []chat.Block{{Role: chat.RoleUser}, {Role: chat.RolePlanning, Streaming: true}},
		SubagentNav:       state.SubagentNavDetail,
		ViewingSubagentID: "sa-1",
	}
	s.Subagents.Start("sa-1", "label", "explore pkg")
	s.SyncDisplayedChat()

	rec := s.Subagents.Get("sa-1")
	if rec == nil {
		t.Fatal("subagent record missing")
	}
	subChatLen := len(rec.Chat)
	if subChatLen == 0 {
		t.Fatal("expected subagent chat blocks")
	}

	UpdatePlanningEnd(s, func() {})

	if hasPlanningRole(s.MainChat) {
		t.Fatalf("MainChat still has planning: %+v", s.MainChat)
	}
	if len(s.MainChat) != 1 || s.MainChat[0].Role != chat.RoleUser {
		t.Fatalf("MainChat = %+v", s.MainChat)
	}
	if len(rec.Chat) != subChatLen {
		t.Fatalf("subagent chat len changed: got %d, want %d", len(rec.Chat), subChatLen)
	}
	if len(s.Chat) != subChatLen {
		t.Fatalf("displayed Chat len = %d, want %d", len(s.Chat), subChatLen)
	}
	if s.Chat[0].Role != chat.RoleUser {
		t.Fatalf("displayed Chat should still be subagent transcript, got role %v", s.Chat[0].Role)
	}
}

func TestUpdatePlanningEnd_skippedWhenInterrupted(t *testing.T) {
	s := &state.State{
		Running: true,
		MainChat: []chat.Block{
			{Role: chat.RoleUser},
			{Role: chat.RolePlanning, Streaming: true},
			{Role: chat.RoleInterrupt},
		},
	}
	s.Chat = s.MainChat

	UpdatePlanningEnd(s, func() {})

	if !hasPlanningRole(s.MainChat) {
		t.Fatalf("MainChat planning cleared while interrupted: %+v", s.MainChat)
	}
}

func TestUpdatePlanningEnd_skippedWhenNotRunning(t *testing.T) {
	s := &state.State{
		Running:  false,
		MainChat: []chat.Block{{Role: chat.RoleUser}, {Role: chat.RolePlanning, Streaming: true}},
	}
	s.Chat = s.MainChat

	UpdatePlanningEnd(s, func() {})

	if !hasPlanningRole(s.MainChat) {
		t.Fatalf("MainChat planning cleared while not running: %+v", s.MainChat)
	}
}
