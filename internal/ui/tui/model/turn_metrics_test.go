package model

import (
	"strings"
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/hejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/turn"
)

func TestFinalizeLastAssistant_onlyTouchesTrailingBlock(t *testing.T) {
	started := time.Now().Add(-30 * time.Second)
	m := New(&deps.Deps{})
	m.Chat = []chat.Block{
		{Role: chat.RoleAssistant, ReasoningStartedAt: started},
		{Role: chat.RoleTool, ToolName: "read_file"},
		{Role: chat.RoleAssistant, ReasoningStartedAt: time.Now().Add(-2 * time.Second), Streaming: true},
	}
	turn.FinalizeLastAssistant(&m.State, time.Now())

	if !m.Chat[0].ReasoningEndedAt.IsZero() {
		t.Fatal("historical assistant block should not be finalized")
	}
	if m.Chat[2].ReasoningEndedAt.IsZero() || m.Chat[2].Streaming {
		t.Fatal("expected last assistant block finalized and not streaming")
	}
}

func TestApplyTurnMetrics_prefersLastAssistantWithContent(t *testing.T) {
	m := New(&deps.Deps{})
	m.Chat = []chat.Block{
		{Role: chat.RoleUser},
		{Role: chat.RoleAssistant},
		{Role: chat.RoleTool, ToolName: "read_file"},
		{Role: chat.RoleAssistant},
	}
	m.Chat[1].Content.WriteString("preamble")
	m.Chat[3].Reasoning.WriteString("think")

	turn.ApplyTurnMetrics(&m.State, &agent.TurnResult{
		TurnDuration:           5*time.Second + 200*time.Millisecond,
		FinalReasoningDuration: 1200 * time.Millisecond,
	})

	if m.Chat[1].TurnDuration != 5*time.Second+200*time.Millisecond {
		t.Fatalf("turnDuration on content block = %v", m.Chat[1].TurnDuration)
	}
	if m.Chat[3].TurnDuration != 0 {
		t.Fatalf("empty trailing block should not get turnDuration: %v", m.Chat[3].TurnDuration)
	}

	out := chat.Render(m.Chat, 60, time.Now(), false)
	if !strings.Contains(out, "task took 5.2s") {
		t.Fatalf("expected turn duration in output:\n%s", out)
	}
}
