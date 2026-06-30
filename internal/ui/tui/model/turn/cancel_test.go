package turn

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
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
	s.Subagents.Start("sa-1", "label", "explore pkg", "Explore", false)
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
	s.Subagents.Start("sa-1", "label", "explore pkg", "Explore", false)
	s.SyncDisplayedChat()

	before := len(s.MainChat)
	AppendInterruptBlock(s, func() {})
	if len(s.MainChat) != before {
		t.Fatalf("duplicate interrupt on MainChat: len %d -> %d", before, len(s.MainChat))
	}
}

func TestHandleWebFetchPromptKey(t *testing.T) {
	noListen := func() tea.Cmd { return nil }

	tests := []struct {
		key  string
		want permission.WebFetchChoice
	}{
		{"1", permission.WebFetchAllowOnce},
		{"a", permission.WebFetchAllowOnce},
		{"2", permission.WebFetchAllowAlways},
		{"s", permission.WebFetchAllowAlways},
		{"3", permission.WebFetchDeny},
		{"d", permission.WebFetchDeny},
		{"esc", permission.WebFetchDeny},
	}
	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			reply := make(chan permission.WebFetchChoice, 1)
			s := &state.State{
				Overlay: state.OverlayWebFetchPrompt,
				WebFetchPrompt: &permission.WebFetchPromptRequest{
					Host:  "example.com",
					URL:   "https://example.com/",
					Reply: reply,
				},
			}
			HandleWebFetchPromptKey(s, tc.key, noListen)
			select {
			case got := <-reply:
				if got != tc.want {
					t.Fatalf("choice = %v, want %v", got, tc.want)
				}
			default:
				t.Fatal("expected reply on channel")
			}
			if s.Overlay != state.OverlayNone {
				t.Fatalf("overlay = %v, want OverlayNone", s.Overlay)
			}
			if s.WebFetchPrompt != nil {
				t.Fatal("expected WebFetchPrompt cleared")
			}
		})
	}
}
