package turn

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
)

func TestCurrentTurnInterrupted_whileViewingSubagent(t *testing.T) {
	s := &state.State{
		Running: true,
		MainChat: []chat.Block{
			{Role: chat.RoleUser},
			{Role: chat.RoleInterrupt},
		},
		SubagentNav:       state.SubagentNavDetail,
		ViewingSubagentID: "sa-1",
	}
	s.Subagents.Start("sa-1", "label", "explore pkg")
	s.SyncDisplayedChat()

	if !CurrentTurnInterrupted(s) {
		t.Fatal("expected interrupt from MainChat while viewing subagent")
	}
	if EventsAllowed(s) {
		t.Fatal("events should be blocked after main-session interrupt")
	}
}

func TestAppendInterruptBlock_idempotentWhileViewingSubagent(t *testing.T) {
	s := &state.State{
		Running: true,
		MainChat: []chat.Block{
			{Role: chat.RoleUser},
			{Role: chat.RoleInterrupt},
		},
		SubagentNav:       state.SubagentNavDetail,
		ViewingSubagentID: "sa-1",
	}
	s.Subagents.Start("sa-1", "label", "explore pkg")
	s.SyncDisplayedChat()

	before := len(s.MainChat)
	AppendInterruptBlock(s, func() {})
	if len(s.MainChat) != before {
		t.Fatalf("duplicate interrupt on MainChat: len %d -> %d", before, len(s.MainChat))
	}
}
